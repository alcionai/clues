package ctats

import (
	"context"
	"testing"

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
