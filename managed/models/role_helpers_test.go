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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	roleATitle = "Role A"
	roleBTitle = "Role B"
)

//nolint:paralleltest
func TestRoleHelpers(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	userID := 1337
	setup := func(t *testing.T) (*reform.TX, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)
		q := tx.Querier

		for _, str := range []reform.Struct{
			&models.UserDetails{ //nolint:exhaustruct
				ID: userID,
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return tx, teardown
	}

	//nolint:paralleltest
	t.Run("shall assign role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.AssignRoles(tx, userID, []int{int(role.ID)}))

		roles, err := models.GetUserRoles(tx.Querier, userID)
		require.NoError(t, err)
		require.Equal(t, roles[0].ID, role.ID)
	})

	//nolint:paralleltest
	t.Run("shall throw on assigning non-existent role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		err := models.AssignRoles(tx, userID, []int{0})
		require.True(t, errors.Is(err, models.ErrRoleNotFound))
	})

	//nolint:paralleltest
	t.Run("shall create a new user on role assign", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.AssignRoles(tx, 24, []int{int(role.ID)}))

		_, err := models.FindUser(tx.Querier, userID)
		require.NoError(t, err)
	})

	//nolint:paralleltest
	t.Run("unassigned role", func(t *testing.T) {
		//nolint:paralleltest
		t.Run("shall delete role with no replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var role models.Role
			role.Title = roleATitle
			require.NoError(t, models.CreateRole(tx.Querier, &role))
			require.NoError(t, models.DeleteRole(tx, int(role.ID), 0))
		})

		//nolint:paralleltest
		t.Run("shall delete role with replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var roleA, roleB models.Role
			roleA.Title = roleATitle
			roleB.Title = roleBTitle
			require.NoError(t, models.CreateRole(tx.Querier, &roleA))
			require.NoError(t, models.CreateRole(tx.Querier, &roleB))
			require.NoError(t, models.DeleteRole(tx, int(roleA.ID), int(roleB.ID)))
		})
	})

	//nolint:paralleltest
	t.Run("single role assigned", func(t *testing.T) {
		//nolint:paralleltest
		t.Run("shall delete role with no replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var role models.Role
			role.Title = roleATitle
			require.NoError(t, models.CreateRole(tx.Querier, &role))
			require.NoError(t, models.AssignRoles(tx, userID, []int{int(role.ID)}))
			require.NoError(t, models.DeleteRole(tx, int(role.ID), 0))

			roles, err := models.GetUserRoles(tx.Querier, userID)
			require.NoError(t, err)
			require.Empty(t, roles)
		})

		//nolint:paralleltest
		t.Run("shall delete role with replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var roleA, roleB models.Role
			roleA.Title = roleATitle
			roleB.Title = roleBTitle
			require.NoError(t, models.CreateRole(tx.Querier, &roleA))
			require.NoError(t, models.CreateRole(tx.Querier, &roleB))
			require.NoError(t, models.AssignRoles(tx, userID, []int{int(roleA.ID)}))
			require.NoError(t, models.DeleteRole(tx, int(roleA.ID), int(roleB.ID)))

			roles, err := models.GetUserRoles(tx.Querier, userID)
			require.NoError(t, err)
			require.Len(t, roles, 1)
			require.Equal(t, roles[0].ID, roleB.ID)
		})
	})

	//nolint:paralleltest
	t.Run("multiple roles assigned", func(t *testing.T) {
		//nolint:paralleltest
		t.Run("shall delete role with no replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var roleA, roleB models.Role
			roleA.Title = roleATitle
			roleB.Title = roleBTitle
			require.NoError(t, models.CreateRole(tx.Querier, &roleA))
			require.NoError(t, models.CreateRole(tx.Querier, &roleB))
			require.NoError(t, models.AssignRoles(tx, userID, []int{int(roleA.ID), int(roleB.ID)}))
			require.NoError(t, models.DeleteRole(tx, int(roleA.ID), 0))

			roles, err := models.GetUserRoles(tx.Querier, userID)
			require.NoError(t, err)
			require.Len(t, roles, 1)
			require.Equal(t, roles[0].ID, roleB.ID)
		})

		//nolint:paralleltest
		t.Run("shall delete role with replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var roleA, roleB models.Role
			roleA.Title = roleATitle
			roleB.Title = roleBTitle
			require.NoError(t, models.CreateRole(tx.Querier, &roleA))
			require.NoError(t, models.CreateRole(tx.Querier, &roleB))
			require.NoError(t, models.AssignRoles(tx, userID, []int{int(roleA.ID), int(roleB.ID)}))
			require.NoError(t, models.DeleteRole(tx, int(roleA.ID), int(roleB.ID)))

			roles, err := models.GetUserRoles(tx.Querier, userID)
			require.NoError(t, err)
			require.Len(t, roles, 1)
			require.Equal(t, roles[0].ID, roleB.ID)
		})
	})

	//nolint:paralleltest
	t.Run("shall not delete default role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.AssignRoles(tx, 24, []int{int(role.ID)}))
		require.NoError(t, models.ChangeDefaultRole(tx, int(role.ID)))

		err := models.DeleteRole(tx, int(role.ID), 0)
		require.Equal(t, err, models.ErrRoleIsDefaultRole)
	})

	//nolint:paralleltest
	t.Run("shall return roles for a user", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var roleA, roleB models.Role
		roleA.Title = roleATitle
		roleB.Title = roleBTitle

		require.NoError(t, models.CreateRole(tx.Querier, &roleA))
		require.NoError(t, models.CreateRole(tx.Querier, &roleB))
		require.NoError(t, models.AssignRoles(tx, userID, []int{int(roleA.ID), int(roleB.ID)}))

		roles, err := models.GetUserRoles(tx.Querier, userID)
		require.NoError(t, err)
		require.Len(t, roles, 2)
		require.Equal(t, roles[0].ID, roleA.ID)
		require.Equal(t, roles[1].ID, roleB.ID)
	})
}
