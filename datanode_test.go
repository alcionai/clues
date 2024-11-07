package clues

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestBytes(t *testing.T) {
	table := []struct {
		name                 string
		node                 func() *dataNode
		expectSerialized     []byte
		expectDeserialized   *dataNode
		expectDeserializeErr require.ErrorAssertionFunc
	}{
		{
			name: "nil",
			node: func() *dataNode {
				return nil
			},
			expectSerialized:     []byte{},
			expectDeserialized:   nil,
			expectDeserializeErr: require.Error,
		},
		{
			name: "empty",
			node: func() *dataNode {
				return &dataNode{}
			},
			expectSerialized:     []byte(`{"otelServiceName":"","values":{},"comments":[]}`),
			expectDeserialized:   &dataNode{},
			expectDeserializeErr: require.NoError,
		},
		{
			name: "with values",
			node: func() *dataNode {
				return &dataNode{
					otel: &otelClient{
						serviceName: "serviceName",
					},
					values: map[string]any{
						"fisher":  "flannigan",
						"fitzbog": nil,
					},
					comment: comment{
						Caller:  "i am caller",
						File:    "i am file",
						Message: "i am message",
					},
				}
			},
			expectSerialized: []byte(`{"otelServiceName":"serviceName",` +
				`"values":{"fisher":"flannigan","fitzbog":""},` +
				`"comments":[{"Caller":"i am caller","File":"i am file","Message":"i am message"}]}`),
			expectDeserialized: &dataNode{
				otel: &otelClient{
					serviceName: "serviceName",
				},
				values: map[string]any{
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
