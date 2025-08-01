package clog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alcionai/clues/clog"
)

type catcher interface {
	Catch(handler func(ctx context.Context, r any))
}

func TestTryCatch(t *testing.T) {
	table := []struct {
		name  string
		setup func(ctx context.Context) catcher
	}{
		{
			name: "no_configuration",
			setup: func(
				ctx context.Context,
			) catcher {
				return clog.Try(ctx)
			},
		},
		{
			name: "with_configuration",
			setup: func(ctx context.Context) catcher {
				return clog.Try(ctx).
					Label("test").
					Comment("this is a test").
					SkipCaller(1).
					With("k", "v").
					Msg("test msg").
					SetSpanToErr()
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			called := false

			require.NotPanics(t, func() {
				defer test.setup(t.Context()).Catch(
					func(ctx context.Context, r any) {
						called = true

						require.NotNil(t, ctx)
						require.NotNil(t, r)
						require.ErrorIs(t, r.(error), assert.AnError)
					},
				)

				panic(assert.AnError)
			})

			require.True(t, called)
		})
	}
}
