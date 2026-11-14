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
	"encoding/json"
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
	assert.JSONEq(t, `{"enabled": true, "scan": true}`, stub.calls[0].body)
}

// switchStub is PMM Extensions holding om_inventory's ENABLED flag: it counts reads of it, failing them while
// unreadable is set, and records each write to it, refusing the next refusals of them.
type switchStub struct {
	refusals   atomic.Int32
	unreadable atomic.Bool
	enabled    atomic.Bool
	reads      atomic.Int32
	mu         sync.Mutex
	writes     []string
}

func (e *switchStub) bodies() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.writes)
}

// awaitReads waits until the stub has answered n more reads, each a tick that compared the two switches.
func (e *switchStub) awaitReads(t *testing.T, n int32) {
	t.Helper()
	target := e.reads.Load() + n
	require.Eventually(t, func() bool { return e.reads.Load() >= target }, 10*time.Second, 5*time.Millisecond)
}

// switchService returns a service syncing the switch saved in db to stub.
func switchService(t *testing.T, stub *switchStub, db *reform.DB) *Service {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/apps/om_inventory/config":
			stub.reads.Add(1)
			if stub.unreadable.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = fmt.Fprintf(w, `[{"key": "ENABLED", "value": %t}, {"key": "AUTO_SCAN", "value": true}]`, stub.enabled.Load())
			return
		case r.Method == http.MethodPut:
			var body struct {
				Enabled bool `json:"enabled"`
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			// Recorded and applied together, so a test that saw the write also sees its effect.
			stub.mu.Lock()
			refused := stub.refusals.Add(-1) >= 0
			stub.writes = append(stub.writes, string(raw))
			if !refused {
				stub.enabled.Store(body.Enabled)
			}
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
	return svc
}

// runWithSwitchStub runs Run, with OM enabled, against stub until the returned stop is called.
func runWithSwitchStub(t *testing.T, stub *switchStub) (*Service, func()) {
	t.Helper()

	db := serviceTestDB(t)
	_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{EnableOM: new(true)})
	require.NoError(t, err)

	svc := switchService(t, stub, db)
	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	wg.Go(func() { svc.Run(ctx) })
	stop := func() {
		cancel()
		wg.Wait()
	}
	t.Cleanup(stop)
	return svc, stop
}

// TestRunRetriesTheStartupWriteUntilItLands covers PMM Extensions not answering yet at startup while it still holds
// ENABLED from before: the matching value must not stand in for the write that asks for the restart sweep.
func TestRunRetriesTheStartupWriteUntilItLands(t *testing.T) {
	stub := &switchStub{}
	stub.enabled.Store(true)
	stub.refusals.Store(1)
	_, stop := runWithSwitchStub(t, stub)

	require.Eventually(t, func() bool { return len(stub.bodies()) == 2 }, 10*time.Second, 5*time.Millisecond,
		"the refused startup write must be retried on a later tick, not abandoned")
	stub.awaitReads(t, 3)
	stop()

	bodies := stub.bodies()
	assert.Len(t, bodies, 2, "once the startup write lands, a matching ENABLED must not be written again")
	for i, body := range bodies {
		assert.JSONEq(t, `{"enabled": true, "scan": true}`, body, "PUT %d carried the wrong state", i)
	}
}

// TestRunRewritesAMatchingSwitchOnlyAtStartup covers a restart: the startup write is what asks for the app's sweep,
// and a tick must not repeat it while ENABLED already matches.
func TestRunRewritesAMatchingSwitchOnlyAtStartup(t *testing.T) {
	stub := &switchStub{}
	stub.enabled.Store(true)
	_, stop := runWithSwitchStub(t, stub)

	require.Eventually(t, func() bool { return len(stub.bodies()) == 1 }, 10*time.Second, 5*time.Millisecond)
	stub.awaitReads(t, 3)
	stop()

	assert.Len(t, stub.bodies(), 1)
}

// TestRunRetriesAFailedSwitchChange covers an ENABLED write that failed after the switch changed: without a
// correction, PMM's switch and ENABLED disagree until the next restart.
func TestRunRetriesAFailedSwitchChange(t *testing.T) {
	stub := &switchStub{}
	svc, stop := runWithSwitchStub(t, stub)

	seen := func() int { return len(stub.bodies()) }
	require.Eventually(t, func() bool { return seen() == 1 }, 10*time.Second, 5*time.Millisecond)

	stub.refusals.Store(1)
	_, err := models.UpdateSettings(svc.db.Querier, &models.ChangeSettingsParams{EnableOM: new(false)})
	require.NoError(t, err)
	svc.SyncInventoryEnabled(t.Context())

	require.Eventually(t, func() bool { return seen() >= 3 && !stub.enabled.Load() }, 10*time.Second,
		5*time.Millisecond, "the failed switch change must be corrected on a later tick")
	stop()

	bodies := stub.bodies()
	assert.JSONEq(t, `{"enabled": false, "scan": false}`, bodies[1], "the switch change must write the switch as saved")
	assert.JSONEq(t, `{"enabled": false, "scan": false}`, bodies[len(bodies)-1])
}

// TestRunCorrectsASwitchChangedElsewhere covers ENABLED moving without this process writing it, as when another HA
// node's write failed or went stale, or PMM Extensions lost its settings, and its correction asks for no sweep.
func TestRunCorrectsASwitchChangedElsewhere(t *testing.T) {
	stub := &switchStub{}
	stub.enabled.Store(true)
	_, stop := runWithSwitchStub(t, stub)

	require.Eventually(t, func() bool { return len(stub.bodies()) == 1 }, 10*time.Second, 5*time.Millisecond)
	stub.enabled.Store(false)

	require.Eventually(t, func() bool { return len(stub.bodies()) == 2 && stub.enabled.Load() }, 10*time.Second,
		5*time.Millisecond, "a tick must restore ENABLED to PMM's switch")
	stop()

	assert.JSONEq(t, `{"enabled": true, "scan": false}`, stub.bodies()[1])
}

// TestASwitchOnThatLandsSettlesAFailedStartupWrite covers a startup write that failed before an admin turned
// Operations on: the switch-on write asks for the sweep, so the next tick must neither rewrite ENABLED nor ask again.
func TestASwitchOnThatLandsSettlesAFailedStartupWrite(t *testing.T) {
	t.Parallel()

	stub := &switchStub{}
	stub.refusals.Store(1)
	svc := switchService(t, stub, switchDB(t, false, true, true))

	svc.syncSwitch(t.Context(), true)
	svc.SyncInventoryEnabled(t.Context())
	svc.syncSwitch(t.Context(), false)

	bodies := stub.bodies()
	require.Len(t, bodies, 2)
	assert.JSONEq(t, `{"enabled": false, "scan": false}`, bodies[0])
	assert.JSONEq(t, `{"enabled": true, "scan": true}`, bodies[1])
	assert.Equal(t, int32(1), stub.reads.Load(), "the tick must compare rather than force another write")
}

// TestATickRetriesARefusedSwitchOnWithItsSweep covers a switch-on write PMM Extensions refused: the tick that retries
// it must ask for the sweep the switch-on would have started.
func TestATickRetriesARefusedSwitchOnWithItsSweep(t *testing.T) {
	t.Parallel()

	stub := &switchStub{}
	stub.refusals.Store(1)
	svc := switchService(t, stub, switchDB(t, true, true))

	svc.SyncInventoryEnabled(t.Context())
	svc.syncSwitch(t.Context(), false)

	bodies := stub.bodies()
	require.Len(t, bodies, 2)
	for i, body := range bodies {
		assert.JSONEq(t, `{"enabled": true, "scan": true}`, body, "PUT %d carried the wrong state", i)
	}
	assert.Zero(t, stub.reads.Load(), "an owed write must not wait on a comparison")
}

// TestRunWritesNothingWhileTheAppCannotBeRead covers PMM Extensions answering writes but not reads: a tick that
// cannot compare must not write.
func TestRunWritesNothingWhileTheAppCannotBeRead(t *testing.T) {
	stub := &switchStub{}
	stub.enabled.Store(true)
	_, stop := runWithSwitchStub(t, stub)

	require.Eventually(t, func() bool { return len(stub.bodies()) == 1 }, 10*time.Second, 5*time.Millisecond)
	stub.unreadable.Store(true)
	// A read already under way may have seen it readable, so let it finish before ENABLED moves.
	stub.awaitReads(t, 1)
	stub.enabled.Store(false)
	stub.awaitReads(t, 3)
	assert.Len(t, stub.bodies(), 1, "only the startup write")

	stub.unreadable.Store(false)
	require.Eventually(t, func() bool { return stub.enabled.Load() }, 10*time.Second, 5*time.Millisecond)
	stop()
}

// switchDB is a settings table whose OpenManager switch reads as each of enabled in turn, for a test that only reads it.
func switchDB(t *testing.T, enabled ...bool) *reform.DB {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet(), "every saved switch value must be read")
		_ = sqlDB.Close()
	})
	for _, e := range enabled {
		mock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(fmt.Sprintf(`{"om": {"enabled": %t}}`, e)))
	}
	return reform.NewDB(sqlDB, postgresql.Dialect, nil)
}

func TestSwitchingOnAsksForASweep(t *testing.T) {
	t.Parallel()

	// Only the app knows whether automatic scans are off, so it decides whether the sweep asked for runs.
	stub := newSEPStub(t, http.StatusOK, `{}`)
	svc := stub.service(t)
	svc.db = switchDB(t, true)

	svc.SyncInventoryEnabled(t.Context())

	require.Len(t, stub.calls, 1)
	assert.Equal(t, http.MethodPut, stub.calls[0].method)
	assert.Equal(t, "/api/apps/om_inventory/enabled", stub.calls[0].path)
	assert.JSONEq(t, `{"enabled": true, "scan": true}`, stub.calls[0].body)
}

func TestSyncWithoutExtensionsIsANoOp(t *testing.T) {
	t.Parallel()

	// No database: a sync that read PMM's switch with nowhere to write it would panic.
	svc := &Service{l: logrus.WithField("test", t.Name())}

	svc.syncSwitch(t.Context(), true)
}
