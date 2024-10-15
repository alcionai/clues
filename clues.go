package clues

import (
	"context"
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
func Initialize(ctx context.Context, serviceName string) (context.Context, error) {
	nc := nodeFromCtx(ctx, defaultNamespace)

	err := nc.init(ctx, serviceName)
	if err != nil {
		return ctx, err
	}

	return setDefaultNodeInCtx(ctx, nc), nil
}

// ---------------------------------------------------------------------------
// key-value metadata
// ---------------------------------------------------------------------------

// Add adds all key-value pairs to the clues.
func Add(ctx context.Context, kvs ...any) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, nc.addValues(normalize(kvs...)))
}

// AddTo adds all key-value pairs to a namespaced set of clues.
func AddTo(
	ctx context.Context,
	namespace string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	return setNodeInCtx(ctx, namespace, nc.addValues(normalize(kvs...)))
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[K comparable, V any](
	ctx context.Context,
	m map[K]V,
) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return setDefaultNodeInCtx(ctx, nc.addValues(normalize(kvs...)))
}

// AddMapTo adds a shallow clone of the map to a namespaced set of clues.
func AddMapTo[K comparable, V any](
	ctx context.Context,
	namespace string,
	m map[K]V,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return setNodeInCtx(ctx, namespace, nc.addValues(normalize(kvs...)))
}

// ---------------------------------------------------------------------------
// traces
// ---------------------------------------------------------------------------

// AddSpan stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddSpan
// can be called without additional values if you only want to add a trace
// marker.  The assumption is that an otel span is generated and attached
// to the node.  Callers should always follow this addition with a closing
// `defer clues.CloseSpan(ctx)`.
//
// --- NOTICE ----
// This is an alpha feature, and may not yet work properly.
func AddSpan(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)

	var node *dataNode
	if len(kvs) > 0 {
		ctx, node = nc.addSpan(ctx, name)
		node.id = name
		node = node.addValues(normalize(kvs...))
	} else {
		ctx, node = nc.addSpan(ctx, name)
		node = node.trace(name)
	}

	return setDefaultNodeInCtx(ctx, node)
}

// AddSpanTo stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddSpanTo
// can be called without additional values if you only want to add a trace
// marker.  The assumption is that an otel span is generated and attached
// to the node.  Callers should always follow this addition with a closing
// `defer clues.CloseSpan(ctx)`.
//
// --- NOTICE ----
// This is an alpha feature, and may not yet work properly.
func AddSpanTo(
	ctx context.Context,
	name, namespace string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))

	var node *dataNode
	if len(kvs) > 0 {
		ctx, node = nc.addSpan(ctx, name)
		node.id = name
		node = node.addValues(normalize(kvs...))
	} else {
		ctx, node = nc.addSpan(ctx, name)
		node = node.trace(name)
	}

	return setNodeInCtx(ctx, namespace, node)
}

// CloseSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func CloseSpan(ctx context.Context) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)

	node := nc.closeSpan()

	return setDefaultNodeInCtx(ctx, node)
}

// CloseSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func CloseSpanTo(
	ctx context.Context,
	namespace string,
) context.Context {
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
func AddComment(
	ctx context.Context,
	msg string,
	vs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)
	nn := nc.addComment(1, msg, vs...)

	return setDefaultNodeInCtx(ctx, nn)
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
func AddCommentTo(
	ctx context.Context,
	namespace, msg string,
	vs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	nn := nc.addComment(1, msg, vs...)

	return setNodeInCtx(ctx, namespace, nn)
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
	nc := nodeFromCtx(ctx, defaultNamespace)
	nn := nc.addAgent(name)

	return setDefaultNodeInCtx(ctx, nn)
}

// Relay adds all key-value pairs to the provided agent.  The agent will
// record those values to the dataNode in which it was created.  All relayed
// values are namespaced to the owning agent.
func Relay(
	ctx context.Context,
	agent string,
	vs ...any,
) {
	nc := nodeFromCtx(ctx, defaultNamespace)
	ag, ok := nc.agents[agent]

	if !ok {
		return
	}

	// set values, not add.  We don't want agents to own a full clues tree.
	ag.data.setValues(normalize(vs...))
}

// ---------------------------------------------------------------------------
// error label counter
// ---------------------------------------------------------------------------

// AddLabelCounter embeds an Adder interface into this context. Any already
// embedded Adder will get replaced.  When adding Labels to a clues.Err the
// LabelCounter will use the label as the key for the Add call, and increment
// the count of that label by one.
func AddLabelCounter(ctx context.Context, counter Adder) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)
	nn := nc.addValues(nil)
	nn.labelCounter = counter

	return setDefaultNodeInCtx(ctx, nn)
}

// AddLabelCounterTo embeds an Adder interface into this context. Any already
// embedded Adder will get replaced.  When adding Labels to a clues.Err the
// LabelCounter will use the label as the key for the Add call, and increment
// the count of that label by one.
func AddLabelCounterTo(
	ctx context.Context,
	namespace string,
	counter Adder,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	nn := nc.addValues(nil)
	nn.labelCounter = counter

	return setNodeInCtx(ctx, namespace, nn)
}
