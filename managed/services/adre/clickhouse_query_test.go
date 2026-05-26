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