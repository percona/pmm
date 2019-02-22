package inventory

// Node is a common interface for all types of Nodes.
type Node interface {
	node()

	ID() string
	Name() string

	// Remote returns true if Node is remote.
	// Agents can't run on Remote Nodes.
	Remote() bool
}

func (*GenericNode) node()         {}
func (*ContainerNode) node()       {}
func (*RemoteNode) node()          {}
func (*RemoteAmazonRDSNode) node() {}

func (n *GenericNode) ID() string         { return n.NodeId }
func (n *ContainerNode) ID() string       { return n.NodeId }
func (n *RemoteNode) ID() string          { return n.NodeId }
func (n *RemoteAmazonRDSNode) ID() string { return n.NodeId }

func (n *GenericNode) Name() string         { return n.NodeName }
func (n *ContainerNode) Name() string       { return n.NodeName }
func (n *RemoteNode) Name() string          { return n.NodeName }
func (n *RemoteAmazonRDSNode) Name() string { return n.NodeName }

func (*GenericNode) Remote() bool         { return false }
func (*ContainerNode) Remote() bool       { return false }
func (*RemoteNode) Remote() bool          { return true }
func (*RemoteAmazonRDSNode) Remote() bool { return true }
