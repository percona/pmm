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
	"io"
	"net/http"
	"net/http/httptest"
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

// panicVM fails the test if queried: GetTopology is a pure read path and must never reach VictoriaMetrics.
type panicVM struct{ t *testing.T }

func (p panicVM) Query(context.Context, string, time.Time, ...v1.Option) (model.Value, v1.Warnings, error) {
	p.t.Helper()
	p.t.Fatal("GetTopology must not query VictoriaMetrics: it is a pure read path")
	return nil, nil, nil
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
	assert.Equal(t, http.MethodPatch, stub.calls[0].method)
	assert.JSONEq(t, `{"ENABLED": true}`, stub.calls[0].body)
}

// TestRunRetriesInventoryEnabledUntilItLands covers PMM Extensions not answering yet at startup: nothing else
// reconciles ENABLED, so a refused startup sync must be retried on later ticks.
func TestRunRetriesInventoryEnabledUntilItLands(t *testing.T) {
	db := serviceTestDB(t)
	_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{EnableOM: new(true)})
	require.NoError(t, err)

	var mu sync.Mutex
	var patches []string
	refuse := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		raw, _ := io.ReadAll(r.Body)

		mu.Lock()
		patches = append(patches, string(raw))
		first := refuse
		refuse = false
		mu.Unlock()

		if first {
			// The side-car is still starting.
			w.WriteHeader(http.StatusServiceUnavailable)
			return
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
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.Run(ctx)
	}()

	seen := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(patches)
	}
	require.Eventually(t, func() bool { return seen() >= 2 }, 10*time.Second, 10*time.Millisecond,
		"the refused ENABLED write must be retried on a later tick, not abandoned")

	settled := seen()
	time.Sleep(10 * svc.tick)
	cancel()
	<-done

	assert.Equal(t, settled, seen(), "reconciling must stop at the first success")
	mu.Lock()
	defer mu.Unlock()
	for i, body := range patches {
		assert.JSONEq(t, `{"ENABLED": true}`, body, "PATCH %d carried the wrong state", i)
	}
}
