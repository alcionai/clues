package clues

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/alcionai/clues/internal/stringify"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
)

type Adder interface {
	Add(key string, n int64)
}

// ---------------------------------------------------------------------------
// nodes
// ---------------------------------------------------------------------------

// dataNodes contain the data tracked by both clues in contexts and in errors.
//
// These nodes compose a tree, such that nodes can walk their ancestry path from
// leaf (the current node) to root (the highest ancestor), but not from root to
// child.  This allows clues to establish sets of common ancestor data with unique
// branches for individual descendants, making the addition of new data inherently
// theadsafe.
//
// For collisions during aggregation, distance from the root denotes priority,
// with the root having the lowest priority.  IE: if a child overwrites a key
// declared by an ancestor, the child's entry takes priority.
type dataNode struct {
	id     int32
	parent int32
	depth  int

	// span is the current otel Span.
	// Spans are kept separately from the otelClient because we want the client to
	// maintain a consistent reference to otel initialization, while the span can
	// get replaced at arbitrary points.
	span trace.Span

	// name is optional and is used primarily for tracing markers.
	// if empty, the trace for that node will get skipped when building the
	// full trace along the node's ancestry path in the tree.
	name string

	// values are they arbitrary key:value pairs that appear in clues when callers
	// use the Add(ctx, k, v) or err.With(k, v) adders.  Each key-value pair added
	// to the node is used to produce the final set of Values() in the dataNode,
	// with lower nodes in the tree taking priority over higher nodes for any
	// collision resolution.
	values map[string]any

	// each node can hold a single commment.  The history of comments produced
	// by the ancestry path through the tree will get concatenated from oldest
	// ancestor to the current node to produce the comment history.
	comment comment

	// agents act as proxy dataNodes that can relay specific, intentional data
	// additions.  They're namespaced so that additions to the agents don't accidentally
	// clobber other values in the dataNode. This also allows agents to protect
	// variations of data from each other, in case users need to compare differences
	// on the same keys.  That's not the goal for agents, exactly, but it is capable.
	agents map[string]*agent
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type nodeCtxKey string

const defaultNodeKey nodeCtxKey = "default_node_ctx_key"

func ctxKey(namespace string) nodeCtxKey {
	return nodeCtxKey(namespace)
}

// nodeIDFromCtx retrieves the node ID from the ctx.
func nodeIDFromCtx(ctx context.Context) int32 {
	dn := ctx.Value(defaultNodeKey)

	if dn == nil {
		return -1
	}

	return dn.(int32)
}

// setNodeIDInCtx adds the node id to the context and returns the updated context.
func setNodeIDInCtx(ctx context.Context, id int32) context.Context {
	return context.WithValue(ctx, defaultNodeKey, id)
}

type regNode struct {
	ok  bool
	id  int32
	reg *registry
}

// nodeFromCtx retrieves a usable node refrerence and registry from the context.
// If a registry is already initialized, the context is returned unchanged.
// If no reistry exists, a new one is created and injected into the ctx.
func nodeFromCtx(ctx context.Context) (context.Context, regNode) {
	ctx, reg := registryFromCtx(ctx)
	id := nodeIDFromCtx(ctx)

	rn := regNode{
		ok:  id >= 0 && reg != nil,
		id:  id,
		reg: reg,
	}

	return ctx, rn
}

func (rn regNode) node() dataNode {
	if !rn.ok {
		return dataNode{}
	}

	return rn.reg.nodes[rn.id]
}

// ---------------------------------------------------------------------------
// setters
// ---------------------------------------------------------------------------

// newNodeWithValuesAndAttributes adds all entries in the map to the dataNode's values.
// automatically propagates values onto the current span.
func (rn regNode) newNodeWithValuesAndAttributes(
	m map[string]any,
) (dataNode, bool) {
	if !rn.ok {
		return dataNode{}, false
	}

	dn := rn.reg.spawnDescendant(rn.id)

	rn.id = dn.id
	rn.addValues(m)
	rn.addSpanAttributes(m)

	return dn, true
}

// addValues adds the values to the specified node.
func (rn regNode) addValues(m map[string]any) {
	if len(m) == 0 || !rn.ok {
		return
	}

	dn := rn.node()

	if len(dn.values) == 0 {
		dn.values = map[string]any{}
		rn.reg.nodes[dn.id] = dn
	}

	maps.Copy(dn.values, m)
}

func (rn regNode) nameNode(
	nodeID int32,
	name string,
) {
	if !rn.ok {
		return
	}

	dn := rn.reg.nodes[nodeID]

	if name == "" {
		name = uuid.NewString()[:8]
	}

	dn.name = name

	rn.reg.nodes[dn.id] = dn
}

// ---------------------------------------------------------------------------
// comments
// ---------------------------------------------------------------------------

type comment struct {
	// the func name in which the comment was created.
	Caller string
	// the name of the file owning the caller.
	File string
	// the comment message itself.
	Message string
}

// shorthand for checking if an empty comment was generated.
func (c comment) isEmpty() bool {
	return len(c.Message) == 0
}

// newComment formats the provided values, and grabs the caller and trace
// info according to the depth.  Depth is a skip-caller count, and any func
// calling this one should provide either `1` (for itself) or `depth+1` (if
// it was already given a depth value).
func newComment(
	depth int,
	template string,
	values ...any,
) comment {
	caller := getCaller(depth + 1)
	_, _, parentFileLine := getDirAndFile(depth + 1)

	return comment{
		Caller:  caller,
		File:    parentFileLine,
		Message: fmt.Sprintf(template, values...),
	}
}

// addComment creates a new dataNode with a comment but no other properties.
func (rn regNode) addComment(
	depth int,
	msg string,
	vs ...any,
) (dataNode, bool) {
	if len(msg) == 0 || !rn.ok {
		return dataNode{}, false
	}

	dn := rn.reg.spawnDescendant(rn.id)
	dn.comment = newComment(depth+1, msg, vs...)

	return dn, true
}

// comments allows us to put a stringer on a slice of comments.
type comments []comment

// String formats the slice of comments as a stack, much like you'd see
// with an error stacktrace.  Comments are listed top-to-bottom from first-
// to-last.
//
// The format for each comment in the stack is:
//
//	<caller> - <file>:<line>
//	  <message>
func (cs comments) String() string {
	result := []string{}

	for _, c := range cs {
		result = append(result, c.Caller+" - "+c.File)
		result = append(result, "\t"+c.Message)
	}

	return strings.Join(result, "\n")
}

// Comments retrieves the full ancestor comment chain.
// The return value is ordered from the first added comment (closest to
// the root) to the most recent one (closest to the leaf).
func (rn regNode) Comments() comments {
	if !rn.ok {
		return comments{}
	}

	result := comments{}

	fn := fnNoder{
		fn: func(dn dataNode) {
			if !dn.comment.isEmpty() {
				result = append(result, dn.comment)
			}
		},
	}

	rn.reg.iterNodesRootToLeaf(rn.id, fn)

	return result
}

// ---------------------------------------------------------------------------
// agents
// ---------------------------------------------------------------------------

// FIXME: need to move agents into a separate map in the registry
type agent struct {
	// the name of the agent
	id string

	// dataNode is used here instead of a basic value map so that
	// we can extend the usage of agents in the future by allowing
	// the full set of dataNode behavior.  We'll need a builder for that,
	// but we'll get there eventually.
	data *dataNode
}

// addAgent adds a new named agent to the dataNode.
func (rn regNode) addAgent(name string) (dataNode, bool) {
	if !rn.ok {
		return dataNode{}, false
	}

	dn := rn.reg.spawnDescendant(rn.id)

	if len(dn.agents) == 0 {
		dn.agents = map[string]*agent{}
	}

	dn.agents[name] = &agent{
		id: name,
		// no spawn here, this needs an isolated node
		data: &dataNode{},
	}

	return dn, true
}

func (ag agent) addValues(
	m map[string]any,
) {
	if len(m) == 0 {
		return
	}

	if len(ag.data.values) == 0 {
		ag.data.values = map[string]any{}
	}

	maps.Copy(ag.data.values, m)
}

// ---------------------------------------------------------------------------
// otel
// ---------------------------------------------------------------------------

// OTELLogger gets the otel logger instance from the otel client.
// Returns nil if otel wasn't initialized.
func (rn regNode) OTELLogger() otellog.Logger {
	// TODO: can I pull this out of the ctx?
	if !rn.ok || rn.reg.otel == nil {
		return nil
	}

	return rn.reg.otel.logger
}

// ------------------------------------------------------------
// span handling
// ------------------------------------------------------------

// traceMapCarrierBase defines the structures that support
// otel traceMapCarrier behavior.  A traceMapCarrier is used
// to pass and receive traces using message delivery headers
// and other metadata.
type traceMapCarrierBase interface {
	map[string]string | http.Header
}

// asTraceMapCarrier converts a traceMapCarrier interface to
// its propagation package implementation for that structure.
// ie: map becomes a MapCarrier, headers become HeaderCarriers.
func asTraceMapCarrier[C traceMapCarrierBase](
	carrier C,
) propagation.TextMapCarrier {
	if carrier == nil {
		return propagation.MapCarrier{}
	}

	if mss, ok := any(carrier).(map[string]string); ok {
		return propagation.MapCarrier(mss)
	}

	if hh, ok := any(carrier).(http.Header); ok {
		return propagation.HeaderCarrier(hh)
	}

	return propagation.MapCarrier{}
}

// injectTrace adds the current trace details to the provided
// carrier.  If otel is not initialized, no-ops.
//
// The carrier data is mutated by this call.
func (rn regNode) injectTrace(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) {
	if !rn.ok {
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// receiveTrace extracts the current trace details from the
// carrier and adds them to the context.  If otel is not
// initialized, no-ops.
//
// The carrier data is mutated by this call.
func (rn regNode) receiveTrace(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) context.Context {
	if !rn.ok {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// newNodeWithSpan adds a new otel span.  If the otel client is nil, no-ops.
// Attrs can be added to the span with addSpanAttrs.  This span will
// continue to be used for that purpose until replaced with another
// span, which will appear in a separate context (and thus a separate,
// dataNode).
func (rn regNode) newNodeWithSpan(
	ctx context.Context,
	name string,
) (context.Context, dataNode, bool) {
	if !rn.ok {
		return ctx, dataNode{}, false
	}

	dn := rn.reg.spawnDescendant(rn.id)
	dn.name = name

	if rn.reg.otel == nil {
		rn.reg.nodes[dn.id] = dn
		return ctx, dn, true
	}

	ctx, span := rn.reg.otel.tracer.Start(ctx, name)
	dn.span = span

	rn.reg.nodes[dn.id] = dn

	return ctx, dn, true
}

// closeSpan closes the otel span and removes it span from the data node.
// If no span is present, no ops.
func (rn regNode) closeSpan() {
	if !rn.ok {
		return
	}

	dn := rn.node()

	if dn.span != nil {
		dn.span.End()
	}

	return
}

// addSpanAttributes adds the values to the current span.  If the span
// is nil (such as if otel wasn't initialized or no span has been generated),
// this call no-ops.
func (rn regNode) addSpanAttributes(values map[string]any) {
	if !rn.ok {
		return
	}

	dn := rn.node()

	if dn.span == nil {
		return
	}

	for k, v := range values {
		// FIXME: otel typed attributes. just need a lib conversion.
		v := stringify.Marshal(v, false)
		attr := attribute.String(k, v)

		dn.span.SetAttributes(attr)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// getDirAndFile retrieves the file and line number of the caller.
// Depth is the skip-caller count.  Clues funcs that call this one should
// provide either `1` (if they do not already have a depth value), or `depth+1`
// otherwise`.
//
// formats:
// dir `absolute/os/path/to/parent/folder`
// fileAndLine `<file>:<line>`
// parentAndFileAndLine `<parent>/<file>:<line>`
//
// TODO: This is alloc heavy and needs to be lightened
func getDirAndFile(
	depth int,
) (dir, fileAndLine, parentAndFileAndLine string) {
	_, file, line, _ := runtime.Caller(depth + 1)
	dir, file = path.Split(file)

	fileLine := fmt.Sprintf("%s:%d", file, line)
	parentFileLine := fileLine

	parent := path.Base(dir)
	if len(parent) > 0 {
		parentFileLine = path.Join(parent, fileLine)
	}

	return dir, fileLine, parentFileLine
}

// getCaller retrieves the func name of the caller. Depth is the  skip-caller
// count.  Clues funcs that call this one should provide either `1` (if they
// do not already have a depth value), or `depth+1` otherwise.`
//
// TODO: This is alloc heavy and needs to be lightened
func getCaller(depth int) string {
	pc, _, _, ok := runtime.Caller(depth + 1)
	if !ok {
		return ""
	}

	funcPath := runtime.FuncForPC(pc).Name()

	// the funcpath base looks something like this:
	// prefix.funcName[...].foo.bar
	// with the [...] only appearing for funcs with generics.
	base := path.Base(funcPath)

	// so when we split it into parts by '.', we get
	// [prefix, funcName[, ], foo, bar]
	parts := strings.Split(base, ".")

	// in certain conditions we'll only get the funcName
	// itself, without the other parts.  In that case, we
	// just need to strip the generic portion from the base.
	if len(parts) < 2 {
		return strings.ReplaceAll(base, "[...]", "")
	}

	// in most cases we'll take the 1th index (the func
	// name) and trim off the bracket that remains from
	// splitting on a period.
	return strings.TrimSuffix(parts[1], "[")
}

// ---------------------------------------------------------------------------
// serialization
// ---------------------------------------------------------------------------

// nodeCore contains the serializable set of data in a dataNode.
type nodeCore struct {
	OTELServiceName string `json:"otelServiceName"`
	// TODO: investigate if map[string]string is really the best structure here.
	// maybe we can get away with a map[string]any, or a []byte slice?
	Values   map[string]string `json:"values"`
	Comments []comment         `json:"comments"`
}

// Bytes serializes the dataNode to a slice of bytes.
// Only attributes and comments are maintained.  All
// values are stringified in the process.
//
// Node hierarchy, clients (such as otel), agents, and
// hooks (such as labelCounter) are all sliced from the
// result.
func (rn regNode) Bytes() ([]byte, error) {
	if !rn.ok {
		return []byte{}, nil
	}

	var serviceName string

	if rn.reg.otel != nil {
		serviceName = rn.reg.otel.serviceName
	}

	core := nodeCore{
		OTELServiceName: serviceName,
	}

	return json.Marshal(core)
}

// FromBytes deserializes the bytes to a new dataNode.
// No clients, agents, or hooks are initialized in this process.
func FromBytes(
	bs []byte,
) (*dataNode, error) {
	core := nodeCore{}

	err := json.Unmarshal(bs, &core)
	if err != nil {
		return nil, err
	}

	node := dataNode{
		// FIXME: do something with the serialized commments.
		// I'm punting on this for now because I want to figure
		// out the best middle ground between avoiding a slice of
		// comments in each node for serialization sake (they
		// are supposed to be one-comment-per-node to use the tree
		// for ordering instead of the parameter), and keeping
		// the full comment history available.  Probably just
		// need to introduce a delimiter.
	}

	if len(core.Values) > 0 {
		node.values = map[string]any{}
	}

	for k, v := range core.Values {
		node.values[k] = v
	}

	// if len(core.OTELServiceName) > 0 {
	// node.otel = &otelClient{
	// 	serviceName: core.OTELServiceName,
	// }
	// }

	return &node, nil
}
