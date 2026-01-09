package ctats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
