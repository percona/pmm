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
	nodev1beta1 "github.com/percona/pmm/api/management/v1/node"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestMgmtNodeService(t *testing.T) {
	t.Run("ListNodes", func(t *testing.T) {
		now = models.Now()

		setup := func(t *testing.T) (context.Context, *MgmtServiceService, func(t *testing.T)) {
			t.Helper()

			origNowF := models.Now
			models.Now = func() time.Time {
				return now
			}

			ctx := logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			ar := &mockAgentsRegistry{}
			ar.Test(t)

			vmdb := &mockPrometheusService{}
			vmdb.Test(t)

			state := &mockAgentsStateUpdater{}
			state.Test(t)

			vmClient := &mockVictoriaMetricsClient{}
			vmClient.Test(t)

			s := NewMgmtServiceService(db, ar, state, vmdb, vmClient)

			teardown := func(t *testing.T) {
				t.Helper()
				models.Now = origNowF
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())
				ar.AssertExpectations(t)
				state.AssertExpectations(t)
				vmdb.AssertExpectations(t)
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
			res, err := s.ListNodes(ctx, &nodev1beta1.ListNodesRequest{})
			require.NoError(t, err)

			expected := &nodev1beta1.ListNodesResponse{
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
						CreatedAt:     timestamppb.New(now),
						UpdatedAt:     timestamppb.New(now),
						Agents: []*nodev1beta1.UniversalNode_Agent{
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
						Services: []*nodev1beta1.UniversalNode_Service{
							{
								ServiceId:   postgresqlServiceID,
								ServiceType: "postgresql",
								ServiceName: "pmm-server-postgresql",
							},
						},
						Status: nodev1beta1.UniversalNode_STATUS_UP,
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

			res, err := s.ListNodes(ctx, &nodev1beta1.ListNodesRequest{
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

			res, err := s.ListNodes(ctx, &nodev1beta1.ListNodesRequest{
				NodeType: inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
			})
			require.NoError(t, err)

			expected := &nodev1beta1.ListNodesResponse{
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
						CreatedAt:     timestamppb.New(now),
						UpdatedAt:     timestamppb.New(now),
						Agents: []*nodev1beta1.UniversalNode_Agent{
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
						Services: []*nodev1beta1.UniversalNode_Service{
							{
								ServiceId:   postgresqlServiceID,
								ServiceType: "postgresql",
								ServiceName: "pmm-server-postgresql",
							},
						},
						Status: nodev1beta1.UniversalNode_STATUS_UP,
					},
				},
			}

			assert.Equal(t, expected, res)
		})
	})

	t.Run("GetNode", func(t *testing.T) {
		now := models.Now()

		setup := func(t *testing.T) (context.Context, *MgmtServiceService, func(t *testing.T)) {
			t.Helper()

			origNowF := models.Now
			models.Now = func() time.Time {
				return now
			}
			ctx := logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			ar := &mockAgentsRegistry{}
			ar.Test(t)

			vmdb := &mockPrometheusService{}
			vmdb.Test(t)

			state := &mockAgentsStateUpdater{}
			state.Test(t)

			vmClient := &mockVictoriaMetricsClient{}
			vmClient.Test(t)

			s := NewMgmtServiceService(db, ar, state, vmdb, vmClient)

			teardown := func(t *testing.T) {
				t.Helper()
				models.Now = origNowF
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())
				ar.AssertExpectations(t)
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
			s.vmClient.(*mockVictoriaMetricsClient).On("Query", ctx, mock.Anything, mock.Anything).Return(metric, nil, nil).Times(2)

			expected := &nodev1beta1.GetNodeResponse{
				Node: &nodev1beta1.UniversalNode{
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
					Status:        nodev1beta1.UniversalNode_STATUS_UP,
				},
			}

			node, err := s.GetNode(ctx, &nodev1beta1.GetNodeRequest{
				NodeId: models.PMMServerNodeID,
			})

			require.NoError(t, err)
			assert.Equal(t, expected, node)
		})

		t.Run("should return an error if such node_id doesn't exist", func(t *testing.T) {
			const nodeID = "00000000-0000-4000-8000-000000000000"
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			node, err := s.GetNode(ctx, &nodev1beta1.GetNodeRequest{
				NodeId: nodeID,
			})

			assert.Nil(t, node)
			tests.AssertGRPCError(t, status.New(codes.NotFound, fmt.Sprintf("Node with ID %q not found.", nodeID)), err)
		})

		t.Run("should return an error if the node_id parameter is empty", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			node, err := s.GetNode(ctx, &nodev1beta1.GetNodeRequest{
				NodeId: "",
			})

			assert.Nil(t, node)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "Empty Node ID."), err)
		})
	})
}
