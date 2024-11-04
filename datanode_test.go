package clues

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

func TestDataNode_Init(t *testing.T) {

	table := []struct {
		name string
		node *dataNode
		ctx  context.Context
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
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := test.node.init(test.ctx, test.name, OTELConfig{})
			require.NoError(t, err)
		})
	}
}
