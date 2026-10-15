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

func TestStepBootstrapTick(t *testing.T) {
	t.Run("does not poll om_bootstrap while OpenManager is off", func(t *testing.T) {
		db := storeTestDB(t)
		stub := newSEPStub(t, http.StatusOK, `{"items": [], "total": 0}`)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithBootstrapSource(stub.server.URL, "test-token")

		svc.stepBootstrapTick(t.Context())

		assert.Empty(t, stub.calls)
	})

	t.Run("polls om_bootstrap while OpenManager is on", func(t *testing.T) {
		db := storeTestDB(t)
		_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{EnableOM: new(true)})
		require.NoError(t, err)
		stub := newSEPStub(t, http.StatusOK, `{"items": [], "total": 0}`)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithBootstrapSource(stub.server.URL, "test-token")

		svc.stepBootstrapTick(t.Context())

		assert.NotEmpty(t, stub.calls)
	})
}

func TestAdminUserCredentials(t *testing.T) {
	t.Run("supplies the administrator's credentials until the step has used them", func(t *testing.T) {
		db := storeTestDB(t)
		require.NoError(t, models.CreateOmBootstrapSecret(db.Querier, &models.OmBootstrapSecret{
			RunID:           "run-abc",
			MongoDBUsername: bootstrapMongoDBUsername,
			MongoDBPassword: "monitoring-secret",
			KeyFile:         "keyfile-content",
			AdminUsername:   "dba",
			AdminPassword:   "chosen-secret",
		}))
		run := `{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu",
		  "mongodb_version": "7.0.8", "replica_set_name": "rs-test", "started_at": "2026-01-01T00:00:00Z", "hosts": [],
		  "run_steps": [{"name": "create_admin_user", "status": "succeeded", "attempt_count": 1},
		                {"name": "create_pmm_monitoring_user", "status": "running", "attempt_count": 1}]}`
		stub := newSEPStub(t, http.StatusOK, run)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithBootstrapSource(stub.server.URL, "test-token")

		params, err := svc.paramsForStep(t.Context(), "run-abc", createAdminUserStep)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"username": "dba", "password": "chosen-secret"}, params)

		svc.stepBootstrapRun(t.Context(), "run-abc")

		secret, err := models.FindOmBootstrapSecretByRunID(db.Querier, "run-abc")
		require.NoError(t, err)
		assert.Empty(t, secret.AdminPassword)
		assert.Equal(t, "monitoring-secret", secret.MongoDBPassword.Reveal(), "the exporter still needs its own password")
		_, err = svc.paramsForStep(t.Context(), "run-abc", createAdminUserStep)
		require.ErrorIs(t, err, errAdminPasswordGone)
	})

	t.Run("cancels a run whose administrator password is not stored", func(t *testing.T) {
		stub := newSEPStub(t, http.StatusOK, `{"id": "run-abc", "status": "running", "cancel_requested": true,
		  "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-test",
		  "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`)
		svc := (&Service{db: storeTestDB(t), l: logrus.WithField("test", t.Name())}).
			WithBootstrapSource(stub.server.URL, "test-token")

		svc.dispatchRunStep(t.Context(), &extensionsBootstrapRun{ID: "run-abc"}, createAdminUserStep)

		require.Len(t, stub.calls, 1)
		assert.Equal(t, "/api/apps/om_bootstrap/runs/run-abc:cancel", stub.calls[0].path)
	})
}

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
		// The sync must precede the probe: the probe reads the app's own service copy, which only PMMSyncer fills.
		assert.Equal(t, "/api/apps/inventory/sync/", stub.calls[1].path)
		assert.JSONEq(t, `{"syncer": "`+pmmSyncerName+`"}`, stub.calls[1].body)
		assert.Equal(t, "/api/apps/om_inventory/runs", stub.calls[2].path)
		assert.JSONEq(t, `{"node_ids": ["`+node01+`"]}`, stub.calls[2].body)
	})

	t.Run("keeps an unconfirmed run in the sweep even when both nudges were accepted", func(t *testing.T) {
		// The sync answers 202 and finishes in the background, so the probe after it can miss the new service.
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

		// The debounce is per node, so a node not probed yet is still due.
		assert.Equal(t, []string{"other-node"}, svc.refreshDue([]string{node00, "other-node"}, time.Now()))
	})

	t.Run("falls back to syncing every syncer when the app does not know PMMSyncer by name", func(t *testing.T) {
		// An older PMM Extensions holds PMMSyncer under a different package path and answers 400 to pmmSyncerName.
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

		// node01 has no inventory app row yet, so PMM cannot register it.
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

		// node00 exhausted its retries and cancel was also requested. With no RollbackSteps, hostRollbackDone
		// reports every rollback finished, so the run is recorded as rolled back on this tick.
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
