package clues

import (
	"context"
)

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

// AddTrace stacks a clues node onto this context.  Adding a node ensures
// that this point in code is identified by an ID, which can later be
// used to correlate and isolate logs to certain trace branches.
// AddTrace is only needed for layers that don't otherwise call Add() or
// similar functions, since those funcs already attach a new node.
func AddTrace(ctx context.Context) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)
	return setDefaultNodeInCtx(ctx, nc.trace(""))
}

// AddTraceTo stacks a clues node onto this context within the specified
// namespace.  Adding a node ensures that a point in code is identified
// by an ID, which can later be used to correlate and isolate logs to
// certain trace branches.  AddTraceTo is only needed for layers that don't
// otherwise call AddTo() or similar functions, since those funcs already
// attach a new node.
func AddTraceTo(ctx context.Context, namespace string) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))
	return setNodeInCtx(ctx, namespace, nc.trace(""))
}

// AddTraceName stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddTraceName
// can be called without additional values if you only want to add a trace marker.
func AddTraceName(
	ctx context.Context,
	name string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, defaultNamespace)

	var node *dataNode
	if len(kvs) > 0 {
		node = nc.addValues(normalize(kvs...))
		node.id = name
	} else {
		node = nc.trace(name)
	}

	return setDefaultNodeInCtx(ctx, node)
}

// AddTraceNameTo stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddTraceNameTo
// can be called without additional values if you only want to add a trace marker.
func AddTraceNameTo(
	ctx context.Context,
	name, namespace string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx, ctxKey(namespace))

	var node *dataNode
	if len(kvs) > 0 {
		node = nc.addValues(normalize(kvs...))
		node.id = name
	} else {
		node = nc.trace(name)
	}

	return setNodeInCtx(ctx, namespace, node)
}

// ---------------------------------------------------------------------------
// comments
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// hooks
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
