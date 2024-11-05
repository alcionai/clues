package node

// ---------------------------------------------------------------------------
// agents
// ---------------------------------------------------------------------------

type Agent struct {
	// the name of the agent
	ID string

	// Data is used here instead of a basic value map so that
	// we can extend the usage of agents in the future by allowing
	// the full set of node behavior.  We'll need a builder for that,
	// but we'll get there eventually.
	Data *Node
}

// AddAgent adds a new named agent to the node.
func (dn *Node) AddAgent(name string) *Node {
	spawn := dn.SpawnDescendant()

	if len(spawn.Agents) == 0 {
		spawn.Agents = map[string]*Agent{}
	}

	spawn.Agents[name] = &Agent{
		ID: name,
		// no spawn here, this needs an isolated node
		Data: &Node{},
	}

	return spawn
}
