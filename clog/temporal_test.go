package clog

import (
	"context"
	"testing"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

func TestNewTemporalAdapter(t *testing.T) {
	var (
		ctx            = Init(context.Background(), Settings{})
		ta  log.Logger = NewTemporalAdapter[context.Context](ctx)
	)

	ta = ta.(log.WithLogger).With("foo", "bar", 1)
	ta = ta.(log.WithSkipCallers).WithCallerSkip(5)

	ta.Debug("debug", 1, 2, 3)
	ta.Info("info", 4, 5, 6)
	ta.Warn("warn", 7, 8, 9)
	ta.Error("error", "smurf", "smarf")
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
