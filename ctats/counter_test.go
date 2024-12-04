package ctats

import (
	"context"
	"testing"

	"github.com/alcionai/clues/internal/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounter(t *testing.T) {
	noc, err := node.NewOTELClient(
		context.Background(),
		t.Name(),
		node.OTELConfig{})
	require.NoError(t, err)

	ctx := node.EmbedInCtx(context.Background(), &node.Node{OTEL: noc})

	ctx, err = Initialize(ctx)
	require.NoError(t, err)

	ctx, err = RegisterCounter(ctx, "reg.c", "test", "testing counter")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assert.Contains(t, metricBus.counters, "reg.c")
	assert.Len(t, metricBus.counters, 1)
	assert.NotContains(t, metricBus.gauges, "reg.c")
	assert.Len(t, metricBus.gauges, 0)
	assert.NotContains(t, metricBus.histograms, "reg.c")
	assert.Len(t, metricBus.histograms, 0)

	Counter[int64]("reg.c").Add(ctx, 1)
	Counter[float64]("reg.c").Add(ctx, 1)
	Counter[int64]("reg.c").Inc(ctx)
	Counter[float64]("reg.c").Inc(ctx)
	Counter[int64]("reg.c").Dec(ctx)
	Counter[float64]("reg.c").Dec(ctx)

	assert.Contains(t, metricBus.counters, "reg.c")
	assert.Len(t, metricBus.counters, 1)
	assert.NotContains(t, metricBus.gauges, "reg.c")
	assert.Len(t, metricBus.gauges, 0)
	assert.NotContains(t, metricBus.histograms, "reg.c")
	assert.Len(t, metricBus.histograms, 0)

	Counter[int8]("c").Add(ctx, 1)
	Counter[float32]("c").Inc(ctx)
	Counter[uint16]("c").Dec(ctx)
	Counter[int]("c").Dec(ctx)

	assert.Contains(t, metricBus.counters, "c")
	assert.Len(t, metricBus.counters, 2)
	assert.NotContains(t, metricBus.gauges, "c")
	assert.Len(t, metricBus.gauges, 0)
	assert.NotContains(t, metricBus.histograms, "c")
	assert.Len(t, metricBus.histograms, 0)
}
