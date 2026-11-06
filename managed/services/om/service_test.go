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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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

// panicVM fails the test if queried: GetTopology is a pure read path and must never reach VictoriaMetrics.
type panicVM struct{ t *testing.T }

func (p panicVM) Query(context.Context, string, time.Time, ...v1.Option) (model.Value, v1.Warnings, v1.Infos, error) {
	p.t.Helper()
	p.t.Fatal("GetTopology must not query VictoriaMetrics: it is a pure read path")
	return nil, nil, nil, nil
}

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

// TestGetTopologyNeverCollects checks GetTopology serves memory or the stored snapshot and never collects.
func TestGetTopologyNeverCollects(t *testing.T) {
	db := serviceTestDB(t)

	// Persisted by one instance and restored by another, as across a restart.
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

	// With the DB closed, a second call can only be served from memory.
	require.NoError(t, db.DBInterface().(interface{ Close() error }).Close())

	second, err := reader.GetTopology(t.Context(), &omv1.GetTopologyRequest{})
	require.NoError(t, err)
	assert.Equal(t, first.Snapshot.RunId, second.Snapshot.RunId)
}

// TestGetTopologyColdStart checks that with nothing collected, GetTopology returns an empty, stale document.
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

// TestGetTopologyConcurrentReadsDoNotRace is meaningful only under -race.
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
		client := &extensionsClient{baseURL: "http://sep.example", http: nil}
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

	// Renaming the table makes persist()'s INSERT fail.
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

// TestRunSyncsInventoryEnabledOnStartup covers a server started already OM-enabled: applyOMSwitch fires only
// on a ChangeSettings transition, so Run must sync PMM Extensions' ENABLED flag itself before the ticker loop.
func TestRunSyncsInventoryEnabledOnStartup(t *testing.T) {
	db := serviceTestDB(t)
	_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{EnableOM: new(true)})
	require.NoError(t, err)

	stub := newSEPStub(t, http.StatusOK, `{}`)
	svc := stub.service(t)
	svc.db = db

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.Run(ctx)
	}()
	time.Sleep(100 * time.Millisecond) // let the startup sync reach the stub before Run sees cancellation.
	cancel()
	<-done

	require.NotEmpty(t, stub.calls, "Run must sync PMM Extensions' ENABLED flag before ever reaching the ticker loop")
	assert.Equal(t, http.MethodPut, stub.calls[0].method)
	assert.Equal(t, "/api/apps/om_inventory/enabled", stub.calls[0].path)
	assert.JSONEq(t, `{"enabled": true}`, stub.calls[0].body)
}

// enabledStub is PMM Extensions recording each ENABLED write and refusing the next refusals of them.
type enabledStub struct {
	refusals atomic.Int32
	mu       sync.Mutex
	writes   []string
}

func (e *enabledStub) bodies() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.writes)
}

// runWithEnabledStub runs Run, with OM enabled, against an enabledStub until the returned stop is called.
func runWithEnabledStub(t *testing.T, refusals int32) (*enabledStub, *Service, func()) {
	t.Helper()

	db := serviceTestDB(t)
	_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{EnableOM: new(true)})
	require.NoError(t, err)

	stub := &enabledStub{}
	stub.refusals.Store(refusals)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut {
			// Decided before the write is recorded, so a test that saw the write cannot change the answer.
			refused := stub.refusals.Add(-1) >= 0
			raw, _ := io.ReadAll(r.Body)
			stub.mu.Lock()
			stub.writes = append(stub.writes, string(raw))
			stub.mu.Unlock()
			if refused {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(server.URL, "test-token")
	svc.db = db
	svc.vmClient = &recordingVM{}
	svc.tick = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	wg.Go(func() { svc.Run(ctx) })
	stop := func() {
		cancel()
		wg.Wait()
	}
	t.Cleanup(stop)
	return stub, svc, stop
}

// TestRunRetriesInventoryEnabledUntilItLands covers PMM Extensions not answering yet at startup: nothing else
// reconciles ENABLED, so a refused startup sync must be retried on later ticks.
func TestRunRetriesInventoryEnabledUntilItLands(t *testing.T) {
	stub, svc, stop := runWithEnabledStub(t, 1)

	seen := func() int { return len(stub.bodies()) }
	require.Eventually(t, func() bool { return seen() >= 2 }, 10*time.Second, 10*time.Millisecond,
		"the refused ENABLED write must be retried on a later tick, not abandoned")

	settled := seen()
	time.Sleep(10 * svc.tick)
	stop()

	assert.Equal(t, settled, seen(), "reconciling must stop at the first success")
	for i, body := range stub.bodies() {
		assert.JSONEq(t, `{"enabled": true}`, body, "PUT %d carried the wrong state", i)
	}
}

// TestRunRetriesAFailedSwitchChange covers an ENABLED write that failed after the switch changed: without a
// retry, PMM's switch and ENABLED disagree until the next restart.
func TestRunRetriesAFailedSwitchChange(t *testing.T) {
	stub, svc, stop := runWithEnabledStub(t, 0)

	seen := func() int { return len(stub.bodies()) }
	require.Eventually(t, func() bool { return seen() == 1 }, 10*time.Second, 10*time.Millisecond)

	stub.refusals.Store(1)
	_, err := models.UpdateSettings(svc.db.Querier, &models.ChangeSettingsParams{EnableOM: new(false)})
	require.NoError(t, err)
	svc.SyncInventoryEnabled(t.Context(), false)

	require.Eventually(t, func() bool { return seen() >= 3 }, 10*time.Second, 10*time.Millisecond,
		"the failed switch change must be retried on a later tick")
	stop()

	bodies := stub.bodies()
	assert.JSONEq(t, `{"enabled": false}`, bodies[len(bodies)-1])
}

// TestRunCorrectsAStaleSwitchWrite covers a write that landed after PMM's switch had moved on, as one from an older
// ChangeSettings can.
func TestRunCorrectsAStaleSwitchWrite(t *testing.T) {
	stub, svc, stop := runWithEnabledStub(t, 0)

	seen := func() int { return len(stub.bodies()) }
	require.Eventually(t, func() bool { return seen() == 1 }, 10*time.Second, 10*time.Millisecond)

	_, err := models.UpdateSettings(svc.db.Querier, &models.ChangeSettingsParams{EnableOM: new(false)})
	require.NoError(t, err)
	svc.SyncInventoryEnabled(t.Context(), true)

	require.Eventually(t, func() bool { return seen() >= 3 }, 10*time.Second, 10*time.Millisecond,
		"a write that no longer matches PMM's switch must be corrected on a later tick")
	stop()

	bodies := stub.bodies()
	assert.JSONEq(t, `{"enabled": false}`, bodies[len(bodies)-1])
}

// switchDB is a settings table holding PMM's OpenManager switch, for a test that only reads it.
func switchDB(t *testing.T, enabled bool) *reform.DB {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	mock.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(fmt.Sprintf(`{"om": {"enabled": %t}}`, enabled)))
	return reform.NewDB(sqlDB, postgresql.Dialect, nil)
}

func TestSwitchingOnLeavesTheSweepToTheApp(t *testing.T) {
	t.Parallel()

	// Only the app knows whether automatic scans are off, so it decides whether a sweep follows.
	stub := newSEPStub(t, http.StatusOK, `{}`)
	svc := stub.service(t)
	svc.db = switchDB(t, true)

	svc.SyncInventoryEnabled(t.Context(), true)

	require.Len(t, stub.calls, 1)
	assert.Equal(t, http.MethodPut, stub.calls[0].method)
	assert.Equal(t, "/api/apps/om_inventory/enabled", stub.calls[0].path)
	assert.JSONEq(t, `{"enabled": true}`, stub.calls[0].body)
}

func TestSyncWithoutExtensionsLeavesNothingToRetry(t *testing.T) {
	t.Parallel()

	svc := &Service{l: logrus.WithField("test", t.Name())}

	svc.SyncInventoryEnabled(t.Context(), true)

	assert.True(t, svc.enabledSynced.Load(), "Run must not re-read the settings on every tick for nothing")
}
