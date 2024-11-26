package node

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// tests
// ---------------------------------------------------------------------------

func TestNode_Init(t *testing.T) {
	table := []struct {
		name string
		node *Node
		ctx  context.Context
	}{
		{
			name: "nil ctx",
			node: &Node{},
			ctx:  nil,
		},
		{
			name: "nil node",
			node: nil,
			ctx:  context.Background(),
		},
		{
			name: "context.Context",
			node: &Node{},
			ctx:  context.Background(),
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := test.node.InitOTEL(test.ctx, test.name, OTELConfig{})
			require.NoError(t, err)
		})
	}
}

func TestBytes(t *testing.T) {
	table := []struct {
		name                 string
		node                 func() *Node
		expectSerialized     []byte
		expectDeserialized   *Node
		expectDeserializeErr require.ErrorAssertionFunc
	}{
		{
			name: "nil",
			node: func() *Node {
				return nil
			},
			expectSerialized:     []byte{},
			expectDeserialized:   nil,
			expectDeserializeErr: require.Error,
		},
		{
			name: "empty",
			node: func() *Node {
				return &Node{}
			},
			expectSerialized:     []byte(`{"otelServiceName":"","values":{},"comments":[]}`),
			expectDeserialized:   &Node{},
			expectDeserializeErr: require.NoError,
		},
		{
			name: "with values",
			node: func() *Node {
				return &Node{
					OTEL: &OTELClient{
						serviceName: "serviceName",
					},
					Values: map[string]any{
						"fisher":  "flannigan",
						"fitzbog": nil,
					},
					Comment: Comment{
						Caller:  "i am caller",
						File:    "i am file",
						Message: "i am message",
					},
				}
			},
			expectSerialized: []byte(`{"otelServiceName":"serviceName",` +
				`"values":{"fisher":"flannigan","fitzbog":""},` +
				`"comments":[{"Caller":"i am caller","File":"i am file","Message":"i am message"}]}`),
			expectDeserialized: &Node{
				OTEL: &OTELClient{
					serviceName: "serviceName",
				},
				Values: map[string]any{
					"fisher":  "flannigan",
					"fitzbog": "",
				},
			},
			expectDeserializeErr: require.NoError,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			serialized, err := test.node().Bytes()
			require.NoError(t, err)

			assert.Equalf(
				t,
				test.expectSerialized,
				serialized,
				"expected:\t%s\ngot:\t\t%s\n",
				string(test.expectSerialized),
				string(serialized))

			deserialized, err := FromBytes(serialized)
			test.expectDeserializeErr(t, err)
			require.Equal(t, test.expectDeserialized, deserialized)
		})
	}
}
