package benchmark

import (
	"context"

	"github.com/alcionai/clues/benchmark/mock"
	"golang.org/x/exp/rand"
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

	mock.StartService(context.Background())
}
