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
		role.Title = "Role A"
		require.NoError(t, models.CreateRole(tx.Querier, &role))

		require.NoError(t, models.AssignRole(tx, userID, int(role.ID)))

		user, err := models.FindUser(tx.Querier, userID)
		require.NoError(t, err)
		require.Equal(t, user.RoleID, role.ID)
	})

	//nolint:paralleltest
	t.Run("shall throw on role not found", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		err := models.AssignRole(tx, userID, 0)
		require.True(t, errors.Is(err, models.ErrRoleNotFound))
	})

	//nolint:paralleltest
	t.Run("shall create new user", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		role.Title = "Role A"
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.AssignRole(tx, 24, int(role.ID)))

		_, err := models.FindUser(tx.Querier, userID)
		require.NoError(t, err)
	})
}
