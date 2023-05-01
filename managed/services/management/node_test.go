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

package management

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	nodev1beta1 "github.com/percona/pmm/api/managementpb/node"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestNodeService(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		setup := func(t *testing.T) (ctx context.Context, s *NodeService, teardown func(t *testing.T)) {
			t.Helper()

			ctx = logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			teardown = func(t *testing.T) {
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())
			}
			var apiKeyProvider mockApiKeyProvider
			apiKeyProvider.Test(t)
			apiKeyProvider.On("CreateAdminAPIKey", ctx, mock.AnythingOfType("string")).Return(int64(0), "test-token", nil)
			s = NewNodeService(db, &apiKeyProvider)

			return
		}

		t.Run("New", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			res, err := s.Register(ctx, &managementpb.RegisterNodeRequest{
				NodeType: inventorypb.NodeType_GENERIC_NODE,
				NodeName: "node",
				Address:  "some.address.org",
				Region:   "region",
			})
			expected := &managementpb.RegisterNodeResponse{
				GenericNode: &inventorypb.GenericNode{
					NodeId:   "/node_id/00000000-0000-4000-8000-000000000005",
					NodeName: "node",
					Address:  "some.address.org",
					Region:   "region",
				},
				ContainerNode: (*inventorypb.ContainerNode)(nil),
				PmmAgent: &inventorypb.PMMAgent{
					AgentId:      "/agent_id/00000000-0000-4000-8000-000000000006",
					RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000005",
				},
				Token: "test-token",
			}
			assert.Equal(t, expected, res)
			assert.NoError(t, err)

			t.Run("Exist", func(t *testing.T) {
				res, err = s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType: inventorypb.NodeType_GENERIC_NODE,
					NodeName: "node",
				})
				assert.Nil(t, res)
				tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "node" already exists.`), err)
			})

			t.Run("Reregister", func(t *testing.T) {
				res, err = s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType:   inventorypb.NodeType_GENERIC_NODE,
					NodeName:   "node",
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				expected := &managementpb.RegisterNodeResponse{
					GenericNode: &inventorypb.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-000000000008",
						NodeName: "node",
						Address:  "some.address.org",
						Region:   "region",
					},
					ContainerNode: (*inventorypb.ContainerNode)(nil),
					PmmAgent: &inventorypb.PMMAgent{
						AgentId:      "/agent_id/00000000-0000-4000-8000-000000000009",
						RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000008",
					},
					Token: "test-token",
				}
				assert.Equal(t, expected, res)
				assert.NoError(t, err)
			})
			t.Run("Reregister-force", func(t *testing.T) {
				res, err = s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType:   inventorypb.NodeType_GENERIC_NODE,
					NodeName:   "node-name-new",
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				expected := &managementpb.RegisterNodeResponse{
					GenericNode: &inventorypb.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-00000000000b",
						NodeName: "node-name-new",
						Address:  "some.address.org",
						Region:   "region",
					},
					ContainerNode: (*inventorypb.ContainerNode)(nil),
					PmmAgent: &inventorypb.PMMAgent{
						AgentId:      "/agent_id/00000000-0000-4000-8000-00000000000c",
						RunsOnNodeId: "/node_id/00000000-0000-4000-8000-00000000000b",
					},
					Token: "test-token",
				}
				assert.Equal(t, expected, res)
				assert.NoError(t, err)
			})
		})
	})

	t.Run("ListNodes", func(t *testing.T) {
		setup := func(t *testing.T) (ctx context.Context, s *MgmtNodeService, teardown func(t *testing.T)) {
			t.Helper()

			ctx = logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			ar := &mockAgentsRegistry{}
			ar.Test(t)

			vmClient := &mockVictoriaMetricsClient{}
			vmClient.Test(t)

			s = NewMgmtNodeService(db, ar, vmClient)

			teardown = func(t *testing.T) {
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())
				ar.AssertExpectations(t)
			}

			return
		}

		const (
			nodeExporterID      = "/agent_id/00000000-0000-4000-8000-000000000001"
			postgresqlServiceId = "/service_id/00000000-0000-4000-8000-000000000002"
		)

		t.Run("should output a list of all nodes, unfiltered", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

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

			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(metric, nil, nil).Times(2)
			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", nodeExporterID).Return(true).Once()
			res, err := s.ListNodes(ctx, &nodev1beta1.ListNodeRequest{})

			expected := &nodev1beta1.ListNodeResponse{
				Nodes: []*nodev1beta1.UniversalNode{
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
						CreatedAt:     nil,
						UpdatedAt:     nil,
						Agents: []*nodev1beta1.UniversalNode_Agent{
							{
								AgentId:     nodeExporterID,
								AgentType:   "node_exporter",
								Status:      "UNKNOWN",
								IsConnected: true,
							},
							{
								AgentId:     models.PMMServerAgentID,
								AgentType:   "pmm-agent",
								Status:      "",
								IsConnected: true,
							},
						},
						Services: []*nodev1beta1.UniversalNode_Service{
							{
								ServiceId:   postgresqlServiceId,
								ServiceType: "postgresql",
								ServiceName: "pmm-server-postgresql",
							},
						},
						Status: nodev1beta1.UniversalNode_UP,
					},
				},
			}

			assert.NoError(t, err)
			assert.Equal(t, expected, res)
		})

		t.Run("should output an empty list of nodes when filter condition is not satisfied", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(model.Vector{}, nil, nil).Times(2)
			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", nodeExporterID).Return(true).Once()

			res, err := s.ListNodes(ctx, &nodev1beta1.ListNodeRequest{
				NodeType: inventorypb.NodeType_REMOTE_NODE,
			})

			assert.NoError(t, err)
			assert.Empty(t, res.Nodes)
		})

		t.Run("should output a list of nodes when filter condition is satisfied", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown(t)

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
			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(metric, nil, nil).Times(2)
			s.r.(*mockAgentsRegistry).On("IsConnected", models.PMMServerAgentID).Return(true).Once()
			s.r.(*mockAgentsRegistry).On("IsConnected", nodeExporterID).Return(true).Once()

			res, err := s.ListNodes(ctx, &nodev1beta1.ListNodeRequest{
				NodeType: inventorypb.NodeType_GENERIC_NODE,
			})

			expected := &nodev1beta1.ListNodeResponse{
				Nodes: []*nodev1beta1.UniversalNode{
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
						CreatedAt:     nil,
						UpdatedAt:     nil,
						Agents: []*nodev1beta1.UniversalNode_Agent{
							{
								AgentId:     nodeExporterID,
								AgentType:   "node_exporter",
								Status:      "UNKNOWN",
								IsConnected: true,
							},
							{
								AgentId:     models.PMMServerAgentID,
								AgentType:   "pmm-agent",
								Status:      "",
								IsConnected: true,
							},
						},
						Services: []*nodev1beta1.UniversalNode_Service{
							{
								ServiceId:   postgresqlServiceId,
								ServiceType: "postgresql",
								ServiceName: "pmm-server-postgresql",
							},
						},
						Status: nodev1beta1.UniversalNode_UP,
					},
				},
			}

			assert.NoError(t, err)
			assert.Equal(t, expected, res)
		})
	})
}
