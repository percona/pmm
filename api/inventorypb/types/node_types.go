package types

import "fmt"

const (
	NodeTypeGenericNode   = "GENERIC_NODE"
	NodeTypeContainerNode = "CONTAINER_NODE"
	NodeTypeRemoteNode    = "REMOTE_NODE"
	NodeTypeRemoteRDSNode = "REMOTE_RDS_NODE"
)

var nodeTypeNames = map[string]string{
	NodeTypeGenericNode:   "generic-node",
	NodeTypeContainerNode: "container-node",
	NodeTypeRemoteNode:    "remote-node",
	NodeTypeRemoteRDSNode: "remote-rds-node",
}

// NodeTypeName returns human friendly agent type to be used in reports
func NodeTypeName(t string) string {
	res := nodeTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Node Type %s", t))
	}

	return res
}
