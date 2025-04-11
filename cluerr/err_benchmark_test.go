package cluerr_test

import (
	"context"
	"strconv"
	"testing"

	"golang.org/x/exp/rand"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/cluerr"
)

var (
	benchKeys []int64
	benchVals []int64
)

const benchSize = 4096

func init() {
	benchKeys, benchVals = make([]int64, benchSize), make([]int64, benchSize)
	for i := 0; i < benchSize; i++ {
		benchKeys[i], benchVals[i] = rand.Int63(), rand.Int63()
	}

	rand.Shuffle(benchSize, func(i, j int) {
		benchKeys[i], benchKeys[j] = benchKeys[j], benchKeys[i]
	})
}

func BenchmarkWith_singleConstKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", "bar")
	}
}

func BenchmarkWith_singleStaticKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchSize-i, i)
	}
}

func BenchmarkWith_singleConstKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", i)
	}
}

func BenchmarkWith_singleStaticKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(i, "bar")
	}
}

func BenchmarkWith_singleConstKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", benchVals[i%benchSize])
	}
}

func BenchmarkWith_singleRandKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchVals[i%benchSize], "bar")
	}
}

func BenchmarkWith_singleRandKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchVals[i%benchSize], benchVals[i%benchSize])
	}
}

func BenchmarkWith_multConstKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", "bar", "baz", "qux")
	}
}

func BenchmarkWith_multStaticKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchSize-i, i, i-benchSize, i)
	}
}

func BenchmarkWith_multConstKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", i, "baz", -i)
	}
}

func BenchmarkWith_multStaticKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(i, "bar", -i, "qux")
	}
}

func BenchmarkWith_multConstKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(
			"foo", benchVals[i%benchSize],
			"baz", -benchVals[i%benchSize])
	}
}

func BenchmarkWith_multRandKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(
			benchVals[i%benchSize], "bar",
			-benchVals[i%benchSize], "qux")
	}
}

func BenchmarkWith_multRandKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(
			benchVals[i%benchSize], benchVals[i%benchSize],
			-benchVals[i%benchSize], -benchVals[i%benchSize])
	}
}

func BenchmarkWith_chainConstKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", "bar").
			With("baz", "qux")
	}
}

func BenchmarkWith_chainStaticKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchSize-i, i).
			With(i-benchSize, i)
	}
}

func BenchmarkWith_chainConstKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", i).
			With("baz", -i)
	}
}

func BenchmarkWith_chainStaticKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(i, "bar").
			With(-i, "qux")
	}
}

func BenchmarkWith_chainConstKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With("foo", benchVals[i%benchSize]).
			With("baz", -benchVals[i%benchSize])
	}
}

func BenchmarkWith_chainRandKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchVals[i%benchSize], "bar").
			With(-benchVals[i%benchSize], "qux")
	}
}

func BenchmarkWith_chainRandKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		err = err.With(benchVals[i%benchSize], benchVals[i%benchSize]).
			With(-benchVals[i%benchSize], -benchVals[i%benchSize])
	}
}

func BenchmarkWithMap_constKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{"foo": "bar", "baz": "qux"}
		err = err.WithMap(m)
	}
}

func BenchmarkWithMap_staticKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{
			strconv.Itoa(benchSize - i): i,
			strconv.Itoa(i - benchSize): i,
		}
		err = err.WithMap(m)
	}
}

func BenchmarkWithMap_constKStaticV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{"foo": i, "baz": -i}
		err = err.WithMap(m)
	}
}

func BenchmarkWithMap_staticKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{
			strconv.Itoa(i):  "bar",
			strconv.Itoa(-i): "qux",
		}
		err = err.WithMap(m)
	}
}

func BenchmarkWithMap_constKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{
			"foo": benchVals[i%benchSize],
			"baz": -benchVals[i%benchSize],
		}
		err = err.WithMap(m)
	}
}

func BenchmarkWithMap_randKConstV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{
			strconv.FormatInt(benchVals[i%benchSize], 10):  "bar",
			strconv.FormatInt(-benchVals[i%benchSize], 10): "qux",
		}
		err = err.WithMap(m)
	}
}

func BenchmarkWithMap_randKRandV(b *testing.B) {
	err := cluerr.New("err")

	for i := 0; i < b.N; i++ {
		m := map[string]any{
			strconv.FormatInt(benchVals[i%benchSize], 10):  benchVals[i%benchSize],
			strconv.FormatInt(-benchVals[i%benchSize], 10): -benchVals[i%benchSize],
		}
		err = err.WithMap(m)
	}
}

func BenchmarkWithClues_constKConstV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", "bar")
		err = err.WithClues(ctx)
	}
}

func BenchmarkWithClues_staticKStaticV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchSize-i, i)
		err = err.WithClues(ctx)
	}
}

func BenchmarkWithClues_constKStaticV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", i)
		err = err.WithClues(ctx)
	}
}

func BenchmarkWithClues_staticKConstV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, i, "bar")
		err = err.WithClues(ctx)
	}
}

func BenchmarkWithClues_constKRandV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", benchVals[i%benchSize])
		err = err.WithClues(ctx)
	}
}

func BenchmarkWithClues_randKConstV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchVals[i%benchSize], "bar")
		err = err.WithClues(ctx)
	}
}

func BenchmarkWithClues_randKRandV(b *testing.B) {
	err := cluerr.New("err")
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchVals[i%benchSize], benchVals[i%benchSize])
		err = err.WithClues(ctx)
	}
}

func BenchmarkInErr_const(b *testing.B) {
	err := cluerr.New("err")

	var m map[string]any

	for i := 0; i < b.N; i++ {
		err = err.With("foo", "bar")
		m = cluerr.CluesIn(err).Map()
	}

	_ = m
}

func BenchmarkInErr_static(b *testing.B) {
	err := cluerr.New("err")

	var m map[string]any

	for i := 0; i < b.N; i++ {
		err = err.With(i, -i)
		m = cluerr.CluesIn(err).Map()
	}

	_ = m
}

func BenchmarkInErr_rand(b *testing.B) {
	err := cluerr.New("err")

	var m map[string]any

	for i := 0; i < b.N; i++ {
		err = err.With(benchVals[i%benchSize], benchVals[i%benchSize])
		m = cluerr.CluesIn(err).Map()
	}

	_ = m
}
