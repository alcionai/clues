package ctats

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
)

// gaugeFromCtx retrieves the gauge instance from the metrics bus
// in the context.  If the ctx has no metrics bus, or if the bus does
// not have a gauge for the provided ID, returns nil.
func gaugeFromCtx(
	ctx context.Context,
	id string,
) metric.Float64Gauge {
	b := fromCtx(ctx)
	if b == nil {
		return nil
	}

	return b.gauges[formatID(id)]
}

// getOrCreateGauge attempts to retrieve a gauge from the
// context with the given ID.  If it is unable to find a gauge
// with that ID, a new gauge is generated.
func getOrCreateGauge(
	ctx context.Context,
	id string,
) (metric.Float64Gauge, error) {
	id = formatID(id)

	gauge := gaugeFromCtx(ctx, id)
	if gauge != nil {
		return gauge, nil
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

	b := fromCtx(ctx)
	b.gauges[id] = gauge

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
	// (optional) a short description about the metric.  Ex: "number of times we saw the fnords".
	description string,
) (context.Context, error) {
	id = formatID(id)

	// if we already have a gauge registered to that ID, do nothing.
	gauge := gaugeFromCtx(ctx, id)
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

	cb := fromCtx(ctx)
	cb.gauges[id] = gauge

	return embedInCtx(ctx, cb), nil
}

// Gauge returns a gauge factory for the provided id.
// If a Gauge instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Gauge[N number](id string) gauge[N] {
	return gauge[N]{base{formatID(id)}}
}

// gauge provides access to the factory functions.
type gauge[N number] struct {
	base
}

// Set sets the gauge to n.
func (c gauge[number]) Set(ctx context.Context, n number) {
	gauge, err := getOrCreateGauge(ctx, c.getID())
	if err != nil {
		fmt.Printf("err getting gauge: %+v\n", err)
		return
	}

	gauge.Record(ctx, float64(n))
}
