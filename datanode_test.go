package clues

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

func TestDataNode_Init(t *testing.T) {

	table := []struct {
		name string
		node *dataNode
		ctx  valuer
	}{
		{
			name: "nil ctx",
			node: &dataNode{},
			ctx:  nil,
		},
		{
			name: "nil node",
			node: nil,
			ctx:  context.Background(),
		},
		{
			name: "context.Context",
			node: &dataNode{},
			ctx:  context.Background(),
		},
		{
			name: "workflow.Context",
			node: &dataNode{},
			ctx:  workflowBackground(),
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := test.node.init(test.ctx, test.name, OTELConfig{})
			require.NoError(t, err)
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// temporal has no context.Background() equivalent.  So we have to hack
// in our own version.  That involves registering a fake test workflow
// just to wrap the theft of a workflow context out of it.
func workflowBackground() workflow.Context {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Use a mock workflow to extract the workflow.Context
	var wCtx workflow.Context

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		wCtx = ctx
		return nil
	})

	return wCtx
}
