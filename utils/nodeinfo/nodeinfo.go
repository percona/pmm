// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package nodeinfo provides information about current node.
package nodeinfo

import (
	"net"
	"os"
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
	_, err := os.Stat("/.dockerenv")
	if err != nil {
		return true
	}
	_, err = os.Stat("/run/.containerenv") // Podman-specific
	if err != nil {
		return true
	}
	// https://stackoverflow.com/a/20012536
	b, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return strings.Contains(string(b), "/docker/") || strings.Contains(string(b), "/lxc/") || strings.Contains(string(b), "/podman/")
	}
	return false
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
		b, _ := os.ReadFile(name) //nolint:gosec
		if len(b) != 0 {
			return strings.TrimSpace(string(b))
		}
	}
	return ""
}

// TODO remove that completely once we have "zero port" feature.
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
