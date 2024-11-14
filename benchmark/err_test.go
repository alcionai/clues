package benchmark

import (
	"context"
	"strconv"
	"testing"

	"github.com/alcionai/clues"
)

func BenchmarkErr_New(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
	}
	_ = err
}

func BenchmarkErr_With_singleConstKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", "bar")
	}
	_ = err
}

func BenchmarkErr_With_singleStaticKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchSize-i, i)
	}
	_ = err
}

func BenchmarkErr_With_singleConstKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", i)
	}
	_ = err
}

func BenchmarkErr_With_singleStaticKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(i, "bar")
	}
	_ = err
}

func BenchmarkErr_With_singleConstKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", benchVals[i%benchSize])
	}
	_ = err
}

func BenchmarkErr_With_singleRandKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchVals[i%benchSize], "bar")
	}
	_ = err
}

func BenchmarkErr_With_singleRandKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchVals[i%benchSize], benchVals[i%benchSize])
	}
	_ = err
}

func BenchmarkErr_With_multConstKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", "bar", "baz", "qux")
	}
	_ = err
}

func BenchmarkErr_With_multStaticKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchSize-i, i, i-benchSize, i)
	}
	_ = err
}

func BenchmarkErr_With_multConstKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", i, "baz", -i)
	}
	_ = err
}

func BenchmarkErr_With_multStaticKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(i, "bar", -i, "qux")
	}
	_ = err
}

func BenchmarkErr_With_multConstKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(
			"foo", benchVals[i%benchSize],
			"baz", -benchVals[i%benchSize])
	}
	_ = err
}

func BenchmarkErr_With_multRandKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(
			benchVals[i%benchSize], "bar",
			-benchVals[i%benchSize], "qux")
	}
	_ = err
}

func BenchmarkErr_With_multRandKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(
			benchVals[i%benchSize], benchVals[i%benchSize],
			-benchVals[i%benchSize], -benchVals[i%benchSize])
	}
	_ = err
}

func BenchmarkErr_With_chainConstKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", "bar").
			With("baz", "qux")
	}
	_ = err
}

func BenchmarkErr_With_chainStaticKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchSize-i, i).
			With(i-benchSize, i)
	}
	_ = err
}

func BenchmarkErr_With_chainConstKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", i).
			With("baz", -i)
	}
	_ = err
}

func BenchmarkErr_With_chainStaticKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(i, "bar").
			With(-i, "qux")
	}
	_ = err
}

func BenchmarkErr_With_chainConstKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", benchVals[i%benchSize]).
			With("baz", -benchVals[i%benchSize])
	}
	_ = err
}

func BenchmarkErr_With_chainRandKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchVals[i%benchSize], "bar").
			With(-benchVals[i%benchSize], "qux")
	}
	_ = err
}

func BenchmarkErr_With_chainRandKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchVals[i%benchSize], benchVals[i%benchSize]).
			With(-benchVals[i%benchSize], -benchVals[i%benchSize])
	}
	_ = err
}

func BenchmarkErr_WithMap_constKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{"foo": "bar", "baz": "qux"}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithMap_staticKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{
			strconv.Itoa(benchSize - i): i,
			strconv.Itoa(i - benchSize): i,
		}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithMap_constKStaticV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{"foo": i, "baz": -i}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithMap_staticKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{
			strconv.Itoa(i):  "bar",
			strconv.Itoa(-i): "qux",
		}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithMap_constKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{
			"foo": benchVals[i%benchSize],
			"baz": -benchVals[i%benchSize],
		}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithMap_randKConstV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{
			strconv.FormatInt(benchVals[i%benchSize], 10):  "bar",
			strconv.FormatInt(-benchVals[i%benchSize], 10): "qux",
		}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithMap_randKRandV(b *testing.B) {
	var err *clues.Err
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		m := map[string]any{
			strconv.FormatInt(benchVals[i%benchSize], 10):  benchVals[i%benchSize],
			strconv.FormatInt(-benchVals[i%benchSize], 10): -benchVals[i%benchSize],
		}
		err = err.WithMap(m)
	}
	_ = err
}

func BenchmarkErr_WithClues_constKConstV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), "foo", "bar")
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_WithClues_staticKStaticV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), benchSize-i, i)
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_WithClues_constKStaticV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), "foo", i)
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_WithClues_staticKConstV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), i, "bar")
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_WithClues_constKRandV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), "foo", benchVals[i%benchSize])
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_WithClues_randKConstV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), benchVals[i%benchSize], "bar")
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_WithClues_randKRandV(b *testing.B) {
	var err *clues.Err
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		ctx = clues.Add(context.Background(), benchVals[i%benchSize], benchVals[i%benchSize])
		err = err.WithClues(ctx)
	}
	_ = err
}

func BenchmarkErr_InErr_const(b *testing.B) {
	var err *clues.Err
	var m map[string]any
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With("foo", "bar")
		m = clues.InErr(err).Map()
	}
	_ = err
	_ = m
}

func BenchmarkErr_InErr_static(b *testing.B) {
	var err *clues.Err
	var m map[string]any
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(i, -i)
		m = clues.InErr(err).Map()
	}
	_ = err
	_ = m
}

func BenchmarkErr_InErr_rand(b *testing.B) {
	var err *clues.Err
	var m map[string]any
	for i := 0; i < b.N; i++ {
		err = clues.New("err")
		err = err.With(benchVals[i%benchSize], benchVals[i%benchSize])
		m = clues.InErr(err).Map()
	}
	_ = err
	_ = m
}
