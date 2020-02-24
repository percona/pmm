package inventorypb

import "fmt"

const (
	AgentTypeGenericNode   = "GENERIC_NODE"
	AgentTypeContainerNode = "CONTAINER_NODE"
	AgentTypeRemoteNode    = "REMOTE_NODE"
	AgentTypeRemoteRdsNode = "REMOTE_RDS_NODE"
)

var nodeTypeNames = map[string]string{
	AgentTypeGenericNode:   "generic-node",
	AgentTypeContainerNode: "container-node",
	AgentTypeRemoteNode:    "remote-node",
	AgentTypeRemoteRdsNode: "remote-rds-node",
}

// NodeTypeName returns human friendly agent type to be used in reports
func NodeTypeName(t string) string {
	res := nodeTypeNames[t]
	if res == "" {
		panic(fmt.Sprintf("no nice string for Node Type %s", t))
	}

	return res
}
