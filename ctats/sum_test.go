package ctats

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSum(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())

	ctx, err := RegisterSum(ctx, "reg.s", "test", "testing sum")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assertContains(t, metricBus.sums, "reg.s")
	assert.Equal(t, metricBus.sums.Size(), 1)
	assertNotContains(t, metricBus.counters, "reg.s")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "reg.s")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertNotContains(t, metricBus.histograms, "reg.s")
	assert.Equal(t, metricBus.histograms.Size(), 0)

	Sum[int64]("reg.s").Add(ctx, 1)
	Sum[float64]("reg.s").Add(ctx, 1)
	Sum[int64]("reg.s").Inc(ctx)
	Sum[float64]("reg.s").Inc(ctx)

	assertContains(t, metricBus.sums, "reg.s")
	assert.Equal(t, metricBus.sums.Size(), 1)
	assertNotContains(t, metricBus.counters, "reg.s")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "reg.s")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertNotContains(t, metricBus.histograms, "reg.s")
	assert.Equal(t, metricBus.histograms.Size(), 0)

	Sum[int8]("s").Add(ctx, 1)
	Sum[float32]("s").Inc(ctx)

	assertContains(t, metricBus.sums, "s")
	assert.Equal(t, metricBus.sums.Size(), 2)
	assertNotContains(t, metricBus.counters, "s")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "s")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertNotContains(t, metricBus.histograms, "s")
	assert.Equal(t, metricBus.histograms.Size(), 0)
}

type recordingAdder struct {
	lastIncr float64
	lastOpts []metric.AddOption
	calls    int
}

func (r *recordingAdder) Add(_ context.Context, incr float64, opts ...metric.AddOption) {
	r.calls++
	r.lastIncr = incr
	r.lastOpts = opts
}

func TestSumWithAttributes(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())
	metricBus := fromCtx(ctx)
	recorder := &recordingAdder{}

	metricBus.sums.Store("with.attrs", recorder)

	attrs := []attribute.KeyValue{attribute.String("key", "val")}

	Sum[int64]("with.attrs").With(attrs...).Inc(ctx)

	assert.Equal(t, 1, recorder.calls)
	assert.Equal(t, 1.0, recorder.lastIncr)
	require.Len(t, recorder.lastOpts, 1)
}

func TestSumWithDoesNotMutateBase(t *testing.T) {
	baseSum := Sum[int64]("mutate")
	attrs := []attribute.KeyValue{attribute.String("foo", "bar")}

	withAttrs := baseSum.With(attrs...)

	assert.Empty(t, baseSum.kvs)
	assert.Equal(t, attrs, withAttrs.kvs)

	second := withAttrs.With(attribute.String("baz", "qux"))

	assert.Equal(t, attrs, withAttrs.kvs)
	assert.Len(t, second.kvs, 2)
}
