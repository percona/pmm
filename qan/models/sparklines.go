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
	"time"

	qanv1 "github.com/percona/pmm/api/qan/v1"
)

const (
	sparklinePoints = 120
	sparklineMinTfS = 60
)

// Sparklines returns a per-second time series over the period for one dimension
// value (filterBy), or the grand total when filterBy is empty. Points are bucketed
// into ~sparklinePoints intervals (minimum 1 minute).
func (r *Reporter) Sparklines(ctx context.Context, p ReportParams, groupBy, filterBy string) ([]*qanv1.Point, error) {
	tf := max((p.ToSec-p.FromSec)/sparklinePoints, sparklineMinTfS)

	dims := p.Dimensions
	if filterBy != "" {
		dims = addDimension(p.Dimensions, groupBy, filterBy)
	}
	sp := ReportParams{FromSec: p.FromSec, ToSec: p.ToSec, GroupBy: groupBy, Dimensions: dims}
	endpoint := needsEndpoint(sp)
	table := PickTable(p.FromSec, p.ToSec)
	if endpoint {
		table = "metrics_by_endpoint_1h"
	}
	where, args := r.buildWhere(sp)

	// metrics_by_endpoint_1h only has query_time + num_queries; zero the rest there
	// so the scan shape stays fixed.
	zero := "toFloat64(0)"
	nqe, nqw, lt, rs, re, ra, bs := zero, zero, zero, zero, zero, zero, zero
	if !endpoint {
		nqe = fmt.Sprintf("sum(num_queries_with_errors) / %d", tf)
		nqw = fmt.Sprintf("sum(num_queries_with_warnings) / %d", tf)
		lt = fmt.Sprintf("sum(m_lock_time_sum) / %d", tf)
		rs = fmt.Sprintf("sum(m_rows_sent_sum) / %d", tf)
		re = fmt.Sprintf("sum(m_rows_examined_sum) / %d", tf)
		ra = fmt.Sprintf("sum(m_rows_affected_sum) / %d", tf)
		bs = fmt.Sprintf("sum(m_bytes_sent_sum) / %d", tf)
	}
	query := fmt.Sprintf(
		"SELECT toUInt32(intDivOrZero(%d - toUnixTimestamp(period_start), %d)) AS point, "+
			"sum(m_query_time_sum) / %d AS load, sum(num_queries) / %d, %s, %s, %s, %s, %s, %s, %s "+
			"FROM %s %s GROUP BY point ORDER BY point ASC",
		p.ToSec, tf, tf, tf, nqe, nqw, lt, rs, re, ra, bs, table, where,
	)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sparkline query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var out []*qanv1.Point
	for rows.Next() {
		var point uint32
		var load, nqps, nqe, nqw, lt, rs, re, ra, bs float64
		err = rows.Scan(&point, &load, &nqps, &nqe, &nqw, &lt, &rs, &re, &ra, &bs)
		if err != nil {
			return nil, err
		}
		out = append(out, &qanv1.Point{
			Point:                        point,
			TimeFrame:                    uint32(tf),
			Timestamp:                    time.Unix(p.ToSec-int64(point)*tf, 0).UTC().Format(time.RFC3339),
			Load:                         float32(load),
			NumQueriesPerSec:             float32(nqps),
			NumQueriesWithErrorsPerSec:   float32(nqe),
			NumQueriesWithWarningsPerSec: float32(nqw),
			MQueryTimeSumPerSec:          float32(load),
			MLockTimeSumPerSec:           float32(lt),
			MRowsSentSumPerSec:           float32(rs),
			MRowsExaminedSumPerSec:       float32(re),
			MRowsAffectedSumPerSec:       float32(ra),
			MBytesSentSumPerSec:          float32(bs),
		})
	}
	return out, rows.Err()
}
