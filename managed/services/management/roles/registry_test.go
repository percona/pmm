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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRegistry(t *testing.T) {
	setup := func(t *testing.T) (*Registry, *MockEntityService, *MockEntityService) {
		user := &MockEntityService{}
		t.Cleanup(func() { user.AssertExpectations(t) })

		team := &MockEntityService{}
		t.Cleanup(func() { team.AssertExpectations(t) })

		r := NewRegistry(map[EntityType]EntityService{
			EntityUser: user,
			EntityTeam: team,
		})

		return r, user, team
	}

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	setupDB := func(t *testing.T) (*reform.TX, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		tx, err := db.Begin()
		require.NoError(t, err)

		teardown := func(t *testing.T) {
			t.Helper()
			require.NoError(t, tx.Rollback())
		}
		return tx, teardown
	}

	t.Run("AssignRoles", func(t *testing.T) {
		t.Parallel()

		t.Run("shall assign roles", func(t *testing.T) {
			t.Parallel()

			r, user, _ := setup(t)
			user.Mock.On("AssignRoles", mock.Anything, 5, []int{10}).Return(nil)

			require.NoError(t, r.AssignRoles(nil, EntityUser, 5, []int{10}))
		})

		t.Run("shall return error on non existent service assign roles", func(t *testing.T) {
			t.Parallel()

			r := NewRegistry(map[EntityType]EntityService{})
			require.Error(t, r.AssignRoles(nil, EntityUser, 5, []int{10}))
		})
	})

	//nolint:paralleltest
	t.Run("AssignDefaultRole", func(t *testing.T) {
		//nolint:paralleltest
		t.Run("shall assign default role to a user", func(t *testing.T) {
			tx, teardown := setupDB(t)
			defer teardown(t)

			var roleA, roleB models.Role
			roleA.Title = roleATitle
			roleB.Title = roleBTitle
			require.NoError(t, models.CreateRole(tx.Querier, &roleA))
			require.NoError(t, models.CreateRole(tx.Querier, &roleB))

			require.NoError(t, models.ChangeDefaultRole(tx, int(roleB.ID)))

			r, user, _ := setup(t)
			user.Mock.On("AssignRoles", mock.Anything, 5, []int{int(roleB.ID)}).Return(nil)

			require.NoError(t, r.AssignDefaultRole(tx, 5))
		})
	})

	t.Run("BeforeDeleteRole", func(t *testing.T) {
		t.Parallel()

		t.Run("shall be called for every service", func(t *testing.T) {
			r, user, team := setup(t)
			user.Mock.On("BeforeDeleteRole", mock.Anything, 5, 10).Return(nil)
			team.Mock.On("BeforeDeleteRole", mock.Anything, 5, 10).Return(nil)

			require.NoError(t, r.BeforeDeleteRole(nil, 5, 10))
		})
	})

	t.Run("GetUserRoles", func(t *testing.T) {
		t.Parallel()

		t.Run("shall be retrieved from user service only", func(t *testing.T) {
			t.Parallel()

			r, user, _ := setup(t)
			user.Mock.On("GetEntityRoles", mock.Anything, []int{5}, mock.Anything).Return([]models.Role{{}}, nil)

			roles, err := r.GetUserRoles(nil, 5, nil)
			require.NoError(t, err)
			require.Equal(t, 1, len(roles))
		})

		t.Run("shall be retrieved from user and team service", func(t *testing.T) {
			t.Parallel()

			r, user, team := setup(t)
			user.Mock.On("GetEntityRoles", mock.Anything, []int{5}, mock.Anything).Return([]models.Role{{ID: 1}}, nil)
			team.Mock.On("GetEntityRoles", mock.Anything, []int{7, 9}, mock.Anything).Return([]models.Role{{ID: 2}, {ID: 3}}, nil)

			roles, err := r.GetUserRoles(nil, 5, []int{7, 9})
			require.NoError(t, err)
			require.Equal(t, 3, len(roles))
		})

		t.Run("shall return distinct roles based on ID", func(t *testing.T) {
			t.Parallel()

			r, user, team := setup(t)
			user.Mock.On("GetEntityRoles", mock.Anything, []int{5}, mock.Anything).Return([]models.Role{{ID: 1}}, nil)
			team.Mock.On("GetEntityRoles", mock.Anything, []int{7, 9}, mock.Anything).Return([]models.Role{{ID: 2}, {ID: 2}}, nil)

			roles, err := r.GetUserRoles(nil, 5, []int{7, 9})
			require.NoError(t, err)
			require.Equal(t, 2, len(roles))
		})
	})
}
