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
	"context"
	"os"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	sqlDB := testdb.Open(t, models.SkipFixtures, pointer.ToInt(77))
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	insertVMFile := func(q *reform.Querier) models.File {
		var err error
		want := models.File{Name: "test"}
		want.Content, err = os.ReadFile("../testdata/victoriametrics/promscrape.base.yml")
		require.NoError(t, err)

		actual, err := models.InsertFile(q, models.InsertFileParams{Name: want.Name, Content: want.Content})
		require.NoError(t, err)
		assert.Equal(t, actual, want)
		return actual
	}

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Run("insert", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		insertVMFile(tx.Querier)
	})

	t.Run("change", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		old := insertVMFile(q)
		want := models.File{Name: "new_test"}
		want.Content, err = os.ReadFile("../testdata/supervisord.d/grafana.ini")
		require.NoError(t, err)

		updated, err := models.UpdateFile(context.Background(), tx, models.UpdateFileParams{OldName: old.Name, NewName: want.Name, Content: want.Content})
		require.NoError(t, err)
		assert.Equal(t, want, updated)
	})

	t.Run("find by name", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		want := insertVMFile(q)
		actual, err := models.GetFile(q, want.Name)
		assert.NoError(t, err)
		assert.Equal(t, want, actual)
	})

	t.Run("delete", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		file := insertVMFile(q)
		err = models.DeleteFile(q, file.Name)
		assert.NoError(t, err)

		_, err = models.GetFile(q, file.Name)
		assert.Equal(t, err, models.ErrFileNotFound)
	})
}
