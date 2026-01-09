package clutel

import (
	"errors"

	"go.opentelemetry.io/contrib/processors/baggagecopy"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/alcionai/clues/internal/node"
)

var ErrMissingOtelGRPCEndpoint = errors.New("missing otel grpc endpoint")

const (
	DefaultOTELGRPCEndpoint = "localhost:4317"
)

type OTELConfig struct {
	// Resource contains information about the thing sourcing logs, metrics, and
	// traces in OTEL. This information will be available in backends on all logs
	// traces, and metrics that are generated from this source.
	//
	// The provided resource should represent the service that's initializing
	// clues. The resource should encapsulate all parts of the metrics that need
	// reporting, not just a subset of them (i.e. it represents the "root" of the
	// information that will be reported to OTEL).
	//
	// If not provided, a minimal Resource containing the service name will be
	// created.
	Resource *resource.Resource

	// specify the endpoint location to use for grpc communication.
	// If empty, no telemetry exporter will be generated.
	// ex: localhost:4317
	// ex: 0.0.0.0:4317
	// ex: opentelemetry-collector.monitoring.svc.cluster.local:4317
	GRPCEndpoint string

	// Filter contains the filter used when copying baggage to a span, by adding span
	// attributes. If no filter is specified, all baggage is copied over to a span.
	Filter baggagecopy.Filter

	// meterExporterOpts contains options to apply to the meter provider's default reader.
	meterExporterOpts []otlpmetricgrpc.Option
}

// NewConfig creates a new OTELConfig with the given parameters and options.  All
// instruments are constructed with default (sensible) behaviors.  Overrides and
// configurations can be applied via options.
func NewConfig(
	rsrc *resource.Resource,
	grpcEndpoint string,
	opts ...option,
) OTELConfig {
	oc := &OTELConfig{
		Resource:     rsrc,
		GRPCEndpoint: grpcEndpoint,
	}

	oc.applyOptions(opts...)

	return *oc
}

type option func(o *OTELConfig)

func (oc *OTELConfig) applyOptions(opts ...option) {
	for _, o := range opts {
		o(oc)
	}
}

// WithDeltaTemporalityMeter sets the OTLP metric exporter to use delta temporality
// for the MeterProvider instrument.  This only affects Sum (ie: OTELCounter) type
// metric aggregations.
func WithDeltaTemporalityMeter() option {
	return func(o *OTELConfig) {
		o.meterExporterOpts = append(
			o.meterExporterOpts,
			otlpmetricgrpc.WithTemporalitySelector(sdkMetric.DeltaTemporalitySelector),
		)
	}
}

// BlockAllBaggage is a filter which blocks copying of all members of
// baggage to a span.
var BlockAllBaggage baggagecopy.Filter = func(baggage.Member) bool { return false }

// WithBlockAllBaggage sets the OTELConfig to block copying all memebers of baggage to
// a span.
func WithBlockAllBaggage() option {
	return func(o *OTELConfig) {
		o.Filter = BlockAllBaggage
	}
}

// AllowAllBaggage is a filter which allows copying of all members of
// baggage to a span.
var AllowAllBaggage baggagecopy.Filter = func(baggage.Member) bool { return true }

// WithAllowAllBaggage sets the OTELConfig to allow copying all memebers of baggage to
// a span.
func WithAllowAllBaggage() option {
	return func(o *OTELConfig) {
		o.Filter = AllowAllBaggage
	}
}

// clues.OTELConfig is a passthrough to the internal otel config.
func (oc OTELConfig) toInternalConfig() node.OTELConfig {
	return node.OTELConfig{
		Resource:          oc.Resource,
		GRPCEndpoint:      oc.GRPCEndpoint,
		Filter:            oc.Filter,
		MeterExporterOpts: oc.meterExporterOpts,
	}
}
