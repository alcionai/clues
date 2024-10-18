package clog

import (
	"context"
	"fmt"
	"reflect"

	"github.com/alcionai/clues"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

// ------------------------------------------------------------------------------------------------
// builder is the primary logging handler
// most funcs that people would use in the daily drive are going
// to modfiy and/or return a builder instance.  The builder aggregates
// data passed to it until a log func is called (debug, info, or error).
// At that time it consumes all of the gathered data to send the log message.
// ------------------------------------------------------------------------------------------------

type builder struct {
	ctx             context.Context
	err             error
	zsl             *zap.SugaredLogger
	with            map[any]any
	labels          map[string]struct{}
	comments        map[string]struct{}
	skipCallerJumps int
}

func newBuilder(ctx context.Context) *builder {
	clgr := fromCtx(ctx)

	return &builder{
		ctx:      ctx,
		zsl:      clgr.zsl,
		with:     map[any]any{},
		labels:   map[string]struct{}{},
		comments: map[string]struct{}{},
	}
}

// log emits the log message and all attributes using the underlying logger.
//
// If otel is configured in clues, a duplicate log will be delivered to the
// otel receiver.  Is this redundant?  Yes.  Would it be better served by
// having a set of log emitters that registered by an interface so that
// we're not coupling usage?  Also yes.  These are known design issues that
// we can chase later.  This is all still in the early/poc stage and needs
// additional polish to shine.
func (b builder) log(l logLevel, msg string) {
	cv := clues.In(b.ctx).Map()
	zsl := b.zsl

	if b.err != nil {
		// error values should override context values.
		maps.Copy(cv, clues.InErr(b.err).Map())

		// attach the error and its labels
		zsl = zsl.
			With("error", b.err).
			With("error_labels", clues.Labels(b.err))
	}

	for k, v := range cv {
		zsl = zsl.With(k, v)
	}

	// plus any values added using builder.With()
	for k, v := range b.with {
		zsl = zsl.With(k, v)
	}

	// finally, make sure we attach the labels and comments
	zsl = zsl.With("clog_labels", maps.Keys(b.labels))
	zsl = zsl.With("clog_comments", maps.Keys(b.comments))

	if b.skipCallerJumps > 0 {
		zsl = zsl.WithOptions(zap.AddCallerSkip(b.skipCallerJumps))
	}

	// then write everything to the logger
	switch l {
	case LevelDebug:
		var ok bool

		for _, l := range cloggerton.set.OnlyLogDebugIfContainsLabel {
			if _, match := b.labels[l]; match {
				ok = true
				break
			}
		}

		if !ok {
			return
		}

		zsl.Debug(msg)
	case LevelInfo:
		zsl.Info(msg)
	case LevelError:
		zsl.Error(msg)
	}
}

// Err attaches the error to the builder.
// When logged, the error will be parsed for any clues parts
// and those values will get added to the resulting log.
//
// ex: if you have some `err := clues.New("badness").With("cause", reason)`
// then this will add both of the following to the log:
// - "error": "badness"
// - "cause": reason
func (b *builder) Err(err error) *builder {
	b.err = err
	return b
}

// Label adds all of the appended labels to the error.
// Adding labels is a great way to categorize your logs into broad scale
// concepts like "configuration", "process kickoff", or "process conclusion".
// they're also a great way to set up various granularities of debugging
// like "api queries" or "fine grained item review", since you  can configure
// clog to automatically filter debug level logging to only deliver if the
// logs match one or more labels, allowing you to only emit some of the
// overwhelming number of debug logs that we all know you produce, you
// little overlogger, you.
func (b *builder) Label(ls ...string) *builder {
	if len(b.labels) == 0 {
		b.labels = map[string]struct{}{}
	}

	for _, l := range ls {
		b.labels[l] = struct{}{}
	}

	return b
}

// Comments are available because why make your devs go all the way back to
// the code to find the comment about this log case?  Add them into the log
// itself!
func (b *builder) Comment(cmnt string) *builder {
	if len(b.comments) == 0 {
		b.comments = map[string]struct{}{}
	}

	b.comments[cmnt] = struct{}{}

	return b
}

// SkipCaller allows the logger to set its stackTrace N levels back from the
// current call.  This is great for helper functions that handle log actions
// which get used by many different consumers, as it will always report the
// log line as the call to the helper function, instead of the line within the
// helper func.
func (b *builder) SkipCaller(nSkips int) *builder {
	b.skipCallerJumps = nSkips
	return b
}

// getValue will return the value if not pointer, or the dereferenced
// value if it is a pointer.
func getValue(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}

		return rv.Elem().Interface()
	}
	return v
}

// With is your standard "With" func.  Add data in K:V pairs here to have them
// added to the log message metadata.  Ex: builder.With("foo", "bar") will add
// "foo": "bar" to the resulting log structure.  An uneven number of pairs will
// give the last key a nil value.
func (b *builder) With(vs ...any) *builder {
	if len(vs) == 0 {
		return b
	}

	if len(b.with) == 0 {
		b.with = map[any]any{}
	}

	for i := 0; i < len(vs); i += 2 {
		k := vs[i]
		var v any

		if (i + 1) < len(vs) {
			v = vs[i+1]
		}

		b.with[k] = getValue(v)
	}

	return b
}

// Debug level logging.  Whenever possible, you should add a debug category
// label to the log, as that will help your org maintain fine grained control
// of debug-level log filtering.
func (b builder) Debug(msgArgs ...any) {
	b.log(LevelDebug, fmt.Sprint(msgArgs...))
}

// Debugf level logging.  Whenever possible, you should add a debug category
// label to the log, as that will help your org maintain fine grained control
// of debug-level log filtering.
// f is for format.
// f is also for "Why?  Why are you using this?  Use Debugw instead, it's much better".
func (b builder) Debugf(tmpl string, vs ...any) {
	b.log(LevelDebug, fmt.Sprintf(tmpl, vs...))
}

// Debugw level logging.  Whenever possible, you should add a debug category
// label to the log, as that will help your org maintain fine grained control
// of debug-level log filtering.
// w is for With(key:values).  log.Debugw("msg", foo, bar) is the same as
// log.With(foo, bar).Debug("msg").
func (b builder) Debugw(msg string, keyValues ...any) {
	b.With(keyValues...).log(LevelDebug, msg)
}

// Info is your standard info log.  You know. For information.
func (b builder) Info(msgArgs ...any) {
	b.log(LevelInfo, fmt.Sprint(msgArgs...))
}

// Infof is your standard info log.  You know. For information.
// f is for format.
// f is also for "Don't make bloated log messages, kids.  Use Infow instead.".
func (b builder) Infof(tmpl string, vs ...any) {
	b.log(LevelInfo, fmt.Sprintf(tmpl, vs...))
}

// Infow is your standard info log.  You know. For information.
// w is for With(key:values).  log.Infow("msg", foo, bar) is the same as
// log.With(foo, bar).Info("msg").
func (b builder) Infow(msg string, keyValues ...any) {
	b.With(keyValues...).log(LevelInfo, msg)
}

// Error is an error level log.  It doesn't require an error, because there's no
// rule about needing an error to log at error level.  Or the reverse; feel free to
// add an error to your info or debug logs.  Log levels are just a fake labeling
// system, anyway.
func (b builder) Error(msgArgs ...any) {
	b.log(LevelError, fmt.Sprint(msgArgs...))
}

// Error is an error level log.  It doesn't require an error, because there's no
// rule about needing an error to log at error level.  Or the reverse; feel free to
// add an error to your info or debug logs.  Log levels are just a fake labeling
// system, anyway.
// f is for format.
// f is also for "Good developers know the value of using Errorw before Errorf."
func (b builder) Errorf(tmpl string, vs ...any) {
	b.log(LevelError, fmt.Sprintf(tmpl, vs...))
}

// Error is an error level log.  It doesn't require an error, because there's no
// rule about needing an error to log at error level.  Or the reverse; feel free to
// add an error to your info or debug logs.  Log levels are just a fake labeling
// system, anyway.
// w is for With(key:values).  log.Errorw("msg", foo, bar) is the same as
// log.With(foo, bar).Error("msg").
func (b builder) Errorw(msg string, keyValues ...any) {
	b.With(keyValues...).log(LevelError, msg)
}

// ------------------------------------------------------------------------------------------------
// wrapper: io.writer
// ------------------------------------------------------------------------------------------------

// Writer is a wrapper that turns the logger embedded in
// the given ctx into an io.Writer.  All logs are currently
// info-level.
type Writer struct {
	Ctx context.Context
}

// Write writes to the the Writer's clogger.
func (w Writer) Write(p []byte) (int, error) {
	Ctx(w.Ctx).log(LevelInfo, string(p))
	return len(p), nil
}
