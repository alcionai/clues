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
// ExponentialBoundaries
// ---------------------------------------------------------------------------

func TestExponentialBoundaries(t *testing.T) {
	testCases := []struct {
		name    string
		min     float64
		max     float64
		count   int
		wantLen int
		wantMin float64
		wantMax float64
	}{
		{
			name:    "standard 20-bucket latency range",
			min:     1,
			max:     60_000,
			count:   20,
			wantLen: 20,
			wantMin: 1,
			wantMax: 60_000,
		},
		{
			name:    "small count",
			min:     10,
			max:     1000,
			count:   5,
			wantLen: 5,
			wantMin: 10,
			wantMax: 1000,
		},
		{
			name:    "count less than 2 returns min and max",
			min:     1,
			max:     100,
			count:   1,
			wantLen: 2,
			wantMin: 1,
			wantMax: 100,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			got := ExponentialBoundaries(test.min, test.max, test.count)

			require.Len(t, got, test.wantLen, "boundary count")
			assert.Equal(t, test.wantMin, got[0], "first boundary must equal min")
			assert.Equal(t, test.wantMax, got[len(got)-1], "last boundary must equal max")

			for i := 1; i < len(got); i++ {
				assert.Greater(t, got[i], got[i-1], "boundaries must be strictly increasing at index %d", i)
			}
		})
	}
}

func TestExponentialBoundariesDefaultLatencyValues(t *testing.T) {
	// Spot-check the documented example output for ExponentialBoundaries(1, 60_000, 20).
	got := ExponentialBoundaries(1, 60_000, 20)

	require.Len(t, got, 20)
	assert.Equal(t, float64(1), got[0])
	assert.Equal(t, float64(60_000), got[19])

	// Mid-range spot checks.
	assert.Equal(t, float64(10), got[4])
	assert.Equal(t, float64(327), got[10])
	assert.Equal(t, float64(10561), got[16])
}

// ---------------------------------------------------------------------------
// PresetLatencyBoundariesMs
// ---------------------------------------------------------------------------

func TestPresetLatencyBoundariesMs(t *testing.T) {
	assert.Len(t, PresetLatencyBoundariesMs, 20, "should have 20 buckets")
	assert.Equal(t, float64(1), PresetLatencyBoundariesMs[0], "first boundary is 1 ms")
	assert.Equal(t, float64(60_000), PresetLatencyBoundariesMs[19], "last boundary is 60,000 ms")

	for i := 1; i < len(PresetLatencyBoundariesMs); i++ {
		assert.Greater(
			t,
			PresetLatencyBoundariesMs[i],
			PresetLatencyBoundariesMs[i-1],
			"boundaries must be strictly increasing at index %d",
			i,
		)
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
// 15,000 falls between bounds[16]=10,561 and bounds[17]=18,845 → bucket index 17.
func TestRecordWithDefaultLatencyBoundaries(t *testing.T) {
	reader := sdkMetric.NewManualReader()
	ctx := ctatsCtx(t, reader)

	Histogram[int64]("op.latency", WithBoundaries(PresetLatencyBoundariesMs...)).Record(ctx, 15_000)

	dp := collectHistogram(t, reader, "op.latency")

	assert.Equal(t, float64(60_000), dp.Bounds[len(dp.Bounds)-1], "last boundary is 60,000 ms")
	assert.Equal(t, uint64(0), dp.BucketCounts[len(dp.BucketCounts)-1], "no overflow")

	// 15,000 ms sits between bounds[16]=10,561 and bounds[17]=18,845
	assert.Equal(t, uint64(1), dp.BucketCounts[17], "15,000 ms lands in bucket 17 (10561–18845 ms)")
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
