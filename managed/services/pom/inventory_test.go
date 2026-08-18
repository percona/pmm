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

package pom

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pomv1 "github.com/percona/pmm/api/pom/v1"
)

// hostsBody is one pom_discovery GET /hosts answer.
//
// Two hosts on purpose, because the pair is the whole reason the host table exists.
// `n1` runs a registered database and reports an unregistered mongod beside it -- the
// arbiter case, which PMM registers no service for, so nothing else in POM would mention
// the process. `n2` carries a PMM client and no database at all, which is the case a
// service-keyed inventory cannot represent.
const hostsBody = `[
  {
    "node_id": "n1", "name": "db00", "address": "10.0.0.1", "executor_host": "db00",
    "observed": {
      "collected_at": "2026-08-18T09:00:00+00:00",
      "os": "Ubuntu 24.04.3 LTS",
      "kernel": "6.17.0-35-generic",
      "executor": {"registered": true, "reachable": true, "driver_healthy": true, "detail": null},
      "unregistered_mongods": [
        {"pid": 10, "port": 27018, "program": "mongod",
         "argv": "/usr/bin/mongod --config /etc/mongod-node.conf",
         "config_path": "/etc/mongod-node.conf"}
      ]
    },
    "first_seen_at": "2026-08-18T08:00:00Z", "last_attempt_at": "2026-08-18T09:00:00Z",
    "last_success_at": "2026-08-18T09:00:00Z", "failing_since": null,
    "consecutive_failures": 0, "last_error": null,
    "services": [
      {
        "service_id": "s1", "node_id": "n1", "name": "mongo-1", "port": 27017, "role": "PRIMARY",
        "observed": {
          "collected_at": "2026-08-18T09:00:00+00:00",
          "installed_version": "7.0.40-22", "version": "7.0.39-21",
          "config_path": "/etc/mongod-node.conf",
          "argv": "/usr/bin/mongod --config /etc/mongod-node.conf",
          "probe_status": "ok", "server_running": true, "uptime_seconds": 11699,
          "replication_set": "rs0"
        },
        "first_seen_at": "2026-08-18T08:00:00Z", "last_attempt_at": "2026-08-18T09:00:00Z",
        "last_success_at": "2026-08-18T09:00:00Z", "failing_since": null,
        "consecutive_failures": 0, "last_error": null
      }
    ]
  },
  {
    "node_id": "n2", "name": "pmm-client-node00", "address": "10.0.0.2", "executor_host": null,
    "observed": {},
    "first_seen_at": "2026-08-18T08:00:00Z", "last_attempt_at": "2026-08-18T09:00:00Z",
    "last_success_at": null, "failing_since": "2026-08-18T08:30:00Z",
    "consecutive_failures": 3, "last_error": "no executor host",
    "services": []
  }
]`

// configBody is one pom_discovery GET /config answer, trimmed to the two rows that make
// the point: a nested schedule leaf and a field the deployment owns outright.
const configBody = `[
  {"key": "SCHEDULE__every", "value": 10, "default_value": null, "type": "int",
   "reload": "hot", "has_override": false, "is_advanced": false, "description": null},
  {"key": "CREDENTIALS_PATH", "value": null, "default_value": null, "type": "str",
   "reload": "not_overridable", "has_override": false, "is_advanced": false,
   "description": null}
]`

// sepStub stands in for the discovery app, recording what it was asked.
type sepStub struct {
	server *httptest.Server

	method string
	path   string
	query  string
	body   string
}

// newSEPStub serves one canned answer and records the request that fetched it.
func newSEPStub(t *testing.T, code int, body string) *sepStub {
	t.Helper()

	stub := &sepStub{}
	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stub.method = r.Method
		stub.path = r.URL.Path
		stub.query = r.URL.RawQuery
		if raw, err := io.ReadAll(r.Body); err == nil { //nolint:noinlineerr
			stub.body = string(raw)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(stub.server.Close)
	return stub
}

// service returns a POM service wired to the stub.
func (s *sepStub) service(t *testing.T) *Service {
	t.Helper()

	svc := &Service{l: logrus.WithField("test", t.Name())}
	return svc.WithProbeSource(s.server.URL, "test-token")
}

func TestInventoryNotConfigured(t *testing.T) {
	t.Parallel()

	// An unconfigured SEP is a deployment that has not been told where SEP is, not a
	// missing feature and not a broken app. FailedPrecondition says so; NotFound would
	// read as "there is no such endpoint" and send the reader looking for a version
	// problem.
	svc := &Service{l: logrus.WithField("test", t.Name())}

	_, err := svc.ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestListInventoryHosts(t *testing.T) {
	t.Parallel()

	t.Run("projects both hosts", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		require.Len(t, response.GetHosts(), 2)
		assert.Equal(t, "/api/apps/pom_discovery/hosts", stub.path)
	})

	t.Run("promotes the attributes a table sorts by", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		host := response.GetHosts()[0]
		assert.Equal(t, "n1", host.GetNodeId())
		assert.Equal(t, "Ubuntu 24.04.3 LTS", host.GetOs().GetValue())
		assert.Equal(t, "6.17.0-35-generic", host.GetKernel().GetValue())
		assert.Equal(t, "db00", host.GetExecutorHost().GetValue())
	})

	t.Run("carries the whole document for a detail panel", func(t *testing.T) {
		t.Parallel()

		// The app stores observations as JSON so a new attribute is a payload change
		// rather than a schema change. Enumerating every attribute as a proto field
		// would put that coupling straight back, so anything without a field of its own
		// still has to arrive.
		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		observed := response.GetHosts()[0].GetObserved().GetFields()
		assert.Contains(t, observed, "unregistered_mongods")
		assert.NotContains(t, observed, observedCollectedAt,
			"collected_at is metadata about the document; freshness already carries the instant")
	})

	t.Run("reports an unregistered mongod on its host", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		strangers := response.GetHosts()[0].GetUnregisteredMongods()
		require.Len(t, strangers, 1)
		assert.Equal(t, int32(27018), strangers[0].GetPort().GetValue())
		assert.Equal(t, "/etc/mongod-node.conf", strangers[0].GetConfigPath().GetValue())
	})

	t.Run("splits the executor state three ways", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		executor := response.GetHosts()[0].GetExecutor()
		require.NotNil(t, executor)
		assert.True(t, executor.GetRegistered())
		assert.True(t, executor.GetReachable())
		assert.True(t, executor.GetDriverHealthy())
	})

	t.Run("a host with no probe reports absent, not false", func(t *testing.T) {
		t.Parallel()

		// Three false flags would claim SEP looked and the answer was no. A nil block
		// says this sweep did not say, which is what an empty document means.
		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		empty := response.GetHosts()[1]
		assert.Nil(t, empty.GetExecutor())
		assert.Nil(t, empty.GetObserved())
		assert.Empty(t, empty.GetServices(), "a host with no database is a row, not an omission")
	})

	t.Run("a failing host carries why and since when", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		freshness := response.GetHosts()[1].GetFreshness()
		assert.Equal(t, int32(3), freshness.GetConsecutiveFailures())
		assert.Equal(t, "no executor host", freshness.GetLastError().GetValue())
		assert.NotNil(t, freshness.GetFailingSince())
		assert.Nil(t, freshness.GetLastSuccessAt(),
			"never having answered is different from having answered nothing")
	})

	t.Run("passes the filters through", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `[]`)

		_, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{
			HasService: wrapperBool(false),
			Failing:    wrapperBool(true),
			Executor:   wrapperString("db00"),
		})

		require.NoError(t, err)
		assert.Contains(t, stub.query, "has_service=false")
		assert.Contains(t, stub.query, "failing=true")
		assert.Contains(t, stub.query, "executor=db00")
	})

	t.Run("an unset filter is not sent as false", func(t *testing.T) {
		t.Parallel()

		// has_service=false means "only hosts with no database", which is a very
		// different listing from the default. Sending it because the caller said nothing
		// would silently hide every host that has one.
		stub := newSEPStub(t, http.StatusOK, `[]`)

		_, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		assert.Empty(t, stub.query)
	})
}

func TestInventoryServiceProjection(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, hostsBody)

	response, err := stub.service(t).ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})
	require.NoError(t, err)

	services := response.GetHosts()[0].GetServices()
	require.Len(t, services, 1)
	service := services[0]

	// installed_version against running version is the whole reason the probe exists:
	// their divergence is the upgraded-but-not-restarted case, and no metric carries it.
	assert.Equal(t, "7.0.40-22", service.GetInstalledVersion().GetValue())
	assert.Equal(t, "7.0.39-21", service.GetRunningVersion().GetValue())
	assert.Equal(t, "PRIMARY", service.GetRole().GetValue())
	assert.Equal(t, int32(27017), service.GetPort().GetValue())
	assert.Equal(t, "ok", service.GetProbeStatus().GetValue())
	assert.True(t, service.GetServerRunning().GetValue())
	assert.InDelta(t, 11699.0, service.GetUptimeSeconds().GetValue(), 0.001)
	assert.Equal(t, "rs0", service.GetReplicationSet().GetValue())
}

func TestTriggerInventoryRefresh(t *testing.T) {
	t.Parallel()

	t.Run("passes node ids through untranslated", func(t *testing.T) {
		t.Parallel()

		// The whole argument for keying the estate on PMM's node ID is that no
		// translation step exists. If one appeared here, it would be the bug that
		// argument was meant to prevent.
		stub := newSEPStub(t, http.StatusAccepted,
			`{"run_id": "r1", "status": "running", "started_at": "2026-08-18T09:00:00Z", "scope": ["n1"]}`)

		response, err := stub.service(t).TriggerInventoryRefresh(t.Context(),
			&pomv1.TriggerInventoryRefreshRequest{NodeIds: []string{"n1"}})

		require.NoError(t, err)
		assert.Equal(t, "r1", response.GetRunId())
		assert.Equal(t, []string{"n1"}, response.GetScope())
		assert.JSONEq(t, `{"node_ids": ["n1"]}`, stub.body)
		assert.Equal(t, http.MethodPost, stub.method)
	})

	t.Run("no scope means the whole estate", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusAccepted,
			`{"run_id": "r2", "status": "running", "started_at": "2026-08-18T09:00:00Z", "scope": null}`)

		response, err := stub.service(t).TriggerInventoryRefresh(t.Context(),
			&pomv1.TriggerInventoryRefreshRequest{})

		require.NoError(t, err)
		assert.Empty(t, response.GetScope())
		assert.JSONEq(t, `{"node_ids": []}`, stub.body)
	})

	t.Run("a held host answers 409, not 500", func(t *testing.T) {
		t.Parallel()

		// Conflict is an answer a caller acts on -- wait, or refresh something else --
		// so it has to survive the hop as a conflict. Aborted is the code the gateway
		// renders as 409.
		stub := newSEPStub(t, http.StatusConflict,
			`{"detail": "Probe run abc is already refreshing n1"}`)

		_, err := stub.service(t).TriggerInventoryRefresh(t.Context(),
			&pomv1.TriggerInventoryRefreshRequest{NodeIds: []string{"n1"}})

		require.Error(t, err)
		assert.Equal(t, codes.Aborted, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "already refreshing n1")
	})
}

func TestInventoryErrorMapping(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		code  int
		body  string
		want  codes.Code
		hides bool
	}{
		{
			name: "a missing host stays missing",
			code: http.StatusNotFound,
			body: `{"detail": "No host with node_id n9"}`,
			want: codes.NotFound,
		},
		{
			name: "a validation failure is the caller's fault",
			code: http.StatusUnprocessableEntity,
			body: `{"detail": [{"loc": ["SCHEDULE__every"], "msg": "must be positive"}]}`,
			want: codes.InvalidArgument,
		},
		{
			// PMM's credential being rejected is an operator's problem, not the
			// browser's. Reflecting the app's 401 would tell the browser to
			// re-authenticate against PMM, which would not fix anything.
			name:  "a rejected credential does not read as the caller's",
			code:  http.StatusUnauthorized,
			body:  `{"detail": "Not authenticated"}`,
			want:  codes.Internal,
			hides: true,
		},
		{
			name: "anything else is internal",
			code: http.StatusInternalServerError,
			body: `{"detail": "boom"}`,
			want: codes.Internal,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := newSEPStub(t, tc.code, tc.body)

			_, err := stub.service(t).GetInventoryHost(t.Context(),
				&pomv1.GetInventoryHostRequest{NodeId: "n9"})

			require.Error(t, err)
			assert.Equal(t, tc.want, status.Code(err))
			if tc.hides {
				assert.Contains(t, status.Convert(err).Message(), "PMM's credential")
			}
		})
	}
}

func TestInventoryDeleteSendsDelete(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusNoContent, ``)

	_, err := stub.service(t).DeleteInventoryHost(t.Context(),
		&pomv1.DeleteInventoryHostRequest{NodeId: "n1"})

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, stub.method)
	assert.Equal(t, "/api/apps/pom_discovery/hosts/n1", stub.path)
}

func TestInventoryConfig(t *testing.T) {
	t.Parallel()

	t.Run("reports every field and where its value came from", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, configBody)

		response, err := stub.service(t).GetInventoryConfig(t.Context(), &pomv1.GetInventoryConfigRequest{})

		require.NoError(t, err)
		require.Len(t, response.GetSettings(), 2)

		schedule := response.GetSettings()[0]
		assert.Equal(t, "SCHEDULE__every", schedule.GetKey())
		assert.InDelta(t, 10.0, schedule.GetValue().GetNumberValue(), 0.001)
		assert.Equal(t, "hot", schedule.GetReload())
		assert.False(t, schedule.GetHasOverride())

		// A field the deployment owns outright is listed rather than hidden, so a UI can
		// show it greyed out instead of leaving the reader to wonder where it went.
		assert.Equal(t, "not_overridable", response.GetSettings()[1].GetReload())
	})

	t.Run("a change is passed through as the app will validate it", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, configBody)
		values, err := structpb.NewStruct(map[string]any{"SCHEDULE__every": 25})
		require.NoError(t, err)

		response, err := stub.service(t).UpdateInventoryConfig(t.Context(),
			&pomv1.UpdateInventoryConfigRequest{Values: values})

		require.NoError(t, err)
		assert.Equal(t, http.MethodPatch, stub.method)
		assert.JSONEq(t, `{"SCHEDULE__every": 25}`, stub.body)
		// The applied rows come back rather than an empty acknowledgement, so a caller
		// can render the new value and its new origin without a second request.
		require.NotEmpty(t, response.GetSettings())
		assert.Equal(t, "SCHEDULE__every", response.GetSettings()[0].GetKey())
	})

	t.Run("an empty batch is refused here", func(t *testing.T) {
		t.Parallel()

		// Not forwarded: an empty PATCH would succeed on the app's side and change
		// nothing, so a caller who built the body wrongly would see success.
		stub := newSEPStub(t, http.StatusOK, configBody)

		_, err := stub.service(t).UpdateInventoryConfig(t.Context(),
			&pomv1.UpdateInventoryConfigRequest{})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Empty(t, stub.method, "nothing should have been sent")
	})

	t.Run("a revert names the field in the path", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusNoContent, ``)

		_, err := stub.service(t).DeleteInventoryConfigOverride(t.Context(),
			&pomv1.DeleteInventoryConfigOverrideRequest{Key: "SCHEDULE__every"})

		require.NoError(t, err)
		assert.Equal(t, http.MethodDelete, stub.method)
		assert.Equal(t, "/api/apps/pom_discovery/config/SCHEDULE__every", stub.path)
	})
}

func TestInventoryRunsDefaultLimit(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, `[]`)

	_, err := stub.service(t).ListInventoryRuns(t.Context(), &pomv1.ListInventoryRunsRequest{})

	require.NoError(t, err)
	assert.Equal(t, "limit=20", stub.query)
}

func TestInventoryBearerIsSent(t *testing.T) {
	t.Parallel()

	// The browser holds no SEP token -- that is the point of proxying -- so this hop is
	// the only place the app's credential is presented. Without it every request is a
	// 401 the page cannot explain.
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(server.URL, "test-token")

	_, err := svc.ListInventoryHosts(t.Context(), &pomv1.ListInventoryHostsRequest{})

	require.NoError(t, err)
	assert.Equal(t, "Bearer test-token", seen)
}

// wrapperBool and wrapperString keep the filter cases above readable.
func wrapperBool(value bool) *wrapperspb.BoolValue { return wrapperspb.Bool(value) }

func wrapperString(value string) *wrapperspb.StringValue { return wrapperspb.String(value) }

// ensure the canned bodies stay valid JSON as they are edited.
func TestInventoryFixturesAreValid(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{"hosts": hostsBody, "config": configBody} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var parsed []map[string]any
			require.NoError(t, json.Unmarshal([]byte(body), &parsed))
			assert.NotEmpty(t, parsed)
		})
	}
}
