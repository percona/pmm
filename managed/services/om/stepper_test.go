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

		run := &sepBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts: []sepBootstrapHost{
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

		require.Len(t, stub.calls, 2)
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.calls[0].path)
		assert.Equal(t, "/api/apps/om_inventory/runs", stub.calls[1].path)
		assert.JSONEq(t, `{"node_ids": ["`+node01+`"]}`, stub.calls[1].body)
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

		run := &sepBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []sepBootstrapHost{{Host: "node00"}},
		}
		svc.completeSucceededRun(t.Context(), run)

		require.Len(t, stub.calls, 1, "no scoped refresh should be triggered")
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.calls[0].path)
	})

	t.Run("records the run as registered, so the sweep stops revisiting it", func(t *testing.T) {
		// The reason this is persisted rather than remembered: a run stays SUCCEEDED
		// in SEP for the rest of its life, so without a record of PMM's own last
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

		svc.completeSucceededRun(t.Context(), &sepBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []sepBootstrapHost{{Host: "node00"}},
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

		svc.completeSucceededRun(t.Context(), &sepBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			Hosts:          []sepBootstrapHost{{Host: "node00"}, {Host: "node01"}},
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
		svc.completeSucceededRun(t.Context(), &sepBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []sepBootstrapHost{{Host: "node00"}},
		})

		services, err := models.FindServices(db.Querier, models.ServiceFilters{NodeID: node00})
		require.NoError(t, err)
		assert.Len(t, services, 1, "the refusal must not cost the registration it follows")

		require.Len(t, stub.calls, 2)
		assert.Equal(t, "/api/apps/om_inventory/runs", stub.calls[1].path)

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.NotContains(t, registered, "run-abc", "a refused nudge has to be retried, so the run stays in the sweep")
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
		svc.completeSucceededRun(t.Context(), &sepBootstrapRun{
			ID:             "run-abc",
			Status:         bootstrapRunSucceeded,
			ReplicaSetName: "rs-test",
			FinishedAt:     &finished,
			Hosts:          []sepBootstrapHost{{Host: "node00"}},
		})

		registered, err := models.FindRegisteredOmBootstrapRunIDs(db.Querier)
		require.NoError(t, err)
		assert.Contains(t, registered, "run-abc", "past the window the run leaves the sweep regardless")
	})
}
