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

	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

type noOpBeforeDelete struct{}

func (n *noOpBeforeDelete) BeforeDeleteRole(tx *reform.TX, roleID, newRoleID int) error {
	return nil
}

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
	t.Run("shall create role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))

		var r models.Role
		require.NoError(t, models.FindAndLockRole(tx, int(role.ID), &r))

		require.Equal(t, role.ID, r.ID)
	})

	//nolint:paralleltest
	t.Run("shall delete role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.DeleteRole(tx, &noOpBeforeDelete{}, int(role.ID), 0))

		var r models.Role
		require.NoError(t, models.FindAndLockRole(tx, int(role.ID), &r))
		require.Equal(t, 0, r.ID)
	})

	//nolint:paralleltest
	t.Run("shall not delete default role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.ChangeDefaultRole(tx, int(role.ID)))

		err := models.DeleteRole(tx, &noOpBeforeDelete{}, int(role.ID), 0)
		require.Equal(t, err, models.ErrRoleIsDefaultRole)
	})

	t.Run("shall change default role", func(t *testing.T) {
		tx, teardown := setup(t)
		defer teardown(t)

		var role models.Role
		require.NoError(t, models.CreateRole(tx.Querier, &role))
		require.NoError(t, models.ChangeDefaultRole(tx, int(role.ID)))

		s, err := models.GetSettings(tx.Querier)
		require.NoError(t, err)
		require.Equal(t, s.DefaultRoleID, int(role.ID))
	})
}
