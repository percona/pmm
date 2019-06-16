// Package nodeinfo provides information about current node.
package nodeinfo

import (
	"io/ioutil"
	"net"
	"runtime"
	"strings"
)

// NodeInfo contains node information.
type NodeInfo struct {
	Container bool
	Distro    string
	MachineID string

	// Public/external address that can be used for scraping by Prometheus.
	PublicAddress string
}

// Get returns node information for current node.
func Get() *NodeInfo {
	return &NodeInfo{
		Container:     checkContainer(),
		Distro:        readDistro(),
		MachineID:     readMachineID(),
		PublicAddress: readPublicAddress(),
	}
}

func checkContainer() bool {
	// https://stackoverflow.com/a/20012536
	b, _ := ioutil.ReadFile("/proc/1/cgroup") //nolint:gosec
	return strings.Contains(string(b), "/docker/") || strings.Contains(string(b), "/lxc/")
}

func readDistro() string {
	// TODO move code from pmm-managed telemetry service there

	return runtime.GOOS
}

func readMachineID() string {
	for _, name := range []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	} {
		b, _ := ioutil.ReadFile(name) //nolint:gosec
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
