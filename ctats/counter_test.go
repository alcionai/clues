package ctats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestCounter(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())

	ctx, err := RegisterCounter(ctx, "reg.c", "test", "testing counter")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assertContains(t, metricBus.counters, "reg.c")
	assert.Equal(t, metricBus.counters.Size(), 1)
	assertNotContains(t, metricBus.gauges, "reg.c")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertNotContains(t, metricBus.histograms, "reg.c")
	assert.Equal(t, metricBus.histograms.Size(), 0)
	assertNotContains(t, metricBus.sums, "reg.c")
	assert.Equal(t, metricBus.sums.Size(), 0)

	Counter[int64]("reg.c").Add(ctx, 1)
	Counter[float64]("reg.c").Add(ctx, 1)
	Counter[int64]("reg.c").Inc(ctx)
	Counter[float64]("reg.c").Inc(ctx)
	Counter[int64]("reg.c").Dec(ctx)
	Counter[float64]("reg.c").Dec(ctx)

	assertContains(t, metricBus.counters, "reg.c")
	assert.Equal(t, metricBus.counters.Size(), 1)
	assertNotContains(t, metricBus.gauges, "reg.c")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertNotContains(t, metricBus.histograms, "reg.c")
	assert.Equal(t, metricBus.histograms.Size(), 0)
	assertNotContains(t, metricBus.sums, "reg.c")
	assert.Equal(t, metricBus.sums.Size(), 0)

	Counter[int8]("c").Add(ctx, 1)
	Counter[float32]("c").Inc(ctx)
	Counter[uint16]("c").Dec(ctx)
	Counter[int]("c").Dec(ctx)

	assertContains(t, metricBus.counters, "c")
	assert.Equal(t, metricBus.counters.Size(), 2)
	assertNotContains(t, metricBus.gauges, "c")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertNotContains(t, metricBus.histograms, "c")
	assert.Equal(t, metricBus.histograms.Size(), 0)
	assertNotContains(t, metricBus.sums, "reg.c")
	assert.Equal(t, metricBus.sums.Size(), 0)
}

func TestCounterWithAttributes(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingAdder{}

	metricBus.counters.Store("with.counter.attrs", recorder)

	attrs := []attribute.KeyValue{attribute.String("key", "val")}

	withAttrs := Counter[int64]("with.counter.attrs").With("key", "val")

	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())

	withAttrs.Inc(ctx)

	assert.Equal(t, 1, recorder.calls)
	require.Len(t, recorder.lastOpts, 1)
}

func TestCounterWithAttributeKeyValue(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingAdder{}

	metricBus.counters.Store("with.counter.kv", recorder)

	withAttrs := Counter[int64]("with.counter.kv").With("status_code", 500)

	expected := []attribute.KeyValue{attribute.String("status_code", "500")}
	assert.Equal(t, expected, withAttrs.getOTELKVAttrs())

	withAttrs.Inc(ctx)

	require.Len(t, recorder.lastOpts, 1)
}

func TestCounterWithDoesNotMutateBase(t *testing.T) {
	baseCounter := Counter[int64]("mutate.counter")
	attrs := []attribute.KeyValue{attribute.String("foo", "bar")}

	withAttrs := baseCounter.With("foo", "bar")

	assert.Nil(t, baseCounter.getOTELKVAttrs())
	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())

	second := withAttrs.With("baz", "qux")

	assert.Equal(t, attrs, withAttrs.getOTELKVAttrs())
	assert.Len(t, second.getOTELKVAttrs(), 2)
}
