package inventory

// Node is a common interface for all types of Nodes.
type Node interface {
	node()

	// Remote returns true if Node is remote.
	// Agents can't run on Remote Nodes.
	Remote() bool
}

func (*GenericNode) node()         {}
func (*ContainerNode) node()       {}
func (*RemoteNode) node()          {}
func (*RemoteAmazonRDSNode) node() {}

func (*GenericNode) Remote() bool         { return false }
func (*ContainerNode) Remote() bool       { return false }
func (*RemoteNode) Remote() bool          { return true }
func (*RemoteAmazonRDSNode) Remote() bool { return true }
