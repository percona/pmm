package inventory

// Node is a common interface for all types of Nodes.
type Node interface {
	node()
}

func (*BareMetalNode) node()      {}
func (*VirtualMachineNode) node() {}
func (*ContainerNode) node()      {}
func (*RemoteNode) node()         {}
func (*AWSRDSNode) node()         {}
