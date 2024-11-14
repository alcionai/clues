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
	"golang.org/x/exp/slices"
)

// ---------------------------------------------------------------------------
// data nodes
// ---------------------------------------------------------------------------

type Adder interface {
	Add(key string, n int64)
}

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
	parent *dataNode

	// otel contains the client instance for the in memory otel runtime.  It is only
	// present if the end user calls the clues initialization step.
	otel *otelClient

	// span is the current otel Span.
	// Spans are kept separately from the otelClient because we want the client to
	// maintain a consistent reference to otel initialization, while the span can
	// get replaced at arbitrary points.
	span trace.Span

	// ids are optional and are used primarily as tracing markers.
	// if empty, the trace for that node will get skipped when building the
	// full trace along the node's ancestry path in the tree.
	id string

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

// spawnDescendant generates a new dataNode that is a descendant of the current
// node.  A descendant maintains a pointer to its parent, and carries any genetic
// necessities (ie, copies of fields) that must be present for continued functionality.
func (dn *dataNode) spawnDescendant() *dataNode {
	agents := maps.Clone(dn.agents)

	if agents == nil && dn.agents != nil {
		agents = map[string]*agent{}
	}

	return &dataNode{
		parent: dn,
		otel:   dn.otel,
		span:   dn.span,
		agents: agents,
	}
}

// ---------------------------------------------------------------------------
// setters
// ---------------------------------------------------------------------------

// addValues adds all entries in the map to the dataNode's values.
// automatically propagates values onto the current span.
func (dn *dataNode) addValues(m map[string]any) *dataNode {
	if m == nil {
		m = map[string]any{}
	}

	spawn := dn.spawnDescendant()
	spawn.setValues(m)
	spawn.addSpanAttributes(m)

	return spawn
}

// extendValues adds all entries in the map to the dataNode's values.
// automatically propagates values onto the current span.
func (dn *dataNode) extendValues(m map[string]any) *dataNode {
	if m == nil {
		m = map[string]any{}
	}

	dn.setValues(m)

	return dn
}

// setValues is a helper called by addValues.
func (dn *dataNode) setValues(m map[string]any) {
	if len(m) == 0 {
		return
	}

	if len(dn.values) == 0 {
		dn.values = map[string]any{}
	}

	maps.Copy(dn.values, m)
}

// trace adds a new leaf containing a trace ID and no other values.
func (dn *dataNode) trace(name string) *dataNode {
	if name == "" {
		name = makeNodeID()
	}

	spawn := dn.spawnDescendant()
	spawn.id = name

	return spawn
}

// ---------------------------------------------------------------------------
// getters
// ---------------------------------------------------------------------------

// lineage runs the fn on every valueNode in the ancestry tree,
// starting at the root and ending at the dataNode.
func (dn *dataNode) lineage(fn func(id string, vs map[string]any)) {
	if dn == nil {
		return
	}

	if dn.parent != nil {
		dn.parent.lineage(fn)
	}

	fn(dn.id, dn.values)
}

// In returns the default dataNode from the context.
// TODO: turn return an interface instead of a dataNode, have dataNodes
// and errors both comply with that wrapper.
func In(ctx context.Context) *dataNode {
	return nodeFromCtx(ctx)
}

// Map flattens the tree of dataNode.values into a map.  Descendant nodes
// take priority over ancestors in cases of collision.
func (dn *dataNode) Map() map[string]any {
	if dn == nil {
		return map[string]any{}
	}

	var (
		m       = map[string]any{}
		nodeIDs = []string{}
	)

	dn.lineage(func(id string, vs map[string]any) {
		if len(id) > 0 {
			nodeIDs = append(nodeIDs, id)
		}

		for k, v := range vs {
			m[k] = v
		}
	})

	if len(nodeIDs) > 0 {
		m["clues_trace"] = strings.Join(nodeIDs, ",")
	}

	if len(dn.agents) == 0 {
		return m
	}

	agentVals := map[string]map[string]any{}

	for _, agent := range dn.agents {
		agentVals[agent.id] = agent.data.Map()
	}

	m["agents"] = agentVals

	return m
}

// Slice flattens the tree of dataNode.values into a Slice where all even
// indices contain the keys, and all odd indices contain values.  Descendant
// nodes take priority over ancestors in cases of collision.
func (dn *dataNode) Slice() []any {
	m := dn.Map()
	s := make([]any, 2*len(m))
	i := 0

	for k, v := range m {
		s[i] = k
		s[i+1] = v
		i += 2
	}

	return s
}

// ---------------------------------------------------------------------------
// initialization
// ---------------------------------------------------------------------------

// init sets up persistent clients in the clues ecosystem such as otel.
// Initialization is NOT required.  It is an optional step that end
// users can take if and when they want those clients running in their
// clues instance.
//
// Multiple initializations will no-op.
func (dn *dataNode) init(
	ctx context.Context,
	name string,
	config OTELConfig,
) error {
	if dn == nil {
		return nil
	}

	// if any of these already exist, initialization was previously called.
	if dn.otel != nil {
		return nil
	}

	cli, err := newOTELClient(ctx, name, config)

	dn.otel = cli

	return Stack(err).OrNil()
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
func (dn *dataNode) addComment(
	depth int,
	msg string,
	vs ...any,
) *dataNode {
	if len(msg) == 0 {
		return dn
	}

	spawn := dn.spawnDescendant()
	spawn.id = makeNodeID()
	spawn.comment = newComment(depth+1, msg, vs...)

	return spawn
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
func (dn *dataNode) Comments() comments {
	if dn == nil {
		return comments{}
	}

	result := comments{}

	if !dn.comment.isEmpty() {
		result = append(result, dn.comment)
	}

	for dn.parent != nil {
		dn = dn.parent
		if !dn.comment.isEmpty() {
			result = append(result, dn.comment)
		}
	}

	slices.Reverse(result)

	return result
}

// ---------------------------------------------------------------------------
// agents
// ---------------------------------------------------------------------------

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
func (dn *dataNode) addAgent(name string) *dataNode {
	spawn := dn.spawnDescendant()

	if len(spawn.agents) == 0 {
		spawn.agents = map[string]*agent{}
	}

	spawn.agents[name] = &agent{
		id: name,
		// no spawn here, this needs an isolated node
		data: &dataNode{},
	}

	return spawn
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type cluesCtxKey string

const defaultCtxKey cluesCtxKey = "default_clues_ctx_key"

func ctxKey(namespace string) cluesCtxKey {
	return cluesCtxKey(namespace)
}

// nodeFromCtx pulls the datanode within a given namespace out of the context.
func nodeFromCtx(ctx context.Context) *dataNode {
	dn := ctx.Value(defaultCtxKey)

	if dn == nil {
		return &dataNode{}
	}

	return dn.(*dataNode)
}

// setNodeInCtx embeds the dataNode in the context, and returns the updated context.
func setNodeInCtx(ctx context.Context, dn *dataNode) context.Context {
	return context.WithValue(ctx, defaultCtxKey, dn)
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
func (dn *dataNode) injectTrace(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) {
	if dn == nil {
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// receiveTrace extracts the current trace details from the
// carrier and adds them to the context.  If otel is not
// initialized, no-ops.
//
// The carrier data is mutated by this call.
func (dn *dataNode) receiveTrace(
	ctx context.Context,
	carrier propagation.TextMapCarrier,
) context.Context {
	if dn == nil {
		return ctx
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// addSpan adds a new otel span.  If the otel client is nil, no-ops.
// Attrs can be added to the span with addSpanAttrs.  This span will
// continue to be used for that purpose until replaced with another
// span, which will appear in a separate context (and thus a separate,
// dataNode).
func (dn *dataNode) addSpan(
	ctx context.Context,
	name string,
) (context.Context, *dataNode) {
	if dn == nil || dn.otel == nil {
		return ctx, dn
	}

	ctx, span := dn.otel.tracer.Start(ctx, name)

	spawn := dn.spawnDescendant()
	spawn.span = span

	return ctx, spawn
}

// closeSpan closes the otel span and removes it span from the data node.
// If no span is present, no ops.
func (dn *dataNode) closeSpan(ctx context.Context) *dataNode {
	if dn == nil || dn.span == nil {
		return dn
	}

	dn.span.End()

	spawn := dn.spawnDescendant()
	spawn.span = nil

	return spawn
}

// addSpanAttributes adds the values to the current span.  If the span
// is nil (such as if otel wasn't initialized or no span has been generated),
// this call no-ops.
func (dn *dataNode) addSpanAttributes(
	values map[string]any,
) {
	if dn == nil || dn.span == nil {
		return
	}

	for k, v := range values {
		dn.span.SetAttributes(attribute.String(k, stringify.Marshal(v, false)))
	}
}

// OTELLogger gets the otel logger instance from the otel client.
// Returns nil if otel wasn't initialized.
func (dn *dataNode) OTELLogger() otellog.Logger {
	if dn == nil || dn.otel == nil {
		return nil
	}

	return dn.otel.logger
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// makeNodeID generates a random hash of 8 characters for use as a node ID.
func makeNodeID() string {
	uns := uuid.NewString()
	return uns[:4] + uns[len(uns)-4:]
}

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
func (dn *dataNode) Bytes() ([]byte, error) {
	if dn == nil {
		return []byte{}, nil
	}

	var serviceName string

	if dn.otel != nil {
		serviceName = dn.otel.serviceName
	}

	core := nodeCore{
		OTELServiceName: serviceName,
		Values:          map[string]string{},
		Comments:        dn.Comments(),
	}

	for k, v := range dn.Map() {
		core.Values[k] = stringify.Marshal(v, false)
	}

	return json.Marshal(core)
}

// FromBytes deserializes the bytes to a new dataNode.
// No clients, agents, or hooks are initialized in this process.
func FromBytes(bs []byte) (*dataNode, error) {
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

	if len(core.OTELServiceName) > 0 {
		node.otel = &otelClient{
			serviceName: core.OTELServiceName,
		}
	}

	return &node, nil
}
