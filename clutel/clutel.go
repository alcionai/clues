package clutel

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

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
// closing `defer clues.EndSpan(ctx)`.
func StartSpan(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	return NewSpan().
		WithAttrs(kvs...).
		Start(ctx, name)
}

// EndSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func EndSpan(ctx context.Context) {
	node.CloseSpan(ctx)
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
