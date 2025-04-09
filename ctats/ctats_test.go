package ctats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
)

func TestInitializeNoop(t *testing.T) {
	// should simply work
	_, err := InitializeNoop(context.Background(), t.Name())
	require.NoError(t, err)
}

func TestFormatID(t *testing.T) {
	table := []struct {
		name   string
		in     string
		expect string
	}{
		{
			name:   "empty",
			in:     "",
			expect: "",
		},
		{
			name:   "simple",
			in:     "foobarbaz",
			expect: "foobarbaz",
		},
		{
			name:   "already correct",
			in:     "foo.bar.baz",
			expect: "foo.bar.baz",
		},
		{
			name:   "only underscore delimited",
			in:     "foo_bar_baz",
			expect: "foo_bar_baz",
		},
		{
			name:   "spaces to underscores",
			in:     "foo bar baz",
			expect: "foo_bar_baz",
		},
		{
			name:   "camel case",
			in:     "FooBarBaz",
			expect: "foo.bar.baz",
		},
		{
			name:   "all caps",
			in:     "FOOBARBAZ",
			expect: "foobarbaz",
		},
		{
			name:   "kebab case",
			in:     "foo-bar-baz",
			expect: "foo.bar.baz",
		},
		{
			name:   "mixed",
			in:     "fooBar baz-fnords",
			expect: "foo.bar_baz.fnords",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := formatID(test.in)
			assert.Equal(t, test.expect, result, "input: %s", test.in)
		})
	}
}

func TestInherit(t *testing.T) {
	stubBus1 := &bus{
		counters:   map[string]metric.Float64UpDownCounter{},
		gauges:     map[string]metric.Float64Gauge{},
		histograms: map[string]metric.Float64Histogram{},
	}
	stubBus2 := &bus{}

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
				b := fromCtx(ctx)
				require.Nil(t, b)
			},
		},
		{
			name: "from: background, to: nil",
			from: func() context.Context { return context.Background() },
			to:   func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.Nil(t, b)
			},
		},
		{
			name: "from: nil, to: background",
			from: func() context.Context { return nil },
			to:   func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.Nil(t, b)
			},
		},
		{
			name: "from: background, to: background",
			from: func() context.Context { return context.Background() },
			to:   func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.Nil(t, b)
			},
		},
		{
			name: "from: populated, to: nil",
			from: func() context.Context {
				return embedInCtx(context.Background(), stubBus1)
			},
			to: func() context.Context { return nil },
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.NotNil(t, b)
				assert.Equal(t, stubBus1, b)
			},
		},
		{
			name: "from: populated, to: background",
			from: func() context.Context {
				return embedInCtx(context.Background(), stubBus1)
			},
			to: func() context.Context { return context.Background() },
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.NotNil(t, b)
				assert.Equal(t, stubBus1, b)
			},
		},
		{
			name: "from: nil, to: populated",
			from: func() context.Context { return nil },
			to: func() context.Context {
				return embedInCtx(context.Background(), stubBus1)
			},
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.NotNil(t, b)
				assert.Equal(t, stubBus1, b)
			},
		},
		{
			name: "from: background, to: populated",
			from: func() context.Context { return context.Background() },
			to: func() context.Context {
				return embedInCtx(context.Background(), stubBus1)
			},
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.NotNil(t, b)
				assert.Equal(t, stubBus1, b)
			},
		},
		{
			name: "from: populated, to: populated",
			from: func() context.Context {
				return embedInCtx(context.Background(), stubBus1)
			},
			to: func() context.Context {
				return embedInCtx(context.Background(), stubBus2)
			},
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.NotNil(t, b)
				assert.Equal(t, stubBus2, b)
			},
		},
		{
			name: "from: populated, to: populated, clobbered",
			from: func() context.Context {
				return embedInCtx(context.Background(), stubBus1)
			},
			to: func() context.Context {
				return embedInCtx(context.Background(), stubBus2)
			},
			clobber: true,
			assert: func(t *testing.T, ctx context.Context) {
				b := fromCtx(ctx)
				require.NotNil(t, b)
				assert.Equal(t, stubBus1, b)
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
