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

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

func TestNodeService(t *testing.T) {
	getTestNodeName := func() string {
		return "test-node"
	}
	t.Run("Register/Unregister", func(t *testing.T) {
		setup := func(t *testing.T) (context.Context, *ManagementService, func(t *testing.T)) {
			t.Helper()

			ctx := logger.Set(context.Background(), t.Name())
			uuid.SetRand(&tests.IDReader{})

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

			teardown := func(t *testing.T) {
				t.Helper()
				uuid.SetRand(nil)

				require.NoError(t, sqlDB.Close())
			}
			md := metadata.New(map[string]string{
				"Authorization": "Basic username:password",
			})
			ctx = metadata.NewIncomingContext(ctx, md)
			vmdb := &mockPrometheusService{}
			vmdb.Test(t)

			serviceAccountID := int(0)
			nodeName := getTestNodeName()
			reregister := false
			force := true

			state := &mockAgentsStateUpdater{}
			state.Test(t)

			ar := &mockAgentsRegistry{}
			ar.Test(t)

			cc := &mockConnectionChecker{}
			cc.Test(t)

			sib := &mockServiceInfoBroker{}
			sib.Test(t)

			vc := &mockVersionCache{}
			vc.Test(t)

			authProvider := &mockGrafanaClient{}
			authProvider.Test(t)
			authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
			authProvider.On("DeleteServiceAccount", ctx, nodeName, force).Return("", nil)

			s := NewManagementService(db, ar, state, cc, sib, vmdb, vc, authProvider)

			return ctx, s, teardown
		}

		t.Run("New", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			t.Cleanup(func() { teardown(t) })

			res, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
				NodeType: inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
				NodeName: "test-node",
				Address:  "some.address.org",
				Region:   "region",
			})
			expected := &managementv1.RegisterNodeResponse{
				GenericNode: &inventoryv1.GenericNode{
					NodeId:   "/node_id/00000000-0000-4000-8000-000000000005",
					NodeName: "test-node",
					Address:  "some.address.org",
					Region:   "region",
				},
				ContainerNode: (*inventoryv1.ContainerNode)(nil),
				PmmAgent: &inventoryv1.PMMAgent{
					AgentId:      "/agent_id/00000000-0000-4000-8000-000000000006",
					RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000005",
				},
				Token: "test-token",
			}
			assert.Equal(t, expected, res)
			assert.NoError(t, err)

			t.Run("Exist", func(t *testing.T) {
				res, err = s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
					NodeType: inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
					NodeName: "test-node",
				})
				assert.Nil(t, res)
				tests.AssertGRPCError(t, status.New(codes.AlreadyExists, `Node with name "test-node" already exists.`), err)
			})

			t.Run("Reregister", func(t *testing.T) {
				serviceAccountID := int(0)
				nodeName := getTestNodeName()
				reregister := true

				authProvider := &mockGrafanaClient{}
				authProvider.Test(t)
				authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
				s.grafanaClient = authProvider

				res, err = s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
					NodeType:   inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
					NodeName:   "test-node",
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				expected := &managementv1.RegisterNodeResponse{
					GenericNode: &inventoryv1.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-000000000008",
						NodeName: "test-node",
						Address:  "some.address.org",
						Region:   "region",
					},
					ContainerNode: (*inventoryv1.ContainerNode)(nil),
					PmmAgent: &inventoryv1.PMMAgent{
						AgentId:      "/agent_id/00000000-0000-4000-8000-000000000009",
						RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000008",
					},
					Token: "test-token",
				}
				assert.Equal(t, expected, res)
				assert.NoError(t, err)
			})

			t.Run("Reregister-force", func(t *testing.T) {
				serviceAccountID := int(0)
				nodeName := "test-node-new"
				reregister := true

				authProvider := &mockGrafanaClient{}
				authProvider.Test(t)
				authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
				s.grafanaClient = authProvider

				res, err = s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
					NodeType:   inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
					NodeName:   "test-node-new",
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				expected := &managementv1.RegisterNodeResponse{
					GenericNode: &inventoryv1.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-00000000000b",
						NodeName: "test-node-new",
						Address:  "some.address.org",
						Region:   "region",
					},
					ContainerNode: (*inventoryv1.ContainerNode)(nil),
					PmmAgent: &inventoryv1.PMMAgent{
						AgentId:      "/agent_id/00000000-0000-4000-8000-00000000000c",
						RunsOnNodeId: "/node_id/00000000-0000-4000-8000-00000000000b",
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

				authProvider := &mockGrafanaClient{}
				authProvider.Test(t)
				authProvider.On("CreateServiceAccount", ctx, nodeName, reregister).Return(serviceAccountID, "test-token", nil)
				authProvider.On("DeleteServiceAccount", ctx, nodeName, force).Return("", nil)
				s.grafanaClient = authProvider

				resRegister, err := s.RegisterNode(ctx, &managementv1.RegisterNodeRequest{
					NodeType:   inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE,
					NodeName:   "test-node",
					Address:    "some.address.org",
					Region:     "region",
					Reregister: true,
				})
				assert.NoError(t, err)

				expected := &managementv1.RegisterNodeResponse{
					GenericNode: &inventoryv1.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-00000000000e",
						NodeName: "test-node",
						Address:  "some.address.org",
						Region:   "region",
					},
					ContainerNode: (*inventoryv1.ContainerNode)(nil),
					PmmAgent: &inventoryv1.PMMAgent{
						AgentId:      "/agent_id/00000000-0000-4000-8000-00000000000f",
						RunsOnNodeId: "/node_id/00000000-0000-4000-8000-00000000000e",
					},
					Token: "test-token",
				}
				assert.Equal(t, expected, resRegister)

				res, err := s.Unregister(ctx, &managementv1.UnregisterNodeRequest{
					NodeId: resRegister.GenericNode.NodeId,
					Force:  true,
				})
				assert.NoError(t, err)
				assert.Equal(t, "", res.Warning)
			})
		})
	})
}
