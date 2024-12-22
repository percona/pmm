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

package management

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestNodeService(t *testing.T) {
	t.Run("NodeRegistration", func(t *testing.T) {
		getTestNodeName := func() string {
			return "test-node"
		}
		setup := func(t *testing.T) (context.Context, *ManagementService, func(t *testing.T)) {
			t.Helper()

			ctx := logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, database.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			serviceAccountID := int(0)
			nodeName := getTestNodeName()
			reregister := false

			r := &mockAgentsRegistry{}
			r.Test(t)

			vmdb := &mockPrometheusService{}
			vmdb.Test(t)

			state := &mockAgentsStateUpdater{}
			state.Test(t)

			authProvider := &mockGrafanaClient{}
			authProvider.Test(t)
			authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)

			vmClient := &mockVictoriaMetricsClient{}
			vmClient.Test(t)

			teardown := func(t *testing.T) {
				t.Helper()
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())

				r.AssertExpectations(t)
				vmdb.AssertExpectations(t)
				state.AssertExpectations(t)
				authProvider.AssertExpectations(t)
				vmClient.AssertExpectations(t)
			}

			s := NewManagementService(db, r, state, nil, nil, vmdb, nil, authProvider, vmClient)

			return ctx, s, teardown
		}

		ctx, s, teardown := setup(t)
		defer teardown(t)

		t.Run("New", func(t *testing.T) {
			nodeName := getTestNodeName()

			res, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
				NodeType: inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
				NodeName: nodeName,
				Address:  "some.address.org",
				Region:   "region",
			})
			expected := &managementv1.RegisterNodeResponse{
				GenericNode: &inventoryv1.GenericNode{
					NodeId:   "00000000-0000-4000-8000-000000000005",
					NodeName: nodeName,
					Address:  "some.address.org",
					Region:   "region",
				},
				ContainerNode: (*inventoryv1.ContainerNode)(nil),
				PmmAgent: &inventoryv1.PMMAgent{
					AgentId:      "00000000-0000-4000-8000-000000000006",
					RunsOnNodeId: "00000000-0000-4000-8000-000000000005",
				},
				Token: "test-token",
			}
			assert.Equal(t, expected, res)
			assert.NoError(t, err)
		})

		t.Run("Exist", func(t *testing.T) {
			res, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
				NodeType: inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
				NodeName: getTestNodeName(),
			})
			assert.Nil(t, res)
			tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test-node" already exists.`), err)
		})

		t.Run("Reregister", func(t *testing.T) {
			serviceAccountID := int(0)
			nodeName := "test-node-new"
			reregister := false

			authProvider := &mockGrafanaClient{}
			authProvider.Test(t)
			authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
			s.grafanaClient = authProvider

			_, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
				NodeType:   inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
				NodeName:   nodeName,
				Address:    "some.address.org",
				Region:     "region",
				Reregister: false,
			})

			tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with instance "some.address.org" and region "region" already exists.`), err)
		})

		t.Run("Reregister-force", func(t *testing.T) {
			serviceAccountID := int(0)
			nodeName := "test-node-new"
			reregister := true

			authProvider := &mockGrafanaClient{}
			authProvider.Test(t)
			authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
			s.grafanaClient = authProvider

			res, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
				NodeType:   inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
				NodeName:   nodeName,
				Address:    "some.address.org",
				Region:     "region",
				Reregister: true,
			})
			expected := &managementv1.RegisterNodeResponse{
				GenericNode: &inventoryv1.GenericNode{
					NodeId:   "00000000-0000-4000-8000-000000000008",
					NodeName: nodeName,
					Address:  "some.address.org",
					Region:   "region",
				},
				ContainerNode: (*inventoryv1.ContainerNode)(nil),
				PmmAgent: &inventoryv1.PMMAgent{
					AgentId:      "00000000-0000-4000-8000-000000000009",
					RunsOnNodeId: "00000000-0000-4000-8000-000000000008",
				},
				Token: "test-token",
			}
			assert.Equal(t, expected, res)
			assert.NoError(t, err)
		})

		t.Run("Register/Unregister", func(t *testing.T) {
			serviceAccountID := int(0)
			nodeName := getTestNodeName()
			reregister := true
			force := true

			authProvider := &mockGrafanaClient{}
			authProvider.Test(t)
			authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
			authProvider.On("DeleteServiceAccount", ctx, nodeName, force).Return("", nil)
			s.grafanaClient = authProvider

			state := &mockAgentsStateUpdater{}
			state.Test(t)
			state.On("RequestStateUpdate", ctx, "00000000-0000-4000-8000-00000000000c")
			s.state = state
			r := &mockAgentsRegistry{}
			r.Test(t)
			r.On("Kick", ctx, "00000000-0000-4000-8000-00000000000c").Return(true)
			s.r = r
			vmdb := &mockPrometheusService{}
			vmdb.Test(t)
			vmdb.On("RequestConfigurationUpdate").Return()
			s.vmdb = vmdb

			resRegister, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
				NodeType:   inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
				NodeName:   nodeName,
				Address:    "some.address.org",
				Region:     "region",
				Reregister: true,
			})
			assert.NoError(t, err)

			res, err := s.UnregisterNode(ctx, &managementv1.UnregisterNodeRequest{
				NodeId: resRegister.GenericNode.NodeId,
				Force:  true,
			})
			assert.NoError(t, err)
			assert.Equal(t, "", res.Warning)
		})
	})

	t.Run("ListNodes", func(t *testing.T) {
		now = models.Now()

		setup := func(t *testing.T) (context.Context, *ManagementService, func(t *testing.T)) {
			t.Helper()

			origNowF := models.Now
			models.Now = func() time.Time {
				return now
			}

			ctx := logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, database.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			ar := &mockAgentsRegistry{}
			ar.Test(t)

			vmdb := &mockPrometheusService{}
			vmdb.Test(t)

			state := &mockAgentsStateUpdater{}
			state.Test(t)

			cc := &mockConnectionChecker{}
			cc.Test(t)

			sib := &mockServiceInfoBroker{}
			sib.Test(t)

			vmClient := &mockVictoriaMetricsClient{}
			vmClient.Test(t)

			vc := &mockVersionCache{}
			vc.Test(t)

			grafanaClient := &mockGrafanaClient{}
			grafanaClient.Test(t)

			s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)

			teardown := func(t *testing.T) {
				t.Helper()
				models.Now = origNowF
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())
				ar.AssertExpectations(t)
				state.AssertExpectations(t)
				cc.AssertExpectations(t)
				sib.AssertExpectations(t)
				vmdb.AssertExpectations(t)
				vc.AssertExpectations(t)
				grafanaClient.AssertExpectations(t)
				vmClient.AssertExpectations(t)
			}

			return ctx, s, teardown
		}

		const (
			nodeExporterID      = "00000000-0000-4000-8000-000000000001"
			postgresqlServiceID = "00000000-0000-4000-8000-000000000002"
		)

		t.Run("should output an unfiltered list of all nodes", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			metric := model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"__name__": "up",
						"node_id":  "pmm-server",
					},
					Timestamp: 1,
					Value:     1,
				},
			}

			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(metric, nil, nil).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", nodeExporterID).Return(true).Once()
			res, err := s.ListNodes(ctx, &managementv1.ListNodesRequest{})
			require.NoError(t, err)

			expected := &managementv1.ListNodesResponse{
				Nodes: []*managementv1.UniversalNode{
					{
						NodeId:        "pmm-server",
						NodeType:      "generic",
						NodeName:      "pmm-server",
						MachineId:     "",
						Distro:        "",
						NodeModel:     "",
						ContainerId:   "",
						ContainerName: "",
						Address:       "127.0.0.1",
						Region:        "",
						Az:            "",
						CustomLabels:  nil,
						CreatedAt:     timestamppb.New(now),
						UpdatedAt:     timestamppb.New(now),
						Agents: []*managementv1.UniversalNode_Agent{
							{
								AgentId:     nodeExporterID,
								AgentType:   "node_exporter",
								Status:      "AGENT_STATUS_UNKNOWN",
								IsConnected: true,
							},
							{
								AgentId:     models.PMMServerAgentID,
								AgentType:   "pmm-agent",
								Status:      "",
								IsConnected: true,
							},
						},
						Services: []*managementv1.UniversalNode_Service{
							{
								ServiceId:   postgresqlServiceID,
								ServiceType: "postgresql",
								ServiceName: "pmm-server-postgresql",
							},
						},
						Status: managementv1.UniversalNode_STATUS_UP,
					},
				},
			}

			assert.Equal(t, expected, res)
		})

		t.Run("should output an empty list of nodes when filter condition is not satisfied", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(model.Vector{}, nil, nil).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", nodeExporterID).Return(true).Once()

			res, err := s.ListNodes(ctx, &managementv1.ListNodesRequest{
				NodeType: inventoryv1.NodeType_NODE_TYPE_REMOTE_NODE,
			})

			require.NoError(t, err)
			assert.Empty(t, res.Nodes)
		})

		t.Run("should output a list of nodes when filter condition is satisfied", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			metric := model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"__name__": "up",
						"node_id":  "pmm-server",
					},
					Timestamp: 1,
					Value:     1,
				},
			}
			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(metric, nil, nil).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", nodeExporterID).Return(true).Once()

			res, err := s.ListNodes(ctx, &managementv1.ListNodesRequest{
				NodeType: inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
			})
			require.NoError(t, err)

			expected := &managementv1.ListNodesResponse{
				Nodes: []*managementv1.UniversalNode{
					{
						NodeId:        "pmm-server",
						NodeType:      "generic",
						NodeName:      "pmm-server",
						MachineId:     "",
						Distro:        "",
						NodeModel:     "",
						ContainerId:   "",
						ContainerName: "",
						Address:       "127.0.0.1",
						Region:        "",
						Az:            "",
						CustomLabels:  nil,
						CreatedAt:     timestamppb.New(now),
						UpdatedAt:     timestamppb.New(now),
						Agents: []*managementv1.UniversalNode_Agent{
							{
								AgentId:     nodeExporterID,
								AgentType:   "node_exporter",
								Status:      "AGENT_STATUS_UNKNOWN",
								IsConnected: true,
							},
							{
								AgentId:     models.PMMServerAgentID,
								AgentType:   "pmm-agent",
								Status:      "",
								IsConnected: true,
							},
						},
						Services: []*managementv1.UniversalNode_Service{
							{
								ServiceId:   postgresqlServiceID,
								ServiceType: "postgresql",
								ServiceName: "pmm-server-postgresql",
							},
						},
						Status: managementv1.UniversalNode_STATUS_UP,
					},
				},
			}

			assert.Equal(t, expected, res)
		})
	})

	t.Run("GetNode", func(t *testing.T) {
		now := models.Now()

		setup := func(t *testing.T) (context.Context, *ManagementService, func(t *testing.T)) {
			t.Helper()

			origNowF := models.Now
			models.Now = func() time.Time {
				return now
			}
			ctx := logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, database.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			ar := &mockAgentsRegistry{}
			ar.Test(t)

			vmdb := &mockPrometheusService{}
			vmdb.Test(t)

			state := &mockAgentsStateUpdater{}
			state.Test(t)

			cc := &mockConnectionChecker{}
			cc.Test(t)

			sib := &mockServiceInfoBroker{}
			sib.Test(t)

			vc := &mockVersionCache{}
			vc.Test(t)

			grafanaClient := &mockGrafanaClient{}
			grafanaClient.Test(t)

			vmClient := &mockVictoriaMetricsClient{}
			vmClient.Test(t)

			s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, grafanaClient, vmClient)

			teardown := func(t *testing.T) {
				t.Helper()
				models.Now = origNowF
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())

				ar.AssertExpectations(t)
				state.AssertExpectations(t)
				cc.AssertExpectations(t)
				sib.AssertExpectations(t)
				vmdb.AssertExpectations(t)
				vc.AssertExpectations(t)
				grafanaClient.AssertExpectations(t)
				vmClient.AssertExpectations(t)
			}

			return ctx, s, teardown
		}

		t.Run("should query the node by its id", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			metric := model.Vector{
				&model.Sample{
					Metric: model.Metric{
						"__name__": "up",
						"node_id":  "pmm-server",
					},
					Timestamp: 1,
					Value:     1,
				},
			}
			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(metric, nil, nil).Times(1)

			expected := &managementv1.GetNodeResponse{
				Node: &managementv1.UniversalNode{
					NodeId:        "pmm-server",
					NodeType:      "generic",
					NodeName:      "pmm-server",
					MachineId:     "",
					Distro:        "",
					NodeModel:     "",
					ContainerId:   "",
					ContainerName: "",
					Address:       "127.0.0.1",
					Region:        "",
					Az:            "",
					CustomLabels:  nil,
					CreatedAt:     timestamppb.New(now),
					UpdatedAt:     timestamppb.New(now),
					Status:        managementv1.UniversalNode_STATUS_UP,
				},
			}

			node, err := s.GetNode(ctx, &managementv1.GetNodeRequest{
				NodeId: models.PMMServerNodeID,
			})

			require.NoError(t, err)
			assert.Equal(t, expected, node)
		})

		t.Run("should return an error if such node_id doesn't exist", func(t *testing.T) {
			const nodeID = "00000000-0000-4000-8000-000000000000"
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			node, err := s.GetNode(ctx, &managementv1.GetNodeRequest{
				NodeId: nodeID,
			})

			assert.Nil(t, node)
			tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Node with ID %q not found.", nodeID)), err)
		})

		t.Run("should return an error if the node_id parameter is empty", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			node, err := s.GetNode(ctx, &managementv1.GetNodeRequest{
				NodeId: "",
			})

			assert.Nil(t, node)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Empty Node ID."), err)
		})
	})
}
