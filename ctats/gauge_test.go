package ctats

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGauge(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())

	ctx, err := RegisterGauge(ctx, "reg.g", "test", "testing gauge")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assertNotContains(t, metricBus.counters, "reg.g")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertContains(t, metricBus.gauges, "reg.g")
	assert.Equal(t, metricBus.gauges.Size(), 1)
	assertNotContains(t, metricBus.histograms, "reg.g")
	assert.Equal(t, metricBus.histograms.Size(), 0)
	assertNotContains(t, metricBus.sums, "reg.g")
	assert.Equal(t, metricBus.sums.Size(), 0)

	Gauge[int64]("reg.g").Set(ctx, 1)
	Gauge[float64]("reg.g").Set(ctx, 1)

	assertNotContains(t, metricBus.counters, "reg.g")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertContains(t, metricBus.gauges, "reg.g")
	assert.Equal(t, metricBus.gauges.Size(), 1)
	assertNotContains(t, metricBus.histograms, "reg.g")
	assert.Equal(t, metricBus.histograms.Size(), 0)
	assertNotContains(t, metricBus.sums, "reg.g")
	assert.Equal(t, metricBus.sums.Size(), 0)

	Gauge[int8]("g").Set(ctx, 1)
	Gauge[int]("g").Set(ctx, 0)

	assertNotContains(t, metricBus.counters, "g")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertContains(t, metricBus.gauges, "g")
	assert.Equal(t, metricBus.gauges.Size(), 2)
	assertNotContains(t, metricBus.histograms, "g")
	assert.Equal(t, metricBus.histograms.Size(), 0)
	assertNotContains(t, metricBus.sums, "reg.g")
	assert.Equal(t, metricBus.sums.Size(), 0)
}

func TestGaugeWithAttributes(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingRecorder{}

	metricBus.gauges.Store("with.gauge.attrs", recorder)

	attrs := []attribute.KeyValue{attribute.String("key", "val")}

	withAttrs := Gauge[int64]("with.gauge.attrs").With("key", "val")

	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())

	withAttrs.Set(ctx, 1)

	assert.Equal(t, 1, recorder.calls)
	require.Len(t, recorder.lastOpts, 1)
}

func TestGaugeWithAttributeKeyValue(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingRecorder{}

	metricBus.gauges.Store("with.gauge.kv", recorder)

	withAttrs := Gauge[int64]("with.gauge.kv").With("status_code", 500)

	expected := []attribute.KeyValue{attribute.String("status_code", "500")}
	assert.Equal(t, expected, withAttrs.getOTELKVAttrs())

	withAttrs.Set(ctx, 1)

	require.Len(t, recorder.lastOpts, 1)
}

func TestGaugeWithDoesNotMutateBase(t *testing.T) {
	baseGauge := Gauge[int64]("mutate.gauge")
	attrs := []attribute.KeyValue{attribute.String("foo", "bar")}

	withAttrs := baseGauge.With("foo", "bar")

	assert.Nil(t, baseGauge.getOTELKVAttrs())
	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())

	second := withAttrs.With("baz", "qux")

	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())
	assert.Len(t, second.getOTELKVAttrs(), 2)
}

type recordingRecorder struct {
	lastValue float64
	lastOpts  []metric.RecordOption
	calls     int
}

func (r *recordingRecorder) Record(_ context.Context, v float64, opts ...metric.RecordOption) {
	r.calls++
	r.lastValue = v
	r.lastOpts = opts
}
