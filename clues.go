package clues

import (
	"context"

	"github.com/alcionai/clues/internal/stringify"
)

// ---------------------------------------------------------------------------
// persistent client initialization
// ---------------------------------------------------------------------------

// Initialize will spin up any persistent clients that are held by clues,
// such as OTEL communication.  Clues will use these optimistically in the
// background to provide additional telemetry hook-ins.
//
// Clues will operate as expected in the event of an error, or if initialization
// is not called.  This is a purely optional step.
func Initialize(
	ctx context.Context,
	serviceName string,
	config OTELConfig,
) (context.Context, error) {
	nc := nodeFromCtx(ctx)

	err := nc.init(ctx, serviceName, config)
	if err != nil {
		return ctx, err
	}

	return setNodeInCtx(ctx, nc), nil
}

// Close will flush all buffered data waiting to be read.  If Initialize was not
// called, this call is a no-op.  Should be called in a defer after initializing.
func Close(ctx context.Context) error {
	nc := nodeFromCtx(ctx)

	if nc.otel != nil {
		err := nc.otel.close(ctx)
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
func Add(ctx context.Context, kvs ...any) context.Context {
	nc := nodeFromCtx(ctx)
	return setNodeInCtx(ctx, nc.addValues(stringify.Normalize(kvs...)))
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[K comparable, V any](
	ctx context.Context,
	m map[K]V,
) context.Context {
	nc := nodeFromCtx(ctx)

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return setNodeInCtx(ctx, nc.addValues(stringify.Normalize(kvs...)))
}

// ---------------------------------------------------------------------------
// spans and traces
// ---------------------------------------------------------------------------

// InjectTrace adds the current trace details to the provided
// headers.  If otel is not initialized, no-ops.
//
// The mapCarrier is mutated by this request.  The passed
// reference is returned mostly as a quality-of-life step
// so that callers don't need to declare the map outside of
// this call.
func InjectTrace[C traceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) C {
	nodeFromCtx(ctx).
		injectTrace(ctx, asTraceMapCarrier(mapCarrier))

	return mapCarrier
}

// ReceiveTrace extracts the current trace details from the
// headers and adds them to the context.  If otel is not
// initialized, no-ops.
func ReceiveTrace[C traceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) context.Context {
	return nodeFromCtx(ctx).
		receiveTrace(ctx, asTraceMapCarrier(mapCarrier))
}

// AddSpan stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddSpan
// can be called without additional values if you only want to add a trace
// marker.  The assumption is that an otel span is generated and attached
// to the node.  Callers should always follow this addition with a closing
// `defer clues.CloseSpan(ctx)`.
func AddSpan(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx)

	var node *dataNode

	if len(kvs) > 0 {
		ctx, node = nc.addSpan(ctx, name)
		node.id = name
		node = node.addValues(stringify.Normalize(kvs...))
	} else {
		ctx, node = nc.addSpan(ctx, name)
		node = node.trace(name)
	}

	return setNodeInCtx(ctx, node)
}

// CloseSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func CloseSpan(ctx context.Context) context.Context {
	nc := nodeFromCtx(ctx).closeSpan(ctx)
	return setNodeInCtx(ctx, nc)
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
func AddComment(
	ctx context.Context,
	msg string,
	vs ...any,
) context.Context {
	nc := nodeFromCtx(ctx)
	nn := nc.addComment(1, msg, vs...)

	return setNodeInCtx(ctx, nn)
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
func AddAgent(
	ctx context.Context,
	name string,
) context.Context {
	nc := nodeFromCtx(ctx)
	nn := nc.addAgent(name)

	return setNodeInCtx(ctx, nn)
}

// Relay adds all key-value pairs to the provided agent.  The agent will
// record those values to the dataNode in which it was created.  All relayed
// values are namespaced to the owning agent.
func Relay(
	ctx context.Context,
	agent string,
	vs ...any,
) {
	nc := nodeFromCtx(ctx)
	ag, ok := nc.agents[agent]

	if !ok {
		return
	}

	// set values, not add.  We don't want agents to own a full clues tree.
	ag.data.setValues(stringify.Normalize(vs...))
}
