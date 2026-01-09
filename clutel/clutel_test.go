package clutel_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"

	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/clutel"
	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/tester"
)

// TestStartSpan_Uninitialized ensures nothing panics if AddSpan is called and
// neither clues nor OTEL is initialized.
func TestStartSpan_Uninitialized(t *testing.T) {
	assert.NotPanics(
		t,
		func() {
			clutel.StartSpan(t.Context(), "test span")
		},
	)
}

// TestStartSpan_Uninitialized_Concurrent ensures that even if OTEL isn't
// initialized there's no race condition when attempting to add spans to a
// parent context concurrently.
func TestStartSpan_Uninitialized_Concurrent(t *testing.T) {
	table := []struct {
		name  string
		attrs []any
	}{
		{
			name: "NoAttributes",
		},
		{
			name:  "Attributes",
			attrs: []any{"key", "value"},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			var (
				wg sync.WaitGroup
				c  = make(chan struct{})
			)

			ctx := clutel.StartSpan(t.Context(), "parent span", "some", "value")

			for range 5 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					<-c

					ctx := clutel.StartSpan(ctx, "worker span", test.attrs...)
					defer clutel.EndSpan(ctx)
				}()
			}

			time.Sleep(500 * time.Millisecond)

			close(c)

			wg.Wait()
		})
	}
}

func TestStartSpan(t *testing.T) {
	table := []struct {
		name        string
		names       []string
		expectTrace string
		kvs         tester.SA
		expectM     tester.MSA
		expectS     tester.SA
	}{
		{
			"single",
			[]string{"single"},
			"single",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"multiple",
			[]string{"single", "multiple"},
			"single,multiple",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"duplicates",
			[]string{"single", "multiple", "multiple"},
			"single,multiple,multiple",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"single with kvs",
			[]string{"single"},
			"single",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"multiple with kvs",
			[]string{"single", "multiple"},
			"single,multiple",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"duplicates with kvs",
			[]string{"single", "multiple", "multiple"},
			"single,multiple,multiple",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := context.Background()

				if init {
					ocfg := clutel.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clutel.Init(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					ctx = ictx
				}

				ctx = context.WithValue(ctx, tester.StubCtx{}, "instance")

				tester.MustEquals(
					t,
					tester.MSA{},
					node.FromCtx(ctx).Map(),
					false,
				)

				for _, name := range test.names {
					ctx = clutel.StartSpan(ctx, name, test.kvs...)
					defer clutel.EndSpan(ctx)
				}

				tester.AssertEq(
					ctx,
					t,
					"",
					test.expectM, tester.MSA{},
					test.expectS, tester.SA{},
				)

				c := node.FromCtx(ctx).Map()
				if c["clues_trace"] != test.expectTrace {
					t.Errorf(
						"expected clues_trace to equal %q, got %q",
						test.expectTrace,
						c["clues_trace"],
					)
				}
			})
		}
	}
}

func TestNewSpan(t *testing.T) {
	table := []struct {
		name    string
		kvs     tester.SA
		expectM tester.MSA
		expectS tester.SA
	}{
		{
			"empty",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"with_attrs",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"with_attrs_odd",
			tester.SA{"k"},
			tester.MSA{"k": nil},
			tester.SA{"k", nil},
		},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := context.Background()

				if init {
					ocfg := clutel.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clutel.Init(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					ctx = ictx
				}

				ctx = context.WithValue(ctx, tester.StubCtx{}, "instance")

				tester.MustEquals(
					t,
					tester.MSA{},
					node.FromCtx(ctx).Map(),
					false,
				)

				ctx = clutel.NewSpan().
					WithAttrs(test.kvs...).
					WithOpts(
						trace.WithSpanKind(trace.SpanKindInternal),
						trace.WithAttributes(attribute.String("clues_trace", test.name)),
					).
					Start(ctx, test.name)
				defer clutel.EndSpan(ctx)

				tester.AssertEq(
					ctx,
					t,
					"",
					test.expectM, tester.MSA{},
					test.expectS, tester.SA{},
				)
			})
		}
	}
}

// TestNewSpan_Uninitialized_Concurrent ensures that even if OTEL isn't
// initialized there's no race condition when attempting to add spans to a
// parent context concurrently.
func TestNewSpan_Uninitialized_Concurrent(t *testing.T) {
	table := []struct {
		name  string
		attrs []any
	}{
		{
			name: "NoAttributes",
		},
		{
			name:  "Attributes",
			attrs: []any{"key", "value"},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			var (
				wg sync.WaitGroup
				c  = make(chan struct{})
			)

			ctx := clutel.NewSpan().
				WithAttrs("some", "value").
				Start(t.Context(), "parent span")

			for range 5 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					<-c

					ctx := clutel.NewSpan().
						WithAttrs(test.attrs...).
						Start(ctx, "worker span")

					defer clutel.EndSpan(ctx)
				}()
			}

			time.Sleep(500 * time.Millisecond)

			close(c)

			wg.Wait()
		})
	}
}

func TestAddBaggage(t *testing.T) {
	ctx, err := clutel.Init(
		t.Context(),
		"test",
		clutel.OTELConfig{},
	)
	require.NoError(t, err)

	table := []struct {
		name    string
		kvs     tester.SA
		expectM tester.MSA
		expectS tester.SA
	}{
		{
			"empty",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"with_attrs",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"with_attrs_odd",
			tester.SA{"k"},
			tester.MSA{"k": nil},
			tester.SA{"k", nil},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx, err := clutel.AddBaggage(ctx, test.kvs...)
			require.NoError(t, err, cluerr.ToCore(err))

			tester.AssertEq(
				ctx,
				t,
				"",
				test.expectM, tester.MSA{},
				test.expectS, tester.SA{},
			)

			bags := baggage.FromContext(ctx)
			kvs := tester.MSA{}

			for _, member := range bags.Members() {
				kvs[member.Key()] = member.Value()
			}
		})
	}
}

func TestNewBaggageProps(t *testing.T) {
	ctx, err := clutel.Init(
		t.Context(),
		"test",
		clutel.OTELConfig{},
	)
	require.NoError(t, err)

	table := []struct {
		name    string
		input   clutel.BaggageProps
		expectM tester.MSA
		expectS tester.SA
	}{
		{
			"empty",
			clutel.BaggageProps{},
			tester.MSA{},
			tester.SA{},
		},
		{
			"only_member_values",
			clutel.NewBaggageProps("k", "v"),
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"member_and_properties",
			clutel.NewBaggageProps("k", "v", "fnord", "smarf"),
			tester.MSA{
				"k":       "v",
				"k_props": map[string]any{"fnord": "smarf"},
			},
			tester.SA{
				"k", "v",
				"k_props",
				map[string]any{"fnord": "smarf"},
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx, err = clutel.AddBaggageProps(ctx, test.input)
			require.NoError(t, err, cluerr.ToCore(err))

			_, found := clutel.GetBaggage(ctx, test.input.MemberKey())

			if test.input.MemberKey() != "" {
				assert.Truef(t, found, "baggage member %q should be found", test.input.MemberKey())
			} else {
				assert.False(t, found, "empty baggage member should not be found")
			}

			tester.AssertEq(
				ctx,
				t,
				"",
				test.expectM, tester.MSA{},
				test.expectS, tester.SA{},
			)
		})
	}
}

func TestEndSpanWithErrors(t *testing.T) {
	table := []struct {
		name string
		err  error
	}{
		{
			"an_err",
			assert.AnError,
		},
		{
			"nil",
			nil,
		},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := t.Context()

				if init {
					ocfg := clutel.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clutel.Init(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					ctx = ictx
				}

				ctx = clutel.NewSpan().
					WithOpts(
						trace.WithSpanKind(trace.SpanKindInternal),
						trace.WithAttributes(attribute.String("clues_trace", test.name)),
					).
					Start(ctx, test.name)

				// not testing much here; mostly catching panics and asserting usage.
				defer clutel.EndSpanWithError(ctx, test.err)
			})
		}
	}
}

func TestSetSpanError(t *testing.T) {
	table := []struct {
		name string
		err  error
	}{
		{
			"an_err",
			assert.AnError,
		},
		{
			"nil",
			nil,
		},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := t.Context()

				if init {
					ocfg := clutel.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clutel.Init(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					ctx = ictx
				}

				ctx = clutel.NewSpan().
					Start(ctx, test.name)

				clutel.SetSpanError(ctx, test.err)
			})
		}
	}
}

func TestSetSpanErrorM(t *testing.T) {
	table := []struct {
		name string
		msg  string
	}{
		{
			"an_err",
			assert.AnError.Error(),
		},
		{
			"empty_message",
			"",
		},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := t.Context()

				if init {
					ocfg := clutel.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clutel.Init(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					ctx = ictx
				}

				ctx = clutel.NewSpan().
					Start(ctx, test.name)

				clutel.SetSpanErrorM(ctx, test.msg)
			})
		}
	}
}

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
			result := clutel.Inherit(test.from(), test.to(), test.clobber)
			require.NotNil(t, result)
			test.assert(t, result)
		})
	}
}
