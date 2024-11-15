package clues

import (
	"context"
	"maps"
	"strings"
	"sync/atomic"

	otellog "go.opentelemetry.io/otel/log"
)

// ---------------------------------------------------------------------------
// registry
// ---------------------------------------------------------------------------

// registry contains a reference to all nodes tracked in this context. Only one
// registry should get initialized in a given context (clues controls this
// production), and it acts as a singleton containing all one-off initializations.
type registry struct {
	// The current ID, each node addition should inc currID by 1.
	// yes, I know this and a map fakes an array.  But I'm trying it out anyway.
	currID *atomic.Int32

	// nodes register each node by integer ID.
	nodes map[int32]dataNode

	// otel contains the client instance for the in memory otel runtime.  It is only
	// present if the end user calls the clues initialization step.
	otel *otelClient
}

func newRegistry() *registry {
	one := atomic.Int32{}
	one.Swap(1)

	return &registry{
		currID: &one,
		nodes:  map[int32]dataNode{},
	}
}

// spawnDescendant generates a new dataNode that is a descendant of the current
// node.  A descendant maintains a pointer to its parent, and carries any genetic
// necessities (ie, copies of fields) that must be present for continued functionality.
func (reg *registry) spawnDescendant(parentID int32) dataNode {
	if len(reg.nodes) < (int(parentID) + 1) {
		parentID = 0
	}

	var (
		id     = reg.currID.Add(1)
		parent = reg.nodes[parentID]
		agents = maps.Clone(parent.agents)
	)

	if agents == nil && parent.agents != nil {
		agents = map[string]*agent{}
	}

	n := dataNode{
		id:     id,
		parent: parentID,
		depth:  parent.depth + 1,
		span:   parent.span,
		agents: agents,
	}

	reg.nodes[id] = n

	return n
}

type noder interface {
	node(node dataNode)
}

type fnNoder struct {
	fn func(dn dataNode)
}

func (n fnNoder) node(dn dataNode) {
	n.fn(dn)
}

// iterNodesRootToLeaf runs the noder against each node in the ancestry
// starting from the parent, and ending at the leaf.
func (reg registry) iterNodesRootToLeaf(
	node int32,
	consumer noder,
) {
	var (
		curr  = reg.nodes[node]
		nodes = make([]dataNode, 0, curr.depth+1)
	)

	for i := curr.depth; i >= 0; i-- {
		nodes[i] = curr
		curr = reg.nodes[curr.parent]
	}

	for _, n := range nodes {
		consumer.node(n)
	}
}

// runs the noder against each node in the ancestry
// starting from the leaf, and ending at the root.
func (reg registry) iterNodesLeafToRoot(
	node int32,
	consumer noder,
) {
	curr := reg.nodes[node]

	for i := curr.depth; i >= 0; i-- {
		consumer.node(curr)
		curr = reg.nodes[curr.parent]
	}
}

// ---------------------------------------------------------------------------
// ctx handling
// ---------------------------------------------------------------------------

type registryCtxKey string

const defaultRegistryKey registryCtxKey = "default_node_ctx_key"

func regCtxKey(namespace string) registryCtxKey {
	return registryCtxKey(namespace)
}

// registryFromCtx retrieves the registry from the context.
func registryFromCtx(ctx context.Context) *registry {
	reg := ctx.Value(defaultRegistryKey)

	if reg == nil {
		return nil
	}

	return reg.(*registry)
}

// setRegistryInCtx adds the registry to the context and returns the updated context.
func setRegistryInCtx(ctx context.Context, reg *registry) context.Context {
	return context.WithValue(ctx, defaultRegistryKey, reg)
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
func (reg *registry) init(
	ctx context.Context,
	name string,
	config OTELConfig,
) error {
	if reg == nil {
		return nil
	}

	// if any of these already exist, initialization was previously called.
	if reg.otel != nil {
		return nil
	}

	cli, err := newOTELClient(ctx, name, config)

	reg.otel = cli

	return Stack(err).OrNil()
}

// ---------------------------------------------------------------------------
// data accessors
// ---------------------------------------------------------------------------

// Map flattens the tree of dataNode.values into a map.  Descendant nodes
// take priority over ancestors in cases of collision.
func (rn regNode) Map(
	nodeID int32,
) map[string]any {
	if !rn.ok || len(rn.reg.nodes) == 0 {
		return map[string]any{}
	}

	var (
		m         = map[string]any{}
		nodeNames = []string{}
	)

	fn := fnNoder{
		fn: func(dn dataNode) {
			if len(dn.name) > 0 {
				nodeNames = append(nodeNames, dn.name)
			}

			for k, v := range dn.values {
				m[k] = v
			}
		},
	}

	rn.reg.iterNodesRootToLeaf(nodeID, fn)

	if len(nodeNames) > 0 {
		m["clues_trace"] = strings.Join(nodeNames, ",")
	}

	agents := rn.reg.nodes[nodeID].agents

	if len(agents) == 0 {
		return m
	}

	agentVals := map[string]map[string]any{}

	for _, agent := range agents {
		agentVals[agent.id] = agent.data.values
	}

	m["agents"] = agentVals

	return m
}

// Slice flattens the tree of dataNode.values into a Slice where all even
// indices contain the keys, and all odd indices contain values.  Descendant
// nodes take priority over ancestors in cases of collision.
func (rn regNode) Slice(
	nodeID int32,
) []any {
	if !rn.ok {
		return []any{}
	}

	m := rn.Map(nodeID)
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
// otel
// ---------------------------------------------------------------------------

// OTELLogger gets the otel logger instance from the otel client.
// Returns nil if otel wasn't initialized.
func (r registry) OTELLogger() otellog.Logger {
	// TODO: can I pull this out of the ctx?
	if r.otel == nil {
		return nil
	}

	return r.otel.logger
}
