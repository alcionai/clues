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
		bldr func(ctx context.Context) *builder[context.Context]
	}{
		{
			name: "standard",
			init: func(ctx context.Context) context.Context {
				return Init(
					ctx,
					Settings{}.EnsureDefaults())
			},
			bldr: func(ctx context.Context) *builder[context.Context] {
				return Ctx(ctx)
			},
		},
		{
			name: "singleton",
			init: func(ctx context.Context) context.Context {
				return Init(
					ctx,
					Settings{}.EnsureDefaults())
			},
			bldr: func(ctx context.Context) *builder[context.Context] {
				return Singleton()
			},
		},
		{
			name: "singleton, no prior init",
			init: func(ctx context.Context) context.Context {
				return ctx
			},
			bldr: func(ctx context.Context) *builder[context.Context] {
				return Singleton()
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

func (suite *BuilderUnitSuite) TestWith() {
	t := suite.T()

	ctx := Init(
		context.Background(),
		Settings{File: Stderr}.EnsureDefaults())
	ctx = clues.Add(ctx, "a", "b")

	// does not contain clues values yet,
	// this gets added at time of logging.
	bldr := Ctx(ctx)
	assert.Empty(t, bldr.with)

	bldr.With("foo", "bar")
	assert.Equal(
		t,
		map[any]any{
			"foo": "bar",
		},
		bldr.with)

	bldr.With(1)
	assert.Equal(
		t,
		map[any]any{
			"foo": "bar",
			1:     nil,
		},
		bldr.with)

	bldr.Info("should work")
	Flush(ctx)
}

func (suite *BuilderUnitSuite) testDebugLogs(bld *builder[context.Context]) {
	bld.Debug("a", "log")
	bld.Debugf("a %s", "log")
	bld.Debugw("a log", "with key")
	bld.Debugw("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Debugw("a log", "with key", "and value")
}

func (suite *BuilderUnitSuite) testInfoLogs(bld *builder[context.Context]) {
	bld.Info("a", "log")
	bld.Infof("a %s", "log")
	bld.Infow("a log", "with key")
	bld.Infow("a log", "with key", "and value")
	// negative skip caller, just to ensure safety
	bld.
		SkipCaller(-1).
		Infow("a log", "with key", "and value")
}

func (suite *BuilderUnitSuite) testErrorLogs(bld *builder[context.Context]) {
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
