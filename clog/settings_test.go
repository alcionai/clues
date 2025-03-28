package clog

import (
	"path/filepath"
	"testing"

	"github.com/alcionai/clues/cluerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettings_LogToFile(t *testing.T) {
	tempDir := t.TempDir()

	table := []struct {
		name           string
		input          string
		expectErr      require.ErrorAssertionFunc
		expectOverride string
	}{
		{
			name:           "empty",
			input:          "",
			expectErr:      require.Error,
			expectOverride: "",
		},
		{
			name:           "doesn't exist",
			input:          filepath.Join(tempDir, "foo", "bar", "baz", "log.log"),
			expectErr:      require.NoError,
			expectOverride: filepath.Join(tempDir, "foo", "bar", "baz", "log.log"),
		},
		{
			name:           "exists",
			input:          filepath.Join(tempDir, "log.log"),
			expectErr:      require.NoError,
			expectOverride: filepath.Join(tempDir, "log.log"),
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			set, err := Settings{}.LogToFile(test.input)
			test.expectErr(t, err, cluerr.ToCore(err))
			assert.Equal(t, test.expectOverride, set.fileOverride)
		})
	}
}

func TestLogLevel_Includes(t *testing.T) {
	table := []struct {
		name     string
		input    logLevel
		allow    []logLevel
		notallow []logLevel
	}{
		{
			name:  "debug allows everything",
			input: LevelDebug,
			allow: []logLevel{
				LevelDebug,
				LevelInfo,
				LevelError,
			},
			notallow: []logLevel{
				LevelDisabled,
			},
		},
		{
			name:  "info",
			input: LevelDebug,
			allow: []logLevel{
				LevelInfo,
				LevelError,
			},
			notallow: []logLevel{
				LevelDebug,
				LevelDisabled,
			},
		},
		{
			name:  "error",
			input: LevelDebug,
			allow: []logLevel{
				LevelError,
			},
			notallow: []logLevel{
				LevelDebug,
				LevelInfo,
				LevelDisabled,
			},
		},
		{
			name:  "disabled allows nothing",
			input: LevelDebug,
			allow: []logLevel{},
			notallow: []logLevel{
				LevelDebug,
				LevelInfo,
				LevelError,
				LevelDisabled,
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			for _, allow := range test.allow {
				assert.True(t, test.input.includes(allow))
			}
			for _, not := range test.notallow {
				assert.False(t, test.input.includes(not))
			}
		})
	}
}
