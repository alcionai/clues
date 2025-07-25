package clutel_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/clutel"
	"github.com/alcionai/clues/internal/tester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
					ocfg := clues.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clues.InitializeOTEL(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					//nolint:lll
					// FIXME: this is causing failures at the moment which are non-trivial to
					// hack around.  Will need to return to it for more complete otel/grpc testing.
					// suggestion: https://github.com/pellared/opentelemetry-go-contrib/blob/8f8e9b60693177b91af45d0495289fc52aa5c50e/instrumentation/google.golang.org/grpc/otelgrpc/test/grpc_test.go#L88
					// defer func() {
					// 	err := clues.Close(ictx)
					// 	require.NoError(t, err, "closing clues")
					// 	if err != nil {
					// 		return
					// 	}
					// }()

					ctx = ictx
				}

				ctx = context.WithValue(ctx, tester.StubCtx{}, "instance")

				tester.MustEquals(
					t,
					tester.MSA{},
					clues.In(ctx).Map(),
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

				c := clues.In(ctx).Map()
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
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := context.Background()

				if init {
					ocfg := clues.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clues.InitializeOTEL(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					//nolint:lll
					// FIXME: this is causing failures at the moment which are non-trivial to
					// hack around.  Will need to return to it for more complete otel/grpc testing.
					// suggestion: https://github.com/pellared/opentelemetry-go-contrib/blob/8f8e9b60693177b91af45d0495289fc52aa5c50e/instrumentation/google.golang.org/grpc/otelgrpc/test/grpc_test.go#L88
					// defer func() {
					// 	err := clues.Close(ictx)
					// 	require.NoError(t, err, "closing clues")
					// 	if err != nil {
					// 		return
					// 	}
					// }()

					ctx = ictx
				}

				ctx = context.WithValue(ctx, tester.StubCtx{}, "instance")

				tester.MustEquals(
					t,
					tester.MSA{},
					clues.In(ctx).Map(),
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
