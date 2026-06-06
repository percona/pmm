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

// Command backfill-rollup populates metrics_rollup_1h from historical data in
// the metrics table. The materialized view only rolls up rows inserted after it
// was created, so this one-time, day-by-day backfill fills the gap. Each day is
// dropped and re-inserted, so it is idempotent and can be resumed with --from.
package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register database/sql driver
	"github.com/alecthomas/kingpin/v2"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// backfillInsert aggregates one day of raw metrics into 1-hour partial aggregate
// states. It MUST stay in sync with the column list of migrations 23/24
// (metrics_rollup_1h and its materialized view). The first %d is the YYYYMMDD
// partition, the second is max_bytes_before_external_group_by.
const backfillInsert = `INSERT INTO metrics_rollup_1h
SELECT
  period_start,
  queryid, service_name, database, schema, username, service_type, environment, cluster, replication_set, fingerprint,
  sumState(num_queries),
  sumState(m_query_time_cnt), sumState(m_query_time_sum), minState(m_query_time_min), maxState(m_query_time_max), avgState(m_query_time_p99),
  sumState(m_lock_time_cnt), sumState(m_lock_time_sum), minState(m_lock_time_min), maxState(m_lock_time_max), avgState(m_lock_time_p99),
  sumState(m_rows_examined_cnt), sumState(m_rows_examined_sum), minState(m_rows_examined_min), maxState(m_rows_examined_max), avgState(m_rows_examined_p99),
  sumState(m_rows_sent_cnt), sumState(m_rows_sent_sum), minState(m_rows_sent_min), maxState(m_rows_sent_max), avgState(m_rows_sent_p99)
FROM
(
  SELECT
    toStartOfHour(period_start) AS period_start,
    queryid, service_name, database, schema, username, service_type, environment, cluster, replication_set, fingerprint,
    num_queries,
    m_query_time_cnt, m_query_time_sum, m_query_time_min, m_query_time_max, m_query_time_p99,
    m_lock_time_cnt, m_lock_time_sum, m_lock_time_min, m_lock_time_max, m_lock_time_p99,
    m_rows_examined_cnt, m_rows_examined_sum, m_rows_examined_min, m_rows_examined_max, m_rows_examined_p99,
    m_rows_sent_cnt, m_rows_sent_sum, m_rows_sent_min, m_rows_sent_max, m_rows_sent_p99
  FROM metrics
  WHERE toYYYYMMDD(period_start) = %d
)
GROUP BY
  period_start,
  queryid, service_name, database, schema, username, service_type, environment, cluster, replication_set, fingerprint
SETTINGS max_bytes_before_external_group_by = %d`

func main() {
	dsnF := kingpin.Flag("dsn", "ClickHouse DSN").
		Default("clickhouse://default:clickhouse@127.0.0.1:9000/pmm").Envar("PMM_CLICKHOUSE_DSN").String()
	fromF := kingpin.Flag("from", "Earliest day to backfill, YYYYMMDD (default: earliest available)").String()
	toF := kingpin.Flag("to", "Latest day to backfill, YYYYMMDD (default: yesterday)").String()
	maxBytesF := kingpin.Flag("max-bytes-before-external-group-by",
		"Spill the per-day aggregation to disk above this many bytes (0 disables)").Default("2000000000").Int64()
	dryRunF := kingpin.Flag("dry-run", "List the day-partitions that would be backfilled and exit").Bool()
	kingpin.Parse()

	l := logrus.WithField("component", "backfill-rollup")

	from, err := parseDay(*fromF)
	if err != nil {
		l.Fatalf("invalid --from: %v", err)
	}
	to, err := parseDay(*toF)
	if err != nil {
		l.Fatalf("invalid --to: %v", err)
	}

	db, err := sqlx.Connect("clickhouse", *dsnF)
	if err != nil {
		l.Fatalf("connect: %v", err)
	}
	defer db.Close() //nolint:errcheck

	// Default upper bound excludes today, which the materialized view is still writing.
	now := time.Now().UTC()
	today := now.Year()*10000 + int(now.Month())*100 + now.Day()
	if to == 0 || to >= today {
		to = today - 1
	}

	days, err := targetDays(db, from, to)
	if err != nil {
		l.Fatalf("list partitions: %v", err)
	}
	if len(days) == 0 {
		l.Info("No metrics day-partitions to backfill in the selected range.")
		return
	}
	l.Infof("Backfilling %d day-partition(s): %d..%d", len(days), days[0], days[len(days)-1])

	if *dryRunF {
		for _, d := range days {
			l.Infof("[dry-run] would backfill %d", d)
		}
		return
	}

	for i, day := range days {
		start := time.Now()
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE metrics_rollup_1h DROP PARTITION %d", day)); err != nil {
			l.Fatalf("drop rollup partition %d: %v", day, err)
		}
		if _, err := db.Exec(fmt.Sprintf(backfillInsert, day, *maxBytesF)); err != nil {
			l.Fatalf("backfill partition %d: %v (resume with --from=%d)", day, err, day)
		}
		l.Infof("Backfilled %d (%d/%d) in %s", day, i+1, len(days), time.Since(start).Round(time.Millisecond))
	}
	l.Infof("Done. Backfilled %d day-partition(s).", len(days))
}

// parseDay validates a YYYYMMDD string and returns it as an int (0 if empty).
func parseDay(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	if len(s) != 8 { //nolint:mnd
		return 0, fmt.Errorf("expected YYYYMMDD, got %q", s)
	}
	d, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("expected YYYYMMDD, got %q", s)
	}
	return d, nil
}

// targetDays returns the metrics day-partitions within [from, to] (0 = unbounded),
// oldest first.
func targetDays(db *sqlx.DB, from, to int) ([]int, error) {
	var partitions []string
	const query = `
		SELECT DISTINCT partition FROM system.parts
		WHERE database = currentDatabase() AND table = 'metrics' AND active = 1
			AND match(partition, '^[0-9]{8}$')
		ORDER BY partition`
	if err := db.Select(&partitions, query); err != nil {
		return nil, err
	}

	days := make([]int, 0, len(partitions))
	for _, p := range partitions {
		day, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			continue
		}
		if from != 0 && day < from {
			continue
		}
		if to != 0 && day > to {
			continue
		}
		days = append(days, day)
	}
	return days, nil
}
