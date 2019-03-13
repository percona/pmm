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

package models_test

import (
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestModels(t *testing.T) {
	sqlDB := tests.OpenTestDB(t)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	now := models.Now()
	origNow := models.Now

	setup := func(t *testing.T) (q *reform.Querier, teardown func(t *testing.T)) {
		models.Now = func() time.Time {
			return now
		}

		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q = tx.Querier

		require.NoError(t, q.Insert(&models.Node{
			NodeID:   "N1",
			NodeType: models.GenericNodeType,
			NodeName: "N1 name",
		}))

		require.NoError(t, q.Insert(&models.Service{
			ServiceID:   "S1",
			ServiceType: models.MySQLServiceType,
			ServiceName: "S1 name",
			NodeID:      "N1",
		}))

		require.NoError(t, q.Insert(&models.Agent{
			AgentID:      "A1",
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToStringOrNil("N1"),
		}))
		require.NoError(t, q.Insert(&models.Agent{
			AgentID:      "A2",
			AgentType:    models.MySQLdExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			RunsOnNodeID: nil,
		}))
		require.NoError(t, q.Insert(&models.Agent{
			AgentID:      "A3",
			AgentType:    models.NodeExporterType,
			PMMAgentID:   pointer.ToStringOrNil("A1"),
			RunsOnNodeID: nil,
		}))

		require.NoError(t, q.Insert(&models.AgentNode{
			AgentID: "A3",
			NodeID:  "N1",
		}))

		require.NoError(t, q.Insert(&models.AgentService{
			AgentID:   "A2",
			ServiceID: "S1",
		}))

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())

			models.Now = origNow
		}
		return
	}

	t.Run("NodesForAgent", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		nodes, err := models.NodesForAgent(q, "A3")
		require.NoError(t, err)
		expected := []*models.Node{
			{
				NodeID:    "N1",
				NodeType:  models.GenericNodeType,
				NodeName:  "N1 name",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		assert.Equal(t, expected, nodes)
	})

	t.Run("ServicesForAgent", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		services, err := models.ServicesForAgent(q, "A2")
		require.NoError(t, err)
		expected := []*models.Service{
			{
				ServiceID:   "S1",
				ServiceType: models.MySQLServiceType,
				ServiceName: "S1 name",
				NodeID:      "N1",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}
		assert.Equal(t, expected, services)
	})

	t.Run("AgentsForNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.AgentsForNode(q, "N1")
		require.NoError(t, err)
		expected := []*models.Agent{
			{
				AgentID:      "A3",
				AgentType:    models.NodeExporterType,
				PMMAgentID:   pointer.ToStringOrNil("A1"),
				RunsOnNodeID: nil,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		}
		assert.Equal(t, expected, agents)
	})

	t.Run("AgentsRunningByPMMAgent", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.AgentsRunningByPMMAgent(q, "A1")
		require.NoError(t, err)
		expected := []*models.Agent{
			{
				AgentID:      "A2",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToStringOrNil("A1"),
				RunsOnNodeID: nil,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			{
				AgentID:      "A3",
				AgentType:    models.NodeExporterType,
				PMMAgentID:   pointer.ToStringOrNil("A1"),
				RunsOnNodeID: nil,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		}
		assert.Equal(t, expected, agents)
	})

	t.Run("AgentsForService", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		agents, err := models.AgentsForService(q, "S1")
		require.NoError(t, err)
		expected := []*models.Agent{
			{
				AgentID:      "A2",
				AgentType:    models.MySQLdExporterType,
				PMMAgentID:   pointer.ToStringOrNil("A1"),
				RunsOnNodeID: nil,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		}
		assert.Equal(t, expected, agents)
	})

	t.Run("PMMAgentsForChangedNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		ids, err := models.PMMAgentsForChangedNode(q, "N1")
		require.NoError(t, err)
		assert.Equal(t, []string{"A1"}, ids)
	})

	t.Run("PMMAgentsForChangedService", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		ids, err := models.PMMAgentsForChangedService(q, "S1")
		require.NoError(t, err)
		assert.Equal(t, []string{"A1"}, ids)
	})
}
