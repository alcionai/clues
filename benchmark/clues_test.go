package benchmark

import (
	"context"
	"testing"

	"github.com/alcionai/clues"
)

func BenchmarkClues_Add_singleConstKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", "bar")
	}
	_ = ctx
}

func BenchmarkClues_Add_singleStaticKStaticV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchSize-i, i)
	}
	_ = ctx
}

func BenchmarkClues_Add_singleConstKStaticV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", i)
	}
	_ = ctx
}

func BenchmarkClues_Add_singleStaticKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), i, "bar")
	}
	_ = ctx
}

func BenchmarkClues_Add_singleConstKRandV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", benchVals[i%benchSize])
	}
	_ = ctx
}

func BenchmarkClues_Add_singleRandKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchVals[i%benchSize], "bar")
	}
	_ = ctx
}

func BenchmarkClues_Add_singleRandKRandV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchVals[i%benchSize], benchVals[i%benchSize])
	}
	_ = ctx
}

func BenchmarkClues_Add_multConstKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", "bar", "baz", "qux")
	}
	_ = ctx
}

func BenchmarkClues_Add_multStaticKStaticV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchSize-i, i, i-benchSize, i)
	}
	_ = ctx
}

func BenchmarkClues_Add_multConstKStaticV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", i, "baz", -i)
	}
	_ = ctx
}

func BenchmarkClues_Add_multStaticKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), i, "bar", -i, "qux")
	}
	_ = ctx
}

func BenchmarkClues_Add_multConstKRandV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(
			context.Background(),
			"foo", benchVals[i%benchSize],
			"baz", -benchVals[i%benchSize])
	}
	_ = ctx
}

func BenchmarkClues_Add_multRandKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(
			context.Background(),
			benchVals[i%benchSize], "bar",
			-benchVals[i%benchSize], "qux")
	}
	_ = ctx
}

func BenchmarkClues_Add_multRandKRandV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(
			context.Background(),
			benchVals[i%benchSize], benchVals[i%benchSize],
			-benchVals[i%benchSize], -benchVals[i%benchSize])
	}
	_ = ctx
}

func BenchmarkClues_AddMap_constKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[string]string{"foo": "bar", "baz": "qux"}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_AddMap_staticKStaticV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[int]int{benchSize - i: i, i - benchSize: i}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_AddMap_constKStaticV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[string]int{"foo": i, "baz": -i}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_AddMap_staticKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[int]string{i: "bar", -i: "qux"}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_AddMap_constKRandV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[string]int64{
			"foo": benchVals[i%benchSize],
			"baz": -benchVals[i%benchSize],
		}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_AddMap_randKConstV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[int64]string{
			benchVals[i%benchSize]:  "bar",
			-benchVals[i%benchSize]: "qux",
		}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_AddMap_randKRandV(b *testing.B) {
	var ctx context.Context
	for i := 0; i < b.N; i++ {
		m := map[int64]int64{
			benchVals[i%benchSize]:  benchVals[i%benchSize],
			-benchVals[i%benchSize]: -benchVals[i%benchSize],
		}
		ctx = clues.AddMap(context.Background(), m)
	}
	_ = ctx
}

func BenchmarkClues_In_constMap(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	m := map[string]any{}
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", "bar")
		dn = clues.In(ctx)
		m = dn.Map()
	}
	_ = dn
	_ = m
	_ = ctx
}

func BenchmarkClues_In_staticMap(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	m := map[string]any{}
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchSize-i, i)
		dn = clues.In(ctx)
		m = dn.Map()
	}
	_ = dn
	_ = m
	_ = ctx
}

func BenchmarkClues_In_linearMap(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	m := map[string]any{}
	j := 0
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), j, j+1)
		dn = clues.In(ctx)
		m = dn.Map()
		j += 2
	}
	_ = dn
	_ = m
	_ = ctx
}

func BenchmarkClues_In_bigMap(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	m := map[string]any{}
	for i := 0; i < b.N; i++ {
		ctx = context.Background()
		for j := range 1000 {
			ctx = clues.Add(ctx, j, j+1)
		}
		dn = clues.In(ctx)
		m = dn.Map()
	}
	_ = dn
	_ = m
	_ = ctx
}

func BenchmarkClues_In_randMap(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	m := map[string]any{}
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchVals[i%benchSize], benchVals[i%benchSize])
		dn = clues.In(ctx)
		m = dn.Map()
	}
	_ = dn
	_ = m
	_ = ctx
}

func BenchmarkClues_In_constSlice(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	s := []any{}
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), "foo", "bar")
		dn = clues.In(ctx)
		s = dn.Slice()
	}
	_ = dn
	_ = s
	_ = ctx
}

func BenchmarkClues_In_staticSlice(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	s := []any{}
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchSize-i, i)
		dn = clues.In(ctx)
		s = dn.Slice()
	}
	_ = dn
	_ = s
	_ = ctx
}

func BenchmarkClues_In_linearSlice(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	s := []any{}
	j := 0
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), j, j+1)
		dn = clues.In(ctx)
		s = dn.Slice()
		j += 2
	}
	_ = dn
	_ = s
	_ = ctx
}

func BenchmarkClues_In_randSlice(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	s := []any{}
	for i := 0; i < b.N; i++ {
		ctx = clues.Add(context.Background(), benchVals[i%benchSize], benchVals[i%benchSize])
		dn = clues.In(ctx)
		s = dn.Slice()
	}
	_ = dn
	_ = s
	_ = ctx
}

func BenchmarkClues_In_bigSlice(b *testing.B) {
	var ctx context.Context
	dn := clues.In(context.Background())
	s := []any{}
	for i := 0; i < b.N; i++ {
		ctx = context.Background()
		for j := range 1000 {
			ctx = clues.Add(ctx, j, j+1)
		}
		dn = clues.In(ctx)
		s = dn.Slice()
	}
	_ = dn
	_ = s
	_ = ctx
}
