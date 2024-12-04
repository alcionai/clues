package clues

import (
	"context"
	"fmt"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
	"go.opentelemetry.io/otel/trace"
)

// ---------------------------------------------------------------------------
// persistent client initialization
// ---------------------------------------------------------------------------

// InitializeOTEL will spin up the OTEL clients that are held by clues,
// Clues will eagerly use these clients in the background to provide
// additional telemetry hook-ins.
//
// Clues will operate as expected in the event of an error, or if OTEL is not
// initialized. This is a purely optional step.
func InitializeOTEL(
	ctx context.Context,
	serviceName string,
	config OTELConfig,
) (context.Context, error) {
	nc := node.FromCtx(ctx)

	err := nc.InitOTEL(ctx, serviceName, config.toInternalConfig())
	if err != nil {
		return ctx, err
	}

	return node.EmbedInCtx(ctx, nc), nil
}

// Close will flush all buffered data waiting to be read.  If Initialize was not
// called, this call is a no-op.  Should be called in a defer after initializing.
func Close(ctx context.Context) error {
	nc := node.FromCtx(ctx)

	if nc.OTEL != nil {
		err := nc.OTEL.Close(ctx)
		if err != nil {
			return fmt.Errorf("closing otel client: %w", err)
		}
	}

	return nil
}

// Inherit propagates all clients from one context to another.  This is particularly
// useful for taking an initialized context from a main() func and ensuring its clients
// are available for request-bound conetxts, such as in a http server pattern.
//
// If the 'to' context already contains an initialized client, no change is made.
// Callers can force a 'from' client to override a 'to' client by setting clobber=true.
func Inherit(
	from, to context.Context,
	clobber bool,
) context.Context {
	fromNode := node.FromCtx(from)
	toNode := node.FromCtx(to)

	if to == nil {
		to = context.Background()
	} else if toNode.Span == nil {
		// A span may already exist in the 'to' context thanks to otel package integration.
		// Likewise, the 'from' ctx is not expected to contain a span, so we only want to
		// maintain the span information that's currently live.
		toNode.Span = trace.SpanFromContext(to)
	}

	// if we have no fromNode OTEL, or are not clobbering, return the toNode.
	if fromNode.OTEL == nil || (toNode.OTEL != nil && !clobber) {
		return node.EmbedInCtx(to, toNode)
	}

	// otherwise pass along the fromNode OTEL client.
	toNode.OTEL = fromNode.OTEL

	return node.EmbedInCtx(to, toNode)
}

// ---------------------------------------------------------------------------
// data access
// ---------------------------------------------------------------------------

// In retrieves the clues structured data from the context.
func In(ctx context.Context) *node.Node {
	return node.FromCtx(ctx)
}

// ---------------------------------------------------------------------------
// key-value metadata
// ---------------------------------------------------------------------------

// Add adds all key-value pairs to the clues.
func Add(ctx context.Context, kvs ...any) context.Context {
	nc := node.FromCtx(ctx)
	return node.EmbedInCtx(ctx, nc.AddValues(stringify.Normalize(kvs...)))
}

// AddMap adds a shallow clone of the map to a namespaced set of clues.
func AddMap[K comparable, V any](
	ctx context.Context,
	m map[K]V,
) context.Context {
	nc := node.FromCtx(ctx)

	kvs := make([]any, 0, len(m)*2)
	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return node.EmbedInCtx(ctx, nc.AddValues(stringify.Normalize(kvs...)))
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
func InjectTrace[C node.TraceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) C {
	node.FromCtx(ctx).
		InjectTrace(ctx, node.AsTraceMapCarrier(mapCarrier))

	return mapCarrier
}

// ReceiveTrace extracts the current trace details from the
// headers and adds them to the context.  If otel is not
// initialized, no-ops.
func ReceiveTrace[C node.TraceMapCarrierBase](
	ctx context.Context,
	mapCarrier C,
) context.Context {
	return node.FromCtx(ctx).
		ReceiveTrace(ctx, node.AsTraceMapCarrier(mapCarrier))
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
	nc := node.FromCtx(ctx)

	var spanned *node.Node

	if len(kvs) > 0 {
		ctx, spanned = nc.AddSpan(ctx, name)
		spanned.ID = name
		spanned = spanned.AddValues(stringify.Normalize(kvs...))
	} else {
		ctx, spanned = nc.AddSpan(ctx, name)
		spanned = spanned.AppendToTree(name)
	}

	fmt.Println("GET SPANNED, YO")

	return node.EmbedInCtx(ctx, spanned)
}

// CloseSpan closes the current span in the clues node.  Should only be called
// following a `clues.AddSpan()` call.
func CloseSpan(ctx context.Context) context.Context {
	return node.EmbedInCtx(
		ctx,
		node.FromCtx(ctx).CloseSpan(ctx))
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
	nc := node.FromCtx(ctx)
	nn := nc.AddComment(1, msg, vs...)

	return node.EmbedInCtx(ctx, nn)
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
	nc := node.FromCtx(ctx)
	nn := nc.AddAgent(name)

	return node.EmbedInCtx(ctx, nn)
}

// Relay adds all key-value pairs to the provided agent.  The agent will
// record those values to the node in which it was created.  All relayed
// values are namespaced to the owning agent.
func Relay(
	ctx context.Context,
	agent string,
	vs ...any,
) {
	nc := node.FromCtx(ctx)
	ag, ok := nc.Agents[agent]

	if !ok {
		return
	}

	// set values, not add.  We don't want agents to own a full clues tree.
	ag.Data.SetValues(stringify.Normalize(vs...))
}
