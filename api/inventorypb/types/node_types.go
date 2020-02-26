package types

import "fmt"

// this list should be in sync with inventorypb/nodes.pb.go
const (
	NodeTypeGenericNode   = "GENERIC_NODE"
	NodeTypeContainerNode = "CONTAINER_NODE"
	NodeTypeRemoteNode    = "REMOTE_NODE"
	NodeTypeRemoteRDSNode = "REMOTE_RDS_NODE"
)

var nodeTypeNames = map[string]string{
	NodeTypeGenericNode:   "Generic",
	NodeTypeContainerNode: "Container",
	NodeTypeRemoteNode:    "Remote",
	NodeTypeRemoteRDSNode: "Remote RDS",
}

// NodeTypeName returns human friendly agent type to be used in reports
func NodeTypeName(t string) string {
	res := nodeTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Node Type %s", t))
	}

	return res
}
