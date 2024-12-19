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
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRoster(t *testing.T) {
	setup := func(t *testing.T) (*roster, func(t *testing.T)) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, sqlDB.Close())
		}

		r := newRoster(db)

		return r, teardown
	}

	t.Run("Add", func(t *testing.T) {
		r, teardown := setup(t)
		defer teardown(t)

		exporters := make(map[*models.Node]*models.Agent, 1)
		node := &models.Node{
			NodeID:   "node1",
			NodeType: models.GenericNodeType,
		}
		awsAccessKey := "aws"
		exporters[node] = &models.Agent{
			AgentID:   "agent1",
			AgentType: models.RDSExporterType,
			AWSOptions: models.AWSOptions{
				AWSAccessKey: awsAccessKey,
			},
		}

		const expected = "pmm-server:rds/aws"
		groupID := r.add("pmm-server", rdsPrefix+awsAccessKey, exporters)
		assert.Equal(t, expected, groupID)

		PMMAgentID, agentIDs, err := r.get(groupID)
		require.NoError(t, err)
		assert.Equal(t, "pmm-server", PMMAgentID)
		assert.Equal(t, []string{"agent1"}, agentIDs)
	})

	t.Run("Get", func(t *testing.T) {
		r, teardown := setup(t)
		defer teardown(t)

		const groupID = "pmm-server:rds/AWSAccessKey"

		PMMAgentID, agentIDs, err := r.get(groupID)
		require.NoError(t, err)
		assert.Equal(t, "pmm-server", PMMAgentID)
		assert.Equal(t, []string{}, agentIDs)
	})

	t.Run("Clear", func(t *testing.T) {
		r, teardown := setup(t)
		defer teardown(t)

		exporters := make(map[*models.Node]*models.Agent, 1)
		node := &models.Node{
			NodeID:   "node1",
			NodeType: models.GenericNodeType,
		}
		awsAccessKey := "aws"
		exporters[node] = &models.Agent{
			AgentID:   "agent1",
			AgentType: models.RDSExporterType,
			AWSOptions: models.AWSOptions{
				AWSAccessKey: awsAccessKey,
			},
		}

		const expectedGroupID = "pmm-server:rds/aws"
		PMMAgentID := "pmm-server"
		groupID := r.add(PMMAgentID, rdsPrefix+awsAccessKey, exporters)
		assert.Equal(t, expectedGroupID, groupID)

		r.clear(PMMAgentID)
		PMMAgentID, agentIDs, err := r.get(groupID)

		require.NoError(t, err)
		assert.Equal(t, "pmm-server", PMMAgentID)
		assert.Equal(t, []string{}, agentIDs)
	})
}
