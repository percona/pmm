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

package adre

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/utils/sqlrows"
)

func TestValidateClickHouseQuery_acceptsQANAndLogs(t *testing.T) {
	t.Parallel()

	qan := `SELECT fingerprint, schema, any(queryid) AS queryid, SUM(m_query_time_sum) AS total_query_time
FROM pmm.metrics
WHERE service_id = 'abc' AND period_start >= '2026-03-22 12:00:00'
GROUP BY fingerprint, schema
ORDER BY total_query_time DESC
LIMIT 5`
	got, err := validateClickHouseQuery("pmm", qan, 500)
	require.NoError(t, err)
	assert.Contains(t, got, "SETTINGS max_execution_time=30")
	assert.Contains(t, got, "readonly=1")

	logs := `SELECT Timestamp, Body FROM otel.logs
WHERE ResourceAttributes['node_name'] = 'mysql'
ORDER BY Timestamp DESC
LIMIT 50`
	got, err = validateClickHouseQuery("otel", logs, 500)
	require.NoError(t, err)
	assert.NotContains(t, got, "LIMIT 500")
}

func TestValidateClickHouseQuery_rejectsUnsafeSQL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		db    string
		query string
	}{
		{"insert", "pmm", "INSERT INTO metrics VALUES (1)"},
		{"multi_statement", "pmm", "SELECT 1; SELECT 2"},
		{"wrong_table", "pmm", "SELECT 1 FROM system.tables"},
		{"cross_db", "pmm", "SELECT 1 FROM otel.logs"},
		{"bad_db", "foo", "SELECT 1 FROM metrics"},
		{"limit_too_high", "pmm", "SELECT count() FROM metrics LIMIT 2000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateClickHouseQuery(tc.db, tc.query, 500)
			require.Error(t, err)
		})
	}
}

func TestValidateClickHouseQuery_appendsLimit(t *testing.T) {
	t.Parallel()

	got, err := validateClickHouseQuery("pmm", "SELECT count() FROM metrics", 100)
	require.NoError(t, err)
	assert.Contains(t, got, "LIMIT 100")
}

func TestValidateClickHouseQuery_allowsReplaceAndFormatFunctions(t *testing.T) {
	t.Parallel()

	cases := []string{
		`SELECT replace(fingerprint, 'x', 'y') FROM metrics LIMIT 1`,
		`SELECT format('{}', fingerprint) FROM metrics LIMIT 1`,
		`SELECT fingerprint FROM metrics WHERE example LIKE '%FORMAT%' LIMIT 1`,
	}
	for _, q := range cases {
		_, err := validateClickHouseQuery("pmm", q, 500)
		require.NoError(t, err, "query: %s", q)
	}
}

func TestValidateClickHouseQuery_stripsSurroundingQuotes(t *testing.T) {
	t.Parallel()

	cases := []string{
		"'SELECT count() FROM metrics LIMIT 1'",
		"\"SELECT count() FROM metrics LIMIT 1\"",
		"''SELECT count() FROM metrics LIMIT 1''",
	}
	for _, q := range cases {
		got, err := validateClickHouseQuery("pmm", q, 500)
		require.NoError(t, err, "query: %s", q)
		assert.Contains(t, got, "SELECT count() FROM metrics")
	}
}

func TestValidateClickHouseQuery_normalizesHolmesShellEscaping(t *testing.T) {
	t.Parallel()

	logs := `'SELECT Timestamp, Body FROM logs WHERE ResourceAttributes['"node_name"'] = '"mysql"' ORDER BY Timestamp DESC LIMIT 10'`
	got, err := validateClickHouseQuery("otel", logs, 10)
	require.NoError(t, err)
	assert.Contains(t, got, "ResourceAttributes['node_name']")
	assert.Contains(t, got, "= 'mysql'")
	assert.NotContains(t, got, `'"`)

	holmesJoin := `SELECT Timestamp, Body FROM logs WHERE ResourceAttributes['"'"'node_name'"'"'] = '"'"'mysql'"'"' ORDER BY Timestamp DESC LIMIT 10`
	got, err = validateClickHouseQuery("otel", holmesJoin, 10)
	require.NoError(t, err)
	assert.Contains(t, got, "ResourceAttributes['node_name']")
	assert.Contains(t, got, "= 'mysql'")
}

func TestValidateClickHouseQuery_normalizesLLMEscaping(t *testing.T) {
	t.Parallel()

	logs := `SELECT Timestamp, Body FROM logs WHERE ResourceAttributes[\'node_name\'] = \'mysql\' ORDER BY Timestamp DESC LIMIT 10`
	got, err := validateClickHouseQuery("otel", logs, 10)
	require.NoError(t, err)
	assert.Contains(t, got, "ResourceAttributes['node_name']")
	assert.Contains(t, got, "= 'mysql'")
	assert.NotContains(t, got, `\`)

	logsDoubleKey := `SELECT Timestamp, Body FROM logs WHERE ResourceAttributes["node_name"] = 'mysql' ORDER BY Timestamp DESC LIMIT 10`
	got, err = validateClickHouseQuery("otel", logsDoubleKey, 10)
	require.NoError(t, err)
	assert.Contains(t, got, "ResourceAttributes['node_name']")
}

func TestValidateClickHouseQuery_repairsDateTimeMissingSpace(t *testing.T) {
	t.Parallel()

	// Non-frontier models (e.g. the Holmes default model used by the Slack path when no model is
	// pinned) sometimes drop the space in datetime literals, producing ClickHouse error 53.
	qan := `SELECT fingerprint, SUM(m_query_time_sum) AS total_query_time
FROM pmm.metrics
WHERE service_id = 'abc' AND period_start >= '2026-06-0408:10:08' AND period_start <= '2026-06-0420:10:08'
GROUP BY fingerprint
LIMIT 5`
	got, err := validateClickHouseQuery("pmm", qan, 500)
	require.NoError(t, err)
	assert.Contains(t, got, "'2026-06-04 08:10:08'")
	assert.Contains(t, got, "'2026-06-04 20:10:08'")
	assert.NotContains(t, got, "'2026-06-0408:10:08'")

	// Already-valid formats (space or ISO 'T') must be left untouched.
	for _, ts := range []string{"'2026-06-04 08:10:08'", "'2026-06-04T08:10:08'"} {
		q := "SELECT count() FROM metrics WHERE period_start >= " + ts + " LIMIT 1"
		got, err := validateClickHouseQuery("pmm", q, 500)
		require.NoError(t, err, "query: %s", q)
		assert.Contains(t, got, ts, "valid datetime literal must be preserved")
	}
}

func TestValidateClickHouseQuery_rejectsFormatExportAndReplaceTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		query string
	}{
		{"format_json", "SELECT 1 FROM metrics LIMIT 1 FORMAT JSON"},
		{"replace_table", "REPLACE TABLE metrics SELECT 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateClickHouseQuery("pmm", tc.query, 500)
			require.Error(t, err)
		})
	}
}

func TestValidateClickHouseQuery_rejectsTableFunctionBypasses(t *testing.T) {
	t.Parallel()

	// Each of these passes the FROM/JOIN table allowlist (the first source is the
	// allowlisted table) but smuggles a ClickHouse table function or a second,
	// non-allowlisted table to read files, reach arbitrary hosts (SSRF), or read
	// other tables. They must all be rejected.
	cases := []struct {
		name  string
		db    string
		query string
	}{
		{"ssrf_comma_cross_join", "pmm", `SELECT * FROM metrics, url('http://169.254.169.254/latest/meta-data/','CSV','a String') LIMIT 1`},
		{"file_read_projection", "pmm", `SELECT file('/etc/passwd') FROM metrics LIMIT 1`},
		{"remote_table_fn", "pmm", `SELECT * FROM remote('1.2.3.4:9000', system, tables) LIMIT 1`},
		{"mysql_table_fn_join", "otel", `SELECT * FROM logs JOIN mysql('h:3306','db','t','u','p') AS m ON m.x = logs.x LIMIT 1`},
		{"merge_other_tables", "pmm", `SELECT * FROM merge('system', '.*') LIMIT 1`},
		{"s3_in_where_subquery", "pmm", `SELECT count() FROM metrics WHERE 1 IN (SELECT 1 FROM s3('http://x/y.csv','CSV','c String')) LIMIT 1`},
		{"view_wrap", "pmm", `SELECT * FROM view(SELECT 1) LIMIT 1`},
		{"comma_forbidden_table", "pmm", `SELECT * FROM metrics, system.tables LIMIT 1`},
		{"comma_forbidden_table_aliased", "pmm", `SELECT * FROM metrics m, system.tables s LIMIT 1`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateClickHouseQuery(tc.db, tc.query, 500)
			require.Error(t, err, "query must be rejected: %s", tc.query)
		})
	}
}

func TestValidateClickHouseQuery_allowsSubqueriesJoinsAndLiterals(t *testing.T) {
	t.Parallel()

	// Legitimate analytics shapes must still pass: subqueries (inner table is the
	// one validated), aliases/JOINs over allowlisted tables, scalar functions that
	// merely share a name prefix, and forbidden tokens that appear only inside a
	// string literal (masked before scanning).
	cases := []struct {
		name  string
		db    string
		query string
	}{
		{"subquery", "pmm", `SELECT fingerprint FROM (SELECT fingerprint FROM metrics WHERE service_id = 'abc') AS t LIMIT 10`},
		{"alias", "pmm", `SELECT count() FROM metrics AS m WHERE m.service_id = 'abc' LIMIT 1`},
		{"replace_scalar", "pmm", `SELECT replace(fingerprint, 'a', 'b') FROM metrics LIMIT 1`},
		{"function_name_in_literal", "otel", `SELECT Timestamp FROM otel.logs WHERE Body LIKE '%url(%' LIMIT 5`},
		{"table_name_in_literal", "otel", `SELECT Timestamp FROM otel.logs WHERE Body LIKE '%system.tables%' LIMIT 5`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateClickHouseQuery(tc.db, tc.query, 500)
			require.NoError(t, err, "query must be allowed: %s", tc.query)
		})
	}
}

func TestValidateClickHouseQuery_rejectsScalarReadersAndCommentBypasses(t *testing.T) {
	t.Parallel()

	// Hardening: scalar dictionary/join readers (which run even under readonly=1 and read data outside
	// the allowlist), table functions that were missing from the denylist, comment-based denylist
	// evasions (url/**/(), and the no-space FROM(fn(...)) form must all be rejected.
	cases := []struct {
		name  string
		db    string
		query string
	}{
		{"dictGet_scalar", "pmm", `SELECT dictGet('some_dict','attr', toUInt64(1)) FROM metrics LIMIT 1`},
		{"dictGetString_scalar", "pmm", `SELECT dictGetString('d','a', toUInt64(service_id)) FROM metrics LIMIT 1`},
		{"joinGet_scalar", "otel", `SELECT joinGet('some_join','val', 1) FROM otel.logs LIMIT 1`},
		{"executable_projection", "pmm", `SELECT executable('id','CSV','x String') FROM metrics LIMIT 1`},
		{"s3queue_projection", "pmm", `SELECT s3queue('http://x','CSV') FROM metrics LIMIT 1`},
		{"mergeTreeIndex_from", "pmm", `SELECT * FROM mergeTreeIndex('system','tables') LIMIT 1`},
		{"gcsCluster_variant", "pmm", `SELECT * FROM gcsCluster('c','http://x','CSV','a String') LIMIT 1`},
		{"url_block_comment_projection", "otel", `SELECT url/**/('http://169.254.169.254/','CSV','x String') FROM otel.logs LIMIT 1`},
		{"file_block_comment_projection", "pmm", `SELECT file/**/('/etc/passwd','LineAsString','x String') FROM metrics LIMIT 1`},
		{"url_line_comment_subquery", "pmm", "SELECT * FROM metrics WHERE 1 = (SELECT url --c\n('http://x','CSV','x String')) LIMIT 1"},
		{"from_no_space_executable", "pmm", `SELECT * FROM metrics UNION ALL SELECT * FROM(executable('id','CSV','x String')) LIMIT 1`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateClickHouseQuery(tc.db, tc.query, 500)
			require.Error(t, err, "query must be rejected: %s", tc.query)
		})
	}
}

func TestValidateClickHouseQuery_allowsComments(t *testing.T) {
	t.Parallel()

	// Comment stripping must not break legitimate queries, and a comment delimiter that appears only
	// inside a string literal must NOT be treated as a comment (the quote-aware lexer keeps it inside
	// the literal, which is then emptied).
	cases := []struct {
		name  string
		db    string
		query string
	}{
		{"block_comment", "pmm", `SELECT /* pick fingerprint */ fingerprint FROM metrics LIMIT 5`},
		{"line_comment", "pmm", "SELECT fingerprint FROM metrics -- only this column\nWHERE service_id = 'abc' LIMIT 5"},
		{"line_comment_token_in_literal", "otel", `SELECT Timestamp FROM otel.logs WHERE Body LIKE '%-- not a comment%' LIMIT 5`},
		{"block_comment_token_in_literal", "otel", `SELECT Timestamp FROM otel.logs WHERE Body LIKE '%/* not a comment */%' LIMIT 5`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := validateClickHouseQuery(tc.db, tc.query, 500)
			require.NoError(t, err, "query must be allowed: %s", tc.query)
		})
	}
}

func TestExecuteClickHouseQuery_mock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rows := sqlmock.NewRows([]string{"cnt"}).AddRow(uint64(42))
	mock.ExpectQuery("SELECT count\\(\\) FROM metrics LIMIT 10 SETTINGS max_execution_time=30, readonly=1").
		WillReturnRows(rows)

	prepared, err := validateClickHouseQuery("pmm", "SELECT count() FROM metrics", 10)
	require.NoError(t, err)

	chRows, err := db.QueryContext(context.Background(), prepared)
	require.NoError(t, err)
	columns, dataRows, err := sqlrows.ReadRows(chRows)
	require.NoError(t, err)

	assert.Equal(t, []string{"cnt"}, columns)
	require.Len(t, dataRows, 1)
}

func TestPostClickHouseQuery_methodNotAllowed(t *testing.T) {
	h := NewHandlers(nil, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
	req := httptest.NewRequest(http.MethodGet, "/v1/adre/clickhouse/query", nil)
	rec := httptest.NewRecorder()
	h.PostClickHouseQuery(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestPostClickHouseQuery_invalidJSON(t *testing.T) {
	h := NewHandlers(nil, &mockGrafanaAlertsFetcher{}, nil, ClickHousePools{})
	req := httptest.NewRequest(http.MethodPost, "/v1/adre/clickhouse/query", strings.NewReader("{"))
	rec := httptest.NewRecorder()
	h.PostClickHouseQuery(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "invalid JSON")
}

func TestPostClickHouseQuery_validationErrorBeforeDB(t *testing.T) {
	_, err := validateClickHouseQuery("pmm", "INSERT INTO metrics VALUES (1)", 500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden keyword")
}
