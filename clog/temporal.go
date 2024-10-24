package clog

// copied from https://github.com/temporalio/samples-go/blob/main/temporalAdapter/zap_adapter.go
//

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// temporalAdapter is a wrapper around the clues logger that complies with
// the temporal logging interface, thus allowing clog to embed (and extract)
// the logger into the workflow context.
type temporalAdapter[CTX valuer] struct {
	builder *builder[CTX]
	clogger *clogger
}

func NewTemporalAdapter[CTX valuer](
	ctx context.Context,
) *temporalAdapter[CTX] {
	return &temporalAdapter[CTX]{
		// Skip one call frame to exclude zap_adapter itself.
		// Or it can be configured when logger is created (not always possible).
		builder: Ctx(ctx.(CTX)).SkipCaller(1),
		clogger: fromCtx(ctx),
	}
}

// TemporalWFCtx is just like clog.Ctx(ctx), except that it is specifically
// used to pull the registered logger out of the temporal workflow.
func TemporalWFCtx(
	ctx workflow.Context,
) *builder[context.Context] {
	tmpLog := workflow.GetLogger(ctx)

	if tmpLog != nil {
		if adapter, ok := tmpLog.(*temporalAdapter[context.Context]); ok {
			return adapter.builder
		}
	}

	return Singleton()
}

// TemporalCtx is just like clog.Ctx(ctx), except that it is specifically
// used to pull the registered logger out of a temporal activity.
func TemporalCtx(
	ctx context.Context,
) *builder[context.Context] {
	tmpLog := activity.GetLogger(ctx)

	if tmpLog != nil {
		if adapter, ok := tmpLog.(*temporalAdapter[context.Context]); ok {
			return adapter.builder
		}
	}

	return Singleton()
}

func (ta *temporalAdapter[CTX]) Debug(
	msg string,
	kvs ...any,
) {
	ta.builder.With(kvs...).Debug(msg)
}

func (ta *temporalAdapter[CTX]) Info(
	msg string,
	kvs ...any,
) {
	ta.builder.With(kvs...).Info(msg)
}

func (ta *temporalAdapter[CTX]) Warn(
	msg string,
	kvs ...any,
) {
	// clog doesn't support Warn
	ta.builder.With(kvs...).Info(msg)
}

func (ta *temporalAdapter[CTX]) Error(
	msg string,
	kvs ...any,
) {
	ta.builder.With(kvs...).Error(msg)
}

func (ta *temporalAdapter[CTX]) With(
	kvs ...any,
) log.Logger {
	return &temporalAdapter[CTX]{
		builder: ta.builder.With(kvs...),
	}
}

func (ta *temporalAdapter[CTX]) WithCallerSkip(
	nCallers int,
) log.Logger {
	return &temporalAdapter[CTX]{
		builder: ta.builder.SkipCaller(nCallers),
	}
}
