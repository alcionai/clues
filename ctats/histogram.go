package ctats

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
)

// histogramFromCtx retrieves the histogram instance from the metrics bus
// in the context.  If the ctx has no metrics bus, or if the bus does
// not have a histogram for the provided ID, returns nil.
func histogramFromCtx(
	ctx context.Context,
	id string,
) metric.Float64Histogram {
	b := fromCtx(ctx)

	if b == nil {
		return nil
	}

	return b.histograms[formatID(id)]
}

// getOrCreateHistogram attempts to retrieve a histogram from the
// context with the given ID.  If it is unable to find a histogram
// with that ID, a new histogram is generated.
func getOrCreateHistogram(
	ctx context.Context,
	id string,
) (metric.Float64Histogram, error) {
	id = formatID(id)

	hist := histogramFromCtx(ctx, id)
	if hist != nil {
		return hist, nil
	}

	// make a new one
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return nil, cluerr.Stack(errNoNodeInCtx)
	}

	hist, err := nc.OTELMeter().Float64Histogram(id)
	if err != nil {
		return nil, errors.Wrap(err, "making new histogram")
	}

	b := fromCtx(ctx)
	b.histograms[id] = hist

	return hist, nil
}

// RegisterHistogram introduces a new histogram with the given unit and description.
// If RegisterHistogram is not called before updating a metric value, a histogram with
// no unit or description is created.  If RegisterHistogram is called for an ID that
// has already been registered, it no-ops.
func RegisterHistogram(
	ctx context.Context,
	// all lowercase, period delimited id of the histogram. Ex: "http.response.status_code"
	id string,
	// (optional) the unit of measurement.  Ex: "byte", "kB", "fnords"
	unit string,
	// (optional) a short description about the metric.  Ex: "number of times we saw the fnords".
	description string,
) (context.Context, error) {
	id = formatID(id)

	// if we already have a histogram registered to that ID, do nothing.
	hist := histogramFromCtx(ctx, id)
	if hist != nil {
		return ctx, nil
	}

	// can't do anything if otel hasn't been initialized.
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return ctx, errors.New("no clues in ctx")
	}

	opts := []metric.Float64HistogramOption{}

	if len(description) > 0 {
		opts = append(opts, metric.WithDescription(description))
	}

	if len(unit) > 0 {
		opts = append(opts, metric.WithUnit(unit))
	}

	// register the histogram
	hist, err := nc.OTELMeter().Float64Histogram(id, opts...)
	if err != nil {
		return ctx, errors.Wrap(err, "creating histogram")
	}

	cb := fromCtx(ctx)
	cb.histograms[id] = hist

	return embedInCtx(ctx, cb), nil
}

// Histogram returns a histogram factory for the provided id.
// If a Histogram instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Histogram[N number](id string) histogram[N] {
	return histogram[N]{base{formatID(id)}}
}

// histogram provides access to the factory functions.
type histogram[N number] struct {
	base
}

// Add increments the histogram by n. n can be negative.
func (c histogram[number]) Record(ctx context.Context, n number) {
	hist, err := getOrCreateHistogram(ctx, c.getID())
	if err != nil {
		fmt.Printf("err getting histogram: %+v\n", err)
		return
	}

	hist.Record(ctx, float64(n))
}
