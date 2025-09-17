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

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	rolev1beta1 "github.com/percona/pmm/api/accesscontrol/v1beta1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/utils/logger"
)

//nolint:paralleltest
func TestAccessControlService(t *testing.T) {
	ctx := logger.Set(context.Background(), t.Name())
	uuid.SetRand(&tests.IDReader{})

	sqlDB := testdb.Open(t, models.SetupFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	defer func(t *testing.T) {
		t.Helper()

		uuid.SetRand(nil)

		require.NoError(t, sqlDB.Close())
	}(t)

	s := NewAccessControlService(db)
	teardown := func(t *testing.T) {
		t.Helper()

		_, err := db.Querier.DeleteFrom(models.RoleTable, "")
		require.NoError(t, err)
		_, err = db.Querier.DeleteFrom(models.UserDetailsTable, "")
		require.NoError(t, err)
	}

	//nolint:paralleltest
	t.Run("Create role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			defer teardown(t)

			res, err := s.CreateRole(ctx, &rolev1beta1.CreateRoleRequest{
				Title:  "Role A",
				Filter: "filter",
			})
			require.NoError(t, err)
			assert.Positive(t, res.RoleId)
		})
	})

	//nolint:paralleltest
	t.Run("Update role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			defer teardown(t)

			_, roleID := createDummyRoles(ctx, t, s)

			_, err := s.UpdateRole(ctx, &rolev1beta1.UpdateRoleRequest{
				RoleId:      roleID,
				Title:       pointer.ToString("Role B - updated"),
				Filter:      pointer.ToString(""), // Filter was reset.
				Description: nil,                  // Description is not updated.
			})
			require.NoError(t, err)

			roles, err := s.ListRoles(ctx, &rolev1beta1.ListRolesRequest{})
			require.NoError(t, err)
			assert.Equal(t, "Role A", roles.Roles[0].Title)
			assert.Equal(t, "filter A", roles.Roles[0].Filter)
			assert.Equal(t, "Role A description", roles.Roles[0].Description)
			assert.Equal(t, "Role B - updated", roles.Roles[1].Title)
			assert.Empty(t, roles.Roles[1].Filter)
			assert.Equal(t, "Role B description", roles.Roles[1].Description)
		})

		t.Run("Shall return not found", func(t *testing.T) {
			defer teardown(t)

			createDummyRoles(ctx, t, s)

			_, err := s.UpdateRole(ctx, &rolev1beta1.UpdateRoleRequest{
				RoleId: 0,
			})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	//nolint:paralleltest
	t.Run("Delete role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			defer teardown(t)

			_, roleID := createDummyRoles(ctx, t, s)

			_, err := s.DeleteRole(ctx, &rolev1beta1.DeleteRoleRequest{RoleId: roleID})
			require.NoError(t, err)

			roles, err := s.ListRoles(ctx, &rolev1beta1.ListRolesRequest{})
			require.NoError(t, err)
			assert.Len(t, roles.Roles, 1)
			assert.Equal(t, roles.Roles[0].Title, "Role A")
		})

		t.Run("Shall return not found", func(t *testing.T) {
			defer teardown(t)

			createDummyRoles(ctx, t, s)

			_, err := s.DeleteRole(ctx, &rolev1beta1.DeleteRoleRequest{RoleId: 0})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	//nolint:paralleltest
	t.Run("Get role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			defer teardown(t)

			_, roleID := createDummyRoles(ctx, t, s)

			res, err := s.GetRole(ctx, &rolev1beta1.GetRoleRequest{RoleId: roleID})
			require.NoError(t, err)
			assert.Equal(t, res.Title, "Role B")
		})

		t.Run("Shall return not found", func(t *testing.T) {
			defer teardown(t)

			createDummyRoles(ctx, t, s)

			_, err := s.GetRole(ctx, &rolev1beta1.GetRoleRequest{RoleId: 0})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	//nolint:paralleltest
	t.Run("List roles", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			defer teardown(t)

			createDummyRoles(ctx, t, s)

			res, err := s.ListRoles(ctx, &rolev1beta1.ListRolesRequest{})
			require.NoError(t, err)
			assert.Len(t, res.Roles, 2)
		})
	})

	//nolint:paralleltest
	t.Run("Assign role", func(t *testing.T) {
		t.Run("Shall assign role to the correct user", func(t *testing.T) {
			defer teardown(t)

			roleIDA, roleIDB := createDummyRoles(ctx, t, s)

			_, err := s.AssignRoles(ctx, &rolev1beta1.AssignRolesRequest{
				RoleIds: []uint32{roleIDA},
				UserId:  1337,
			})
			require.NoError(t, err)

			_, err = s.AssignRoles(ctx, &rolev1beta1.AssignRolesRequest{
				RoleIds: []uint32{roleIDB},
				UserId:  1338,
			})
			require.NoError(t, err)

			roles, err := models.GetUserRoles(db.Querier, 1337)
			require.NoError(t, err)
			assert.Len(t, roles, 1)
			assert.Equal(t, roles[0].ID, roleIDA)
		})

		t.Run("Shall assign multiple roles", func(t *testing.T) {
			defer teardown(t)

			roleIDA, roleIDB := createDummyRoles(ctx, t, s)

			_, err := s.AssignRoles(ctx, &rolev1beta1.AssignRolesRequest{
				RoleIds: []uint32{roleIDA, roleIDB},
				UserId:  1337,
			})
			require.NoError(t, err)

			roles, err := models.GetUserRoles(db.Querier, 1337)
			require.NoError(t, err)
			assert.Len(t, roles, 2)
			assert.Equal(t, roles[0].ID, roleIDA)
			assert.Equal(t, roles[1].ID, roleIDB)
		})

		t.Run("Shall return not found for non-existent role", func(t *testing.T) {
			defer teardown(t)

			createDummyRoles(ctx, t, s)

			_, err := s.AssignRoles(ctx, &rolev1beta1.AssignRolesRequest{
				RoleIds: []uint32{0},
				UserId:  1337,
			})
			tests.AssertGRPCErrorCode(t, codes.NotFound, err)
		})
	})

	t.Run("Set default role", func(t *testing.T) {
		t.Run("Shall work", func(t *testing.T) {
			defer teardown(t)
			settings, err := models.GetSettings(db)
			require.NoError(t, err)

			roleID, _ := createDummyRoles(ctx, t, s)
			assert.NotEqual(t, settings.DefaultRoleID, roleID)

			_, err = s.SetDefaultRole(ctx, &rolev1beta1.SetDefaultRoleRequest{
				RoleId: roleID,
			})
			require.NoError(t, err)

			settingsNew, err := models.GetSettings(db)
			require.NoError(t, err)
			assert.Equal(t, settingsNew.DefaultRoleID, int(roleID))
		})

		t.Run("shall return error on non existent role", func(t *testing.T) {
			defer teardown(t)
			_, err := s.SetDefaultRole(ctx, &rolev1beta1.SetDefaultRoleRequest{
				RoleId: 1337,
			})

			assert.Error(t, err)
		})
	})
}

func createDummyRoles(ctx context.Context, t *testing.T, s *AccessControlService) (uint32, uint32) {
	t.Helper()

	rA, err := s.CreateRole(ctx, &rolev1beta1.CreateRoleRequest{
		Title:       "Role A",
		Filter:      "filter A",
		Description: "Role A description",
	})
	require.NoError(t, err)

	rB, err := s.CreateRole(ctx, &rolev1beta1.CreateRoleRequest{
		Title:       "Role B",
		Filter:      "filter B",
		Description: "Role B description",
	})
	require.NoError(t, err)

	return rA.RoleId, rB.RoleId
}
