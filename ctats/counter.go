package ctats

import (
	"context"
	"log"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
)

// getOrCreateCounter attempts to retrieve a counter from the
// context with the given ID.  If it is unable to find a counter
// with that ID, a new counter is generated.
func getOrCreateCounter(
	ctx context.Context,
	id string,
) (adder, error) {
	id = formatID(id)
	b := fromCtx(ctx)

	var ctr adder

	if b != nil {
		// if we already have a counter registered to that ID, do nothing.
		ctr = b.getCounter(id)
		if ctr != nil {
			return ctr, nil
		}
	}

	// make a new one
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return nil, errors.New("no node in ctx")
	}

	ctr, err := nc.OTELMeter().Float64UpDownCounter(id)
	if err != nil {
		return nil, errors.Wrap(err, "making new counter")
	}

	if b != nil {
		b.counters.Store(id, ctr)
	}

	return ctr, nil
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
	// (optional) a short description about the metric.
	// Ex: "number of times we saw the fnords".
	description string,
) (context.Context, error) {
	id = formatID(id)

	b := fromCtx(ctx)
	if b == nil {
		b = newBus()
	}

	var ctr adder

	// if we already have a counter registered to that ID, do nothing.
	ctr = b.getCounter(id)
	if ctr != nil {
		return ctx, nil
	}

	// can't do anything if otel hasn't been initialized.
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return ctx, cluerr.Stack(errNoNodeInCtx)
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

	b.counters.Store(id, ctr)

	return embedInCtx(ctx, b), nil
}

// If a Counter instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Counter[N number](id string) counter[N] {
	return counter[N]{base: base{id: formatID(id)}}
}

// counter provides access to the factory functions.
type counter[N number] struct {
	base
}

func (c counter[N]) With(kvs ...any) counter[N] {
	return counter[N]{base: c.base.with(kvs...)}
}

type adder interface {
	Add(ctx context.Context, incr float64, options ...metric.AddOption)
}

type noopAdder struct{}

func (n noopAdder) Add(context.Context, float64, ...metric.AddOption) {}

// Add increments the counter by n. n can be negative.
func (c counter[number]) Add(ctx context.Context, n number) {
	ctr, err := getOrCreateCounter(ctx, c.getID())
	if err != nil {
		log.Printf("err getting counter: %+v\n", err)
		return
	}

	attrs := c.attrs()

	if len(attrs) == 0 {
		ctr.Add(ctx, float64(n))
		return
	}

	ctr.Add(ctx, float64(n), metric.WithAttributes(attrs...))
}

// Inc is shorthand for Add(ctx, 1).
func (c counter[number]) Inc(ctx context.Context) {
	c.Add(ctx, 1.0)
}

// Dec is shorthand for Add(ctx, -1).
func (c counter[number]) Dec(ctx context.Context) {
	negOne := int64(-1)

	c.Add(ctx, number(negOne))
}
