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

func TestCheckResultsCleaner(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	q := db.Querier

	// Default retention is 30 days, so this row is past it and must be removed.
	require.NoError(t, models.CreateCheckResult(q, &models.CheckResult{
		CheckName: "old", ServiceID: "svc", ServiceName: "svc", NodeName: "node",
		Status: models.CheckResultFailed, Summary: "s", CheckedAt: models.Now().Add(-31 * 24 * time.Hour),
	}))
	require.NoError(t, models.CreateCheckResult(q, &models.CheckResult{
		CheckName: "new", ServiceID: "svc", ServiceName: "svc", NodeName: "node",
		Status: models.CheckResultFailed, Summary: "s", CheckedAt: models.Now(),
	}))

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
	defer cancel()

	go NewCheckResults(db).Run(ctx, 5*time.Second)
	// Give the cleaner the chance to run one iteration.
	time.Sleep(100 * time.Millisecond)

	results, err := models.FindCheckResults(q, models.CheckResultFilters{ServiceID: "svc"}, 0, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "new", results[0].CheckName)
}
