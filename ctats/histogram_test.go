package ctats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistogram(t *testing.T) {
	ctx := InitializeNoop(context.Background(), t.Name())

	ctx, err := RegisterHistogram(ctx, "reg.h", "test", "testing histogram")
	require.NoError(t, err)

	metricBus := fromCtx(ctx)

	assert.NotContains(t, metricBus.counters, "reg.h")
	assert.Len(t, metricBus.counters, 0)
	assert.NotContains(t, metricBus.gauges, "reg.h")
	assert.Len(t, metricBus.gauges, 0)
	assert.Contains(t, metricBus.histograms, "reg.h")
	assert.Len(t, metricBus.histograms, 1)

	Histogram[int64]("reg.h").Record(ctx, 1)
	Histogram[float64]("reg.h").Record(ctx, 1)

	assert.NotContains(t, metricBus.counters, "reg.h")
	assert.Len(t, metricBus.counters, 0)
	assert.NotContains(t, metricBus.gauges, "reg.h")
	assert.Len(t, metricBus.gauges, 0)
	assert.Contains(t, metricBus.histograms, "reg.h")
	assert.Len(t, metricBus.histograms, 1)

	Histogram[int8]("h").Record(ctx, 1)
	Histogram[int]("h").Record(ctx, -1)
	Histogram[uint8]("h").Record(ctx, 0)

	assert.NotContains(t, metricBus.counters, "h")
	assert.Len(t, metricBus.counters, 0)
	assert.NotContains(t, metricBus.gauges, "h")
	assert.Len(t, metricBus.gauges, 0)
	assert.Contains(t, metricBus.histograms, "h")
	assert.Len(t, metricBus.histograms, 2)
}
