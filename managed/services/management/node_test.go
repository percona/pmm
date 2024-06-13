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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestNodeService(t *testing.T) {
	getTestNodeName := func() string {
		return "test-node"
	}
	t.Run("Register/Unregister", func(t *testing.T) {
		setup := func(t *testing.T) (ctx context.Context, s *NodeService, teardown func(t *testing.T)) {
			t.Helper()

			ctx = logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
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

			authProvider := &mockAuthProvider{}
			authProvider.Test(t)

			teardown = func(t *testing.T) {
				t.Helper()
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())

				r.AssertExpectations(t)
				vmdb.AssertExpectations(t)
				state.AssertExpectations(t)
				authProvider.AssertExpectations(t)
			}
			md := metadata.New(map[string]string{
				"Authorization": "Basic username:password",
			})
			ctx = metadata.NewIncomingContext(ctx, md)

			authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)

			s = NewNodeService(db, authProvider, inventory.NewNodesService(db, r, state, vmdb))

			return
		}

		ctx, s, teardown := setup(t)
		defer teardown(t)

		t.Run("New", func(t *testing.T) {
			nodeName := getTestNodeName()

			res, err := s.Register(ctx, &managementpb.RegisterNodeRequest{
				NodeType: inventorypb.NodeType_GENERIC_NODE,
				NodeName: nodeName,
				Address:  "some.address.org",
				Region:   "region",
			})
			expected := &managementpb.RegisterNodeResponse{
				GenericNode: &inventorypb.GenericNode{
					NodeId:   "/node_id/00000000-0000-4000-8000-000000000005",
					NodeName: nodeName,
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
					NodeName: getTestNodeName(),
				})
				assert.Nil(t, res)
				tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test-node" already exists.`), err)
			})

			t.Run("Reregister", func(t *testing.T) {
				serviceAccountID := int(0)
				nodeName := "test-node-new"
				reregister := false

				var authProvider mockAuthProvider
				authProvider.Test(t)
				authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
				s.ap = &authProvider

				res, err = s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType:   inventorypb.NodeType_GENERIC_NODE,
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

				var authProvider mockAuthProvider
				authProvider.Test(t)
				authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
				s.ap = &authProvider

				res, err = s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType:   inventorypb.NodeType_GENERIC_NODE,
					NodeName:   nodeName,
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				expected := &managementpb.RegisterNodeResponse{
					GenericNode: &inventorypb.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-000000000008",
						NodeName: nodeName,
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

			t.Run("Unregister", func(t *testing.T) {
				serviceAccountID := int(0)
				nodeName := getTestNodeName()
				reregister := true
				force := true

				var authProvider mockAuthProvider
				authProvider.Test(t)
				authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
				authProvider.On("DeleteServiceAccount", ctx, nodeName, force).Return("", nil)
				s.ap = &authProvider

				state := &mockAgentsStateUpdater{}
				state.Test(t)
				state.On("RequestStateUpdate", ctx, "/agent_id/00000000-0000-4000-8000-00000000000c")
				r := &mockAgentsRegistry{}
				r.Test(t)
				r.On("Kick", ctx, "/agent_id/00000000-0000-4000-8000-00000000000c").Return(true)
				vmdb := &mockPrometheusService{}
				vmdb.Test(t)
				vmdb.On("RequestConfigurationUpdate").Return()
				s.ns = inventory.NewNodesService(s.db, r, state, vmdb)

				resRegister, err := s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType:   inventorypb.NodeType_GENERIC_NODE,
					NodeName:   nodeName,
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				assert.NoError(t, err)

				expected := &managementpb.RegisterNodeResponse{
					GenericNode: &inventorypb.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-00000000000b",
						NodeName: nodeName,
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
				assert.Equal(t, expected, resRegister)

				res, err := s.Unregister(ctx, &managementpb.UnregisterNodeRequest{
					NodeId: resRegister.GenericNode.NodeId,
					Force:  true,
				})
				assert.NoError(t, err)
				assert.Equal(t, "", res.Warning)
			})
		})
	})
}
