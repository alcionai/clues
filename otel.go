package clues

import (
	"errors"

	"github.com/alcionai/clues/internal/node"
)

var ErrMissingOtelGRPCEndpoint = errors.New("missing otel grpc endpoint")

const (
	DefaultOTELGRPCEndpoint = "localhost:4317"
)

type OTELConfig struct {
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
		GRPCEndpoint: oc.GRPCEndpoint,
	}
}
