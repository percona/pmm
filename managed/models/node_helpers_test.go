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

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestNodeHelpers(t *testing.T) {
	now, origNowF := models.Now(), models.Now
	models.Now = func() time.Time {
		return now
	}
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	defer func() {
		models.Now = origNowF
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:    "MySQLNode",
				NodeType:  models.ContainerNodeType,
				NodeName:  "Node for MySQL Service",
				MachineID: pointer.ToString("/machine_id/MySQLNode"),
			},
			&models.Service{
				ServiceID:   "MySQL",
				ServiceType: models.MySQLServiceType,
				ServiceName: "MySQL on MySQLNode",
				NodeID:      "MySQLNode",
				Socket:      pointer.ToStringOrNil("/var/run/mysqld/mysqld.sock"),
			},

			&models.Node{
				NodeID:    "GenericNode",
				NodeType:  models.GenericNodeType,
				NodeName:  "Node for Agents",
				MachineID: pointer.ToString("/machine_id/GenericNode"),
			},
			&models.Agent{
				AgentID:      "pmm-agent",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("GenericNode"),
			},

			&models.Agent{
				AgentID:    "node_exporter",
				AgentType:  models.NodeExporterType,
				PMMAgentID: pointer.ToString("pmm-agent"),
				NodeID:     pointer.ToString("GenericNode"),
			},

			&models.Agent{
				AgentID:    "mysqld_exporter",
				AgentType:  models.MySQLdExporterType,
				PMMAgentID: pointer.ToString("pmm-agent"),
				ServiceID:  pointer.ToString("MySQL"),
			},

			&models.Node{
				NodeID:   "NodeWithPMMAgent",
				NodeType: models.GenericNodeType,
				NodeName: "Node With pmm-agent",
			},
			&models.Agent{
				AgentID:      "pmm-agent1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: pointer.ToString("NodeWithPMMAgent"),
			},

			&models.Node{
				NodeID:   "EmptyNode",
				NodeType: models.GenericNodeType,
				NodeName: "Empty Node",
			},
		} {
			require.NoError(t, q.Insert(str), "failed to INSERT %+v", str)
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return q, teardown
	}

	t.Run("CreateNode", func(t *testing.T) {
		t.Run("DuplicateMachineID", func(t *testing.T) {
			// https://jira.percona.com/browse/PMM-4196

			q, teardown := setup(t)
			defer teardown(t)

			machineID := "/machine_id/GenericNode"
			_, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
				NodeName:  t.Name(),
				MachineID: pointer.ToString(machineID + "\n"),
			})
			assert.NoError(t, err)

			structs, err := q.SelectAllFrom(models.NodeTable, "WHERE machine_id = $1 ORDER BY node_id", machineID)
			require.NoError(t, err)
			require.Len(t, structs, 2)
			expected := &models.Node{
				NodeID:    "GenericNode",
				NodeType:  models.GenericNodeType,
				NodeName:  "Node for Agents",
				MachineID: &machineID, // \n trimmed
				CreatedAt: now,
				UpdatedAt: now,
			}
			assert.Equal(t, expected, structs[0])
			expected = &models.Node{
				NodeID:    structs[1].(*models.Node).NodeID,
				NodeType:  models.GenericNodeType,
				NodeName:  t.Name(),
				MachineID: &machineID,
				CreatedAt: now,
				UpdatedAt: now,
			}
			assert.Equal(t, expected, structs[1])
		})
	})

	t.Run("FindNodes", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		nodes, err := models.FindNodes(q, models.NodeFilters{})
		require.NoError(t, err)

		expected := []*models.Node{{
			NodeID:    "EmptyNode",
			NodeType:  models.GenericNodeType,
			NodeName:  "Empty Node",
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:    "GenericNode",
			NodeType:  models.GenericNodeType,
			NodeName:  "Node for Agents",
			MachineID: pointer.ToString("/machine_id/GenericNode"),
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:    "MySQLNode",
			NodeType:  models.ContainerNodeType,
			NodeName:  "Node for MySQL Service",
			MachineID: pointer.ToString("/machine_id/MySQLNode"),
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:    "NodeWithPMMAgent",
			NodeType:  models.GenericNodeType,
			NodeName:  "Node With pmm-agent",
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:    models.PMMServerNodeID,
			NodeType:  models.GenericNodeType,
			NodeName:  "pmm-server",
			Address:   "127.0.0.1",
			CreatedAt: now,
			UpdatedAt: now,
		}}
		require.Equal(t, expected, nodes)
	})

	t.Run("FindNodesByType", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		nodes, err := models.FindNodes(q, models.NodeFilters{NodeType: pointerToNodeType(models.ContainerNodeType)})
		require.NoError(t, err)

		expected := []*models.Node{
			{
				NodeID:    "MySQLNode",
				NodeType:  models.ContainerNodeType,
				NodeName:  "Node for MySQL Service",
				MachineID: pointer.ToString("/machine_id/MySQLNode"),
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		require.Equal(t, expected, nodes)
	})

	t.Run("RemoveNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		err := models.RemoveNode(q, "", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Empty Node ID.`), err)

		err = models.RemoveNode(q, models.PMMServerNodeID, models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.PermissionDenied, `PMM Server node can't be removed.`), err)

		err = models.RemoveNode(q, "NoSuchNode", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "NoSuchNode" not found.`), err)

		err = models.RemoveNode(q, "GenericNode", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Node with ID "GenericNode" has agents.`), err)
		err = models.RemoveNode(q, "NodeWithPMMAgent", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Node with ID "NodeWithPMMAgent" has pmm-agent.`), err)
		err = models.RemoveNode(q, "MySQLNode", models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.FailedPrecondition, `Node with ID "MySQLNode" has services.`), err)

		err = models.RemoveNode(q, "EmptyNode", models.RemoveRestrict)
		assert.NoError(t, err)

		err = models.RemoveNode(q, "GenericNode", models.RemoveCascade)
		assert.NoError(t, err)
		err = models.RemoveNode(q, "NodeWithPMMAgent", models.RemoveCascade)
		assert.NoError(t, err)
		err = models.RemoveNode(q, "MySQLNode", models.RemoveCascade)
		assert.NoError(t, err)

		nodes, err := models.FindNodes(q, models.NodeFilters{})
		assert.NoError(t, err)
		require.Len(t, nodes, 1) // PMM Server
	})
}

func pointerToNodeType(nodeType models.NodeType) *models.NodeType {
	return &nodeType
}
