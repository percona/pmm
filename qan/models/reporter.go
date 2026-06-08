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
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Reporter serves QAN read queries against the rollup tables.
type Reporter struct {
	conn driver.Conn
}

// NewReporter returns a Reporter reading through conn.
func NewReporter(conn driver.Conn) *Reporter {
	return &Reporter{conn: conn}
}

// PickTable routes a time range to the coarsest rollup that covers it.
func PickTable(fromSec, toSec int64) string {
	switch span := toSec - fromSec; {
	case span <= 2*3600:
		return "metrics_raw"
	case span <= 30*86400:
		return "metrics_1h"
	default:
		return "metrics_1d"
	}
}

// GroupByColumn maps an allowed group_by dimension to its column expression.
var GroupByColumn = map[string]string{
	"queryid":      "queryid",
	"service_name": "service_name",
	"database":     "`database`",
	"schema":       "`schema`",
	"cmd_type":     "cmd_type",
	"username":     "username",
	"client_host":  "client_host",
}

// filterColumn maps an allowed dimension filter key to its column expression.
var filterColumn = map[string]string{
	"queryid": "queryid", "service_name": "service_name", "service_id": "service_id",
	"database": "`database`", "schema": "`schema`", "cmd_type": "cmd_type",
	"cluster": "cluster", "environment": "environment",
	"replication_set": "replication_set", "node_name": "node_name",
	"username": "username", "client_host": "client_host",
}

// endpointDimensions exist only in metrics_by_endpoint_1h (excluded from the base grain).
var endpointDimensions = map[string]bool{"username": true, "client_host": true}

// needsEndpoint reports whether a query must use the endpoint rollup because it groups
// by or filters on username/client_host. That rollup keeps only query_time + num_queries.
func needsEndpoint(p ReportParams) bool {
	if endpointDimensions[p.GroupBy] {
		return true
	}
	for k := range p.Dimensions {
		if endpointDimensions[k] {
			return true
		}
	}
	return false
}

// endpointAggregates is the reduced projection available on metrics_by_endpoint_1h.
const endpointAggregates = `
	sum(num_queries) AS num_queries,
	sum(m_query_time_sum) AS m_query_time_sum, sum(m_query_time_cnt) AS m_query_time_cnt,
	sumMap(m_query_time_sketch) AS m_query_time_sketch`

// reportTableAndAggregates picks the table and aggregate projection for a query.
func reportTableAndAggregates(p ReportParams) (string, string) {
	if needsEndpoint(p) {
		return "metrics_by_endpoint_1h", endpointAggregates
	}
	return PickTable(p.FromSec, p.ToSec), reportAggregates
}

// ReportMetric describes a metric exposed by the report.
type ReportMetric struct{ IsTime bool }

// ReportMetrics is the set of metric roots the report can return as columns.
var ReportMetrics = map[string]ReportMetric{
	"query_time": {IsTime: true}, "lock_time": {IsTime: true},
	"rows_sent": {}, "rows_examined": {}, "rows_affected": {}, "bytes_sent": {},
}

// reportAggregates is the fixed aggregate projection, valid on metrics_raw and the
// rollups alike (sum/min/max/sumMap over plain values or SimpleAggregateFunction values).
const reportAggregates = `
	sum(num_queries) AS num_queries,
	sum(num_queries_with_errors) AS num_queries_with_errors,
	sum(num_queries_with_warnings) AS num_queries_with_warnings,
	sum(m_query_time_sum) AS m_query_time_sum, sum(m_query_time_cnt) AS m_query_time_cnt,
	min(m_query_time_min) AS m_query_time_min, max(m_query_time_max) AS m_query_time_max,
	sumMap(m_query_time_sketch) AS m_query_time_sketch,
	sum(m_lock_time_sum) AS m_lock_time_sum, sum(m_lock_time_cnt) AS m_lock_time_cnt,
	min(m_lock_time_min) AS m_lock_time_min, max(m_lock_time_max) AS m_lock_time_max,
	sumMap(m_lock_time_sketch) AS m_lock_time_sketch,
	sum(m_rows_sent_sum) AS m_rows_sent_sum, sum(m_rows_sent_cnt) AS m_rows_sent_cnt,
	min(m_rows_sent_min) AS m_rows_sent_min, max(m_rows_sent_max) AS m_rows_sent_max,
	sum(m_rows_examined_sum) AS m_rows_examined_sum, sum(m_rows_examined_cnt) AS m_rows_examined_cnt,
	min(m_rows_examined_min) AS m_rows_examined_min, max(m_rows_examined_max) AS m_rows_examined_max,
	sum(m_rows_affected_sum) AS m_rows_affected_sum, sum(m_rows_affected_cnt) AS m_rows_affected_cnt,
	min(m_rows_affected_min) AS m_rows_affected_min, max(m_rows_affected_max) AS m_rows_affected_max,
	sum(m_bytes_sent_sum) AS m_bytes_sent_sum, sum(m_bytes_sent_cnt) AS m_bytes_sent_cnt,
	min(m_bytes_sent_min) AS m_bytes_sent_min, max(m_bytes_sent_max) AS m_bytes_sent_max`

// ReportRow holds one aggregated report row (a dimension value or the grand total).
type ReportRow struct {
	Dimension              string            `ch:"dimension"`
	Database               string            `ch:"database_name"`
	TotalRows              uint64            `ch:"total_rows"`
	NumQueries             float64           `ch:"num_queries"`
	NumQueriesWithErrors   float64           `ch:"num_queries_with_errors"`
	NumQueriesWithWarnings float64           `ch:"num_queries_with_warnings"`
	QueryTimeSum           float64           `ch:"m_query_time_sum"`
	QueryTimeCnt           uint64            `ch:"m_query_time_cnt"`
	QueryTimeMin           float32           `ch:"m_query_time_min"`
	QueryTimeMax           float32           `ch:"m_query_time_max"`
	QueryTimeSketch        map[uint16]uint64 `ch:"m_query_time_sketch"`
	LockTimeSum            float64           `ch:"m_lock_time_sum"`
	LockTimeCnt            uint64            `ch:"m_lock_time_cnt"`
	LockTimeMin            float32           `ch:"m_lock_time_min"`
	LockTimeMax            float32           `ch:"m_lock_time_max"`
	LockTimeSketch         map[uint16]uint64 `ch:"m_lock_time_sketch"`
	RowsSentSum            float64           `ch:"m_rows_sent_sum"`
	RowsSentCnt            uint64            `ch:"m_rows_sent_cnt"`
	RowsSentMin            float32           `ch:"m_rows_sent_min"`
	RowsSentMax            float32           `ch:"m_rows_sent_max"`
	RowsExaminedSum        float64           `ch:"m_rows_examined_sum"`
	RowsExaminedCnt        uint64            `ch:"m_rows_examined_cnt"`
	RowsExaminedMin        float32           `ch:"m_rows_examined_min"`
	RowsExaminedMax        float32           `ch:"m_rows_examined_max"`
	RowsAffectedSum        float64           `ch:"m_rows_affected_sum"`
	RowsAffectedCnt        uint64            `ch:"m_rows_affected_cnt"`
	RowsAffectedMin        float32           `ch:"m_rows_affected_min"`
	RowsAffectedMax        float32           `ch:"m_rows_affected_max"`
	BytesSentSum           float64           `ch:"m_bytes_sent_sum"`
	BytesSentCnt           uint64            `ch:"m_bytes_sent_cnt"`
	BytesSentMin           float32           `ch:"m_bytes_sent_min"`
	BytesSentMax           float32           `ch:"m_bytes_sent_max"`
}

// Metric returns a metric root's aggregated values from the row.
func (r *ReportRow) Metric(root string) (sum float64, cnt uint64, mn, mx float32, sketch map[uint16]uint64) { //nolint:nonamedreturns
	switch root {
	case "query_time":
		return r.QueryTimeSum, r.QueryTimeCnt, r.QueryTimeMin, r.QueryTimeMax, r.QueryTimeSketch
	case "lock_time":
		return r.LockTimeSum, r.LockTimeCnt, r.LockTimeMin, r.LockTimeMax, r.LockTimeSketch
	case "rows_sent":
		return r.RowsSentSum, r.RowsSentCnt, r.RowsSentMin, r.RowsSentMax, nil
	case "rows_examined":
		return r.RowsExaminedSum, r.RowsExaminedCnt, r.RowsExaminedMin, r.RowsExaminedMax, nil
	case "rows_affected":
		return r.RowsAffectedSum, r.RowsAffectedCnt, r.RowsAffectedMin, r.RowsAffectedMax, nil
	case "bytes_sent":
		return r.BytesSentSum, r.BytesSentCnt, r.BytesSentMin, r.BytesSentMax, nil
	default:
		return 0, 0, 0, 0, nil
	}
}

// ReportParams are the inputs to Report.
type ReportParams struct {
	FromSec, ToSec int64
	GroupBy        string
	Dimensions     map[string][]string
	OrderBy        string
	Offset, Limit  uint32
	Search         string
}

// ReportResult is the outcome of a report query.
type ReportResult struct {
	Total        ReportRow
	Rows         []ReportRow
	Fingerprints map[string]string // queryid -> fingerprint (only when GroupBy == queryid)
}

// Report runs the profile query: a grand-total row, the page of dimension rows, and
// (for queryid grouping) their fingerprints from dim_query.
func (r *Reporter) Report(ctx context.Context, p ReportParams) (*ReportResult, error) {
	groupCol, ok := GroupByColumn[p.GroupBy]
	if !ok {
		return nil, fmt.Errorf("unsupported group_by: %q", p.GroupBy)
	}
	table, aggregates := reportTableAndAggregates(p)

	where, args := r.buildWhere(p)

	if p.Search != "" {
		if p.GroupBy == "queryid" {
			ids, err := r.searchQueryIDs(ctx, p.Search)
			if err != nil {
				return nil, err
			}
			if len(ids) == 0 {
				return &ReportResult{}, nil
			}
			ph := make([]string, len(ids))
			for i, id := range ids {
				ph[i] = "?"
				args = append(args, id)
			}
			where += " AND queryid IN (" + strings.Join(ph, ", ") + ")"
		} else {
			where += " AND positionCaseInsensitiveUTF8(" + groupCol + ", ?) > 0"
			args = append(args, p.Search)
		}
	}

	// Grand total (+ distinct dimension count) in one pass.
	var total ReportRow
	totalsQuery := fmt.Sprintf(
		"SELECT '' AS dimension, '' AS database_name, %s, uniqExact(%s) AS total_rows FROM %s %s",
		aggregates, groupCol, table, where,
	)
	err := r.conn.QueryRow(ctx, totalsQuery, args...).ScanStruct(&total)
	if err != nil {
		return nil, fmt.Errorf("totals query: %w", err)
	}

	// Page of rows.
	rowsQuery := fmt.Sprintf(
		"SELECT %s AS dimension, anyLast(`database`) AS database_name, %s FROM %s %s GROUP BY dimension ORDER BY %s LIMIT %d, %d",
		groupCol, aggregates, table, where, reportOrderExpr(p.OrderBy, needsEndpoint(p)), p.Offset, p.Limit,
	)
	rows, err := r.conn.Query(ctx, rowsQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("rows query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	res := &ReportResult{Total: total}
	for rows.Next() {
		var row ReportRow
		err = rows.ScanStruct(&row)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		res.Rows = append(res.Rows, row)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if p.GroupBy == "queryid" && len(res.Rows) > 0 {
		res.Fingerprints, err = r.fingerprints(ctx, res.Rows)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// buildWhere returns the shared WHERE clause and positional args (time range + dimension filters).
func (r *Reporter) buildWhere(p ReportParams) (string, []any) {
	conds := []string{"period_start >= ? AND period_start <= ?"}
	args := []any{time.Unix(p.FromSec, 0).UTC(), time.Unix(p.ToSec, 0).UTC()}
	for key, vals := range p.Dimensions {
		if len(vals) == 0 {
			continue
		}
		if col, ok := filterColumn[key]; ok {
			ph := make([]string, len(vals))
			for i, v := range vals {
				ph[i] = "?"
				args = append(args, v)
			}
			conds = append(conds, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ", ")))
			continue
		}
		// Non-standard key: filter on the custom labels Map column, labels[key] IN (vals).
		args = append(args, key)
		ph := make([]string, len(vals))
		for i, v := range vals {
			ph[i] = "?"
			args = append(args, v)
		}
		conds = append(conds, fmt.Sprintf("labels[?] IN (%s)", strings.Join(ph, ", ")))
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

// reportOrderExpr maps an order_by request to a safe ORDER BY expression.
func reportOrderExpr(orderBy string, endpoint bool) string {
	dir := "ASC"
	col := orderBy
	if after, found := strings.CutPrefix(col, "-"); found {
		col, dir = after, "DESC"
	}
	if endpoint {
		switch col {
		case "count", "num_queries":
			return "num_queries " + dir
		case "query_time":
			return "m_query_time_sum / nullIf(m_query_time_cnt, 0) " + dir
		default:
			return "m_query_time_sum DESC"
		}
	}
	switch col {
	case "", "load":
		return "m_query_time_sum DESC"
	case "count", "num_queries":
		return "num_queries " + dir
	case "query_time", "lock_time":
		return fmt.Sprintf("m_%s_sum / nullIf(m_%s_cnt, 0) %s", col, col, dir)
	case "rows_sent", "rows_examined", "rows_affected", "bytes_sent":
		return fmt.Sprintf("m_%s_sum %s", col, dir)
	default:
		return "m_query_time_sum DESC"
	}
}

func (r *Reporter) fingerprints(ctx context.Context, rows []ReportRow) (map[string]string, error) {
	ph := make([]string, len(rows))
	args := make([]any, len(rows))
	for i, row := range rows {
		ph[i] = "?"
		args[i] = row.Dimension
	}
	query := fmt.Sprintf("SELECT queryid, anyLast(fingerprint) FROM dim_query WHERE queryid IN (%s) GROUP BY queryid", strings.Join(ph, ", "))
	res, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fingerprints query: %w", err)
	}
	defer res.Close() //nolint:errcheck

	out := make(map[string]string, len(rows))
	for res.Next() {
		var id, fp string
		err = res.Scan(&id, &fp)
		if err != nil {
			return nil, err
		}
		out[id] = fp
	}
	return out, res.Err()
}

// searchQueryIDs returns queryids whose fingerprint matches the search term (case-insensitive).
func (r *Reporter) searchQueryIDs(ctx context.Context, search string) ([]string, error) {
	rows, err := r.conn.Query(ctx,
		"SELECT DISTINCT queryid FROM dim_query WHERE positionCaseInsensitiveUTF8(fingerprint, ?) > 0 LIMIT 1000", search)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ids []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FilterValue is a dimension value with its share of the main metric.
type FilterValue struct {
	Value   string
	Percent float32
	PerSec  float32
}

// Filters returns, per filterable dimension, its values weighted by num_queries —
// read from the precomputed dim_values table (never scanning the fact tables).
func (r *Reporter) Filters(ctx context.Context, fromSec, toSec int64) (map[string][]FilterValue, error) {
	durSec := float32(toSec - fromSec)
	if durSec <= 0 {
		durSec = 1
	}
	rows, err := r.conn.Query(ctx,
		"SELECT dimension, value, sum(weight) AS w FROM dim_values WHERE period_start >= ? AND period_start <= ? GROUP BY dimension, value ORDER BY dimension, w DESC",
		time.Unix(fromSec, 0).UTC(), time.Unix(toSec, 0).UTC())
	if err != nil {
		return nil, fmt.Errorf("filters query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	out := make(map[string][]FilterValue)
	totals := make(map[string]float64)
	for rows.Next() {
		var dim, val string
		var w float64
		err = rows.Scan(&dim, &val, &w)
		if err != nil {
			return nil, err
		}
		out[dim] = append(out[dim], FilterValue{Value: val, PerSec: float32(w) / durSec})
		totals[dim] += w
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	for dim, vals := range out {
		if totals[dim] == 0 {
			continue
		}
		for i := range vals {
			vals[i].Percent = float32(float64(vals[i].PerSec) * float64(durSec) / totals[dim] * 100)
		}
	}
	return out, nil
}

// QueryExampleRow is one stored example for a query.
type QueryExampleRow struct {
	Example     string `ch:"example"`
	ExampleType string `ch:"example_type"`
	IsTruncated uint8  `ch:"is_truncated"`
	QueryPlan   string `ch:"query_plan"`
}

// QueryExamples returns recent stored examples for a queryid.
func (r *Reporter) QueryExamples(ctx context.Context, queryid string, fromSec, toSec int64, limit uint32) ([]QueryExampleRow, error) {
	if limit == 0 {
		limit = 5
	}
	rows, err := r.conn.Query(ctx,
		fmt.Sprintf("SELECT example, example_type, is_truncated, query_plan FROM query_examples WHERE queryid = ? AND period_start >= ? AND period_start <= ? ORDER BY period_start DESC LIMIT %d", limit), //nolint:lll
		queryid, time.Unix(fromSec, 0).UTC(), time.Unix(toSec, 0).UTC())
	if err != nil {
		return nil, fmt.Errorf("examples query: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var out []QueryExampleRow
	for rows.Next() {
		var e QueryExampleRow
		err = rows.ScanStruct(&e)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
