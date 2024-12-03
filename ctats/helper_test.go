package ctats

import (
	"go.opentelemetry.io/otel/metric"

	"github.com/alcionai/clues/internal/node"
)

var _ node.Meterer = &meter{}

type meter struct {
	err error
}

func (m meter) meter() {}

// embedded.Meter
func (m meter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, m.err
}

func (m meter) Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error) {
	return nil, m.err
}

func (m meter) Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	return nil, m.err
}

func (m meter) Int64Gauge(name string, options ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	return nil, m.err
}

func (m meter) Int64ObservableCounter(name string, options ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error) {
	return nil, m.err
}

func (m meter) Int64ObservableUpDownCounter(name string, options ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error) {
	return nil, m.err
}

func (m meter) Int64ObservableGauge(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	return nil, m.err
}

func (m meter) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	return nil, m.err
}

func (m meter) Float64Gauge(name string, options ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	return nil, m.err
}

func (m meter) Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error) {
	return nil, m.err
}

func (m meter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return nil, m.err
}

func (m meter) Float64ObservableCounter(name string, options ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error) {
	return nil, m.err
}

func (m meter) Float64ObservableUpDownCounter(name string, options ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error) {
	return nil, m.err
}

func (m meter) Float64ObservableGauge(name string, options ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error) {
	return nil, m.err
}

func (m meter) RegisterCallback(f metric.Callback, instruments ...metric.Observable) (metric.Registration, error) {
	return nil, m.err
}
