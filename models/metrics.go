// qan-api2
// Copyright (C) 2019 Percona LLC
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
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"

	"github.com/percona/pmm/api/qanpb"
)

// Metrics represents methods to work with metrics.
type Metrics struct {
	db *sqlx.DB
}

// NewMetrics initialize Metrics with db instance.
func NewMetrics(db *sqlx.DB) Metrics {
	return Metrics{db: db}
}

// Get select metrics for specific queryid, hostname, etc.
func (m *Metrics) Get(ctx context.Context, from, to time.Time, filter, group string, dbServers, dbSchemas, dbUsernames,
	clientHosts []string, dbLabels map[string][]string) ([]M, error) {
	arg := map[string]interface{}{
		"from":    from,
		"to":      to,
		"filter":  filter,
		"group":   group,
		"servers": dbServers,
		"schemas": dbSchemas,
		"users":   dbUsernames,
		"hosts":   clientHosts,
		"labels":  dbLabels,
	}
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryMetricsTmpl").Funcs(funcMap).Parse(queryMetricsTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	var results []M
	// set mapper to reuse json tags. Have to be unset
	m.db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	defer func() { m.db.Mapper = reflectx.NewMapperFunc("db", strings.ToLower) }()
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return results, fmt.Errorf("prepare named:%v", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return results, fmt.Errorf("populate agruments in IN clause:%v", err)
	}
	query = m.db.Rebind(query)

	rows, err := m.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return results, fmt.Errorf("QueryxContext error:%v", err)
	}
	for rows.Next() {
		result := make(M)
		err = rows.MapScan(result)
		if err != nil {
			fmt.Printf("DimensionMetrics Scan error: %v", err)
		}
		results = append(results, result)
	}
	rows.NextResultSet()
	total := make(M)
	for rows.Next() {
		err = rows.MapScan(total)
		if err != nil {
			fmt.Printf("DimensionMetrics Scan TOTALS error: %v", err)
		}
		results = append(results, total)
	}

	return results, err
}

const queryMetricsTmpl = `
SELECT

SUM(num_queries) AS num_queries,

SUM(m_query_time_cnt) AS m_query_time_cnt,
SUM(m_query_time_sum) AS m_query_time_sum,
MIN(m_query_time_min) AS m_query_time_min,
MAX(m_query_time_max) AS m_query_time_max,
AVG(m_query_time_p99) AS m_query_time_p99,

SUM(m_lock_time_cnt) AS m_lock_time_cnt,
SUM(m_lock_time_sum) AS m_lock_time_sum,
MIN(m_lock_time_min) AS m_lock_time_min,
MAX(m_lock_time_max) AS m_lock_time_max,
AVG(m_lock_time_p99) AS m_lock_time_p99,

SUM(m_rows_sent_cnt) AS m_rows_sent_cnt,
SUM(m_rows_sent_sum) AS m_rows_sent_sum,
MIN(m_rows_sent_min) AS m_rows_sent_min,
MAX(m_rows_sent_max) AS m_rows_sent_max,
AVG(m_rows_sent_p99) AS m_rows_sent_p99,

SUM(m_rows_examined_cnt) AS m_rows_examined_cnt,
SUM(m_rows_examined_sum) AS m_rows_examined_sum,
MIN(m_rows_examined_min) AS m_rows_examined_min,
MAX(m_rows_examined_max) AS m_rows_examined_max,
AVG(m_rows_examined_p99) AS m_rows_examined_p99,

SUM(m_rows_affected_cnt) AS m_rows_affected_cnt,
SUM(m_rows_affected_sum) AS m_rows_affected_sum,
MIN(m_rows_affected_min) AS m_rows_affected_min,
MAX(m_rows_affected_max) AS m_rows_affected_max,
AVG(m_rows_affected_p99) AS m_rows_affected_p99,

SUM(m_rows_read_cnt) AS m_rows_read_cnt,
SUM(m_rows_read_sum) AS m_rows_read_sum,
MIN(m_rows_read_min) AS m_rows_read_min,
MAX(m_rows_read_max) AS m_rows_read_max,
AVG(m_rows_read_p99) AS m_rows_read_p99,

SUM(m_merge_passes_cnt) AS m_merge_passes_cnt,
SUM(m_merge_passes_sum) AS m_merge_passes_sum,
MIN(m_merge_passes_min) AS m_merge_passes_min,
MAX(m_merge_passes_max) AS m_merge_passes_max,
AVG(m_merge_passes_p99) AS m_merge_passes_p99,

SUM(m_innodb_io_r_ops_cnt) AS m_innodb_io_r_ops_cnt,
SUM(m_innodb_io_r_ops_sum) AS m_innodb_io_r_ops_sum,
MIN(m_innodb_io_r_ops_min) AS m_innodb_io_r_ops_min,
MAX(m_innodb_io_r_ops_max) AS m_innodb_io_r_ops_max,
AVG(m_innodb_io_r_ops_p99) AS m_innodb_io_r_ops_p99,

SUM(m_innodb_io_r_bytes_cnt) AS m_innodb_io_r_bytes_cnt,
SUM(m_innodb_io_r_bytes_sum) AS m_innodb_io_r_bytes_sum,
MIN(m_innodb_io_r_bytes_min) AS m_innodb_io_r_bytes_min,
MAX(m_innodb_io_r_bytes_max) AS m_innodb_io_r_bytes_max,
AVG(m_innodb_io_r_bytes_p99) AS m_innodb_io_r_bytes_p99,

SUM(m_innodb_io_r_wait_cnt) AS m_innodb_io_r_wait_cnt,
SUM(m_innodb_io_r_wait_sum) AS m_innodb_io_r_wait_sum,
MIN(m_innodb_io_r_wait_min) AS m_innodb_io_r_wait_min,
MAX(m_innodb_io_r_wait_max) AS m_innodb_io_r_wait_max,
AVG(m_innodb_io_r_wait_p99) AS m_innodb_io_r_wait_p99,

SUM(m_innodb_rec_lock_wait_cnt) AS m_innodb_rec_lock_wait_cnt,
SUM(m_innodb_rec_lock_wait_sum) AS m_innodb_rec_lock_wait_sum,
MIN(m_innodb_rec_lock_wait_min) AS m_innodb_rec_lock_wait_min,
MAX(m_innodb_rec_lock_wait_max) AS m_innodb_rec_lock_wait_max,
AVG(m_innodb_rec_lock_wait_p99) AS m_innodb_rec_lock_wait_p99,

SUM(m_innodb_queue_wait_cnt) AS m_innodb_queue_wait_cnt,
SUM(m_innodb_queue_wait_sum) AS m_innodb_queue_wait_sum,
MIN(m_innodb_queue_wait_min) AS m_innodb_queue_wait_min,
MAX(m_innodb_queue_wait_max) AS m_innodb_queue_wait_max,
AVG(m_innodb_queue_wait_p99) AS m_innodb_queue_wait_p99,

SUM(m_innodb_pages_distinct_cnt) AS m_innodb_pages_distinct_cnt,
SUM(m_innodb_pages_distinct_sum) AS m_innodb_pages_distinct_sum,
MIN(m_innodb_pages_distinct_min) AS m_innodb_pages_distinct_min,
MAX(m_innodb_pages_distinct_max) AS m_innodb_pages_distinct_max,
AVG(m_innodb_pages_distinct_p99) AS m_innodb_pages_distinct_p99,

SUM(m_query_length_cnt) AS m_query_length_cnt,
SUM(m_query_length_sum) AS m_query_length_sum,
MIN(m_query_length_min) AS m_query_length_min,
MAX(m_query_length_max) AS m_query_length_max,
AVG(m_query_length_p99) AS m_query_length_p99,

SUM(m_bytes_sent_cnt) AS m_bytes_sent_cnt,
SUM(m_bytes_sent_sum) AS m_bytes_sent_sum,
MIN(m_bytes_sent_min) AS m_bytes_sent_min,
MAX(m_bytes_sent_max) AS m_bytes_sent_max,
AVG(m_bytes_sent_p99) AS m_bytes_sent_p99,

SUM(m_tmp_tables_cnt) AS m_tmp_tables_cnt,
SUM(m_tmp_tables_sum) AS m_tmp_tables_sum,
MIN(m_tmp_tables_min) AS m_tmp_tables_min,
MAX(m_tmp_tables_max) AS m_tmp_tables_max,
AVG(m_tmp_tables_p99) AS m_tmp_tables_p99,

SUM(m_tmp_disk_tables_cnt) AS m_tmp_disk_tables_cnt,
SUM(m_tmp_disk_tables_sum) AS m_tmp_disk_tables_sum,
MIN(m_tmp_disk_tables_min) AS m_tmp_disk_tables_min,
MAX(m_tmp_disk_tables_max) AS m_tmp_disk_tables_max,
AVG(m_tmp_disk_tables_p99) AS m_tmp_disk_tables_p99,

SUM(m_tmp_table_sizes_cnt) AS m_tmp_table_sizes_cnt,
SUM(m_tmp_table_sizes_sum) AS m_tmp_table_sizes_sum,
MIN(m_tmp_table_sizes_min) AS m_tmp_table_sizes_min,
MAX(m_tmp_table_sizes_max) AS m_tmp_table_sizes_max,
AVG(m_tmp_table_sizes_p99) AS m_tmp_table_sizes_p99,

SUM(m_qc_hit_sum) AS m_qc_hit_sum,
SUM(m_full_scan_sum) AS m_full_scan_sum,
SUM(m_full_join_sum) AS m_full_join_sum,
SUM(m_tmp_table_sum) AS m_tmp_table_sum,
SUM(m_tmp_table_on_disk_sum) AS m_tmp_table_on_disk_sum,
SUM(m_filesort_sum) AS m_filesort_sum,
SUM(m_filesort_on_disk_sum) AS m_filesort_on_disk_sum,
SUM(m_select_full_range_join_sum) AS m_select_full_range_join_sum,
SUM(m_select_range_sum) AS m_select_range_sum,
SUM(m_select_range_check_sum) AS m_select_range_check_sum,
SUM(m_sort_range_sum) AS m_sort_range_sum,
SUM(m_sort_rows_sum) AS m_sort_rows_sum,
SUM(m_sort_scan_sum) AS m_sort_scan_sum,
SUM(m_no_index_used_sum) AS m_no_index_used_sum,
SUM(m_no_good_index_used_sum) AS m_no_good_index_used_sum,

SUM(m_docs_returned_cnt) AS m_docs_returned_cnt,
SUM(m_docs_returned_sum) AS m_docs_returned_sum,
MIN(m_docs_returned_min) AS m_docs_returned_min,
MAX(m_docs_returned_max) AS m_docs_returned_max,
AVG(m_docs_returned_p99) AS m_docs_returned_p99,

SUM(m_response_length_cnt) AS m_response_length_cnt,
SUM(m_response_length_sum) AS m_response_length_sum,
MIN(m_response_length_min) AS m_response_length_min,
MAX(m_response_length_max) AS m_response_length_max,
AVG(m_response_length_p99) AS m_response_length_p99,

SUM(m_docs_scanned_cnt) AS m_docs_scanned_cnt,
SUM(m_docs_scanned_sum) AS m_docs_scanned_sum,
MIN(m_docs_scanned_min) AS m_docs_scanned_min,
MAX(m_docs_scanned_max) AS m_docs_scanned_max,
AVG(m_docs_scanned_p99) AS m_docs_scanned_p99

FROM metrics
WHERE period_start >= :from AND period_start <= :to
{{ if index . "filter" }} AND {{ index . "group" }} = :filter {{ end }}
{{ if index . "servers" }} AND db_server IN ( :servers ) {{ end }}
{{ if index . "schemas" }} AND db_schema IN ( :schemas ) {{ end }}
{{ if index . "users" }} AND db_username IN ( :users ) {{ end }}
{{ if index . "hosts" }} AND client_host IN ( :hosts ) {{ end }}
{{ if index . "labels" }}
	AND (
		{{$i := 0}}
		{{range $key, $val := index . "labels"}}
			{{ $i = inc $i}} {{ if gt $i 1}} OR  {{ end }}
				labels.value[indexOf(labels.key, :{{ $key }})] = :{{ $val }}
		{{ end }}
	)
{{ end }}
GROUP BY {{ index . "group" }}
	WITH TOTALS;
`

const queryExampleTmpl = `
SELECT example, toUInt8(example_format) AS example_format,
       is_truncated, toUInt8(example_type) AS example_type, example_metrics
  FROM metrics
 WHERE period_start >= :from AND period_start <= :to
  	   {{ if index . "filter" }} AND {{ index . "group" }} = :filter {{ end }}
 LIMIT :limit
`

// SelectQueryExamples selects query examples and related stuff for given time range.
func (m *Metrics) SelectQueryExamples(ctx context.Context, from, to time.Time, filter,
	group string, limit uint32) (*qanpb.QueryExampleReply, error) {
	arg := map[string]interface{}{
		"from":   from,
		"to":     to,
		"filter": filter,
		"group":  group,
		"limit":  limit,
	}
	var queryBuffer bytes.Buffer
	if tmpl, err := template.New("queryExampleTmpl").Funcs(funcMap).Parse(queryExampleTmpl); err != nil {
		log.Fatalln(err)
	} else if err = tmpl.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	res := qanpb.QueryExampleReply{}
	// set mapper to reuse json tags. Have to be unset
	m.db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	defer func() { m.db.Mapper = reflectx.NewMapperFunc("db", strings.ToLower) }()
	nstmt, err := m.db.PrepareNamed(queryBuffer.String())
	if err != nil {
		return &res, fmt.Errorf("cannot prepare named statement of select query examples:%v", err)
	}
	err = nstmt.SelectContext(ctx, &res.QueryExamples, arg)
	if err != nil {
		return &res, fmt.Errorf("cannot select query examples:%v", err)
	}
	return &res, nil
}

const queryObjectDetailsLabelsTmpl = `
	SELECT d_server, d_database, d_schema, d_username, d_client_host, labels.key AS lkey, labels.value AS lvalue
	  FROM metrics
	 ARRAY JOIN labels
	 WHERE period_start >= :from AND period_start <= :to
	 {{ if index . "filter" }} AND {{ index . "group" }} = :filter {{ end }}
	 ORDER BY d_server, d_database, d_schema, d_username, d_client_host, labels.key, labels.value
`

var tmplObjectDetailsLabels = template.Must(template.New("queryObjectDetailsLabelsTmpl").Funcs(funcMap).Parse(queryObjectDetailsLabelsTmpl))

// SelectObjectDetailsLabels selects object details labels for given time range and object.
func (m *Metrics) SelectObjectDetailsLabels(ctx context.Context, from, to time.Time, filter,
	group string) (*qanpb.ObjectDetailsLabelsReply, error) {
	arg := map[string]interface{}{
		"from":   from,
		"to":     to,
		"filter": filter,
		"group":  group,
	}
	var queryBuffer bytes.Buffer
	if err := tmplObjectDetailsLabels.Execute(&queryBuffer, arg); err != nil {
		log.Fatalln(err)
	}
	res := qanpb.ObjectDetailsLabelsReply{}
	nstmt, err := m.db.PrepareNamed(queryBuffer.String())
	if err != nil {
		return nil, errors.Wrap(err, "cannot prepare named statement of select object details labels")
	}
	type queryRowsLabels struct {
		DServer     string `db:"d_server"`
		DDatabase   string `db:"d_database"`
		DSchema     string `db:"d_schema"`
		DClientHost string `db:"d_client_host"`
		DUsername   string `db:"d_username"`
		LabelKey    string `db:"lkey"`
		LabelValue  string `db:"lvalue"`
	}
	rows := []queryRowsLabels{}

	err = nstmt.SelectContext(ctx, &rows, arg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot select object details labels")
	}
	labels := map[string]map[string]struct{}{}
	labels["d_server"] = map[string]struct{}{}
	labels["d_database"] = map[string]struct{}{}
	labels["d_schema"] = map[string]struct{}{}
	labels["d_client_host"] = map[string]struct{}{}
	labels["d_username"] = map[string]struct{}{}
	// convert rows to array of unique label keys - values.
	for _, row := range rows {
		labels["d_server"][row.DServer] = struct{}{}
		labels["d_database"][row.DDatabase] = struct{}{}
		labels["d_schema"][row.DSchema] = struct{}{}
		labels["d_client_host"][row.DClientHost] = struct{}{}
		labels["d_username"][row.DUsername] = struct{}{}
		if labels[row.LabelKey] == nil {
			labels[row.LabelKey] = map[string]struct{}{}
		}
		labels[row.LabelKey][row.LabelValue] = struct{}{}
	}

	res.Labels = map[string]*qanpb.ListLabelValues{}
	// rearrange labels into gRPC response structure.
	for key, values := range labels {
		if res.Labels[key] == nil {
			res.Labels[key] = &qanpb.ListLabelValues{
				Values: []string{},
			}
		}
		for value := range values {
			res.Labels[key].Values = append(res.Labels[key].Values, value)
		}
		sort.Strings(res.Labels[key].Values)
	}

	return &res, nil
}
