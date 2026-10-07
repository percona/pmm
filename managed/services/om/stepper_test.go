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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestCompleteSucceededRun(t *testing.T) {
	t.Run("triggers a scoped inventory refresh only for hosts the inventory app has not confirmed yet", func(t *testing.T) {
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		node01, _ := registerTestNode(t, db, "node01")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		// node00's service is already visible to the inventory app; node01's is not
		// yet -- the sweep that would show it just hasn't happened.
		stub := newSEPStubSeq(
			t, http.StatusOK,
			fmt.Sprintf(
				`{"items": [
					{"node_id": %q, "executor_host": "node00", "services": [{"service_id": "s1"}]},
					{"node_id": %q, "executor_host": "node01", "services": []}
				], "total": 2, "offset": 0, "limit": 200}`, node00, node01,
			),
			`{}`,
		)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		run := &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts: []extensionsBootstrapHost{
				{Host: "node00"},
				{Host: "node01"},
			},
		}
		svc.completeSucceededRun(t.Context(), run)

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: node00})
		require.NoError(t, err)
		assert.Len(t, services, 1, "node00 should still be registered")
		services, err = models.FindServices(db.Querier, models.ServiceFilters{NodeID: node01})
		require.NoError(t, err)
		assert.Len(t, services, 1, "node01 should still be registered")

		require.Len(t, stub.calls, 3)
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.calls[0].path)
		// The sync comes between reading the estate and re-probing it: the
		// service was just created in PMM, and the probe resolves services from
		// the app's own copy, which only PMMSyncer fills.
		assert.Equal(t, "/api/apps/inventory/sync/", stub.calls[1].path)
		assert.JSONEq(t, `{"syncer": "`+pmmSyncerName+`"}`, stub.calls[1].body)
		assert.Equal(t, "/api/apps/om_inventory/runs", stub.calls[2].path)
		assert.JSONEq(t, `{"node_ids": ["`+node01+`"]}`, stub.calls[2].body)
	})

	t.Run("keeps an unconfirmed run in the sweep even when both nudges were accepted", func(t *testing.T) {
		// The regression this pins, and the reason the hold is on the estate's
		// answer rather than on whether the requests were taken.
		//
		// The sync answers 202 and finishes in the background, so the probe
		// fired straight after it can easily run before the service it is
		// looking for arrives. Measured on a real deployment: two runs finished
		// at 08:40:31 and 08:41:31, each nudged at exactly that second, both
		// found nothing, and confirm_monitoring read "running" until the app's
		// own 10-minute sweep at 08:50:58 - nine minutes of a wholly successful
		// bootstrap looking hung. Treating "both calls accepted" as done is what
		// let the run leave the sweep before the data it was waiting for existed.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		// Every call succeeds; the estate simply has no service yet.
		stub := newSEPStub(t, http.StatusOK,
			fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00))
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		finished := time.Now()
		svc.completeSucceededRun(t.Context(), &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		})

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.NotContains(t, registered, "run-abc",
			"an accepted sync and refresh do not mean the host is confirmed; the run must stay for the next tick")
	})

	t.Run("debounces the sync, so one bootstrap does not cause a pull per tick", func(t *testing.T) {
		// completeSucceededRun is reached every 15s for as long as
		// refreshRetryWindowOpen holds a run, and the sync is global rather than
		// per-run, so one pull serves every run finishing together.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		stub := newSEPStub(t, http.StatusOK,
			fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00))
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		finished := time.Now()
		run := &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		}
		svc.completeSucceededRun(t.Context(), run)
		svc.completeSucceededRun(t.Context(), run)

		syncs := 0
		for _, c := range stub.calls {
			if c.path == "/api/apps/inventory/sync/" {
				syncs++
			}
		}
		assert.Equal(t, 1, syncs, "the second tick must reuse the pull already in flight")
	})

	t.Run("debounces the scoped refresh per node, so a held run does not re-probe every tick", func(t *testing.T) {
		// The sync got a debounce for the every-tick hold; the probe it feeds
		// costs more and must not be left asking faster than the sync can
		// deliver anything new for it to read.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		stub := newSEPStub(t, http.StatusOK,
			fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00))
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		finished := time.Now()
		run := &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		}
		svc.completeSucceededRun(t.Context(), run)
		svc.completeSucceededRun(t.Context(), run)

		refreshes := 0
		for _, c := range stub.calls {
			if c.path == "/api/apps/om_inventory/runs" {
				refreshes++
			}
		}
		assert.Equal(t, 1, refreshes, "the second tick is inside inventoryRefreshDebounce for this node")

		// A node first seen on the second tick is not held by the first one's
		// window: the debounce is per node precisely so a run finishing while
		// another is held still gets its own first probe straight away.
		assert.Equal(t, []string{"other-node"}, svc.refreshDue([]string{node00, "other-node"}, time.Now()))
	})

	t.Run("falls back to syncing every syncer when the app does not know PMMSyncer by name", func(t *testing.T) {
		// pmmSyncerName is a Python dotted path across a release boundary: PMM
		// Extensions renamed the package holding it, and an older side-car
		// answers 400 rather than no-opping. Without the fallback that 400
		// restores the nine-minute hang the sync exists to remove.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		hosts := fmt.Sprintf(
			`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00,
		)
		stub := newSEPStubSeqCodes(t,
			[]int{http.StatusOK, http.StatusBadRequest, http.StatusAccepted, http.StatusAccepted},
			[]string{hosts, `{"detail": "unknown syncer"}`, "", ""})
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		svc.completeSucceededRun(t.Context(), &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		})

		require.Len(t, stub.calls, 4)
		assert.Equal(t, "/api/apps/inventory/sync/", stub.calls[1].path)
		assert.JSONEq(t, `{"syncer": "`+pmmSyncerName+`"}`, stub.calls[1].body)
		assert.Equal(t, "/api/apps/inventory/sync/", stub.calls[2].path)
		assert.JSONEq(t, `{}`, stub.calls[2].body, "an absent syncer runs every configured one")
		assert.Equal(t, "/api/apps/om_inventory/runs", stub.calls[3].path)
	})

	t.Run("retries the sync on the next tick when the request never reached the app", func(t *testing.T) {
		// The debounce is armed before the request so two runs completing in one
		// tick cannot both ask. A failure has to hand that window back, or a
		// single unreachable moment costs 45s of a 5-minute retry budget for a
		// pull nobody made.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		hosts := fmt.Sprintf(
			`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00,
		)
		stub := newSEPStubSeqCodes(t,
			[]int{http.StatusOK, http.StatusInternalServerError, http.StatusAccepted, http.StatusOK, http.StatusAccepted},
			[]string{hosts, `{"detail": "boom"}`, "", hosts, ""})
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		finished := time.Now()
		run := &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		}
		svc.completeSucceededRun(t.Context(), run)
		svc.completeSucceededRun(t.Context(), run)

		syncs := 0
		for _, c := range stub.calls {
			if c.path == "/api/apps/inventory/sync/" {
				syncs++
			}
		}
		assert.Equal(t, 2, syncs, "a sync that failed must not hold the next tick off")
	})

	t.Run("triggers no refresh once the inventory app has confirmed every host", func(t *testing.T) {
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		stub := newSEPStub(t, http.StatusOK,
			fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": [{"service_id": "s1"}]}], "total": 1, "offset": 0, "limit": 200}`, node00))
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		run := &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		}
		svc.completeSucceededRun(t.Context(), run)

		require.Len(t, stub.calls, 1, "no scoped refresh should be triggered")
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.calls[0].path)
	})

	t.Run("records the run as registered, so the sweep stops revisiting it", func(t *testing.T) {
		// The reason this is persisted rather than remembered: a run stays SUCCEEDED
		// in PMM Extensions for the rest of its life, so without a record of PMM's own last
		// step every finished run in the history would be re-fetched, re-matched
		// against the whole estate and re-registered on every 15s tick, forever --
		// and a new leader after a failover would start over.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		stub := newSEPStub(t, http.StatusOK,
			fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": [{"service_id": "s1"}]}], "total": 1, "offset": 0, "limit": 200}`, node00))
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		svc.completeSucceededRun(t.Context(), &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		})

		config, err := models.FindOmBootstrapRunConfigByRunID(db.Querier, "run-abc")
		require.NoError(t, err, "an unlabelled run gets its row on completion")
		assert.NotNil(t, config.RegisteredAt)

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.Contains(t, registered, "run-abc")
	})

	t.Run("leaves a run whose host the inventory app has not seen for the next tick", func(t *testing.T) {
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		// node01 bootstrapped, but the inventory app has no row for it yet, so PMM
		// cannot register it and the run is not finished from PMM's side.
		stub := newSEPStubSeq(t, http.StatusOK,
			fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00),
			`{}`)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		svc.completeSucceededRun(t.Context(), &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}, {Host: "node01"}},
		})

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.NotContains(t, registered, "run-abc", "a run with an unregistered host has to come back next tick")
	})

	t.Run("keeps a run whose refresh the inventory app refused, so the next tick asks again", func(t *testing.T) {
		// The refusal to expect is a 409 from the app's own estate sweep already
		// holding this host -- which is likeliest exactly when a run finishes. Every
		// host here registered, so nothing but the nudge is outstanding, and marking
		// the run registered is what takes it out of the sweep for good: dropping the
		// nudge here means it is never retried at all.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		stub := newSEPStubSeqCodes(t,
			[]int{http.StatusOK, http.StatusConflict},
			[]string{
				fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00),
				`{"detail": "node00 is already being refreshed"}`,
			})
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		finished := time.Now()
		svc.completeSucceededRun(t.Context(), &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		})

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: node00})
		require.NoError(t, err)
		assert.Len(t, services, 1, "the refusal must not cost the registration it follows")

		require.Len(t, stub.calls, 3)
		assert.Equal(t, "/api/apps/inventory/sync/", stub.calls[1].path)
		assert.Equal(t, "/api/apps/om_inventory/runs", stub.calls[2].path)

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.NotContains(t, registered, "run-abc", "an unconfirmed host has to be looked at again, so the run stays in the sweep")
	})

	t.Run("gives up on the refresh once the run is too old, rather than sweeping it forever", func(t *testing.T) {
		// The other half of the branch above: a host the app will never accept a
		// refresh for must not pin the run in a list re-read every 15 seconds for the
		// life of the server. confirm_monitoring then waits out the app's own
		// schedule, which is where it stood before the nudge existed.
		db := storeTestDB(t)
		node00, _ := registerTestNode(t, db, "node00")
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: "admin",
			MongoDBPassword: "secret",
			KeyFile:         "keyfile-content",
		}))

		stub := newSEPStubSeqCodes(t,
			[]int{http.StatusOK, http.StatusConflict},
			[]string{
				fmt.Sprintf(`{"items": [{"node_id": %q, "executor_host": "node00", "services": []}], "total": 1, "offset": 0, "limit": 200}`, node00),
				`{"detail": "node00 is already being refreshed"}`,
			})
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")

		finished := time.Now().Add(-bootstrapInventoryRefreshWindow - time.Minute)
		svc.completeSucceededRun(t.Context(), &extensionsBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []extensionsBootstrapHost{{Host: "node00"}},
		})

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.Contains(t, registered, "run-abc", "past the window the run leaves the sweep regardless")
	})
}

func TestAdvanceRunningRunRollbackReason(t *testing.T) {
	t.Parallel()

	t.Run("credits retry exhaustion over a cancellation requested after rollback was already needed", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{}`)
		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithBootstrapSource(stub.server.URL, "test-token")

		// node00 already exhausted its retries -- rollback is underway for that
		// reason alone -- and an operator also clicked cancel while it was still
		// in flight. Empty RollbackSteps means hostRollbackDone trivially agrees
		// every host's rollback already finished, so this run is ready to be
		// recorded as ROLLED_BACK on this tick.
		run := &extensionsBootstrapRun{
			ID:              "run-abc",
			Status:          bootstrapRunRunning,
			CancelRequested: true,
			Hosts: []extensionsBootstrapHost{
				{
					Host: "node00",
					Steps: []extensionsBootstrapStep{
						{Name: "install_package", Status: bootstrapStepFailed, AttemptCount: bootstrapMaxAttempts},
					},
				},
			},
		}
		svc.advanceRunningRun(t.Context(), run)

		require.Len(t, stub.calls, 1)
		assert.Equal(t, "/api/apps/om_bootstrap/runs/run-abc:finish", stub.calls[0].path)
		assert.JSONEq(t,
			`{"status": "rolled_back", "error": "install_package failed on node00 after 2 attempts; every host was rolled back"}`,
			stub.calls[0].body)
	})
}
