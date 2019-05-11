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

package management

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/managementpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/utils/logger"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestNodeService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, s *NodeService, teardown func()) {
		t.Helper()

		uuid.SetRand(new(tests.IDReader))

		ctx = logger.Set(context.Background(), t.Name())

		sqlDB := testdb.Open(t)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		r := new(mockRegistry)
		r.Test(t)
		s = NewNodeService(db, r)

		teardown = func() {
			require.NoError(t, sqlDB.Close())
			r.AssertExpectations(t)
		}

		return
	}

	t.Run("Register", func(t *testing.T) {
		t.Run("New", func(t *testing.T) {
			ctx, s, teardown := setup(t)
			defer teardown()

			s.registry.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000002").Return(false)
			res, err := s.Register(ctx, &managementpb.RegisterNodeRequest{
				NodeType: inventorypb.NodeType_GENERIC_NODE,
				NodeName: "node",
			})
			expected := &managementpb.RegisterNodeResponse{
				GenericNode: &inventorypb.GenericNode{
					NodeId:   "/node_id/00000000-0000-4000-8000-000000000001",
					NodeName: "node",
				},
				PmmAgent: &inventorypb.PMMAgent{
					AgentId:      "/agent_id/00000000-0000-4000-8000-000000000002",
					RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000001",
				},
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
				s.registry.(*mockRegistry).On("IsConnected", "/agent_id/00000000-0000-4000-8000-000000000005").Return(false)
				res, err = s.Register(ctx, &managementpb.RegisterNodeRequest{
					NodeType:   inventorypb.NodeType_GENERIC_NODE,
					NodeName:   "node",
					Reregister: true,
				})
				expected := &managementpb.RegisterNodeResponse{
					GenericNode: &inventorypb.GenericNode{
						NodeId:   "/node_id/00000000-0000-4000-8000-000000000004",
						NodeName: "node",
					},
					PmmAgent: &inventorypb.PMMAgent{
						AgentId:      "/agent_id/00000000-0000-4000-8000-000000000005",
						RunsOnNodeId: "/node_id/00000000-0000-4000-8000-000000000004",
					},
				}
				assert.Equal(t, expected, res)
				assert.NoError(t, err)
			})
		})
	})
}
