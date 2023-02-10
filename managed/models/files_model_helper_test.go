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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestFile(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	insertVMFile := func(q *reform.Querier) models.File {
		var err error
		want := models.File{Name: "test", UpdatedAt: models.Now()}
		want.Content, err = os.ReadFile("../testdata/victoriametrics/promscrape.base.yml")
		require.NoError(t, err)

		fp := models.InsertFileParams{Name: want.Name, Content: want.Content}
		err = fp.Validate()
		require.NoError(t, err)

		actual, err := models.InsertFile(q, fp)
		require.NoError(t, err)
		assert.Equal(t, want.Name, actual.Name)
		assert.Equal(t, want.Content, actual.Content)
		assert.WithinDuration(t, want.UpdatedAt, actual.UpdatedAt, 10*time.Second)
		return actual
	}

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	t.Run("insert", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		old := insertVMFile(tx.Querier)
		duplicate, err := models.InsertFile(tx.Querier, models.InsertFileParams{Name: old.Name, Content: []byte("test")})
		require.NoError(t, err)
		assert.Equal(t, old, duplicate)
	})

	t.Run("upsert", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		old := insertVMFile(tx.Querier)
		want := models.File{Name: old.Name, UpdatedAt: models.Now()}
		want.Content, err = os.ReadFile("../testdata/supervisord.d/grafana.ini")
		require.NoError(t, err)

		fp := models.InsertFileParams{Name: old.Name, Content: want.Content}
		err = fp.Validate()
		require.NoError(t, err)

		updated, err := models.UpsertFile(tx.Querier, fp)
		require.NoError(t, err)
		assert.Equal(t, want.Name, updated.Name)
		assert.Equal(t, want.Content, updated.Content)
		assert.WithinDuration(t, want.UpdatedAt, updated.UpdatedAt, 10*time.Second)
	})

	t.Run("get and insert", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		want, err := models.GetOrInsertFile(tx.Querier, "../testdata/victoriametrics/promscrape.base.yml")
		require.NoError(t, err)
		actual, err := models.GetOrInsertFile(tx.Querier, "../non-existent-path/promscrape.base.yml")
		require.NoError(t, err)
		assert.Equal(t, want, actual)
	})

	t.Run("update", func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tx.Rollback())
		})

		q := tx.Querier
		old := insertVMFile(q)
		want := models.File{Name: old.Name, UpdatedAt: models.Now()}
		want.Content, err = os.ReadFile("../testdata/supervisord.d/grafana.ini")
		require.NoError(t, err)

		fp := models.UpdateFileParams{Name: old.Name, Content: want.Content}
		err = fp.Validate()
		require.NoError(t, err)

		updated, err := models.UpdateFile(tx.Context(), tx, fp)
		require.NoError(t, err)
		assert.Equal(t, want.Name, updated.Name)
		assert.Equal(t, want.Content, updated.Content)
		assert.WithinDuration(t, want.UpdatedAt, updated.UpdatedAt, 10*time.Second)
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
		assert.Equal(t, want.Name, actual.Name)
		assert.Equal(t, want.Content, actual.Content)
		assert.WithinDuration(t, want.UpdatedAt, actual.UpdatedAt, 10*time.Second)
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
		assert.Equal(t, err, models.ErrNotFound)
	})
}
