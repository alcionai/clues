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
	if nc == nil {
		return ctx
	}

	return nc.AddSpan(
		ctx,
		name,
		stringify.Normalize(kvs...),
		opts...,
	)
}
