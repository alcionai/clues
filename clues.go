package clues

import (
	"context"

	"github.com/alcionai/clues/internal/stringify"
)

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
// traces
// ---------------------------------------------------------------------------

// AddTrace stacks a clues node onto this context.  Adding a node ensures
// that this point in code is identified by an ID, which can later be
// used to correlate and isolate logs to certain trace branches.
// AddTrace is only needed for layers that don't otherwise call Add() or
// similar functions, since those funcs already attach a new node.
func AddTrace(
	ctx context.Context,
	traceID string,
) context.Context {
	nc := nodeFromCtx(ctx)
	return setNodeInCtx(ctx, nc.trace(traceID))
}

// AddTraceWith stacks a clues node onto this context and uses the provided
// name for the trace id, instead of a randomly generated hash. AddTraceWith
// can be called without additional values if you only want to add a trace marker.
func AddTraceWith(
	ctx context.Context,
	traceID string,
	kvs ...any,
) context.Context {
	nc := nodeFromCtx(ctx)

	var node *dataNode
	if len(kvs) > 0 {
		node = nc.addValues(stringify.Normalize(kvs...))
		node.id = traceID
	} else {
		node = nc.trace(traceID)
	}

	return setNodeInCtx(ctx, node)
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
