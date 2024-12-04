package ctats

import (
	"context"
	"testing"

	"github.com/alcionai/clues/internal/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGauge(t *testing.T) {
	noc, err := node.NewOTELClient(
		context.Background(),
		t.Name(),
		node.OTELConfig{})
	require.NoError(t, err)

	ctx := node.EmbedInCtx(context.Background(), &node.Node{OTEL: noc})

	ctx, err = Initialize(ctx)
	require.NoError(t, err)

	ctx, err = RegisterGauge(ctx, "reg.g", "test", "testing gauge")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assert.NotContains(t, metricBus.counters, "reg.g")
	assert.Len(t, metricBus.counters, 0)
	assert.Contains(t, metricBus.gauges, "reg.g")
	assert.Len(t, metricBus.gauges, 1)
	assert.NotContains(t, metricBus.histograms, "reg.g")
	assert.Len(t, metricBus.histograms, 0)

	Gauge[int64]("reg.g").Set(ctx, 1)
	Gauge[float64]("reg.g").Set(ctx, 1)

	assert.NotContains(t, metricBus.counters, "reg.g")
	assert.Len(t, metricBus.counters, 0)
	assert.Contains(t, metricBus.gauges, "reg.g")
	assert.Len(t, metricBus.gauges, 1)
	assert.NotContains(t, metricBus.histograms, "reg.g")
	assert.Len(t, metricBus.histograms, 0)

	Gauge[int8]("g").Set(ctx, 1)
	Gauge[int]("g").Set(ctx, 0)

	assert.NotContains(t, metricBus.counters, "g")
	assert.Len(t, metricBus.counters, 0)
	assert.Contains(t, metricBus.gauges, "g")
	assert.Len(t, metricBus.gauges, 2)
	assert.NotContains(t, metricBus.histograms, "g")
	assert.Len(t, metricBus.histograms, 0)
}
