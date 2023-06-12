package clues_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/alcionai/clues"
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

func BenchmarkAdd_singleConstKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", "bar")
		clues.In(ctx)
	}
}

func BenchmarkAdd_singleStaticKStaticV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchSize-i, i)
		clues.In(ctx)
	}
}

func BenchmarkAdd_singleConstKStaticV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", i)
		clues.In(ctx)
	}
}

func BenchmarkAdd_singleStaticKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, i, "bar")
		clues.In(ctx)
	}
}

func BenchmarkAdd_singleConstKRandV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", benchVals[i%benchSize])
		clues.In(ctx)
	}
}

func BenchmarkAdd_singleRandKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchVals[i%benchSize], "bar")
		clues.In(ctx)
	}
}

func BenchmarkAdd_singleRandKRandV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchVals[i%benchSize], benchVals[i%benchSize])
		clues.In(ctx)
	}
}

func BenchmarkAdd_multConstKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", "bar", "baz", "qux")
		clues.In(ctx)
	}
}

func BenchmarkAdd_multStaticKStaticV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchSize-i, i, i-benchSize, i)
		clues.In(ctx)
	}
}

func BenchmarkAdd_multConstKStaticV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", i, "baz", -i)
		clues.In(ctx)
	}
}

func BenchmarkAdd_multStaticKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, i, "bar", -i, "qux")
		clues.In(ctx)
	}
}

func BenchmarkAdd_multConstKRandV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(
			ctx,
			"foo", benchVals[i%benchSize],
			"baz", -benchVals[i%benchSize])
		clues.In(ctx)
	}
}

func BenchmarkAdd_multRandKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(
			ctx,
			benchVals[i%benchSize], "bar",
			-benchVals[i%benchSize], "qux")
		clues.In(ctx)
	}
}

func BenchmarkAdd_multRandKRandV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(
			ctx,
			benchVals[i%benchSize], benchVals[i%benchSize],
			-benchVals[i%benchSize], -benchVals[i%benchSize])
		clues.In(ctx)
	}
}

func BenchmarkAddMap_constKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[string]string{"foo": "bar", "baz": "qux"}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkAddMap_staticKStaticV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[int]int{benchSize - i: i, i - benchSize: i}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkAddMap_constKStaticV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[string]int{"foo": i, "baz": -i}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkAddMap_staticKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[int]string{i: "bar", -i: "qux"}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkAddMap_constKRandV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[string]int64{
			"foo": benchVals[i%benchSize],
			"baz": -benchVals[i%benchSize],
		}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkAddMap_randKConstV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[int64]string{
			benchVals[i%benchSize]:  "bar",
			-benchVals[i%benchSize]: "qux",
		}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkAddMap_randKRandV(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		m := map[int64]int64{
			benchVals[i%benchSize]:  benchVals[i%benchSize],
			-benchVals[i%benchSize]: -benchVals[i%benchSize],
		}
		ctx = clues.AddMap(ctx, m)
		clues.In(ctx)
	}
}

func BenchmarkIn_const(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, "foo", "bar")
		clues.In(ctx)
	}
}

func BenchmarkIn_static(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchSize-i, i)
		clues.In(ctx)
	}
}

func BenchmarkIn_rand(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(ctx, benchVals[i%benchSize], benchVals[i%benchSize])
		clues.In(ctx)
	}
}