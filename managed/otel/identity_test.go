package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckPhase1ResourceIdentity_OK(t *testing.T) {
	t.Parallel()
	c := CheckPhase1ResourceIdentity(map[string]string{
		AttrServiceName:      "checkout",
		AttrPMMNodeID:        "node-1",
		AttrPMMAgentID:       "agent-1",
		AttrNetPeerName:      "10.0.0.5:3306",
		AttrDBSystem:         "mysql",
		AttrPMMComponentRole: "app",
	})
	assert.True(t, c.OK)
	assert.Empty(t, c.Missing)
}

func TestCheckPhase1ResourceIdentity_Missing(t *testing.T) {
	t.Parallel()
	c := CheckPhase1ResourceIdentity(map[string]string{
		AttrServiceName: "checkout",
	})
	assert.False(t, c.OK)
	assert.NotEmpty(t, c.Missing)
}

func TestCheckPhase1ResourceIdentity_PeerIPPort(t *testing.T) {
	t.Parallel()
	c := CheckPhase1ResourceIdentity(map[string]string{
		AttrServiceName:      "checkout",
		AttrPMMNodeID:        "node-1",
		AttrPMMAgentID:       "agent-1",
		AttrNetPeerIP:        "10.0.0.5",
		AttrNetPeerPort:      "3306",
		AttrDBSystem:         "mysql",
		AttrPMMComponentRole: "database",
	})
	assert.True(t, c.OK)
}
