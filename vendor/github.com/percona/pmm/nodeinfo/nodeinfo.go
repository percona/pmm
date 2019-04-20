// Package nodeinfo provides information about current node.
package nodeinfo

import (
	"io/ioutil"
	"net"
	"runtime"
)

// NodeInfo contains node information.
type NodeInfo struct {
	Distro    string
	MachineID string

	// Public/external address that can be used for scraping by Prometheus.
	PublicAddress string
}

// Get returns node information for current node.
func Get() *NodeInfo {
	return &NodeInfo{
		Distro:        readDistro(),
		MachineID:     readMachineID(),
		PublicAddress: readPublicAddress(),
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

// TODO remove that completely once we have "zero port" feature
func readPublicAddress() string {
	var res string

	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		// skip down and loopback interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			s := addr.String()
			if ipnet, _ := addr.(*net.IPNet); ipnet != nil {
				s = ipnet.IP.String()
			}
			if ip := net.ParseIP(s); ip != nil {
				// prefer (return first) IPv4 address, but fallback to any IPv6
				res = ip.String()
				ip = ip.To4()
				if ip != nil {
					return ip.String()
				}
			}
		}
	}

	return res
}
