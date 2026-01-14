package ctats

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	assert.Equal(t, attrs, withAttrs.attrs())

	withAttrs.Record(ctx, 2)

	assert.Equal(t, 1, recorder.calls)
	require.Len(t, recorder.lastOpts, 1)
}

func TestHistogramWithAttributeKeyValue(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingRecorder{}

	metricBus.histograms.Store("with.hist.kv", recorder)

	withAttrs := Histogram[int64]("with.hist.kv").With(attribute.Int("status_code", 500))

	expected := []attribute.KeyValue{attribute.Int("status_code", 500)}
	assert.Equal(t, expected, withAttrs.attrs())

	withAttrs.Record(ctx, 2)

	require.Len(t, recorder.lastOpts, 1)
}

func TestHistogramWithDoesNotMutateBase(t *testing.T) {
	baseHist := Histogram[int64]("mutate.hist")
	attrs := []attribute.KeyValue{attribute.String("foo", "bar")}

	withAttrs := baseHist.With("foo", "bar")

	assert.Nil(t, baseHist.attrs())
	assert.Equal(t, attrs, withAttrs.attrs())

	second := withAttrs.With("baz", "qux")

	assert.Equal(t, attrs, withAttrs.attrs())
	assert.Len(t, second.attrs(), 2)
}
