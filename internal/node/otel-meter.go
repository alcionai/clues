package node

import "go.opentelemetry.io/otel/metric"

// Meterer is a copy of the OTEL Meter interface with the embedded.Meter
// removed for testability.
type Meterer interface {
	// Int64Counter returns a new Int64Counter instrument identified by name
	// and configured with options. The instrument is used to synchronously
	// record increasing int64 measurements during a computational operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error)

	// Int64UpDownCounter returns a new Int64UpDownCounter instrument
	// identified by name and configured with options. The instrument is used
	// to synchronously record int64 measurements during a computational
	// operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64UpDownCounter(name string, options ...metric.Int64UpDownCounterOption) (metric.Int64UpDownCounter, error)

	// Int64Histogram returns a new Int64Histogram instrument identified by
	// name and configured with options. The instrument is used to
	// synchronously record the distribution of int64 measurements during a
	// computational operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64Histogram(name string, options ...metric.Int64HistogramOption) (metric.Int64Histogram, error)

	// Int64Gauge returns a new Int64Gauge instrument identified by name and
	// configured with options. The instrument is used to synchronously record
	// instantaneous int64 measurements during a computational operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64Gauge(name string, options ...metric.Int64GaugeOption) (metric.Int64Gauge, error)

	// Int64ObservableCounter returns a new Int64ObservableCounter identified
	// by name and configured with options. The instrument is used to
	// asynchronously record increasing int64 measurements once per a
	// measurement collection cycle.
	//
	// Measurements for the returned instrument are made via a callback. Use
	// the WithInt64Callback option to register the callback here, or use the
	// RegisterCallback method of this Meter to register one later. See the
	// Measurements section of the package documentation for more information.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64ObservableCounter(name string, options ...metric.Int64ObservableCounterOption) (metric.Int64ObservableCounter, error)

	// Int64ObservableUpDownCounter returns a new Int64ObservableUpDownCounter
	// instrument identified by name and configured with options. The
	// instrument is used to asynchronously record int64 measurements once per
	// a measurement collection cycle.
	//
	// Measurements for the returned instrument are made via a callback. Use
	// the WithInt64Callback option to register the callback here, or use the
	// RegisterCallback method of this Meter to register one later. See the
	// Measurements section of the package documentation for more information.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64ObservableUpDownCounter(name string, options ...metric.Int64ObservableUpDownCounterOption) (metric.Int64ObservableUpDownCounter, error)

	// Int64ObservableGauge returns a new Int64ObservableGauge instrument
	// identified by name and configured with options. The instrument is used
	// to asynchronously record instantaneous int64 measurements once per a
	// measurement collection cycle.
	//
	// Measurements for the returned instrument are made via a callback. Use
	// the WithInt64Callback option to register the callback here, or use the
	// RegisterCallback method of this Meter to register one later. See the
	// Measurements section of the package documentation for more information.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Int64ObservableGauge(name string, options ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error)

	// Float64Counter returns a new Float64Counter instrument identified by
	// name and configured with options. The instrument is used to
	// synchronously record increasing float64 measurements during a
	// computational operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error)

	// Float64UpDownCounter returns a new Float64UpDownCounter instrument
	// identified by name and configured with options. The instrument is used
	// to synchronously record float64 measurements during a computational
	// operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64UpDownCounter(name string, options ...metric.Float64UpDownCounterOption) (metric.Float64UpDownCounter, error)

	// Float64Histogram returns a new Float64Histogram instrument identified by
	// name and configured with options. The instrument is used to
	// synchronously record the distribution of float64 measurements during a
	// computational operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error)

	// Float64Gauge returns a new Float64Gauge instrument identified by name and
	// configured with options. The instrument is used to synchronously record
	// instantaneous float64 measurements during a computational operation.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64Gauge(name string, options ...metric.Float64GaugeOption) (metric.Float64Gauge, error)

	// Float64ObservableCounter returns a new Float64ObservableCounter
	// instrument identified by name and configured with options. The
	// instrument is used to asynchronously record increasing float64
	// measurements once per a measurement collection cycle.
	//
	// Measurements for the returned instrument are made via a callback. Use
	// the WithFloat64Callback option to register the callback here, or use the
	// RegisterCallback method of this Meter to register one later. See the
	// Measurements section of the package documentation for more information.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64ObservableCounter(name string, options ...metric.Float64ObservableCounterOption) (metric.Float64ObservableCounter, error)

	// Float64ObservableUpDownCounter returns a new
	// Float64ObservableUpDownCounter instrument identified by name and
	// configured with options. The instrument is used to asynchronously record
	// float64 measurements once per a measurement collection cycle.
	//
	// Measurements for the returned instrument are made via a callback. Use
	// the WithFloat64Callback option to register the callback here, or use the
	// RegisterCallback method of this Meter to register one later. See the
	// Measurements section of the package documentation for more information.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64ObservableUpDownCounter(name string, options ...metric.Float64ObservableUpDownCounterOption) (metric.Float64ObservableUpDownCounter, error)

	// Float64ObservableGauge returns a new Float64ObservableGauge instrument
	// identified by name and configured with options. The instrument is used
	// to asynchronously record instantaneous float64 measurements once per a
	// measurement collection cycle.
	//
	// Measurements for the returned instrument are made via a callback. Use
	// the WithFloat64Callback option to register the callback here, or use the
	// RegisterCallback method of this Meter to register one later. See the
	// Measurements section of the package documentation for more information.
	//
	// The name needs to conform to the OpenTelemetry instrument name syntax.
	// See the Instrument Name section of the package documentation for more
	// information.
	Float64ObservableGauge(name string, options ...metric.Float64ObservableGaugeOption) (metric.Float64ObservableGauge, error)

	// RegisterCallback registers f to be called during the collection of a
	// measurement cycle.
	//
	// If Unregister of the returned Registration is called, f needs to be
	// unregistered and not called during collection.
	//
	// The instruments f is registered with are the only instruments that f may
	// observe values for.
	//
	// If no instruments are passed, f should not be registered nor called
	// during collection.
	//
	// The function f needs to be concurrent safe.
	RegisterCallback(f metric.Callback, instruments ...metric.Observable) (metric.Registration, error)
}
