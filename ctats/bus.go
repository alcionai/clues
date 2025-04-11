package ctats

import (
	"context"

	"github.com/puzpuzpuz/xsync/v4"
)

type metricsBusKey string

const defaultCtxKey metricsBusKey = "default_metrics_bus_key"

func fromCtx(ctx context.Context) *bus {
	if ctx == nil {
		return nil
	}

	dn := ctx.Value(defaultCtxKey)

	if dn == nil {
		return nil
	}

	return dn.(*bus)
}

func embedInCtx(ctx context.Context, b *bus) context.Context {
	return context.WithValue(ctx, defaultCtxKey, b)
}

type bus struct {
	counters   *xsync.Map[string, adder]
	gauges     *xsync.Map[string, recorder]
	histograms *xsync.Map[string, recorder]

	// initializedToNoop is a testing convenience flag that identifies
	// whether the OTEL client should be configured or not.
	initializedToNoop bool
}

func newBus() *bus {
	return &bus{
		counters:   xsync.NewMap[string, adder](),
		gauges:     xsync.NewMap[string, recorder](),
		histograms: xsync.NewMap[string, recorder](),
	}
}

// counterFromCtx retrieves the counter instance from the metrics bus
// in the context.  If the ctx has no metrics bus, or if the bus does
// not have a counter for the provided ID, returns nil.
func (b bus) getCounter(
	id string,
) adder {
	if b.initializedToNoop {
		c, _ := b.counters.LoadOrStore(formatID(id), &noopAdder{})
		return c
	}

	if b.counters == nil {
		return nil
	}

	c, _ := b.counters.Load(formatID(id))

	return c
}

// gaugeFromCtx retrieves the gauge instance from the metrics bus
// in the context.  If the ctx has no metrics bus, or if the bus does
// not have a gauge for the provided ID, returns nil.
func (b bus) getGauge(
	id string,
) recorder {
	if b.initializedToNoop {
		g, _ := b.gauges.LoadOrStore(formatID(id), &noopRecorder{})
		return g
	}

	if b.gauges == nil {
		return nil
	}

	g, _ := b.gauges.Load(formatID(id))

	return g
}

// histogramFromCtx retrieves the histogram instance from the metrics bus
// in the context.  If the ctx has no metrics bus, or if the bus does
// not have a histogram for the provided ID, returns nil.
func (b bus) getHistogram(
	id string,
) recorder {
	if b.initializedToNoop {
		h, _ := b.histograms.LoadOrStore(formatID(id), &noopRecorder{})
		return h
	}

	if b.histograms == nil {
		return nil
	}

	h, _ := b.histograms.Load(formatID(id))

	return h
}
