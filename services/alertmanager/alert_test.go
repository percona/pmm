// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package alertmanager

import (
	"testing"

	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-managed/models"
)

func TestMakeAlert(t *testing.T) {
	agent := &models.Agent{
		AgentID: "/agent_id/123",
	}
	node := &models.Node{
		NodeID:   "/node_id/456",
		NodeName: "nodename",
	}
	name, alert, err := makeAlertPMMAgentNotConnected(agent, node)
	require.NoError(t, err)

	assert.Equal(t, "pmm_agent_not_connected", name)

	expected := &ammodels.PostableAlert{
		Alert: ammodels.Alert{
			Labels: ammodels.LabelSet{
				"agent_id":  "/agent_id/123",
				"alertname": "pmm_agent_not_connected",
				"node_id":   "/node_id/456",
				"node_name": "nodename",
				"severity":  "warning",
				"stt_check": "1",
			},
		},
		Annotations: ammodels.LabelSet{
			"summary":     "pmm-agent is not connected to PMM Server",
			"description": "Node name: nodename",
		},
	}
	assert.Equal(t, expected, alert)
}
