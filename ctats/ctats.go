package ctats

import (
	"context"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

// Initialize ensures that a metrics collector exists in the ctx.
// If the ctx has not already run clues.Initialize() and generated
// OTEL connection details, an error is returned.
//
// Multiple calls to Initialize will no-op all after the first.
func Initialize(ctx context.Context) (context.Context, error) {
	if fromCtx(ctx) != nil {
		return ctx, nil
	}

	nc := node.FromCtx(ctx)
	if nc == nil || nc.OTEL == nil {
		return ctx, errors.New("clues.Initialize has not been run on this context")
	}

	b := newBus()

	return embedInCtx(ctx, b), nil
}

// InitializeNoop is a convenience function for conditions where ctats gets
// invoked downstream, but has no expectations of upstream initialization,
// such as during unit testing.  A noop init runs ctats as normal but without
// expecting any connection with clients such as OTEL.
func InitializeNoop(ctx context.Context, svc string) context.Context {
	// if already initialized, do nothing
	if fromCtx(ctx) != nil {
		return ctx
	}

	b := newBus()
	b.initializedToNoop = true

	return embedInCtx(ctx, b)
}

// Inherit propagates the ctats client from one context to another.  This is particularly
// useful for taking an initialized context from a main() func and ensuring the ctats
// is available for request-bound conetxts, such as in a http server pattern.
//
// If the 'to' context already contains an initialized ctats, no change is made.
// Callers can force a 'from' ctats to override a 'to' ctats by setting clobber=true.
func Inherit(
	from, to context.Context,
	clobber bool,
) context.Context {
	fromBus := fromCtx(from)
	toBus := fromCtx(to)

	if to == nil {
		to = context.Background()
	}

	// return the 'to' context unmodified if we won't update the context.
	if fromBus == nil || (toBus != nil && !clobber) {
		return to
	}

	return embedInCtx(to, fromBus)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// number covers the values that callers are allowed to provide
// to the metrics factories.  No matter the provided value, a
// float64 will be recorded to the metrics collector.
type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// base contains the properties common to all metrics factories.
type base struct {
	id   string
	data *node.Node
}

func (b base) getID() string {
	return formatID(b.id)
}

func (b base) with(kvs ...any) base {
	if len(kvs) == 0 {
		return b
	}

	if b.data == nil {
		b.data = &node.Node{}
	}

	b.data = b.data.AddValues(context.Background(), normalizeKVs(kvs...))

	return b
}

func (b base) attrs() []attribute.KeyValue {
	if b.data == nil {
		return nil
	}

	return b.data.OTELAttributes()
}

func normalizeKVs(kvs ...any) map[string]any {
	if len(kvs) == 0 {
		return nil
	}

	result := map[string]any{}
	remaining := make([]any, 0, len(kvs))

	for _, kv := range kvs {
		if attr, ok := kv.(attribute.KeyValue); ok {
			result[string(attr.Key)] = attrValueToAny(attr.Value)
			continue
		}

		remaining = append(remaining, kv)
	}

	if len(remaining) > 0 {
		norm := stringify.Normalize(remaining...)
		for k, v := range norm {
			result[k] = v
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func attrValueToAny(v attribute.Value) any {
	switch v.Type() {
	case attribute.BOOL:
		return v.AsBool()
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return v.AsFloat64()
	case attribute.STRING:
		return v.AsString()
	case attribute.BOOLSLICE:
		return v.AsBoolSlice()
	case attribute.INT64SLICE:
		return v.AsInt64Slice()
	case attribute.FLOAT64SLICE:
		return v.AsFloat64Slice()
	case attribute.STRINGSLICE:
		return v.AsStringSlice()
	default:
		return stringify.Marshal(v.AsInterface(), false)
	}
}

var camel = regexp.MustCompile("([a-z0-9])([A-Z])")

// formatID transforms kebab-case and camelCase to dot.delimited case,
// replaces all spaces with underscores, and lowers the string.
func formatID(id string) string {
	id = strings.ReplaceAll(id, " ", "_")
	id = camel.ReplaceAllString(id, "$1.$2")
	id = strings.ReplaceAll(id, "-", ".")

	return strings.ToLower(id)
}
