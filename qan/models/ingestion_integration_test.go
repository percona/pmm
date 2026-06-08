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
	"os"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan/migrations"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// TestIngestIntegration ingests buckets into a throwaway DB and verifies the
// rollup, long-tail map, and dedup. Skips when ClickHouse is unreachable (CI runs it).
func TestIngestIntegration(t *testing.T) {
	addr := envOr("PMM_CLICKHOUSE_ADDR", "127.0.0.1:9000")
	user := envOr("PMM_CLICKHOUSE_USER", "default")
	pass := envOr("PMM_CLICKHOUSE_PASSWORD", "clickhouse")
	ctx := t.Context()

	admin, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: "default", Username: user, Password: pass},
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

	const db = "qan_itest"
	require.NoError(t, admin.Exec(ctx, "DROP DATABASE IF EXISTS "+db))
	require.NoError(t, admin.Exec(ctx, "CREATE DATABASE "+db))
	t.Cleanup(func() { _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+db) })

	dsn := fmt.Sprintf("clickhouse://%s:%s@%s/%s?x-migrations-table=qan_schema_migrations", user, pass, addr, db)
	require.NoError(t, migrations.Run(dsn))

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: db, Username: user, Password: pass},
	})
	require.NoError(t, err)
	ing := NewIngestor(conn)

	mk := func(period uint32, nq, qtSum, qtCnt float32) *qanv1.MetricsBucket {
		return &qanv1.MetricsBucket{
			Queryid: "q1", ServiceId: "svc1", ServiceName: "mysql-1", Database: "db1", Schema: "public",
			CmdType: "SELECT", PeriodStartUnixSecs: period, NumQueries: nq,
			MQueryTimeSum: qtSum, MQueryTimeCnt: qtCnt, MQueryTimeMin: 0.01, MQueryTimeMax: 0.4,
			MRowsReadSum: 7, MRowsReadCnt: qtCnt, // non-core -> long-tail map
		}
	}

	// Two buckets in the same recent hour (within the metrics_raw TTL).
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-2 * time.Hour).Unix())
	p1, p2 := base, base+1800
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{
		mk(p1, 5, 0.5, 5),
		mk(p2, 3, 0.3, 3),
	}))
	// Resend the first bucket: at-least-once retry must be deduped.
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{mk(p1, 5, 0.5, 5)}))

	var rawCount uint64
	require.NoError(t, conn.QueryRow(ctx, "SELECT count() FROM metrics_raw").Scan(&rawCount))
	require.Equal(t, uint64(2), rawCount, "duplicate bucket must be dropped")

	var nq, qtSum float64
	rowsRead := map[string]float64{}
	require.NoError(t, conn.QueryRow(
		ctx,
		"SELECT sum(num_queries), sum(m_query_time_sum), sumMap(m_sum) FROM metrics_1h GROUP BY queryid",
	).Scan(&nq, &qtSum, &rowsRead))
	require.InDelta(t, float64(8), nq, 1e-9)
	require.InDelta(t, 0.8, qtSum, 1e-6)
	require.InDelta(t, 14.0, rowsRead["rows_read"], 1e-6) // 7 + 7
}
