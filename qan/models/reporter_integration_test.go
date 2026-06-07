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

package models

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan/migrations"
	"github.com/percona/pmm/utils/ddsketch"
)

// setupTestDB creates a throwaway DB with the qan schema and returns a connection.
// Skips the test when ClickHouse is unreachable (CI runs it).
func setupTestDB(t *testing.T, db string) driver.Conn {
	t.Helper()
	addr := envOr("PMM_CLICKHOUSE_ADDR", "127.0.0.1:9000")
	user := envOr("PMM_CLICKHOUSE_USER", "default")
	pass := envOr("PMM_CLICKHOUSE_PASSWORD", "clickhouse")
	ctx := t.Context()

	admin, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr}, Auth: clickhouse.Auth{Database: "default", Username: user, Password: pass},
	})
	if err != nil {
		t.Skipf("ClickHouse unavailable: %v", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	err = admin.Ping(pingCtx)
	if err != nil {
		t.Skipf("ClickHouse unavailable: %v", err)
	}

	require.NoError(t, admin.Exec(ctx, "DROP DATABASE IF EXISTS "+db))
	require.NoError(t, admin.Exec(ctx, "CREATE DATABASE "+db))
	t.Cleanup(func() { _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+db) })

	dsn := fmt.Sprintf("clickhouse://%s:%s@%s/%s?x-migrations-table=qan_schema_migrations", user, pass, addr, db)
	require.NoError(t, migrations.Run(dsn))

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr}, Auth: clickhouse.Auth{Database: db, Username: user, Password: pass},
	})
	require.NoError(t, err)
	return conn
}

func TestReportIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_rtest")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)

	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())
	mk := func(period uint32, nq, qtSum, qtCnt float32) *qanv1.MetricsBucket {
		return &qanv1.MetricsBucket{
			Queryid: "q1", ServiceId: "svc1", ServiceName: "mysql-1", Database: "db1", Schema: "public",
			CmdType: "SELECT", Fingerprint: "SELECT ?", PeriodStartUnixSecs: period, NumQueries: nq,
			MQueryTimeSum: qtSum, MQueryTimeCnt: qtCnt, MQueryTimeMin: 0.01, MQueryTimeMax: 0.4,
			MRowsSentSum: 10, MRowsSentCnt: qtCnt,
		}
	}
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{mk(base, 5, 0.5, 5), mk(base+60, 3, 0.3, 3)}))

	from, to := int64(base)-3600, int64(base)+3600
	res, err := rep.Report(ctx, ReportParams{FromSec: from, ToSec: to, GroupBy: "queryid", OrderBy: "-load", Limit: 10})
	require.NoError(t, err)

	require.Equal(t, uint64(1), res.Total.TotalRows)
	require.InDelta(t, 0.8, res.Total.QueryTimeSum, 1e-6)
	require.Equal(t, "SELECT ?", res.Fingerprints["q1"])

	require.Len(t, res.Rows, 1)
	row := res.Rows[0]
	require.Equal(t, "q1", row.Dimension)
	require.Equal(t, "db1", row.Database)
	require.InDelta(t, 8, row.NumQueries, 1e-6)
	require.InDelta(t, 0.8, row.QueryTimeSum, 1e-6)
	require.Equal(t, uint64(8), row.QueryTimeCnt)
	require.InDelta(t, 0.01, float64(row.QueryTimeMin), 1e-6) // exact min preserved
	require.InDelta(t, 0.4, float64(row.QueryTimeMax), 1e-6)  // exact max preserved
	require.InDelta(t, 20, row.RowsSentSum, 1e-6)             // 10 + 10
}

// TestIngestSketchIntegration proves the Phase B wire sketch flows through ingestion
// (bucketArgs + sketchToMap) into storage and back out as a correct percentile.
func TestIngestSketchIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_wiresk")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())

	dense := ddsketch.New()
	for i := 1; i <= 1000; i++ {
		ddsketch.Add(dense, float64(i)/1000.0)
	}
	wire := map[uint32]uint64{}
	for i, c := range dense {
		if c > 0 {
			wire[uint32(i)] = c
		}
	}

	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{{
		Queryid: "q1", ServiceId: "svc1", Database: "db1", Schema: "public", CmdType: "SELECT",
		PeriodStartUnixSecs: base, NumQueries: 1000, MQueryTimeSum: 500.5, MQueryTimeCnt: 1000,
		MQueryTimeSketch: wire,
	}}))

	got, err := rep.Histogram(ctx, "q1", int64(base)-3600, int64(base)+3600)
	require.NoError(t, err)
	require.NotEmpty(t, got, "ingested wire sketch must reach storage")
	require.InDelta(t, 0.989, ddsketch.QuantileFromMap(got, 0.99), 0.989*ddsketch.Alpha+1e-9)
}

func TestReportSketchP99Integration(t *testing.T) {
	conn := setupTestDB(t, "qan_sktest")
	ctx := t.Context()
	rep := NewReporter(conn)
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())

	// Build a sketch from a known latency distribution (1ms..1000ms).
	dense := ddsketch.New()
	for i := 1; i <= 1000; i++ {
		ddsketch.Add(dense, float64(i)/1000.0)
	}
	sketch := map[uint16]uint64{}
	for i, c := range dense {
		if c > 0 {
			sketch[uint16(i)] = c
		}
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO metrics_raw (queryid, service_id, `database`, `schema`, cmd_type, period_start, num_queries, m_query_time_sum, m_query_time_cnt, m_query_time_sketch)")
	require.NoError(t, err)
	require.NoError(t, batch.Append("q2", "svc1", "db1", "public", "SELECT", time.Unix(int64(base), 0).UTC(), float64(1000), 500.5, uint64(1000), sketch))
	require.NoError(t, batch.Send())

	from, to := int64(base)-3600, int64(base)+3600
	res, err := rep.Report(ctx, ReportParams{FromSec: from, ToSec: to, GroupBy: "queryid", Limit: 10})
	require.NoError(t, err)
	require.Len(t, res.Rows, 1)
	require.NotEmpty(t, res.Rows[0].QueryTimeSketch, "sketch must survive the sumMap round-trip")

	got := ddsketch.QuantileFromMap(res.Rows[0].QueryTimeSketch, 0.99)
	want := 0.989 // exact p99 of 0.001..1.000 at rank 0.99*(1000-1)
	require.InDelta(t, want, got, want*ddsketch.Alpha+1e-9)
}
