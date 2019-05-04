package inventorypb

//go-sumtype:decl Node

// Node is a common interface for all types of Nodes.
type Node interface {
	sealedNode() //nolint:unused

	// TODO Remove it.
	// Deprecated: use NodeId field instead.
	ID() string
}

func (*GenericNode) sealedNode()         {}
func (*ContainerNode) sealedNode()       {}
func (*RemoteNode) sealedNode()          {}
func (*RemoteAmazonRDSNode) sealedNode() {}

func (m *GenericNode) ID() string         { return m.NodeId }
func (m *ContainerNode) ID() string       { return m.NodeId }
func (m *RemoteNode) ID() string          { return m.NodeId }
func (m *RemoteAmazonRDSNode) ID() string { return m.NodeId }
