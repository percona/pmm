// Package nodeinfo provides information about current node.
package nodeinfo

import (
	"io/ioutil"
	"runtime"
)

// NodeInfo contains node information.
type NodeInfo struct {
	Distro    string
	MachineID string
}

// Get returns node information for current node.
func Get() *NodeInfo {
	return &NodeInfo{
		Distro:    readDistro(),
		MachineID: readMachineID(),
	}
}

func readDistro() string {
	// TODO move code from pmm-managed telemetry service there

	return runtime.GOOS
}

func readMachineID() string {
	for _, name := range []string{
		"/etc/machine-id",
	} {
		b, _ := ioutil.ReadFile(name)
		if len(b) != 0 {
			return string(b)
		}
	}
	return ""
}
