package inventory

// Node is a common interface for all types of Nodes.
type Node interface {
	node()

	// Remote returns true if Node is remote.
	// Agents can't run on Remote Nodes.
	Remote() bool
}

func (*GenericNode) node()        {}
func (*GenericNode) Remote() bool { return false }

func (*ContainerNode) node()        {}
func (*ContainerNode) Remote() bool { return false }

func (*RemoteNode) node()        {}
func (*RemoteNode) Remote() bool { return true }

func (*RemoteAmazonRDSNode) node()        {}
func (*RemoteAmazonRDSNode) Remote() bool { return true }
