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

package roles

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestUser(t *testing.T) {
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

	u := NewUser("user_id", func() EntityModel {
		return &models.UserRoles{}
	}, models.UserRolesView)

	//nolint:paralleltest
	t.Run("unassigned role", func(t *testing.T) {
		//nolint:paralleltest
		t.Run("shall delete role with no replacement", func(t *testing.T) {
			tx, teardown := setup(t)
			defer teardown(t)

			var role models.Role
			role.Title = roleATitle
			require.NoError(t, models.CreateRole(tx.Querier, &role))
			require.NoError(t, u.BeforeDeleteRole(tx, int(role.ID), 0))
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
			require.NoError(t, u.BeforeDeleteRole(tx, int(roleA.ID), int(roleB.ID)))
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
			require.NoError(t, u.AssignRoles(tx, userID, []int{int(role.ID)}))
			require.NoError(t, u.BeforeDeleteRole(tx, int(role.ID), 0))

			roles, err := u.GetEntityRoles(tx.Querier, []int{userID})
			require.NoError(t, err)
			require.Equal(t, len(roles), 0)
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
			require.NoError(t, u.AssignRoles(tx, userID, []int{int(roleA.ID)}))
			require.NoError(t, u.BeforeDeleteRole(tx, int(roleA.ID), int(roleB.ID)))

			roles, err := u.GetEntityRoles(tx.Querier, []int{userID})
			require.NoError(t, err)
			require.Equal(t, 1, len(roles))
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
			require.NoError(t, u.AssignRoles(tx, userID, []int{int(roleA.ID), int(roleB.ID)}))
			require.NoError(t, u.BeforeDeleteRole(tx, int(roleA.ID), 0))

			roles, err := u.GetEntityRoles(tx.Querier, []int{userID})
			require.NoError(t, err)
			require.Equal(t, len(roles), 1)
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
			require.NoError(t, u.AssignRoles(tx, userID, []int{int(roleA.ID), int(roleB.ID)}))
			require.NoError(t, models.DeleteRole(tx, &noOpBeforeDelete{}, int(roleA.ID), int(roleB.ID)))

			roles, err := u.GetEntityRoles(tx.Querier, []int{userID})
			require.NoError(t, err)
			require.Equal(t, len(roles), 1)
			require.Equal(t, roles[0].ID, roleB.ID)
		})
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
		require.NoError(t, u.AssignRoles(tx, userID, []int{int(roleA.ID), int(roleB.ID)}))

		roles, err := u.GetEntityRoles(tx.Querier, []int{userID})
		require.NoError(t, err)
		require.Equal(t, len(roles), 2)
		require.Equal(t, roles[0].ID, roleA.ID)
		require.Equal(t, roles[1].ID, roleB.ID)
	})
}
