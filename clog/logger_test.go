package clog_test

import (
	"testing"

	"github.com/alcionai/clues/clog"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LoggerUnitSuite struct {
	suite.Suite
}

func TestLoggerUnitSuite(t *testing.T) {
	suite.Run(t, new(LoggerUnitSuite))
}

func (suite *LoggerUnitSuite) TestSettings_ensureDefaults() {
	t := suite.T()

	s := clog.Settings{}
	require.Empty(t, s.File, "file")
	require.Empty(t, s.Level, "level")
	require.Empty(t, s.Format, "format")
	require.Empty(t, s.SensitiveInfoHandling, "piialg")
	require.Empty(t, s.OnlyLogDebugIfContainsLabel, "debug filter")

	s = s.EnsureDefaults()
	require.NotEmpty(t, s.File, "file")
	require.NotEmpty(t, s.Level, "level")
	require.NotEmpty(t, s.Format, "format")
	require.NotEmpty(t, s.SensitiveInfoHandling, "piialg")
	require.Empty(t, s.OnlyLogDebugIfContainsLabel, "debug filter")
}
