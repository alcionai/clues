package clog_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alcionai/clues/clog"
)

func TestSettings_ensureDefaults(t *testing.T) {
	s := clog.Settings{}
	require.Empty(t, s.Level, "level")
	require.Empty(t, s.Format, "format")
	require.Empty(t, s.SensitiveInfoHandling, "piialg")
	require.Empty(t, s.OnlyLogDebugIfContainsLabel, "debug filter")

	s = s.EnsureDefaults()
	require.NotEmpty(t, s.Level, "level")
	require.NotEmpty(t, s.Format, "format")
	require.NotEmpty(t, s.SensitiveInfoHandling, "piialg")
	require.Empty(t, s.OnlyLogDebugIfContainsLabel, "debug filter")
}
