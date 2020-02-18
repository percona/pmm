package inventorypb

import fmt "fmt"

//go-sumtype:decl Node

// Node is a common interface for all types of Nodes.
type Node interface {
	sealedNode() //nolint:unused
}

var nodeTypeNames = map[string]string{
	"GENERIC_NODE":    "generic-node",
	"CONTAINER_NODE":  "container-node",
	"REMOTE_NODE":     "remote-node",
	"REMOTE_RDS_NODE": "remote-rds-node",
}

// NodeTypeName returns human friendly agent type to be used in reports
func NodeTypeName(t string) string {
	res := nodeTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Node Type %s", t))
	}

	return res
}

// in order of NodeType enum

func (*GenericNode) sealedNode()   {}
func (*ContainerNode) sealedNode() {}
func (*RemoteNode) sealedNode()    {}
func (*RemoteRDSNode) sealedNode() {}
