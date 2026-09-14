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
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
)

// sequenceFixtures answers a path with successive fixtures (polling).
type sequenceFixtures struct {
	files []string
	calls atomic.Int32
}

func actionRoutes(t *testing.T, start, done string) (*fakePMM, *sequenceFixtures) {
	t.Helper()

	routes := qanRoutes()
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

func TestExplainTool(t *testing.T) {
	t.Parallel()

	t.Run("MySQLLive", func(t *testing.T) {
		t.Parallel()

		fake, seq := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA", "database": "shop"})
		assert.False(t, isError)
		assert.True(t, strings.HasPrefix(text, "EXPLAIN (json, mysql shop-mysql):\n```\n-- SELECT * FROM customers WHERE email = 'user42@example.com'\n{\n  \"query_block\""), text)
		assert.Contains(t, text, `"access_type": "ALL"`)
		assert.Contains(t, text, "details_tab=explain&filter_by=QID-AAA")
		fake.assertForwardedAuth(t)

		// The stored-plan probe ran first, then the action was started and polled.
		assert.Len(t, fake.requestsTo("/v1/qan/query/QID-AAA/plan"), 1)
		starts := fake.requestsTo("/v1/actions:startServiceAction")
		require.Len(t, starts, 1)
		body := unmarshalBody[actions_service.StartServiceActionBody](t, starts[0])
		require.NotNil(t, body.MysqlExplainJSON)
		assert.Equal(t, "svc-1", body.MysqlExplainJSON.ServiceID)
		assert.Equal(t, "QID-AAA", body.MysqlExplainJSON.QueryID)
		assert.Equal(t, "shop", body.MysqlExplainJSON.Database)
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
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-PG"})
		assert.False(t, isError)
		assert.True(t, strings.HasPrefix(text, "Stored plan (pg_stat_monitor, planid PLAN-1):\n```\nSeq Scan on customers"), text)
		assert.Empty(t, fake.requestsTo("/v1/actions:startServiceAction"))
	})

	t.Run("PostgreSQLWithoutPlan", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_explain.json", "action_done_explain.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-2", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: not_found\nno stored plan for queryid 'QID-AAA'"), text)
		assert.Contains(t, text, "pgsm_enable_query_plan=on")
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

		routes := qanRoutes()
		routes["GET /v1/qan/query/QID-AAA/plan"] = fixture{file: "plan_empty.json"}
		routes["POST /v1/actions:startServiceAction"] = fixture{file: "error_no_agent.json", status: http.StatusBadRequest}
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
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nqueryid or query is required"), text)

		text, isError = callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "QID-AAA", "format": "tree"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nformat"), text)

		text, isError = callText(t, session, "pmm_get_explain", map[string]any{"service_id": "svc-404", "queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: not_found\nservice 'svc-404' not found"), text)
	})
}

func TestSchemaTool(t *testing.T) {
	t.Parallel()

	t.Run("MySQL", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop", "table": "customers"})
		assert.False(t, isError)
		assert.Equal(t, "```sql\nCREATE TABLE `customers` (\n  `id` bigint NOT NULL AUTO_INCREMENT,\n  `email` varchar(255) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB\n```", text)
		fake.assertForwardedAuth(t)
		body := unmarshalBody[actions_service.StartServiceActionBody](t, fake.requestsTo("/v1/actions:startServiceAction")[0])
		require.NotNil(t, body.MysqlShowCreateTable)
		assert.Equal(t, "customers", body.MysqlShowCreateTable.TableName)
		assert.Equal(t, "shop", body.MysqlShowCreateTable.Database)
	})

	t.Run("PostgreSQLWithIndexes", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-2", "database": "shop", "table_name": "customers", "include_indexes": true})
		assert.False(t, isError)
		assert.Contains(t, text, "\n\nindexes:\n```\n")
		starts := fake.requestsTo("/v1/actions:startServiceAction")
		require.Len(t, starts, 2)
		ddl := unmarshalBody[actions_service.StartServiceActionBody](t, starts[0])
		require.NotNil(t, ddl.PostgresShowCreateTable)
		idx := unmarshalBody[actions_service.StartServiceActionBody](t, starts[1])
		require.NotNil(t, idx.PostgresShowIndex)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Parallel()

		fake, _ := actionRoutes(t, "action_start_showcreate.json", "action_done_ddl.json")
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_get_schema", map[string]any{"service_id": "svc-1", "database": "shop"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nservice_id, database and table are required"), text)
	})
}

func TestConfigTool(t *testing.T) {
	t.Parallel()

	routes := qanRoutes()
	fake := newFakePMM(t, routes)
	// The query path is shared by the variables and the version lookups:
	// answer by PromQL content.
	prev := fake.server.Config.Handler
	fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/api/v1/query") {
			file := "vm_mysql_version.json"
			if strings.Contains(req.URL.Query().Get("query"), "mysql_global_variables_") {
				file = "vm_variables.json"
			}
			fake.mu.Lock()
			fake.requests = append(fake.requests, recordedRequest{method: req.Method, path: req.URL.Path, query: req.URL.RawQuery, header: req.Header.Clone()})
			fake.mu.Unlock()
			rw.Header().Set("Content-Type", "application/json")
			_, _ = rw.Write(readFixture(t, file))
			return
		}
		prev.ServeHTTP(rw, req)
	})
	session := connect(t, newQANService(t, fake, true))

	text, isError := callText(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "at": "now-1h"})
	assert.False(t, isError)
	assert.Equal(t, strings.Join([]string{
		"-- mysql 8.0.46-37 - 4 variables (numeric knobs from PMM metrics)",
		"innodb_buffer_pool_size\t134217728",
		"innodb_io_capacity\t200",
		"long_query_time\t0.5",
		"max_connections\t151",
		"",
		"[View this service in PMM ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from=" + ms(testNow.Add(-2*time.Hour)) + "&to=" + ms(testNow.Add(-time.Hour)) + "&var-service_name=shop-mysql)",
	}, "\n"), text)
	fake.assertForwardedAuth(t)

	queries := fake.requestsTo("/graph/api/datasources/proxy/uid/metrics-uid/api/v1/query")
	require.Len(t, queries, 2)
	assert.Equal(t, `query=%7B__name__%3D~%22mysql_global_variables_.%2B%22%2C+service_name%3D%22shop-mysql%22%7D&time=`+strconv.FormatInt(testNow.Add(-time.Hour).Unix(), 10), queries[0].query)

	text, isError = callText(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "filter": "innodb"})
	assert.False(t, isError)
	assert.NotContains(t, text, "max_connections")
	assert.Contains(t, text, "innodb_io_capacity\t200")

	text, isError = callText(t, session, "pmm_get_config", map[string]any{"service_name": "shop-mysql", "engine": "mongodb"})
	assert.True(t, isError)
	assert.True(t, strings.HasPrefix(text, "error: invalid_input\nengine"), text)
}

func TestMapActionError(t *testing.T) {
	t.Parallel()

	for msg, code := range map[string]errorCode{
		"Error 1142 (42000): SELECT command denied to user":             codeInsufficientPrivileges,
		"dial tcp 10.0.0.5:3306: connect: connection refused":           codeAgentUnreachable,
		"Error 1064 (42000): You have an error in your SQL syntax":      codeInvalidInput,
		"Error 1146 (42S02): Table 'shop.nope' doesn't exist":           codeNotFound,
		"table not found: sql: no rows in result set":                   codeNotFound,
		"query EXPLAIN functionality is supported only for DML queries": codePMMUnavailable,
	} {
		assert.Equal(t, code, mapActionError(msg).code, msg)
	}

	assert.Equal(t, codeAgentUnreachable, mapError(mapActionStartError(&statusError{status: 400, message: "Cannot find right agent"})).code)
	assert.Equal(t, codeAgentUnreachable, mapError(mapActionStartError(&statusError{status: 404, message: "No pmm-agent running node x"})).code)
	assert.Equal(t, codeInvalidInput, mapError(mapActionStartError(&statusError{status: 400, message: "invalid field ServiceId"})).code)
}

func TestDecodeExplainOutput(t *testing.T) {
	t.Parallel()

	envelope := `{"explain_result":"cGxhbg==","explained_query":"SELECT 1","is_dml":false}`
	assert.Equal(t, "-- SELECT 1\nplan", decodeExplainOutput(envelope, true))
	assert.Equal(t, "plan", decodeExplainOutput(envelope, false))
	assert.Equal(t, "-- UPDATE t\n-- (DML statement rewritten to an equivalent SELECT by pmm-agent)\nplan",
		decodeExplainOutput(`{"explain_result":"cGxhbg==","explained_query":"UPDATE t","is_dml":true}`, true))
	assert.Equal(t, "not json", decodeExplainOutput("not json", true))
	assert.Equal(t, `{"other":1}`, decodeExplainOutput(`{"other":1}`, true))
}
