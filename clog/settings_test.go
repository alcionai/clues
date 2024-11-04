package clog

import (
	"path/filepath"
	"testing"

	"github.com/alcionai/clues"
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
			test.expectErr(t, err, clues.ToCore(err))
			assert.Equal(t, test.expectOverride, set.fileOverride)
		})
	}
}
