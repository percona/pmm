package inventory

// Node is a common interface for all types of Nodes.
type Node interface {
	node()

	// Remote returns true if Node is remote.
	// Agents can't be run on remote Nodes.
	Remote() bool
}

func (*GenericNode) node()        {}
func (*GenericNode) Remote() bool { return false }

func (*RemoteNode) node()        {}
func (*RemoteNode) Remote() bool { return true }

func (*AmazonRDSRemoteNode) node()        {}
func (*AmazonRDSRemoteNode) Remote() bool { return true }
