package node

import (
	"context"
	"encoding/json"
	"maps"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/alcionai/clues/internal/stringify"
)

// ---------------------------------------------------------------------------
// data nodes
// ---------------------------------------------------------------------------

type Noder interface {
	Node() *Node
}

// Node contains the data tracked by both clues in contexts and in errors.
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
type Node struct {
	Parent *Node

	// OTEL contains the client instance for the in memory OTEL runtime.  It is only
	// present if the end user calls the clues initialization step.
	OTEL *OTELClient

	// Span is the current otel Span.
	// Spans are kept separately from the otelClient because we want the client to
	// maintain a consistent reference to otel initialization, while the Span can
	// get replaced at arbitrary points.
	Span trace.Span

	// ids are optional and are used primarily as tracing markers.
	// if empty, the trace for that node will get skipped when building the
	// full trace along the node's ancestry path in the tree.
	ID string

	// Values are they arbitrary key:value pairs that appear in clues when callers
	// use the Add(ctx, k, v) or err.With(k, v) adders.  Each key-value pair added
	// to the node is used to produce the final set of Values() in the node,
	// with lower nodes in the tree taking priority over higher nodes for any
	// collision resolution.
	Values map[string]any

	// each node can hold a single comment.  The history of comments produced
	// by the ancestry path through the tree will get concatenated from oldest
	// ancestor to the current node to produce the Comment history.
	Comment Comment

	// Agents act as proxy node that can relay specific, intentional data
	// additions.  They're namespaced so that additions to the Agents don't accidentally
	// clobber other values in the node. This also allows Agents to protect
	// variations of data from each other, in case users need to compare differences
	// on the same keys.  That's not the goal for Agents, exactly, but it is capable.
	Agents map[string]*Agent
}

// SpawnDescendant generates a new node that is a descendant of the current
// node.  A descendant maintains a pointer to its parent, and carries any genetic
// necessities (ie, copies of fields) that must be present for continued functionality.
func (dn *Node) SpawnDescendant() *Node {
	agents := maps.Clone(dn.Agents)

	if agents == nil && dn.Agents != nil {
		agents = map[string]*Agent{}
	}

	return &Node{
		Parent: dn,
		OTEL:   dn.OTEL,
		Span:   dn.Span,
		Agents: agents,
	}
}

// ---------------------------------------------------------------------------
// setters
// ---------------------------------------------------------------------------

type addAttrConfig struct {
	doNotAddToSpan       bool
	addToOTELHTTPLabeler bool
	otelHTTPLabeler      otelHTTPLabeler
}

type addAttrOptions func(*addAttrConfig)

func makeAddAttrConfig(opts ...addAttrOptions) addAttrConfig {
	cfg := addAttrConfig{}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// DoNotAddToSpan prevents the values from being added to the current span.
func DoNotAddToSpan() addAttrOptions {
	return func(cfg *addAttrConfig) {
		cfg.doNotAddToSpan = true
	}
}

// AddToOTELHTTPLabeler adds the values to the otelHTTPLabeler, which will
// hold the values in reserve until the next span creation.  Naturally, the
// values will not be added to the current span.
func AddToOTELHTTPLabeler(labeler otelHTTPLabeler) addAttrOptions {
	return func(cfg *addAttrConfig) {
		cfg.addToOTELHTTPLabeler = true
		cfg.otelHTTPLabeler = labeler
	}
}

// AddValues adds all entries in the map to the node's values.
// automatically propagates values onto the current span.
func (dn *Node) AddValues(
	ctx context.Context,
	m map[string]any,
	opts ...addAttrOptions,
) *Node {
	if m == nil {
		m = map[string]any{}
	}

	cfg := makeAddAttrConfig(opts...)

	spawn := dn.SpawnDescendant()
	spawn.SetValues(m)

	if cfg.addToOTELHTTPLabeler {
		spawn.AddToOTELHTTPLabeler(cfg.otelHTTPLabeler, m)
	}

	if !cfg.doNotAddToSpan && !cfg.addToOTELHTTPLabeler {
		spawn.AddSpanAttributes(m)
	}

	return spawn
}

// SetValues is generally a helper called by addValues.  In
// certain corner cases (like agents) it may get called directly.
func (dn *Node) SetValues(m map[string]any) {
	if len(m) == 0 {
		return
	}

	if len(dn.Values) == 0 {
		dn.Values = map[string]any{}
	}

	maps.Copy(dn.Values, m)
}

// ---------------------------------------------------------------------------
// getters
// ---------------------------------------------------------------------------

// RunLineage runs the fn on every valueNode in the ancestry tree,
// starting at the root and ending at the node.
func (dn *Node) RunLineage(fn func(id string, vs map[string]any)) {
	if dn == nil {
		return
	}

	if dn.Parent != nil {
		dn.Parent.RunLineage(fn)
	}

	fn(dn.ID, dn.Values)
}

// Map flattens the tree of node.values into a map.  Descendant nodes
// take priority over ancestors in cases of collision.
func (dn *Node) Map() map[string]any {
	var (
		m       = map[string]any{}
		nodeIDs = []string{}
	)

	dn.RunLineage(func(id string, vs map[string]any) {
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

	if len(dn.Agents) == 0 {
		return m
	}

	agentVals := map[string]map[string]any{}

	for _, agent := range dn.Agents {
		agentVals[agent.ID] = agent.Data.Map()
	}

	m["agents"] = agentVals

	return m
}

// Slice flattens the tree of node.values into a Slice where all even
// indices contain the keys, and all odd indices contain values.  Descendant
// nodes take priority over ancestors in cases of collision.
func (dn *Node) Slice() []any {
	m := dn.Map()
	s := make([]any, 0, 2*len(m))

	for k, v := range m {
		s = append(s, k, v)
	}

	return s
}

// ---------------------------------------------------------------------------
// initialization
// ---------------------------------------------------------------------------

// Init sets up persistent clients in the clues ecosystem such as otel.
// Initialization is NOT required.  It is an optional step that end
// users can take if and when they want those clients running in their
// clues instance.
//
// Multiple initializations will no-op.
func (dn *Node) InitOTEL(
	ctx context.Context,
	name string,
	config OTELConfig,
) error {
	if dn == nil {
		return nil
	}

	// if any of these already exist, initialization was previously called.
	if dn.OTEL != nil {
		return nil
	}

	cli, err := NewOTELClient(ctx, name, config)

	dn.OTEL = cli

	if err != nil {
		return err
	}

	return nil
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type CluesCtxKey string

const defaultCtxKey CluesCtxKey = "default_clues_ctx_key"

func CtxKey(namespace string) CluesCtxKey {
	return CluesCtxKey(namespace)
}

// FromCtx pulls the node within a given namespace out of the context.
func FromCtx(ctx context.Context) *Node {
	if ctx == nil {
		return &Node{}
	}

	dn := ctx.Value(defaultCtxKey)

	if dn == nil {
		return &Node{}
	}

	return dn.(*Node)
}

// EmbedInCtx adds the node in the context, and returns the updated context.
func EmbedInCtx(ctx context.Context, dn *Node) context.Context {
	return context.WithValue(ctx, defaultCtxKey, dn)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// randomNodeID generates a random hash of 8 characters for use as a node ID.
func randomNodeID() string {
	uns := uuid.NewString()
	return uns[:4] + uns[len(uns)-4:]
}

// ---------------------------------------------------------------------------
// serialization
// ---------------------------------------------------------------------------

// nodeCore contains the serializable set of data in a Node.
type nodeCore struct {
	OTELServiceName string `json:"otelServiceName"`
	// TODO: investigate if map[string]string is really the best structure here.
	// maybe we can get away with a map[string]any, or a []byte slice?
	Values   map[string]string `json:"values"`
	Comments []Comment         `json:"comments"`
}

// Bytes serializes the Node to a slice of bytes.
// Only attributes and comments are maintained.  All
// values are stringified in the process.
//
// Node hierarchy, clients (such as otel), agents, and
// hooks (such as labelCounter) are all sliced from the
// result.
func (dn *Node) Bytes() ([]byte, error) {
	if dn == nil {
		return []byte{}, nil
	}

	var serviceName string

	if dn.OTEL != nil {
		serviceName = dn.OTEL.ServiceName
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

// FromBytes deserializes the bytes to a new Node.
// No clients, agents, or hooks are initialized in this process.
func FromBytes(bs []byte) (*Node, error) {
	core := nodeCore{}

	err := json.Unmarshal(bs, &core)
	if err != nil {
		return nil, err
	}

	node := Node{
		// FIXME: do something with the serialized comments.
		// I'm punting on this for now because I want to figure
		// out the best middle ground between avoiding a slice of
		// comments in each node for serialization sake (they
		// are supposed to be one-comment-per-node to use the tree
		// for ordering instead of the parameter), and keeping
		// the full comment history available.  Probably just
		// need to introduce a delimiter.
	}

	if len(core.Values) > 0 {
		node.Values = map[string]any{}
	}

	for k, v := range core.Values {
		node.Values[k] = v
	}

	if len(core.OTELServiceName) > 0 {
		node.OTEL = &OTELClient{
			ServiceName: core.OTELServiceName,
		}
	}

	return &node, nil
}
