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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/utils/ddsketch"
)

func TestObjectDetailsIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_odtest")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)

	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())
	bucket := func(qid string, nq, qtSum, qtCnt float32) *qanv1.MetricsBucket {
		return &qanv1.MetricsBucket{
			Queryid: qid, ServiceId: "svc1", ServiceName: "mysql-1", Database: "db1", Schema: "public",
			CmdType: "SELECT", Fingerprint: "SELECT 1", ExplainFingerprint: "SELECT ?", PlaceholdersCount: 1,
			Example: "SELECT 1", QueryPlan: "Seq Scan", Planid: "plan123",
			PeriodStartUnixSecs: base, NumQueries: nq,
			MQueryTimeSum: qtSum, MQueryTimeCnt: qtCnt, MQueryTimeMin: 0.01, MQueryTimeMax: 0.2,
			MRowsSentSum: 10, MRowsSentCnt: qtCnt,
		}
	}
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{bucket("q1", 5, 0.5, 5), bucket("q2", 3, 0.9, 3)}))

	// Direct-insert a sketch-only row for q1 (Phase A ingestion sends empty sketches).
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
	// Carry the same identity as the ingested bucket: every raw row for a queryid has
	// the same fingerprint (queryid is its hash), so dim_query stays consistent.
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO metrics_raw (queryid, service_id, service_name, `database`, `schema`, cmd_type, fingerprint, explain_fingerprint, placeholders_count, period_start, num_queries, m_query_time_sum, m_query_time_cnt, m_query_time_sketch)")
	require.NoError(t, err)
	require.NoError(t, batch.Append("q1", "svc1", "mysql-1", "db1", "public", "SELECT", "SELECT 1", "SELECT ?", uint32(1), time.Unix(int64(base), 0).UTC(), float64(0), float64(0), uint64(0), sketch))
	require.NoError(t, batch.Send())

	from, to := int64(base)-3600, int64(base)+3600

	t.Run("Metrics", func(t *testing.T) {
		value, total, err := rep.Metrics(ctx, ReportParams{FromSec: from, ToSec: to, GroupBy: "queryid"}, "q1")
		require.NoError(t, err)
		require.InDelta(t, 0.5, value.QueryTimeSum, 1e-6) // q1 only
		require.InDelta(t, 5, value.NumQueries, 1e-6)
		require.InDelta(t, 1.4, total.QueryTimeSum, 1e-6) // q1 + q2
		require.InDelta(t, 8, total.NumQueries, 1e-6)
	})

	t.Run("Histogram", func(t *testing.T) {
		got, err := rep.Histogram(ctx, "q1", from, to)
		require.NoError(t, err)
		require.NotEmpty(t, got)
		require.InDelta(t, 0.989, ddsketch.QuantileFromMap(got, 0.99), 0.989*ddsketch.Alpha+1e-9)
	})

	t.Run("QueryExists", func(t *testing.T) {
		exists, err := rep.QueryExists(ctx, "q1")
		require.NoError(t, err)
		require.True(t, exists)
		exists, err = rep.QueryExists(ctx, "missing")
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("Identity", func(t *testing.T) {
		fp, err := rep.Fingerprint(ctx, "q1")
		require.NoError(t, err)
		require.Equal(t, "SELECT 1", fp)
		efp, placeholders, err := rep.ExplainFingerprint(ctx, "q1")
		require.NoError(t, err)
		require.Equal(t, "SELECT ?", efp)
		require.Equal(t, uint32(1), placeholders)
		schema, err := rep.SchemaForQuery(ctx, "svc1", "q1")
		require.NoError(t, err)
		require.Equal(t, "public", schema)
		planid, plan, err := rep.QueryPlan(ctx, "q1")
		require.NoError(t, err)
		require.Equal(t, "plan123", planid)
		require.Equal(t, "Seq Scan", plan)
	})

	t.Run("LabelsAndMetadata", func(t *testing.T) {
		labels, err := rep.LabelsForQuery(ctx, "queryid", "q1", from, to)
		require.NoError(t, err)
		assert.Contains(t, labels["service_name"], "mysql-1")
		assert.Contains(t, labels["cmd_type"], "SELECT")

		meta, err := rep.Metadata(ctx, "queryid", "q1", from, to)
		require.NoError(t, err)
		assert.Equal(t, "mysql-1", meta.ServiceName)
		assert.Equal(t, "svc1", meta.ServiceID)
		assert.Equal(t, "public", meta.Schema)
	})
}
