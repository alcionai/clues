package clues

import "github.com/alcionai/clues/internal/node"

var ErrMissingOtelGRPCEndpoint = New("missing otel grpc endpoint").NoTrace()

const (
	DefaultOTELGRPCEndpoint = "localhost:4317"
)

// ConfigOTEL generates a new otel configuration that can be passed
// to clues.Initialize(ctx, cfg).
func ConfigOTEL(
	// grpcEndpoint is the grpc endpoint of your collector.  Probably localhost:4317.
	grpcEndpoint string,
) (node.OTELConfig, error) {
	if len(grpcEndpoint) == 0 {
		return node.OTELConfig{}, ErrMissingOtelGRPCEndpoint
	}

	return node.OTELConfig{
		GRPCEndpoint: grpcEndpoint,
	}, nil
}
