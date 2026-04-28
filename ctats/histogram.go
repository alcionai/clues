package ctats

import (
	"context"
	"log"
	"math"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
)

// MakeExponentialHistogramBoundaries returns count boundaries spaced logarithmically
// between min and max (both inclusive). For background on explicit bucket histograms
// and how boundaries map to OTel buckets, see the OTel metrics SDK spec:
// https://opentelemetry.io/docs/specs/otel/metrics/sdk/#explicit-bucket-histogram-aggregation
//
// scalingFactor controls how densely buckets are packed toward the low end of the
// range. At 1 (the default for any value ≤ 1), positions are uniformly log-spaced —
// constant growth ratio between consecutive buckets. Values greater than 1 warp the
// position distribution so that more bucket edges cluster near min.
//
// Example:
//
//	MakeExponentialHistogramBoundaries(1, 60_000, 15, 1)
//	// → [1 2 5 11 23 51 112 245 537 1179 2588 5679 12461 27344 60000]
//
//	MakeExponentialHistogramBoundaries(10, 1000, 5, 1)
//	// → [10 32 100 316 1000]   (uniform log-spacing)
//
//	MakeExponentialHistogramBoundaries(10, 1000, 5, 2)
//	// → [10 13 32 133 1000]    (denser at low end, same range)
func MakeExponentialHistogramBoundaries(min, max float64, count int, scalingFactor float64) []float64 {
	if scalingFactor <= 1 {
		scalingFactor = 1
	}

	if count < 2 {
		return []float64{min, max}
	}

	b := make([]float64, count)

	for i := range b {
		t := math.Pow(float64(i)/float64(count-1), scalingFactor)
		b[i] = math.Round(min * math.Pow(max/min, t))
	}

	b[0] = min       // guarantee exact floor, no rounding drift
	b[count-1] = max // guarantee exact ceiling, no rounding drift

	return b
}

type histogramCfg struct {
	boundaries []float64
}

type HistogramOption func(*histogramCfg)

// WithBoundaries sets explicit bucket boundaries on the histogram.
// Boundaries are passed to the OTel SDK at instrument creation time and are
// ignored if a matching MeterProvider View is already configured.
func WithBoundaries(boundaries ...float64) HistogramOption {
	return func(c *histogramCfg) {
		c.boundaries = boundaries
	}
}

// getOrCreateHistogram attempts to retrieve a histogram from the
// context with the given ID.  If it is unable to find a histogram
// with that ID, a new histogram is generated.
func getOrCreateHistogram(
	ctx context.Context,
	id string,
	boundaries []float64,
) (recorder, error) {
	id = formatID(id)
	b := fromCtx(ctx)

	var hist recorder

	if b != nil {
		hist = b.getHistogram(id)
		if hist != nil {
			return hist, nil
		}
	}

	// make a new one
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return nil, cluerr.Stack(errNoNodeInCtx)
	}

	var opts []metric.Float64HistogramOption
	if len(boundaries) > 0 {
		opts = append(opts, metric.WithExplicitBucketBoundaries(boundaries...))
	}

	// register the histogram
	hist, err := nc.OTELMeter().Float64Histogram(id, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "making new histogram")
	}

	if b != nil {
		b.histograms.Store(id, hist)
	}

	return hist, nil
}

// RegisterHistogram introduces a new histogram with the given unit and description.
// If RegisterHistogram is not called before updating a metric value, a histogram with
// no unit or description is created. If RegisterHistogram is called for an ID that
// has already been registered, it no-ops.
func RegisterHistogram(
	ctx context.Context,
	// all lowercase, period delimited id of the histogram. Ex: "http.response.size"
	id string,
	// (optional) the unit of measurement.  Ex: "byte", "kB", "fnords"
	unit string,
	// (optional) a short description about the metric.
	// Ex: "number of times we saw the fnords".
	description string,
	// (optional) histogram specific options
	opts ...HistogramOption,
) (context.Context, error) {
	id = formatID(id)

	b := fromCtx(ctx)
	if b == nil {
		b = newBus()
	}

	var hist recorder

	hist = b.getHistogram(id)
	if hist != nil {
		return ctx, nil
	}

	// can't do anything if otel hasn't been initialized.
	nc := node.FromCtx(ctx)
	if nc.OTEL == nil {
		return ctx, errors.New("no clues in ctx")
	}

	var cfg histogramCfg
	for _, o := range opts {
		o(&cfg)
	}

	var metricHistogramOpts []metric.Float64HistogramOption

	if len(description) > 0 {
		metricHistogramOpts = append(metricHistogramOpts, metric.WithDescription(description))
	}

	if len(unit) > 0 {
		metricHistogramOpts = append(metricHistogramOpts, metric.WithUnit(unit))
	}

	if len(cfg.boundaries) > 0 {
		metricHistogramOpts = append(metricHistogramOpts, metric.WithExplicitBucketBoundaries(cfg.boundaries...))
	}

	hist, err := nc.OTELMeter().Float64Histogram(id, metricHistogramOpts...)
	if err != nil {
		return ctx, errors.Wrap(err, "creating histogram")
	}

	b.histograms.Store(id, hist)

	return embedInCtx(ctx, b), nil
}

// Histogram returns a histogram factory for the provided id.
// If a Histogram instance has been registered for that ID, the
// registered instance will be used.  If not, a new instance
// will get generated.
func Histogram[N number](id string, opts ...HistogramOption) histogram[N] {
	hgm := histogram[N]{base: base{id: formatID(id)}}
	for _, o := range opts {
		o(&hgm.histogramCfg)
	}

	return hgm
}

// histogram provides access to the factory functions.
type histogram[N number] struct {
	base
	histogramCfg
}

func (c histogram[N]) With(kvs ...any) histogram[N] {
	return histogram[N]{base: c.with(kvs...), histogramCfg: c.histogramCfg}
}

type recorder interface {
	Record(ctx context.Context, incr float64, options ...metric.RecordOption)
}

type noopRecorder struct{}

func (n noopRecorder) Record(context.Context, float64, ...metric.RecordOption) {}

// Record records the measurement of n in the histogram.
func (c histogram[number]) Record(ctx context.Context, n number) {
	hist, err := getOrCreateHistogram(ctx, c.getID(), c.boundaries)
	if err != nil {
		log.Printf("err getting histogram: %+v\n", err)
		return
	}

	attrs := c.getOTELKVAttrs()

	if len(attrs) == 0 {
		hist.Record(ctx, float64(n))
		return
	}

	hist.Record(ctx, float64(n), metric.WithAttributes(attrs...))
}
