package ctats

import (
	"context"
	"testing"

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
