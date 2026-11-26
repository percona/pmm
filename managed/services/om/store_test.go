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

package om

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func storeTestDB(t *testing.T) *reform.DB {
	t.Helper()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
}

func testTopologyResponse(runID string, generatedAt, observedAt time.Time) (*omv1.GetTopologyResponse, *omv1.TopologyRun) {
	response := &omv1.GetTopologyResponse{
		Snapshot: &omv1.Snapshot{
			GeneratedAt:   timestamppb.New(generatedAt),
			ObservedAt:    timestamppb.New(observedAt),
			Stale:         false,
			SchemaVersion: schemaVersion,
			RunId:         runID,
		},
		OriginNode:    optional("pmm-server"),
		SourceQueries: sourceQueries,
		Summary: &omv1.Summary{
			Environments:      1,
			Clusters:          1,
			TotalServices:     1,
			UpServices:        1,
			ProcessRoleCounts: map[string]int32{"PROCESS_ROLE_MONGOD": 1},
		},
		Environments: []*omv1.Environment{
			{
				EnvName: optional("prod"),
				Clusters: []*omv1.Cluster{
					{
						Name: optional("rs0"),
						Id:   clusterID("prod", "rs0"),
						Type: omv1.ClusterType_CLUSTER_TYPE_REPLICA_SET,
						Services: []*omv1.TopologyService{
							{
								ServiceName: "mongo-1",
								ServiceId:   optional("svc-1"),
								Status:      omv1.ServiceStatus_SERVICE_STATUS_UP,
								ProcessRole: omv1.ProcessRole_PROCESS_ROLE_MONGOD,
								ObservedAt:  timestamppb.New(observedAt),
							},
						},
					},
				},
			},
		},
	}
	run := &omv1.TopologyRun{
		RunId:     runID,
		Status:    omv1.RunStatus_RUN_STATUS_SUCCESS,
		StartTime: timestamppb.New(generatedAt.Add(-time.Second)),
		EndTime:   timestamppb.New(generatedAt),
		Counts: &omv1.TopologyRunCounts{
			TotalServices:    1,
			ResolvedServices: 1,
			SuccessfulProbes: 1,
		},
		Sources: []*omv1.SourceReport{
			{Source: sourceMetrics, Status: omv1.SourceStatus_SOURCE_STATUS_OK, Facts: 3},
		},
		Errors: []*omv1.TopologyRunError{},
	}
	return response, run
}

func TestStorePersistRestoreRoundTrip(t *testing.T) {
	db := storeTestDB(t)
	svc := &Service{db: db, l: logrus.WithField("test", t.Name())}

	// Recent enough that restore() still finds it fresh, and truncated to the microsecond precision
	// that Postgres timestamp columns keep.
	generatedAt := time.Now().Truncate(time.Microsecond)
	observedAt := generatedAt.Add(-4 * time.Second)
	runID := uuid.New().String()
	response, run := testTopologyResponse(runID, generatedAt, observedAt)

	err := svc.persist(t.Context(), response, run, "pmm-server")
	require.NoError(t, err)

	restored, restoredGeneratedAt, err := svc.restore(t.Context())
	require.NoError(t, err)
	require.NotNil(t, restored)
	assert.WithinDuration(t, generatedAt, restoredGeneratedAt, time.Millisecond)

	diff := cmp.Diff(response, restored, protocmp.Transform())
	assert.Empty(t, diff, "restored document should equal what was persisted")

	run2, err := svc.getRun(t.Context(), runID)
	require.NoError(t, err)
	assert.Empty(t, cmp.Diff(run, run2, protocmp.Transform()))
}

func TestStoreRestoreRestalesAgainstNow(t *testing.T) {
	db := storeTestDB(t)
	svc := &Service{db: db, l: logrus.WithField("test", t.Name())}

	// Stored as fresh but an hour old: restore() must recompute Stale rather than trust the stored value.
	generatedAt := time.Now().Add(-time.Hour)
	observedAt := generatedAt
	response, run := testTopologyResponse(uuid.New().String(), generatedAt, observedAt)
	response.Snapshot.Stale = false

	require.NoError(t, svc.persist(t.Context(), response, run, "pmm-server"))

	restored, _, err := svc.restore(t.Context())
	require.NoError(t, err)
	assert.True(t, restored.Snapshot.Stale, "an hour-old observation must restore as stale")
}

func TestStoreRestoreNoSnapshot(t *testing.T) {
	db := storeTestDB(t)
	svc := &Service{db: db, l: logrus.WithField("test", t.Name())}

	response, generatedAt, err := svc.restore(t.Context())
	require.NoError(t, err)
	assert.Nil(t, response)
	assert.True(t, generatedAt.IsZero())
}

func TestPruneOmTopologyRunsKeepsExactlyKeep(t *testing.T) {
	db := storeTestDB(t)
	svc := &Service{db: db, l: logrus.WithField("test", t.Name())}

	const total = 5
	const keep = 3
	base := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	runIDs := make([]string, 0, total)
	for i := range total {
		generatedAt := base.Add(time.Duration(i) * time.Minute)
		runID := uuid.New().String()
		runIDs = append(runIDs, runID)
		response, run := testTopologyResponse(runID, generatedAt, generatedAt)
		run.StartTime = timestamppb.New(generatedAt)
		require.NoError(t, svc.persist(t.Context(), response, run, "pmm-server"))
	}

	err := db.InTransactionContext(t.Context(), nil, func(tx *reform.TX) error {
		return models.PruneOmTopologyRuns(tx.Querier, keep)
	})
	require.NoError(t, err)

	runs, err := svc.listRuns(t.Context(), total)
	require.NoError(t, err)
	require.Len(t, runs, keep)

	gotIDs := make([]string, 0, len(runs))
	for _, r := range runs {
		gotIDs = append(gotIDs, r.RunId)
	}
	wantIDs := []string{runIDs[4], runIDs[3], runIDs[2]}
	assert.Equal(t, wantIDs, gotIDs)

	_, err = models.FindOmTopologyRunByID(db.Querier, runIDs[0])
	require.ErrorIs(t, err, models.ErrNotFound)

	_, err = db.FindByPrimaryKeyFrom(models.OmTopologySnapshotTable, runIDs[0])
	require.ErrorIs(t, err, reform.ErrNoRows)
}
