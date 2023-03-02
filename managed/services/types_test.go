package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestTarget_Copy(t1 *testing.T) {
	target := Target{
		AgentID:     "agent_id",
		ServiceID:   "service_id",
		ServiceName: "service_name",
		NodeName:    "node_name",
		Labels:      map[string]string{"label": "value"},
		DSN:         "dsn",
		Files:       map[string]string{"file": "test"},
		TDP: &models.DelimiterPair{
			Left:  "[",
			Right: "]",
		},
		TLSSkipVerify: true,
	}

	newTarget := target.Copy()
	require.Equal(t1, target, newTarget)

	// Change all values in newTarget
	newTarget.AgentID = "new_agent_id"
	newTarget.ServiceID = "new_service_id"
	newTarget.ServiceName = "new_service_name"
	newTarget.NodeName = "new_node_name"
	newTarget.Labels["new_label"] = "new_value"
	newTarget.DSN = "new_dsn"
	newTarget.Files["new_file"] = "new_test"
	newTarget.TDP.Left = "{"
	newTarget.TDP.Right = "}"
	newTarget.TLSSkipVerify = false

	// Check that original target was unchanged
	assert.Equal(t1, "agent_id", target.AgentID)
	assert.Equal(t1, "service_id", target.ServiceID)
	assert.Equal(t1, "service_name", target.ServiceName)
	assert.Equal(t1, "node_name", target.NodeName)
	assert.Equal(t1, map[string]string{"label": "value"}, target.Labels)
	assert.Equal(t1, "dsn", target.DSN)
	assert.Equal(t1, map[string]string{"file": "test"}, target.Files)
	assert.Equal(t1, "[", target.TDP.Left)
	assert.Equal(t1, "]", target.TDP.Right)
	assert.Equal(t1, true, target.TLSSkipVerify)
}
