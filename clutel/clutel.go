package clutel

import (
	"context"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
	"go.opentelemetry.io/otel/trace"
)

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

	return node.EmbedInCtx(ctx, nc.AddValues(
		ctx,
		stringify.Normalize(kvs...),
		node.DoNotAddToSpan(),
		node.AddToOTELHTTPLabeler(ctx),
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

	// nc.AddValues(stringify.Normalize(kvs...))
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
	ctx, spanned := nc.AddSpan(ctx, name, sb.opts...)

	if len(sb.kvs) > 0 {
		spanned.ID = name
		spanned = spanned.AddValues(ctx, sb.kvs)
	} else {
		spanned = spanned.AppendToTree(name)
	}

	return node.EmbedInCtx(ctx, spanned)
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
	nc := node.FromCtx(ctx)
	ctx, spanned := nc.AddSpan(ctx, name)

	if len(kvs) > 0 {
		spanned.ID = name
		spanned = spanned.AddValues(ctx, stringify.Normalize(kvs...))
	} else {
		spanned = spanned.AppendToTree(name)
	}

	return node.EmbedInCtx(ctx, spanned)
}

// EndSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func EndSpan(ctx context.Context) {
	node.FromCtx(ctx).CloseSpan(ctx)
}
