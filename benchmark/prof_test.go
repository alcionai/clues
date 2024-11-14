package benchmark

import (
	"context"
	"testing"

	"github.com/alcionai/clues/benchmark/mock"
)

func BenchmarkProf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mock.StartService(context.Background()).Call("perf")
	}
}
