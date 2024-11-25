package clues

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type grpcMock struct {
}

func (m grpcMock) Invoke(
	context.Context,
	string,
	any,
	any,
	...grpc.CallOption,
) error {
	return nil
}

func (m grpcMock) NewStream(
	context.Context,
	*grpc.StreamDesc,
	string,
	...grpc.CallOption,
) (grpc.ClientStream, error) {
	return nil, nil
}

func TestAddSpan(t *testing.T) {
	table := []struct {
		name        string
		names       []string
		expectTrace string
		kvs         []any
		expectM     map[string]any
		expectS     []any
	}{
		{"single", []string{"single"}, "single", nil, map[string]any{}, []any{}},
		{"multiple", []string{"single", "multiple"}, "single,multiple", nil, map[string]any{}, []any{}},
		{"duplicates", []string{"single", "multiple", "multiple"}, "single,multiple,multiple", nil, map[string]any{}, []any{}},
		{"single with kvs", []string{"single"}, "single", []any{"k", "v"}, map[string]any{"k": "v"}, []any{"k", "v"}},
		{"multiple with kvs", []string{"single", "multiple"}, "single,multiple", []any{"k", "v"}, map[string]any{"k": "v"}, []any{"k", "v"}},
		{"duplicates with kvs", []string{"single", "multiple", "multiple"}, "single,multiple,multiple", []any{"k", "v"}, map[string]any{"k": "v"}, []any{"k", "v"}},
	}
	for _, test := range table {
		for _, doInit := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, doInit)

			t.Run(tname, func(t *testing.T) {
				ctx := context.Background()

				if doInit {
					ictx, err := InitializeOTEL(ctx, test.name, OTELConfig{
						GRPCEndpoint: "localhost:4317",
					})
					if err != nil {
						t.Error("initializing clues", err)
						return
					}

					// we have to pull out the registry and replace the grpc endpoint
					// with a mock for this to work.
					ictx, reg := registryFromCtx(ictx)
					reg.otel.grpcConn = &grpcMock{}

					defer func() {
						err := Close(ictx)
						if err != nil {
							t.Error("closing clues:", err)
							return
						}
					}()

					ctx = ictx
				}

				assert.Equal(t, map[string]any{}, In(ctx).Map(), false)

				for _, name := range test.names {
					fmt.Println("ADDING SPAN", name)
					ctx = AddSpan(ctx, name, test.kvs...)
					defer CloseSpan(ctx)
				}

				fmt.Println("SPANS ADDED")
				in := In(ctx)

				assert.Equal(t, test.expectM, in.Map())
				assert.ElementsMatch(t, test.expectS, in.Slice())
				assert.Equal(t, test.expectTrace, in.Map()["clues_trace"])
				fmt.Println("ASSERTINOS MADE")
			})
		}
	}
}
