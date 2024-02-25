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

// Package clean has the old actions results cleaner tests.
package clean

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestCleaner(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	setup := func(t *testing.T) (*reform.DB, *reform.Querier, func(t *testing.T)) {
		t.Helper()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		q := db.Querier
		now, origNowF := models.Now(), models.Now
		models.Now = func() time.Time {
			// fake "old" rows
			return now.Add(-10 * time.Second)
		}

		for _, str := range []reform.Struct{
			&models.ActionResult{
				ID:         "A1",
				PMMAgentID: "A1",
			},
			&models.ActionResult{
				ID:         "A2",
				PMMAgentID: "A1",
			},
			&models.ActionResult{
				ID:         "A3",
				PMMAgentID: "A1",
			},
		} {
			require.NoError(t, q.Insert(str))
		}

		// This row is to have something that won't be deleted
		models.Now = origNowF
		str := &models.ActionResult{
			ID:         "A4",
			PMMAgentID: "A1",
		}
		require.NoError(t, q.Insert(str))

		teardown := func(t *testing.T) {
			t.Helper()
			assert.NoError(t, models.CleanupOldActionResults(db.Querier, models.Now()))
		}
		return db, q, teardown
	}

	t.Run("CheckActionResultByID", func(t *testing.T) {
		db, q, teardown := setup(t)
		defer teardown(t)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		c := New(db)
		go func() {
			c.Run(ctx, 5*time.Second, 5*time.Second) // delete rows older that 5 seconds
		}()
		// give the cleaner the chance to run
		time.Sleep(100 * time.Millisecond)

		_, err := models.FindActionResultByID(q, "A1")
		assert.Error(t, err)

		_, err = models.FindActionResultByID(q, "A2")
		assert.Error(t, err)

		_, err = models.FindActionResultByID(q, "A3")
		assert.Error(t, err)

		a, err := models.FindActionResultByID(q, "A4")
		assert.NoError(t, err)
		assert.NotEmpty(t, a.ID)
	})
}
