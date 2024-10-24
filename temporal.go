package clues

import (
	"context"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/workflow"
)

// ---------------------------------------------------------------------------
// dataNode ctx propagation
// ---------------------------------------------------------------------------

type dataNodePropagator struct {
	key string
}

func NewDataNodePropagator() workflow.ContextPropagator {
	return &dataNodePropagator{
		key: "clues_data_node_core",
	}
}

// Inject injects values from context into headers for propagation
func (s *dataNodePropagator) Inject(
	ctx context.Context,
	writer workflow.HeaderWriter,
) error {
	node := nodeFromCtx(ctx, defaultNamespace)

	if s != nil && node != nil && len(s.key) > 0 {
		return writeNode(s.key, node, writer)
	}

	return nil
}

// InjectFromWorkflow injects values from context into headers for propagation
func (s *dataNodePropagator) InjectFromWorkflow(
	ctx workflow.Context,
	writer workflow.HeaderWriter,
) error {
	node := nodeFromCtx(ctx, defaultNamespace)

	if s != nil && node != nil && len(s.key) > 0 {
		return writeNode(s.key, node, writer)
	}

	return nil
}

func writeNode(
	key string,
	node *dataNode,
	writer workflow.HeaderWriter,
) error {
	bs, err := node.Bytes()
	if err != nil {
		return Wrap(err, "serializing node to bytes")
	}

	encoded, err := converter.GetDefaultDataConverter().ToPayload(bs)
	if err != nil {
		return Wrap(err, "encoding serialized node")
	}

	// TODO: should probably encrypt this
	writer.Set(key, encoded)

	return nil
}

// Extract extracts values from headers and puts them into context
func (s *dataNodePropagator) Extract(
	ctx context.Context,
	reader workflow.HeaderReader,
) (context.Context, error) {
	node, err := findNodeInReader(s.key, reader)

	if node != nil {
		ctx = setDefaultNodeInCtx(ctx, node)
	}

	return ctx, Stack(err).OrNil()
}

// ExtractToWorkflow extracts values from headers and puts them into context
func (s *dataNodePropagator) ExtractToWorkflow(
	ctx workflow.Context,
	reader workflow.HeaderReader,
) (workflow.Context, error) {
	node, err := findNodeInReader(s.key, reader)

	if node != nil {
		ctx = setDefaultNodeInCtx(ctx, node)
	}

	return ctx, Stack(err).OrNil()
}

func findNodeInReader(
	key string,
	reader workflow.HeaderReader,
) (*dataNode, error) {
	var (
		err  error
		node *dataNode
	)

	iterFn := func(key string, value *commonpb.Payload) error {
		if key == key {
			node, err = readNode(value, reader)
		}

		return nil
	}

	err = reader.ForEachKey(iterFn)

	return node, Stack(err)
}

func readNode(
	value *commonpb.Payload,
	reader workflow.HeaderReader,
) (*dataNode, error) {
	var bs []byte

	err := converter.GetDefaultDataConverter().FromPayload(value, &bs)
	if err != nil {
		return nil, Wrap(err, "retrieving node bytes from payload")
	}

	node, err := FromBytes(bs)
	if err != nil {
		return nil, Wrap(err, "building node from bytes")
	}

	return node, nil
}

// ---------------------------------------------------------------------------
// otel tracing interceptor
// ---------------------------------------------------------------------------

// newTemporalTracingInterceptor generates the temporal trace interceptor.
func newTemporalTracingInterceptor() (interceptor.Interceptor, error) {
	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	return tracingInterceptor, Wrap(err, "creating temporal otel interceptor").OrNil()
}
