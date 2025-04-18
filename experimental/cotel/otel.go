package cotel

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

func AddSpanWithOpts(
	ctx context.Context,
	name string,
	kvs []any,
	opts ...trace.SpanStartOption,
) context.Context {
	nc := node.FromCtx(ctx)
	ctx, spanned := nc.AddSpan(ctx, name, opts...)

	if len(kvs) > 0 {
		spanned.ID = name
		spanned = spanned.AddValues(stringify.Normalize(kvs...))
	} else {
		spanned = spanned.AppendToTree(name)
	}

	return node.EmbedInCtx(ctx, spanned)
}
