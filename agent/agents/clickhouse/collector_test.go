// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clickhouse

import (
	"database/sql"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check that Collector satisfies the prometheus.Collector contract.
var _ prometheus.Collector = (*Collector)(nil)

// queryLogSQL must match exactly the query issued by Collector.Collect.
const queryLogSQL = "SELECT count(*) FROM system.query_log WHERE event_time >= now() - interval 1 minute"

// newTestCollector builds a Collector backed by the given *sql.DB, mirroring
// the descriptors NewCollector creates — without opening a real connection.
func newTestCollector(db *sql.DB) *Collector {
	return &Collector{
		queryCount: prometheus.NewDesc(
			"clickhouse_query_count",
			"Número de queries executadas no ClickHouse (último minuto)",
			nil, nil,
		),
		scrapeSeconds: prometheus.NewDesc(
			"clickhouse_scrape_duration_seconds",
			"Tempo gasto para coletar métricas do ClickHouse",
			nil, nil,
		),
		client: db,
	}
}

// newMockDB returns a *sql.DB whose queries are matched by exact string equality.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	return db, mock
}

func TestCollectorDescribe(t *testing.T) {
	c := newTestCollector(nil)

	ch := make(chan *prometheus.Desc, 2)
	c.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}
	assert.Len(t, descs, 2, "Describe must emit both metric descriptors")
}

func TestCollectorCollectSuccess(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(queryLogSQL).
		WillReturnRows(sqlmock.NewRows([]string{"count()"}).AddRow(42))

	c := newTestCollector(db)

	// The scrape-duration metric is time-dependent, so compare only the
	// deterministic query-count metric.
	expected := `
# HELP clickhouse_query_count Número de queries executadas no ClickHouse (último minuto)
# TYPE clickhouse_query_count gauge
clickhouse_query_count 42
`
	err := testutil.CollectAndCompare(c, strings.NewReader(expected), "clickhouse_query_count")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectorCollectEmitsBothMetrics(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(queryLogSQL).
		WillReturnRows(sqlmock.NewRows([]string{"count()"}).AddRow(7))

	c := newTestCollector(db)

	assert.Equal(t, 2, testutil.CollectAndCount(c), "Collect must emit query-count and scrape-duration")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectorCollectQueryError(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(queryLogSQL).WillReturnError(sql.ErrConnDone)

	c := newTestCollector(db)

	// On a query failure Collect logs the error and emits nothing — and must
	// never panic or block.
	assert.Equal(t, 0, testutil.CollectAndCount(c))
	assert.NoError(t, mock.ExpectationsWereMet())
}
