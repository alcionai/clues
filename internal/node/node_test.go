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

func TestNode_InitOTEL(t *testing.T) {
	table := []struct {
		name    string
		node    *Node
		ctx     context.Context
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:    "nil ctx",
			node:    &Node{},
			ctx:     nil,
			wantErr: require.Error,
		},
		{
			name:    "nil node",
			node:    nil,
			ctx:     context.Background(),
			wantErr: require.NoError,
		},
		{
			name:    "context.Context",
			node:    &Node{},
			ctx:     context.Background(),
			wantErr: require.NoError,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := test.node.InitOTEL(test.ctx, test.name, OTELConfig{})
			test.wantErr(t, err)
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
						ServiceName: "serviceName",
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
					ServiceName: "serviceName",
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

func TestInheritAttrs(t *testing.T) {
	var (
		ctx    = t.Context()
		fooBar = map[string]any{"foo": "bar"}
		baz    = map[string]any{"baz": 42}
		alt    = EmbedInCtx(
			ctx,
			(&Node{}).AddValues(ctx, baz),
		)
	)

	ctx = EmbedInCtx(
		ctx,
		(&Node{}).AddValues(ctx, fooBar),
	)

	table := []struct {
		name    string
		from    context.Context
		to      context.Context
		clobber bool
		want    map[string]any
	}{
		{
			name: "from: nil, to: nil",
			from: nil,
			to:   nil,
			want: map[string]any{},
		},
		{
			name: "from: ctx, to: nil",
			from: ctx,
			to:   nil,
			want: fooBar,
		},
		{
			name: "from: nil, to: ctx",
			from: nil,
			to:   ctx,
			want: fooBar,
		},
		{
			name: "from: ctx, to: alt, no clobber",
			from: ctx,
			to:   alt,
			want: baz,
		},
		{
			name:    "from: ctx, to: alt, clobber",
			from:    ctx,
			to:      alt,
			clobber: true,
			want:    fooBar,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			n := FromCtx(test.to)
			require.NotNil(t, n)

			result := InheritAttrs(test.from, test.to, test.clobber)
			require.NotNil(t, result)

			assert.Equal(t, test.want, FromCtx(result).Map())
		})
	}
}
