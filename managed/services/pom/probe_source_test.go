// Copyright (C) 2026 Percona LLC
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

package pom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pomv1 "github.com/percona/pmm/api/pom/v1"
	"github.com/percona/pmm/managed/models"
)

// factsBody is one pom_discovery GET /facts answer, shaped as the app really sends it.
const factsBody = `{
  "run_id": "c97781e8-05cc-46a7-bcdc-9ff4711fcb37",
  "status": "success",
  "observed_at": "2026-08-11T11:32:30Z",
  "age_seconds": 12.0,
  "stale": false,
  "counts": {"services_total": 2, "services_resolved": 2, "services_orphaned": 0, "services_answered": 2},
  "facts": [
    {"service_id": "s1", "field": "installed_version", "value": "7.0.40-22", "observed_at": "2026-08-11T11:32:30Z"},
    {"service_id": "s1", "field": "config_path", "value": "/etc/mongod-node.conf", "observed_at": "2026-08-11T11:32:30Z"},
    {"service_id": "s1", "field": "argv", "value": "/usr/bin/mongod --config /etc/mongod-node.conf", "observed_at": "2026-08-11T11:32:30Z"},
    {"service_id": "s1", "field": "uptime_seconds", "value": 10295, "observed_at": "2026-08-11T11:32:30Z"},
    {"service_id": "s1", "field": "server_running", "value": true, "observed_at": "2026-08-11T11:32:30Z"},
    {"service_id": "gone", "field": "installed_version", "value": "6.0.1", "observed_at": "2026-08-11T11:32:30Z"}
  ]
}`

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
	return probeSource{
		sepURL: server.URL,
		token:  "test-token",
		client: server.Client(),
		l:      logrus.WithField("test", t.Name()),
	}
}

func TestProbeSource(t *testing.T) {
	t.Parallel()

	t.Run("facts arrive typed, with their own age", func(t *testing.T) {
		t.Parallel()

		var gotAuth, gotPath string
		source := newProbeSource(t, func(w http.ResponseWriter, r *http.Request) {
			gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
			_, _ = w.Write([]byte(factsBody))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, "Bearer test-token", gotAuth)
		// The caller configures where SEP is; this side appends the app path. A path
		// of just "/facts" would mean the app name had leaked into configuration.
		assert.Equal(t, "/"+probeAppPath+"/facts", gotPath)
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
	})

	t.Run("facts for services this PMM does not have are counted, not merged", func(t *testing.T) {
		t.Parallel()

		// The app reads its own inventory, which can be a moment ahead or behind. A
		// fact for a service with no row here has nothing to attach to.
		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(factsBody))
		})

		result := source.collect(context.Background(), probeTestServices())

		for _, fact := range result.Facts {
			assert.NotEqual(t, "gone", fact.Service)
		}
		assert.Equal(t, "1", detailStrings(result.Detail)["services_unknown"])
	})

	t.Run("an unconfigured source is disabled, not failed", func(t *testing.T) {
		t.Parallel()

		// "No probe has run here" is a normal state of a PMM that has not enabled the
		// app, and must not read as an error on the run.
		source := probeSource{l: logrus.WithField("test", t.Name())} // no sepURL
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
		assert.Empty(t, result.Facts)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "probe_fetch_failed", result.Errors[0].Code)
		assert.Equal(t, "source", result.Errors[0].Scope)
	})

	t.Run("an app that has never swept is OK and empty", func(t *testing.T) {
		t.Parallel()

		// The app answers 200 with a null run before its first sweep completes.
		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"run_id":null,"status":null,"observed_at":null,"age_seconds":null,"stale":false,"counts":null,"facts":[]}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourceOK, result.Status)
		assert.Empty(t, result.Facts)
		assert.Empty(t, result.Errors)
	})

	t.Run("a failed sweep fails the source and says why", func(t *testing.T) {
		t.Parallel()

		// The app is reachable and answering; its last sweep is what went wrong. Reporting
		// that as "ok, 0 facts" reads as "there is nothing to probe here", which is the
		// opposite of what happened.
		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"run_id":"5b59a63d","status":"failed","observed_at":"2026-08-11T19:53:44Z",
			  "age_seconds":450.0,"stale":false,"counts":null,"facts":[],
			  "error":"Cannot connect to host localhost:8000"}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourceFailed, result.Status)
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "probe_sweep_failed", result.Errors[0].Code)
		// The cause travels with it, so the receipt is readable without going to ask
		// the other service.
		assert.Contains(t, result.Errors[0].Message, "Cannot connect to host")
		assert.Contains(t, result.Errors[0].Message, "5b59a63d")
	})

	t.Run("a failed sweep with no recorded reason still says so", func(t *testing.T) {
		t.Parallel()

		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"run_id":"abc","status":"failed","facts":[],"error":null}`))
		})

		result := source.collect(context.Background(), probeTestServices())
		assert.Equal(t, SourceFailed, result.Status)
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "no reason recorded")
	})

	t.Run("a partial sweep is partial even when it covered everything it reached", func(t *testing.T) {
		t.Parallel()

		// The app reached fewer nodes than it wanted to. Its own verdict carries, rather
		// than being recomputed from a coverage count that cannot see what it attempted.
		source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"run_id":"abc","status":"partial","facts":[
			  {"service_id":"s1","field":"installed_version","value":"7.0.40-22","observed_at":"2026-08-11T11:32:30Z"},
			  {"service_id":"s2","field":"installed_version","value":"7.0.40-22","observed_at":"2026-08-11T11:32:30Z"}]}`))
		})

		result := source.collect(context.Background(), probeTestServices())

		assert.Equal(t, SourcePartial, result.Status, "both services covered, but the sweep knows it fell short")
		assert.Len(t, result.Facts, 2)
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

	// End to end through the seam: what the app serves, merged under the precedence
	// table, projected into the document.
	source := newProbeSource(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(factsBody))
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

	var svc *pomv1.Service
	for _, cluster := range doc.environments[0].Clusters {
		for _, candidate := range cluster.Services {
			if candidate.ServiceName == "mongo-1" {
				svc = candidate
			}
		}
	}
	require.NotNil(t, svc)

	// The upgraded-but-not-restarted case, which is the whole reason the probe exists:
	// the binary on disk is ahead of the server that is running.
	assert.Equal(t, "7.0.40-22", svc.InstalledVersion.GetValue())
	assert.Equal(t, "7.0.39-21", svc.Version.GetValue(), "metrics still own the running version")
	assert.Equal(t, "/etc/mongod-node.conf", svc.ConfigPath.GetValue())
	assert.Contains(t, svc.Argv.GetValue(), "mongod")

	// And the service the probe never reached carries nulls rather than another's facts.
	for _, cluster := range doc.environments[0].Clusters {
		for _, candidate := range cluster.Services {
			if candidate.ServiceName == "mongo-2" {
				assert.Nil(t, candidate.InstalledVersion)
				assert.Nil(t, candidate.ConfigPath)
			}
		}
	}
}
