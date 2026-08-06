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
				MachineID: new("MySQLNode"),
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
				MachineID: new("GenericNode"),
			},
			&models.Agent{
				AgentID:      "pmm-agent",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: new("GenericNode"),
			},

			&models.Agent{
				AgentID:    "node_exporter",
				AgentType:  models.NodeExporterType,
				PMMAgentID: new("pmm-agent"),
				NodeID:     new("GenericNode"),
			},

			&models.Agent{
				AgentID:    "mysqld_exporter",
				AgentType:  models.MySQLdExporterType,
				PMMAgentID: new("pmm-agent"),
				ServiceID:  new("MySQL"),
			},

			&models.Node{
				NodeID:   "NodeWithPMMAgent",
				NodeType: models.GenericNodeType,
				NodeName: "Node With pmm-agent",
			},
			&models.Agent{
				AgentID:      "pmm-agent1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: new("NodeWithPMMAgent"),
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

			machineID := "GenericNode"
			_, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
				NodeName:  t.Name(),
				MachineID: new(machineID + "\n"),
			})
			require.NoError(t, err)

			structs, err := q.SelectAllFrom(models.NodeTable, "WHERE machine_id = $1 ORDER BY node_id", machineID)
			require.NoError(t, err)
			require.Len(t, structs, 2)
			expected := &models.Node{
				NodeID:    structs[0].(*models.Node).NodeID,
				NodeType:  models.GenericNodeType,
				NodeName:  t.Name(),
				MachineID: &machineID,
				CreatedAt: now,
				UpdatedAt: now,
			}
			assert.Equal(t, expected, structs[0])
			expected = &models.Node{
				NodeID:    "GenericNode",
				NodeType:  models.GenericNodeType,
				NodeName:  "Node for Agents",
				MachineID: &machineID, // \n trimmed
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
			MachineID: new("GenericNode"),
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:    "MySQLNode",
			NodeType:  models.ContainerNodeType,
			NodeName:  "Node for MySQL Service",
			MachineID: new("MySQLNode"),
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:    "NodeWithPMMAgent",
			NodeType:  models.GenericNodeType,
			NodeName:  "Node With pmm-agent",
			CreatedAt: now,
			UpdatedAt: now,
		}, {
			NodeID:          models.PMMServerNodeID,
			NodeType:        models.GenericNodeType,
			NodeName:        "pmm-server",
			Address:         "127.0.0.1",
			CreatedAt:       now,
			UpdatedAt:       now,
			IsPMMServerNode: true,
		}}
		require.Equal(t, expected, nodes)
	})

	t.Run("FindNodesByType", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		nodes, err := models.FindNodes(q, models.NodeFilters{NodeType: new(models.ContainerNodeType)})
		require.NoError(t, err)

		expected := []*models.Node{
			{
				NodeID:    "MySQLNode",
				NodeType:  models.ContainerNodeType,
				NodeName:  "Node for MySQL Service",
				MachineID: new("MySQLNode"),
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

		// PMM Server node with a generated ID (HA setup) is protected by the IsPMMServerNode flag
		haNode, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
			NodeName:        "pmm-server-ha",
			IsPMMServerNode: true,
		})
		require.NoError(t, err)
		err = models.RemoveNode(q, haNode.NodeID, models.RemoveRestrict)
		tests.AssertGRPCError(t, status.New(codes.PermissionDenied, `PMM Server node can't be removed.`), err)
		err = models.RemoveNode(q, haNode.NodeID, models.RemoveCascade)
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
		require.NoError(t, err)

		err = models.RemoveNode(q, "GenericNode", models.RemoveCascade)
		require.NoError(t, err)
		err = models.RemoveNode(q, "NodeWithPMMAgent", models.RemoveCascade)
		require.NoError(t, err)
		err = models.RemoveNode(q, "MySQLNode", models.RemoveCascade)
		require.NoError(t, err)

		nodes, err := models.FindNodes(q, models.NodeFilters{})
		require.NoError(t, err)
		require.Len(t, nodes, 2) // PMM Server + HA PMM Server node
	})
}

func TestRemoveStaleHANodes(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	// Two HA replica Nodes, one with a node_exporter, plus an unrelated monitored Node.
	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.Node{
				NodeID:          "ha-node-1",
				NodeType:        models.GenericNodeType,
				NodeName:        "pmm-ha-1",
				Address:         models.LocalhostAddr,
				IsPMMServerNode: true,
			},
			&models.Agent{
				AgentID:      "ha-agent-1",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: new("ha-node-1"),
			},
			&models.Node{
				NodeID:          "ha-node-2",
				NodeType:        models.GenericNodeType,
				NodeName:        "pmm-ha-2",
				Address:         models.LocalhostAddr,
				IsPMMServerNode: true,
			},
			&models.Agent{
				AgentID:      "ha-agent-2",
				AgentType:    models.PMMAgentType,
				RunsOnNodeID: new("ha-node-2"),
			},
			&models.Agent{
				AgentID:    "ha-node-exporter-2",
				AgentType:  models.NodeExporterType,
				PMMAgentID: new("ha-agent-2"),
				NodeID:     new("ha-node-2"),
			},
			&models.Node{
				NodeID:   "monitored-node",
				NodeType: models.GenericNodeType,
				NodeName: "Monitored Node",
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

	assertNodeExists := func(t *testing.T, q *reform.Querier, nodeID string) {
		t.Helper()
		_, err := models.FindNodeByID(q, nodeID)
		assert.NoError(t, err)
	}

	t.Run("RemovesScaledDownReplicaWithItsAgents", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		peers := []string{"pmm-ha-0.pmm-ha.pmm.svc.cluster.local:9761", " pmm-ha-1.pmm-ha.pmm.svc.cluster.local "}
		require.NoError(t, models.RemoveStaleHANodes(q, "pmm-ha-1", peers))

		assertNodeExists(t, q, "ha-node-1")
		_, err := models.FindAgentByID(q, "ha-agent-1")
		require.NoError(t, err)

		_, err = models.FindNodeByID(q, "ha-node-2")
		tests.AssertGRPCErrorCode(t, codes.NotFound, err)

		// the removal cascades to the agents of the stale node
		for _, agentID := range []string{"ha-agent-2", "ha-node-exporter-2"} {
			_, err := models.FindAgentByID(q, agentID)
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		}

		// neither monitored nodes nor the pre-HA pmm-server Node are touched
		assertNodeExists(t, q, "monitored-node")
		assertNodeExists(t, q, models.PMMServerNodeID)
	})

	t.Run("KeepsAllReplicasWhenNothingWasScaledDown", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// a dotless host with a port is what a hand-written PMM_HA_PEERS looks like
		peers := []string{"pmm-ha-1.pmm-ha:9761", "pmm-ha-2:9761"}
		require.NoError(t, models.RemoveStaleHANodes(q, "pmm-ha-1", peers))

		assertNodeExists(t, q, "ha-node-1")
		assertNodeExists(t, q, "ha-node-2")
	})

	t.Run("KeepsScaledDownReplicaThatStillMonitorsServices", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// an exporter for a remote instance, bound to the scaled-down replica's pmm-agent
		for _, str := range []reform.Struct{
			&models.Service{
				ServiceID:   "rds-service",
				ServiceType: models.MySQLServiceType,
				ServiceName: "RDS instance",
				NodeID:      "monitored-node",
				Address:     new("rds.example.com"),
				Port:        new(uint16(3306)),
			},
			&models.Agent{
				AgentID:    "rds-exporter",
				AgentType:  models.MySQLdExporterType,
				PMMAgentID: new("ha-agent-2"),
				ServiceID:  new("rds-service"),
			},
		} {
			require.NoError(t, q.Insert(str), "failed to INSERT %+v", str)
		}

		peers := []string{"pmm-ha-0.pmm-ha:9761", "pmm-ha-1.pmm-ha:9761"}
		require.NoError(t, models.RemoveStaleHANodes(q, "pmm-ha-1", peers))

		assertNodeExists(t, q, "ha-node-2")
		_, err := models.FindAgentByID(q, "rds-exporter")
		require.NoError(t, err)
	})

	t.Run("DoesNothingWhenPeersCantBeTrusted", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		for _, peers := range [][]string{
			{"pmm-ha-2.pmm-ha:9761"},                      // lists only the other replica
			{"10.244.1.7:9761", "10.244.2.8:9761"},        // no node names to read
			{"pmm-ha-1.pmm-ha:9761", "10.244.2.8:9761"},   // mixed: one entry hides a live replica
			{"pmm-ha-1.pmm-ha:9761", "pmm-ha-2/10.0.0.2"}, // memberlist "name/address" form
			nil,
		} {
			require.NoError(t, models.RemoveStaleHANodes(q, "pmm-ha-1", peers))

			assertNodeExists(t, q, "ha-node-1")
			assertNodeExists(t, q, "ha-node-2")
		}
	})
}
