package clues

import (
	"errors"

	"go.opentelemetry.io/contrib/processors/baggagecopy"
	"go.opentelemetry.io/otel/baggage"
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
}

// clues.OTELConfig is a passthrough to the internal otel config.
func (oc OTELConfig) toInternalConfig() node.OTELConfig {
	return node.OTELConfig{
		Resource:     oc.Resource,
		GRPCEndpoint: oc.GRPCEndpoint,
		Filter:       oc.Filter,
	}
}

// BlockAllMembers is a filter which blocks copying of all members of
// baggage to a span.
var BlockAllMembers baggagecopy.Filter = func(baggage.Member) bool { return false }
