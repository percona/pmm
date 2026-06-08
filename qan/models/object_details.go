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
	"maps"
	"strings"
	"time"
)

// QueryMetadata holds the descriptive fields shown in the query-details panel.
type QueryMetadata struct {
	ServiceName, ServiceID, ServiceType, Database, Schema string
	Cluster, Environment, ReplicationSet, NodeName        string
}

// Metrics returns the aggregates for a single dimension value plus the grand total
// over the period (for percent-of-total), both as ReportRows.
func (r *Reporter) Metrics(ctx context.Context, p ReportParams, filterBy string) (value, total ReportRow, err error) { //nolint:nonamedreturns
	if _, ok := GroupByColumn[p.GroupBy]; !ok {
		return value, total, fmt.Errorf("unsupported group_by: %q", p.GroupBy)
	}
	table, aggregates := reportTableAndAggregates(p)

	whereTotal, argsTotal := r.buildWhere(p)
	totalQuery := fmt.Sprintf("SELECT '' AS dimension, '' AS database_name, %s FROM %s %s", aggregates, table, whereTotal)
	err = r.conn.QueryRow(ctx, totalQuery, argsTotal...).ScanStruct(&total)
	if err != nil {
		return value, total, fmt.Errorf("metrics total: %w", err)
	}

	pv := p
	pv.Dimensions = addDimension(p.Dimensions, p.GroupBy, filterBy)
	whereVal, argsVal := r.buildWhere(pv)
	valueQuery := fmt.Sprintf("SELECT '' AS dimension, anyLast(`database`) AS database_name, %s FROM %s %s", aggregates, table, whereVal)
	err = r.conn.QueryRow(ctx, valueQuery, argsVal...).ScanStruct(&value)
	if err != nil {
		return value, total, fmt.Errorf("metrics value: %w", err)
	}
	return value, total, nil
}

// addDimension returns a copy of dims with val added under key.
func addDimension(dims map[string][]string, key, val string) map[string][]string {
	out := make(map[string][]string, len(dims)+1)
	maps.Copy(out, dims)
	out[key] = append(append([]string{}, out[key]...), val)
	return out
}

// Metadata returns the descriptive fields for a dimension value (anyLast over the tier).
func (r *Reporter) Metadata(ctx context.Context, groupBy, filterBy string, fromSec, toSec int64) (*QueryMetadata, error) {
	groupCol, ok := GroupByColumn[groupBy]
	if !ok {
		return nil, fmt.Errorf("unsupported group_by: %q", groupBy)
	}
	table := PickTable(fromSec, toSec)
	m := &QueryMetadata{}
	query := fmt.Sprintf("SELECT anyLast(service_name), any(service_id), anyLast(service_type), any(`database`), any(`schema`), "+
		"anyLast(cluster), anyLast(environment), anyLast(replication_set), anyLast(node_name) "+
		"FROM %s WHERE %s = ? AND period_start >= ? AND period_start <= ?", table, groupCol)
	err := r.conn.QueryRow(ctx, query, filterBy, time.Unix(fromSec, 0).UTC(), time.Unix(toSec, 0).UTC()).Scan(
		&m.ServiceName, &m.ServiceID, &m.ServiceType, &m.Database, &m.Schema,
		&m.Cluster, &m.Environment, &m.ReplicationSet, &m.NodeName,
	)
	return m, err
}

// Fingerprint returns the stored fingerprint for a queryid.
func (r *Reporter) Fingerprint(ctx context.Context, queryid string) (string, error) {
	var fp string
	err := r.conn.QueryRow(ctx, "SELECT anyLast(fingerprint) FROM dim_query WHERE queryid = ?", queryid).Scan(&fp)
	return fp, err
}

// ExplainFingerprint returns the stored explain fingerprint and placeholders count for a queryid.
func (r *Reporter) ExplainFingerprint(ctx context.Context, queryid string) (string, uint32, error) {
	var fp string
	var placeholders uint32
	err := r.conn.QueryRow(ctx, "SELECT anyLast(explain_fingerprint), anyLast(placeholders_count) FROM dim_query WHERE queryid = ?", queryid).Scan(&fp, &placeholders)
	return fp, placeholders, err
}

// QueryExists reports whether a queryid has been seen.
func (r *Reporter) QueryExists(ctx context.Context, queryid string) (bool, error) {
	var n uint64
	err := r.conn.QueryRow(ctx, "SELECT count() FROM dim_query WHERE queryid = ?", queryid).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// SchemaForQuery returns the schema a query ran against for a given service.
func (r *Reporter) SchemaForQuery(ctx context.Context, serviceID, queryID string) (string, error) {
	var schema string
	err := r.conn.QueryRow(ctx, "SELECT anyLast(`schema`) FROM metrics_1d WHERE service_id = ? AND queryid = ?", serviceID, queryID).Scan(&schema)
	return schema, err
}

// QueryPlan returns the most recent stored plan id and query plan for a queryid.
func (r *Reporter) QueryPlan(ctx context.Context, queryid string) (planid, plan string, err error) { //nolint:nonamedreturns
	rows, err := r.conn.Query(ctx, "SELECT planid, query_plan FROM query_examples WHERE queryid = ? ORDER BY period_start DESC LIMIT 1", queryid)
	if err != nil {
		return "", "", err
	}
	defer rows.Close() //nolint:errcheck

	if rows.Next() {
		err = rows.Scan(&planid, &plan)
		if err != nil {
			return "", "", err
		}
	}
	return planid, plan, rows.Err()
}

// Histogram returns the merged query_time DDSketch bucket map for a queryid.
func (r *Reporter) Histogram(ctx context.Context, queryid string, fromSec, toSec int64) (map[uint16]uint64, error) {
	table := PickTable(fromSec, toSec)
	sketch := map[uint16]uint64{}
	query := fmt.Sprintf("SELECT sumMap(m_query_time_sketch) FROM %s WHERE queryid = ? AND period_start >= ? AND period_start <= ?", table)
	err := r.conn.QueryRow(ctx, query, queryid, time.Unix(fromSec, 0).UTC(), time.Unix(toSec, 0).UTC()).Scan(&sketch)
	return sketch, err
}

// LabelsForQuery returns the distinct dimension values a query appears with.
func (r *Reporter) LabelsForQuery(ctx context.Context, groupBy, filterBy string, fromSec, toSec int64) (map[string][]string, error) {
	groupCol, ok := GroupByColumn[groupBy]
	if !ok {
		return nil, fmt.Errorf("unsupported group_by: %q", groupBy)
	}
	table := PickTable(fromSec, toSec)
	dims := []string{"service_name", "database", "schema", "cluster", "environment", "replication_set", "node_name", "cmd_type"}
	selects := make([]string, len(dims))
	for i, d := range dims {
		col := d
		if d == "database" || d == "schema" {
			col = "`" + d + "`"
		}
		selects[i] = fmt.Sprintf("groupUniqArray(%s)", col)
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? AND period_start >= ? AND period_start <= ?",
		strings.Join(selects, ", "), table, groupCol)

	arrs := make([][]string, len(dims))
	dest := make([]any, len(dims))
	for i := range arrs {
		dest[i] = &arrs[i]
	}
	err := r.conn.QueryRow(ctx, query, filterBy, time.Unix(fromSec, 0).UTC(), time.Unix(toSec, 0).UTC()).Scan(dest...)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string)
	for i, d := range dims {
		if len(arrs[i]) > 0 {
			out[d] = arrs[i]
		}
	}
	return out, nil
}
