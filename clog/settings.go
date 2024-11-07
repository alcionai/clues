package clog

import (
	"os"
	"path/filepath"

	"golang.org/x/exp/slices"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/cecrets"
)

// ---------------------------------------------------
// consts
// ---------------------------------------------------

const clogLogFileEnv = "CLOG_LOG_FILE"

type logLevel string

const (
	LevelDebug    logLevel = "debug"
	LevelInfo     logLevel = "info"
	LevelError    logLevel = "error"
	LevelDisabled logLevel = "disabled"
)

type logFormat string

const (
	// use for cli/terminal
	FormatForHumans logFormat = "human"
	// use for cloud logging
	FormatToJSON logFormat = "json"
)

type sensitiveInfoHandlingAlgo string

const (
	HashSensitiveInfo            sensitiveInfoHandlingAlgo = "hash"
	MaskSensitiveInfo            sensitiveInfoHandlingAlgo = "mask"
	ShowSensitiveInfoInPlainText sensitiveInfoHandlingAlgo = "plaintext"
)

const (
	Stderr = "stderr"
	Stdout = "stdout"
)

// ---------------------------------------------------
// configuration
// ---------------------------------------------------

// Settings records the user's preferred logging settings.
type Settings struct {
	// the log file isn't exposed to end users because we
	// want to ensure a default of StdErr until they call
	// one of the file override hooks.
	fileOverride string

	// Format defines the output structure, standard design is
	// as text (human-at-a-console) or json (automation).
	Format logFormat
	// Level determines the minimum logging level.  Anything
	// below this level (following standard semantics) will
	// not get logged.
	Level logLevel

	// more fiddly bits
	SensitiveInfoHandling sensitiveInfoHandlingAlgo // how to obscure pii
	// when non-empty, only debuglogs with a label that matches
	// the provided labels will get delivered.  All other debug
	// logs get dropped.  Good way to expose a little bit of debug
	// logs without flooding your system.
	OnlyLogDebugIfContainsLabel []string
}

// LogToStdOut swaps the log output from Stderr to Stdout.
func (s Settings) LogToStdOut() Settings {
	s.fileOverride = Stdout
	return s
}

// LogToFile defines a system file to write all logs onto.
func (s Settings) LogToFile(pathToFile string) (Settings, error) {
	if len(pathToFile) == 0 {
		return s, clues.New("missing filepath for logging")
	}

	logdir := filepath.Dir(pathToFile)

	err := os.MkdirAll(logdir, 0o755)
	if err != nil {
		return s, clues.Wrap(err, "ensuring log file dir exists").
			With("log_dir", logdir)
	}

	s.fileOverride = pathToFile

	return s, nil
}

// EnsureDefaults sets any non-populated settings to their default value.
// exported for testing without circular dependencies.
func (s Settings) EnsureDefaults() Settings {
	set := s

	levels := []logLevel{LevelDisabled, LevelDebug, LevelInfo, LevelError}
	if len(set.Level) == 0 || !slices.Contains(levels, set.Level) {
		set.Level = LevelInfo
	}

	formats := []logFormat{FormatForHumans, FormatToJSON}
	if len(set.Format) == 0 || !slices.Contains(formats, set.Format) {
		set.Format = FormatForHumans
	}

	algs := []sensitiveInfoHandlingAlgo{ShowSensitiveInfoInPlainText, MaskSensitiveInfo, HashSensitiveInfo}
	if len(set.SensitiveInfoHandling) == 0 || !slices.Contains(algs, set.SensitiveInfoHandling) {
		set.SensitiveInfoHandling = ShowSensitiveInfoInPlainText
	}

	return set
}

func setCluesSecretsHash(alg sensitiveInfoHandlingAlgo) {
	switch alg {
	case HashSensitiveInfo:
		cecrets.SetHasher(cecrets.DefaultHash())
	case MaskSensitiveInfo:
		cecrets.SetHasher(cecrets.HashCfg{HashAlg: cecrets.Flatmask})
	case ShowSensitiveInfoInPlainText:
		cecrets.SetHasher(cecrets.NoHash())
	}
}
