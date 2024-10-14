package clues

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/workflow"
)

// ---------------------------------------------------------------------------
// persistent client initialization
// ---------------------------------------------------------------------------

// Initialize will spin up any persistent clients that are held by clues,
// such as OTEL communication.  Clues will use these optimistically in the
// background to provide additional telemetry hook-ins.
//
// --- NOTICE ----
// This is an alpha feature, and may not yet work properly.
//
// If initialization has already been run, this func no-ops.
//
// Clues will operate as expected in the event of an error, or of initialization
// is not called.  This is a purely optional step.
func Initialize[CTX valuer](
	ctx CTX,
	serviceName string,
	config OTELConfig,
) (CTX, error) {
	nc := nodeFromCtx(ctx, defaultNamespace)

	err := nc.init(ctx, serviceName, config)
	if err != nil {
		return ctx, err
	}

	return setDefaultNodeInCtx(ctx, nc), nil
}

// Close will flush all buffered data waiting to be read.  If Initialize was not
// called, this call is a no-op.  Should be called in a defer after initializing.
func Close[CTX valuer](
	ctx CTX,
) error {
	nc := nodeFromCtx(ctx, defaultNamespace)

	if nc.otel != nil {
		var cCtx context.Context = valuer(ctx).(context.Context)

		// FIXME: there's probably a better way to extract a context in case of temporal
		// in case of temporal, extract the context from the workflow ctx.  Maybe the
		// best thing to do is mock the Done status from the workflow channel, since it's
		// not being used for anything other than that.
		if _, match := valuer(ctx).(workflow.Context); match {
			// only effect here is that TODO won't hit the Done() channel select.
			cCtx = context.TODO()
		}

		err := nc.otel.close(cCtx)
		if err != nil {
			return Wrap(err, "closing otel client")
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// key-value metadata
// ---------------------------------------------------------------------------

// Add adds all key-value pairs to the clues.
func Add[CTX valuer](
	ctx CTX,
	kvs ...any,
) CTX {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, add(nc, kvs...))
}

// AddTo adds all key-value pairs to a namespaced set of clues.
func AddTo[CTX valuer](
	ctx CTX,
	namespace string,
	kvs ...any,
) CTX {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	return setNodeInCtx(ctx, namespace, add(nc, kvs...))
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[CTX valuer, K comparable, V any](
	ctx CTX,
	m map[K]V,
) CTX {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, addMap(nc, m))
}

// AddMapTo adds a shallow clone of the map to a namespaced set of clues.
func AddMapTo[CTX valuer, K comparable, V any](
	ctx CTX,
	namespace string,
	m map[K]V,
) CTX {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	return setNodeInCtx(ctx, namespace, addMap(nc, m))
}

// ---------------------------------------------------------------------------
// traces
// ---------------------------------------------------------------------------

// AddSpan stacks a clues node onto this context and uses the provided
// name for the trace id. AddSpan can be called without additional values
// if you only want to add a trace marker.  The assumption is that an otel
// span is generated and attached to the node.  Callers should always follow
// this addition with a closing `defer clues.CloseSpan(ctx)`.
func AddSpan(
	// span addition only works with context.Context, and cannot be called
	// on a temporal workflow.Context.
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)
	ctx, nc = nc.addSpan(ctx, name)

	return setDefaultNodeInCtx(ctx, addSpanWith(nc, name, kvs...))
}

// PassTrace adds the current trace details to the provided
// headers.  If otel is not initialized, no-ops.
//
// The mapCarrier is mutated by this request.  The passed
// reference is returned mostly as a quality-of-life step
// so that callers don't need to declare the map outside of
// this call.
func PassTrace[C traceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) C {
	nodeFromCtx(ctx, defaultNamespace).
		passTrace(ctx, asTraceMapCarrier(mapCarrier))

	return mapCarrier
}

// ReceiveTrace extracts the current trace details from the
// headers and adds them to the context.  If otel is not
// initialized, no-ops.
func ReceiveTrace[C traceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) context.Context {
	return nodeFromCtx(ctx, defaultNamespace).
		receiveTrace(ctx, asTraceMapCarrier(mapCarrier))
}

// AddSpanTo stacks a clues node onto this context and uses the provided
// name for the trace id. AddSpanTo can be called without additional values
// if you only want to add a trace marker.  The assumption is that an otel
// span is generated and attached to the node.  Callers should always follow
// this addition with a closing `defer clues.CloseSpan(ctx)`.
func AddSpanTo(
	// span addition only works with context.Context, and cannot be called
	// on a temporal workflow.Context.
	ctx context.Context,
	name, namespace string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	ctx, nc = nc.addSpan(ctx, name)

	return setNodeInCtx(ctx, namespace, addSpanWith(nc, name, kvs...))
}

// CurrentSpan retrieves the current span context details
// If otel is not initialized, returns nil.
func SpanContext(
	ctx context.Context,
) trace.SpanContext {
	return nodeFromCtx(ctx, defaultNamespace).
		currentSpan(ctx).
		SpanContext()
}

// CloseSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func CloseSpan[CTX valuer](
	ctx CTX,
) CTX {
	nc := nodeFromCtx(ctx, defaultNamespace)
	node := nc.closeSpan()

	return setDefaultNodeInCtx(ctx, node)
}

// CloseSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func CloseSpanTo[CTX valuer](
	ctx CTX,
	namespace string,
) CTX {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	node := nc.closeSpan()

	return setNodeInCtx(ctx, namespace, node)
}

// ---------------------------------------------------------------------------
// comments
// ---------------------------------------------------------------------------

// AddComment adds a long form comment to the clues.
//
// Comments are special case additions to the context.  They're here to, well,
// let you add comments!  Why?  Because sometimes it's not sufficient to have a
// log let you know that a line of code was reached. Even a bunch of clues to
// describe system state may not be enough.  Sometimes what you need in order
// to debug the situation is a long-form explanation (you do already add that
// to your code, don't you?).  Or, even better, a linear history of long-form
// explanations, each one building on the prior (which you can't easily do in
// code).
//
// Should you transfer all your comments to clues?  Absolutely not.  But in
// cases where extra explantion is truly important to debugging production,
// when all you've got are some logs and (maybe if you're lucky) a span trace?
// Those are the ones you want.
//
// Unlike other additions, which are added as top-level key:value pairs to the
// context, comments are all held as a single array of additions, persisted in
// order of appearance, and prefixed by the file and line in which they appeared.
// This means comments are always added to the context and never clobber each
// other, regardless of their location.  IE: don't add them to a loop.
func AddComment[CTX valuer](
	ctx CTX,
	msg string,
	vs ...any,
) CTX {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, nc.addComment(1, msg, vs...))
}

// AddCommentTo adds a long form comment to the clues in a certain namespace.
//
// Comments are special case additions to the context.  They're here to, well,
// let you add comments!  Why?  Because sometimes it's not sufficient to have a
// log let you know that a line of code was reached. Even a bunch of clues to
// describe system state may not be enough.  Sometimes what you need in order
// to debug the situation is a long-form explanation (you do already add that
// to your code, don't you?).  Or, even better, a linear history of long-form
// explanations, each one building on the prior (which you can't easily do in
// code).
//
// Should you transfer all your comments to clues?  Absolutely not.  But in
// cases where extra explantion is truly important to debugging production,
// when all you've got are some logs and (maybe if you're lucky) a span trace?
// Those are the ones you want.
//
// Unlike other additions, which are added as top-level key:value pairs to the
// context, comments are all held as a single array of additions, persisted in
// order of appearance, and prefixed by the file and line in which they appeared.
// This means comments are always added to the context and never clobber each
// other, regardless of their location.  IE: don't add them to a loop.
func AddCommentTo[CTX valuer](
	ctx CTX,
	namespace, msg string,
	vs ...any,
) CTX {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	return setNodeInCtx(ctx, namespace, nc.addComment(1, msg, vs...))
}

// ---------------------------------------------------------------------------
// agents
// ---------------------------------------------------------------------------

// AddAgent adds an agent with a given name to the context.  What's an agent?
// It's a special case data adder that you can spawn to collect clues for
// you.  Unlike standard clues additions, you have to tell the agent exactly
// what data you want it to Relay() for you.
//
// Agents are recorded in the current clues node and all of its descendants.
// Data relayed by the agent will appear as part of the standard data map,
// namespaced by each agent.
//
// Agents are specifically handy in a certain set of uncommon cases where
// retrieving clues is otherwise difficult to do, such as working with
// middleware that doesn't allow control over error creation.  In these cases
// your only option is to relay that data back to some prior clues node.
func AddAgent[CTX valuer](
	ctx CTX,
	name string,
) CTX {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, nc.addAgent(name))
}

// Relay adds all key-value pairs to the provided agent.  The agent will
// record those values to the dataNode in which it was created.  All relayed
// values are namespaced to the owning agent.
func Relay[CTX valuer](
	ctx CTX,
	agent string,
	kvs ...any,
) {
	relay(
		nodeFromCtx(ctx, defaultNamespace),
		agent,
		kvs...)
}

// ---------------------------------------------------------------------------
// error label counter
// ---------------------------------------------------------------------------

// AddLabelCounter embeds an Adder interface into this context. Any already
// embedded Adder will get replaced.  When adding Labels to a clues.Err the
// LabelCounter will use the label as the key for the Add call, and increment
// the count of that label by one.
func AddLabelCounter[CTX valuer](
	ctx CTX,
	counter Adder,
) CTX {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, addLabelCounter(nc, counter))
}

// AddLabelCounterTo embeds an Adder interface into this context. Any already
// embedded Adder will get replaced.  When adding Labels to a clues.Err the
// LabelCounter will use the label as the key for the Add call, and increment
// the count of that label by one.
func AddLabelCounterTo[CTX valuer](
	ctx CTX,
	namespace string,
	counter Adder,
) CTX {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	return setNodeInCtx(ctx, namespace, addLabelCounter(nc, counter))
}
