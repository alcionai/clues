package clog

import (
	"context"
	"testing"

	"github.com/alcionai/clues"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BuilderUnitSuite struct {
	suite.Suite
}

func TestBuilderUnitSuite(t *testing.T) {
	suite.Run(t, new(BuilderUnitSuite))
}

func (suite *BuilderUnitSuite) TestBuilder() {
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
		var (
			t   = suite.T()
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
		suite.testDebugLogs(bld)
		suite.testInfoLogs(bld)
		suite.testErrorLogs(bld)
	}
}

func (suite *BuilderUnitSuite) testDebugLogs(bld *builder) {
	bld.Debug("a", "log")
	bld.Debugf("a %s", "log")
	bld.Debugw("a log", "with key")
	bld.Debugw("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Debugw("a log", "with key", "and value")
}

func (suite *BuilderUnitSuite) testInfoLogs(bld *builder) {
	bld.Info("a", "log")
	bld.Infof("a %s", "log")
	bld.Infow("a log", "with key")
	bld.Infow("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Infow("a log", "with key", "and value")
}

func (suite *BuilderUnitSuite) testErrorLogs(bld *builder) {
	bld.Error("a", "log")
	bld.Errorf("a %s", "log")
	bld.Errorw("a log", "with key")
	bld.Errorw("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Errorw("a log", "with key", "and value")
}

func (suite *BuilderUnitSuite) TestGetValue() {
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
			name:     "string",
			value:    "foo",
			expected: "foo",
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
			name: "nil pointer",
			value: func() *string {
				return nil
			}(),
			expected: nil,
		},
	}

	for _, tt := range table {
		suite.T().Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, getValue(tt.value))
		})
	}
}
