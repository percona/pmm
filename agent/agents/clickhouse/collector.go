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

// Package clickhouse provides a Prometheus collector for ClickHouse metrics.
//
// It emits metric families under the same names as ClickHouse's own native
// Prometheus endpoint (the <prometheus> server config section) —
// ClickHouseMetrics_*, ClickHouseAsyncMetrics_* and ClickHouseProfileEvents_* —
// so a single PMM dashboard set works whether metrics come from this exporter
// or from the native endpoint.
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // database/sql driver "clickhouse"
	"github.com/prometheus/client_golang/prometheus"
)

// Metric-family prefixes, identical to the native ClickHouse endpoint.
const (
	prefixMetrics        = "ClickHouseMetrics_"
	prefixAsyncMetrics   = "ClickHouseAsyncMetrics_"
	prefixProfileEvents  = "ClickHouseProfileEvents_"
	exporterScrapePrefix = "clickhouse_exporter_"
)

// scrapeTimeout bounds a single /metrics scrape of the ClickHouse server.
const scrapeTimeout = 10 * time.Second

// systemTable describes one ClickHouse system table that maps name/value rows
// to a Prometheus metric family.
type systemTable struct {
	table      string
	nameColumn string
	prefix     string
	valueType  prometheus.ValueType
}

// systemTables are the always-populated tables the collector scrapes. They are
// populated on a fresh server — unlike system.query_log, which is empty until
// the server has served traffic.
var systemTables = []systemTable{
	{"system.metrics", "metric", prefixMetrics, prometheus.GaugeValue},
	{"system.asynchronous_metrics", "metric", prefixAsyncMetrics, prometheus.GaugeValue},
	{"system.events", "event", prefixProfileEvents, prometheus.CounterValue},
}

// Collector is an unchecked Prometheus collector: the concrete metric names are
// discovered from the ClickHouse server at scrape time, so Describe emits
// nothing.
type Collector struct {
	client         *sql.DB
	scrapeDuration *prometheus.Desc
	scrapeSuccess  *prometheus.Desc
}

// NewCollector opens a connection pool to ClickHouse and returns a Collector.
// It fails fast when the server is unreachable.
func NewCollector(dsn string) (*Collector, error) {
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()
	pingErr := db.PingContext(pingCtx)
	if pingErr != nil {
		_ = db.Close()
		return nil, pingErr
	}
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
	}, nil
}

// Close releases the underlying connection pool.
func (c *Collector) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// Describe implements prometheus.Collector. The collector is unchecked: metric
// names depend on the ClickHouse server, so no descriptors are pre-declared.
func (c *Collector) Describe(chan<- *prometheus.Desc) {}

// Collect implements prometheus.Collector. It scrapes every system table and
// emits one metric per row plus the exporter's own scrape metrics. A failure
// on one table is logged and reflected in last_scrape_success; it never panics.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	success := 1.0
	for _, st := range systemTables {
		err := c.collectTable(ctx, ch, st)
		if err != nil {
			success = 0
			log.Printf("clickhouse_exporter: failed to scrape %s: %v", st.table, err)
		}
	}

	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, time.Since(start).Seconds())
	ch <- prometheus.MustNewConstMetric(c.scrapeSuccess, prometheus.GaugeValue, success)
}

// collectTable scrapes one ClickHouse system table and emits a metric per row.
// The value is cast to Float64 server-side so Int64 and Float64 columns are
// read uniformly across ClickHouse versions.
func (c *Collector) collectTable(ctx context.Context, ch chan<- prometheus.Metric, st systemTable) error {
	query := fmt.Sprintf("SELECT %s, toFloat64(value) FROM %s", st.nameColumn, st.table) //nolint:gosec
	rows, err := c.client.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close() //nolint:errcheck

	seen := make(map[string]struct{})
	for rows.Next() {
		var name string
		var value float64
		scanErr := rows.Scan(&name, &value)
		if scanErr != nil {
			return scanErr
		}

		metricName := st.prefix + sanitizeMetricName(name)
		// ClickHouse can expose the same name twice (e.g. duplicated async
		// metrics); Prometheus rejects duplicates, so keep the first.
		if _, dup := seen[metricName]; dup {
			continue
		}
		seen[metricName] = struct{}{}

		desc := prometheus.NewDesc(metricName, fmt.Sprintf("ClickHouse %s %q.", st.table, name), nil, nil)
		ch <- prometheus.MustNewConstMetric(desc, st.valueType, value)
	}
	return rows.Err()
}

// sanitizeMetricName replaces every character that is invalid in a Prometheus
// metric name with an underscore, matching the native ClickHouse endpoint
// (async-metric names such as "jemalloc.arenas.all.muzzy" contain dots).
func sanitizeMetricName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_':
			return r
		default:
			return '_'
		}
	}, name)
}
