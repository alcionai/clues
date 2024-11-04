package clog

import (
	"context"
	"testing"

	"github.com/alcionai/clues"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	table := []struct {
		name string
		init func(ctx context.Context) context.Context
		bldr func(ctx context.Context) *builder
	}{
		{
			name: "standard",
			init: func(ctx context.Context) context.Context {
				return Init(
					ctx,
					Settings{}.EnsureDefaults())
			},
			bldr: func(ctx context.Context) *builder {
				return Ctx(ctx)
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			var (
				ctx = test.init(context.Background())
				bld = test.bldr(ctx)
			)

			// standard builder checks
			bld.With("foo", "bar", "baz", 1)
			assert.Contains(t, bld.with, "foo")
			assert.Equal(t, bld.with["foo"].(string), "bar")
			assert.Contains(t, bld.with, "baz")
			assert.Equal(t, bld.with["baz"].(int), 1)

			bld.Label("l1", "l2", "l1")
			assert.Contains(t, bld.labels, "l1")
			assert.Contains(t, bld.labels, "l2")

			bld.Comment("a comment")
			bld.Comment("another comment")
			bld.Comment("a comment")
			assert.Contains(t, bld.comments, "a comment")
			assert.Contains(t, bld.comments, "another comment")

			bld.SkipCaller(1)

			// ensure no collision between separate builders
			// using the same ctx.
			err := clues.New("an error").
				With("fnords", "i have seen them").
				Label("errLabel")

			other := CtxErr(ctx, err)
			assert.Empty(t, other.with)
			assert.Empty(t, other.labels)
			assert.Empty(t, other.comments)
			assert.ErrorIs(t, other.err, err, clues.ToCore(err))

			other.With("foo", "smarf")
			assert.Contains(t, other.with, "foo")
			assert.Equal(t, bld.with["foo"].(string), "bar")

			other.Label("l3")
			assert.Contains(t, other.labels, "l3")
			assert.NotContains(t, bld.labels, "l3")

			other.Comment("comment a")
			assert.Contains(t, other.comments, "comment a")
			assert.NotContains(t, bld.comments, "comment a")

			// ensure no panics when logging
			runDebugLogs(bld)
			runInfoLogs(bld)
			runErrorLogs(bld)
		})
	}
}

func runDebugLogs(
	bld *builder,
) {
	bld.Debug("a", "log")
	bld.Debugf("a %s", "log")
	bld.Debugw("a log", "with key")
	bld.Debugw("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Debugw("a log", "with key", "and value")
}

func runInfoLogs(
	bld *builder,
) {
	bld.Info("a", "log")
	bld.Infof("a %s", "log")
	bld.Infow("a log", "with key")
	bld.Infow("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Infow("a log", "with key", "and value")
}

func runErrorLogs(
	bld *builder,
) {
	bld.Error("a", "log")
	bld.Errorf("a %s", "log")
	bld.Errorw("a log", "with key")
	bld.Errorw("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Errorw("a log", "with key", "and value")
}

func TestGetValue(t *testing.T) {
	var (
		p1 int    = 1
		ps string = "ptr"
		pn any
	)

	table := []struct {
		name     string
		value    any
		expected any
	}{
		{
			name:     "integer",
			value:    1,
			expected: 1,
		},
		{
			name:     "integer value pointer",
			value:    &p1,
			expected: p1,
		},
		{
			name:     "integer value",
			value:    p1,
			expected: p1,
		},
		{
			name: "pointer to integer",
			value: func() *int {
				i := 8
				return &i
			}(),
			expected: 8,
		},
		{
			name:     "string",
			value:    "foo",
			expected: "foo",
		},
		{
			name:     "string value pointer",
			value:    &ps,
			expected: ps,
		},
		{
			name:     "string value",
			value:    ps,
			expected: ps,
		},
		{
			name: "pointer to string",
			value: func() *string {
				s := "foo"
				return &s
			}(),
			expected: "foo",
		},
		{
			name:     "nil",
			value:    nil,
			expected: nil,
		},
		{
			name:     "nil value pointer",
			value:    &pn,
			expected: nil,
		},
		{
			name:     "nil value",
			value:    pn,
			expected: nil,
		},
		{
			name: "nil pointer",
			value: func() *string {
				return nil
			}(),
			expected: nil,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, getValue(test.value))
		})
	}
}
