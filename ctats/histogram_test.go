package ctats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/alcionai/clues/internal/node"
)

func TestHistogram(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())

	ctx, err := RegisterHistogram(ctx, "reg.h", "test", "testing histogram")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assertNotContains(t, metricBus.counters, "reg.h")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "reg.h")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertContains(t, metricBus.histograms, "reg.h")
	assert.Equal(t, metricBus.histograms.Size(), 1)
	assertNotContains(t, metricBus.sums, "reg.h")
	assert.Equal(t, metricBus.sums.Size(), 0)

	Histogram[int64]("reg.h").Record(ctx, 1)
	Histogram[float64]("reg.h").Record(ctx, 1)

	assertNotContains(t, metricBus.counters, "reg.h")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "reg.h")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertContains(t, metricBus.histograms, "reg.h")
	assert.Equal(t, metricBus.histograms.Size(), 1)
	assertNotContains(t, metricBus.sums, "reg.h")
	assert.Equal(t, metricBus.sums.Size(), 0)

	Histogram[int8]("h").Record(ctx, 1)
	Histogram[int]("h").Record(ctx, -1)
	Histogram[uint8]("h").Record(ctx, 0)

	assertNotContains(t, metricBus.counters, "h")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "h")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertContains(t, metricBus.histograms, "h")
	assert.Equal(t, metricBus.histograms.Size(), 2)
	assertNotContains(t, metricBus.sums, "reg.h")
	assert.Equal(t, metricBus.sums.Size(), 0)
}

func TestHistogramWithAttributes(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingRecorder{}

	metricBus.histograms.Store("with.hist.attrs", recorder)

	attrs := []attribute.KeyValue{attribute.String("key", "val")}

	withAttrs := Histogram[int64]("with.hist.attrs").With("key", "val")

	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())

	withAttrs.Record(ctx, 2)

	assert.Equal(t, 1, recorder.calls)
	require.Len(t, recorder.lastOpts, 1)
}

func TestHistogramWithAttributeKeyValue(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingRecorder{}

	metricBus.histograms.Store("with.hist.kv", recorder)

	withAttrs := Histogram[int64]("with.hist.kv").With("status_code", 500)

	expected := []attribute.KeyValue{attribute.String("status_code", "500")}
	assert.Equal(t, expected, withAttrs.getOTELKVAttrs())

	withAttrs.Record(ctx, 2)

	require.Len(t, recorder.lastOpts, 1)
}

func TestHistogramWithDoesNotMutateBase(t *testing.T) {
	baseHist := Histogram[int64]("mutate.hist")
	attrs := []attribute.KeyValue{attribute.String("foo", "bar")}

	withAttrs := baseHist.With("foo", "bar")

	assert.Nil(t, baseHist.getOTELKVAttrs())
	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())

	second := withAttrs.With("baz", "qux")

	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())
	assert.Len(t, second.getOTELKVAttrs(), 2)
}

// ---------------------------------------------------------------------------
// MakeExponentialHistogramBoundaries
// ---------------------------------------------------------------------------

func TestBoundaries(t *testing.T) {
	testCases := []struct {
		name string
		got  []float64
		want []float64
	}{
		{
			name: "count less than 2 returns [min, max]",
			got:  MakeExponentialHistogramBoundaries(1, 100, 1, 0),
			want: []float64{1, 100},
		},
		{
			name: "count less than 2 ignores scaling factor",
			got:  MakeExponentialHistogramBoundaries(1, 100, 1, 5),
			want: []float64{1, 100},
		},
		{
			name: "scaling factor 0 treated as 1 (no-op)",
			got:  MakeExponentialHistogramBoundaries(10, 1000, 5, 0),
			want: []float64{10, 32, 100, 316, 1000},
		},
		{
			name: "negative scaling factor treated as 1 (no-op)",
			got:  MakeExponentialHistogramBoundaries(10, 1000, 5, -3),
			want: []float64{10, 32, 100, 316, 1000},
		},
		{
			name: "5 buckets, uniform log-spacing",
			got:  MakeExponentialHistogramBoundaries(10, 1000, 5, 1),
			want: []float64{10, 32, 100, 316, 1000},
		},
		{
			name: "15 buckets, uniform log-spacing",
			got:  MakeExponentialHistogramBoundaries(1, 60_000, 15, 1),
			want: []float64{1, 2, 5, 11, 23, 51, 112, 245, 537, 1179, 2588, 5679, 12461, 27344, 60000},
		},
		{
			name: "20 buckets, uniform log-spacing",
			got:  MakeExponentialHistogramBoundaries(1, 60_000, 20, 1),
			want: []float64{1, 2, 3, 6, 10, 18, 32, 58, 103, 183, 327, 584, 1042, 1859, 3317, 5919, 10561, 18845, 33626, 60000},
		},
		{
			name: "scaling factor 2: finer resolution at low end, min and max preserved",
			got:  MakeExponentialHistogramBoundaries(10, 1000, 5, 2),
			want: []float64{10, 13, 32, 133, 1000},
		},
		{
			name: "scaling factor 3: even finer resolution at low end",
			got:  MakeExponentialHistogramBoundaries(10, 1000, 5, 3),
			want: []float64{10, 11, 18, 70, 1000},
		},
		{
			name: "scaling factor 5: extreme skewness, first intermediate bucket saturates to min",
			got:  MakeExponentialHistogramBoundaries(10, 1000, 5, 5),
			want: []float64{10, 10, 12, 30, 1000},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, test.got)
		})
	}
}

// ---------------------------------------------------------------------------
// WithBoundaries option
// ---------------------------------------------------------------------------

func TestHistogramWithBoundariesOption(t *testing.T) {
	want := []float64{1, 10, 100, 1000}

	h := Histogram[int64]("bounds.hist", WithBoundaries(want...))

	assert.Equal(t, want, h.boundaries)
}

func TestHistogramWithBoundariesPreservedByWith(t *testing.T) {
	boundaries := []float64{5, 50, 500}

	base := Histogram[int64]("preserve.bounds", WithBoundaries(boundaries...))
	child := base.With("key", "val")

	assert.Equal(t, boundaries, base.boundaries, "base boundaries unchanged")
	assert.Equal(t, boundaries, child.boundaries, "With must copy boundaries to child")
}

func TestHistogramWithBoundariesDoesNotMutateBase(t *testing.T) {
	boundaries := []float64{1, 2, 3}

	base := Histogram[int64]("nomutate.bounds", WithBoundaries(boundaries...))
	assert.Nil(t, base.getOTELKVAttrs(), "base has no attributes before With")

	child := base.With("k", "v")

	assert.Nil(t, base.getOTELKVAttrs(), "base attributes still nil after With")
	assert.Len(t, child.getOTELKVAttrs(), 1)
	assert.Equal(t, boundaries, child.boundaries, "child carries boundaries")
}

func TestHistogramNoBoundariesByDefault(t *testing.T) {
	h := Histogram[float64]("no.bounds")
	assert.Nil(t, h.boundaries, "no boundaries by default")
}

func TestHistogramFirstBoundariesWin(t *testing.T) {
	first := []float64{1, 10, 100}
	second := []float64{500, 1000, 5000}

	testCases := []struct {
		name  string
		setup func(t *testing.T, ctx context.Context) context.Context
	}{
		{
			name: "first via factory Record",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				Histogram[int64]("first.wins", WithBoundaries(first...)).Record(ctx, 50)
				return ctx
			},
		},
		{
			name: "first via RegisterHistogram",
			setup: func(t *testing.T, ctx context.Context) context.Context {
				ctx, err := RegisterHistogram(ctx, "first.wins", "", "", WithBoundaries(first...))
				require.NoError(t, err)
				return ctx
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			reader := sdkMetric.NewManualReader()
			ctx := ctatsCtx(t, reader)
			ctx = test.setup(t, ctx)

			Histogram[int64]("first.wins", WithBoundaries(second...)).Record(ctx, 50)

			dp := collectHistogram(t, reader, "first.wins")
			assert.Equal(t, float64(100), dp.Bounds[len(dp.Bounds)-1], "second boundaries ignored, first wins")
		})
	}
}

// ---------------------------------------------------------------------------
// Record end-to-end with real OTel MeterProvider
// ---------------------------------------------------------------------------

// ctatsCtx returns a context wired with a real OTel MeterProvider backed by
// the given ManualReader, suitable for testing ctats.Record end-to-end.
func ctatsCtx(t *testing.T, reader *sdkMetric.ManualReader) context.Context {
	t.Helper()

	mp := sdkMetric.NewMeterProvider(sdkMetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	otelClient := &node.OTELClient{
		Meter:         mp.Meter("ctats-test"),
		MeterProvider: mp,
	}

	n := &node.Node{OTEL: otelClient}
	ctx := node.EmbedInCtx(context.Background(), n)

	ctx, err := Initialize(ctx)
	require.NoError(t, err)

	return ctx
}

// collectHistogram retrieves the first data point for a named histogram from a
// ManualReader snapshot.
func collectHistogram(
	t *testing.T,
	reader *sdkMetric.ManualReader,
	name string,
) metricdata.HistogramDataPoint[float64] {
	t.Helper()

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				h, ok := m.Data.(metricdata.Histogram[float64])
				require.True(t, ok, "metric %q is not a Histogram[float64]", name)
				require.NotEmpty(t, h.DataPoints)
				return h.DataPoints[0]
			}
		}
	}

	t.Fatalf("histogram %q not found", name)

	return metricdata.HistogramDataPoint[float64]{}
}

// TestRecordWithDefaultLatencyBoundaries records a 15,000 ms value through the
// full ctats.Record path.
//
// 15,000 falls between bounds[12]=12,461 and bounds[13]=27,344 → bucket index 13.
func TestRecordWithDefaultLatencyBoundaries(t *testing.T) {
	reader := sdkMetric.NewManualReader()
	ctx := ctatsCtx(t, reader)

	boundaries := MakeExponentialHistogramBoundaries(1, 60_000, 15, 1)
	Histogram[int64]("op.latency", WithBoundaries(boundaries...)).Record(ctx, 15_000)

	dp := collectHistogram(t, reader, "op.latency")

	assert.Equal(t, float64(60_000), dp.Bounds[len(dp.Bounds)-1], "last boundary is 60,000 ms")
	assert.Equal(t, uint64(0), dp.BucketCounts[len(dp.BucketCounts)-1], "no overflow")

	// 15,000 ms sits between bounds[12]=12,461 and bounds[13]=27,344
	assert.Equal(t, uint64(1), dp.BucketCounts[13], "15,000 ms lands in bucket 13 (12461–27344 ms)")
}

// TestRecordDefaultOTelBoundariesOverflow shows that without WithBoundaries,
// the OTel SDK default ceiling of 10,000 ms causes 15,000 ms to overflow.
func TestRecordDefaultOTelBoundariesOverflow(t *testing.T) {
	reader := sdkMetric.NewManualReader()
	ctx := ctatsCtx(t, reader)

	Histogram[int64]("op.latency.default").Record(ctx, 15_000)

	dp := collectHistogram(t, reader, "op.latency.default")

	assert.Equal(t, float64(10_000), dp.Bounds[len(dp.Bounds)-1], "default ceiling is 10,000 ms")
	assert.Equal(t, uint64(1), dp.BucketCounts[len(dp.BucketCounts)-1], "15,000 ms overflows to +Inf")
}
