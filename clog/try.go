package clog

import (
	"context"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

// ---------------------------------------------------------------------------
// try is a qol handler for catching and logging errors.
// ---------------------------------------------------------------------------

type tryBuilder struct {
	ctx        context.Context
	logBuilder *builder

	msg        string // log message override
	setSpanErr bool   // sets the span status to error
}

func newTryBuilder(ctx context.Context) *tryBuilder {
	b := newBuilder(ctx)

	// always skip one caller at minimum to avoid referencing the
	// tryBuilder wrapper.
	b.SkipCaller(1)

	return &tryBuilder{
		ctx:        ctx,
		logBuilder: b,
	}
}

// Label adds all of the appended labels to the log produced if a
// panic occurs.
func (tb *tryBuilder) Label(ls ...string) *tryBuilder {
	tb.logBuilder.Label(ls...)
	return tb
}

// Comment adds a long form message to the log that gets produced if
// a panic occurs.
func (tb *tryBuilder) Comment(cmnt string) *tryBuilder {
	tb.logBuilder.Comment(cmnt)
	return tb
}

// SkipCaller allows the logger to set its stackTrace N levels back from the
// current call.
func (tb *tryBuilder) SkipCaller(nSkips int) *tryBuilder {
	// always skip one extra caller at minimum to avoid referencing the
	// tryBuilder wrapper.
	tb.logBuilder.SkipCaller(nSkips + 1)
	return tb
}

// With is your standard "With" func.  Add data in K:V pairs here to have them
// added to the log message metadata.  Since you're using a Try builder with
// a ctx reference, these values will also get added to the context that gets
// passed back to you in the Catch() handler.
func (tb *tryBuilder) With(kvs ...any) *tryBuilder {
	nc := node.FromCtx(tb.ctx)

	tb.ctx = node.EmbedInCtx(
		tb.ctx,
		nc.AddValues(
			tb.ctx,
			stringify.Normalize(kvs...),
		),
	)

	tb.logBuilder.With(kvs...)

	return tb
}

// SetSpanToErr will set the status of the otel span in the ctx (if one exists)
// to `error`, and will attach the appropriate attributes describing the error
// and its stacktrace.
func (tb *tryBuilder) SetSpanToErr() *tryBuilder {
	tb.setSpanErr = true
	return tb
}

// Msg overrides the log message.  Just in case you want something personalized.
// Otherwise the default message will be "PANIC recovered at <code_line>".
func (tb *tryBuilder) Msg(msg string) *tryBuilder {
	tb.msg = msg
	return tb
}

// Try is a convenience function for easy panic handling.  Should a panic occur,
// the catch will automatically log the recovered error, along with a stack trace
// and other details.  Why is this in the logging package?  Because panics should
// be logged.  But you're still using go. So don't get any wise ideas here. And
// don't panic.
//
// Like any use of recovery in go, this must be called in a deferred function
// to work properly.
//
//	defer clog.Try(ctx).
//	  Catch(func(ctx context.Context, recovered any) {
//	    // your spectacular handling here
//	  })
//
// Try produces a new builder that can be configured similar to a clog log line.
// Any additional values from .With(...) will automatically be applied to the
// context provided to the Try(ctx) constructor.  These amendments are present
// both in any logging within the panic handling, and in the ctx passed to the
// Catch() handler.
//
// Try must get followed by a Catch() call to actually, you know, catch anything.
// Not much point in just trying.
func Try(ctx context.Context) *tryBuilder {
	return newTryBuilder(ctx)
}

type catchHandler = func(ctx context.Context, recovered any)

// Catch operates the actual panic recovery.  If no panic occurs, nothing happens.
// If a panic is recovered, the provided handler is passed the original context
// (augmented by any With, Label, or Comment calls) and the recovered value.
//
// Panics are always logged, so you only need to handle the panic if you want to
// respond to the condition, hand the recovered value back to the caller, or
// repanic.  No changes will be made to the recovered value.
func (tb *tryBuilder) Catch(handler catchHandler) {
	r := recover()

	if r == nil {
		return
	}

	msg := "PANIC recovered at " + node.GetCaller(tb.logBuilder.skipCallerJumps)

	if tb.msg != "" {
		msg += tb.msg
	}

	tb.logBuilder.
		Err(r.(error)).
		StackTrace("exception.stacktrace").
		Comment(msg)

	if tb.setSpanErr {
		node.SetSpanError(tb.ctx, r.(error), msg)
	}

	if handler != nil {
		handler(tb.ctx, r)
	}
}
