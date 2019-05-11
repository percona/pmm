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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestNodeHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (q *reform.Querier, teardown func(t *testing.T)) {
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q = tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:   "N1",
				NodeType: models.GenericNodeType,
				NodeName: "Node with Service",
			},
			&models.Node{
				NodeID:   "N2",
				NodeType: models.GenericNodeType,
				NodeName: "Node with pmm-agent",
			},
			&models.Node{
				NodeID:   "N3",
				NodeType: models.GenericNodeType,
				NodeName: "Node with node_exporter",
			},
			&models.Node{
				NodeID:   "N4",
				NodeType: models.GenericNodeType,
				NodeName: "Empty Node",
			},

			&models.Service{
				ServiceID:   "S1",
				ServiceType: models.MySQLServiceType,
				ServiceName: "Service on N1",
				NodeID:      "N1",
			},

			&models.Agent{
				AgentID:      "A1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("N2"),
			},
			&models.Agent{
				AgentID:    "A2",
				AgentType:  models.NodeExporterType,
				PMMAgentID: pointer.ToString("A1"),
			},

			&models.AgentNode{
				AgentID: "A2",
				NodeID:  "N3",
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		teardown = func(t *testing.T) {
			require.NoError(t, tx.Rollback())
		}
		return
	}

	t.Run("FindNodesForAgentID", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		nodes, err := models.FindNodesForAgentID(q, "A2")
		require.NoError(t, err)
		expected := []*models.Node{{
			NodeID:    "N3",
			NodeType:  models.GenericNodeType,
			NodeName:  "Node with node_exporter",
			CreatedAt: now,
			UpdatedAt: now,
		}}
		assert.Equal(t, expected, nodes)
	})

	t.Run("RemoveNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		err := models.RemoveNode(q, "", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node ID.`), err)

		err = models.RemoveNode(q, "N0", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "N0" not found.`), err)
		err = models.RemoveNode(q, "N1", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Node with ID "N1" has services.`), err)
		err = models.RemoveNode(q, "N2", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Node with ID "N2" has pmm-agent.`), err)
		err = models.RemoveNode(q, "N3", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Node with ID "N3" has agents.`), err)

		err = models.RemoveNode(q, "N4", models.RemoveRestrict)
		assert.NoError(t, err)

		// in reverse order to cover all branches
		err = models.RemoveNode(q, "N3", models.RemoveCascade)
		assert.NoError(t, err)
		err = models.RemoveNode(q, "N2", models.RemoveCascade)
		assert.NoError(t, err)
		err = models.RemoveNode(q, "N1", models.RemoveCascade)
		assert.NoError(t, err)

		nodes, err := models.FindAllNodes(q)
		assert.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, models.PMMServerNodeID, nodes[0].NodeID)
	})
}
