package clog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInherit(t *testing.T) {
	stubClogger1 := &clogger{
		set: Settings{
			Level: "test",
		},
	}
	stubClogger2 := &clogger{}

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
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.False(t, found)
			},
		},
		{
			name: "from: background, to: nil",
			from: func() context.Context { return context.Background() },
			to:   func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.False(t, found)
			},
		},
		{
			name: "from: nil, to: background",
			from: func() context.Context { return nil },
			to:   func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.False(t, found)
			},
		},
		{
			name: "from: background, to: background",
			from: func() context.Context { return context.Background() },
			to:   func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.False(t, found)
			},
		},
		{
			name: "from: populated, to: nil",
			from: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger1)
			},
			to: func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.True(t, found)
				assert.Equal(t, stubClogger1, logger)
			},
		},
		{
			name: "from: populated, to: background",
			from: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger1)
			},
			to: func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.True(t, found)
				assert.Equal(t, stubClogger1, logger)
			},
		},
		{
			name: "from: nil, to: populated",
			from: func() context.Context { return nil },
			to: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger1)
			},
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.True(t, found)
				assert.Equal(t, stubClogger1, logger)
			},
		},
		{
			name: "from: background, to: populated",
			from: func() context.Context { return context.Background() },
			to: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger1)
			},
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.True(t, found)
				assert.Equal(t, stubClogger1, logger)
			},
		},
		{
			name: "from: populated, to: populated",
			from: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger1)
			},
			to: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger2)
			},
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.True(t, found)
				assert.Equal(t, stubClogger2, logger)
			},
		},
		{
			name: "from: populated, to: populated, clobbered",
			from: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger1)
			},
			to: func() context.Context {
				return plantLoggerInCtx(context.Background(), stubClogger2)
			},
			clobber: true,
			assert: func(t *testing.T, ctx context.Context) {
				logger, found := fromCtx(ctx)
				require.NotNil(t, logger)
				assert.True(t, found)
				assert.Equal(t, stubClogger1, logger)
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
