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

		t.Run("EnvironmentTakesOverCustomLabel", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			labels := map[string]string{"environment": "staging", "team": "db"}
			node, err := models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
				NodeName:     t.Name(),
				Environment:  "prod",
				CustomLabels: labels,
			})
			require.NoError(t, err)
			custom, err := node.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"team": "db"}, custom)
			assert.Equal(t, "staging", labels["environment"], "the caller's labels are left as they are")

			// Without an environment the custom label stays: it was the way to give a Node one before the field.
			node, err = models.CreateNode(q, models.GenericNodeType, &models.CreateNodeParams{
				NodeName:     t.Name() + "-custom",
				CustomLabels: labels,
			})
			require.NoError(t, err)
			unified, err := node.UnifiedLabels()
			require.NoError(t, err)
			assert.Equal(t, "staging", unified["environment"])
		})
	})

	t.Run("ChangeNode", func(t *testing.T) {
		t.Run("ChangesOnlyGivenFields", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			_, err := models.ChangeNode(q, "GenericNode", &models.ChangeNodeParams{NodeModel: new("m5.large")})
			require.NoError(t, err)

			node, err := models.ChangeNode(q, "GenericNode", &models.ChangeNodeParams{
				Environment:  new("prod"),
				CustomLabels: map[string]string{"team": "db"},
			})
			require.NoError(t, err)

			stored, err := models.FindNodeByID(q, "GenericNode")
			require.NoError(t, err)
			assert.Equal(t, node, stored)
			assert.Equal(t, "prod", stored.Environment)
			assert.Equal(t, "m5.large", stored.NodeModel)

			labels, err := stored.UnifiedLabels()
			require.NoError(t, err)
			assert.Equal(t, "prod", labels["environment"])
			assert.Equal(t, "db", labels["team"])
		})

		t.Run("NothingToChange", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			node, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{})
			require.NoError(t, err)
			assert.Equal(t, "Empty Node", node.NodeName)
		})

		t.Run("EmptyCustomLabelsRemoveThem", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			_, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{CustomLabels: map[string]string{"team": "db"}})
			require.NoError(t, err)
			_, err = models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{CustomLabels: map[string]string{}})
			require.NoError(t, err)

			stored, err := models.FindNodeByID(q, "EmptyNode")
			require.NoError(t, err)
			labels, err := stored.GetCustomLabels()
			require.NoError(t, err)
			assert.Empty(t, labels)
		})

		t.Run("EnvironmentTakesOverCustomLabel", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			// Before the environment field, a custom label was the way to give a Node an environment.
			_, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{
				CustomLabels: map[string]string{"environment": "staging", "team": "db"},
			})
			require.NoError(t, err)

			node, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{Environment: new("prod")})
			require.NoError(t, err)
			custom, err := node.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"team": "db"}, custom)
			labels, err := node.UnifiedLabels()
			require.NoError(t, err)
			assert.Equal(t, "prod", labels["environment"])

			// The same goes for a custom label given in the same change.
			node, err = models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{
				Environment:  new("qa"),
				CustomLabels: map[string]string{"environment": "staging", "team": "ops"},
			})
			require.NoError(t, err)
			custom, err = node.GetCustomLabels()
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"team": "ops"}, custom)
			assert.Equal(t, "qa", node.Environment)

			// Next to a set environment the custom label would override it again.
			_, err = models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{
				CustomLabels: map[string]string{"environment": "staging"},
			})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument,
				`Node Empty Node has an environment, so it can't also have an "environment" custom label; change the environment instead.`), err)
		})

		t.Run("ClearingEnvironmentRemovesCustomLabel", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			_, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{
				CustomLabels: map[string]string{"environment": "staging"},
			})
			require.NoError(t, err)

			node, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{Environment: new("")})
			require.NoError(t, err)
			labels, err := node.UnifiedLabels()
			require.NoError(t, err)
			assert.NotContains(t, labels, "environment")
		})

		t.Run("InvalidLabelName", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			_, err := models.ChangeNode(q, "EmptyNode", &models.ChangeNodeParams{CustomLabels: map[string]string{"__name__": "x"}})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `Invalid label name "__name__".`), err)
		})

		t.Run("NotFound", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			_, err := models.ChangeNode(q, "NoSuchNode", &models.ChangeNodeParams{Environment: new("prod")})
			tests.AssertGRPCError(t, status.New(codes.NotFound, `Node with ID "NoSuchNode" not found.`), err)
		})

		t.Run("RemoteRDSRegionIsFixed", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			rds, err := models.CreateNode(q, models.RemoteRDSNodeType, &models.CreateNodeParams{
				NodeName:   "rds",
				Address:    "rds.abc.us-east-1.rds.amazonaws.com",
				InstanceID: "rds",
				Region:     new("us-east-1"),
			})
			require.NoError(t, err)

			_, err = models.ChangeNode(q, rds.NodeID, &models.ChangeNodeParams{Region: new("eu-west-1")})
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, `The region of a Remote RDS Node can't be changed.`), err)

			node, err := models.ChangeNode(q, rds.NodeID, &models.ChangeNodeParams{Region: new("us-east-1"), AZ: new("us-east-1a")})
			require.NoError(t, err)
			assert.Equal(t, "us-east-1", pointer.GetString(node.Region))
			assert.Equal(t, "us-east-1a", node.AZ)
		})

		t.Run("RegionStaysUniqueForAddress", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			for _, n := range []struct{ name, region string }{{"remote1", "r1"}, {"remote2", "r2"}} {
				_, err := models.CreateNode(q, models.RemoteNodeType, &models.CreateNodeParams{
					NodeName: n.name,
					Address:  "10.0.0.1",
					Region:   new(n.region),
				})
				require.NoError(t, err)
			}
			remote2, err := models.FindNodeByName(q, "remote2")
			require.NoError(t, err)

			_, err = models.ChangeNode(q, remote2.NodeID, &models.ChangeNodeParams{Region: new("r1")})
			tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with address "10.0.0.1" and region "r1" already exists.`), err)

			node, err := models.ChangeNode(q, remote2.NodeID, &models.ChangeNodeParams{Region: new("")})
			require.NoError(t, err)
			assert.Nil(t, node.Region)
		})

		t.Run("ServiceEnvironmentOverridesNodeEnvironment", func(t *testing.T) {
			q, teardown := setup(t)
			defer teardown(t)

			node, err := models.ChangeNode(q, "MySQLNode", &models.ChangeNodeParams{Environment: new("prod")})
			require.NoError(t, err)
			service, err := models.FindServiceByID(q, "MySQL")
			require.NoError(t, err)

			labels, err := models.MergeLabels(node, service, nil)
			require.NoError(t, err)
			assert.Equal(t, "prod", labels["environment"])

			service.Environment = "qa"
			labels, err = models.MergeLabels(node, service, nil)
			require.NoError(t, err)
			assert.Equal(t, "qa", labels["environment"])
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
			NodeType:        models.ContainerNodeType,
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
			}, {
				NodeID:          models.PMMServerNodeID,
				NodeType:        models.ContainerNodeType,
				NodeName:        "pmm-server",
				Address:         "127.0.0.1",
				CreatedAt:       now,
				UpdatedAt:       now,
				IsPMMServerNode: true,
			},
		}
		require.Equal(t, expected, nodes)
	})

	t.Run("FindNodesByIsPMMServerNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		nodes, err := models.FindNodes(q, models.NodeFilters{IsPMMServerNode: new(true)})
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		assert.Equal(t, models.PMMServerNodeID, nodes[0].NodeID)

		nodes, err = models.FindNodes(q, models.NodeFilters{IsPMMServerNode: new(false)})
		require.NoError(t, err)
		nodeIDs := make([]string, len(nodes))
		for i, node := range nodes {
			nodeIDs[i] = node.NodeID
		}
		assert.Equal(t, []string{"EmptyNode", "GenericNode", "MySQLNode", "NodeWithPMMAgent"}, nodeIDs)
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

// insertHAFixtures adds two HA replica Nodes, one with a node_exporter, plus an unrelated
// monitored Node, on top of the PMM Server fixtures.
func insertHAFixtures(t *testing.T, q *reform.Querier) {
	t.Helper()

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
}

func assertNodeExists(t *testing.T, q *reform.Querier, nodeID string) {
	t.Helper()
	_, err := models.FindNodeByID(q, nodeID)
	require.NoError(t, err)
}

func TestFindStaleHANodes(t *testing.T) {
	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	setup := func(t *testing.T) (*reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		insertHAFixtures(t, tx.Querier)

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return tx.Querier, teardown
	}

	assertStale := func(t *testing.T, nodes []*models.Node, nodeIDs ...string) {
		t.Helper()
		actual := make([]string, 0, len(nodes))
		for _, node := range nodes {
			actual = append(actual, node.NodeID)
		}
		assert.ElementsMatch(t, nodeIDs, actual)
	}

	t.Run("ReportsScaledDownReplica", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		peers := []string{"pmm-ha-0.pmm-ha.pmm.svc.cluster.local:9761", " pmm-ha-1.pmm-ha.pmm.svc.cluster.local "}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		// neither the live replica, the monitored nodes nor the pre-HA pmm-server Node are reported
		assertStale(t, stale, "ha-node-2")
	})

	t.Run("ReportsScaledDownReplicaDespiteBlankPeers", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// a trailing comma in PMM_HA_PEERS, or a blank element in the list the chart joins
		peers := []string{"pmm-ha-0.pmm-ha:9761", "pmm-ha-1.pmm-ha:9761", "", "   "}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale, "ha-node-2")
	})

	t.Run("ReportsEveryDepartedReplicaWhenScaledToOne", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// The surviving replica's own Node. It goes here rather than into insertHAFixtures, where
		// ReportsNothingWhenNothingWasScaledDown would then see it as stale.
		require.NoError(t, q.Insert(&models.Node{
			NodeID:          "ha-node-0",
			NodeType:        models.GenericNodeType,
			NodeName:        "pmm-ha-0",
			Address:         models.LocalhostAddr,
			IsPMMServerNode: true,
		}))

		// what the chart renders at replicas: 1 - a single entry, and it is this pod
		peers := []string{"pmm-ha-0.monitoring-service.pmm.svc.cluster.local"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-0", peers)
		require.NoError(t, err)

		// both departed replicas in one sweep, the survivor's own Node untouched
		assertStale(t, stale, "ha-node-1", "ha-node-2")
	})

	t.Run("ReportsNothingWhenNothingWasScaledDown", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// a dotless host with a port is what a hand-written PMM_HA_PEERS looks like
		peers := []string{"pmm-ha-1.pmm-ha:9761", "pmm-ha-2:9761"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale)
	})

	// A Node with no pmm-agent of its own yields no IDs to filter the second query by. An empty
	// PMMAgentIDs is not a filter, so an unguarded query there would match every Agent in the
	// inventory and the Node would look like it monitors every Service.
	t.Run("ReportsScaledDownReplicaWithNoAgents", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		require.NoError(t, q.Insert(&models.Node{
			NodeID:          "ha-node-3",
			NodeType:        models.GenericNodeType,
			NodeName:        "pmm-ha-3",
			Address:         models.LocalhostAddr,
			IsPMMServerNode: true,
		}))

		peers := []string{"pmm-ha-1.pmm-ha:9761", "pmm-ha-2:9761"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale, "ha-node-3")
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
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale)
	})

	t.Run("KeepsScaledDownReplicaWithAServiceOnIt", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// a Service registered against the replica's own Node, monitored from somewhere else
		require.NoError(t, q.Insert(&models.Service{
			ServiceID:   "service-on-replica",
			ServiceType: models.MySQLServiceType,
			ServiceName: "MySQL on the replica",
			NodeID:      "ha-node-2",
			Address:     new("mysql.example.com"),
			Port:        new(uint16(3306)),
		}))

		peers := []string{"pmm-ha-0.pmm-ha:9761", "pmm-ha-1.pmm-ha:9761"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale)
	})

	t.Run("KeepsScaledDownReplicaRunningAnExternalExporter", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// an external exporter in pull mode: no pmm-agent owns it, it just runs on the replica
		for _, str := range []reform.Struct{
			&models.Service{
				ServiceID:     "external-service",
				ServiceType:   models.ExternalServiceType,
				ServiceName:   "External instance",
				NodeID:        "monitored-node",
				Address:       new("external.example.com"),
				Port:          new(uint16(9100)),
				ExternalGroup: "external",
			},
			&models.Agent{
				AgentID:      "external-exporter",
				AgentType:    models.ExternalExporterType,
				RunsOnNodeID: new("ha-node-2"),
				ServiceID:    new("external-service"),
			},
		} {
			require.NoError(t, q.Insert(str), "failed to INSERT %+v", str)
		}

		peers := []string{"pmm-ha-0.pmm-ha:9761", "pmm-ha-1.pmm-ha:9761"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale)
	})

	t.Run("KeepsPreHAPMMServerNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// A deployment converted from non-HA keeps a Node with the literal "pmm-server" ID; without
		// the internal PostgreSQL Service nothing marks it as monitoring, so only its ID keeps it.
		service, err := models.FindServiceByName(q, models.PMMServerPostgreSQLServiceName)
		require.NoError(t, err)
		require.NoError(t, models.RemoveService(q, service.ServiceID, models.RemoveCascade))

		peers := []string{"pmm-ha-0.pmm-ha:9761", "pmm-ha-1.pmm-ha:9761"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale, "ha-node-2")
	})

	t.Run("KeepsPreHAPMMServerNodeAfterSetupRetry", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		// setupPMMServerHAAgents points PMMServerNodeID at the replica's generated Node ID, and a
		// failed commit makes pmm-managed retry the whole setup; the guards can't key off that var.
		restore := models.PMMServerNodeID
		models.PMMServerNodeID = "ha-node-1"
		t.Cleanup(func() { models.PMMServerNodeID = restore })

		service, err := models.FindServiceByName(q, models.PMMServerPostgreSQLServiceName)
		require.NoError(t, err)
		require.NoError(t, models.RemoveService(q, service.ServiceID, models.RemoveCascade))

		peers := []string{"pmm-ha-0.pmm-ha:9761", "pmm-ha-1.pmm-ha:9761"}
		stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
		require.NoError(t, err)

		assertStale(t, stale, "ha-node-2")
	})

	t.Run("ReportsNothingWhenPeersCarryNoNames", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		for _, peers := range [][]string{
			{"10.244.1.7:9761", "10.244.2.8:9761"},        // no node names to read
			{"pmm-ha-1.pmm-ha:9761", "10.244.2.8:9761"},   // mixed: one entry hides a live replica
			{"pmm-ha-1.pmm-ha:9761", "pmm-ha-2/10.0.0.2"}, // memberlist "name/address" form
			{"pmm-ha-1.pmm-ha:9761", "2001:db8::7"},       // an unbracketed IPv6 entry hides a live replica
			{"pmm-ha-1.pmm-ha:9761", "[2001:db8::7]:9761"},
		} {
			stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
			require.NoError(t, err, "peers: %v", peers)

			assertStale(t, stale)
		}
	})

	t.Run("FailsWhenPeersDontDescribeThisNode", func(t *testing.T) {
		q, teardown := setup(t)
		defer teardown(t)

		for _, peers := range [][]string{
			{"pmm-ha-2.pmm-ha:9761"}, // lists only the other replica
			{"", "   "},              // only blank entries, so nothing is left to compare against
			{},
			nil,
		} {
			stale, err := models.FindStaleHANodes(q, "pmm-ha-1", peers)
			require.Error(t, err, "peers: %v", peers)

			assertStale(t, stale)
		}
	})
}

func TestRemoveStaleHANode(t *testing.T) {
	// The removal is meant to run in a transaction of its own, which wouldn't see fixtures held in
	// an uncommitted one - so every subtest gets a database of its own instead.
	setup := func(t *testing.T) *reform.DB {
		t.Helper()
		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		insertHAFixtures(t, db.Querier)

		return db
	}

	t.Run("RemovesTheNodeWithItsAgents", func(t *testing.T) {
		db := setup(t)
		q := db.Querier

		require.NoError(t, db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, "ha-node-2")
		}))

		_, err := models.FindNodeByID(q, "ha-node-2")
		tests.AssertGRPCErrorCode(t, codes.NotFound, err)

		// the removal cascades to the agents of the stale node
		for _, agentID := range []string{"ha-agent-2", "ha-node-exporter-2"} {
			_, err := models.FindAgentByID(q, agentID)
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		}

		for _, nodeID := range []string{"ha-node-1", "monitored-node", models.PMMServerNodeID} {
			assertNodeExists(t, q, nodeID)
		}

		// the cascade must not reach past the stale Node: the live replica's pmm-agent, and PMM
		// Server's own Service and pmm-agent, are untouched
		_, err = models.FindAgentByID(q, "ha-agent-1")
		require.NoError(t, err)
		_, err = models.FindServiceByName(q, models.PMMServerPostgreSQLServiceName)
		require.NoError(t, err)
		_, err = models.FindAgentByID(q, models.PMMServerAgentID)
		require.NoError(t, err)
	})

	t.Run("ReportsNotFoundWhenAlreadyRemoved", func(t *testing.T) {
		db := setup(t)
		q := db.Querier

		require.NoError(t, db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, "ha-node-2")
		}))

		// what a replica sees when another one won the race; removeStaleHANodes reads this as
		// "already removed by another replica" rather than as a failure
		err := db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, "ha-node-2")
		})
		tests.AssertGRPCErrorCode(t, codes.NotFound, err)

		assertNodeExists(t, q, "ha-node-1")
	})

	t.Run("LeavesTheNodeWholeWhenRemovalFails", func(t *testing.T) {
		db := setup(t)
		q := db.Querier

		// RemoveAgent refuses to delete PMMServerAgentID, which HA setup points at the local
		// replica's own agent; pointing it at ha-agent-2 makes removing ha-node-2 fail after its
		// node_exporter, which removeNode deletes first, is already gone.
		restore := models.PMMServerAgentID
		models.PMMServerAgentID = "ha-agent-2"
		t.Cleanup(func() { models.PMMServerAgentID = restore })

		err := db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, "ha-node-2")
		})
		tests.AssertGRPCErrorCode(t, codes.PermissionDenied, err)

		// the transaction rolled the whole Node back, node_exporter included
		assertNodeExists(t, q, "ha-node-2")
		for _, agentID := range []string{"ha-agent-2", "ha-node-exporter-2"} {
			_, err := models.FindAgentByID(q, agentID)
			require.NoError(t, err)
		}
	})

	t.Run("RefusesANodeThatStillMonitorsServices", func(t *testing.T) {
		db := setup(t)
		q := db.Querier

		// a Service bound to the stale replica's pmm-agent between FindStaleHANodes and the removal
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

		err := db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, "ha-node-2")
		})
		tests.AssertGRPCErrorCode(t, codes.FailedPrecondition, err)

		assertNodeExists(t, q, "ha-node-2")
		_, err = models.FindAgentByID(q, "rds-exporter")
		require.NoError(t, err)
	})

	t.Run("RefusesANodeWithAServiceOnIt", func(t *testing.T) {
		db := setup(t)
		q := db.Querier

		// the re-check is the last line of defence, so it has to cover the same ground as the
		// selection: a Service attached to the Node would be cascaded away with it
		require.NoError(t, q.Insert(&models.Service{
			ServiceID:   "service-on-replica",
			ServiceType: models.MySQLServiceType,
			ServiceName: "MySQL on the replica",
			NodeID:      "ha-node-2",
			Address:     new("mysql.example.com"),
			Port:        new(uint16(3306)),
		}))

		err := db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, "ha-node-2")
		})
		tests.AssertGRPCErrorCode(t, codes.FailedPrecondition, err)

		assertNodeExists(t, q, "ha-node-2")
		_, err = models.FindServiceByID(q, "service-on-replica")
		require.NoError(t, err)
	})

	t.Run("RefusesThePreHAPMMServerNode", func(t *testing.T) {
		db := setup(t)
		q := db.Querier

		// With the internal PostgreSQL Service gone, nothing marks it as monitoring, so only the
		// literal "pmm-server" ID stands between the lifted PMM Server ban and the Node.
		service, err := models.FindServiceByName(q, models.PMMServerPostgreSQLServiceName)
		require.NoError(t, err)
		require.NoError(t, models.RemoveService(q, service.ServiceID, models.RemoveCascade))

		err = db.InTransaction(func(tx *reform.TX) error {
			return models.RemoveStaleHANode(tx.Querier, models.PMMServerNodeID)
		})
		// the Node ban, not RemoveAgent's ban on the PMM Server pmm-agent, which also answers
		// PermissionDenied once the removal gets that far
		tests.AssertGRPCError(t, status.New(codes.PermissionDenied, `PMM Server node can't be removed.`), err)

		assertNodeExists(t, q, models.PMMServerNodeID)
	})
}
