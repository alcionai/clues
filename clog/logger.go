package clog

import (
	"context"
	"os"
	"sync"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/alcionai/clues"
)

// Yes, we just hijack zap for our logging needs here.
// This package isn't about writing a logger, it's about
// adding an opinionated shell around the zap logger.
var (
	cloggerton *clogger
	singleMu   sync.Mutex
)

type clogger struct {
	otel otellog.Logger
	zsl  *zap.SugaredLogger
	set  Settings
}

// ---------------------------------------------------------------------------
// constructors
// ---------------------------------------------------------------------------

func genLogger(set Settings) *zap.SugaredLogger {
	// when testing, ensure debug logging matches the test.v setting
	for _, arg := range os.Args {
		if arg == `--test.v=true` {
			set.Level = LevelDebug
		}
	}

	var (
		// this will be the backbone logger for the clogs
		// TODO: would be nice to accept a variety of loggers here, and
		// treat this all as a shim.  Oh well, gotta start somewhere.
		zlog *zap.Logger
		zcfg zap.Config
		// by default only add stacktraces to panics, else it gets too noisy.
		zopts = []zap.Option{
			zap.AddStacktrace(zapcore.PanicLevel),
			zap.AddCallerSkip(2),
		}
	)

	switch set.Format {
	// JSON means each row should appear as a single json object.
	case FormatToJSON:
		zcfg = setLevel(zap.NewProductionConfig(), set.Level)
		zcfg.OutputPaths = []string{set.File}
		// by default we'll use the columnar non-json format, which uses tab
		// separated values within each line, and may contain multiple json objs.
	default:
		zcfg = setLevel(zap.NewDevelopmentConfig(), set.Level)

		zcfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.StampMilli)

		// when printing to stdout/stderr, colorize things!
		if set.File == Stderr || set.File == Stdout {
			zcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	}

	zcfg.OutputPaths = []string{set.File}

	zlog, err := zcfg.Build(zopts...)
	if err != nil {
		zlog = zapcoreFallback(set)
	}

	// TODO: wrap the sugar logger to be a sugar... clogger...
	return zlog.Sugar()
}

// set up a logger core to use as a fallback in case the config doesn't work.
// we shouldn't ever need this, but it's nice to know there's a fallback in
// case configuration gets buggery, because everyone still wants their logs.
func zapcoreFallback(set Settings) *zap.Logger {
	levelFilter := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		switch set.Level {
		case LevelInfo:
			return lvl >= zapcore.InfoLevel
		case LevelError:
			return lvl >= zapcore.ErrorLevel
		case LevelDisabled:
			return false
		default:
			// default to debug
			return true
		}
	})

	// build out the zapcore fallback
	var (
		out            = zapcore.Lock(os.Stderr)
		consoleEncoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		core           = zapcore.NewTee(zapcore.NewCore(consoleEncoder, out, levelFilter))
	)

	return zap.New(core)
}

// converts a given logLevel into the zapcore level enum.
func setLevel(cfg zap.Config, level logLevel) zap.Config {
	switch level {
	case LevelInfo:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case LevelError:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case LevelDisabled:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	}

	return cfg
}

// singleton is the constructor and getter in one. Since we manage a global
// singleton for each instance, we only ever want one alive at any given time.
func singleton(ctx context.Context, set Settings) *clogger {
	singleMu.Lock()
	defer singleMu.Unlock()

	if cloggerton != nil {
		return cloggerton
	}

	set = set.EnsureDefaults()
	setCluesSecretsHash(set.SensitiveInfoHandling)

	zsl := genLogger(set)

	otelLogger := clues.In(ctx).OTELLogger()

	cloggerton = &clogger{
		otel: otelLogger,
		zsl:  zsl,
		set:  set,
	}

	return cloggerton
}

// ------------------------------------------------------------------------------------------------
// context management
// ------------------------------------------------------------------------------------------------

type loggingKey string

const ctxKey loggingKey = "clog_logger"

// Init embeds a logger within the context for later retrieval.
// It is a preferred, but not necessary, initialization step.
// If you don't call this and you start logging, or you call
// Singleton(), then the package will initialize a logger instance
// with the default values.  If you need to configure your logs,
// make sure to embed this first.
func Init(ctx context.Context, set Settings) context.Context {
	clogged := singleton(ctx, set)
	clogged.zsl.Debugw("seeding logger", "logger_settings", set)

	return plantLoggerInCtx(ctx, clogged)
}

// PlantLogger allows users to embed their own zap.SugaredLogger within the context.
// It's good for inheriting a logger instance that was generated elsewhere, in case
// you have a downstream package that wants to clog the code with a different zsl.
func PlantLogger(ctx context.Context, seed *zap.SugaredLogger) context.Context {
	return plantLoggerInCtx(ctx, &clogger{zsl: seed})
}

// plantLoggerInCtx allows users to embed their own zap.SugaredLogger within the
// context and with the given logger settings.
func plantLoggerInCtx(
	ctx context.Context,
	clogger *clogger,
) context.Context {
	if clogger == nil {
		return ctx
	}

	return context.WithValue(ctx, ctxKey, clogger)
}

// fromCtx pulls the clogger out of the context.  If no logger exists in the
// ctx, it returns the global singleton.
func fromCtx(ctx context.Context) *clogger {
	l := ctx.Value(ctxKey)
	// if l is still nil, we need to grab the global singleton or construct a singleton.
	if l == nil {
		l = singleton(ctx, Settings{}.EnsureDefaults())
	}

	return l.(*clogger)
}

// Ctx retrieves the logger embedded in the context.
// It also extracts any clues from the ctx and adds all k:v pairs to that log instance.
func Ctx(ctx context.Context) *builder {
	return newBuilder(ctx)
}

// CtxErr is a shorthand for clog.Ctx(ctx).Err(err)
func CtxErr(ctx context.Context, err error) *builder {
	nb := newBuilder(ctx)
	nb.err = err

	return nb
}

// Flush writes out all buffered logs.
// Probably good to do before shutting down whatever instance
// had initialized the singleton.
func Flush(ctx context.Context) {
	_ = Ctx(ctx).zsl.Sync()
}
