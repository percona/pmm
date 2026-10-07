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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	omv1 "github.com/percona/pmm/api/om/v1"
	"github.com/percona/pmm/managed/models"
)

// hostsBody is one om_inventory GET /hosts answer, wrapped in PMM Extensions' own
// paginated envelope (PMM-15326: "Bound the estate listings" --
// app/core/pagination/models.py's PaginatedResponse) rather than a bare array.
//
// Two hosts on purpose, because the pair is the whole reason the host table exists.
// `n1` runs a registered database and reports an unregistered mongod beside it -- the
// arbiter case, which PMM registers no service for, so nothing else in OM would mention
// the process. `n2` carries a PMM client and no database at all, which is the case a
// service-keyed inventory cannot represent.
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
    "services": []
  }
], "total": 2, "offset": 0, "limit": 200}`

// configBody is one om_inventory GET /config answer, trimmed to the two rows that make
// the point: a nested schedule leaf and a field the deployment owns outright.
const configBody = `[
  {"key": "SCHEDULE__every", "value": 10, "default_value": null, "type": "int",
   "reload": "hot", "has_override": false, "is_advanced": false, "description": null},
  {"key": "CREDENTIALS_PATH", "value": null, "default_value": null, "type": "str",
   "reload": "not_overridable", "has_override": false, "is_advanced": false,
   "description": null}
]`

// stubCall is one request the stub was asked to serve.
type stubCall struct {
	method string
	path   string
	query  string
	body   string
}

// extensionsStub stands in for the inventory app, recording what it was asked.
type extensionsStub struct {
	server *httptest.Server

	method string
	path   string
	query  string
	body   string

	// calls records every request in order. The fields above keep only the last, which
	// is enough for a handler that makes one call and wrong for one that writes and then
	// reads back.
	calls []stubCall
	// bodies, when set, is served one entry per request so a read-after-write can be
	// given a different answer from the write itself. The last entry repeats once
	// exhausted.
	bodies []string
}

// newSEPStub serves one canned answer and records the request that fetched it.
func newSEPStub(t *testing.T, code int, body string) *extensionsStub {
	t.Helper()

	return newSEPStubSeq(t, code, body)
}

// newSEPStubSeq serves one canned answer per request, in order, all with the same
// status code. See newSEPStubSeqCodes for a stub whose calls answer with different
// codes too -- a handler that reads then writes, where the write is the one that
// fails.
func newSEPStubSeq(t *testing.T, code int, bodies ...string) *extensionsStub {
	t.Helper()

	codes := make([]int, len(bodies))
	for i := range codes {
		codes[i] = code
	}
	return newSEPStubSeqCodes(t, codes, bodies)
}

// newSEPStubSeqCodes serves one canned (code, body) answer per request, in order.
// The last pair repeats once exhausted, same as newSEPStubSeq's bodies-only form.
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

// service returns an OM service wired to the stub.
func (s *extensionsStub) service(t *testing.T) *Service {
	t.Helper()

	svc := &Service{l: logrus.WithField("test", t.Name())}
	return svc.WithProbeSource(s.server.URL, "test-token")
}

func TestInventoryNotConfigured(t *testing.T) {
	t.Parallel()

	// An unconfigured PMM Extensions is a deployment that has not been told where PMM Extensions is, not a
	// missing feature and not a broken app. FailedPrecondition says so; NotFound would
	// read as "there is no such endpoint" and send the reader looking for a version
	// problem.
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

		// The app stores observations as JSON so a new attribute is a payload change
		// rather than a schema change. Enumerating every attribute as a proto field
		// would put that coupling straight back, so anything without a field of its own
		// still has to arrive.
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

		// stub.service(t) never calls WithAgentRegistry, matching every other test in
		// this file -- the fail-closed default from PMM-15347's PoC (see
		// agentConnectionChecker's doc comment): a host with a fully healthy executor
		// still reads as ineligible when PMM-managed has no wired signal for
		// pmm-agent connectivity.
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

		// Three false flags would claim PMM Extensions looked and the answer was no. A nil block
		// says this sweep did not say, which is what an empty document means.
		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		empty := response.GetHosts()[1]
		assert.Nil(t, empty.GetExecutor())
		assert.Nil(t, empty.GetObserved())
		assert.Empty(t, empty.GetServices(), "a host with no database is a row, not an omission")
	})

	t.Run("a failing host carries why and since when", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, hostsBody)

		response, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})
		require.NoError(t, err)

		freshness := response.GetHosts()[1].GetFreshness()
		assert.Equal(t, int32(3), freshness.GetConsecutiveFailures())
		assert.Equal(t, "no executor host", freshness.GetLastError())
		assert.NotNil(t, freshness.GetFailingSince())
		assert.Nil(t, freshness.GetLastSuccessAt(),
			"never having answered is different from having answered nothing")
	})

	t.Run("passes the filters through", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{"items": [], "total": 0, "offset": 0, "limit": 200}`)

		// Addressable locals rather than a helper: these are proto3 `optional` bools, so
		// what the request carries is a plain *bool, and false has to be distinguishable
		// from unset -- which is the whole point of the three assertions below.
		hasService, failing, executor := false, true, false

		_, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{
			HasService: &hasService,
			Failing:    &failing,
			Executor:   &executor,
		})

		require.NoError(t, err)
		assert.Contains(t, stub.query, "has_service=false")
		assert.Contains(t, stub.query, "failing=true")
		// The app types this one bool as well: it filters on whether an executor is
		// matched at all, not on which client. A host name here is a 422 from the app,
		// which is what this assertion used to require.
		assert.Contains(t, stub.query, "executor=false")
	})

	t.Run("an unset filter is not sent as false", func(t *testing.T) {
		t.Parallel()

		// has_service=false means "only hosts with no database", which is a very
		// different listing from the default. Sending it because the caller said nothing
		// would silently hide every host that has one.
		stub := newSEPStub(t, http.StatusOK, `{"items": [], "total": 0, "offset": 0, "limit": 200}`)

		_, err := stub.service(t).ListInventoryHosts(t.Context(), &omv1.ListInventoryHostsRequest{})

		require.NoError(t, err)
		assert.NotContains(t, stub.query, "has_service")
		assert.NotContains(t, stub.query, "failing")
		assert.NotContains(t, stub.query, "executor")
	})

	t.Run("walks every page", func(t *testing.T) {
		t.Parallel()

		// total (3) exceeds the first page's own item count (2), which is what makes
		// fetchAllPages loop instead of stopping after one request -- every other test
		// in this file sets total equal to len(items), so a regression that stopped
		// after the first page or miscomputed the next offset would pass them all.
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

		// GET /services answers the same PaginatedResponse envelope GET /hosts does
		// (PMM-15326: "Bound the estate listings") -- decoding a bare array here is
		// exactly the bug this PR fixes for /hosts, and this handler had no coverage
		// at all, which is how it was missed.
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
		// The caller's filters have to survive being copied onto the per-page query.
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
}

func TestAutomationEligibility(t *testing.T) {
	t.Parallel()

	healthyExecutor := &omv1.InventoryExecutor{Registered: true, Reachable: true, DriverHealthy: true}
	reachableOnlyExecutor := &omv1.InventoryExecutor{Registered: true, Reachable: true, DriverHealthy: false}

	// A host that passes every check this function makes, so each case below isolates
	// the one condition it names. Built by a helper rather than written as
	// `extensionsHost{}`: os_id and the address are now checked too, so the zero value
	// fails three ways at once and every assertion would be about all of them.
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

		// A nil executor is what om_inventory sends for a host it has never
		// dispatched to -- see inventory_test.go's "a host with no probe reports
		// absent, not false". Reporting a driver-health failure on top of that would
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

	// No user-facing reason names Nomad. These strings are joined straight into the
	// tooltip on the Nodes page, so they are product copy, and PMM-15623 set out to
	// keep the scheduler's name out of it.
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

	// Every condition above asks whether OM *can* reach this machine. Those below
	// ask whether it *should* touch it, which is a different question and the one
	// PMM-15664 exists to answer: a perfectly reachable node can still be the last
	// thing anyone wants a database installed onto.
	t.Run("the PMM Server's own node is never eligible, however healthy", func(t *testing.T) {
		t.Parallel()

		eligible, reasons, byDesign := automationEligibility(healthyExecutor, true, true, installable())

		assert.False(t, eligible)
		assert.Equal(t, []string{
			"this is the node PMM Server itself runs on, which Operations never installs onto",
		}, reasons)
		assert.True(t, byDesign, "the server's own node is a fact about it, not a fault on it")
	})

	// It is the only reason, not the first of several. PMM Server's own image reports
	// os_id "ol", so without the short circuit the row also advised that Operations
	// cannot install onto "ol" -- which reads as though a different OS would help.
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

	// Installed but stopped: no service, nothing running, so only the scan's reading
	// of the binary shows it - and an install onto it would leave two mongods.
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

	// A node usually has both: the registered service and the mongod serving it. One
	// problem, so one reason - otherwise a reader counts two faults where there is one.
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

	// The two preconditions that used to fail on the wizard's last click,
	// inside TriggerHostBootstrap, after the whole form was filled in.
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

	// Named rather than asserted as a substring: the sentence lists the supported
	// distributions, and a reader acting on it needs them to be the ones the trigger
	// actually accepts.
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

	// installed_version against running version is the whole reason the probe exists:
	// their divergence is the upgraded-but-not-restarted case, and no metric carries it.
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

		// The whole argument for keying the estate on PMM's node ID is that no
		// translation step exists. If one appeared here, it would be the bug that
		// argument was meant to prevent.
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

		// Conflict is an answer a caller acts on -- wait, or refresh something else --
		// so it has to survive the hop as a conflict. Aborted is the code the gateway
		// renders as 409.
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
		// The dispatched-to host is n1's *executor*, not its node id -- see
		// TriggerHostBootstrap's own doc comment on why they can differ, even
		// though this fixture happens to give them the same value.
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
		// The complaint was the advice, not only the wording: it told the user to
		// wait after 160 runs had already failed.
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
		// Just the host lookup -- rejected before ever reaching om_bootstrap,
		// not a second call that would be the actual trigger request.
		require.Len(t, stub.calls, 1)
	})

	t.Run("a host with no usable executor answers FailedPrecondition, not 500", func(t *testing.T) {
		t.Parallel()

		// No executor_host at all -- an unprobed host, or one om_inventory
		// never matched to a Nomad client. Caught locally, before ever
		// reaching om_bootstrap: dispatching to node_id here (this bug's own
		// root cause -- see TriggerHostBootstrap's doc comment) would instead
		// reach PMM Extensions and fail there, confusingly, once every host in flight.
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

		// Raised on the PMM Extensions side of this work: without this check the run is created
		// first and an unreachable Nomad client only surfaces when pre_check -- itself
		// dispatched through Nomad -- fails seconds later, with the UI already showing
		// the run as in progress. Only one call is served here, so nothing reached
		// om_bootstrap.
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

	// MongoDB will not elect a member clients cannot see, so rs.initiate() refuses a
	// hidden member with priority above 0. PMM-15661 adds this rule in the install
	// wizard and scopes the backend out; without it here the browser is the only thing
	// between a direct API call and a run that installs mongod on every host and then
	// fails inside rs.initiate.
	// The readiness checks. Three nodes with three different problems used to
	// produce one message about one of them, so a user fixing them discovered the
	// second only by fixing the first and running the trigger again.
	t.Run("names every unready node and every problem, not the first", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			// No OS reported at all.
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1", "observed": {}}`,
			// An OS om_bootstrap cannot install onto.
			`{"node_id": "n2", "name": "db-02", "executor_host": "exec-n2", "observed": {"os_id": "windows"}}`,
			// No automation agent, and no OS either -- two problems on one node.
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
		// Both of db-03's problems, not just the one that was found first.
		assert.Contains(t, message, "no automation agent is registered")
		assert.Contains(t, message, "windows")
		// The advice is to fix the nodes, and it points at where the reason came from.
		assert.Contains(t, message, "Nodes page")
		assert.NotContains(t, message, "wait")
		// Nothing was planned.
		require.Len(t, stub.calls, 3)
	})

	// A mixed selection is the selection's problem, not any one node's, so it names
	// the groups rather than blaming whichever node happened to be second.
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
		// Both groups, with their members, so the reader can see which two to keep.
		assert.Contains(t, message, "rocky on db-02")
		assert.Contains(t, message, "ubuntu on db-01, db-03")
	})

	// The safety gate, and the reason it is not left to the UI. automation_eligible is
	// advisory: computed for a list request, minutes stale by the time anyone clicks,
	// and never read at all by a direct API call. These are the two cases that are
	// most damaging thing Operations can do.
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
		// By name, not by the node id the caller sent: a UUID is not what anyone's
		// inventory, runbook or ticket calls the machine.
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

	// Every blocking node, not the first. A user fixing three nodes should not have to
	// run the trigger three times to discover there were three.
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
		// The healthy one is not blamed.
		assert.NotContains(t, message, "db-02")
		// All three were looked up before anything was refused, and om_bootstrap was
		// never asked to plan: three host calls, no trigger call.
		require.Len(t, stub.calls, 3)
	})

	// A node PMM has no name for still has to be identifiable, and its id is the only
	// identifier that exists in that case.
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
					// Priority left unset, which is the same mistake as setting it
					// wrong: unset means MongoDB's default of 1.
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

	// The other side of it: hidden is perfectly legitimate with priority 0, and a
	// rule that refused that would block the topology the field exists for. Checked
	// against validateMemberConfigs directly, because a one-member set with priority
	// 0 is refused by the no-electable-member rule instead and would prove nothing.
	// The member rules. Two misconfigured members in one plan used to report
	// one of them. Checked directly rather than through the trigger, because the point
	// is the set of violations rather than the RPC around it.
	t.Run("returns every member violation in one error", func(t *testing.T) {
		t.Parallel()

		err := validateMemberConfigs(
			[]string{"n1", "n2", "n3"},
			map[string]*omv1.BootstrapMemberConfig{
				// Delayed, but still electable and voting.
				"n1": {DelaySecs: 300},
				// Hidden, but still electable.
				"n2": {Hidden: true},
				// Not in the selection at all.
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

	// Map iteration order is random, so an error built by ranging one reshuffles
	// itself between two identical attempts and cannot be diffed against the last.
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

		// The trap this closes: proto3 zero values are the opposite of MongoDB's
		// defaults, so a member config that set neither used to send priority 0 and
		// votes false anyway -- three such hosts is a replica set with no voting
		// member and nothing electable, which rs.initiate rejects minutes into a run.
		// Unset now means unsent, and om_bootstrap's own MemberConfig defaults
		// (priority 1, votes on) apply.
		//
		// An empty member config is the vehicle, rather than `{Hidden: true}` as it
		// once was: a hidden member must set priority 0, so that input is now refused
		// before it can demonstrate anything about serialization.
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

	// An om_bootstrap older than the per-member bind_ip accepts the request and
	// ignores the field -- pydantic drops what it does not know -- and mongod would
	// then come up on the run-level address, which is the 0.0.0.0 the per-member value
	// exists to avoid. The echo is what makes that detectable.
	t.Run("refuses a side-car that ignored a member's own bind address", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStubSeq(
			t, http.StatusOK,
			`{"node_id": "n1", "name": "db-01", "executor_host": "exec-n1", "observed": {"os_id": "ubuntu"}}`,
			// Echoed without bind_ip, which is what an older app sends back.
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
		// And the run it would not configure is cancelled rather than left running.
		require.Len(t, stub.calls, 3)
	})

	// The other way round: a member that named no address must not read as a mismatch
	// just because om_bootstrap echoes None for it.
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
					// Nothing set, so om_bootstrap's own defaults apply and it echoes
					// bind_ip as null.
					"n1": {},
				},
			})

		require.NoError(t, err)
	})

	t.Run("refuses a side-car that accepted the run but ignored its settings", func(t *testing.T) {
		t.Parallel()

		// Raised in review: om_bootstrap's own TriggerRunRequest is a plain
		// pydantic model, so an app older than percona/PMM Extensions#1534 ignores these
		// fields rather than rejecting them -- the run would be accepted and come
		// up on PMM Extensions' defaults, with a member meant to be hidden and non-voting
		// joining as an ordinary one and nothing saying so. PMM Extensions echoes the
		// configuration it accepted, so an older one is the run below: no
		// data_path, no member_configs.
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
		// Every setting it dropped, not the first. This side-car is older than
		// all four fields, so naming one made the gap look like a single mis-set
		// value rather than what it is -- a PMM talking to an older PMM Extensions.
		for _, setting := range []string{"the data path", "the log path", "the port", "the bind address"} {
			assert.Contains(t, message, setting)
		}
		assert.Contains(t, message, "older than this PMM")
		// And the run it would not configure is not left running.
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
		// Not t.Parallel(): storeTestDB drops and recreates one fixed-name
		// database, which two parallel subtests would race on -- see
		// TestRegisterBootstrapHost's own subtests, which follow the same rule.
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
		// Not t.Parallel() -- see the previous subtest's own comment.
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
		// confirm_monitoring is PMM's own synthetic step, appended regardless of what
		// PMM Extensions returned -- "pending" here because the run itself has not succeeded yet,
		// same as rollback_steps stay "pending" until a run actually rolls back.
		assert.Equal(t, "confirm_monitoring", response.GetHosts()[0].GetFinalizeSteps()[1].GetName())
		assert.Equal(t, "pending", response.GetHosts()[0].GetFinalizeSteps()[1].GetStatus())
		require.Len(t, response.GetRunSteps(), 1)
		assert.Equal(t, "rs_initiate", response.GetRunSteps()[0].GetName())
	})

	t.Run("confirms monitoring against the inventory app once the run has succeeded", func(t *testing.T) {
		t.Parallel()

		// Two responses in order: the bootstrap run itself, then the inventory app's
		// own host list that confirmMonitoringLookup fetches once it sees "succeeded".
		// n1 already has a service the inventory sweep noticed; n2's bootstrap
		// succeeded too but the sweep has not caught up to it yet.
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
			// The second response: confirmMonitoringLookup's own GET /hosts, fetched
			// once because run-abc is "succeeded" -- run-def being "running" would
			// never trigger it on its own. n1 already has a service.
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
		// confirm_monitoring is the last finalize step on every host, appended by PMM
		// itself -- "succeeded" here because n1 already has a service.
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

		// A field the deployment owns outright is listed rather than hidden, so a UI can
		// show it greyed out instead of leaving the reader to wonder where it went.
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

		// The app answers a PATCH with one row per key *named in the request*. That is
		// misleading for a nested write -- overriding a parent moves what its children
		// resolve to, and no row says so -- so the handler reads the configuration back
		// and answers with that. The two canned bodies differ precisely so this asserts
		// which one was returned.
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
		// Two rows, from the read-back -- not the one row the write echoed.
		require.Len(t, response.GetSettings(), 2)
		assert.Equal(t, "CREDENTIALS_PATH", response.GetSettings()[1].GetKey())
	})

	t.Run("a failed read-back does not report the write as failed", func(t *testing.T) {
		t.Parallel()

		// The write landed. Answering with an error would tell a caller to retry a change
		// that already applied, so the narrower body is served instead.
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

		// Not forwarded: an empty PATCH would succeed on the app's side and change
		// nothing, so a caller who built the body wrongly would see success.
		stub := newSEPStub(t, http.StatusOK, configBody)

		_, err := stub.service(t).UpdateInventoryConfig(t.Context(),
			&omv1.UpdateInventoryConfigRequest{})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Empty(t, stub.method, "nothing should have been sent")
	})

	t.Run("a batch wider than the settings class is refused here", func(t *testing.T) {
		t.Parallel()

		// Nothing bounds this on either hop: protojson takes 200k fields inside the 4MB
		// gRPC default, and the app would then attempt every one of them.
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

		// The app is Python, where the default recursion limit is 1000, while protojson
		// accepts nesting just short of 10k. Forwarding that would make PMM the thing
		// that broke PMM Extensions.
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

		// The bounds must not touch what the settings class actually looks like: a
		// whole-object write of SCHEDULE, which is the deepest legitimate shape.
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

	// A refresh attempts hosts as well as the services on them. Counting only
	// services made a refresh of a host with no database read as "0 of 0", which is
	// what a run that did nothing also looks like -- on the one kind of host OM most
	// exists to describe.
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
	// Zero services is the honest answer for a host-only refresh, and the reason the
	// host counters had to exist rather than the service ones being reinterpreted.
	assert.Equal(t, int32(0), counts.GetTotalServices())
}

func TestInventoryRunDetailIsHostOriented(t *testing.T) {
	t.Parallel()

	// A refresh attempts hosts, so the receipt lists hosts. A flat service list -- which
	// this was -- cannot show a machine carrying a PMM client and no database, however
	// many times it is probed, and that machine is the case OM most exists to describe.
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
	// The pointer to this attempt's raw output -- the observations deliberately not
	// kept on the receipt itself.
	assert.Equal(t, int64(4711), withDatabase.GetTaskHistoryId())
	require.Len(t, withDatabase.GetServices(), 1)
	assert.Equal(t, "mongo-1", withDatabase.GetServices()[0].GetServiceName())

	// The row a service-oriented receipt could not produce at all: probed, answered,
	// and carrying no services because there is no database on it.
	bare := res.GetEntities()[1]
	assert.Equal(t, "pmm-client-node00", bare.GetHostName())
	assert.True(t, bare.GetAnswered())
	assert.Empty(t, bare.GetServices())

	// An orphan keeps its row rather than being dropped: "2 of 3 answered" cannot say
	// which one was missed, and the orphan is the one worth acting on.
	orphan := res.GetEntities()[2]
	assert.Equal(t, omv1.ExecutorResolution_EXECUTOR_RESOLUTION_ORPHANED, orphan.GetResolution())
	assert.False(t, orphan.GetAnswered())
	// Asserted on the fields rather than the getters: these are proto3 `optional`
	// scalars, so the getter returns the zero value for an unset field and only the
	// pointer distinguishes "no executor host" from "an empty one". protojson omits
	// the key altogether, which is the wire form the UI reads as absent.
	assert.Nil(t, orphan.ExecutorHost)
	assert.Nil(t, orphan.DurationSeconds)
	assert.Nil(t, orphan.TaskHistoryId)
	assert.Equal(t, "no executor host", orphan.GetError())
}

func TestInventoryBearerIsSent(t *testing.T) {
	t.Parallel()

	// The browser holds no PMM Extensions token -- that is the point of proxying -- so this hop is
	// the only place the app's credential is presented. Without it every request is a
	// 401 the page cannot explain.
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

// ensure the canned bodies stay valid JSON as they are edited.
func TestInventoryFixturesAreValid(t *testing.T) {
	t.Parallel()

	// hosts is checked separately from config: it is wrapped in PMM Extensions' own
	// paginated envelope (PMM-15326: "Bound the estate listings") and config
	// deliberately is not -- see extensionsPage's own comment on which endpoints
	// changed shape and which stayed a bare array.
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

// TestNudgeDebouncesExpire covers the half of each debounce the end-to-end tests
// in TestCompleteSucceededRun cannot reach: they prove a second ask inside the
// window is suppressed, but nothing proved the window ever reopens. Both
// failures are silent and permanent -- a node or a server that stops asking
// leaves every later run waiting out bootstrapInventoryRefreshWindow before
// confirm_monitoring resolves.
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
