package clutel

import (
	"context"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
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
	ctx, labeler := node.OTELHTTPLabelerFromCtx(ctx)

	return node.EmbedInCtx(ctx, nc.AddValues(
		ctx,
		stringify.Normalize(kvs...),
		node.AddToOTELHTTPLabeler(labeler),
	))
}
