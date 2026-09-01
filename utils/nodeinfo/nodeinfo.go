// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package nodeinfo provides information about current node.
package nodeinfo

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// containerMarkerFiles are files that container runtimes create inside the container:
// Docker creates /.dockerenv, Podman creates /run/.containerenv.
var containerMarkerFiles = []string{".dockerenv", "run/.containerenv"}

// containerCgroupMarkers are substrings of the /proc/1/cgroup paths under cgroup v1, where those
// paths carry the runtime name and the container ID. Under cgroup v2 the file usually holds just
// "0::/", so it can confirm a container but never rule one out.
var containerCgroupMarkers = []string{"/docker/", "/lxc/", "/kubepods", "containerd", "crio-", "libpod"}

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
		Container:     checkContainer("/"),
		Distro:        readDistro(),
		MachineID:     readMachineID(),
		PublicAddress: readPublicAddress(),
	}
}

// checkContainer reports whether the current process runs inside a container.
// The root argument is the filesystem root to probe; it is "/" outside of tests.
func checkContainer(root string) bool {
	for _, name := range containerMarkerFiles {
		_, err := os.Stat(filepath.Join(root, name))
		if err == nil {
			return true
		}
	}

	// LXC, Podman and systemd-nspawn set "container"; Kubernetes injects its service host into every Pod.
	if os.Getenv("container") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	b, _ := os.ReadFile(filepath.Join(root, "proc/1/cgroup")) //nolint:gosec
	cgroup := string(b)

	return slices.ContainsFunc(containerCgroupMarkers, func(marker string) bool {
		return strings.Contains(cgroup, marker)
	})
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
