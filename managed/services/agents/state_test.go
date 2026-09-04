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

package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestGroupRDSExporters(t *testing.T) {
	t.Parallel()

	rdsExporter := func(id, accessKey, roleARN string) (*models.Node, *models.Agent) {
		node := &models.Node{NodeID: "node-" + id, NodeType: models.RemoteRDSNodeType}
		agent := &models.Agent{
			AgentID:   "agent-" + id,
			AgentType: models.RDSExporterType,
			NodeID:    &node.NodeID,
			AWSOptions: models.AWSOptions{
				AWSAccessKey: accessKey,
				AWSRoleARN:   roleARN,
			},
		}
		return node, agent
	}

	t.Run("two roles in one region are not merged", func(t *testing.T) {
		t.Parallel()

		nodeA, agentA := rdsExporter("a", "", "arn:aws:iam::111111111111:role/pmm")
		nodeB, agentB := rdsExporter("b", "", "arn:aws:iam::222222222222:role/pmm")

		grouped := groupRDSExporters(map[*models.Node]*models.Agent{nodeA: agentA, nodeB: agentB})

		require.Len(t, grouped, 2)
		assert.Equal(t,
			map[*models.Node]*models.Agent{nodeA: agentA},
			grouped[agentA.AWSOptions.CredentialsKey()])
		assert.Equal(t,
			map[*models.Node]*models.Agent{nodeB: agentB},
			grouped[agentB.AWSOptions.CredentialsKey()])
	})

	t.Run("same role shares one group", func(t *testing.T) {
		t.Parallel()

		const roleARN = "arn:aws:iam::111111111111:role/pmm"
		nodeA, agentA := rdsExporter("a", "", roleARN)
		nodeB, agentB := rdsExporter("b", "", roleARN)

		grouped := groupRDSExporters(map[*models.Node]*models.Agent{nodeA: agentA, nodeB: agentB})

		require.Len(t, grouped, 1)
		assert.Len(t, grouped[agentA.AWSOptions.CredentialsKey()], 2)
	})

	t.Run("static keys group by access key", func(t *testing.T) {
		t.Parallel()

		nodeA, agentA := rdsExporter("a", "AKIAONE", "")
		nodeB, agentB := rdsExporter("b", "AKIATWO", "")

		grouped := groupRDSExporters(map[*models.Node]*models.Agent{nodeA: agentA, nodeB: agentB})

		require.Len(t, grouped, 2)
		assert.Contains(t, grouped, "AKIAONE")
		assert.Contains(t, grouped, "AKIATWO")
	})

	t.Run("ambient credentials share the empty key", func(t *testing.T) {
		t.Parallel()

		nodeA, agentA := rdsExporter("a", "", "")
		nodeB, agentB := rdsExporter("b", "", "")

		grouped := groupRDSExporters(map[*models.Node]*models.Agent{nodeA: agentA, nodeB: agentB})

		require.Len(t, grouped, 1)
		assert.Len(t, grouped[""], 2)
	})
}
