package flags

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
)

func mustNew(t *testing.T, grammar any) *kong.Kong {
	k, err := kong.New(grammar, kong.Vars{
		"socketPath": "",
	})
	assert.NoError(t, err)

	return k
}

func TestAddressConfiguration(t *testing.T) {
	t.Run("Socket is set by default", func(t *testing.T) {
		var opts GlobalFlags
		k := mustNew(t, &opts)

		k.Parse([]string{})
		assert.Equal(t, opts.PMMAgentSocket, SocketPath)
		assert.Equal(t, opts.PMMAgentListenPort, uint32(0))
	})

	t.Run("Port is configured", func(t *testing.T) {
		var opts GlobalFlags
		k := mustNew(t, &opts)

		k.Parse([]string{"--pmm-agent-listen-port", "7777"})
		assert.Equal(t, opts.PMMAgentSocket, "")
		assert.Equal(t, opts.PMMAgentListenPort, uint32(7777))
	})
}
