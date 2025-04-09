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

	assertNotContains(t, metricBus.counters, "reg.h")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "reg.h")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertContains(t, metricBus.histograms, "reg.h")
	assert.Equal(t, metricBus.histograms.Size(), 1)

	Histogram[int64]("reg.h").Record(ctx, 1)
	Histogram[float64]("reg.h").Record(ctx, 1)

	assertNotContains(t, metricBus.counters, "reg.h")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "reg.h")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertContains(t, metricBus.histograms, "reg.h")
	assert.Equal(t, metricBus.histograms.Size(), 1)

	Histogram[int8]("h").Record(ctx, 1)
	Histogram[int]("h").Record(ctx, -1)
	Histogram[uint8]("h").Record(ctx, 0)

	assertNotContains(t, metricBus.counters, "h")
	assert.Equal(t, metricBus.counters.Size(), 0)
	assertNotContains(t, metricBus.gauges, "h")
	assert.Equal(t, metricBus.gauges.Size(), 0)
	assertContains(t, metricBus.histograms, "h")
	assert.Equal(t, metricBus.histograms.Size(), 2)
}
