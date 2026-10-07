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
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	detail := (&queryDetail{queryID: "Q", fingerprint: raw}).render(false, "", testNow.Add(-time.Hour), testNow)
	assert.NotContains(t, detail, "user42@example.com")
	assert.NotContains(t, detail, "4417")

	// explain_fingerprint is shown only when it passes the same quote check.
	for fp, shown := range map[string]bool{
		"SELECT `id` FROM `t` WHERE `email` = :1":           true,
		"SELECT `id` FROM `t` WHERE `email` = 'a@b.com'":    false,
		`SELECT "id" FROM "t" WHERE "email" = "leak@b.com"`: false,
	} {
		d := &queryDetail{queryID: "Q", engine: engineMySQL, example: &qan_service.GetQueryExampleOKBodyQueryExamplesItems0{ExplainFingerprint: fp}}
		assert.Equal(t, shown, strings.Contains(d.render(false, "", testNow.Add(-time.Hour), testNow), fp), fp)
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
	// none or several, or when QAN cannot tell, the fingerprint is withheld.
	for file, want := range map[string]string{
		"filters_mysql.json": "SELECT `id` FROM `o'neil` WHERE `id` = ?",
		"filters_mixed.json": withheldFingerprint,
		"":                   withheldFingerprint,
	} {
		if file != "" {
			routes["POST /v1/qan/metrics:getFilters"] = fixture{file: file}
		}
		session := connect(t, newQANService(t, newFakePMM(t, routes), false))
		text, isError = callText(t, session, "pmm_top_queries", map[string]any{"period_from": "now-1h"})
		require.False(t, isError, text)
		assert.Contains(t, text, want, file)
	}
}

func TestTopQueriesTool(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, qanRoutes())
	session := connect(t, newQANService(t, fake, true))

	text, isError := callText(t, session, "pmm_top_queries", map[string]any{"service_name": "shop-mysql", "period_from": "now-1h"})
	assert.False(t, isError)
	assert.Equal(t, strings.Join([]string{
		"Top 2 queries (order_by=load):",
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
		assert.Equal(t, "No queries found for that service / time window.", text)
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
			"[View this query in PMM ↗](https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?" +
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

		text, isError := callText(t, session, "pmm_query_detail", map[string]any{"queryid": "QID-AAA"})
		assert.False(t, isError)
		assert.NotContains(t, text, "user42@example.com")
		assert.Contains(t, text, "example (normalized, literals stripped; raw SQL disabled):\n```sql\n")
		assert.Contains(t, text, "FROM `customers` WHERE `email` = :1\n```")
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

func TestLinks(t *testing.T) {
	t.Parallel()

	from, to := testNow.Add(-time.Hour), testNow
	assert.Equal(t, "https://pmm.example.com/graph/d/pmm-qan/pmm-query-analytics?from="+ms(from)+"&to="+ms(to),
		qanOverviewURL("https://pmm.example.com/", "", from, to))
	assert.Equal(t, "graph/d/pmm-qan/pmm-query-analytics?details_tab=explain&filter_by=QID-1&from="+ms(from)+"&query_selected=true&to="+ms(to)+"&var-service_name=s1",
		qanQueryURL("", "QID-1", "s1", from, to, "explain"))

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
