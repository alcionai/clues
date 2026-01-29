package ctats

import (
	"context"
	"log"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
)

// getOrCreateGauge attempts to retrieve a gauge from the
// context with the given ID.  If it is unable to find a gauge
// with that ID, a new gauge is generated.
func getOrCreateGauge(
	ctx context.Context,
	id string,
) (recorder, error) {
	id = formatID(id)
	b := fromCtx(ctx)

	var gauge recorder

	if b != nil {
		gauge = b.getGauge(id)
		if gauge != nil {
			return gauge, nil
		}
	}

	// make a new one
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return nil, cluerr.Stack(errNoNodeInCtx)
	}

	gauge, err := nc.OTELMeter().Float64Gauge(id)
	if err != nil {
		return nil, errors.Wrap(err, "making new gauge")
	}

	if b != nil {
		b.gauges.Store(id, gauge)
	}

	return gauge, nil
}

// RegisterGauge introduces a new gauge with the given unit and description.
// If RegisterGauge is not called before updating a metric value, a gauge with
// no unit or description is created.  If RegisterGauge is called for an ID that
// has already been registered, it no-ops.
func RegisterGauge(
	ctx context.Context,
	// all lowercase, period delimited id of the gauge. Ex: "http.response.status_code"
	id string,
	// (optional) the unit of measurement.  Ex: "byte", "kB", "fnords"
	unit string,
	// (optional) a short description about the metric.
	// Ex: "number of times we saw the fnords".
	description string,
) (context.Context, error) {
	id = formatID(id)

	b := fromCtx(ctx)
	if b == nil {
		b = newBus()
	}

	var gauge recorder

	gauge = b.getGauge(id)
	if gauge != nil {
		return ctx, nil
	}

	// can't do anything if otel hasn't been initialized.
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return ctx, errors.New("no clues in ctx")
	}

	opts := []metric.Float64GaugeOption{}

	if len(description) > 0 {
		opts = append(opts, metric.WithDescription(description))
	}

	if len(unit) > 0 {
		opts = append(opts, metric.WithUnit(unit))
	}

	// register the gauge
	gauge, err := nc.OTELMeter().Float64Gauge(id, opts...)
	if err != nil {
		return ctx, errors.Wrap(err, "creating gauge")
	}

	b.gauges.Store(id, gauge)

	return embedInCtx(ctx, b), nil
}

// Gauge returns a gauge factory for the provided id.
// If a Gauge instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Gauge[N number](id string) gauge[N] {
	return gauge[N]{base: base{id: formatID(id)}}
}

// gauge provides access to the factory functions.
type gauge[N number] struct {
	base
}

func (c gauge[N]) With(kvs ...any) gauge[N] {
	return gauge[N]{base: c.with(kvs...)}
}

// Set sets the gauge to n.
func (c gauge[number]) Set(ctx context.Context, n number) {
	gauge, err := getOrCreateGauge(ctx, c.getID())
	if err != nil {
		log.Printf("err getting gauge: %+v\n", err)
		return
	}

	attrs := c.getOTELKVAttrs()

	if len(attrs) == 0 {
		gauge.Record(ctx, float64(n))
		return
	}

	gauge.Record(ctx, float64(n), metric.WithAttributes(attrs...))
}
