package clog

import (
	"os"
	"path/filepath"
	"time"

	"golang.org/x/exp/slices"

	"github.com/alcionai/clues"
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

// Default location for writing log files.
var defaultLogFileDir = filepath.Join(os.Getenv("HOME"), "Library", "Logs")

// ResolvedLogFile is the first log file established by the caller.
// It gets eagerly populated on the first act of ensuring settings
// defaults, which normally occurs during the Init call.
//
// If Init gets called more than once, or different settings are
// ensured, it's possible to override this value by manually specifying
// the log file in the settings used for that action.  But if no file
// is provided, the default will fall back to this resolved file first.
var ResolvedLogFile string

// Settings records the user's preferred logging settings.
type Settings struct {
	// core settings
	File   string    // what file to log to (alt: stderr, stdout)
	Format logFormat // whether to format as text (console) or json (cloud)
	Level  logLevel  // what level to log at

	// more fiddly bits
	SensitiveInfoHandling sensitiveInfoHandlingAlgo // how to obscure pii
	// when non-empty, only debuglogs with a label that matches
	// the provided labels will get delivered.  All other debug
	// logs get dropped.  Good way to expose a little bit of debug
	// logs without flooding your system.
	OnlyLogDebugIfContainsLabel []string
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

	if len(set.File) == 0 {
		set.File = GetLogFileOrDefault("")
	}

	if len(ResolvedLogFile) == 0 {
		ResolvedLogFile = set.File
	}

	return set
}

// Returns the default location for log file storage.
func defaultLogLocation() string {
	return filepath.Join(
		defaultLogFileDir,
		"clog",
		time.Now().UTC().Format("2006-01-02T15-04-05Z")+".log")
}

// GetLogFileOrDefault finds the log file in the users local system.
// Uses the env var declaration, if populated, else defaults to stderr.
// If this has already been called once before, uses the result of that
// prior call.
func GetLogFileOrDefault(useThisFile string) string {
	if len(ResolvedLogFile) > 0 {
		return ResolvedLogFile
	}

	// start by preferring the file given to us by the caller.
	r := useThisFile

	// if no file was provided, look for a configured location using
	// the default ENV.
	if len(r) == 0 {
		r = os.Getenv(clogLogFileEnv)
	}

	// if no file was provided, fall back to the default file location.
	if len(r) == 0 {
		r = defaultLogLocation()
	}

	// direct to Stdout if provided '-'.
	if r == "-" {
		r = Stdout
	}

	// if outputting to a file, make sure we can access the file.
	if r != Stdout && r != Stderr {
		logdir := filepath.Dir(r)

		err := os.MkdirAll(logdir, 0o755)
		if err != nil {
			return Stderr
		}
	}

	return r
}

func setCluesSecretsHash(alg sensitiveInfoHandlingAlgo) {
	switch alg {
	case HashSensitiveInfo:
		clues.SetHasher(clues.DefaultHash())
	case MaskSensitiveInfo:
		clues.SetHasher(clues.HashCfg{HashAlg: clues.Flatmask})
	case ShowSensitiveInfoInPlainText:
		clues.SetHasher(clues.NoHash())
	}
}
