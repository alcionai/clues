package clues

import (
	"errors"

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
	// If not provided, a minimal Resource containing the service name will be
	// created.
	Resource *resource.Resource

	// specify the endpoint location to use for grpc communication.
	// If empty, no telemetry exporter will be generated.
	// ex: localhost:4317
	// ex: 0.0.0.0:4317
	// ex: opentelemetry-collector.monitoring.svc.cluster.local:4317
	GRPCEndpoint string
}

// clues.OTELConfig is a passthrough to the internal otel config.
func (oc OTELConfig) toInternalConfig() node.OTELConfig {
	return node.OTELConfig{
		Resource:     oc.Resource,
		GRPCEndpoint: oc.GRPCEndpoint,
	}
}
