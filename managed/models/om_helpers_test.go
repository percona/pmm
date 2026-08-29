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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestOmTopologyRuns(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	base := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)

	// insert records one run and its document, minutes apart so ordering is unambiguous.
	insert := func(t *testing.T, q *reform.Querier, id string, offset time.Duration, document string) {
		t.Helper()
		started := base.Add(offset)
		finished := started.Add(time.Second)
		observed := started.Add(-5 * time.Second)

		run := &models.OmTopologyRun{
			RunID: id, StartedAt: started, FinishedAt: &finished,
			Status: models.OmTopologyRunSuccess, ServicesTotal: 14, ServicesResolved: 14,
			ProbesOK: 13, ServicesStale: 1, OriginNode: "pmm-server",
			Sources: models.OmTopologySourceReports{
				{Source: "inventory", Status: "ok", Facts: 81},
				{Source: "metrics", Status: "ok", Facts: 219, Detail: map[string]string{"queries": "13"}},
			},
			Errors: models.OmTopologyRunErrors{
				{Scope: "query", Code: "vm_query_failed", Message: "boom"},
			},
		}
		snapshot := &models.OmTopologySnapshot{
			GeneratedAt: finished, ObservedAt: &observed,
			SchemaVersion: 3, Document: []byte(document),
		}
		require.NoError(t, models.CreateOmTopologyRun(q, run, snapshot))
	}

	t.Run("a run round-trips with its receipt and its document", func(t *testing.T) {
		q := db.Querier
		insert(t, q, "run-1", 0, `{"origin_node":"pmm-server"}`)

		run, err := models.FindOmTopologyRunByID(q, "run-1")
		require.NoError(t, err)
		assert.Equal(t, models.OmTopologyRunSuccess, run.Status)
		assert.Equal(t, int32(14), run.ServicesTotal)
		assert.Equal(t, int32(1), run.ServicesStale)
		assert.Equal(t, "pmm-server", run.OriginNode)

		// The per-source receipt is the part that makes a thin document legible.
		require.Len(t, run.Sources, 2)
		assert.Equal(t, "metrics", run.Sources[1].Source)
		assert.Equal(t, "13", run.Sources[1].Detail["queries"])
		require.Len(t, run.Errors, 1)
		assert.Equal(t, "vm_query_failed", run.Errors[0].Code)

		snapshot, err := models.FindLatestOmTopologySnapshot(q)
		require.NoError(t, err)
		assert.Equal(t, "run-1", snapshot.RunID)
		assert.JSONEq(t, `{"origin_node":"pmm-server"}`, string(snapshot.Document))
		assert.Equal(t, int32(3), snapshot.SchemaVersion)
	})

	t.Run("runs come back newest first", func(t *testing.T) {
		q := db.Querier
		insert(t, q, "run-2", time.Minute, `{"n":2}`)
		insert(t, q, "run-3", 2*time.Minute, `{"n":3}`)

		runs, err := models.FindOmTopologyRuns(q, 2)
		require.NoError(t, err)
		require.Len(t, runs, 2)
		assert.Equal(t, "run-3", runs[0].RunID)
		assert.Equal(t, "run-2", runs[1].RunID)

		// And the newest document is the one a cold start restores.
		snapshot, err := models.FindLatestOmTopologySnapshot(q)
		require.NoError(t, err)
		assert.Equal(t, "run-3", snapshot.RunID)
	})

	t.Run("pruning bounds the history and takes the documents with it", func(t *testing.T) {
		q := db.Querier
		for i := range 5 {
			insert(t, q, fmt.Sprintf("prune-%d", i), time.Duration(10+i)*time.Minute, `{}`)
		}

		require.NoError(t, models.PruneOmTopologyRuns(q, 3))

		runs, err := models.FindOmTopologyRuns(q, 100)
		require.NoError(t, err)
		require.Len(t, runs, 3, "only the newest are kept")
		assert.Equal(t, "prune-4", runs[0].RunID)

		// The snapshot cascades, so retention needs no second sweep to stay bounded.
		_, err = models.FindOmTopologyRunByID(q, "prune-0")
		require.ErrorIs(t, err, models.ErrNotFound)

		var count int
		require.NoError(t, db.QueryRow("SELECT count(*) FROM om_topology_snapshots").Scan(&count))
		assert.Equal(t, 3, count)
	})

	t.Run("absent things report absence, not an empty value", func(t *testing.T) {
		q := db.Querier

		_, err := models.FindOmTopologyRunByID(q, "nope")
		require.ErrorIs(t, err, models.ErrNotFound)

		_, err = models.FindOmTopologyRunByID(q, "")
		require.Error(t, err, "an empty id is a caller mistake, not a miss")

		runs, err := models.FindOmTopologyRuns(q, 0)
		require.NoError(t, err)
		assert.Empty(t, runs)

		require.Error(t, models.PruneOmTopologyRuns(q, 0), "keeping nothing is never what a caller meant")
	})
}

func TestOmTopologySnapshotAbsentOnEmptyDatabase(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// The cold-start path depends on this being a clean miss rather than an error.
	_, err := models.FindLatestOmTopologySnapshot(db.Querier)
	require.ErrorIs(t, err, models.ErrNotFound)
}
