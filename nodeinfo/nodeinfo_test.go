package nodeinfo

import (
	"net"
	"runtime"
	"testing"
)

func TestGet(t *testing.T) {
	info := Get()

	if info.Container {
		t.Errorf("not expected to be run inside a container")
	}

	if runtime.GOOS != info.Distro {
		t.Errorf("expected %q distro, got %q", runtime.GOOS, info.Distro)
	}

	// all our test environments have IPv4 addresses
	ip := net.ParseIP(info.PublicAddress)
	if ip == nil {
		t.Fatalf("failed to parse %q as IP address", info.PublicAddress)
	}
	if ip.To4() == nil {
		t.Fatalf("failed to parse %q as IPv4 address", info.PublicAddress)
	}
}
