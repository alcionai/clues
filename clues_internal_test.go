package clues

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alcionai/clues/internal/node"
)

func TestInherit(t *testing.T) {
	table := []struct {
		name    string
		from    func() context.Context
		to      func() context.Context
		clobber bool
		assert  func(t *testing.T, ctx context.Context)
	}{
		{
			name: "from: nil, to: nil",
			from: func() context.Context { return nil },
			to:   func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.Nil(t, n.OTEL)
			},
		},
		{
			name: "from: background, to: nil",
			from: func() context.Context { return context.Background() },
			to:   func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.Nil(t, n.OTEL)
			},
		},
		{
			name: "from: nil, to: background",
			from: func() context.Context { return nil },
			to:   func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.Nil(t, n.OTEL)
			},
		},
		{
			name: "from: background, to: background",
			from: func() context.Context { return context.Background() },
			to:   func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.Nil(t, n.OTEL)
			},
		},
		{
			name: "from: populated, to: nil",
			from: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "test",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			to: func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.NotNil(t, n.OTEL)
				assert.Equal(t, "test", n.OTEL.ServiceName)
			},
		},
		{
			name: "from: populated, to: background",
			from: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "test",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			to: func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.NotNil(t, n.OTEL)
				assert.Equal(t, "test", n.OTEL.ServiceName)
			},
		},
		{
			name: "from: nil, to: populated",
			from: func() context.Context { return nil },
			to: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "to",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.NotNil(t, n.OTEL)
				assert.Equal(t, "to", n.OTEL.ServiceName)
			},
		},
		{
			name: "from: background, to: populated",
			from: func() context.Context { return context.Background() },
			to: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "to",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.NotNil(t, n.OTEL)
				assert.Equal(t, "to", n.OTEL.ServiceName)
			},
		},
		{
			name: "from: populated, to: populated",
			from: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "from",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			to: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "to",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.NotNil(t, n.OTEL)
				assert.Equal(t, "to", n.OTEL.ServiceName)
			},
		},
		{
			name: "from: populated, to: populated, clobbered",
			from: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "from",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			to: func() context.Context {
				n := node.Node{
					OTEL: &node.OTELClient{
						ServiceName: "to",
					},
				}

				return node.EmbedInCtx(context.Background(), &n)
			},
			clobber: true,
			assert: func(t *testing.T, ctx context.Context) {
				n := node.FromCtx(ctx)
				require.NotNil(t, n.OTEL)
				assert.Equal(t, "from", n.OTEL.ServiceName)
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := Inherit(test.from(), test.to(), test.clobber)
			require.NotNil(t, result)
			test.assert(t, result)
		})
	}
}
