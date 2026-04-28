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

// PresetLatencyBoundariesMs are logarithmically-spaced bucket boundaries from
// 1 to 60_000, suitable for measuring operation latency in milliseconds up to 60s.
var PresetLatencyBoundariesMs = ExponentialBoundaries(1, 60_000, 20)

// ExponentialBoundaries returns count boundaries spaced logarithmically between
// min and max (both inclusive), mirroring Prometheus's ExponentialBucketsRange:
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#ExponentialBucketsRange
//
// Example:
//
//	ExponentialBoundaries(1, 60_000, 20)
//	// → [1 2 3 6 10 18 32 58 103 183 327 584 1042 1859 3317 5919 10561 18845 33626 60000]
func ExponentialBoundaries(min, max float64, count int) []float64 {
	if count < 2 {
		return []float64{min, max}
	}

	factor := math.Pow(max/min, 1/float64(count-1))
	b := make([]float64, count)

	for i := range b {
		b[i] = math.Round(min * math.Pow(factor, float64(i)))
	}

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

// Histogram returns a histogram factory for the given id. If the id was
// previously registered via RegisterHistogram that instance is reused;
// otherwise a new one is created on the first Record call.
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
