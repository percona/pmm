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
	"context"
	"sync"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

// panicVM fails the test the moment it is queried. GetTopology's whole point after the
// PMM-15326 read-path fix is that it never reaches VictoriaMetrics -- collection is
// leader-only and request-triggered nowhere -- so any test that exercises the read path
// wires this in and lets a query panic rather than silently succeed.
type panicVM struct{ t *testing.T }

func (p panicVM) Query(context.Context, string, time.Time, ...v1.Option) (model.Value, v1.Warnings, error) {
	p.t.Helper()
	p.t.Fatal("GetTopology must not query VictoriaMetrics: it is a pure read path")
	return nil, nil, nil
}

// fakeHA is a haChecker double.
type fakeHA struct {
	leader   bool
	leaderID string
}

func (f fakeHA) IsLeader() bool   { return f.leader }
func (f fakeHA) LeaderID() string { return f.leaderID }

func serviceTestDB(t *testing.T) *reform.DB {
	t.Helper()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	return reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
}

// TestGetTopologyNeverCollects is the regression guard for PMM-15326's read-path fix:
// GetTopology must serve memory or the stored snapshot and never reach collect(), on a
// freshly restored service or on any later call. Before the fix, a restored document
// older than 30s missed the cache immediately and every request rebuilt it -- see
// service.go's package comment history in the PR for the mechanism.
func TestGetTopologyNeverCollects(t *testing.T) {
	db := serviceTestDB(t)

	// Written by one Service instance, as if by a leader that has since gone away --
	// restored by a second, as pmm-managed does across a restart.
	writer := &Service{db: db, l: logrus.WithField("test", t.Name())}
	generatedAt := time.Now().Truncate(time.Microsecond)
	response, run := testTopologyResponse("restored-run", generatedAt, generatedAt.Add(-2*time.Second))
	require.NoError(t, writer.persist(t.Context(), response, run, "pmm-server"))

	reader := &Service{db: db, vmClient: panicVM{t: t}, l: logrus.WithField("test", t.Name())}

	first, err := reader.GetTopology(t.Context(), &omv1.GetTopologyRequest{})
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "restored-run", first.Snapshot.RunId)
	assert.False(t, first.Snapshot.Stale, "just-restored document should read as fresh")

	// The DB is gone from here on. sync.Once guarantees restoreOnce's own query never
	// repeats, but this proves the read path holds that guarantee rather than assuming
	// it -- a second GetTopology call has nothing to fall back on but memory.
	require.NoError(t, db.DBInterface().(interface{ Close() error }).Close())

	second, err := reader.GetTopology(t.Context(), &omv1.GetTopologyRequest{})
	require.NoError(t, err)
	assert.Equal(t, first.Snapshot.RunId, second.Snapshot.RunId)
}

// TestGetTopologyColdStart covers the estate that has never collected anywhere: no
// memory, nothing stored. GetTopology must answer with an empty, explicitly stale
// document rather than an error or a wait for a collection that GetTopology itself must
// never start.
func TestGetTopologyColdStart(t *testing.T) {
	db := serviceTestDB(t)
	svc := &Service{db: db, vmClient: panicVM{t: t}, l: logrus.WithField("test", t.Name())}

	response, err := svc.GetTopology(t.Context(), &omv1.GetTopologyRequest{})
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.True(t, response.Snapshot.Stale)
	assert.Nil(t, response.Snapshot.ObservedAt)
	assert.Empty(t, response.Environments)
}

// TestGetTopologyConcurrentReadsDoNotRace exercises the read path from many goroutines
// at once. Run with -race: the point is that GetTopology touches only s.mu and
// s.restored, never s.running, so concurrent readers never contend with each other or
// with a collection.
func TestGetTopologyConcurrentReadsDoNotRace(t *testing.T) {
	db := serviceTestDB(t)
	svc := &Service{db: db, vmClient: panicVM{t: t}, l: logrus.WithField("test", t.Name())}

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			_, err := svc.GetTopology(t.Context(), &omv1.GetTopologyRequest{})
			assert.NoError(t, err)
		})
	}
	wg.Wait()
}

func TestTriggerTopologyCollectionRefusesOnNonLeader(t *testing.T) {
	svc := &Service{ha: fakeHA{leader: false, leaderID: "node-b"}, l: logrus.WithField("test", t.Name())}

	_, err := svc.TriggerTopologyCollection(t.Context(), &omv1.TriggerTopologyCollectionRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), "node-b")
}

func TestTriggerTopologyCollectionRefusesOnNonLeaderWithoutKnownLeader(t *testing.T) {
	svc := &Service{ha: fakeHA{leader: false, leaderID: ""}, l: logrus.WithField("test", t.Name())}

	_, err := svc.TriggerTopologyCollection(t.Context(), &omv1.TriggerTopologyCollectionRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

// TestTriggerTopologyCollectionAllowsLeaderOrNoHA covers the common cases: an explicit
// leader, and no ha wired in at all -- every test in this package but the ones above, and
// single-node PMM without HA enabled at all (ha.Service.IsLeader() is true when HA is
// disabled, so the same "every node is the leader" reading applies).
func TestTriggerTopologyCollectionAllowsLeaderOrNoHA(t *testing.T) {
	for name, ha := range map[string]haChecker{"leader": fakeHA{leader: true}, "no ha wired in": nil} {
		t.Run(name, func(t *testing.T) {
			svc := &Service{ha: ha, l: logrus.WithField("test", t.Name())}
			svc.running.Lock() // short-circuit before collect() needs a database.
			defer svc.running.Unlock()

			_, err := svc.TriggerTopologyCollection(t.Context(), &omv1.TriggerTopologyCollectionRequest{})

			require.Error(t, err)
			assert.Equal(t, codes.Aborted, status.Code(err), "should reach the in-flight check, not refuse as a follower")
		})
	}
}

func TestTriggerTopologyCollectionAbortsWhileRunInFlight(t *testing.T) {
	svc := &Service{l: logrus.WithField("test", t.Name())}
	svc.running.Lock()
	defer svc.running.Unlock()

	_, err := svc.TriggerTopologyCollection(t.Context(), &omv1.TriggerTopologyCollectionRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

func TestSourcesOrdering(t *testing.T) {
	t.Run("without a probe, just inventory and metrics", func(t *testing.T) {
		svc := &Service{l: logrus.WithField("test", t.Name())}
		sources := svc.sources(nil, time.Now())

		require.Len(t, sources, 2)
		assert.Equal(t, sourceInventory, sources[0].key())
		assert.Equal(t, sourceMetrics, sources[1].key())
	})

	t.Run("with a probe configured, probe is appended last", func(t *testing.T) {
		client := &sepClient{baseURL: "http://sep.example", http: nil}
		svc := &Service{
			l:     logrus.WithField("test", t.Name()),
			probe: &probeSource{app: client.app(probeAppModule), l: logrus.WithField("source", sourceProbe)},
		}
		sources := svc.sources(nil, time.Now())

		require.Len(t, sources, 3)
		assert.Equal(t, sourceInventory, sources[0].key())
		assert.Equal(t, sourceMetrics, sources[1].key())
		assert.Equal(t, sourceProbe, sources[2].key())
	})
}

// TestCollectPersistFailureStillReturnsDocument covers the run whose write fails: the
// document was already built and is already correct, so losing the run record must not
// turn into losing the response.
func TestCollectPersistFailureStillReturnsDocument(t *testing.T) {
	db := serviceTestDB(t)

	node, err := models.CreateNode(db.Querier, models.GenericNodeType, &models.CreateNodeParams{NodeName: "node00"})
	require.NoError(t, err)
	port := uint16(27017)
	address := "127.0.0.1"
	_, err = models.AddNewService(db.Querier, models.MongoDBServiceType, &models.AddDBMSServiceParams{
		ServiceName: "mongo-1",
		NodeID:      node.NodeID,
		Address:     &address,
		Port:        &port,
	})
	require.NoError(t, err)

	// om_topology_runs is gone, so persist()'s INSERT fails -- collect() must log that
	// and still hand back the response it already built and published.
	_, err = db.Exec(`ALTER TABLE om_topology_runs RENAME TO om_topology_runs_test_gone`)
	require.NoError(t, err)

	svc := &Service{db: db, vmClient: &recordingVM{}, l: logrus.WithField("test", t.Name())}
	response, run, err := svc.collect(t.Context())

	require.NoError(t, err, "a failed persist must not fail collect")
	require.NotNil(t, response)
	require.NotNil(t, run)
	assert.Equal(t, int32(1), response.Summary.TotalServices)
	assert.Same(t, response, svc.snapshot(), "the response collect returned is the one it published")
}
