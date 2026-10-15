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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// hostsBody is one paginated GET /hosts answer: n1 runs a registered database beside an unregistered mongod,
// n2 has a PMM client and no database.
const hostsBody = `{"items": [
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
    "last_error_code": "scan_lost", "last_run_id": "8b9b2f3e-1c4d-4e5f-9a6b-7c8d9e0f1a2b",
    "services": []
  }
], "total": 2, "offset": 0, "limit": 200}`

const configBody = `[
  {"key": "SCHEDULE__every", "value": 10, "default_value": null, "type": "int",
   "reload": "hot", "has_override": false, "is_advanced": false, "description": null},
  {"key": "CREDENTIALS_PATH", "value": null, "default_value": null, "type": "str",
   "reload": "not_overridable", "has_override": false, "is_advanced": false,
   "description": null}
]`

type stubCall struct {
	method string
	path   string
	query  string
	body   string
}

type extensionsStub struct {
	server *httptest.Server

	method string
	path   string
	query  string
	body   string

	// calls records every request in order; the fields above keep only the last.
	calls []stubCall
	// bodies is served one entry per request; the last entry repeats once exhausted.
	bodies []string
}

func newSEPStub(t *testing.T, code int, body string) *extensionsStub {
	t.Helper()

	return newSEPStubSeq(t, code, body)
}

func newSEPStubSeq(t *testing.T, code int, bodies ...string) *extensionsStub {
	t.Helper()

	codes := make([]int, len(bodies))
	for i := range codes {
		codes[i] = code
	}
	return newSEPStubSeqCodes(t, codes, bodies)
}

func newSEPStubSeqCodes(t *testing.T, codes []int, bodies []string) *extensionsStub {
	t.Helper()

	stub := &extensionsStub{bodies: bodies}
	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := stubCall{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery}
		raw, err := io.ReadAll(r.Body)
		if err == nil {
			call.body = string(raw)
		}
		stub.method, stub.path, stub.query, stub.body = call.method, call.path, call.query, call.body
		index := min(len(stub.calls), len(bodies)-1)
		body := ""
		if len(bodies) > 0 {
			body = bodies[index]
		}
		code := http.StatusOK
		if len(codes) > 0 {
			code = codes[min(len(stub.calls), len(codes)-1)]
		}
		stub.calls = append(stub.calls, call)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *extensionsStub) service(t *testing.T) *Service {
	t.Helper()

	svc := &Service{l: logrus.WithField("test", t.Name())}
	return svc.WithProbeSource(s.server.URL, "test-token")
}

func TestInventoryNotConfigured(t *testing.T) {
	t.Parallel()

	// FailedPrecondition, not NotFound: NotFound would read as a missing endpoint and suggest a version problem.
	svc := &Service{l: logrus.WithField("test", t.Name())}

	_, err := svc.ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})

	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestListInventoryHosts(t *testing.T) {
	t.Parallel()

	t.Run("projects both hosts", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		require.Len(t, response.GetHosts(), 2)
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.path)
	})

	t.Run("promotes the attributes a table sorts by", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		host := response.GetHosts()[0]
		assert.Equal(t, "n1", host.GetNodeId())
		assert.Equal(t, "Ubuntu 24.04.3 LTS", host.GetOs())
		assert.Equal(t, "6.17.0-35-generic", host.GetKernel())
		assert.Equal(t, "db00", host.GetExecutorHost())
	})

	t.Run("carries the whole document for a detail panel", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		observed := response.GetHosts()[0].GetObserved().GetFields()
		assert.Contains(t, observed, "unregistered_mongods")
		assert.NotContains(t, observed, observedCollectedAt,
			"collected_at is metadata about the document; freshness already carries the instant")
	})

	t.Run("reports an unregistered mongod on its host", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		strangers := response.GetHosts()[0].GetUnregisteredMongods()
		require.Len(t, strangers, 1)
		assert.Equal(t, int32(27018), strangers[0].GetPort())
		assert.Equal(t, "/etc/mongod-node.conf", strangers[0].GetConfigPath())
	})

	t.Run("splits the executor state three ways", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		executor := response.GetHosts()[0].GetExecutor()
		require.NotNil(t, executor)
		assert.True(t, executor.GetRegistered())
		assert.True(t, executor.GetReachable())
		assert.True(t, executor.GetDriverHealthy())
	})

	t.Run("without an agent registry, no host is eligible", func(t *testing.T) {
		t.Parallel()

		// stub.service(t) wires no agent registry, so even a host with a healthy executor is ineligible.
		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		healthyExecutor := response.GetHosts()[0]
		assert.False(t, healthyExecutor.GetPmmAgentConnected())
		assert.False(t, healthyExecutor.GetAutomationEligible())
		assert.Contains(t, healthyExecutor.GetAutomationBlockedReasons(),
			"PMM Client is not installed or not connected")
		assert.NotContains(t, healthyExecutor.GetAutomationBlockedReasons(),
			"this node has no automation agent that answers",
			"n1's executor is fully healthy -- only the missing agent signal should block it")

		noExecutor := response.GetHosts()[1]
		assert.False(t, noExecutor.GetAutomationEligible())
		assert.Contains(t, noExecutor.GetAutomationBlockedReasons(),
			"PMM Client is not installed or not connected")
		assert.Contains(t, noExecutor.GetAutomationBlockedReasons(),
			"this node has no automation agent that answers")
	})

	t.Run("the automation_eligible filter excludes every host when nothing is eligible", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		eligible := true
		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{
			AutomationEligible: &eligible,
		})
		require.NoError(t, err)
		assert.Empty(t, response.GetHosts(),
			"no host has a connected agent in this stub, so none should pass the filter")
	})

	t.Run("a host with no probe reports absent, not false", func(t *testing.T) {
		t.Parallel()

		// A nil block means the sweep did not report; three false flags would claim it looked and saw no.
		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		empty := response.GetHosts()[1]
		assert.Nil(t, empty.GetExecutor())
		assert.Nil(t, empty.GetObserved())
		assert.Empty(t, empty.GetServices(), "a host with no database is a row, not an omission")
	})

	t.Run("a healthy host carries no failure kind", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		assert.Nil(t, response.GetHosts()[0].GetFreshness().LastErrorCode,
			"unset, not an empty string, so a reader cannot mistake it for a kind")
	})

	t.Run("a failing host carries why and since when", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		freshness := response.GetHosts()[1].GetFreshness()
		assert.Equal(t, int32(3), freshness.GetConsecutiveFailures())
		assert.Equal(t, "no executor host", freshness.GetLastError())
		assert.Equal(t, "scan_lost", freshness.GetLastErrorCode())
		assert.Equal(t, "8b9b2f3e-1c4d-4e5f-9a6b-7c8d9e0f1a2b", freshness.GetLastRunId(),
			"the run that produced the failure, for linking to it")
		assert.NotNil(t, freshness.GetFailingSince())
		assert.Nil(t, freshness.GetLastSuccessAt(),
			"never having answered is different from having answered nothing")
	})

	t.Run("passes the filters through", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"items": [], "total": 0, "offset": 0, "limit": 200}`)

		hasService, failing, executor := false, true, false

		_, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{
			HasService: &hasService,
			Failing:    &failing,
			Executor:   &executor,
		})

		require.NoError(t, err)
		assert.Contains(t, stub.query, "has_service=false")
		assert.Contains(t, stub.query, "failing=true")
		// The app filters on whether an executor is matched at all; a host name here is a 422.
		assert.Contains(t, stub.query, "executor=false")
	})

	t.Run("an unset filter is not sent as false", func(t *testing.T) {
		t.Parallel()

		// has_service=false means "only hosts with no database"; sending it unasked would hide every other host.
		stub := newSEPStub(t, http.StatusOK, `{"items": [], "total": 0, "offset": 0, "limit": 200}`)

		_, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		assert.NotContains(t, stub.query, "has_service")
		assert.NotContains(t, stub.query, "failing")
		assert.NotContains(t, stub.query, "executor")
	})

	t.Run("walks every page", func(t *testing.T) {
		t.Parallel()

		// total exceeds the first page's item count, so fetchAllPages must request a second page.
		firstPage := `{"items": [
		  {"node_id": "n1", "name": "db00"},
		  {"node_id": "n2", "name": "db01"}
		], "total": 3, "offset": 0, "limit": 200}`
		secondPage := `{"items": [
		  {"node_id": "n3", "name": "db02"}
		], "total": 3, "offset": 2, "limit": 200}`
		stub := newSEPStubSeq(t, http.StatusOK, firstPage, secondPage)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		require.Len(t, response.GetHosts(), 3, "both pages' items should be concatenated")
		assert.Equal(t, "n1", response.GetHosts()[0].GetNodeId())
		assert.Equal(t, "n3", response.GetHosts()[2].GetNodeId())

		require.Len(t, stub.calls, 2)
		assert.Contains(t, stub.calls[0].query, "offset=0")
		assert.Contains(t, stub.calls[1].query, "offset=2",
			"the second request's offset should be the first page's item count, not its own offset field")
		assert.Contains(t, stub.calls[1].query, "limit=200")
	})
}

func TestListInventoryServices(t *testing.T) {
	t.Parallel()

	t.Run("reads the paginated envelope, keeping the caller's filters", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"items": [
		  {"service_id": "s1", "node_id": "n1", "name": "mongo-1", "port": 27017, "role": "PRIMARY"},
		  {"service_id": "s2", "node_id": "n1", "name": "mongo-2", "port": 27018, "role": null}
		], "total": 2, "offset": 0, "limit": 200}`)
		nodeID, failing := "n1", true

		response, err := stub.service(t).ListInventoryServices(t.Context(), &omv1.ListInventoryServicesRequest{
			NodeId:  &nodeID,
			Failing: &failing,
		})

		require.NoError(t, err)
		require.Len(t, response.GetServices(), 2)
		assert.Equal(t, "s1", response.GetServices()[0].GetServiceId())
		assert.Equal(t, "/api/apps/om_inventory/services", stub.path)
		assert.Contains(t, stub.query, "node_id=n1")
		assert.Contains(t, stub.query, "failing=true")
	})

	t.Run("walks every page", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(t, http.StatusOK,
			`{"items": [{"service_id": "s1", "node_id": "n1", "name": "mongo-1"}], "total": 2, "offset": 0, "limit": 200}`,
			`{"items": [{"service_id": "s2", "node_id": "n2", "name": "mongo-2"}], "total": 2, "offset": 1, "limit": 200}`)

		response, err := stub.service(t).ListInventoryServices(t.Context(), &omv1.ListInventoryServicesRequest{})

		require.NoError(t, err)
		require.Len(t, stub.calls, 2)
		require.Len(t, response.GetServices(), 2)
		assert.Equal(t, "s1", response.GetServices()[0].GetServiceId())
		assert.Equal(t, "s2", response.GetServices()[1].GetServiceId())
		assert.Contains(t, stub.calls[1].query, "offset=1")
	})

	t.Run("a failing service carries the kind and run of its failure", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"items": [
		  {"service_id": "s1", "node_id": "n1", "name": "mongo-1",
		   "failing_since": "2026-08-18T08:30:00Z", "consecutive_failures": 2,
		   "last_error": "probe timed out", "last_error_code": "timed_out",
		   "last_run_id": "3c1d2e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f"},
		  {"service_id": "s2", "node_id": "n1", "name": "mongo-2"}
		], "total": 2, "offset": 0, "limit": 200}`)

		response, err := stub.service(t).ListInventoryServices(t.Context(), &omv1.ListInventoryServicesRequest{})
		require.NoError(t, err)
		require.Len(t, response.GetServices(), 2)

		failing := response.GetServices()[0].GetFreshness()
		assert.Equal(t, "timed_out", failing.GetLastErrorCode())
		assert.Equal(t, "3c1d2e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f", failing.GetLastRunId())
		assert.Nil(t, response.GetServices()[1].GetFreshness().LastErrorCode)
	})
}

func TestAutomationEligibility(t *testing.T) {
	t.Parallel()

	healthyExecutor := &omv1.InventoryExecutor{Registered: true, Reachable: true, DriverHealthy: true}
	reachableOnlyExecutor := &omv1.InventoryExecutor{Registered: true, Reachable: true, DriverHealthy: false}

	// installable returns a host that passes every check, so each case isolates the one condition it names.
	installable := func(mutate ...func(*extensionsHost)) extensionsHost {
		address := "node-1.example"
		host := extensionsHost{
			Address:  &address,
			Observed: map[string]any{"os_id": "ubuntu"},
		}
		for _, m := range mutate {
			m(&host)
		}
		return host
	}

	t.Run("connected agent and healthy executor is eligible with no reasons", func(t *testing.T) {
		t.Parallel()

		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, installable())

		assert.True(t, eligible)
		assert.Empty(t, reasons)
		assert.False(t, byDesign)
	})

	t.Run("a disconnected agent blocks even a healthy executor", func(t *testing.T) {
		t.Parallel()

		eligible, reasons, byDesign := automationEligibility(healthyExecutor, false, false, installable())

		assert.False(t, eligible)
		assert.False(t, byDesign, "a disconnected agent is a fault, not a property of the node")
		assert.Equal(t, []string{"PMM Client is not installed or not connected"}, reasons)
	})

	t.Run("no executor block at all blocks on reachability, not driver health", func(t *testing.T) {
		t.Parallel()

		// A nil executor means the host was never dispatched to; a driver-health reason would
		// claim a health check ran when none did.
		eligible, reasons, byDesign := automationEligibility(nil, true, false, installable())

		assert.False(t, eligible)
		assert.False(t, byDesign)
		assert.Equal(t, []string{"this node has no automation agent that answers"}, reasons)
	})

	t.Run("reachable but unhealthy driver blocks on the driver, not reachability", func(t *testing.T) {
		t.Parallel()

		eligible, reasons, byDesign := automationEligibility(reachableOnlyExecutor, true, false, installable())

		assert.False(t, eligible)
		assert.False(t, byDesign)
		assert.Equal(t, []string{"this node's automation agent cannot run jobs"}, reasons)
	})

	// The reasons are product copy shown on the Nodes page, which must not name the scheduler.
	t.Run("no reason names the scheduler", func(t *testing.T) {
		t.Parallel()

		for _, executor := range []*omv1.InventoryExecutor{nil, reachableOnlyExecutor} {
			_, reasons, _ := automationEligibility(executor, false, false, extensionsHost{})
			for _, reason := range reasons {
				assert.NotContains(t, strings.ToLower(reason), "nomad")
				assert.NotContains(t, strings.ToLower(reason), "raw_exec")
			}
		}
	})

	t.Run("every reachability condition unmet reports every reason", func(t *testing.T) {
		t.Parallel()

		eligible, reasons, byDesign := automationEligibility(nil, false, false, installable())

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"PMM Client is not installed or not connected",
			"this node has no automation agent that answers",
		}, reasons)
		assert.False(t, byDesign)
	})

	// The cases below ask whether OM should touch a node, not whether it can reach it.
	t.Run("the PMM Server's own node is never eligible, however healthy", func(t *testing.T) {
		t.Parallel()

		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, true, installable())

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"this is the node PMM Server itself runs on, which Operations never installs onto",
		}, reasons)
		assert.True(t, byDesign, "the server's own node is a fact about it, not a fault on it")
	})

	// PMM Server's image reports os_id "ol"; an OS reason as well would suggest a different OS would help.
	t.Run("the PMM Server's node reports that reason alone", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) {
			h.Address = nil
			h.Observed = map[string]any{"os_id": "ol"}
			h.Services = []extensionsService{{ServiceID: "30"}}
		})
		eligible, reasons, byDesign := automationEligibility(nil, false, true, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"this is the node PMM Server itself runs on, which Operations never installs onto",
		}, reasons)
		assert.True(t, byDesign)
	})

	t.Run("a node with a registered MongoDB service is not eligible", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) {
			h.Services = []extensionsService{{ServiceID: "30"}}
		})
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"a MongoDB service is already registered on this node",
		}, reasons)
		assert.True(t, byDesign)
	})

	t.Run("a mongod a scan found but PMM has no service for also blocks", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) {
			h.Observed["unregistered_mongods"] = []any{map[string]any{"port": 27017}}
		})
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"a scan found a mongod running here that PMM has no service for",
		}, reasons)
		assert.True(t, byDesign)
	})

	// Installed but stopped: only the scan's reading of the binary shows it, and an install would leave two mongods.
	t.Run("MongoDB a scan found installed but not running also blocks", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) {
			h.Observed["installed_version"] = "7.0.43-23"
		})
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"a scan found MongoDB 7.0.43-23 already installed on this node",
		}, reasons)
		assert.True(t, byDesign)
	})

	// The registered service and the mongod serving it are one problem, so they get one reason.
	t.Run("a registered service suppresses the unregistered-mongod reason", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) {
			h.Services = []extensionsService{{ServiceID: "30"}}
			h.Observed["unregistered_mongods"] = []any{map[string]any{"port": 27017}}
		})
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"a MongoDB service is already registered on this node",
		}, reasons)
		assert.True(t, byDesign)
	})

	t.Run("a node no scan has reported an OS for is not eligible", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) { h.Observed = map[string]any{} })
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"no scan has reported this node's operating system yet",
		}, reasons)
		assert.False(t, byDesign, "a missing OS means scans are not landing, which is a fault")
	})

	t.Run("an OS om_bootstrap cannot install onto is not eligible, and is by design", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) { h.Observed = map[string]any{"os_id": "debian"} })
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"this node runs debian, which Operations cannot install onto (supported: rocky, ubuntu)",
		}, reasons)
		assert.True(t, byDesign, "nothing is wrong with a Debian node; it is simply not a target")
	})

	// The reason lists the supported distributions, so they must be the ones the trigger accepts.
	t.Run("the supported list matches what the trigger accepts", func(t *testing.T) {
		t.Parallel()

		for id := range supportedBootstrapOSIDs {
			assert.Contains(t, supportedBootstrapOSNames(), id)
		}
	})

	t.Run("a node PMM has no address for is not eligible", func(t *testing.T) {
		t.Parallel()

		host := installable(func(h *extensionsHost) { h.Address = nil })
		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{"PMM has no address for this node"}, reasons)
		assert.False(t, byDesign)
	})

	t.Run("an empty address counts as no address", func(t *testing.T) {
		t.Parallel()

		empty := ""
		host := installable(func(h *extensionsHost) { h.Address = &empty })
		eligible, reasons, _ := automationEligibility(healthyExecutor, true, false, host)

		assert.False(t, eligible)
		assert.Equal(t, []string{"PMM has no address for this node"}, reasons)
	})
}

func TestInventoryServiceProjection(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, hostsBody)

	response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
	require.NoError(t, err)

	services := response.GetHosts()[0].GetServices()
	require.Len(t, services, 1)
	service := services[0]

	// Diverging installed and running versions mean upgraded but not restarted, which no metric carries.
	assert.Equal(t, "7.0.40-22", service.GetInstalledVersion())
	assert.Equal(t, "7.0.39-21", service.GetRunningVersion())
	assert.Equal(t, "PRIMARY", service.GetRole())
	assert.Equal(t, int32(27017), service.GetPort())
	assert.Equal(t, "ok", service.GetProbeStatus())
	assert.True(t, service.GetServerRunning())
	assert.InDelta(t, 11699.0, service.GetUptimeSeconds(), 0.001)
	assert.Equal(t, "rs0", service.GetReplicationSet())
}

func TestTriggerInventoryRefresh(t *testing.T) {
	t.Parallel()

	t.Run("passes node ids through untranslated", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusAccepted,
			`{"run_id": "r1", "status": "running", "started_at": "2026-08-18T09:00:00Z", "scope": ["n1"]}`)

		response, err := stub.service(t).TriggerInventoryRefresh(t.Context(),
			&omv1.TriggerInventoryRefreshRequest{NodeIds: []string{"n1"}})

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
			&omv1.TriggerInventoryRefreshRequest{})

		require.NoError(t, err)
		assert.Empty(t, response.GetScope())
		assert.JSONEq(t, `{"node_ids": []}`, stub.body)
	})

	t.Run("a held host answers 409, not 500", func(t *testing.T) {
		t.Parallel()

		// The gateway renders Aborted as 409.
		stub := newSEPStub(t, http.StatusConflict,
			`{"detail": "Probe run abc is already refreshing n1"}`)

		_, err := stub.service(t).TriggerInventoryRefresh(t.Context(),
			&omv1.TriggerInventoryRefreshRequest{NodeIds: []string{"n1"}})

		require.Error(t, err)
		assert.Equal(t, codes.Aborted, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "already refreshing n1")
	})
}

func TestTriggerHostBootstrap(t *testing.T) {
	t.Parallel()

	t.Run("reads the host's OS from inventory, then plans a run with om_bootstrap", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				DataPath:       "/var/lib/mongo",
				LogPath:        "/var/log/mongodb/mongod.log",
				Port:           27017,
				BindIp:         "0.0.0.0",
			})

		require.NoError(t, err)
		assert.Equal(t, "run-abc", response.GetRunId())
		require.Len(t, stub.calls, 2)
		assert.Equal(t, "/api/apps/om_inventory/hosts/n1", stub.calls[0].path)
		assert.Equal(t, "/api/apps/om_bootstrap/runs", stub.calls[1].path)
		// hosts carries n1's executor host, not its node ID; this fixture just gives them the same value.
		assert.JSONEq(t,
			`{"hosts": ["n1"], "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8",
			  "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo",
			  "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0"}`,
			stub.calls[1].body)
	})

	t.Run("a host with no known OS yet answers FailedPrecondition, not 500", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"node_id": "n1", "executor_host": "n1", "observed": {}}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		assert.Contains(t, message, "no scan has reported its operating system")
		// Waiting is the wrong advice when scans may have been failing for many runs.
		assert.NotContains(t, message, "wait")
	})

	t.Run("a host running an unsupported OS answers FailedPrecondition, not a run that starts and fails", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "windows"}}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "does not support")
		// Only the host lookup: rejected before reaching om_bootstrap.
		require.Len(t, stub.calls, 1)
	})

	t.Run("a host with no usable executor answers FailedPrecondition, not 500", func(t *testing.T) {
		t.Parallel()

		// No executor_host: falling back to node_id would instead fail inside PMM Extensions,
		// once every host is in flight.
		stub := newSEPStub(t, http.StatusOK, `{"node_id": "n1", "observed": {"os_id": "ubuntu"}}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		assert.Contains(t, message, "no automation agent is registered")
		assert.NotContains(t, message, "Nomad")
	})

	t.Run("an unreachable executor answers FailedPrecondition before a run exists", func(t *testing.T) {
		t.Parallel()

		// Without this check the run is created first and an unreachable client only surfaces
		// when pre_check fails, with the UI already showing the run as in progress.
		stub := newSEPStub(t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu",
			  "executor": {"reachable": false, "driver_healthy": true}}}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "not reachable")
		assert.NotContains(t, status.Convert(err).Message(), "Nomad")
		require.Len(t, stub.calls, 1, "no run should be planned")
		assert.Equal(t, "/api/apps/om_inventory/hosts/n1", stub.calls[0].path)
	})

	t.Run("an unhealthy executor driver answers FailedPrecondition", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu",
			  "executor": {"reachable": true, "driver_healthy": false}}}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "driver is not healthy")
		assert.NotContains(t, status.Convert(err).Message(), "Nomad")
	})

	t.Run("a healthy executor plans the run", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu",
			  "executor": {"reachable": true, "driver_healthy": true}}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.NoError(t, err)
		assert.Equal(t, "run-abc", response.GetRunId())
	})

	t.Run("plans a three-host run when every host runs the same OS", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n2", "executor_host": "n2", "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n3", "executor_host": "n3", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				DataPath:       "/var/lib/mongo",
				LogPath:        "/var/log/mongodb/mongod.log",
				Port:           27017,
				BindIp:         "0.0.0.0",
			})

		require.NoError(t, err)
		assert.Equal(t, "run-abc", response.GetRunId())
		require.Len(t, stub.calls, 4)
		assert.JSONEq(t,
			`{"hosts": ["n1", "n2", "n3"], "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8",
			  "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo",
			  "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0"}`,
			stub.calls[3].body)
	})

	t.Run("rejects two hosts -- phase-1 supports one or three, not two", func(t *testing.T) {
		t.Parallel()

		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithProbeSource("http://unused.invalid", "").
			WithBootstrapSource("http://unused.invalid", "")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "one or three")
	})

	t.Run("rejects a mixed-OS selection", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n2", "executor_host": "n2", "observed": {"os_id": "rocky"}}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "mixed-OS")
	})

	t.Run("rejects member_configs naming a host outside node_ids", func(t *testing.T) {
		t.Parallel()

		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithProbeSource("http://unused.invalid", "").
			WithBootstrapSource("http://unused.invalid", "")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				MemberConfigs:  map[string]*omv1.BootstrapMemberConfig{"n2": {}},
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "not in node_ids")
	})

	t.Run("rejects a delayed member that still has priority or a vote", func(t *testing.T) {
		t.Parallel()

		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithProbeSource("http://unused.invalid", "").
			WithBootstrapSource("http://unused.invalid", "")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {DelaySecs: 300, Priority: new(uint32(1)), Votes: new(true)},
				},
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "priority 0 and votes off")
	})

	t.Run("names every unready node and every problem, not the first", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1", "observed": {}}`,
			`{"node_id": "n2", "name": "db-02", "executor_host": "exec-n2", "observed": {"os_id": "windows"}}`,
			// No automation agent and no OS: two problems on one node.
			`{"node_id": "n3", "name": "db-03", "observed": {}}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		for _, name := range []string{"db-01", "db-02", "db-03"} {
			assert.Contains(t, message, name)
		}
		assert.Contains(t, message, "no automation agent is registered")
		assert.Contains(t, message, "windows")
		assert.Contains(t, message, "Nodes page")
		assert.NotContains(t, message, "wait")
		// Nothing was planned.
		require.Len(t, stub.calls, 3)
	})

	t.Run("names both OS groups when the selection is mixed", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n2", "name": "db-02", "executor_host": "exec-n2", "observed": {"os_id": "rocky"}}`,
			`{"node_id": "n3", "name": "db-03", "executor_host": "exec-n3", "observed": {"os_id": "ubuntu"}}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		message := status.Convert(err).Message()
		assert.Contains(t, message, "mixed-OS")
		assert.Contains(t, message, "rocky on db-02")
		assert.Contains(t, message, "ubuntu on db-01, db-03")
	})

	// automation_eligible is advisory and may be stale, and a direct API call never reads it,
	// so the trigger refuses these nodes itself.
	t.Run("refuses a node that already has a registered MongoDB service", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK,
			`{"node_id": "n1", "name": "rs-member-00", "executor_host": "exec-n1",
			  "observed": {"os_id": "ubuntu"}, "services": [{"service_id": "s1"}]}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		assert.Contains(t, message, "already registered")
		// By name, not by node ID: a UUID is not what anyone calls the machine.
		assert.Contains(t, message, "rs-member-00")
		assert.NotContains(t, message, "n1:")
		// Rejected before om_bootstrap is ever asked to plan anything.
		require.Len(t, stub.calls, 1)
	})

	t.Run("refuses a node a scan found a mongod on", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK,
			`{"node_id": "n1", "name": "db-07", "executor_host": "exec-n1",
			  "observed": {"os_id": "ubuntu", "unregistered_mongods": [{"port": 27017}]}}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "db-07")
		assert.Contains(t, status.Convert(err).Message(), "no service for")
		require.Len(t, stub.calls, 1)
	})

	t.Run("names every blocking node, not just the first", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1",
			  "observed": {"os_id": "ubuntu"}, "services": [{"service_id": "s1"}]}`,
			`{"node_id": "n2", "name": "db-02", "executor_host": "exec-n2",
			  "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n3", "name": "db-03", "executor_host": "exec-n3",
			  "observed": {"os_id": "ubuntu", "unregistered_mongods": [{"port": 27017}]}}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		assert.Contains(t, message, "db-01")
		assert.Contains(t, message, "db-03")
		assert.NotContains(t, message, "db-02")
		// Three host lookups and no trigger call.
		require.Len(t, stub.calls, 3)
	})

	t.Run("falls back to the node id when the host has no name", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "exec-n1",
			  "observed": {"os_id": "ubuntu"}, "services": [{"service_id": "s1"}]}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})

		require.Error(t, err)
		assert.Contains(t, status.Convert(err).Message(), "n1")
	})

	// rs.initiate() refuses a hidden member with priority above 0, so a direct API call must be refused too.
	t.Run("rejects a hidden member that could still be elected", func(t *testing.T) {
		t.Parallel()

		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithProbeSource("http://unused.invalid", "").
			WithBootstrapSource("http://unused.invalid", "")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					// Unset priority means MongoDB's default of 1.
					"n1": {Hidden: true},
				},
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "priority 0")
	})

	t.Run("rejects a hidden member with priority above zero", func(t *testing.T) {
		t.Parallel()

		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithProbeSource("http://unused.invalid", "").
			WithBootstrapSource("http://unused.invalid", "")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {Hidden: true, Priority: new(uint32(2))},
				},
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "priority 0")
	})

	t.Run("returns every member violation in one error", func(t *testing.T) {
		t.Parallel()

		err := validateMemberConfigs(
			[]string{"n1", "n2", "n3"},
			map[string]*omv1.BootstrapMemberConfig{
				"n1": {DelaySecs: 300},
				"n2": {Hidden: true},
				"n9": {},
			},
		)

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		message := status.Convert(err).Message()
		assert.Contains(t, message, "priority 0 and votes off")
		assert.Contains(t, message, "clients cannot see")
		assert.Contains(t, message, "not in node_ids")
	})

	t.Run("orders the violations the same way every time", func(t *testing.T) {
		t.Parallel()

		configs := map[string]*omv1.BootstrapMemberConfig{
			"n1": {DelaySecs: 300},
			"n2": {Hidden: true},
			"n3": {DelaySecs: 300},
		}
		first := status.Convert(validateMemberConfigs([]string{"n1", "n2", "n3"}, configs)).Message()
		for range 8 {
			again := status.Convert(validateMemberConfigs([]string{"n1", "n2", "n3"}, configs)).Message()
			require.Equal(t, first, again)
		}
	})

	// Checked directly: a one-member set with priority 0 is refused by the no-electable-member rule instead.
	t.Run("accepts a hidden member that cannot be elected", func(t *testing.T) {
		t.Parallel()

		err := validateMemberConfigs(
			[]string{"n1", "n2", "n3"},
			map[string]*omv1.BootstrapMemberConfig{
				"n3": {Hidden: true, Priority: new(uint32(0))},
			},
		)

		require.NoError(t, err)
	})

	t.Run("translates member_configs from node id to executor host", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n2", "executor_host": "exec-n2", "observed": {"os_id": "ubuntu"}}`,
			`{"node_id": "n3", "executor_host": "exec-n3", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "member_configs": {"exec-n2": {"priority": 0, "votes": false, "hidden": true, "delay_secs": 300}}, "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				DataPath:       "/var/lib/mongo",
				LogPath:        "/var/log/mongodb/mongod.log",
				Port:           27017,
				BindIp:         "0.0.0.0",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n2": {
						Priority:  new(uint32(0)),
						Votes:     new(false),
						Hidden:    true,
						DelaySecs: 300,
					},
				},
			})

		require.NoError(t, err)
		require.Len(t, stub.calls, 4)
		assert.JSONEq(t,
			`{"hosts": ["exec-n1", "exec-n2", "exec-n3"], "install_method": "packages", "os": "ubuntu",
			  "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo",
			  "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0",
			  "member_configs": {"exec-n2": {"priority": 0, "votes": false, "hidden": true, "delay_secs": 300}}}`,
			stub.calls[3].body)
	})

	t.Run("leaves unset priority and votes out, so MongoDB's defaults apply", func(t *testing.T) {
		t.Parallel()

		// proto3 zero values are the opposite of MongoDB's defaults; sending them would leave
		// no voting or electable member, which rs.initiate rejects minutes into a run.
		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "member_configs": {"exec-n1": {"priority": 1, "votes": true, "hidden": false, "delay_secs": 0}}, "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				DataPath:       "/var/lib/mongo",
				LogPath:        "/var/log/mongodb/mongod.log",
				Port:           27017,
				BindIp:         "0.0.0.0",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {},
				},
			})

		require.NoError(t, err)
		require.Len(t, stub.calls, 2)
		assert.JSONEq(t,
			`{"hosts": ["exec-n1"], "install_method": "packages", "os": "ubuntu",
			  "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo",
			  "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0",
			  "member_configs": {"exec-n1": {"hidden": false, "delay_secs": 0}}}`,
			stub.calls[1].body)
	})

	// An older om_bootstrap silently drops per-member bind_ip, so mongod would come up on the run-level
	// address; only the echo shows it.
	t.Run("refuses a side-car that ignored a member's own bind address", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			// Echoed without bind_ip, as an older app does.
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu",
			  "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "bind_ip": "0.0.0.0",
			  "member_configs": {"exec-n1": {"priority": 1, "votes": true, "hidden": false, "delay_secs": 0}},
			  "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
			`{"id": "run-abc", "status": "running", "cancel_requested": true, "install_method": "packages",
			  "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod",
			  "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				BindIp:         "0.0.0.0",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {BindIp: new("10.0.0.1")},
				},
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		assert.Contains(t, message, "the per-member replica-set settings")
		assert.Contains(t, message, "older than this PMM")
		// The run is cancelled rather than left running.
		require.Len(t, stub.calls, 3)
	})

	// om_bootstrap echoes a null bind_ip for a member that named none, which is not a mismatch.
	t.Run("accepts a run whose members named no bind address", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu",
			  "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "bind_ip": "0.0.0.0",
			  "member_configs": {"exec-n1": {"priority": 1, "votes": true, "hidden": false, "delay_secs": 0, "bind_ip": null}},
			  "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				BindIp:         "0.0.0.0",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {},
				},
			})

		require.NoError(t, err)
	})

	t.Run("refuses a side-car that accepted the run but ignored its settings", func(t *testing.T) {
		t.Parallel()

		// An older om_bootstrap ignores unknown fields rather than rejecting them; its echo of
		// the accepted configuration lacks them, as below.
		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
			`{"id": "run-abc", "status": "running", "cancel_requested": true, "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				DataPath:       "/srv/mongo",
				LogPath:        "/var/log/mongodb/mongod.log",
				Port:           27018,
				BindIp:         "0.0.0.0",
			})

		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		message := status.Convert(err).Message()
		for _, setting := range []string{"the data path", "the log path", "the port", "the bind address"} {
			assert.Contains(t, message, setting)
		}
		assert.Contains(t, message, "older than this PMM")
		// The run is not left running.
		require.Len(t, stub.calls, 3)
		assert.Equal(t, "/api/apps/om_bootstrap/runs/run-abc:cancel", stub.calls[2].path)
	})

	t.Run("rejects a replica set with no voting member", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {Votes: new(false)},
					"n2": {Votes: new(false)},
					"n3": {Votes: new(false)},
				},
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "at least one voting member")
		assert.Empty(t, stub.calls, "nothing should reach PMM Extensions")
	})

	t.Run("rejects a replica set nothing can be elected in", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1", "n2", "n3"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				MemberConfigs: map[string]*omv1.BootstrapMemberConfig{
					"n1": {Priority: new(uint32(0))},
					"n2": {Priority: new(uint32(0))},
					"n3": {Priority: new(uint32(0))},
				},
			})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "can become primary")
	})

	t.Run("persists environment and cluster, and echoes them back through GetBootstrapRun", func(t *testing.T) {
		// Not parallel: storeTestDB drops and recreates one fixed-name database.
		db := storeTestDB(t)
		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token").
			WithBootstrapSource(stub.server.URL, "test-token")
		environment, cluster := "staging", "orders"

		triggered, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
				Environment:    &environment,
				Cluster:        &cluster,
			})
		require.NoError(t, err)

		config, err := models.FindOmBootstrapRunConfigByRunID(db.Querier, triggered.GetRunId())
		require.NoError(t, err)
		assert.Equal(t, "staging", config.Environment)
		assert.Equal(t, "orders", config.Cluster)

		response, err := svc.GetBootstrapRun(t.Context(),
			&omv1.GetBootstrapRunRequest{RunId: triggered.GetRunId()})
		require.NoError(t, err)
		assert.Equal(t, "staging", response.GetEnvironment())
		assert.Equal(t, "orders", response.GetCluster())
	})

	t.Run("leaves no config row, and echoes no environment or cluster, when neither was given", func(t *testing.T) {
		// Not parallel, for the same reason as the previous subtest.
		db := storeTestDB(t)
		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "executor_host": "n1", "observed": {"os_id": "ubuntu"}}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
			`{"id": "run-abc", "status": "running", "install_method": "packages", "os": "ubuntu", "mongodb_version": "7.0.8", "replica_set_name": "rs-orders-prod", "data_path": "/var/lib/mongo", "log_path": "/var/log/mongodb/mongod.log", "port": 27017, "bind_ip": "0.0.0.0", "started_at": "2026-01-01T00:00:00Z", "hosts": [], "run_steps": []}`,
		)
		svc := (&Service{db: db, l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token").
			WithBootstrapSource(stub.server.URL, "test-token")

		triggered, err := svc.TriggerHostBootstrap(t.Context(),
			&omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: "rs-orders-prod",
				MongodbVersion: "7.0.8",
			})
		require.NoError(t, err)

		_, err = models.FindOmBootstrapRunConfigByRunID(db.Querier, triggered.GetRunId())
		require.ErrorIs(t, err, models.ErrNotFound)

		response, err := svc.GetBootstrapRun(t.Context(),
			&omv1.GetBootstrapRunRequest{RunId: triggered.GetRunId()})
		require.NoError(t, err)
		assert.Nil(t, response.Environment)
		assert.Nil(t, response.Cluster)
	})
}

// Proto validation runs in the gRPC interceptor, not in the service, so the
// request goes through the same interceptor pmm-managed installs.
func TestTriggerHostBootstrapReplicaSetName(t *testing.T) {
	t.Parallel()

	validate := grpc_validator.UnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: omv1.OmService_TriggerHostBootstrap_FullMethodName}

	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"rs0", true},
		{"rs-orders_1", true},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
		{"", false},
		{"my set", false},
		{"bad/name", false},
		{"a:b", false},
		{"rs#1", false},
		{"rs.0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := &omv1.TriggerHostBootstrapRequest{
				NodeIds:        []string{"n1"},
				ReplicaSetName: tc.name,
				MongodbVersion: "7.0",
				DataPath:       "/var/lib/mongo",
				LogPath:        "/var/log/mongodb/mongod.log",
				Port:           27017,
				BindIp:         "0.0.0.0",
			}
			reached := false
			handler := func(context.Context, any) (any, error) {
				reached = true
				return &omv1.TriggerHostBootstrapResponse{}, nil
			}

			_, err := validate(t.Context(), req, info, handler)

			if tc.valid {
				require.NoError(t, err)
				assert.True(t, reached)
				return
			}
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.False(t, reached)
		})
	}
}

func TestGetBootstrapRun(t *testing.T) {
	t.Parallel()

	t.Run("projects a run's hosts, rollback steps, and run-level steps", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{
			"id": "run-abc",
			"status": "running",
			"install_method": "packages",
			"os": "ubuntu",
			"mongodb_version": "7.0.8",
			"replica_set_name": "rs-orders-prod",
			"started_at": "2026-01-01T00:00:00Z",
			"hosts": [
				{
					"host": "n1",
					"steps": [{"name": "pre_check", "status": "succeeded", "attempt_count": 1}],
					"rollback_steps": [{"name": "stop_service", "status": "pending", "attempt_count": 0}],
					"finalize_steps": [{"name": "enable_auth", "status": "pending", "attempt_count": 0}]
				}
			],
			"run_steps": [{"name": "rs_initiate", "status": "pending", "attempt_count": 0}]
		}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.GetBootstrapRun(t.Context(),
			&omv1.GetBootstrapRunRequest{RunId: "run-abc"})

		require.NoError(t, err)
		assert.Equal(t, "run-abc", response.GetRunId())
		assert.Equal(t, "running", response.GetStatus())
		assert.Equal(t, "rs-orders-prod", response.GetReplicaSetName())
		assert.Equal(t, "7.0.8", response.GetMongodbVersion())
		assert.Equal(t, mustParseTime(t, "2026-01-01T00:00:00Z"), response.GetStartedAt().AsTime())
		assert.Nil(t, response.GetFinishedAt())
		assert.Equal(t, "/api/apps/om_bootstrap/runs/run-abc", stub.path)
		require.Len(t, response.GetHosts(), 1)
		assert.Equal(t, "n1", response.GetHosts()[0].GetHost())
		assert.Equal(t, "pre_check", response.GetHosts()[0].GetSteps()[0].GetName())
		assert.Equal(t, "succeeded", response.GetHosts()[0].GetSteps()[0].GetStatus())
		assert.Equal(t, "stop_service", response.GetHosts()[0].GetRollbackSteps()[0].GetName())
		require.Len(t, response.GetHosts()[0].GetFinalizeSteps(), 2)
		assert.Equal(t, "enable_auth", response.GetHosts()[0].GetFinalizeSteps()[0].GetName())
		// confirm_monitoring is appended by PMM itself and stays pending until the run succeeds.
		assert.Equal(t, "confirm_monitoring", response.GetHosts()[0].GetFinalizeSteps()[1].GetName())
		assert.Equal(t, "pending", response.GetHosts()[0].GetFinalizeSteps()[1].GetStatus())
		require.Len(t, response.GetRunSteps(), 1)
		assert.Equal(t, "rs_initiate", response.GetRunSteps()[0].GetName())
	})

	t.Run("confirms monitoring against the inventory app once the run has succeeded", func(t *testing.T) {
		t.Parallel()

		// The run, then the host list confirmMonitoringLookup fetches once it sees "succeeded".
		// n1 already has a service; the inventory sweep has not caught up with n2 yet.
		stub := newSEPStubSeq(t, http.StatusOK,
			`{
				"id": "run-abc",
				"status": "succeeded",
				"install_method": "packages",
				"os": "ubuntu",
				"mongodb_version": "7.0.8",
				"replica_set_name": "rs-orders-prod",
				"started_at": "2026-01-01T00:00:00Z",
				"hosts": [
					{"host": "n1", "steps": [], "rollback_steps": [], "finalize_steps": []},
					{"host": "n2", "steps": [], "rollback_steps": [], "finalize_steps": []}
				],
				"run_steps": []
			}`,
			`{"items": [
				{"node_id": "node-1", "executor_host": "n1", "services": [{"service_id": "s1"}]},
				{"node_id": "node-2", "executor_host": "n2", "services": []}
			], "total": 2, "offset": 0, "limit": 200}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.GetBootstrapRun(t.Context(),
			&omv1.GetBootstrapRunRequest{RunId: "run-abc"})

		require.NoError(t, err)
		require.Len(t, stub.calls, 2)
		assert.Equal(t, "/api/apps/om_bootstrap/runs/run-abc", stub.calls[0].path)
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.calls[1].path)
		require.Len(t, response.GetHosts(), 2)
		n1Confirm := response.GetHosts()[0].GetFinalizeSteps()[0]
		assert.Equal(t, "confirm_monitoring", n1Confirm.GetName())
		assert.Equal(t, "succeeded", n1Confirm.GetStatus())
		n2Confirm := response.GetHosts()[1].GetFinalizeSteps()[0]
		assert.Equal(t, "confirm_monitoring", n2Confirm.GetName())
		assert.Equal(t, "running", n2Confirm.GetStatus())
	})

	t.Run("a run nobody created answers NotFound, not 500", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusNotFound, `{"detail": "Bootstrap run run-missing not found"}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.GetBootstrapRun(t.Context(), &omv1.GetBootstrapRunRequest{RunId: "run-missing"})

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})
}

func TestCancelBootstrapRun(t *testing.T) {
	t.Parallel()

	t.Run("proxies to PMM Extensions' :cancel route and reports cancel_requested", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{
			"id": "run-abc",
			"status": "running",
			"install_method": "packages",
			"os": "ubuntu",
			"mongodb_version": "7.0.8",
			"replica_set_name": "rs-orders-prod",
			"started_at": "2026-01-01T00:00:00Z",
			"hosts": [],
			"run_steps": [],
			"cancel_requested": true
		}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.CancelBootstrapRun(t.Context(),
			&omv1.CancelBootstrapRunRequest{RunId: "run-abc"})

		require.NoError(t, err)
		assert.Equal(t, "run-abc", response.GetRun().GetRunId())
		assert.True(t, response.GetRun().GetCancelRequested())
		assert.Equal(t, "/api/apps/om_bootstrap/runs/run-abc:cancel", stub.path)
	})

	t.Run("a run nobody created answers NotFound, not 500", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusNotFound, `{"detail": "Bootstrap run run-missing not found"}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.CancelBootstrapRun(t.Context(), &omv1.CancelBootstrapRunRequest{RunId: "run-missing"})

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("an already-terminal run answers the conflict PMM Extensions reports", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusConflict, `{"detail": "Run run-abc is already rolled_back"}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.CancelBootstrapRun(t.Context(), &omv1.CancelBootstrapRunRequest{RunId: "run-abc"})

		require.Error(t, err)
		assert.Equal(t, codes.Aborted, status.Code(err))
	})
}

func TestListBootstrapRuns(t *testing.T) {
	t.Parallel()

	t.Run("projects every run in the same shape GetBootstrapRun answers with", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(t, http.StatusOK, `[
			{
				"id": "run-abc",
				"status": "succeeded",
				"install_method": "packages",
				"os": "ubuntu",
				"mongodb_version": "7.0.8",
				"replica_set_name": "rs-orders-prod",
				"started_at": "2026-01-01T00:00:00Z",
				"finished_at": "2026-01-01T00:05:00Z",
				"hosts": [{"host": "n1", "steps": [{"name": "pre_check", "status": "succeeded", "attempt_count": 1}]}],
				"run_steps": []
			},
			{
				"id": "run-def",
				"status": "running",
				"install_method": "packages",
				"os": "ubuntu",
				"mongodb_version": "7.0.8",
				"replica_set_name": "rs-billing",
				"started_at": "2026-01-02T00:00:00Z",
				"hosts": [],
				"run_steps": []
			}
		]`,
			// confirmMonitoringLookup's GET /hosts, fetched because run-abc succeeded.
			`{"items": [{"node_id": "node-1", "executor_host": "n1", "services": [{"service_id": "s1"}]}], "total": 1, "offset": 0, "limit": 200}`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		response, err := svc.ListBootstrapRuns(t.Context(), &omv1.ListBootstrapRunsRequest{})

		require.NoError(t, err)
		require.Len(t, stub.calls, 2)
		assert.Equal(t, "/api/apps/om_bootstrap/runs", stub.calls[0].path)
		assert.Equal(t, "limit=20", stub.calls[0].query)
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.calls[1].path)
		require.Len(t, response.GetRuns(), 2)
		assert.Equal(t, "run-abc", response.GetRuns()[0].GetRunId())
		assert.Equal(t, "succeeded", response.GetRuns()[0].GetStatus())
		assert.Equal(t, "rs-orders-prod", response.GetRuns()[0].GetReplicaSetName())
		assert.Equal(t, mustParseTime(t, "2026-01-01T00:05:00Z"), response.GetRuns()[0].GetFinishedAt().AsTime())
		assert.Equal(t, "n1", response.GetRuns()[0].GetHosts()[0].GetHost())
		// confirm_monitoring succeeded because n1 already has a service.
		n1FinalizeSteps := response.GetRuns()[0].GetHosts()[0].GetFinalizeSteps()
		assert.Equal(t, "confirm_monitoring", n1FinalizeSteps[len(n1FinalizeSteps)-1].GetName())
		assert.Equal(t, "succeeded", n1FinalizeSteps[len(n1FinalizeSteps)-1].GetStatus())
		assert.Equal(t, "run-def", response.GetRuns()[1].GetRunId())
		assert.Nil(t, response.GetRuns()[1].GetFinishedAt())
	})

	t.Run("clamps an over-large limit before forwarding it", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `[]`)
		svc := stub.service(t).WithBootstrapSource(stub.server.URL, "test-token")

		_, err := svc.ListBootstrapRuns(t.Context(), &omv1.ListBootstrapRunsRequest{Limit: 500})

		require.NoError(t, err)
		assert.Equal(t, "limit=100", stub.query)
	})
}

func TestNodeIDForExecutorHost(t *testing.T) {
	t.Parallel()

	t.Run("finds the node id whose executor matches", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"items": [
			{"node_id": "c58168e8-...", "executor_host": "pmm-client-node00"},
			{"node_id": "other-node", "executor_host": "pmm-client-node01"}
		], "total": 2, "offset": 0, "limit": 200}`)
		svc := stub.service(t)

		nodeID, err := svc.nodeIDForExecutorHost(t.Context(), "pmm-client-node00")

		require.NoError(t, err)
		assert.Equal(t, "c58168e8-...", nodeID)
		assert.Equal(t, "/api/apps/om_inventory/hosts", stub.path)
	})

	t.Run("answers NotFound when no host has that executor", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"items": [{"node_id": "n1", "executor_host": "pmm-client-node00"}], "total": 1, "offset": 0, "limit": 200}`)
		svc := stub.service(t)

		_, err := svc.nodeIDForExecutorHost(t.Context(), "no-such-executor")

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
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
			// Reflecting the app's 401 would tell the browser to re-authenticate against PMM, which fixes nothing.
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
				&omv1.GetInventoryHostRequest{NodeId: "n9"})

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
		&omv1.DeleteInventoryHostRequest{NodeId: "n1"})

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, stub.method)
	assert.Equal(t, "/api/apps/om_inventory/hosts/n1", stub.path)
}

func TestInventoryConfig(t *testing.T) {
	t.Parallel()

	t.Run("reports every field and where its value came from", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, configBody)

		response, err := stub.service(t).GetInventoryConfig(t.Context(), &omv1.GetInventoryConfigRequest{})

		require.NoError(t, err)
		require.Len(t, response.GetSettings(), 2)

		schedule := response.GetSettings()[0]
		assert.Equal(t, "SCHEDULE__every", schedule.GetKey())
		assert.InDelta(t, 10.0, schedule.GetValue().GetNumberValue(), 0.001)
		assert.Equal(t, omv1.SettingReload_SETTING_RELOAD_HOT, schedule.GetReload())
		assert.False(t, schedule.GetHasOverride())

		// A field the deployment owns is listed rather than hidden, so a UI can show it greyed out.
		assert.Equal(t, omv1.SettingReload_SETTING_RELOAD_NOT_OVERRIDABLE, response.GetSettings()[1].GetReload())
	})

	t.Run("a change is passed through as the app will validate it", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, configBody)
		values, err := structpb.NewStruct(map[string]any{"SCHEDULE__every": 25})
		require.NoError(t, err)

		_, err = stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{Values: values})

		require.NoError(t, err)
		require.Len(t, stub.calls, 2, "the write, then the read-back")
		assert.Equal(t, http.MethodPatch, stub.calls[0].method, "the app's own verb, not PMM's PUT")
		assert.JSONEq(t, `{"SCHEDULE__every": 25}`, stub.calls[0].body)
	})

	t.Run("the answer is the whole configuration, not the submitted keys", func(t *testing.T) {
		t.Parallel()

		// The app answers a PATCH with only the named keys, but overriding a parent also moves its children,
		// so the handler reads the configuration back.
		applied := `[{"key": "SCHEDULE__every", "value": 25, "default_value": null,
		              "type": "int", "reload": "hot", "has_override": true,
		              "is_advanced": false, "description": null}]`
		stub := newSEPStubSeq(t, http.StatusOK, applied, configBody)
		values, err := structpb.NewStruct(map[string]any{"SCHEDULE__every": 25})
		require.NoError(t, err)

		response, err := stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{Values: values})

		require.NoError(t, err)
		require.Len(t, stub.calls, 2)
		assert.Equal(t, http.MethodGet, stub.calls[1].method)
		assert.Equal(t, "/api/apps/om_inventory/config", stub.calls[1].path)
		// From the read-back, not the single row the write echoed.
		require.Len(t, response.GetSettings(), 2)
		assert.Equal(t, "CREDENTIALS_PATH", response.GetSettings()[1].GetKey())
	})

	t.Run("a failed read-back does not report the write as failed", func(t *testing.T) {
		t.Parallel()

		// The write landed; an error would tell the caller to retry a change that already applied.
		applied := `[{"key": "SCHEDULE__every", "value": 25, "default_value": null,
		              "type": "int", "reload": "hot", "has_override": true,
		              "is_advanced": false, "description": null}]`
		stub := newSEPStubSeq(t, http.StatusOK, applied, `not json`)
		values, err := structpb.NewStruct(map[string]any{"SCHEDULE__every": 25})
		require.NoError(t, err)

		response, err := stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{Values: values})

		require.NoError(t, err)
		require.Len(t, response.GetSettings(), 1)
		assert.Equal(t, "SCHEDULE__every", response.GetSettings()[0].GetKey())
	})

	t.Run("an empty batch is refused here", func(t *testing.T) {
		t.Parallel()

		// An empty PATCH would succeed on the app's side and change nothing.
		stub := newSEPStub(t, http.StatusOK, configBody)

		_, err := stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Empty(t, stub.method, "nothing should have been sent")
	})

	t.Run("a batch wider than the settings class is refused here", func(t *testing.T) {
		t.Parallel()

		// Nothing else bounds this: protojson takes 200k fields inside the 4MB gRPC default.
		stub := newSEPStub(t, http.StatusOK, configBody)
		wide := map[string]any{}
		for i := 0; i <= maxConfigFields; i++ {
			wide["KEY_"+strconv.Itoa(i)] = i
		}
		values, err := structpb.NewStruct(wide)
		require.NoError(t, err)

		_, err = stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{Values: values})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "at most 100 may change in one call")
		assert.Empty(t, stub.method, "nothing should have been sent")
	})

	t.Run("a batch nested deeper than the app can parse is refused here", func(t *testing.T) {
		t.Parallel()

		// The app's Python recursion limit is 1000, while protojson accepts nesting just short of 10k.
		stub := newSEPStub(t, http.StatusOK, configBody)
		nested := map[string]any{"leaf": 1}
		for range maxConfigDepth + 1 {
			nested = map[string]any{"SCHEDULE": nested}
		}
		values, err := structpb.NewStruct(nested)
		require.NoError(t, err)

		_, err = stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{Values: values})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), "nests deeper than 10 levels")
		assert.Empty(t, stub.method, "nothing should have been sent")
	})

	t.Run("the nesting a real batch uses is not refused", func(t *testing.T) {
		t.Parallel()

		// A whole-object write of SCHEDULE is the deepest legitimate shape.
		stub := newSEPStub(t, http.StatusOK, configBody)
		values, err := structpb.NewStruct(map[string]any{
			"SCHEDULE": map[string]any{"every": 25, "period": "seconds"},
		})
		require.NoError(t, err)

		_, err = stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{Values: values})

		require.NoError(t, err)
		assert.JSONEq(t, `{"SCHEDULE": {"every": 25, "period": "seconds"}}`, stub.calls[0].body)
	})

	t.Run("a revert names the field in the path", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusNoContent, ``)

		_, err := stub.service(t).DeleteInventoryConfigOverride(t.Context(),
			&omv1.DeleteInventoryConfigOverrideRequest{Key: "SCHEDULE__every"})

		require.NoError(t, err)
		assert.Equal(t, http.MethodDelete, stub.method)
		assert.Equal(t, "/api/apps/om_inventory/config/SCHEDULE__every", stub.path)
	})
}

func TestInventoryRunsDefaultLimit(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, `[]`)

	_, err := stub.service(t).ListInventoryRuns(t.Context(), &omv1.ListInventoryRunsRequest{})

	require.NoError(t, err)
	assert.Equal(t, "limit=20", stub.query)
}

func TestInventoryRunsForwardsDateRange(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, `[]`)
	since := timestamppb.New(mustParseTime(t, "2026-08-18T00:00:00Z"))
	until := timestamppb.New(mustParseTime(t, "2026-08-25T00:00:00Z"))

	_, err := stub.service(t).ListInventoryRuns(t.Context(), &omv1.ListInventoryRunsRequest{
		Since: since,
		Until: until,
	})

	require.NoError(t, err)
	query, err := url.ParseQuery(stub.query)
	require.NoError(t, err)
	assert.Equal(t, "20", query.Get("limit"))
	assert.Equal(t, "2026-08-18T00:00:00Z", query.Get("since"))
	assert.Equal(t, "2026-08-25T00:00:00Z", query.Get("until"))
}

func TestInventoryRunsRejectsInvertedDateRange(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, `[]`)

	_, err := stub.service(t).ListInventoryRuns(t.Context(), &omv1.ListInventoryRunsRequest{
		Since: timestamppb.New(mustParseTime(t, "2026-08-25T00:00:00Z")),
		Until: timestamppb.New(mustParseTime(t, "2026-08-18T00:00:00Z")),
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Empty(t, stub.query)
}

func TestInventoryRunCarriesHostCounters(t *testing.T) {
	t.Parallel()

	// Counting only services would make a refresh of a host with no database read as "0 of 0".
	stub := newSEPStub(t, http.StatusOK, `[{
	  "run_id": "r1", "status": "success",
	  "started_at": "2026-08-18T12:00:00Z", "finished_at": "2026-08-18T12:00:30Z",
	  "counts": {"services_total": 0, "services_resolved": 0, "services_orphaned": 0,
	             "services_answered": 0,
	             "hosts_total": 3, "hosts_probeable": 2, "hosts_answered": 1,
	             "hosts_finished": 2},
	  "scope": ["node-1"], "error": null
	}]`)

	res, err := stub.service(t).ListInventoryRuns(t.Context(), &omv1.ListInventoryRunsRequest{})

	require.NoError(t, err)
	require.Len(t, res.GetRuns(), 1)
	counts := res.GetRuns()[0].GetCounts()
	assert.Equal(t, int32(3), counts.GetTotalHosts())
	assert.Equal(t, int32(2), counts.GetProbeableHosts())
	assert.Equal(t, int32(1), counts.GetAnsweredHosts())
	assert.Equal(t, int32(2), counts.GetFinishedHosts())
	assert.Equal(t, int32(0), counts.GetTotalServices())
}

func TestInventoryRunNamesItsFailingNodes(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, `[{
	  "run_id": "r1", "status": "partial",
	  "started_at": "2026-10-07T12:00:00Z", "finished_at": "2026-10-07T12:00:30Z",
	  "counts": {"services_total": 0, "services_resolved": 0, "services_orphaned": 0,
	             "services_answered": 0,
	             "hosts_total": 3, "hosts_probeable": 3, "hosts_answered": 1},
	  "scope": null, "error": null,
	  "failing_nodes": [{"node_id": "n2", "name": "node02"}, {"node_id": "n3", "name": null}]
	}, {
	  "run_id": "r0", "status": "success",
	  "started_at": "2026-10-07T11:50:00Z", "finished_at": "2026-10-07T11:50:30Z",
	  "counts": {"services_total": 0, "services_resolved": 0, "services_orphaned": 0,
	             "services_answered": 0,
	             "hosts_total": 3, "hosts_probeable": 3, "hosts_answered": 3},
	  "scope": null, "error": null
	}]`)

	res, err := stub.service(t).ListInventoryRuns(t.Context(), &omv1.ListInventoryRunsRequest{})

	require.NoError(t, err)
	require.Len(t, res.GetRuns(), 2)
	failing := res.GetRuns()[0].GetFailingNodes()
	require.Len(t, failing, 2)
	assert.Equal(t, "n2", failing[0].GetNodeId())
	assert.Equal(t, "node02", failing[0].GetName())
	assert.Empty(t, failing[1].GetName(), "a node the run recorded no name for")
	// A server that predates the field sends none.
	assert.Empty(t, res.GetRuns()[1].GetFailingNodes())
}

func TestInventoryRunDetailIsHostOriented(t *testing.T) {
	t.Parallel()

	stub := newSEPStub(t, http.StatusOK, `{
	  "run_id": "r1", "status": "partial",
	  "started_at": "2026-08-19T12:00:00Z", "finished_at": "2026-08-19T12:00:30Z",
	  "counts": {"services_total": 1, "services_resolved": 1, "services_orphaned": 0,
	             "services_answered": 1,
	             "hosts_total": 3, "hosts_probeable": 2, "hosts_answered": 2},
	  "scope": [], "error": null,
	  "nodes": [
	    {"node_id": "n1", "host_name": "db00", "executor_host": "db00",
	     "resolution": "name", "answered": true, "duration_seconds": 12.5,
	     "task_history_id": 4711, "error": null,
	     "services": [{"service_id": "s1", "service_name": "mongo-1",
	                   "answered": true, "error": null}]},
	    {"node_id": "n2", "host_name": "pmm-client-node00",
	     "executor_host": "pmm-client-node00", "resolution": "name",
	     "answered": true, "duration_seconds": 8.0, "task_history_id": null,
	     "error": null, "services": []},
	    {"node_id": "n3", "host_name": "stranded", "executor_host": null,
	     "resolution": "orphaned", "answered": false, "duration_seconds": null,
	     "task_history_id": null, "error": "no executor host", "services": []}
	  ]
	}`)

	res, err := stub.service(t).GetInventoryRun(t.Context(), &omv1.GetInventoryRunRequest{RunId: "r1"})

	require.NoError(t, err)
	require.Len(t, res.GetEntities(), 3)

	withDatabase := res.GetEntities()[0]
	assert.Equal(t, "db00", withDatabase.GetHostName())
	assert.True(t, withDatabase.GetAnswered())
	assert.InDelta(t, 12.5, withDatabase.GetDurationSeconds(), 0.001)
	// Points at this attempt's raw output, which the receipt does not keep.
	assert.Equal(t, int64(4711), withDatabase.GetTaskHistoryId())
	require.Len(t, withDatabase.GetServices(), 1)
	assert.Equal(t, "mongo-1", withDatabase.GetServices()[0].GetServiceName())

	bare := res.GetEntities()[1]
	assert.Equal(t, "pmm-client-node00", bare.GetHostName())
	assert.True(t, bare.GetAnswered())
	assert.Empty(t, bare.GetServices())

	// An orphan keeps its row: "2 of 3 answered" cannot say which one was missed.
	orphan := res.GetEntities()[2]
	assert.Equal(t, omv1.ExecutorResolution_EXECUTOR_RESOLUTION_ORPHANED, orphan.GetResolution())
	assert.False(t, orphan.GetAnswered())
	// Fields, not getters: for proto3 optional scalars only the pointer distinguishes unset from empty.
	assert.Nil(t, orphan.ExecutorHost)
	assert.Nil(t, orphan.DurationSeconds)
	assert.Nil(t, orphan.TaskHistoryId)
	assert.Equal(t, "no executor host", orphan.GetError())
}

func TestInventoryBearerIsSent(t *testing.T) {
	t.Parallel()

	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items": [], "total": 0, "offset": 0, "limit": 200}`))
	}))
	t.Cleanup(server.Close)

	svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(server.URL, "test-token")

	_, err := svc.ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})

	require.NoError(t, err)
	assert.Equal(t, "Bearer test-token", seen)
}

func mustParseTime(t *testing.T, stamp string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, stamp)
	require.NoError(t, err)
	return parsed
}

func TestInventoryFixturesAreValid(t *testing.T) {
	t.Parallel()

	t.Run("hosts", func(t *testing.T) {
		t.Parallel()

		var parsed struct {
			Items []map[string]any `json:"items"`
		}
		require.NoError(t, json.Unmarshal([]byte(hostsBody), &parsed))
		assert.NotEmpty(t, parsed.Items)
	})

	t.Run("config", func(t *testing.T) {
		t.Parallel()

		var parsed []map[string]any
		require.NoError(t, json.Unmarshal([]byte(configBody), &parsed))
		assert.NotEmpty(t, parsed)
	})
}

// A debounce window that never reopens silently makes every later run wait out
// bootstrapInventoryRefreshWindow before confirm_monitoring resolves.
func TestNudgeDebouncesExpire(t *testing.T) {
	t.Parallel()

	t.Run("a node whose refresh window has passed is due again", func(t *testing.T) {
		t.Parallel()

		svc := &Service{refreshRequested: map[string]time.Time{
			"node00": time.Now().Add(-inventoryRefreshDebounce),
			"node01": time.Now(),
		}}

		assert.Equal(t, []string{"node00"}, svc.refreshDue([]string{"node00", "node01"}, time.Now()),
			"node00's stamp is exactly one window old, node01's is fresh")
		assert.NotContains(t, svc.refreshRequested, "node00",
			"the aged-out entry is pruned, so the map stays bounded by the nodes under bootstrap")
		assert.Contains(t, svc.refreshRequested, "node01")
	})

	t.Run("a sync whose window has passed is sent again", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusAccepted, "")
		svc := (&Service{l: logrus.WithField("test", t.Name())}).
			WithProbeSource(stub.server.URL, "test-token")
		svc.syncRequested = time.Now().Add(-inventorySyncDebounce)

		svc.triggerInventorySync(t.Context())

		require.Len(t, stub.calls, 1, "a stamp one window old must not hold the next sync off")
		assert.Equal(t, "/api/apps/inventory/sync/", stub.calls[0].path)
	})
}
