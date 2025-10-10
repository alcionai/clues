package clutel

import (
	"context"
	"maps"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/errs"
	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

// ---------------------------------------------------------------------------
// persistent client initialization
// ---------------------------------------------------------------------------

// Init will spin up the OTEL clients that are held by cluetel,
// Clues will eagerly use these clients in the background to provide
// additional telemetry hook-ins.
//
// Clues will operate as expected in the event of an error, or if OTEL is not
// initialized. This is a purely optional step.
func Init(
	ctx context.Context,
	serviceName string,
	config OTELConfig,
) (context.Context, error) {
	nc := node.FromCtx(ctx)

	err := nc.InitOTEL(ctx, serviceName, config.toInternalConfig())
	if err != nil {
		return ctx, err
	}

	return node.EmbedInCtx(ctx, nc), nil
}

// Close will flush all buffered data waiting to be read.  If Initialize was not
// called, this call is a no-op.  Should be called in a defer after initializing.
func Close(ctx context.Context) error {
	nc := node.FromCtx(ctx)

	if nc.OTEL == nil {
		return nil
	}

	err := nc.OTEL.Close(ctx)
	if err != nil {
		return errors.Wrap(err, "closing otel client")
	}

	return nil
}

// Inherit propagates all clients and otel data (ie: live trace and baggage) from
// one context to another.  This is particularly useful for taking an initialized
// context from a main() func and ensuring its clients are available for request-
// bound conetxts, such as in a http server pattern.
//
// If the 'to' context already contains an initialized client, no change is made.
// Callers can force a 'from' client to override a 'to' client by setting clobber=true.
func Inherit(
	from, to context.Context,
	clobber bool,
) context.Context {
	fromNode := node.FromCtx(from)

	if to == nil {
		to = context.Background()
	}

	toNode := node.FromCtx(to)

	// if we have no fromNode OTEL, or are not clobbering, return the toNode.
	if fromNode.OTEL == nil || (toNode.OTEL != nil && !clobber) {
		return node.EmbedInCtx(to, toNode)
	}

	// otherwise pass along the fromNode OTEL client.
	toNode.OTEL = fromNode.OTEL

	to = node.EmbedInCtx(to, toNode)

	details := map[string]string{}
	ReceiveTrace(from, details)
	InjectTrace(to, details)

	return to
}

// ---------------------------------------------------------------------------
// spans
// ---------------------------------------------------------------------------

// AddToOTELHTTPLabeler adds key-value pairs to both the current
// context and the OpenTelemetry HTTP labeler, but not the current
// span.  The labeler will hold onto these values until the next
// request arrives at the otelhttp transport, at which point they
// are added to the span for that transport.
//
// The best use case for this func is to wait until the last wrapper
// used to handle a http.Request.Do() call.  Add your http request
// details (url, payload metadata, etc) at that point so that they
// appear both in the next span, and in any errors you handle from
// that wrapper.
func AddToOTELHTTPLabeler(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	nc := node.FromCtx(ctx)
	ctx, labeler := node.OTELHTTPLabelerFromCtx(ctx)

	return node.EmbedInCtx(ctx, nc.AddValues(
		ctx,
		stringify.Normalize(kvs...),
		node.AddToOTELHTTPLabeler(labeler),
	))
}

// GetSpan retrieves the current OpenTelemetry span from the context.
func GetSpan(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func GetTraceID(ctx context.Context) string {
	span := GetSpan(ctx)
	if span == nil {
		return ""
	}

	return span.SpanContext().TraceID().String()
}

type spanBuilder struct {
	kvs  map[string]any
	opts []trace.SpanStartOption
}

// NewSpan produces a span builder that allows complete configuration and
// attribution of the span before it gets started.
func NewSpan() *spanBuilder {
	return &spanBuilder{}
}

// WithAttrs adds attrs to the span.
func (sb *spanBuilder) WithAttrs(kvs ...any) *spanBuilder {
	if sb.kvs == nil {
		sb.kvs = make(map[string]any)
	}

	sb.kvs = stringify.Normalize(kvs...)

	return sb
}

// WithOpts configures the span with specific otel options.  Note that one
// of the options can be `trace.WithAttributes()`; don't use that, use
// spanBuilder.WithAttrs() instead.
func (sb *spanBuilder) WithOpts(opts ...trace.SpanStartOption) *spanBuilder {
	if sb.opts == nil {
		sb.opts = make([]trace.SpanStartOption, 0, len(opts))
	}

	sb.opts = append(sb.opts, opts...)

	return sb
}

// Start begins the span with the provided name and attaches it to the context.
func (sb *spanBuilder) Start(
	ctx context.Context,
	name string,
) context.Context {
	nc := node.FromCtx(ctx)
	if nc == nil {
		return ctx
	}

	return nc.AddSpan(
		ctx,
		name,
		sb.kvs,
		sb.opts...,
	)
}

// StartSpan stacks a clues node onto this context and uses the provided
// name to generate an OTEL span name. StartSpan can be called without
// adding attributes. Callers should always follow this addition with a
// closing `defer clutel.EndSpan(ctx)`.
func StartSpan(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	return NewSpan().
		WithAttrs(kvs...).
		Start(ctx, name)
}

// EndSpan closes the current span.  Should only be called following a
// `clutel.StartSpan()` call.
func EndSpan(ctx context.Context) {
	node.CloseSpan(ctx)
}

// EndSpanWithError closes the current span, setting the span status to an
// Error, but only if the provided error is not nil.  Should only be called
// following a `clutel.StartSpan()` call.
func EndSpanWithError(ctx context.Context, err error) {
	if errs.IsNilIface(err) {
		EndSpan(ctx)
		return
	}

	node.SetSpanError(ctx, err, "")
}

// SetSpanError sets the current span to Error, using the provided error.
// No-ops if the error is nil.
func SetSpanError(ctx context.Context, err error) {
	if errs.IsNilIface(err) {
		return
	}

	node.SetSpanError(ctx, err, "")
}

// SetSpanErrorM sets the current span to Error, using the provided error message.
// No-ops if the message is empty.
func SetSpanErrorM(ctx context.Context, msg string) {
	if len(msg) == 0 {
		return
	}

	node.SetSpanError(ctx, nil, msg)
}

// ---------------------------------------------------------------------------
// traces
// ---------------------------------------------------------------------------

// InjectTrace adds the current trace details to the provided
// headers.  If otel is not initialized, no-ops.
//
// The mapCarrier is mutated by this request.  The passed
// reference is returned mostly as a quality-of-life step
// so that callers don't need to declare the map outside of
// this call.
func InjectTrace[C node.TraceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) C {
	node.FromCtx(ctx).
		InjectTrace(ctx, node.AsTraceMapCarrier(mapCarrier))

	return mapCarrier
}

// ReceiveTrace extracts the current trace details from the
// headers and adds them to the context.  If otel is not
// initialized, no-ops.
func ReceiveTrace[C node.TraceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) context.Context {
	return node.FromCtx(ctx).
		ReceiveTrace(ctx, node.AsTraceMapCarrier(mapCarrier))
}

// ---------------------------------------------------------------------------
// baggage
// ---------------------------------------------------------------------------

// AddBaggage adds each key-value pair to the context as member-level
// baggages. The values are also added to the context as clues.
func AddBaggage(
	ctx context.Context,
	kvs ...any,
) (context.Context, error) {
	var (
		nc   = node.FromCtx(ctx)
		bag  = baggage.FromContext(ctx)
		nKvs = stringify.Normalize(kvs...)
	)

	for k, v := range nKvs {
		mem, err := baggage.NewMemberRaw(k, stringify.Marshal(v, false))
		if err != nil {
			return ctx, cluerr.WrapWC(ctx, err, "creating baggage member").
				With("bag_key", k, "bag_value", v)
		}

		bag, err = bag.SetMember(mem)
		if err != nil {
			return ctx, cluerr.WrapWC(ctx, err, "adding baggage member").
				With("bag_key", k, "bag_value", v)
		}
	}

	nc = nc.AddValues(ctx, nKvs, node.DoNotAddToSpan())
	ctx = baggage.ContextWithBaggage(ctx, bag)

	return node.EmbedInCtx(ctx, nc), nil
}

type BaggageProps struct {
	memberKey   string
	memberValue any
	props       map[string]string
}

func (bp BaggageProps) MemberKey() string {
	return bp.memberKey
}

// NewBaggageProps transitions all the provided key-value pairs into a
// single baggage member.  The first two values define the baggage key
// and value.  Tuples beyond that are added to the baggage as additional
// properties.
func NewBaggageProps(kvs ...any) BaggageProps {
	switch len(kvs) {
	case 0:
		return BaggageProps{}
	case 1:
		return BaggageProps{
			memberKey:   stringify.Marshal(kvs[0], false),
			memberValue: "<nil>",
		}
	case 2:
		return BaggageProps{
			memberKey:   stringify.Marshal(kvs[0], false),
			memberValue: stringify.Marshal(kvs[1], false),
		}
	default:
		return BaggageProps{
			memberKey:   stringify.Marshal(kvs[0], false),
			memberValue: stringify.Marshal(kvs[1], false),
			props:       stringify.Stringalize(kvs[2:]...),
		}
	}
}

// ToMember converts the baggageProps into a baggage.Member,
// adding all properties as baggage properties.
func (bp BaggageProps) ToMember() (baggage.Member, bool, error) {
	if len(bp.memberKey) == 0 {
		return baggage.Member{}, false, nil
	}

	props := []baggage.Property{}

	for k, v := range bp.props {
		prop, err := baggage.NewKeyValuePropertyRaw(k, stringify.Marshal(v, false))
		if err != nil {
			return baggage.Member{}, false, cluerr.Wrap(err, "creating baggage property").
				With("key", k, "value", v)
		}

		props = append(props, prop)
	}

	mem, err := baggage.NewMemberRaw(
		bp.memberKey,
		stringify.Marshal(bp.memberValue, false),
		props...,
	)
	if err != nil {
		return baggage.Member{}, false, cluerr.Wrap(err, "creating baggage member").
			With("key", bp.memberKey, "value", bp.memberValue)
	}

	return mem, true, nil
}

// ToMapStringAny converts the baggageProps into a map[string]any
// suitable for adding to a clues node.
func (bp BaggageProps) ToMapStringAny() map[string]any {
	if len(bp.memberKey) == 0 {
		return nil
	}

	m := map[string]any{
		bp.memberKey: bp.memberValue,
	}

	if len(bp.props) > 0 {
		m[bp.memberKey+"_props"] = bp.props
	}

	return m
}

type BaggagePropper interface {
	// ToMember converts the BaggagePropper into a baggage.Member.
	ToMember() (
		baggage.Member,
		bool, // false if the member contains no data.
		error,
	)

	// ToMapStringAny converts the BaggagePropper into a map[string]any
	// suitable for adding to a clues node.
	ToMapStringAny() map[string]any
}

// AddBaggageProps adds each BaggagePropper to the context as member-level
// baggages. Additional properties (as provided to each BaggagePropper) are
// included in the member.  The values are also added to the context as clues,
// formatted according to the BaggagePropper's ToMapStringAny() method.
func AddBaggageProps(
	ctx context.Context,
	bagProps ...BaggagePropper,
) (context.Context, error) {
	var (
		nc       = node.FromCtx(ctx)
		bag      = baggage.FromContext(ctx)
		cluesKVs = map[string]any{}
	)

	for _, bp := range bagProps {
		mem, ok, err := bp.ToMember()
		if err != nil {
			return ctx, cluerr.WrapWC(ctx, err, "creating baggage member")
		}

		if !ok {
			continue
		}

		bagKVs := bp.ToMapStringAny()

		bag, err = bag.SetMember(mem)
		if err != nil {
			return ctx, cluerr.WrapWC(ctx, err, "adding baggage member").
				With("baggage_values", bagKVs)
		}

		maps.Copy(cluesKVs, bagKVs)
	}

	nc = nc.AddValues(ctx, cluesKVs, node.DoNotAddToSpan())
	ctx = baggage.ContextWithBaggage(ctx, bag)

	return node.EmbedInCtx(ctx, nc), nil
}

func GetBaggage(
	ctx context.Context,
	memberKey string,
) (baggage.Member, bool) {
	bags := baggage.FromContext(ctx)
	members := bags.Members()

	for _, member := range members {
		if member.Key() == memberKey {
			return member, true
		}
	}

	return baggage.Member{}, false
}
