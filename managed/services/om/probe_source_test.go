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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// `gone` is a service the app knows and this PMM does not. `total` matches the item count so
// fetchAllPages stops after one page.
const servicesBody = `{"items": [
  {
    "service_id": "s1", "node_id": "n1", "name": "mongo-1", "port": 27017, "role": null,
    "observed": {
      "collected_at": "2026-08-11T11:32:30+00:00",
      "installed_version": "7.0.40-22",
      "config_path": "/etc/mongod-node.conf",
      "argv": "/usr/bin/mongod --config /etc/mongod-node.conf",
      "uptime_seconds": 10295,
      "server_running": true
    },
    "first_seen_at": "2026-08-11T10:00:00Z", "last_attempt_at": "2026-08-11T11:32:30Z",
    "last_success_at": "2026-08-11T11:32:30Z", "failing_since": null,
    "consecutive_failures": 0, "last_error": null
  },
  {
    "service_id": "gone", "node_id": "n9", "name": "mongo-9", "port": 27017, "role": null,
    "observed": {"collected_at": "2026-08-11T11:32:30+00:00", "installed_version": "6.0.1"},
    "first_seen_at": "2026-08-11T10:00:00Z", "last_attempt_at": "2026-08-11T11:32:30Z",
    "last_success_at": "2026-08-11T11:32:30Z", "failing_since": null,
    "consecutive_failures": 0, "last_error": null
  }
], "total": 2, "offset": 0, "limit": 200}`

const unprobedBody = `{"items": [
  {
    "service_id": "s1", "node_id": "n1", "name": "mongo-1", "port": 27017, "role": null,
    "observed": {}, "first_seen_at": "2026-08-11T10:00:00Z",
    "last_attempt_at": null, "last_success_at": null, "failing_since": null,
    "consecutive_failures": 0, "last_error": null
  }
], "total": 1, "offset": 0, "limit": 200}`

func probeTestServices() []*models.Service {
	return []*models.Service{
		mongoService("s1", "mongo-1", "cl1", "rs0", "prod"),
		mongoService("s2", "mongo-2", "cl1", "rs0", "prod"),
	}
}

func newProbeSource(t *testing.T, handler http.HandlerFunc) probeSource {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &extensionsClient{baseURL: server.URL, token: "test-token", http: server.Client()}
	return probeSource{
		app: client.app(probeAppModule),
		l:   logrus.WithField("test", t.Name()),
	}
}

func TestProbeSource(t *testing.T) {
	t.Parallel()

	t.Run("facts arrive typed, with their own age", func(t *testing.T) {
		t.Parallel()

		var gotAuth, gotPath string
		source := newProbeSource(t, func(w http.ResponseWriter, r *http.Request) {
			gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
			_, _ = w.Write([]byte(servicesBody))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, "Bearer test-token", gotAuth)
		assert.Equal(t, "/api/apps/"+probeAppModule+"/"+probeServicesPath, gotPath)
		assert.Equal(t, SourcePartial, result.Status, "one of two services was covered")

		byField := make(map[string]any, len(result.Facts))
		for _, fact := range result.Facts {
			assert.Equal(t, sourceProbe, fact.Source)
			require.NotNil(t, fact.ObservedAt, "a probe fact has to be datable")
			byField[fact.Field] = fact.Value
		}
		assert.Equal(t, "7.0.40-22", byField[fieldInstalledVersion])
		assert.Equal(t, "/etc/mongod-node.conf", byField[fieldConfigPath])
		assert.InDelta(t, 10295.0, byField["uptime_seconds"], 0.001, "a JSON number decodes to float64")
		assert.Equal(t, true, byField["server_running"])
		// collected_at dates the document, and its age is already on every fact.
		assert.NotContains(t, byField, observedCollectedAt)
	})

	t.Run("facts for services this PMM does not have are counted, not merged", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(servicesBody))
		})

		result := source.collect(context.Background(), probeTestServices())

		for _, fact := range result.Facts {
			assert.NotEqual(t, "gone", fact.Service)
		}
		assert.Equal(t, "1", detailStrings(result.Detail)["services_unknown"])
	})

	t.Run("an unconfigured source is disabled, not failed", func(t *testing.T) {
		t.Parallel()

		source := probeSource{l: logrus.WithField("test", t.Name())}
		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourceDisabled, result.Status)
		assert.Empty(t, result.Facts)
		assert.Empty(t, result.Errors)
	})

	t.Run("an unreachable app fails the source and not the run", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourceFailed, result.Status)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "probe_fetch_failed", result.Errors[0].Code)
		assert.Empty(t, result.Facts)
	})

	t.Run("an estate with no rows is partial and empty", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"items": [], "total": 0, "offset": 0, "limit": 200}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourcePartial, result.Status, "nothing covered, but nothing failing either")
		assert.Empty(t, result.Facts)
		assert.Empty(t, result.Errors)
	})

	t.Run("a service seen but never probed contributes no facts", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(unprobedBody))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Empty(t, result.Facts)
		assert.Equal(t, SourcePartial, result.Status)
		assert.Equal(t, "0", detailStrings(result.Detail)["services_covered"])
	})

	t.Run("an estate that is entirely failing fails the source and says why", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"items": [
			  {"service_id":"s1","node_id":"n1","name":"mongo-1","observed":{},
			   "first_seen_at":"2026-08-11T10:00:00Z","last_attempt_at":"2026-08-11T19:53:44Z",
			   "last_success_at":null,"failing_since":"2026-08-09T19:53:44Z",
			   "consecutive_failures":37,"last_error":"Cannot connect to host localhost:8000"},
			  {"service_id":"s2","node_id":"n2","name":"mongo-2","observed":{},
			   "first_seen_at":"2026-08-11T10:00:00Z","last_attempt_at":"2026-08-11T19:53:44Z",
			   "last_success_at":null,"failing_since":"2026-08-09T19:53:44Z",
			   "consecutive_failures":37,"last_error":"Cannot connect to host localhost:8000"}],
			  "total": 2, "offset": 0, "limit": 200}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourceFailed, result.Status)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "probe_all_failing", result.Errors[0].Code)
		assert.Contains(t, result.Errors[0].Message, "Cannot connect to host")
		assert.Contains(t, result.Errors[0].Message, "mongo-1")
		assert.Equal(t, "2", detailStrings(result.Detail)["services_failing"])
	})

	t.Run("a failure with no recorded reason still says since when", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"items": [{"service_id":"s1","node_id":"n1","name":"mongo-1","observed":{},
			  "first_seen_at":"2026-08-11T10:00:00Z","last_attempt_at":"2026-08-11T19:53:44Z",
			  "last_success_at":null,"failing_since":"2026-08-09T19:53:44Z",
			  "consecutive_failures":37,"last_error":null}],
			  "total": 1, "offset": 0, "limit": 200}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourceFailed, result.Status)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "failing since")
	})

	t.Run("some covered and some failing is partial", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"items": [
			  {"service_id":"s1","node_id":"n1","name":"mongo-1",
			   "observed":{"collected_at":"2026-08-11T11:32:30+00:00","installed_version":"7.0.40-22"},
			   "first_seen_at":"2026-08-11T10:00:00Z","last_attempt_at":"2026-08-11T11:32:30Z",
			   "last_success_at":"2026-08-11T11:32:30Z","failing_since":null,
			   "consecutive_failures":0,"last_error":null},
			  {"service_id":"s2","node_id":"n2","name":"mongo-2","observed":{},
			   "first_seen_at":"2026-08-11T10:00:00Z","last_attempt_at":"2026-08-11T11:32:30Z",
			   "last_success_at":null,"failing_since":"2026-08-11T11:00:00Z",
			   "consecutive_failures":2,"last_error":"unreachable"}],
			  "total": 2, "offset": 0, "limit": 200}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourcePartial, result.Status)
		assert.Len(t, result.Facts, 1)
		assert.Equal(t, "1", detailStrings(result.Detail)["services_failing"])
	})

	t.Run("garbage does not become facts", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("this is not json"))
		})

		result := source.collect(context.Background(), probeTestServices())
		assert.Equal(t, SourceFailed, result.Status)
		assert.Empty(t, result.Facts)
	})
}

func TestProbeFactsReachTheDocument(t *testing.T) {
	t.Parallel()

	source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(servicesBody))
	})
	services := probeTestServices()
	observed := projectionNow.Add(-5 * time.Second)

	metrics := SourceResult{Source: sourceMetrics, Facts: []Fact{
		{Service: "s1", Field: fieldExporterUp, Value: 1.0, Source: sourceMetrics, ObservedAt: &observed},
		{Service: "s1", Field: fieldVersion, Value: "7.0.39-21", Source: sourceMetrics, ObservedAt: &observed},
	}}
	probe := source.collect(context.Background(), services)

	merged := mergeFacts([]SourceResult{metrics, probe}, defaultPrecedence)
	doc := buildDocument(services, merged, projectionNow, projectionMaxAge)

	var svc *omv1.TopologyService
	for _, cluster := range doc.environments[0].Clusters {
		for _, candidate := range cluster.Services {
			if candidate.ServiceName == "mongo-1" {
				svc = candidate
			}
		}
	}
	require.NotNil(t, svc)

	// Upgraded but not restarted: the binary on disk is ahead of the running server.
	assert.Equal(t, "7.0.40-22", svc.GetInstalledVersion())
	assert.Equal(t, "7.0.39-21", svc.GetVersion(), "metrics still own the running version")
	assert.Equal(t, "/etc/mongod-node.conf", svc.GetConfigPath())
	assert.Contains(t, svc.GetArgv(), "mongod")

	// The service the probe never reached carries nulls, not another service's facts.
	for _, cluster := range doc.environments[0].Clusters {
		for _, candidate := range cluster.Services {
			if candidate.ServiceName == "mongo-2" {
				assert.Nil(t, candidate.InstalledVersion)
				assert.Nil(t, candidate.ConfigPath)
			}
		}
	}
}
