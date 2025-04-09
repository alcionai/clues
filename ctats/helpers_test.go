package ctats

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type loader[K comparable, V any] interface {
	Load(k K) (V, bool)
}

func assertNotContains[K comparable, V any](
	t *testing.T,
	l loader[K, V],
	k K,
) {
	_, ok := l.Load(k)
	require.False(t, ok)
}

func assertContains[K comparable, V any](
	t *testing.T,
	l loader[K, V],
	k K,
) {
	_, ok := l.Load(k)
	require.True(t, ok)
}
