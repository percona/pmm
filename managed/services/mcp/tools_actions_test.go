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

package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

// sequenceFixtures answers a path with successive fixtures (polling).
type sequenceFixtures struct {
	files []string
	calls atomic.Int32
}

func actionRoutes(t *testing.T, start, done string) (*fakePMM, *sequenceFixtures) {
	t.Helper()

	routes := qanRoutes()
	routes["POST "+reportPath] = fixture{file: reportOne}
	routes["GET /v1/qan/query/QID-AAA/plan"] = fixture{file: "plan_empty.json"}
	routes["GET /v1/qan/query/QID-PG/plan"] = fixture{file: "plan.json"}
	routes["POST /v1/actions:startServiceAction"] = fixture{file: start}
	seq := &sequenceFixtures{files: []string{"action_running.json", "action_running.json", done}}
	fake := newFakePMM(t, routes)

	// GetAction polls: two "not done" answers, then the final fixture.
	prev := fake.server.Config.Handler
	fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/v1/actions/") {
			i := int(seq.calls.Add(1)) - 1
			if i >= len(seq.files) {
				i = len(seq.files) - 1
			}
			fake.mu.Lock()
			fake.requests = append(fake.requests, recordedRequest{method: req.Method, path: req.URL.Path, header: req.Header.Clone()})
			fake.mu.Unlock()
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write(readFixture(t, seq.files[i]))
			return
		}
		prev.ServeHTTP(rw, req)
	})
	return fake, seq
}

// explainRoutes returns the routes of an explain call on a service: the resolver's report names it.
func explainRoutes(report string, extra map[string]fixture) map[string]fixture {
	routes := withRoutes(qanRoutes(), map[string]fixture{
		"POST " + reportPath: {file: report},
	})
	return withRoutes(routes, extra)
}

// mongoExplain makes a fake PMM resolve a MongoDB query that has a stored example.
func mongoExplain(fake *fakePMM) {
	fake.routes["POST "+reportPath] = fixture{file: "report_by_service_mongo.json"}
	fake.routes["POST /v1/qan:getMetrics"] = fixture{file: "metrics_mongo.json"}
	fake.routes["POST /v1/qan/query:getExample"] = fixture{file: "example_mongo.json"}
}

func TestExplainTool(t *testing.T) {
	t.Parallel()

	t.Run("MySQLLive", func(t *testing.T) {
		t.Parallel()

		fake, seq := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		text, isError := textOf(t, res), res.IsError
		assert.False(t, isError)
		assert.True(t, strings.HasPrefix(text, "EXPLAIN (json, mysql shop-mysql):\n```\n-- SELECT * FROM customers WHERE email = 'user42@example.com'\n{\n  \"query_block\""), text)
		assert.Contains(t, text, `"access_type": "ALL"`)
		assert.Contains(t, text, "details_tab=explain&filter_by=QID-AAA")
		fake.assertForwardedAuth(t)

		got := structured[explainResult](t, res)
		assert.Equal(t, engineMySQL, got.Engine)
		assert.Equal(t, "shop-mysql", got.ServiceName)
		assert.Equal(t, "json", got.Format)
		assert.True(t, got.Live)
		assert.False(t, got.Truncated)
		assert.Equal(t, &queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: testNow.Add(-time.Hour), PeriodTo: testNow}, got.QueryRef)
		assert.Contains(t, text, "```\n"+got.Plan+"\n```\n"+livePlanNote+"\n\n", "the plan the text shows, then the note")
		assert.Equal(t, livePlanNote, got.Note)
		assert.Equal(t, "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?details_tab=explain&filter_by=QID-AAA"+
			"&from="+ms(testNow.Add(-time.Hour))+"&query_selected=true&to="+ms(testNow)+"&var-service_name=shop-mysql", got.PMMURL)
		assert.Contains(t, text, "("+got.PMMURL+")")

		// MySQL has no stored plans, so there is no probe: the action is started
		// and polled straight away.
		assert.Empty(t, fake.requestsTo("/v1/qan/query/QID-AAA/plan"))
		starts := fake.requestsTo("/v1/actions:startServiceAction")
		require.Len(t, starts, 1)
		body := unmarshalBody[actions_service.StartServiceActionBody](t, starts[0])
		require.NotNil(t, body.MysqlExplainJSON)
		assert.Equal(t, "svc-1", body.MysqlExplainJSON.ServiceID)
		assert.Equal(t, "QID-AAA", body.MysqlExplainJSON.QueryID)
		assert.Equal(t, "shop", body.MysqlExplainJSON.Database, "the database QAN recorded")
		assert.EqualValues(t, 3, seq.calls.Load())
		assert.Equal(t, "/v1/actions//action_id/explain-1", fake.requestsTo("/v1/actions//action_id/explain-1")[0].path)
	})

	t.Run("MySQLTraditionalRedacted", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, false))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA", "format": "traditional", "placeholders": []string{"a"}})
		assert.False(t, isError)
		assert.NotContains(t, text, "user42@example.com'\n{")
		assert.True(t, strings.HasPrefix(text, "EXPLAIN (traditional, mysql shop-mysql):\n```\n{\n"), text)
		body := unmarshalBody[actions_service.StartServiceActionBody](t, fake.requestsTo("/v1/actions:startServiceAction")[0])
		require.NotNil(t, body.MysqlExplain)
		assert.Equal(t, []string{"a"}, body.MysqlExplain.Placeholders)
	})

	t.Run("StoredPlan", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		fake.routes["POST "+reportPath] = fixture{file: "report_by_service_postgres.json"}
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-PG"})
		text, isError := textOf(t, res), res.IsError
		assert.False(t, isError)
		assert.True(t, strings.HasPrefix(text, "Stored plan (pg_stat_monitor, planid PLAN-1):\n```\nSeq Scan on customers"), text)
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))

		got := structured[explainResult](t, res)
		assert.Equal(t, explainResult{
			Engine:      enginePostgreSQL,
			ServiceName: "shop-postgres",
			Format:      formatText,
			Plan:        "Seq Scan on customers  (cost=0.00..458.00 rows=1 width=64)\n  Filter: ((email)::text = 'user42@example.com'::text)",
			QueryRef:    &queryRef{QueryID: "QID-PG", ServiceID: "svc-2", PeriodFrom: testNow.Add(-time.Hour), PeriodTo: testNow},
			PMMURL: "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?details_tab=explain&filter_by=QID-PG" +
				"&from=" + ms(testNow.Add(-time.Hour)) + "&query_selected=true&to=" + ms(testNow) + "&var-service_name=shop-postgres",
		}, got)
		assert.Contains(t, text, "```\n"+got.Plan+"\n```\n\n[View this query in PMM QAN ↗]("+got.PMMURL+")")
		assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), `"note"`)
		assert.NotContains(t, text, "Live plan")

		reports := fake.requestsTo(reportPath)
		require.Len(t, reports, 1, "the queryid alone went to the resolver")
		plans := fake.requestsTo("/v1/qan/query/QID-PG/plan")
		require.Len(t, plans, 1)
		assert.Equal(t, "service_id=svc-2", plans[0].query, "the plan is requested for the resolved service")
	})

	t.Run("PostgreSQLWithoutPlan", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		fake.routes["POST "+reportPath] = fixture{file: "report_by_service_postgres.json"}
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: not_found\nno stored plan for this query. "), text)
		assert.NotContains(t, text, "QID-AAA")
		assert.Contains(t, text, "pgsm_enable_query_plan=on")
	})

	t.Run("PlanCapped", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = strings.Repeat("x", 70<<10)
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		got := structured[explainResult](t, res)
		assert.True(t, got.Truncated)
		assert.Equal(t, strings.Repeat("x", 64<<10), got.Plan)
		assert.Contains(t, textOf(t, res), "```\n"+got.Plan+"\n```\n(plan cut at 64 KiB)\n"+livePlanNote+"\n\n[View this query in PMM QAN ↗]("+got.PMMURL+")")
	})

	t.Run("MongoDBStoredExample", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		mongoExplain(fake)
		stub.explain = strings.TrimSpace(string(readFixture(t, "mongo_explain.json")))
		session := connect(t, newQANService(t, fake, false))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA", "format": "traditional"})
		got := structured[explainResult](t, res)
		text := textOf(t, res)
		assert.Equal(t, engineMongoDB, got.Engine)
		assert.Equal(t, "shop-mongo", got.ServiceName)
		assert.Equal(t, "json", got.Format, "MongoDB's explain answers JSON whatever format asks for")
		assert.True(t, got.Live)
		assert.Contains(t, got.Plan, `"stage":"COLLSCAN"`)
		assert.True(t, strings.HasPrefix(text, "EXPLAIN (json, mongodb shop-mongo):\n```\n"+got.Plan+"\n```\n"+livePlanNote+"\n\n"), text)
		assert.Equal(t, livePlanNote, got.Note)
		for _, literal := range []string{"user42@example.com", "4417"} {
			assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), literal)
			assert.NotContains(t, text, literal)
		}
		assert.Equal(t, "svc-mongo", got.QueryRef.ServiceID)
		assert.Contains(t, got.PMMURL, "details_tab=explain&filter_by=QID-AAA")
		assert.Contains(t, got.PMMURL, "var-service_name=shop-mongo")

		require.Len(t, stub.starts, 1)
		require.NotNil(t, stub.starts[0].MongodbExplain)
		assert.Equal(t, "svc-mongo", stub.starts[0].MongodbExplain.ServiceID)
		assert.Equal(t, `db.customers.find({"email": "user42@example.com", "age": {"$gt": 4417}})`, stub.starts[0].MongodbExplain.Query)
	})

	t.Run("MongoDBWithoutExample", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		mongoExplain(fake)
		fake.routes["POST /v1/qan/query:getExample"] = fixture{file: "example_empty.json"}
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: no_query_source\nno stored query example for queryid 'QID-AAA' in 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z"), text)
		assert.NotContains(t, text, "pass an explicit query")
		assert.Empty(t, stub.starts)
	})

	// A failed example call is not an absent example: its error is the answer.
	t.Run("MongoDBExampleFetchFails", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		mongoExplain(fake)
		failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/qan/query:getExample" })
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
		assert.NotContains(t, text, "no_query_source")
		assert.Empty(t, stub.starts)
	})

	t.Run("MySQLDetailFetchFails", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/qan/query:getExample" })
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
		assert.Empty(t, stub.starts)
	})

	t.Run("QueryRefForm", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		from, to := testNow.Add(-3*time.Hour), testNow.Add(-2*time.Hour)
		ref := map[string]any{"queryid": "QID-AAA", "service_id": "svc-1", "period_from": from.Format(time.RFC3339), "period_to": to.Format(time.RFC3339)}
		res := callTool(t, session, "pmm_get_explain", map[string]any{"query_ref": ref})
		got := structured[explainResult](t, res)
		assert.Equal(t, &queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: from, PeriodTo: to}, got.QueryRef)
		assert.Contains(t, got.PMMURL, "from="+ms(from)+"&query_selected=true&to="+ms(to))
	})

	t.Run("QueryRefWithQueryID", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		ref := map[string]any{"queryid": "QID-AAA", "period_from": "2026-09-14T11:00:00Z", "period_to": "2026-09-14T12:00:00Z"}
		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"query_ref": ref, "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\npass query_ref alone"), text)
		assert.Empty(t, fake.requestsTo(reportPath))
	})

	t.Run("DatabaseOverridesResolved", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		_, isError := callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA", "database": "other"})
		require.False(t, isError)
		body := unmarshalBody[actions_service.StartServiceActionBody](t, fake.requestsTo("/v1/actions:startServiceAction")[0])
		require.NotNil(t, body.MysqlExplainJSON)
		assert.Equal(t, "other", body.MysqlExplainJSON.Database)
		assert.Equal(t, "svc-1", body.MysqlExplainJSON.ServiceID, "the resolved service")
	})

	t.Run("PostgreSQLIgnoresDetailError", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		fake.routes["POST "+reportPath] = fixture{file: "report_by_service_postgres.json"}
		failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/qan/query:getExample" })
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-PG"})
		got := structured[explainResult](t, res)
		assert.Equal(t, enginePostgreSQL, got.Engine)
		assert.Contains(t, got.Plan, "Seq Scan on customers")
	})

	t.Run("PlaceholdersCountError", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		fake.routes["POST /v1/actions:startServiceAction"] = fixture{file: "error_400_placeholders.json", status: http.StatusBadRequest}
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.Equal(t, "error: invalid_input\nthe number of placeholders doesn't match the statement: pass one value per ? in the fingerprint, "+
			"in order, or none when QAN stored a query example", text)
		assert.Contains(t, text, "placeholders")
		assert.NotContains(t, text, "EXPLAIN could not run")
	})

	t.Run("PlaceholderError", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_error_1064.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nEXPLAIN could not run on the fingerprint (placeholder syntax): Error 1064"), text)
		assert.Contains(t, text, "pass placeholders")
	})

	t.Run("PrivilegeError", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_error_denied.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: insufficient_privileges\nError 1142"), text)
	})

	t.Run("MongoDBUnsupportedOperation", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		mongoExplain(fake)
		stub.explainErr = "cannot explain this type of query\noperation insert is not supported for explain"
		session := connect(t, newQANService(t, fake, false))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, res.IsError)
		assert.Nil(t, res.StructuredContent)
		assert.Equal(t, map[string]any{"code": "invalid_input", "retryable": false}, res.Meta["com.percona/error"])
		text := textOf(t, res)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\ncannot explain this type of query\noperation insert is not supported for explain"), text)
		assert.Contains(t, text, "MongoDB can explain read operations only (find, aggregate, count, distinct); pick a read query from pmm_top_queries")
	})

	t.Run("TrimmedQuery", func(t *testing.T) {
		t.Parallel()

		routes := explainRoutes(reportOne, map[string]fixture{
			"GET /v1/qan/query/QID-AAA/plan":      {file: "plan_empty.json"},
			"POST /v1/actions:startServiceAction": {file: "error_500_query_trimmed.json", status: http.StatusInternalServerError},
		})
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		assert.True(t, res.IsError)
		assert.Nil(t, res.StructuredContent)
		assert.Equal(t, map[string]any{"code": "invalid_input", "retryable": false}, res.Meta["com.percona/error"])
		text := textOf(t, res)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		assert.Contains(t, text, "QAN stored this query trimmed (pmm-agent max-query-length), so it can't be explained; pick another query, or raise max-query-length on the agent")
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_running.json")
		s := newQANService(t, fake, true)
		s.actionTimeout = func() time.Duration { return 700 * time.Millisecond }
		session := connect(t, s)

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: timeout\naction /action_id/explain-1 did not complete within 700ms"), text)
	})

	t.Run("AgentUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := explainRoutes(reportOne, map[string]fixture{
			"GET /v1/qan/query/QID-AAA/plan":      {file: "plan_empty.json"},
			"POST /v1/actions:startServiceAction": {file: "error_no_agent.json", status: http.StatusBadRequest},
		})
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: agent_unreachable\n"), text)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nservice_id, period_from and period_to need a queryid"), text)

		text, isError = callText(t, session, "pmm_get_explain", map[string]any{})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\npass query_ref"), text)

		text, isError = callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA", "format": "tree"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nformat"), text)
		assert.Empty(t, fake.requestsTo(reportPath))

		text, isError = callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-404", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.Equal(t, "error: not_found\n"+errNotInList(window{From: testNow.Add(-time.Hour), To: testNow}, "").message, text)
	})

	// Only MySQL's 1064 gets the placeholder advice, not any invalid input.
	t.Run("ExplainStartRejectedHasNoPlaceholderAdvice", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		fake.routes["POST /v1/actions:startServiceAction"] = fixture{file: "error_400_invalid.json", status: http.StatusBadRequest}
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input"), text)
		assert.NotContains(t, text, "placeholder syntax")
	})
}

// explainOutput is the output of a mysql_explain_json action that explained query with plan.
func explainOutput(t *testing.T, plan []byte, query string) string {
	t.Helper()

	return string(mustJSON(t, map[string]any{
		"explain_result":  base64.StdEncoding.EncodeToString(plan),
		"explained_query": query,
		"is_dml":          false,
	}))
}

// secretRegion is the literal that secretExplain's statement and its plan's Note 1003 hold.
const secretRegion = "region-secret-42"

// secretExplain returns the explain output of a two-table join whose statement and Note 1003 hold
// secretRegion, and those two texts.
func secretExplain(t *testing.T) (string, []string) {
	t.Helper()

	query := "SELECT o.id, c.name FROM orders o JOIN customers c ON c.id = o.customer_id WHERE c.region = '" + secretRegion + "'"
	plan := strings.ReplaceAll(string(readFixture(t, "explain/mysql-8.0-join-aliases.json")), "'eu'", "'"+secretRegion+"'")
	require.Contains(t, plan, secretRegion)
	return explainOutput(t, []byte(plan), query), []string{query, plan}
}

// postgresQuery makes a fake PMM resolve QID-AAA to the PostgreSQL service, whose example names two tables.
func postgresQuery(fake *fakePMM) {
	fake.routes["POST "+reportPath] = fixture{file: "report_by_service_postgres.json"}
	fake.routes["POST /v1/qan:getMetrics"] = fixture{file: "metrics_pg.json"}
	fake.routes["POST /v1/qan/query:getExample"] = fixture{file: "example_pg.json"}
}

// ddlTables returns the table names of the SHOW CREATE TABLE actions in starts, and fails on any other action.
func ddlTables(t *testing.T, starts []actions_service.StartServiceActionBody, serviceID, database string) []string {
	t.Helper()

	var out []string
	for _, b := range starts {
		switch {
		case b.MysqlShowCreateTable != nil:
			assert.Equal(t, serviceID, b.MysqlShowCreateTable.ServiceID)
			assert.Equal(t, database, b.MysqlShowCreateTable.Database)
			out = append(out, b.MysqlShowCreateTable.TableName)
		case b.PostgresShowCreateTable != nil:
			assert.Equal(t, serviceID, b.PostgresShowCreateTable.ServiceID)
			assert.Equal(t, database, b.PostgresShowCreateTable.Database)
			out = append(out, b.PostgresShowCreateTable.TableName)
		default:
			require.FailNow(t, "not a SHOW CREATE TABLE action", "%+v", b)
		}
	}
	return out
}

func TestSchemaTool(t *testing.T) {
	t.Parallel()

	const (
		lastHour  = "queryid: QID-AAA\nWindow: 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z\n"
		onMySQL   = lastHour + "service: shop-mysql (mysql)\n\n"
		onPostgre = lastHour + "service: shop-postgres (postgresql)\n\n"
	)

	t.Run("MySQL", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}})
		text, isError := textOf(t, res), res.IsError
		assert.False(t, isError)
		assert.Equal(t, "### shop.customers\n```sql\nCREATE TABLE `customers` (\n  `id` bigint NOT NULL AUTO_INCREMENT,\n  `email` varchar(255) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB\n```", text)
		fake.assertForwardedAuth(t)
		assert.Equal(t, schemaResult{Tables: []tableResult{{
			Database:   "shop",
			Name:       "customers",
			Definition: "CREATE TABLE `customers` (\n  `id` bigint NOT NULL AUTO_INCREMENT,\n  `email` varchar(255) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB",
		}}}, structured[schemaResult](t, res))
		body := unmarshalBody[actions_service.StartServiceActionBody](t, fake.requestsTo("/v1/actions:startServiceAction")[0])
		require.NotNil(t, body.MysqlShowCreateTable)
		assert.Equal(t, "customers", body.MysqlShowCreateTable.TableName)
		assert.Equal(t, "shop", body.MysqlShowCreateTable.Database)
		assert.Empty(t, fake.requestsTo(reportPath), "the tables are named: no query to resolve")
		assert.Empty(t, fake.requestsTo("/v1/qan:getMetrics"))
	})

	t.Run("PostgreSQLWithIndexes", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-2", "database": "shop", "tables": []string{"customers"}, "info_types": "all"})
		text, isError := textOf(t, res), res.IsError
		assert.False(t, isError)
		assert.Equal(t, "### shop.customers\n```sql\nCREATE TABLE `customers` (`id` int)\n```\nindexes:\n```\nindex of customers\n```", text)
		assert.Equal(t, schemaResult{Tables: []tableResult{{
			Database: "shop", Name: "customers", Definition: "CREATE TABLE `customers` (`id` int)", Indexes: "index of customers",
		}}}, structured[schemaResult](t, res))
		require.Len(t, stub.starts, 2)
		assert.True(t, slices.ContainsFunc(stub.starts, func(b actions_service.StartServiceActionBody) bool { return b.PostgresShowCreateTable != nil }))
		assert.True(t, slices.ContainsFunc(stub.starts, func(b actions_service.StartServiceActionBody) bool { return b.PostgresShowIndex != nil }))
	})

	t.Run("TableTextCapped", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.outputs = map[string]string{"ddl-a": strings.Repeat("d", 40<<10), "index-b": strings.Repeat("i", 33<<10)}
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a", "b", "c"}, "info_types": "all"})
		got := structured[schemaResult](t, res)
		require.Len(t, got.Tables, 3)
		assert.Equal(t, tableResult{Database: "shop", Name: "a", Definition: strings.Repeat("d", 32<<10), Indexes: "index of a", Truncated: true}, got.Tables[0])
		assert.Equal(t, tableResult{Database: "shop", Name: "b", Definition: "CREATE TABLE `b` (`id` int)", Indexes: strings.Repeat("i", 32<<10), Truncated: true}, got.Tables[1])
		assert.False(t, got.Tables[2].Truncated)
		assert.False(t, got.Partial)

		text := textOf(t, res)
		assert.Contains(t, text, "### shop.a\n```sql\n"+got.Tables[0].Definition+"\n```\n(definition cut at 32 KiB)\nindexes:\n```\nindex of a\n```\n\n")
		assert.Contains(t, text, "### shop.b\n```sql\nCREATE TABLE `b` (`id` int)\n```\nindexes:\n```\n"+got.Tables[1].Indexes+"\n```\n(indexes cut at 32 KiB)\n\n")
		assert.True(t, strings.HasSuffix(text, "### shop.c\n```sql\nCREATE TABLE `c` (`id` int)\n```\nindexes:\n```\nindex of c\n```"), text)
	})

	// A syntax error from SHOW CREATE TABLE is not EXPLAIN's placeholder
	// problem, and the named-tables form takes no placeholders.
	t.Run("SyntaxErrorHasNoExplainAdvice", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_error_1064.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nError 1064"), text)
		assert.NotContains(t, text, "placeholders")
	})

	t.Run("QueryIDAloneReachesTheResolver", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, readFixture(t, "explain/mysql-8.0-join-aliases.json"), "SELECT 1")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		require.False(t, isError, text)
		reports := fake.requestsTo(reportPath)
		require.Len(t, reports, 1)
		body := unmarshalBody[qan_service.GetReportBody](t, reports[0])
		assert.Equal(t, testNow.Add(-time.Hour), time.Time(body.PeriodStartFrom).UTC(), "a queryid alone searches the last hour")
		assert.Equal(t, testNow, time.Time(body.PeriodStartTo).UTC())
		labels := map[string][]string{}
		for _, l := range body.Labels {
			labels[l.Key] = l.Value
		}
		assert.Equal(t, []string{"QID-AAA"}, labels["queryid"])
		assert.ElementsMatch(t, []string{"svc-1", "svc-2", "svc-mongo", "svc-pmm-pg"}, labels["service_id"], "the caller's service list")
		assert.True(t, strings.HasPrefix(text, onMySQL), text)
	})

	t.Run("PostgreSQLByQueryID", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		postgresQuery(fake)
		s := newQANService(t, fake, true)
		var rawReads atomic.Int32
		s.rawSQL = func() bool {
			rawReads.Add(1)
			return true
		}
		session := connect(t, s)

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA", "placeholders": []string{"ignored"}})
		assert.Equal(t, schemaResult{
			Tables: []tableResult{
				{Database: "shop_pg", Name: "orders", Definition: "CREATE TABLE `orders` (`id` int)"},
				{Database: "shop_pg", Name: "customers", Definition: "CREATE TABLE `customers` (`id` int)"},
			},
			QueryRef: &queryRef{QueryID: "QID-AAA", ServiceID: "svc-2", PeriodFrom: testNow.Add(-time.Hour), PeriodTo: testNow},
		}, structured[schemaResult](t, res), "the example's tables, in its order, and no live-plan note")
		assert.Equal(t, onPostgre+
			"### shop_pg.orders\n```sql\nCREATE TABLE `orders` (`id` int)\n```\n\n"+
			"### shop_pg.customers\n```sql\nCREATE TABLE `customers` (`id` int)\n```", textOf(t, res))
		assert.ElementsMatch(t, []string{"orders", "customers"}, ddlTables(t, stub.starts, "svc-2", "shop_pg"), "no EXPLAIN for PostgreSQL")
		assert.Zero(t, rawReads.Load(), "PostgreSQL renders no statement")
	})

	t.Run("PostgreSQLManyTables", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		postgresQuery(fake)
		fake.routes["POST /v1/qan/query:getExample"] = fixture{file: "example_pg_many_tables.json"}
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		got := structured[schemaResult](t, res)
		names := make([]string, 0, len(got.Tables))
		for _, tbl := range got.Tables {
			names = append(names, tbl.Name)
		}
		assert.Equal(t, strings.Split("public.t01,public.t02,public.t03,public.t04,public.t05,public.t06,public.t07,public.t08,public.t09,public.t10", ","), names,
			"the first 10, in QAN's order")
		assert.Equal(t, []string{"shop_pg.public.t11", "shop_pg.public.t12"}, got.NotFetched)
		text := textOf(t, res)
		assert.Len(t, headers(text), 10)
		assert.True(t, strings.HasSuffix(text, "\n\nNot fetched, the 10-table cap was reached: shop_pg.public.t11, shop_pg.public.t12; "+
			"pass them as service_id, database and tables."), text)
		assert.Len(t, ddlTables(t, stub.starts, "svc-2", "shop_pg"), 10)
	})

	t.Run("PostgreSQLWithoutTables", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		postgresQuery(fake)
		fake.routes["POST /v1/qan/query:getExample"] = fixture{file: "example_pg_no_tables.json"}
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.Equal(t, "error: no_query_source\nQAN has no tables for queryid 'QID-AAA' in 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z; "+
			"pass service_id, database and tables to name them", text)
		assert.Empty(t, stub.starts)
	})

	// A failed example call is not an example without tables: its error is the answer.
	t.Run("PostgreSQLExampleFetchFails", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		postgresQuery(fake)
		failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/qan/query:getExample" })
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
		assert.Empty(t, stub.starts)
	})

	t.Run("MySQLByQueryID", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, readFixture(t, "explain/mysql-8.0-derived-table.json"), "SELECT 1")
		s := newQANService(t, fake, true)
		var rawReads atomic.Int32
		s.rawSQL = func() bool {
			rawReads.Add(1)
			return true
		}
		session := connect(t, s)

		from, to := testNow.Add(-3*time.Hour), testNow.Add(-2*time.Hour)
		res := callTool(t, session, "pmm_get_schema", map[string]any{
			"queryid": "QID-AAA", "period_from": from.Format(time.RFC3339), "period_to": to.Format(time.RFC3339), "placeholders": []string{"7"},
		})
		assert.Equal(t, schemaResult{
			Tables: []tableResult{
				{Database: "shop", Name: "customers", Definition: "CREATE TABLE `customers` (`id` int)"},
				{Database: "shop", Name: "orders", Definition: "CREATE TABLE `orders` (`id` int)"},
			},
			Unresolved: []string{"d"},
			QueryRef:   &queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: from, PeriodTo: to},
			Note:       liveTablesNote,
		}, structured[schemaResult](t, res), "the plan's tables in plan order, its derived table unresolved, the window used")
		assert.Equal(t, "queryid: QID-AAA\nWindow: 2026-09-14T09:00:00Z – 2026-09-14T10:00:00Z\nservice: shop-mysql (mysql)\n\n"+
			"### shop.customers\n```sql\nCREATE TABLE `customers` (`id` int)\n```\n\n"+
			"### shop.orders\n```sql\nCREATE TABLE `orders` (`id` int)\n```\n\n"+
			"Unresolved, not fetched (derived tables, CTEs, unions and names the plan doesn't tie to a table): d; "+
			"pass the tables behind them as service_id, database and tables.\n"+
			"Tables found in a live plan: what the database would choose now, not during the window.", textOf(t, res))
		assert.EqualValues(t, 1, rawReads.Load(), "raw SQL is read once")

		require.Len(t, stub.starts, 3)
		assert.Equal(t, &actions_service.StartServiceActionParamsBodyMysqlExplainJSON{
			ServiceID: "svc-1", QueryID: "QID-AAA", Database: "shop", Placeholders: []string{"7"},
		}, stub.starts[0].MysqlExplainJSON, "one EXPLAIN first, on the resolved service and database")
		assert.ElementsMatch(t, []string{"customers", "orders"}, ddlTables(t, stub.starts[1:], "svc-1", "shop"))

		reports := fake.requestsTo(reportPath)
		require.Len(t, reports, 1)
		body := unmarshalBody[qan_service.GetReportBody](t, reports[0])
		assert.Equal(t, from, time.Time(body.PeriodStartFrom).UTC(), "the resolver searched the window passed")
	})

	t.Run("MySQLQueryRef", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, readFixture(t, "explain/mysql-8.4-v2-join-aliases.json"), "SELECT 1")
		session := connect(t, newQANService(t, fake, true))

		from, to := testNow.Add(-5*time.Hour), testNow.Add(-4*time.Hour)
		ref := map[string]any{"queryid": "QID-AAA", "service_id": "svc-1", "period_from": from.Format(time.RFC3339), "period_to": to.Format(time.RFC3339)}
		got := structured[schemaResult](t, callTool(t, session, "pmm_get_schema", map[string]any{"query_ref": ref, "info_types": "indexes"}))
		assert.Equal(t, &queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: from, PeriodTo: to}, got.QueryRef)
		assert.Equal(t, []tableResult{
			{Database: "shop", Name: "customers", Indexes: "index of customers"},
			{Database: "shop", Name: "orders", Indexes: "index of orders"},
		}, got.Tables)
		assert.Empty(t, got.Unresolved)
		require.Len(t, stub.starts, 3)
		require.NotNil(t, stub.starts[0].MysqlExplainJSON)
	})

	t.Run("MySQLPlaceholderError", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explainErr = "Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server " +
			"version for the right syntax to use near '?' at line 1"
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		text := textOf(t, res)
		assert.True(t, res.IsError)
		assert.Equal(t, map[string]any{"code": "invalid_input", "retryable": false}, res.Meta["com.percona/error"])
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nEXPLAIN could not run on the fingerprint (placeholder syntax): Error 1064"), text)
		assert.Contains(t, text, "pass placeholders (one value per ?, in order), or pass service_id, database and tables")
		require.Len(t, stub.starts, 1, "no DDL action after a failed EXPLAIN")
		assert.NotNil(t, stub.starts[0].MysqlExplainJSON)
	})

	t.Run("MySQLPlaceholdersCountError", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		fake.routes["POST /v1/actions:startServiceAction"] = fixture{file: "error_400_placeholders.json", status: http.StatusBadRequest}
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA", "placeholders": []string{"1", "2"}})
		assert.True(t, isError)
		assert.Equal(t, "error: invalid_input\nthe number of placeholders doesn't match the statement: pass one value per ? in the fingerprint, "+
			"in order, or none when QAN stored a query example; or pass service_id, database and tables instead of a query reference", text)
		assert.Len(t, fake.requestsTo("/v1/actions:startServiceAction"), 1)
	})

	// A failed metrics call can drop the database the EXPLAIN needs: its error is the answer.
	t.Run("MySQLDetailFetchFails", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, readFixture(t, "explain/mysql-8.0-join-aliases.json"), "SELECT 1")
		failWhen(t, fake, func(req *http.Request) bool { return req.URL.Path == "/v1/qan:getMetrics" })
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\n"), text)
		assert.Empty(t, stub.starts)
	})

	// The explained statement only feeds the alias map: even with raw SQL on it is not returned.
	t.Run("MySQLStatementNeverRendered", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain, _ = secretExplain(t)
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		got := structured[schemaResult](t, res)
		require.Len(t, got.Tables, 2, "the join's tables were found")
		assert.NotContains(t, textOf(t, res), secretRegion)
		assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), secretRegion)
	})

	t.Run("MySQLPlanWithoutTables", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, []byte(`{"query_block": {"select_id": 1, "message": "No tables used"}}`), "SELECT 1")
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		got := structured[schemaResult](t, res)
		assert.Empty(t, got.Tables)
		assert.Contains(t, string(mustJSON(t, res.StructuredContent)), `"tables":[]`)
		assert.Equal(t, onMySQL+"No base table found in the plan.\n\n"+liveTablesNote, textOf(t, res))
		require.Len(t, stub.starts, 1, "the EXPLAIN only")
	})

	t.Run("MySQLPlanAllUnresolved", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		plan := `{"query_block": {"select_id": 1, "table": {"table_name": "d", "access_type": "ALL",
			"materialized_from_subquery": {"query_block": {"select_id": 2, "message": "No tables used"}}}}}`
		stub.explain = explainOutput(t, []byte(plan), "SELECT 1")
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		got := structured[schemaResult](t, res)
		assert.Empty(t, got.Tables)
		assert.Equal(t, []string{"d"}, got.Unresolved)
		assert.Equal(t, onMySQL+"No base table found in the plan.\n\n"+
			"Unresolved, not fetched (derived tables, CTEs, unions and names the plan doesn't tie to a table): d; "+
			"pass the tables behind them as service_id, database and tables.\n"+liveTablesNote, textOf(t, res))
		require.Len(t, stub.starts, 1, "the EXPLAIN only")
	})

	t.Run("TwelveTables", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, readFixture(t, "explain_join_12.json"), "SELECT 1")
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		got := structured[schemaResult](t, res)
		names := make([]string, 0, len(got.Tables))
		for _, tbl := range got.Tables {
			names = append(names, tbl.Name)
		}
		assert.Equal(t, strings.Split("t01,t02,t03,t04,t05,t06,t07,t08,t09,t10", ","), names, "the first 10, in plan order")
		assert.Equal(t, []string{"shop.t11", "shop.t12"}, got.NotFetched)
		assert.Empty(t, got.Unresolved)

		text := textOf(t, res)
		assert.Len(t, headers(text), 10)
		assert.Contains(t, text, "\n\nNot fetched, the 10-table cap was reached: shop.t11, shop.t12; pass them as service_id, database and tables.\n"+liveTablesNote)
		assert.Len(t, ddlTables(t, stub.starts[1:], "svc-1", "shop"), 10)
	})

	// MongoDB is refused before any slot is taken: a busy service or caller does not turn it into rate_limited.
	t.Run("MongoDB", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		mongoExplain(fake)
		s := newQANService(t, fake, true)
		session := connect(t, s)
		key := callerKey(callerAuth{authorization: testToken})
		relService, hit := s.liveByService.acquire("svc-mongo")
		require.Equal(t, limitNone, hit)
		defer relService()
		rel1, hit := s.live.acquire(key)
		require.Equal(t, limitNone, hit)
		defer rel1()
		rel2, hit := s.live.acquire(key)
		require.Equal(t, limitNone, hit)
		defer rel2()

		for _, args := range []map[string]any{
			{"queryid": "QID-AAA"},
			{"service_id": "svc-mongo", "database": "shop", "tables": []string{"customers"}},
		} {
			res := callTool(t, session, "pmm_get_schema", args)
			text := textOf(t, res)
			assert.True(t, res.IsError, "%v", args)
			assert.Equal(t, "error: invalid_input\nSHOW CREATE TABLE is not available for engine 'mongodb'", text, "%v", args)
		}
		assert.Empty(t, stub.starts)
	})

	t.Run("RateLimitedBeforeAnyAction", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		stub.explain = explainOutput(t, readFixture(t, "explain/mysql-8.0-join-aliases.json"), "SELECT 1")
		s := newQANService(t, fake, true)
		session := connect(t, s)
		key := callerKey(callerAuth{authorization: testToken})
		rel1, hit := s.live.acquire(key)
		require.Equal(t, limitNone, hit)
		rel2, hit := s.live.acquire(key)
		require.Equal(t, limitNone, hit)

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\n"), text)
		assert.Contains(t, text, "these credentials")
		assert.Empty(t, stub.starts)
		rel1()
		rel2()

		held, hit := s.liveByService.acquire("svc-1")
		require.Equal(t, limitNone, hit)
		text, isError = callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited\na live-inspection call on this service is already in flight"), text)
		assert.Empty(t, stub.starts)
		held()

		text, isError = callText(t, session, "pmm_get_schema", map[string]any{"queryid": "QID-AAA"})
		assert.False(t, isError, text)
		assert.Len(t, stub.starts, 3)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0)
		session := connect(t, newQANService(t, fake, true))
		ref := map[string]any{"queryid": "QID-AAA", "period_from": "2026-09-14T11:00:00Z", "period_to": "2026-09-14T12:00:00Z"}
		const needsForm = "pass query_ref or queryid to fetch a query's tables, or service_id, database and tables to name them"
		const noTablesWithQuery = "database and tables name the tables themselves: pass them with service_id, without query_ref or queryid"
		for _, tc := range []struct {
			args   map[string]any
			prefix string
		}{
			{map[string]any{}, needsForm},
			{map[string]any{"service_id": "svc-1", "database": "shop"}, needsForm},
			{map[string]any{"service_id": "svc-1", "tables": []string{"a"}}, needsForm},
			{map[string]any{"database": "shop", "tables": []string{"a"}}, needsForm},
			{map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{}}, needsForm},
			{map[string]any{"service_id": "svc-1", "database": "shop", "tables": strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",")}, "tables takes 1 to 10 names; got 11"},
			{map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a", " "}}, "tables must not contain an empty name"},
			{map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a"}, "info_types": "ddl"}, "info_types must be"},
			{map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a"}, "placeholders": []string{"1"}}, "placeholders go with query_ref or queryid"},
			{map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a"}, "period_from": "now-2h"}, "period_from and period_to go with queryid"},
			{map[string]any{"queryid": "QID-AAA", "tables": []string{"a"}}, noTablesWithQuery},
			{map[string]any{"queryid": "QID-AAA", "tables": []string{}}, noTablesWithQuery},
			{map[string]any{"queryid": "QID-AAA", "database": "shop"}, noTablesWithQuery},
			{map[string]any{"query_ref": ref, "database": "shop", "tables": []string{"a"}}, noTablesWithQuery},
			{map[string]any{"query_ref": ref, "queryid": "QID-AAA"}, "pass query_ref alone"},
			{map[string]any{"queryid": "QID-AAA", "info_types": "ddl"}, "info_types must be"},
		} {
			res := callTool(t, session, "pmm_get_schema", tc.args)
			text := textOf(t, res)
			assert.True(t, res.IsError, "%v", tc.args)
			assert.Nil(t, res.StructuredContent, "%v", tc.args)
			assert.Equal(t, map[string]any{"code": "invalid_input", "retryable": false}, res.Meta["com.percona/error"], "the handler's envelope, not an SDK refusal: %v", tc.args)
			assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"+tc.prefix), "%v: %s", tc.args, text)
		}
		assert.Empty(t, stub.starts)
		assert.Empty(t, fake.requestsTo(reportPath))
		assert.Zero(t, listQueries(fake))
	})
}

func TestConfigTool(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{vmQuery + " ?mysql_global_variables_": {file: "vm_variables.json"}}))
	session := connect(t, newQANService(t, fake, true))

	res := callTool(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "at": "now-1h"})
	text, isError := textOf(t, res), res.IsError
	assert.False(t, isError)
	assert.Equal(t, strings.Join([]string{
		"-- mysql 8.0.46-37 - 4 variables (numeric knobs from PMM metrics)",
		"-- At: 2026-09-14T11:00:00Z",
		"innodb_buffer_pool_size\t134217728",
		"innodb_io_capacity\t200",
		"long_query_time\t0.5",
		"max_connections\t151",
		"",
		"[View this service in PMM QAN ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from=" + ms(testNow.Add(-2*time.Hour)) + "&to=" + ms(testNow.Add(-time.Hour)) + "&var-service_name=shop-mysql)",
	}, "\n"), text)
	fake.assertForwardedAuth(t)
	assert.Equal(t, configResult{
		ServiceName: "shop-mysql",
		Engine:      engineMySQL,
		Version:     "8.0.46-37",
		At:          testNow.Add(-time.Hour),
		Variables: []variable{
			{Name: "innodb_buffer_pool_size", Value: "134217728"},
			{Name: "innodb_io_capacity", Value: "200"},
			{Name: "long_query_time", Value: "0.5"},
			{Name: "max_connections", Value: "151"},
		},
		Page:   page{Total: 4, Limit: 200},
		PMMURL: "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from=" + ms(testNow.Add(-2*time.Hour)) + "&to=" + ms(testNow.Add(-time.Hour)) + "&var-service_name=shop-mysql",
	}, structured[configResult](t, res))
	assert.Contains(t, string(mustJSON(t, res.StructuredContent)), `"at":"2026-09-14T11:00:00Z"`)

	queries := queriesWith(fake, "mysql_global_variables_")
	require.Len(t, queries, 1)
	assert.Equal(t, `query=%7B__name__%3D~%22mysql_global_variables_.%2B%22%2C+service_name%3D%22shop-mysql%22%7D&time=`+strconv.FormatInt(testNow.Add(-time.Hour).Unix(), 10)+`&timeout=25s`, queries[0].query)
	list := queriesWith(fake, "tlast_over_time(up[3600s])")
	require.Len(t, list, 1)
	assert.Equal(t, strconv.FormatInt(testNow.Add(-time.Hour).Unix(), 10), queryOf(t, list[0]).Get("time"))

	text, isError = callText(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "filter": "innodb"})
	assert.False(t, isError)
	assert.NotContains(t, text, "max_connections")
	assert.Contains(t, text, "innodb_io_capacity\t200")

	text, isError = callText(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "engine": "mongodb"})
	assert.True(t, isError)
	assert.True(t, strings.HasPrefix(text, "error: invalid_input\nengine"), text)

	// No variable matches: the text keeps its message and gains the time and the link.
	res = callTool(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "filter": "nothing_matches"})
	got := structured[configResult](t, res)
	assert.Empty(t, got.Variables)
	assert.Equal(t, page{Limit: 200}, got.Page)
	assert.Equal(t, strings.Join([]string{
		"No configuration variables found for that service (check the service and that its metrics are being collected).",
		"At: 2026-09-14T12:00:00Z",
		"",
		"[View this service in PMM QAN ↗](" + got.PMMURL + ")",
	}, "\n"), textOf(t, res))
	assert.Equal(t, "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from="+ms(testNow.Add(-time.Hour))+"&to="+ms(testNow)+"&var-service_name=shop-mysql", got.PMMURL)

	t.Run("InvalidPagingBeforeAnyCall", func(t *testing.T) {
		t.Parallel()

		// No routes: a backing call would answer not_found instead.
		fake := newFakePMM(t, nil)
		session := connect(t, newQANService(t, fake, true))
		for want, args := range map[string]map[string]any{
			"limit must be between 1 and 1000; got 1001": {"service_id": "svc-unknown", "limit": 1001},
			"offset must not be negative; got -1":        {"service_name": "shop-mysql", "offset": -1},
		} {
			text, isError := callText(t, session, "pmm_get_config", args)
			assert.True(t, isError, "%v", args)
			assert.Equal(t, "error: invalid_input\n"+want, text)
		}
		assert.Zero(t, fake.requestCount())
	})

	t.Run("Paging", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, inventoryRoutes())
		prev := fake.server.Config.Handler
		fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/api/v1/query") && strings.Contains(req.URL.Query().Get("query"), "mysql_global_variables_") {
				writeJSON(rw, vmVariables(250))
				return
			}
			prev.ServeHTTP(rw, req)
		})
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1"})
		got := structured[configResult](t, res)
		assert.Equal(t, page{Total: 250, Limit: 200, Truncated: true}, got.Page)
		require.Len(t, got.Variables, 200)
		assert.Equal(t, variable{Name: "var_000", Value: "0"}, got.Variables[0])
		assert.Equal(t, variable{Name: "var_199", Value: "199"}, got.Variables[199])
		assert.Equal(t, "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from="+ms(testNow.Add(-time.Hour))+"&to="+ms(testNow)+"&var-service_name=shop-mysql", got.PMMURL)
		text := textOf(t, res)
		assert.True(t, strings.HasPrefix(text, "-- mysql 8.0.46-37 - 250 variables (numeric knobs from PMM metrics)\n-- At: 2026-09-14T12:00:00Z\nvar_000\t0\n"), text)
		assert.True(t, strings.HasSuffix(text, "\nvar_199\t199\n\n[View this service in PMM QAN ↗]("+got.PMMURL+")\nshowing 1–200 of 250; pass offset=200 for more"), text)
		assert.Equal(t, 200, strings.Count(text, "\t"))

		res = callTool(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1", "offset": 200})
		got = structured[configResult](t, res)
		assert.Equal(t, page{Total: 250, Offset: 200, Limit: 200}, got.Page)
		require.Len(t, got.Variables, 50)
		assert.Equal(t, "var_200", got.Variables[0].Name)
		assert.NotContains(t, textOf(t, res), "showing ")

		res = callTool(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1", "offset": 300})
		got = structured[configResult](t, res)
		assert.Equal(t, page{Total: 250, Offset: 300, Limit: 200}, got.Page)
		assert.Empty(t, got.Variables)
		assert.Equal(t, strings.Join([]string{
			"No rows at offset 300; the variable list has 250.",
			"At: 2026-09-14T12:00:00Z",
			"",
			"[View this service in PMM QAN ↗](" + got.PMMURL + ")",
		}, "\n"), textOf(t, res))

		text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_id": "svc-1", "limit": 1001})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nlimit must be between 1 and 1000; got 1001"), text)
	})
}

// vmVariables is an instant-query answer with n MySQL variables, var_000 = 0 and up.
func vmVariables(n int) any {
	result := make([]any, 0, n)
	for i := range n {
		result = append(result, map[string]any{
			"metric": map[string]string{"__name__": fmt.Sprintf("mysql_global_variables_var_%03d", i), "service_name": "shop-mysql"},
			"value":  []any{1782379897, strconv.Itoa(i)},
		})
	}
	return map[string]any{"status": "success", "data": map[string]any{"resultType": "vector", "result": result}}
}

func TestMapActionError(t *testing.T) {
	t.Parallel()

	for msg, code := range map[string]errorCode{
		"Error 1142 (42000): SELECT command denied to user":             codeInsufficientPrivileges,
		"dial tcp 10.0.0.5:3306: connect: connection refused":           codeAgentUnreachable,
		"Error 1064 (42000): You have an error in your SQL syntax":      codeInvalidInput,
		"Error 1146 (42S02): Table 'shop.nope' doesn't exist":           codeNotFound,
		"Error 1698 (28000): Access denied for user 'root'@'localhost'": codeInsufficientPrivileges,
		"Error 1040 (08004): Too many connections":                      codeAgentUnreachable,
		"Error 1305 (42000): FUNCTION shop.f does not exist":            codeNotFound,
		"table not found: sql: no rows in result set":                   codeNotFound,
		"query EXPLAIN functionality is supported only for DML queries": codePMMUnavailable,
		// pmm-agent ends every action at pmm-managed's fixed limit and reports this.
		"context deadline exceeded": codeTimeout,
	} {
		assert.Equal(t, code, mapActionError(msg, false).code, msg)
	}

	multi := mapActionError("cannot explain this type of query\noperation insert is not supported for explain\n(BadValue) ns=shop.c Tree: age $eq 42", true)
	assert.Equal(t, codePMMUnavailable, multi.code)
	assert.NotContains(t, multi.Error(), "42")
	assert.NotContains(t, multi.Error(), "pmm_top_queries")

	assert.Equal(t, codeAgentUnreachable, mapError(mapActionStartError(&statusError{status: 400, message: "Cannot find right agent"})).code)
	assert.Equal(t, codeAgentUnreachable, mapError(mapActionStartError(&statusError{status: 404, message: "No pmm-agent running node x"})).code)
	assert.Equal(t, codeInvalidInput, mapError(mapActionStartError(&statusError{status: 400, message: "invalid field ServiceId"})).code)
}

func TestDecodeExplainOutput(t *testing.T) {
	t.Parallel()

	envelope := `{"explain_result":"cGxhbg==","explained_query":"SELECT 1","is_dml":false}`
	assert.Equal(t, "-- SELECT 1\nplan", first(decodeExplainOutput(envelope, true)))
	assert.Equal(t, "plan", first(decodeExplainOutput(envelope, false)))
	assert.Equal(t, "-- UPDATE t\n-- (DML statement rewritten to an equivalent SELECT by pmm-agent)\nplan",
		first(decodeExplainOutput(`{"explain_result":"cGxhbg==","explained_query":"UPDATE t","is_dml":true}`, true)))
	assert.Equal(t, "not json", first(decodeExplainOutput("not json", true)))
	assert.Equal(t, `{"other":1}`, first(decodeExplainOutput(`{"other":1}`, true)))
}

// actionStub fakes the actions API with one action per start, named
// <kind>-<table> (kind ddl, index or explain). GetAction answers done at once,
// except for the action ids in slow, which never finish. The first barrier
// starts wait for each other, for up to 2 s, so a caller that starts actions
// one at a time fails.
type actionStub struct {
	barrier int
	slow    []string
	// explain is the output of an explain action.
	explain string
	// explainErr is the error of an explain action, instead of its output.
	explainErr string
	// outputs replaces the output of the action ids it names.
	outputs map[string]string

	mu          sync.Mutex
	release     chan struct{}
	starts      []actions_service.StartServiceActionBody
	polls       int
	inFlight    int
	maxInFlight int
	// cancels and cancelAuth record each cancelAction call's action id and Authorization.
	cancels    []string
	cancelAuth []string
}

// cancelled returns the action ids cancelAction was called for, in call order.
func (a *actionStub) cancelled() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.cancels)
}

func newActionStub(t *testing.T, barrier int, slow ...string) (*fakePMM, *actionStub) {
	t.Helper()

	stub := &actionStub{barrier: barrier, slow: slow, release: make(chan struct{})}
	if barrier == 0 {
		close(stub.release)
	}
	fake := newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{"POST " + reportPath: {file: reportOne}}))
	prev := fake.server.Config.Handler
	fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.Path == "/v1/actions:startServiceAction":
			body, _ := io.ReadAll(req.Body)
			stub.start(rw, body)
		case req.URL.Path == "/v1/actions:cancelAction":
			body, _ := io.ReadAll(req.Body)
			stub.cancel(rw, body, req.Header.Get("Authorization"))
		case strings.HasPrefix(req.URL.Path, "/v1/actions/"):
			stub.get(rw, strings.TrimPrefix(req.URL.Path, "/v1/actions/"))
		default:
			prev.ServeHTTP(rw, req)
		}
	})
	return fake, stub
}

func (a *actionStub) start(rw http.ResponseWriter, body []byte) {
	var b actions_service.StartServiceActionBody
	err := json.Unmarshal(body, &b)
	if err != nil {
		http.Error(rw, `{"message":"cannot decode the start body"}`, http.StatusBadRequest)
		return
	}
	id := "explain-"
	switch {
	case b.MysqlShowCreateTable != nil:
		id = "ddl-" + b.MysqlShowCreateTable.TableName
	case b.MysqlShowIndex != nil:
		id = "index-" + b.MysqlShowIndex.TableName
	case b.PostgresShowCreateTable != nil:
		id = "ddl-" + b.PostgresShowCreateTable.TableName
	case b.PostgresShowIndex != nil:
		id = "index-" + b.PostgresShowIndex.TableName
	}

	a.mu.Lock()
	a.starts = append(a.starts, b)
	a.inFlight++
	a.maxInFlight = max(a.maxInFlight, a.inFlight)
	if len(a.starts) == a.barrier {
		close(a.release)
	}
	a.mu.Unlock()

	select {
	case <-a.release:
	case <-time.After(2 * time.Second):
		http.Error(rw, `{"message":"barrier not reached: the actions did not start concurrently"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(rw, map[string]map[string]string{"mysql_show_create_table": {"action_id": id}})
}

func (a *actionStub) cancel(rw http.ResponseWriter, body []byte, authorization string) {
	var b actions_service.CancelActionBody
	err := json.Unmarshal(body, &b)
	if err != nil {
		http.Error(rw, `{"message":"cannot decode the cancel body"}`, http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	a.cancels = append(a.cancels, b.ActionID)
	a.cancelAuth = append(a.cancelAuth, authorization)
	a.mu.Unlock()
	writeJSON(rw, struct{}{})
}

func (a *actionStub) get(rw http.ResponseWriter, id string) {
	done := !slices.Contains(a.slow, id)
	a.mu.Lock()
	a.polls++
	if done {
		a.inFlight--
	}
	a.mu.Unlock()
	kind, table, _ := strings.Cut(id, "-")
	output := map[string]string{"ddl": fmt.Sprintf("CREATE TABLE `%s` (`id` int)", table), "index": "index of " + table, "explain": a.explain}[kind]
	if o, ok := a.outputs[id]; ok {
		output = o
	}
	var actionErr string
	if kind == "explain" {
		actionErr = a.explainErr
	}
	writeJSON(rw, struct {
		ActionID string `json:"action_id"`
		Done     bool   `json:"done"`
		Output   string `json:"output"`
		Error    string `json:"error"`
	}{id, done, output, actionErr})
}

// writeJSON answers with v as JSON.
func writeJSON(rw http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	_, _ = rw.Write(b)
}

// TestActionDeadlineIsPerCall pins that one PMM_MCP_ACTION_TIMEOUT bounds a
// whole call: the DDL finishes at once and SHOW INDEX never does.
func TestActionDeadlineIsPerCall(t *testing.T) {
	t.Parallel()

	fake, stub := newActionStub(t, 0, "index-customers")
	s := newQANService(t, fake, true)
	s.actionTimeout = func() time.Duration { return time.Second }
	session := connect(t, s)

	start := time.Now()
	res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}, "info_types": "all"})
	elapsed := time.Since(start)
	text, isError := textOf(t, res), res.IsError
	assert.False(t, isError, text)
	assert.Contains(t, text, "CREATE TABLE `customers`")
	assert.Contains(t, text, "error: timeout\naction index-customers did not complete within 1s")
	// A deadline per action would end the call at about 1.4 s: polls at 0, 0.3, 0.75 and 1.4 s.
	assert.Less(t, elapsed, 1250*time.Millisecond)
	assert.Equal(t, []string{"index-customers"}, stub.cancelled())
	assert.Equal(t, []string{testToken}, stub.cancelAuth)

	got := structured[schemaResult](t, res)
	assert.True(t, got.Partial)
	require.Len(t, got.Tables, 1)
	assert.Equal(t, "CREATE TABLE `customers` (`id` int)", got.Tables[0].Definition)
	assert.Nil(t, got.Tables[0].DefinitionError)
	assert.Empty(t, got.Tables[0].Indexes)
	require.NotNil(t, got.Tables[0].IndexesError)
	assert.Equal(t, codeTimeout, got.Tables[0].IndexesError.Code)
	assert.Equal(t, "action index-customers did not complete within 1s (PMM_MCP_ACTION_TIMEOUT)", got.Tables[0].IndexesError.Message)
}

// TestSchemaManyTables pins the multi-table pmm_get_schema: actions run in
// parallel, at most 4 at a time, under one deadline, and every table gets a
// block in input order even when some fail.
func TestSchemaManyTables(t *testing.T) {
	t.Parallel()

	t.Run("Concurrent", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 4)
		session := connect(t, newQANService(t, fake, true))
		res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"d", "b", "a", "c"}})
		text, isError := textOf(t, res), res.IsError
		require.False(t, isError, text)
		assert.Len(t, stub.starts, 4)
		assert.Empty(t, stub.cancelled())
		assert.Equal(t, []string{"### shop.d", "### shop.b", "### shop.a", "### shop.c"}, headers(text), "input order")
		got := structured[schemaResult](t, res)
		assert.False(t, got.Partial)
		names := make([]string, 0, len(got.Tables))
		for _, tbl := range got.Tables {
			names = append(names, tbl.Name)
		}
		assert.Equal(t, []string{"d", "b", "a", "c"}, names, "input order")
	})

	t.Run("AtMostFourInFlight", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 4)
		session := connect(t, newQANService(t, fake, true))
		tables := strings.Split("a,b,c,d,e,f,g,h,i,j,a", ",")
		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": tables, "info_types": "all"})
		require.False(t, isError, text)
		assert.Len(t, stub.starts, 20, "10 distinct tables, DDL and indexes")
		assert.Equal(t, 4, stub.maxInFlight)
		assert.Empty(t, stub.cancelled())
		assert.Equal(t, strings.Split("### shop.a,### shop.b,### shop.c,### shop.d,### shop.e,### shop.f,### shop.g,### shop.h,### shop.i,### shop.j", ","), headers(text))
	})

	t.Run("InfoTypes", func(t *testing.T) {
		t.Parallel()

		for infoTypes, want := range map[string]string{
			"":           "### shop.a\n```sql\nCREATE TABLE `a` (`id` int)\n```",
			"definition": "### shop.a\n```sql\nCREATE TABLE `a` (`id` int)\n```",
			"indexes":    "### shop.a\nindexes:\n```\nindex of a\n```",
		} {
			fake, stub := newActionStub(t, 0)
			session := connect(t, newQANService(t, fake, true))
			text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a"}, "info_types": infoTypes})
			require.False(t, isError, text)
			assert.Empty(t, stub.cancelled())
			assert.Equal(t, want, text, infoTypes)
		}
	})

	t.Run("SlowTableTimesOutAlone", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0, "ddl-slow")
		s := newQANService(t, fake, true)
		s.actionTimeout = func() time.Duration { return time.Second }
		session := connect(t, s)

		start := time.Now()
		res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"a", "slow", "b"}})
		elapsed := time.Since(start)
		text, isError := textOf(t, res), res.IsError
		require.False(t, isError, text)
		assert.Less(t, elapsed, 1250*time.Millisecond)
		assert.Contains(t, text, "### shop.slow\nerror: timeout\naction ddl-slow did not complete within 1s (PMM_MCP_ACTION_TIMEOUT)\n\n### shop.b\n```sql\nCREATE TABLE `b`")
		assert.Contains(t, text, "### shop.a\n```sql\nCREATE TABLE `a`")
		assert.Equal(t, []string{"ddl-slow"}, stub.cancelled())

		got := structured[schemaResult](t, res)
		assert.True(t, got.Partial)
		require.Len(t, got.Tables, 3)
		assert.Equal(t, tableResult{Database: "shop", Name: "a", Definition: "CREATE TABLE `a` (`id` int)"}, got.Tables[0])
		assert.Equal(t, tableResult{Database: "shop", Name: "b", Definition: "CREATE TABLE `b` (`id` int)"}, got.Tables[2])
		assert.Equal(t, "slow", got.Tables[1].Name)
		assert.Empty(t, got.Tables[1].Definition)
		require.NotNil(t, got.Tables[1].DefinitionError)
		assert.Equal(t, codeTimeout, got.Tables[1].DefinitionError.Code)
		assert.Equal(t, "action ddl-slow did not complete within 1s (PMM_MCP_ACTION_TIMEOUT)", got.Tables[1].DefinitionError.Message)
		assert.Nil(t, got.Tables[1].IndexesError)
	})

	t.Run("EveryActionFails", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0, "ddl-x", "ddl-y")
		s := newQANService(t, fake, true)
		s.actionTimeout = func() time.Duration { return 500 * time.Millisecond }
		session := connect(t, s)
		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"x", "y"}})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: timeout\naction ddl-x did not complete"), text)
		assert.ElementsMatch(t, []string{"ddl-x", "ddl-y"}, stub.cancelled())
	})
}

// headers returns the "### db.table" lines of a pmm_get_schema result.
func headers(text string) []string {
	var out []string
	for l := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(l, "### ") {
			out = append(out, l)
		}
	}
	return out
}

func TestLiveInspectionRateLimited(t *testing.T) {
	t.Parallel()

	key := callerKey(callerAuth{authorization: testToken})
	take := func(t *testing.T, s *Service, key string) func() {
		t.Helper()

		rel, hit := s.live.acquire(key)
		require.Equal(t, limitNone, hit)
		return rel
	}
	schemaArgs := map[string]any{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}}

	t.Run("PerCaller", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		s := newQANService(t, fake, true)
		session := connect(t, s)
		rel1 := take(t, s, key)
		rel2 := take(t, s, key)

		text, isError := callText(t, session, "pmm_get_schema", schemaArgs)
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited"), text)
		assert.Contains(t, text, "these credentials")
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))

		// The stored PostgreSQL plan is passive and not limited.
		fake.routes["POST "+reportPath] = fixture{file: "report_by_service_postgres.json"}
		text, isError = callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-PG"})
		assert.False(t, isError, text)

		rel1()
		text, isError = callText(t, session, "pmm_get_schema", schemaArgs)
		assert.False(t, isError, text)
		rel2()
	})

	t.Run("PerReplica", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		s := newQANService(t, fake, true)
		session := connect(t, s)
		rels := make([]func(), 0, maxLiveCallsPerReplica)
		for i := range maxLiveCallsPerReplica {
			rels = append(rels, take(t, s, "other-"+string(rune('0'+i))))
		}

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited"), text)
		assert.Contains(t, text, "this replica")
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))
		for _, rel := range rels {
			rel()
		}
	})

	t.Run("PerService", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		s := newQANService(t, fake, true)
		session := connect(t, s)
		held, hit := s.liveByService.acquire("svc-1")
		require.Equal(t, limitNone, hit)

		text, isError := callText(t, session, "pmm_get_schema", schemaArgs)
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: rate_limited"), text)
		assert.Contains(t, text, "a live-inspection call on this service is already in flight; retry when it finishes")
		assert.NotContains(t, text, "svc-1")
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))

		text, isError = callText(t, session, "pmm_get_explain", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.Contains(t, text, "already in flight")
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))

		text, isError = callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-2", "database": "shop", "tables": []string{"customers"}})
		assert.False(t, isError, text)

		held()
		text, isError = callText(t, session, "pmm_get_schema", schemaArgs)
		assert.False(t, isError, text)
		_, hit = s.liveByService.acquire("svc-1")
		assert.Equal(t, limitNone, hit)
	})

	t.Run("PerServiceTwoCallers", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0, "ddl-customers")
		s := newQANService(t, fake, true)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{Header: http.Header{"Authorization": {testToken}}}}
		done := make(chan error, 1)
		go func() {
			_, err := s.schema(ctx, req, schemaInput{ServiceID: "svc-1", Database: "shop", Tables: []string{"customers"}})
			done <- err
		}()
		require.Eventually(t, func() bool {
			stub.mu.Lock()
			defer stub.mu.Unlock()
			return stub.polls > 0
		}, 5*time.Second, 10*time.Millisecond)

		other := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{Header: http.Header{"Authorization": {"Bearer other_token"}}}}
		_, err := s.schema(t.Context(), other, schemaInput{ServiceID: "svc-1", Database: "shop", Tables: []string{"customers"}})
		var te *toolError
		require.ErrorAs(t, err, &te)
		assert.Equal(t, codeRateLimited, te.code)
		assert.NotContains(t, te.message, "svc-1")

		cancel()
		select {
		case err := <-done:
			require.Error(t, err)
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the cancelled call did not return")
		}
		rel, hit := s.liveByService.acquire("svc-1")
		assert.Equal(t, limitNone, hit)
		rel()
	})

	t.Run("FailedCallReleases", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_error_1064.json")
		s := newQANService(t, fake, true)
		session := connect(t, s)

		_, isError := callText(t, session, "pmm_get_schema", schemaArgs)
		require.True(t, isError)
		rel1 := take(t, s, key)
		rel2 := take(t, s, key)
		rel1()
		rel2()
		rel, hit := s.liveByService.acquire("svc-1")
		assert.Equal(t, limitNone, hit)
		rel()
	})

	t.Run("CancelledCallReleases", func(t *testing.T) {
		t.Parallel()

		fake, stub := newActionStub(t, 0, "ddl-customers")
		s := newQANService(t, fake, true)

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{Header: http.Header{"Authorization": {testToken}}}}
		done := make(chan error, 1)
		go func() {
			_, err := s.schema(ctx, req, schemaInput{ServiceID: "svc-1", Database: "shop", Tables: []string{"customers"}})
			done <- err
		}()

		require.Eventually(t, func() bool {
			stub.mu.Lock()
			defer stub.mu.Unlock()
			return stub.polls > 0
		}, 5*time.Second, 10*time.Millisecond)
		// The running call holds one of the two slots.
		rel := take(t, s, key)
		_, hit := s.live.acquire(key)
		require.Equal(t, limitPerCaller, hit)
		rel()
		cancel()

		select {
		case err := <-done:
			require.Error(t, err)
			assert.Equal(t, []string{"ddl-customers"}, stub.cancelled())
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the cancelled call did not return")
		}
		rel1 := take(t, s, key)
		rel2 := take(t, s, key)
		rel1()
		rel2()
	})
}

func TestEngineOfOutsideList(t *testing.T) {
	t.Parallel()

	fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
	s := newQANService(t, fake, true)
	session := connect(t, s)

	res := callTool(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-secret", "database": "shop", "tables": []string{"orders"}})
	require.True(t, res.IsError)
	text := textOf(t, res)
	assert.Equal(t, "error: not_found\n"+errNotInList(s.lastHour(), "; use pmm_inventory to list service ids").message, text)
	for _, name := range []string{"svc-secret", "svc-1", "svc-2", "svc-mongo", "svc-pmm-pg", "shop-mysql", "shop-postgres", "shop-mongo", "pmm-server-postgresql"} {
		assert.NotContains(t, text, name)
	}
	assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))
}
