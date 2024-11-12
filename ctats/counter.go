package ctats

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/internal/node"
)

// counterFromCtx retrieves the counter instance from the metrics bus
// in the context.  If the ctx has no metrics bus, or if the bus does
// not have a counter for the provided ID, returns nil.
func counterFromCtx(
	ctx context.Context,
	id string,
) metric.Float64UpDownCounter {
	b := fromCtx(ctx)
	if b == nil {
		return nil
	}

	return b.counters[formatID(id)]
}

// getOrCreateCounter attempts to retrieve a counter from the
// context with the given ID.  If it is unable to find a counter
// with that ID, a new counter is generated.
func getOrCreateCounter(
	ctx context.Context,
	id string,
) (metric.Float64UpDownCounter, error) {
	id = formatID(id)

	ctr := counterFromCtx(ctx, id)
	if ctr != nil {
		return ctr, nil
	}

	// make a new one
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return nil, errors.New("no node in ctx")
	}

	return nc.OTELMeter().Float64UpDownCounter(id)
}

// RegisterCounter introduces a new counter with the given unit and description.
// If RegisterCounter is not called before updating a metric value, a counter with
// no unit or description is created.  If RegisterCounter is called for an ID that
// has already been registered, it no-ops.
func RegisterCounter(
	ctx context.Context,
	// all lowercase, period delimited id of the counter. Ex: "http.response.status_code"
	id string,
	// (optional) the unit of measurement.  Ex: "byte", "kB", "fnords"
	unit string,
	// (optional) a short description about the metric.  Ex: "number of times we saw the fnords".
	description string,
) (context.Context, error) {
	id = formatID(id)

	// if we already have a counter registered to that ID, do nothing.
	ctr := counterFromCtx(ctx, id)
	if ctr != nil {
		return ctx, nil
	}

	// can't do anything if otel hasn't been initialized.
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return ctx, errors.New("no clues in ctx")
	}

	opts := []metric.Float64UpDownCounterOption{}

	if len(description) > 0 {
		opts = append(opts, metric.WithDescription(description))
	}

	if len(unit) > 0 {
		opts = append(opts, metric.WithUnit(unit))
	}

	// register the counter
	ctr, err := nc.OTELMeter().Float64UpDownCounter(id, opts...)
	if err != nil {
		return ctx, errors.Wrap(err, "creating counter")
	}

	cb := fromCtx(ctx)
	cb.counters[id] = ctr

	return embedInCtx(ctx, cb), nil
}

// Counter returns a counter factory for the provided id.
// If a Counter instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Counter[N number](id string) counter[N] {
	return counter[N]{base{formatID(id)}}
}

// counter provides access to the factory functions.
type counter[N number] struct {
	base
}

// Add increments the counter by n. n can be negative.
func (c counter[number]) Add(ctx context.Context, n number) {
	ctr, err := getOrCreateCounter(ctx, c.getID())
	if err != nil {
		fmt.Printf("err getting counter: %+v\n", err)
		return
	}

	ctr.Add(ctx, float64(n))
}

// Inc is shorthand for Add(ctx, 1).
func (c counter[number]) Inc(ctx context.Context) {
	ctr, err := getOrCreateCounter(ctx, c.getID())
	if err != nil {
		fmt.Printf("err getting counter: %+v\n", err)
		return
	}

	ctr.Add(ctx, 1.0)
}

// Dec is shorthand for Add(ctx, -1).
func (c counter[number]) Dec(ctx context.Context) {
	ctr, err := getOrCreateCounter(ctx, c.getID())
	if err != nil {
		fmt.Printf("err getting counter: %+v\n", err)
		return
	}

	ctr.Add(ctx, -1.0)
}
