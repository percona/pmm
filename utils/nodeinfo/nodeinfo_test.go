package nodeinfo

import (
	"net"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	info := Get()
	require.False(t, info.Container, "not expected to be run inside a container")
	assert.Equal(t, runtime.GOOS, info.Distro)

	// all our test environments have IPv4 addresses
	ip := net.ParseIP(info.PublicAddress)
	require.NotNil(t, ip)
	assert.NotNil(t, ip.To4())

	assert.False(t, strings.HasSuffix(info.MachineID, "\n"), "%q", info.MachineID)
}
