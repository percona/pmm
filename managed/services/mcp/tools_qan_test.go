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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

var testNow = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

// ms renders a time as epoch milliseconds, as Grafana links expect.
func ms(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

func qanRoutes() map[string]fixture {
	routes := inventoryRoutes()
	routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report.json"}
	routes["POST /v1/qan:getMetrics"] = fixture{file: "metrics.json"}
	routes["POST /v1/qan/query:getExample"] = fixture{file: "example.json"}
	return routes
}

func newQANService(t *testing.T, fake *fakePMM, rawSQL bool) *Service {
	t.Helper()

	s := newTestService(t, fake)
	s.rawSQL = func() bool { return rawSQL }
	s.now = func() time.Time { return testNow }
	s.publicAddress = func(_ context.Context) string { return "pmm.example.com" }
	return s
}

// textOf returns a tool result's text content.
func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()

	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "unexpected content %T", res.Content[0])
	return text.Text
}

// structured decodes a successful tool result's structured content into T.
func structured[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()

	require.False(t, res.IsError, textOf(t, res))
	var out T
	require.NoError(t, json.Unmarshal(mustJSON(t, res.StructuredContent), &out))
	return out
}

// TestRawFingerprintIsMasked pins the case where pmm-agent stores a raw
// statement as the fingerprint: a PostgreSQL query pg_stat_monitor truncated
// or could not parse. With raw SQL off its literals must not reach the model.
func TestRawFingerprintIsMasked(t *testing.T) {
	t.Parallel()

	const raw = "INSERT INTO users (email, pin) VALUES ('user42@example.com', 4417"

	routes := qanRoutes()
	routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report_raw_fingerprint.json"}
	for rawSQL, want := range map[bool]string{false: withheldFingerprint, true: raw} {
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, rawSQL))
		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
		require.False(t, isError, text)
		assert.Contains(t, text, "   "+want+"\n", "raw SQL %t", rawSQL)
		assert.Contains(t, text, "   -1234567890123\n", "a missing fingerprint falls back to the queryid, unmasked")
	}

	// The structured rows carry the fingerprint the text shows, masked the same way.
	for rawSQL, want := range map[bool]string{false: withheldFingerprint, true: raw} {
		session := connect(t, newQANService(t, newFakePMM(t, routes), rawSQL))
		res := callTool(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
		got := structured[topQueriesResult](t, res)
		require.Len(t, got.Rows, 2)
		assert.Equal(t, want, got.Rows[0].Fingerprint, "raw SQL %t", rawSQL)
		assert.Contains(t, textOf(t, res), "\n   "+got.Rows[0].Fingerprint+"\n", "raw SQL %t", rawSQL)
		assert.Empty(t, got.Rows[1].Fingerprint, "no fingerprint: only the text falls back to the queryid")
		if !rawSQL {
			assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), "user42@example.com")
			assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), "4417")
		}
	}

	result := (&queryDetail{queryID: "Q", fingerprint: raw}).result(false, "", testNow.Add(-time.Hour), testNow)
	detail := result.render()
	assert.NotContains(t, detail, "user42@example.com")
	assert.NotContains(t, detail, "4417")
	assert.Equal(t, withheldFingerprint, result.Fingerprint)
	assert.Contains(t, detail, "fingerprint:\n"+result.Fingerprint+"\n")
	assert.NotContains(t, string(mustJSON(t, result)), "4417")

	// explain_fingerprint is withheld when quoted, and keeps its numbered placeholders.
	for fp, want := range map[string]string{
		"SELECT `id` FROM `t` WHERE `email` = :1":           "SELECT `id` FROM `t` WHERE `email` = :1",
		"SELECT `id` FROM `t` WHERE `email` = 'a@b.com'":    "",
		`SELECT "id" FROM "t" WHERE "email" = "leak@b.com"`: "",
	} {
		d := &queryDetail{queryID: "Q", engine: engineMySQL, example: &qan_service.GetQueryExampleOKBodyQueryExamplesItems0{ExplainFingerprint: fp}}
		r := d.result(false, "", testNow.Add(-time.Hour), testNow)
		got := r.render()
		assert.Equal(t, want, r.Example, "the structured example is the one the text shows, or none: %s", fp)
		if want == "" {
			assert.NotContains(t, got, "example (normalized", fp)
			continue
		}
		assert.Contains(t, got, want, fp)
	}

	// A raw statement is withheld however its quoting reads: a nested
	// comment, or an apostrophe in a comment, cannot make a literal pass.
	for _, fp := range []string{
		"SELECT /* a /* b */ 'x */ 1 FROM t WHERE y = 'secret'",
		"SELECT * FROM t WHERE note = 'x' -- it's\n AND ssn = 'SECRET'",
		"SELECT `a FROM t WHERE ssn = 'SECRET'",
	} {
		assert.Equal(t, withheldFingerprint, fingerprintText(fp, "", false), fp)
	}
	// Normalizers keep GROUP BY ordinals and type lengths: those fingerprints
	// are shown, with the bare numbers masked.
	assert.Equal(t, "SELECT status, count(*) FROM orders GROUP BY ? ORDER BY ? DESC LIMIT $1",
		fingerprintText("SELECT status, count(*) FROM orders GROUP BY 1 ORDER BY 2 DESC LIMIT $1", "", false))
	assert.Equal(t, "SELECT CAST($1 AS varchar(?)) FROM t", fingerprintText("SELECT CAST($1 AS varchar(255)) FROM t", "", false))

	// MySQL quoting for a MySQL service: a backticked name is a name.
	assert.Equal(t, "SELECT `id` FROM `2fa_tokens` WHERE `o'neil` = ?",
		fingerprintText("SELECT `id` FROM `2fa_tokens` WHERE `o'neil` = ?", engineMySQL, false))

	// A normalized fingerprint is shown, with only its comment text masked.
	ormQuoted := `/* app='shop',route='/users/42' */ SELECT "users"."id" FROM "users" WHERE "users"."email" = $1`
	assert.Equal(t, `/*?*/ SELECT "users"."id" FROM "users" WHERE "users"."email" = $1`, fingerprintText(ormQuoted, enginePostgreSQL, false))

	// With the engine unknown, a double quote may open a MySQL string, so the
	// fingerprint is withheld rather than read with PostgreSQL quoting.
	doubleQuoted := `SELECT id FROM users WHERE email = "a@b.com"`
	assert.Equal(t, withheldFingerprint, fingerprintText(doubleQuoted, "", false))
	assert.Equal(t, withheldFingerprint, fingerprintText(doubleQuoted, engineMySQL, false))
	assert.Equal(t, withheldFingerprint, fingerprintText(ormQuoted, "", false))
	assert.Equal(t, doubleQuoted, fingerprintText(doubleQuoted, enginePostgreSQL, false), "a PostgreSQL identifier")
	assert.Equal(t, "SELECT `id` FROM `orders` WHERE `id` = ?", fingerprintText("SELECT `id` FROM `orders` WHERE `id` = ?", "", false))

	// pmm-agent masks only $match stages of an aggregation; the rest keep
	// their values, in double quotes that SQL quoting would take for names.
	mongo := `db.orders.aggregate([{"$match":{"email":"?"}}, {"$lookup":{"from":"users","pipeline":[{"$match":{"email":"ceo@corp.com"}}]}}, ` +
		`{"$project":{"t":{"$cond":[true,"gold",250]}}}])`
	got := fingerprintText(mongo, "", false)
	for _, literal := range []string{"ceo@corp.com", "gold", "250", "users"} {
		assert.NotContains(t, got, literal)
	}
	assert.Contains(t, got, `db.orders.aggregate([{"$match":{"email":"?"}}, {"$lookup":{"from":?,"pipeline":[{"$match":{"email":?}}]}}`,
		"keys, stages and the agent's own ? survive")
}

// TestTopQueriesQuotingFollowsTheService pins that a report filtered to one
// service reads its fingerprints with that engine's quoting.
func TestTopQueriesQuotingFollowsTheService(t *testing.T) {
	t.Parallel()

	routes := qanRoutes()
	routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report_quoted_identifier.json"}
	session := connect(t, newQANService(t, newFakePMM(t, routes), false))

	text, isError := callText(t, session, "pmm_top_queries", map[string]any{"service_name": "shop-mysql", "period_from": "now-1h"})
	require.False(t, isError, text)
	assert.Contains(t, text, "SELECT `id` FROM `o'neil` WHERE `id` = ?")

	// Unfiltered, the engine is QAN's only SQL engine in the window; with
	// several, or when QAN cannot tell, one more report decides per row. The
	// fake answers it with the same row, so all of that query's calls are MySQL.
	for file, reports := range map[string]int{"filters_mysql.json": 1, "filters_mixed.json": 2, "": 2} {
		routes := qanRoutes()
		routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report_quoted_identifier.json"}
		if file != "" {
			routes["POST /v1/qan/metrics:getFilters"] = fixture{file: file}
		}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, false))
		text, isError = callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
		require.False(t, isError, text)
		assert.Contains(t, text, "SELECT `id` FROM `o'neil` WHERE `id` = ?", file)
		assert.Len(t, fake.requestsTo("/v1/qan/metrics:getFilters"), 1, file)
		assert.Len(t, fake.requestsTo("/v1/qan/metrics:getReport"), reports, file)
	}

	// Fingerprints that read the same under both quotings need no engine.
	fake := newFakePMM(t, qanRoutes())
	session = connect(t, newQANService(t, fake, false))
	text, isError = callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
	require.False(t, isError, text)
	assert.Contains(t, text, "FROM `customers` WHERE `email` = ?")
	assert.Empty(t, fake.requestsTo("/v1/qan/metrics:getFilters"))
}

func TestTopQueriesTool(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, qanRoutes())
	session := connect(t, newQANService(t, fake, true))

	text, isError := callText(t, session, "pmm_top_queries", map[string]any{"service_name": "shop-mysql", "period_from": "now-1h"})
	assert.False(t, isError)
	assert.Equal(t, strings.Join([]string{
		"Rows 1-2 of 2 (group_by=queryid, order_by=load):",
		"Window: 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z",
		"",
		"1. [QID-AAA] load=0.42 calls=1200 total=88.5s avg=0.073s",
		"   SELECT * FROM `customers` WHERE `email` = ?",
		"2. [QID-BBB] load=0.18 calls=197 total=40s avg=0.203s",
		"   SELECT * FROM `orders` ORDER BY `RAND` ( ) LIMIT ?",
		"",
		"[Open this workload in PMM QAN ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
			"from=" + ms(testNow.Add(-time.Hour)) + "&to=" + ms(testNow) + "&var-service_name=shop-mysql)",
	}, "\n"), text)
	fake.assertForwardedAuth(t)

	reqs := fake.requestsTo("/v1/qan/metrics:getReport")
	require.Len(t, reqs, 1)
	body := unmarshalBody[qan_service.GetReportBody](t, reqs[0])
	assert.Equal(t, "queryid", body.GroupBy)
	assert.Equal(t, "-load", body.OrderBy)
	assert.Equal(t, "load", body.MainMetric)
	assert.EqualValues(t, 10, body.Limit)
	assert.Equal(t, testNow.Add(-time.Hour), time.Time(body.PeriodStartFrom))
	assert.Equal(t, testNow, time.Time(body.PeriodStartTo))
	require.Len(t, body.Labels, 1)
	assert.Equal(t, "service_name", body.Labels[0].Key)
	assert.Equal(t, []string{"shop-mysql"}, body.Labels[0].Value)

	t.Run("OrderAndLimit", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		_, isError := callText(t, session, "pmm_top_queries", map[string]any{"service_id": "svc-1", "order_by": "count", "limit": 5})
		assert.False(t, isError)
		body := unmarshalBody[qan_service.GetReportBody](t, fake.requestsTo("/v1/qan/metrics:getReport")[0])
		assert.Equal(t, "-num_queries", body.OrderBy)
		assert.EqualValues(t, 5, body.Limit)
		assert.Equal(t, "service_id", body.Labels[0].Key)

		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"order_by": "rows"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\norder_by"), text)

		text, isError = callText(t, session, "pmm_top_queries", map[string]any{"limit": 500})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nlimit"), text)

		text, isError = callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h", "period_to": "now-2h"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\nperiod_to"), text)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan/metrics:getReport"] = fixture{file: "example_empty.json"}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_top_queries", map[string]any{})
		assert.False(t, isError)
		assert.Equal(t, strings.Join([]string{
			"No queries found for that service / time window.",
			"Window: 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z",
			"",
			"[Open this workload in PMM QAN ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
				"from=" + ms(testNow.Add(-time.Hour)) + "&to=" + ms(testNow) + ")",
		}, "\n"), text)
	})

	t.Run("Structured", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_top_queries", map[string]any{"service_name": "shop-mysql", "period_from": "now-1h"})
		got := structured[topQueriesResult](t, res)
		from := testNow.Add(-time.Hour)
		ref := func(queryID string) *queryRef {
			return &queryRef{QueryID: queryID, ServiceID: "svc-1", PeriodFrom: from, PeriodTo: testNow}
		}
		assert.Equal(t, topQueriesResult{
			Rows: []queryRow{
				{
					Rank: 1, Dimension: "QID-AAA", Fingerprint: "SELECT * FROM `customers` WHERE `email` = ?",
					Load: 0.42, Calls: 1200, QueryTimeTotal: 88.5, QueryTimeAvg: 0.073, QueryRef: ref("QID-AAA"),
				},
				{
					Rank: 2, Dimension: "QID-BBB", Fingerprint: "SELECT * FROM `orders` ORDER BY `RAND` ( ) LIMIT ?",
					Load: 0.18, Calls: 197, QueryTimeTotal: 40, QueryTimeAvg: 40.0 / 197, QueryRef: ref("QID-BBB"),
				},
			},
			Page:    page{Total: 2, Offset: 0, Limit: 10},
			GroupBy: "queryid",
			OrderBy: "load",
			Window:  window{From: from, To: testNow},
			PMMURL: "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
				"from=" + ms(from) + "&to=" + ms(testNow) + "&var-service_name=shop-mysql",
		}, got)

		text := textOf(t, res)
		for _, row := range got.Rows {
			assert.Contains(t, text, fmt.Sprintf("%d. [%s] load=%s calls=%s total=%s avg=%s\n   %s\n", row.Rank, row.Dimension,
				fmtNum3(row.Load), fmtNum(row.Calls), fmtSeconds(row.QueryTimeTotal), fmtSeconds(row.QueryTimeAvg), row.Fingerprint))
		}
		assert.Contains(t, text, "("+got.PMMURL+")")
		assert.Contains(t, text, "\nWindow: "+got.Window.From.Format(time.RFC3339)+" – "+got.Window.To.Format(time.RFC3339)+"\n")
		assert.Contains(t, string(mustJSON(t, res.StructuredContent)), `"window":{"period_from":"2026-09-14T11:00:00Z","period_to":"2026-09-14T12:00:00Z"}`)
		assert.Equal(t, 1, listQueries(fake), "the service_name is looked up once, for query_ref")
	})

	t.Run("QueryRef", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			args      map[string]any
			serviceID string
			lookups   int
		}{
			{map[string]any{}, "", 0},
			{map[string]any{"service_id": "svc-9"}, "svc-9", 0},
			{map[string]any{"labels": map[string]any{"service_id": []string{"svc-2"}}}, "svc-2", 0},
			{map[string]any{"labels": map[string]any{"service_id": []string{"svc-1", "svc-2"}}}, "", 0},
			{map[string]any{"service_name": "shop-mysql"}, "svc-1", 1},
			{map[string]any{"labels": map[string]any{"service_name": []string{"shop-postgres"}}}, "svc-2", 1},
			{map[string]any{"labels": map[string]any{"service_name": []string{"shop-mysql", "shop-postgres"}}}, "", 0},
			{map[string]any{"service_name": "not-in-the-list"}, "", 1},
		} {
			fake := newFakePMM(t, qanRoutes())
			session := connect(t, newQANService(t, fake, true))
			tc.args["period_from"], tc.args["period_to"] = "now-2h", "now-1h"

			got := structured[topQueriesResult](t, callTool(t, session, "pmm_top_queries", tc.args))
			require.Len(t, got.Rows, 2, "%v", tc.args)
			for _, row := range got.Rows {
				assert.Equal(t, &queryRef{
					QueryID: row.Dimension, ServiceID: tc.serviceID, PeriodFrom: testNow.Add(-2 * time.Hour), PeriodTo: testNow.Add(-time.Hour),
				}, row.QueryRef, "%v", tc.args)
			}
			assert.Equal(t, tc.lookups, listQueries(fake), "%v", tc.args)
		}

		// A failed lookup leaves service_id out; the call still succeeds.
		routes := qanRoutes()
		routes[serviceListKey] = fixture{file: "error_403.json", status: http.StatusForbidden}
		session := connect(t, newQANService(t, newFakePMM(t, routes), true))
		got := structured[topQueriesResult](t, callTool(t, session, "pmm_top_queries", map[string]any{"service_name": "shop-mysql"}))
		require.Len(t, got.Rows, 2)
		require.NotNil(t, got.Rows[0].QueryRef)
		assert.Equal(t, "QID-AAA", got.Rows[0].QueryRef.QueryID)
		assert.Empty(t, got.Rows[0].QueryRef.ServiceID)
	})

	t.Run("ColumnsAndPaging", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report_by_user.json"}
		session := connect(t, newQANService(t, newFakePMM(t, routes), false))

		res := callTool(t, session, "pmm_top_queries", map[string]any{
			"group_by": "username", "columns": []string{"rows_examined", "full_scan", "query_time"},
			"order_by": "-rows_examined", "offset": 10, "limit": 2,
		})
		got := structured[topQueriesResult](t, res)
		assert.Equal(t, []queryRow{
			{
				Rank: 11, Dimension: "app", Load: 0.4, Calls: 700, QueryTimeTotal: 120, QueryTimeAvg: 0.171,
				Columns: map[string]float64{"rows_examined": 100, "full_scan": 12},
			},
			{
				Rank: 12, Dimension: "", Load: 0.1, Calls: 200, QueryTimeTotal: 30, QueryTimeAvg: 0.15,
				Columns: map[string]float64{"rows_examined": 0, "full_scan": 0},
			},
		}, got.Rows, "avg columns carry the average, the others the sum; base columns are not repeated; no query_ref off queryid")
		assert.Equal(t, page{Total: 14, Offset: 10, Limit: 2, Truncated: true}, got.Page)

		text := textOf(t, res)
		assert.Contains(t, text, "11. [app] load=0.4 calls=700 total=120s avg=0.171s rows_examined_avg=100 rows_examined=70000 full_scan=12\n")
		assert.True(t, strings.HasPrefix(text, "Rows 11-12 of 14 (group_by=username, order_by=-rows_examined):\n"), text)
		assert.True(t, strings.HasSuffix(text, "\nshowing 11–12 of 14; pass offset=12 for more"), text)
	})
}

func TestQueryDetailTool(t *testing.T) {
	t.Parallel()

	t.Run("RawSQL", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-1"})
		assert.False(t, isError)
		assert.Equal(t, strings.Join([]string{
			"queryid: QID-AAA",
			"Window: 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z",
			"engine: mysql 8.0.46-37",
			"service: shop-mysql",
			"database: shop_db",
			"schema: shop",
			"tables: customers",
			"fingerprint:",
			"SELECT * FROM `customers` WHERE `email` = ?",
			"metrics: calls=1200, query_time_sum=88.5, query_time_avg=0.073, rows_examined_avg=20000, rows_sent_avg=1, no_index_used=1200, full_scan=1200",
			"example:",
			"```sql",
			"SELECT * FROM customers WHERE email = 'user42@example.com'",
			"```",
			"",
			"[View this query in PMM QAN ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
				"filter_by=QID-AAA&from=" + ms(testNow.Add(-time.Hour)) + "&query_selected=true&to=" + ms(testNow) + "&var-service_name=shop-mysql)",
		}, "\n"), text)
		fake.assertForwardedAuth(t)

		body := unmarshalBody[qan_service.GetMetricsBody](t, fake.requestsTo("/v1/qan:getMetrics")[0])
		assert.Equal(t, "QID-AAA", body.FilterBy)
		assert.Equal(t, "queryid", body.GroupBy)
		require.Len(t, body.Labels, 2)
		assert.Equal(t, "service_id", body.Labels[1].Key)
	})

	t.Run("Redacted", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, false))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		text, isError := textOf(t, res), res.IsError
		assert.False(t, isError)
		assert.NotContains(t, text, "user42@example.com")
		assert.Contains(t, text, "example (normalized, literals stripped; raw SQL disabled):\n```sql\n")
		assert.Contains(t, text, "FROM `customers` WHERE `email` = :1\n```")

		got := structured[queryDetailResult](t, res)
		assert.NotContains(t, string(mustJSON(t, res.StructuredContent)), "user42@example.com")
		assert.Equal(t, "SELECT * FROM `customers` WHERE `email` = :1", got.Example) //nolint:unqueryvet
		assert.Contains(t, text, "```sql\n"+got.Example+"\n```", "the example the text shows")
		assert.Contains(t, text, "fingerprint:\n"+got.Fingerprint+"\n", "the fingerprint the text shows")
	})

	t.Run("ExampleUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan/query:getExample"] = fixture{file: "example_empty.json"}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		assert.False(t, isError)
		assert.NotContains(t, text, "example:")
		assert.NotContains(t, text, "user42")
		assert.Contains(t, text, "fingerprint:\nSELECT")
	})

	t.Run("MetricsUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan:getMetrics"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		assert.False(t, isError)
		assert.Contains(t, text, "schema: shop")
		assert.Contains(t, text, "example:")
	})

	t.Run("BothUnavailable", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan:getMetrics"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		routes["POST /v1/qan/query:getExample"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: unauthorized\n"), text)
	})

	t.Run("NoData", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan:getMetrics"] = fixture{file: "metrics_empty.json"}
		routes["POST /v1/qan/query:getExample"] = fixture{file: "example_empty.json"}
		fake := newFakePMM(t, routes)
		session := connect(t, newQANService(t, fake, true))

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-ZZZ"})
		assert.True(t, isError)
		assert.True(t, strings.HasPrefix(text, "error: not_found\n"), text)
		assert.Contains(t, text, "no data for this query in 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z;")
		assert.NotContains(t, text, "QID-ZZZ")
	})

	t.Run("ServiceOutsideList", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-foreign"})
		assert.True(t, res.IsError)
		assert.Equal(t, "error: not_found\n"+errNotInList(window{From: testNow.Add(-time.Hour), To: testNow}, "").message, textOf(t, res))
		assert.Empty(t, fake.requestsTo("/v1/qan:getMetrics"))
		assert.Empty(t, fake.requestsTo("/v1/qan/query:getExample"))
	})

	t.Run("NoServiceIDCarriesTheList", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		require.False(t, res.IsError, textOf(t, res))

		want := []string{"svc-1", "svc-2", "svc-mongo", "svc-pmm-pg"}
		metrics := fake.requestsTo("/v1/qan:getMetrics")
		require.Len(t, metrics, 1)
		mb := unmarshalBody[qan_service.GetMetricsBody](t, metrics[0])
		require.Len(t, mb.Labels, 2)
		assert.Equal(t, "service_id", mb.Labels[1].Key)
		assert.ElementsMatch(t, want, mb.Labels[1].Value)

		examples := fake.requestsTo("/v1/qan/query:getExample")
		require.Len(t, examples, 1)
		eb := unmarshalBody[qan_service.GetQueryExampleBody](t, examples[0])
		require.Len(t, eb.Labels, 2)
		assert.Equal(t, "service_id", eb.Labels[1].Key)
		assert.ElementsMatch(t, want, eb.Labels[1].Value)
	})

	t.Run("GivenServiceIDCarriesOnlyIt", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-2"})
		require.False(t, res.IsError, textOf(t, res))
		mb := unmarshalBody[qan_service.GetMetricsBody](t, fake.requestsTo("/v1/qan:getMetrics")[0])
		assert.Equal(t, []string{"svc-2"}, mb.Labels[1].Value)
		eb := unmarshalBody[qan_service.GetQueryExampleBody](t, fake.requestsTo("/v1/qan/query:getExample")[0])
		assert.Equal(t, []string{"svc-2"}, eb.Labels[1].Value)
	})

	t.Run("EmptyServiceList", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, withRoutes(qanRoutes(), map[string]fixture{serviceListKey: {file: "vm_empty.json"}}))
		session := connect(t, newQANService(t, fake, true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		assert.True(t, res.IsError)
		assert.Equal(t, "error: not_found\n"+errNotInList(window{From: testNow.Add(-time.Hour), To: testNow}, "").message, textOf(t, res))
		assert.Empty(t, fake.requestsTo("/v1/qan:getMetrics"))
		assert.Empty(t, fake.requestsTo("/v1/qan/query:getExample"))
	})

	t.Run("FailedPartAndNoOtherData", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name   string
			routes map[string]fixture
			path   string
		}{
			{
				name:   "MetricsFailNoExample",
				routes: map[string]fixture{"POST /v1/qan:getMetrics": failed, "POST /v1/qan/query:getExample": {file: "example_empty.json"}},
				path:   "/v1/qan:getMetrics",
			},
			{
				name:   "ExampleFailsNoMetrics",
				routes: map[string]fixture{"POST /v1/qan:getMetrics": {file: "metrics_empty.json"}, "POST /v1/qan/query:getExample": failed},
				path:   "/v1/qan/query:getExample",
			},
		} {
			session := connect(t, newQANService(t, newFakePMM(t, withRoutes(qanRoutes(), tc.routes)), true))

			res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
			text := textOf(t, res)
			assert.True(t, res.IsError, "%s: %s", tc.name, text)
			assert.True(t, strings.HasPrefix(text, "error: pmm_unavailable\nPMM returned 500 on POST "+tc.path+": "), "%s: %s", tc.name, text)
			assert.Equal(t, map[string]any{"code": "pmm_unavailable", "retryable": true}, res.Meta["com.percona/error"], tc.name)
		}
	})

	from := testNow.Add(-time.Hour)
	rawExample := "SELECT * FROM customers WHERE email = 'user42@example.com'" //nolint:unqueryvet
	detailURL := "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
		"filter_by=QID-AAA&from=" + ms(from) + "&query_selected=true&to=" + ms(testNow) + "&var-service_name=shop-mysql"

	t.Run("Structured", func(t *testing.T) {
		t.Parallel()

		session := connect(t, newQANService(t, newFakePMM(t, qanRoutes()), true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc-1"})
		got := structured[queryDetailResult](t, res)
		assert.Equal(t, queryDetailResult{
			QueryID:     "QID-AAA",
			Engine:      "mysql",
			Version:     "8.0.46-37",
			ServiceName: "shop-mysql",
			Database:    "shop_db",
			Schema:      "shop",
			Fingerprint: "SELECT * FROM `customers` WHERE `email` = ?",
			Tables:      []string{"customers"},
			Metrics: map[string]float64{
				"calls": 1200, "query_time_sum": 88.5, "query_time_avg": 0.073, "rows_examined_avg": 20000,
				"rows_sent_avg": 1, "no_index_used": 1200, "full_scan": 1200,
			},
			Example:  rawExample,
			QueryRef: queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: from, PeriodTo: testNow},
			Window:   window{From: from, To: testNow},
			PMMURL:   detailURL,
		}, got)

		text := textOf(t, res)
		for name := range got.Metrics {
			assert.Contains(t, text, " "+name+"=", "the text prints every structured metric under its name")
		}
		assert.Contains(t, text, "("+got.PMMURL+")")
		assert.Contains(t, text, "\nWindow: "+got.Window.From.Format(time.RFC3339)+" – "+got.Window.To.Format(time.RFC3339)+"\n")
		assert.Contains(t, string(mustJSON(t, res.StructuredContent)), `"window":{"period_from":"2026-09-14T11:00:00Z","period_to":"2026-09-14T12:00:00Z"}`)
		assert.NotContains(t, text, "partial")
	})

	t.Run("ExampleFails", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan/query:getExample"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		session := connect(t, newQANService(t, newFakePMM(t, routes), true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		got := structured[queryDetailResult](t, res)
		assert.True(t, got.Partial)
		require.NotNil(t, got.ExampleError)
		assert.Equal(t, codeUnauthorized, got.ExampleError.Code)
		assert.Contains(t, got.ExampleError.Message, "Access denied")
		assert.Nil(t, got.MetricsError)
		assert.InDelta(t, 1200, got.Metrics["calls"], 1e-9)
		assert.Empty(t, got.Example)
		assert.Empty(t, got.QueryRef.ServiceID, "neither the request nor a stored example names the service")

		text := textOf(t, res)
		assert.Contains(t, text, "\npartial: qan/query:getExample failed (unauthorized)\n")
		assert.Contains(t, text, "\nmetrics: calls=1200, ")
	})

	t.Run("MetricsFail", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan:getMetrics"] = fixture{file: "error_403.json", status: http.StatusForbidden}
		session := connect(t, newQANService(t, newFakePMM(t, routes), true))

		res := callTool(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		got := structured[queryDetailResult](t, res)
		assert.True(t, got.Partial)
		require.NotNil(t, got.MetricsError)
		assert.Equal(t, codeUnauthorized, got.MetricsError.Code)
		assert.Nil(t, got.ExampleError)
		assert.Nil(t, got.Metrics)
		assert.Equal(t, "shop", got.Schema)
		assert.Equal(t, rawExample, got.Example)
		assert.Equal(t, "svc-1", got.QueryRef.ServiceID, "from the stored example")
		assert.Contains(t, textOf(t, res), "\npartial: qan:getMetrics failed (unauthorized)\n")
	})

	t.Run("QueryRefServiceID", func(t *testing.T) {
		t.Parallel()

		example := &qan_service.GetQueryExampleOKBody{QueryExamples: []*qan_service.GetQueryExampleOKBodyQueryExamplesItems0{{ServiceID: "svc-example"}}}
		noExample := &qan_service.GetQueryExampleOKBody{}
		for _, tc := range []struct {
			name      string
			serviceID string
			examples  *qan_service.GetQueryExampleOKBody
			want      string
		}{
			{"RequestWithoutExample", "svc-request", noExample, "svc-request"},
			{"RequestOverExample", "svc-request", example, "svc-request"},
			{"Example", "", example, "svc-example"},
			{"Neither", "", noExample, ""},
		} {
			api := newMockPmmAPI(t)
			// qan-api2 never fills the metadata's service_id; one that is set must still be ignored.
			api.On("GetMetrics", mock.Anything, mock.Anything, mock.Anything).
				Return(&queryMetrics{Fingerprint: "SELECT ?", Metadata: &queryMetadata{ServiceID: "svc-metadata"}}, nil)
			api.On("GetQueryExample", mock.Anything, mock.Anything, mock.Anything).Return(tc.examples, nil)
			s, err := New(Params{API: api, Enabled: func() bool { return true }})
			require.NoError(t, err)

			d, err := s.fetchQueryDetail(t.Context(), callerAuth{}, queryDetailInput{QueryID: "Q", ServiceID: tc.serviceID}, []string{"svc-a"}, from, testNow)
			require.NoError(t, err)
			ref := d.result(true, "", from, testNow).QueryRef
			assert.Equal(t, queryRef{QueryID: "Q", ServiceID: tc.want, PeriodFrom: from, PeriodTo: testNow}, ref, tc.name)
			if tc.want == "" {
				assert.NotContains(t, string(mustJSON(t, ref)), "service_id", tc.name)
			}
		}
	})
}

func TestParseTime(t *testing.T) {
	t.Parallel()

	for expr, want := range map[string]time.Time{
		"":                     testNow,
		"now":                  testNow,
		"NOW":                  testNow,
		"now-1h":               testNow.Add(-time.Hour),
		"now - 30m":            testNow.Add(-30 * time.Minute),
		"now-2d":               testNow.Add(-48 * time.Hour),
		"now-45s":              testNow.Add(-45 * time.Second),
		"2026-09-14T10:00:00Z": time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC),
	} {
		got, err := parseTime(expr, testNow)
		require.NoError(t, err, expr)
		assert.True(t, want.Equal(got), "%s: want %s, got %s", expr, want, got)
	}

	for _, expr := range []string{"yesterday", "now+1h", "now-1y", "2026-09-14"} {
		_, err := parseTime(expr, testNow)
		require.Error(t, err, expr)
		assert.Equal(t, codeInvalidInput, mapError(err).code, expr)
	}
}

// TestParseWindowIsUTCSeconds pins that a window is UTC in whole seconds, so
// that the text's RFC 3339 window matches the structured one.
func TestParseWindowIsUTCSeconds(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 14, 14, 0, 0, 123456789, time.FixedZone("CEST", 2*60*60))
	from, to, err := parseWindow("now-1h", "", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-14T11:00:00Z", from.Format(time.RFC3339Nano))
	assert.Equal(t, "2026-09-14T12:00:00Z", to.Format(time.RFC3339Nano))

	from, to, err = parseWindow("2026-09-14T13:30:00.5+02:00", "2026-09-14T11:45:00Z", now)
	require.NoError(t, err)
	assert.Equal(t, "2026-09-14T11:30:00Z", from.Format(time.RFC3339Nano))
	assert.Equal(t, "2026-09-14T11:45:00Z", to.Format(time.RFC3339Nano))
}

func TestLinks(t *testing.T) {
	t.Parallel()

	from, to := testNow.Add(-time.Hour), testNow
	assert.Equal(t, "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from="+ms(from)+"&to="+ms(to),
		qanOverviewURL("https://pmm.example.com/", "", from, to))
	assert.Equal(t, "graph/d/pmm-qan/pmm-query-analytics?details_tab=explain&filter_by=QID-1&from="+ms(from)+"&query_selected=true&to="+ms(to)+"&var-service_name=s1",
		qanQueryURL("", "QID-1", "s1", from, to, "explain"))

	for engine, uid := range map[string]string{"mysql": "mysql", "postgresql": "postgresql", "mongodb": "mongodb"} {
		assert.Equal(t, "https://pmm.example.com/graph/d/"+uid+"-instance-summary?var-service_name=svc+1",
			serviceURL("https://pmm.example.com/", serviceInfo{ServiceName: "svc 1", Engine: engine}))
	}
	assert.Equal(t, "https://pmm.example.com/graph/d/haproxy-instance-summary?var-service_name=ha+1",
		serviceURL("https://pmm.example.com/", serviceInfo{ServiceName: "ha 1", NodeName: "n", Engine: "haproxy"}))
	assert.Equal(t, "https://pmm.example.com/graph/d/proxysql-instance-summary?var-proxysql=demo+node",
		serviceURL("https://pmm.example.com/", serviceInfo{ServiceName: "px", NodeName: "demo node", Engine: "proxysql"}))
	assert.Empty(t, serviceURL("https://pmm.example.com/", serviceInfo{ServiceName: "px", Engine: "proxysql"}))
	assert.Empty(t, serviceURL("https://pmm.example.com/", serviceInfo{ServiceName: "x", Engine: "valkey"}))
	assert.Empty(t, serviceURL("https://pmm.example.com/", serviceInfo{ServiceName: "x", Engine: "external"}))

	s, err := New(Params{API: &mockPmmAPI{}, Enabled: func() bool { return true }})
	require.NoError(t, err)
	assert.Empty(t, s.publicBaseURL(t.Context(), http.Header{}))
	assert.Equal(t, "https://pmm.local/", s.publicBaseURL(t.Context(), http.Header{"X-Forwarded-Host": {"pmm.local"}}))
	assert.Equal(t, "http://pmm.local:8080/", s.publicBaseURL(t.Context(), http.Header{"X-Forwarded-Host": {"pmm.local:8080"}, "X-Forwarded-Proto": {"http"}}))
	s.publicAddress = func(_ context.Context) string { return "pmm.example.com" }
	assert.Equal(t, "https://pmm.example.com/", s.publicBaseURL(t.Context(), http.Header{"X-Forwarded-Host": {"pmm.local"}}))
	s.publicAddress = func(_ context.Context) string { return "http://pmm.example.com:8080/" }
	assert.Equal(t, "http://pmm.example.com:8080/", s.publicBaseURL(t.Context(), http.Header{}))
}

// TestQANValuesAreChecked pins that a QAN filter value with a quote or a
// backslash is rejected before any QAN call (PMM-15715).
func TestQANValuesAreChecked(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tool string
		args map[string]any
	}{
		{"pmm_top_queries", map[string]any{"service_name": "svc'1"}},
		{"pmm_top_queries", map[string]any{"service_id": `svc\`}},
		{"pmm_query_detail", map[string]any{"queryid": "Q'1"}},
		{"pmm_query_detail", map[string]any{"queryid": "QID-AAA", "service_id": "svc'"}},
		{"pmm_get_explain", map[string]any{"service_id": "svc-1", "queryid": "Q'"}},
	} {
		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, false))

		text, isError := callText(t, session, tc.tool, tc.args)
		assert.True(t, isError, "%s %v", tc.tool, tc.args)
		assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"), text)
		for _, path := range []string{"/v1/qan/metrics:getReport", "/v1/qan:getMetrics", "/v1/qan/query:getExample"} {
			assert.Empty(t, fake.requestsTo(path), "%s %v -> %s", tc.tool, tc.args, path)
		}
		assert.Zero(t, listQueries(fake), "%s %v", tc.tool, tc.args)
	}
}

// TestKeyMetricsMongoDB pins the MongoDB counters pmm_query_detail shows, so the
// examined-to-returned ratio is visible; zero counters are left out.
func TestKeyMetricsMongoDB(t *testing.T) {
	t.Parallel()

	got := metricsText(keyMetrics(map[string]metricStats{
		"num_queries":                            {Sum: 50},
		"query_time":                             {Sum: 2.5, Avg: 0.05},
		"docs_examined":                          {Sum: 500000, Avg: 10000},
		"keys_examined":                          {},
		"docs_returned":                          {Sum: 50, Avg: 1},
		"response_length":                        {Sum: 6400, Avg: 128},
		"storage_bytes_read":                     {},
		"locks_global_acquire_count_read_shared": {Sum: 100, Avg: 2},
	}))
	assert.Equal(t, "calls=50, query_time_sum=2.5, query_time_avg=0.05, docs_examined_avg=10000, docs_returned_avg=1, "+
		"response_length_avg=128, locks_global_acquire_count_read_shared=100", got)
}

// TestRenderRowPrintsRankedValue checks that the order_by column also prints the value qan-api2 sorted by:
// the total for counters, the per-call average for time metrics.
func TestRenderRowPrintsRankedValue(t *testing.T) {
	t.Parallel()

	var row qanReportRow
	err := json.Unmarshal([]byte(`{"dimension": "app", "metrics": {
		"shared_blk_read_time": {"stats": {"sum": 30, "avg": 3}},
		"docs_examined": {"stats": {"sum": 20000, "avg": 2000}}}}`), &row)
	require.NoError(t, err)

	for _, tc := range []struct {
		orderBy string
		want    []string
		notWant []string
	}{
		{"-shared_blk_read_time", []string{" shared_blk_read_time_avg=3 shared_blk_read_time=30", " docs_examined_avg=2000"}, []string{"docs_examined=20000"}},
		{"-docs_examined", []string{" docs_examined_avg=2000 docs_examined=20000", " shared_blk_read_time=30"}, []string{"shared_blk_read_time_avg="}},
	} {
		t.Run(tc.orderBy, func(t *testing.T) {
			t.Parallel()

			q := &reportQuery{groupBy: "username", orderBy: tc.orderBy, columns: []string{"shared_blk_read_time", "docs_examined"}}
			r := q.row(row, 1, "", false)
			line := topQueriesResult{GroupBy: q.groupBy, OrderBy: q.orderBy}.renderRow(r)
			assert.Equal(t, map[string]float64{"shared_blk_read_time": 30, "docs_examined": 2000}, r.Columns,
				"the catalogue's statistic, whatever the ranking")
			for _, w := range tc.want {
				assert.Contains(t, line, w)
			}
			for _, w := range tc.notWant {
				assert.NotContains(t, line, w)
			}
		})
	}
}

// TestTopQueriesReportInputs pins the getReport body that each input produces,
// and the header, paging and columns of a report not grouped by queryid.
func TestTopQueriesReportInputs(t *testing.T) {
	t.Parallel()

	routes := qanRoutes()
	routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report_by_user.json"}
	fake := newFakePMM(t, routes)
	session := connect(t, newQANService(t, fake, false))

	text, isError := callText(t, session, "pmm_top_queries", map[string]any{
		"service_name": "shop-mysql", "group_by": "username", "labels": map[string]any{"client_host": []string{"10.0.0.5"}},
		"columns": []string{"rows_examined", "full_scan", "rows_examined"}, "order_by": "-rows_examined",
		"offset": 10, "limit": 2, "search": "app",
	})
	require.False(t, isError, text)
	assert.Equal(t, strings.Join([]string{
		"Rows 11-12 of 14 (group_by=username, order_by=-rows_examined):",
		"Window: 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z",
		"",
		"11. [app] load=0.4 calls=700 total=120s avg=0.171s rows_examined_avg=100 rows_examined=70000 full_scan=12",
		"12. [(empty)] load=0.1 calls=200 total=30s avg=0.15s rows_examined_avg=0 rows_examined=0 full_scan=0",
		"",
		"[Open this workload in PMM QAN ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
			"from=" + ms(testNow.Add(-time.Hour)) + "&to=" + ms(testNow) + "&var-service_name=shop-mysql)",
		"showing 11–12 of 14; pass offset=12 for more",
	}, "\n"), text)

	body := unmarshalBody[qan_service.GetReportBody](t, fake.requestsTo("/v1/qan/metrics:getReport")[0])
	assert.Equal(t, "username", body.GroupBy)
	assert.Equal(t, "-rows_examined", body.OrderBy)
	assert.Equal(t, "rows_examined", body.MainMetric)
	assert.Equal(t, []string{"load", "num_queries", "query_time", "rows_examined", "full_scan"}, body.Columns)
	assert.EqualValues(t, 10, body.Offset)
	assert.EqualValues(t, 2, body.Limit)
	assert.Equal(t, "app", body.Search)
	require.Len(t, body.Labels, 2)
	assert.Equal(t, "client_host", body.Labels[0].Key)
	assert.Equal(t, []string{"10.0.0.5"}, body.Labels[0].Value)
	assert.Equal(t, "service_name", body.Labels[1].Key)
	assert.Empty(t, fake.requestsTo("/v1/qan/metrics:getFilters"), "no fingerprints, so no engine lookup")

	t.Run("OrderColumnIsAdded", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))
		_, isError := callText(t, session, "pmm_top_queries", map[string]any{"order_by": "lock_time"})
		require.False(t, isError)
		body := unmarshalBody[qan_service.GetReportBody](t, fake.requestsTo("/v1/qan/metrics:getReport")[0])
		assert.Equal(t, "lock_time", body.OrderBy, "no - means ascending")
		assert.Equal(t, []string{"load", "num_queries", "query_time", "lock_time"}, body.Columns)
	})

	t.Run("PastTheEnd", func(t *testing.T) {
		t.Parallel()

		routes := qanRoutes()
		routes["POST /v1/qan/metrics:getReport"] = fixture{file: "report_past_end.json"}
		session := connect(t, newQANService(t, newFakePMM(t, routes), true))
		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"offset": 50})
		require.False(t, isError)
		assert.Equal(t, strings.Join([]string{
			"No rows at offset 50; the report has 14.",
			"Window: 2026-09-14T11:00:00Z – 2026-09-14T12:00:00Z",
			"",
			"[Open this workload in PMM QAN ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
				"from=" + ms(testNow.Add(-time.Hour)) + "&to=" + ms(testNow) + ")",
		}, "\n"), text)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Parallel()

		fake := newFakePMM(t, qanRoutes())
		session := connect(t, newQANService(t, fake, true))
		for _, tc := range []struct {
			args   map[string]any
			prefix string
		}{
			{map[string]any{"group_by": "service_id"}, "group_by must be one of"},
			{map[string]any{"columns": []string{"rows_examined", "nope"}}, "unknown column 'nope'; pmm_catalog"},
			{map[string]any{"columns": []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}}, "columns takes at most 10"},
			{map[string]any{"order_by": "-nope"}, "order_by must be load, total_query_time, avg_query_time, count, or a column from pmm_catalog"},
			{map[string]any{"offset": -1}, "offset must not be negative; got -1"},
			{map[string]any{"limit": -1}, "limit must be between 1 and 100; got -1"},
			{map[string]any{"limit": 101}, "limit must be between 1 and 100; got 101"},
			{map[string]any{"search": strings.Repeat("x", 201)}, "search must be at most 200"},
			{map[string]any{"labels": map[string]any{"user name": []string{"a"}}}, "label name 'user name'"},
			{map[string]any{"labels": map[string]any{"username": []string{}}}, "labels.username needs at least one value"},
			{map[string]any{"labels": map[string]any{"username": []string{"o'neil"}}}, "labels.username must not contain"},
			{map[string]any{"service_name": "a", "labels": map[string]any{"service_name": []string{"b"}}}, "service_name and labels.service_name disagree"},
		} {
			text, isError := callText(t, session, "pmm_top_queries", tc.args)
			assert.True(t, isError, "%v", tc.args)
			assert.True(t, strings.HasPrefix(text, "error: invalid_input\n"+tc.prefix), "%v: %s", tc.args, text)
		}
		assert.Empty(t, fake.requestsTo("/v1/qan/metrics:getReport"))
	})

	t.Run("PointsToCatalog", func(t *testing.T) {
		t.Parallel()

		session := connect(t, newQANService(t, newFakePMM(t, qanRoutes()), true))
		tool := listTools(t, session)["pmm_top_queries"]
		require.NotNil(t, tool)
		assert.NotContains(t, tool.Description, "docs_examined")
		assert.Contains(t, tool.Description, "pmm_catalog")
		props := string(mustJSON(t, tool.InputSchema))
		var schema struct {
			Properties map[string]struct {
				Description string `json:"description"`
			} `json:"properties"`
		}
		require.NoError(t, json.Unmarshal([]byte(props), &schema))
		assert.Contains(t, schema.Properties["columns"].Description, "pmm_catalog")
		assert.Contains(t, schema.Properties["order_by"].Description, "pmm_catalog")
	})
}

// mixedEngineReports answers getReport with the unfiltered report, or with its
// MySQL-only version when the request filters on service_type; status, when set,
// is the answer to that second request instead.
func mixedEngineReports(t *testing.T, status int) *fakePMM {
	t.Helper()

	routes := qanRoutes()
	routes["POST /v1/qan/metrics:getFilters"] = fixture{file: "filters_mixed.json"}
	fake := newFakePMM(t, routes)
	prev := fake.server.Config.Handler
	fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/qan/metrics:getReport" {
			prev.ServeHTTP(rw, req)
			return
		}
		body, _ := io.ReadAll(req.Body)
		fake.mu.Lock()
		fake.requests = append(fake.requests, recordedRequest{method: req.Method, path: req.URL.Path, header: req.Header.Clone(), body: body})
		fake.mu.Unlock()

		file := "report_mixed_engines.json"
		if strings.Contains(string(body), `"service_type"`) {
			if status != 0 {
				rw.WriteHeader(status)
				return
			}
			file = "report_mixed_engines_mysql.json"
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write(readFixture(t, file))
	})
	return fake
}

// fingerprintOf returns the fingerprint line under the row of a queryid.
func fingerprintOf(t *testing.T, text, queryID string) string {
	t.Helper()

	lines := strings.Split(text, "\n")
	i := slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, "["+queryID+"]") })
	require.GreaterOrEqual(t, i, 0, text)
	require.Less(t, i+1, len(lines), text)
	return strings.TrimPrefix(lines[i+1], "   ")
}

// TestTopQueriesEnginePerRow pins the per-row engine of an unfiltered report over
// MySQL and PostgreSQL: one more report, filtered to MySQL, decides each row.
func TestTopQueriesEnginePerRow(t *testing.T) {
	t.Parallel()

	fake := mixedEngineReports(t, 0)
	session := connect(t, newQANService(t, fake, false))
	text, isError := callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
	require.False(t, isError, text)
	assert.Equal(t, "SELECT `id` FROM `o'neil` WHERE `id` = ?", fingerprintOf(t, text, "Q-MY"))
	assert.Equal(t, `SELECT "id" FROM "users" WHERE "email" = $1`, fingerprintOf(t, text, "Q-PG"))
	assert.Equal(t, withheldFingerprint, fingerprintOf(t, text, "Q-BOTH"), "calls under both engines")

	reqs := fake.requestsTo("/v1/qan/metrics:getReport")
	require.Len(t, reqs, 2)
	body := unmarshalBody[qan_service.GetReportBody](t, reqs[1])
	assert.EqualValues(t, 3, body.Limit)
	require.Len(t, body.Labels, 2)
	assert.Equal(t, "queryid", body.Labels[0].Key)
	assert.Equal(t, []string{"Q-MY", "Q-BOTH", "Q-PG"}, body.Labels[0].Value)
	assert.Equal(t, "service_type", body.Labels[1].Key)
	assert.Equal(t, []string{"mysql"}, body.Labels[1].Value)

	t.Run("LookupFails", func(t *testing.T) {
		t.Parallel()

		session := connect(t, newQANService(t, mixedEngineReports(t, http.StatusInternalServerError), false))
		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
		require.False(t, isError, text)
		for _, id := range []string{"Q-MY", "Q-BOTH", "Q-PG"} {
			assert.Equal(t, withheldFingerprint, fingerprintOf(t, text, id), id)
		}
	})

	t.Run("CallerFiltersServiceType", func(t *testing.T) {
		t.Parallel()

		fake := mixedEngineReports(t, 0)
		session := connect(t, newQANService(t, fake, false))
		text, isError := callText(t, session, "pmm_top_queries", map[string]any{"labels": map[string]any{"service_type": []string{"postgresql"}}})
		require.False(t, isError, text)
		assert.Equal(t, `SELECT "id" FROM "users" WHERE "email" = $1`, fingerprintOf(t, text, "Q-BOTH"), "read as PostgreSQL")
		assert.Len(t, fake.requestsTo("/v1/qan/metrics:getReport"), 1)
	})
}
