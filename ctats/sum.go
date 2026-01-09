package ctats

import (
	"context"
	"log"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
)

// getOrCreateSum attempts to retrieve a sum from the
// context with the given ID.  If it is unable to find a sum
// with that ID, a new sum is generated.
func getOrCreateSum(
	ctx context.Context,
	id string,
) (adder, error) {
	id = formatID(id)
	b := fromCtx(ctx)

	var ctr adder

	if b != nil {
		// if we already have a sum registered to that ID, do nothing.
		ctr = b.getSum(id)
		if ctr != nil {
			return ctr, nil
		}
	}

	// make a new one
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return nil, errors.New("no node in ctx")
	}

	ctr, err := nc.OTELMeter().Float64Counter(id)
	if err != nil {
		return nil, errors.Wrap(err, "making new sum")
	}

	if b != nil {
		b.sums.Store(id, ctr)
	}

	return ctr, nil
}

// RegisterSum introduces a new sum with the given unit and description.
// If RegisterSum is not called before updating a metric value, a sum with
// no unit or description is created.  If RegisterSum is called for an ID that
// has already been registered, it no-ops.
func RegisterSum(
	ctx context.Context,
	// all lowercase, period delimited id of the sum. Ex: "http.response.status_code"
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

	// if we already have a sum registered to that ID, do nothing.
	ctr = b.getSum(id)
	if ctr != nil {
		return ctx, nil
	}

	// can't do anything if otel hasn't been initialized.
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return ctx, cluerr.Stack(errNoNodeInCtx)
	}

	opts := []metric.Float64CounterOption{}

	if len(description) > 0 {
		opts = append(opts, metric.WithDescription(description))
	}

	if len(unit) > 0 {
		opts = append(opts, metric.WithUnit(unit))
	}

	// register the sum
	ctr, err := nc.OTELMeter().Float64Counter(id, opts...)
	if err != nil {
		return ctx, errors.Wrap(err, "creating sum")
	}

	b.sums.Store(id, ctr)

	return embedInCtx(ctx, b), nil
}

// If a Sum instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Sum[N number](id string) sum[N] {
	return sum[N]{base{formatID(id)}}
}

// sum provides access to the factory functions.
type sum[N number] struct {
	base
}

// Add increments the sum by n. n can be negative.
func (c sum[number]) Add(ctx context.Context, n number) {
	ctr, err := getOrCreateSum(ctx, c.getID())
	if err != nil {
		log.Printf("err getting sum: %+v\n", err)
		return
	}

	ctr.Add(ctx, float64(n))
}

// Inc is shorthand for Add(ctx, 1).
func (c sum[number]) Inc(ctx context.Context) {
	ctr, err := getOrCreateSum(ctx, c.getID())
	if err != nil {
		log.Printf("err getting sum: %+v\n", err)
		return
	}

	ctr.Add(ctx, 1.0)
}
