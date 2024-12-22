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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/database"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestChecksSettings(t *testing.T) { //nolint:tparallel
	t.Parallel()
	sqlDB := testdb.Open(t, database.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("create", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier

		actual, err := models.CreateCheckSettings(q, "check-name", models.Standard)
		require.NoError(t, err)
		assert.Equal(t, "check-name", actual.Name)
		assert.Equal(t, models.Standard, actual.Interval)
	})

	t.Run("change", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier

		oldState, err := models.CreateCheckSettings(q, "check-name", models.Standard)
		require.NoError(t, err)
		assert.Equal(t, "check-name", oldState.Name)
		assert.Equal(t, models.Standard, oldState.Interval)

		newState, err := models.ChangeCheckSettings(q, "check-name", models.Rare)
		require.NoError(t, err)
		assert.Equal(t, oldState.Name, newState.Name)
		assert.NotEqual(t, oldState.Interval, newState.Interval)
		assert.Equal(t, models.Rare, newState.Interval)
	})

	t.Run("find by name", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier

		expected, err := models.CreateCheckSettings(q, "check-name", models.Standard)
		require.NoError(t, err)

		actual, err := models.FindCheckSettingsByName(q, "check-name")
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("find all", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier

		_, err = models.CreateCheckSettings(q, "check1", models.Standard)
		require.NoError(t, err)
		_, err = models.CreateCheckSettings(q, "check2", models.Standard)
		require.NoError(t, err)

		actual, err := models.FindCheckSettings(q)
		require.NoError(t, err)
		assert.Len(t, actual, 2)
		assert.Equal(t, actual["check1"], models.Standard)
		assert.Equal(t, actual["check2"], models.Standard)
	})
}
