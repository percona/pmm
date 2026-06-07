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

	"github.com/stretchr/testify/require"

	qanv1 "github.com/percona/pmm/api/qan/v1"
)

func TestSparklinesIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_spark")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())

	mk := func(period uint32, nq, qtSum float32) *qanv1.MetricsBucket {
		return &qanv1.MetricsBucket{
			Queryid: "q1", ServiceId: "svc1", Database: "db1", Schema: "public", CmdType: "SELECT",
			PeriodStartUnixSecs: period, NumQueries: nq, MQueryTimeSum: qtSum, MQueryTimeCnt: nq,
		}
	}
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{mk(base, 5, 0.5), mk(base+600, 3, 0.3)}))

	params := ReportParams{FromSec: int64(base) - 3600, ToSec: int64(base) + 3600, GroupBy: "queryid"}
	points, err := rep.Sparklines(ctx, params, "queryid", "q1")
	require.NoError(t, err)
	require.NotEmpty(t, points)

	var totalLoad float32
	for _, pt := range points {
		require.NotZero(t, pt.TimeFrame)
		require.NotEmpty(t, pt.Timestamp)
		totalLoad += pt.Load
	}
	require.Greater(t, totalLoad, float32(0))
}

func TestEndpointDrilldownIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_endpoint")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())

	mk := func(user string, nq, qtSum float32) *qanv1.MetricsBucket {
		return &qanv1.MetricsBucket{
			Queryid: "q1", ServiceId: "svc1", Database: "db1", Schema: "public", CmdType: "SELECT",
			Username: user, ClientHost: "host1", PeriodStartUnixSecs: base,
			NumQueries: nq, MQueryTimeSum: qtSum, MQueryTimeCnt: nq,
		}
	}
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{mk("alice", 5, 0.5), mk("bob", 3, 0.3)}))

	from, to := int64(base)-3600, int64(base)+3600
	res, err := rep.Report(ctx, ReportParams{FromSec: from, ToSec: to, GroupBy: "username", OrderBy: "-load", Limit: 10})
	require.NoError(t, err)
	require.Len(t, res.Rows, 2)
	require.Equal(t, "alice", res.Rows[0].Dimension) // ordered by load desc
	require.InDelta(t, 0.5, res.Rows[0].QueryTimeSum, 1e-6)
	require.InDelta(t, 5, res.Rows[0].NumQueries, 1e-6)
	require.Zero(t, res.Rows[0].LockTimeSum) // not available in the endpoint rollup
	require.InDelta(t, 0.8, res.Total.QueryTimeSum, 1e-6)
	require.Equal(t, uint64(2), res.Total.TotalRows)
}

func TestSearchIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_search")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())

	mk := func(qid, fp string) *qanv1.MetricsBucket {
		return &qanv1.MetricsBucket{
			Queryid: qid, ServiceId: "svc1", Database: "db1", Schema: "public", CmdType: "SELECT",
			Fingerprint: fp, PeriodStartUnixSecs: base, NumQueries: 5, MQueryTimeSum: 0.5, MQueryTimeCnt: 5,
		}
	}
	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{
		mk("q1", "SELECT * FROM users"),
		mk("q2", "DELETE FROM orders"),
	}))

	from, to := int64(base)-3600, int64(base)+3600

	res, err := rep.Report(ctx, ReportParams{FromSec: from, ToSec: to, GroupBy: "queryid", Limit: 10, Search: "USERS"})
	require.NoError(t, err)
	require.Len(t, res.Rows, 1, "case-insensitive fingerprint search should match one query")
	require.Equal(t, "q1", res.Rows[0].Dimension)

	res, err = rep.Report(ctx, ReportParams{FromSec: from, ToSec: to, GroupBy: "queryid", Limit: 10, Search: "zzznomatch"})
	require.NoError(t, err)
	require.Empty(t, res.Rows, "no fingerprint matches the search")
}

func TestLabelFilterIntegration(t *testing.T) {
	conn := setupTestDB(t, "qan_lblfilter")
	ctx := t.Context()
	ing := NewIngestor(conn)
	rep := NewReporter(conn)
	base := uint32(time.Now().UTC().Truncate(time.Hour).Add(-time.Hour).Unix())

	require.NoError(t, ing.Save(ctx, []*qanv1.MetricsBucket{{
		Queryid: "q1", ServiceId: "svc1", Database: "db1", Schema: "public", CmdType: "SELECT",
		PeriodStartUnixSecs: base, NumQueries: 5, MQueryTimeSum: 0.5, MQueryTimeCnt: 5,
		Labels: map[string]string{"env": "prod"},
	}}))

	from, to := int64(base)-3600, int64(base)+3600

	res, err := rep.Report(ctx, ReportParams{
		FromSec: from, ToSec: to, GroupBy: "queryid", Limit: 10,
		Dimensions: map[string][]string{"env": {"prod"}},
	})
	require.NoError(t, err)
	require.Len(t, res.Rows, 1, "custom-label filter should match")

	res, err = rep.Report(ctx, ReportParams{
		FromSec: from, ToSec: to, GroupBy: "queryid", Limit: 10,
		Dimensions: map[string][]string{"env": {"dev"}},
	})
	require.NoError(t, err)
	require.Empty(t, res.Rows, "non-matching custom-label filter should exclude the row")
}
