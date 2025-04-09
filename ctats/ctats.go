package ctats

import (
	"context"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
	"github.com/pkg/errors"
)

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type metricsBusKey string

const defaultCtxKey metricsBusKey = "default_metrics_bus_key"

func fromCtx(ctx context.Context) *bus {
	if ctx == nil {
		return nil
	}

	dn := ctx.Value(defaultCtxKey)

	if dn == nil {
		return nil
	}

	return dn.(*bus)
}

func embedInCtx(ctx context.Context, b *bus) context.Context {
	return context.WithValue(ctx, defaultCtxKey, b)
}

type bus struct {
	counters   map[string]metric.Float64UpDownCounter
	gauges     map[string]metric.Float64Gauge
	histograms map[string]metric.Float64Histogram
}

// Initialize ensures that a metrics collector exists in the ctx.
// If the ctx has not already run clues.Initialize() and generated
// OTEL connection details, an error is returned.
//
// Multiple calls to Initialize will no-op all after the first.
func Initialize(ctx context.Context) (context.Context, error) {
	nc := node.FromCtx(ctx)
	if nc == nil || nc.OTEL == nil {
		return ctx, errors.New("clues.Initialize has not been run on this context")
	}

	if fromCtx(ctx) != nil {
		return ctx, nil
	}

	b := &bus{
		counters:   map[string]metric.Float64UpDownCounter{},
		gauges:     map[string]metric.Float64Gauge{},
		histograms: map[string]metric.Float64Histogram{},
	}

	return embedInCtx(ctx, b), nil
}

// InitializeNoop is a convenience function for conditions where ctats gets
// invoked downstream, but has no expectations of upstream initialization,
// such as during unit testing.  A noop init runs ctats as normal but without
// expecting any connection with clients such as OTEL.
func InitializeNoop(ctx context.Context, svc string) (context.Context, error) {
	// if already initialized, do nothing
	if fromCtx(ctx) != nil {
		return ctx, nil
	}

	nc := node.FromCtx(ctx)
	if nc == nil {
		nc = &node.Node{}
	}

	// if otel config already present, we can continue using it.
	if nc.OTEL == nil {
		noc, err := node.NewOTELClient(ctx, svc, node.OTELConfig{})
		if err != nil {
			return ctx, cluerr.Wrap(err, "generating noop otel client")
		}

		nc.OTEL = noc
	}

	ctx = node.EmbedInCtx(ctx, nc)

	ctx, err := Initialize(ctx)

	return ctx, cluerr.Wrap(err, "initializing noop ctats client").OrNil()
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
	id string
}

func (b base) getID() string {
	return formatID(b.id)
}

var (
	camel = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// formatID transforms kebab-case and camelCase to dot.delimited case,
// replaces all spaces with underscores, and lowers the string.
func formatID(id string) string {
	id = strings.ReplaceAll(id, " ", "_")
	id = camel.ReplaceAllString(id, "$1.$2")
	id = strings.ReplaceAll(id, "-", ".")
	return strings.ToLower(id)
}
