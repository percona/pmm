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
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check that Collector satisfies the prometheus.Collector contract.
var _ prometheus.Collector = (*Collector)(nil)

// Queries must match exactly the strings issued by Collector.collectTable.
const (
	metricsSQL      = "SELECT metric, toFloat64(value) FROM system.metrics"
	asyncMetricsSQL = "SELECT metric, toFloat64(value) FROM system.asynchronous_metrics"
	eventsSQL       = "SELECT event, toFloat64(value) FROM system.events"
)

// newTestCollector builds a Collector backed by the given *sql.DB, mirroring
// the descriptors NewCollector creates — without opening a real connection.
func newTestCollector(db *sql.DB) *Collector {
	return &Collector{
		client: db,
		scrapeDuration: prometheus.NewDesc(
			exporterScrapePrefix+"scrape_duration_seconds",
			"Duration of the last ClickHouse scrape in seconds.",
			nil, nil,
		),
		scrapeSuccess: prometheus.NewDesc(
			exporterScrapePrefix+"last_scrape_success",
			"Whether the last ClickHouse scrape succeeded (1) or not (0).",
			nil, nil,
		),
	}
}

// newMockDB returns a *sql.DB whose queries are matched by exact string equality.
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	return db, mock
}

// expectAllTables queues mocked responses for the three system-table queries.
func expectAllTables(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(metricsSQL).WillReturnRows(
		sqlmock.NewRows([]string{"metric", "value"}).AddRow("Query", 3.0).AddRow("Merge", 1.0))
	mock.ExpectQuery(asyncMetricsSQL).WillReturnRows(
		sqlmock.NewRows([]string{"metric", "value"}).AddRow("Uptime", 1234.0).AddRow("jemalloc.arenas.all.muzzy", 7.0))
	mock.ExpectQuery(eventsSQL).WillReturnRows(
		sqlmock.NewRows([]string{"event", "value"}).AddRow("SelectQuery", 42.0))
}

// gatheredValues registers the collector on a private registry, gathers it, and
// returns a metric-name -> value map.
func gatheredValues(t *testing.T, c *Collector) map[string]float64 {
	t.Helper()

	reg := prometheus.NewRegistry()
	require.NoError(t, reg.Register(c))

	mfs, err := reg.Gather()
	require.NoError(t, err)

	values := make(map[string]float64)
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			switch {
			case m.GetGauge() != nil:
				values[mf.GetName()] = m.GetGauge().GetValue()
			case m.GetCounter() != nil:
				values[mf.GetName()] = m.GetCounter().GetValue()
			}
		}
	}
	return values
}

func TestCollectorDescribeIsUnchecked(t *testing.T) {
	c := newTestCollector(nil)

	ch := make(chan *prometheus.Desc, 1)
	c.Describe(ch)
	close(ch)

	assert.Empty(t, ch, "an unchecked collector must not pre-declare descriptors")
}

func TestCollectorCollectNativeMetricNames(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close() //nolint:errcheck
	expectAllTables(mock)

	values := gatheredValues(t, newTestCollector(db))

	// Metric families carry the native ClickHouse prefixes and values.
	assert.InDelta(t, 3.0, values["ClickHouseMetrics_Query"], 0.0001)
	assert.InDelta(t, 1.0, values["ClickHouseMetrics_Merge"], 0.0001)
	assert.InDelta(t, 1234.0, values["ClickHouseAsyncMetrics_Uptime"], 0.0001)
	assert.InDelta(t, 42.0, values["ClickHouseProfileEvents_SelectQuery"], 0.0001)
	// Invalid characters in async-metric names are sanitized.
	assert.Contains(t, values, "ClickHouseAsyncMetrics_jemalloc_arenas_all_muzzy")
	// The exporter always reports its own scrape metrics.
	assert.Contains(t, values, "clickhouse_exporter_scrape_duration_seconds")
	assert.InDelta(t, 1.0, values["clickhouse_exporter_last_scrape_success"], 0.0001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollectorCollectTableErrorIsNotFatal(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close() //nolint:errcheck

	mock.ExpectQuery(metricsSQL).WillReturnError(sql.ErrConnDone)
	mock.ExpectQuery(asyncMetricsSQL).WillReturnRows(
		sqlmock.NewRows([]string{"metric", "value"}).AddRow("Uptime", 1.0))
	mock.ExpectQuery(eventsSQL).WillReturnRows(
		sqlmock.NewRows([]string{"event", "value"}).AddRow("SelectQuery", 1.0))

	values := gatheredValues(t, newTestCollector(db))

	// A failing table must not panic and must surface in last_scrape_success.
	assert.InDelta(t, 0.0, values["clickhouse_exporter_last_scrape_success"], 0.0001)
	// The tables that did succeed are still emitted.
	assert.InDelta(t, 1.0, values["ClickHouseAsyncMetrics_Uptime"], 0.0001)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSanitizeMetricName(t *testing.T) {
	cases := map[string]string{
		"Query":                     "Query",
		"jemalloc.arenas.all.muzzy": "jemalloc_arenas_all_muzzy",
		"already_clean_1":           "already_clean_1",
		"weird-name/with chars":     "weird_name_with_chars",
	}
	for in, want := range cases {
		assert.Equal(t, want, sanitizeMetricName(in))
	}
}
