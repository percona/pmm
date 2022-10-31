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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/logger"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestRoleService(t *testing.T) {
	setup := func(t *testing.T) (ctx context.Context, r *RoleService, db *reform.DB, teardown func(t *testing.T)) {
		t.Helper()

		ctx = logger.Set(context.Background(), t.Name())
		uuid.SetRand(&tests.IDReader{})

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		teardown = func(t *testing.T) {
			uuid.SetRand(nil)

			require.NoError(t, sqlDB.Close())
		}
		r = NewRoleService(db)

		return
	}

	t.Run("Create role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			res, err := s.CreateRole(ctx, &managementpb.RoleData{
				Title:  "Role A",
				Filter: "filter",
			})
			assert.NoError(t, err)
			assert.True(t, res.RoleId > 0)
		})
	})

	t.Run("Update role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			roleID, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = s.UpdateRole(ctx, &managementpb.RoleData{
				RoleId: roleID,
				Title:  "Role B - updated",
				Filter: "filter B - updated",
			})
			assert.NoError(t, err)

			roles, err := s.ListRoles(ctx, &managementpb.ListRolesRequest{})
			assert.NoError(t, err)
			assert.Equal(t, roles.Roles[0].Title, "Role A")
			assert.Equal(t, roles.Roles[1].Title, "Role B - updated")
		})

		t.Run("Shall return not found", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			_, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = s.UpdateRole(ctx, &managementpb.RoleData{
				RoleId: 0,
				Title:  "",
				Filter: "",
			})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	t.Run("Delete role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			roleID, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = s.DeleteRole(ctx, &managementpb.RoleID{RoleId: roleID})
			assert.NoError(t, err)

			roles, err := s.ListRoles(ctx, &managementpb.ListRolesRequest{})
			assert.NoError(t, err)
			assert.Equal(t, len(roles.Roles), 1)
			assert.Equal(t, roles.Roles[0].Title, "Role A")
		})

		t.Run("Shall return not found", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			_, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = s.DeleteRole(ctx, &managementpb.RoleID{RoleId: 0})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	t.Run("Get role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			roleID, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			res, err := s.GetRole(ctx, &managementpb.RoleID{RoleId: roleID})
			assert.NoError(t, err)
			assert.Equal(t, res.Title, "Role B")
		})

		t.Run("Shall return not found", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			_, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = s.GetRole(ctx, &managementpb.RoleID{RoleId: 0})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	t.Run("List roles", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			ctx, s, _, teardown := setup(t)
			defer teardown(t)

			_, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			res, err := s.ListRoles(ctx, &managementpb.ListRolesRequest{})
			assert.NoError(t, err)
			assert.Equal(t, len(res.Roles), 2)
		})
	})

	t.Run("Assign role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			ctx, s, db, teardown := setup(t)
			defer teardown(t)

			roleID, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = models.GetOrCreateUser(db.Querier, 1337)
			assert.NoError(t, err)

			user, err := models.GetOrCreateUser(db.Querier, 1338)
			assert.NoError(t, err)

			_, err = s.AssignRole(ctx, &managementpb.AssignRoleRequest{
				RoleId: roleID,
				UserId: uint32(user.ID),
			})
			assert.NoError(t, err)

			user, err = models.GetOrCreateUser(db.Querier, user.ID)
			assert.NoError(t, err)
			assert.Equal(t, user.RoleID, roleID)
		})

		t.Run("Shall create new user", func(t *testing.T) {
			ctx, s, db, teardown := setup(t)
			defer teardown(t)

			roleID, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			_, err = s.AssignRole(ctx, &managementpb.AssignRoleRequest{
				RoleId: roleID,
				UserId: 1337,
			})
			assert.NoError(t, err)

			_, err = models.FindUser(db.Querier, 1337)
			assert.NoError(t, err)
		})

		t.Run("Shall return not found for role", func(t *testing.T) {
			ctx, s, db, teardown := setup(t)
			defer teardown(t)

			_, err := createDummyRoles(t, ctx, s)
			assert.NoError(t, err)

			user, err := models.GetOrCreateUser(db.Querier, 1337)
			assert.NoError(t, err)

			_, err = s.AssignRole(ctx, &managementpb.AssignRoleRequest{
				RoleId: 0,
				UserId: uint32(user.ID),
			})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})
}

func createDummyRoles(t *testing.T, ctx context.Context, s *RoleService) (uint32, error) {
	_, err := s.CreateRole(ctx, &managementpb.RoleData{
		Title:  "Role A",
		Filter: "filter A",
	})
	assert.NoError(t, err)

	res, err := s.CreateRole(ctx, &managementpb.RoleData{
		Title:  "Role B",
		Filter: "filter B",
	})
	assert.NoError(t, err)

	return res.RoleId, nil
}
