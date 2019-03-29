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
	"strings"
	"text/template"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
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
func (m *Metrics) Get(ctx context.Context, from, to, digest string, dbServers, dbSchemas, dbUsernames,
	clientHosts []string, dbLabels map[string][]string) (*qanpb.MetricsReply, error) {
	arg := map[string]interface{}{
		"from":    from,
		"to":      to,
		"digest":  digest,
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

	// set mapper to reuse json tags. Have to be unset
	m.db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	defer func() { m.db.Mapper = reflectx.NewMapperFunc("db", strings.ToLower) }()
	res := qanpb.MetricsReply{}
	query, args, err := sqlx.Named(queryBuffer.String(), arg)
	if err != nil {
		return &res, fmt.Errorf("prepare named:%v", err)
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		return &res, fmt.Errorf("populate agruments in IN clause:%v", err)
	}
	query = m.db.Rebind(query)
	err = m.db.GetContext(ctx, &res, query, args...)
	return &res, err
}

const queryMetricsTmpl = `
SELECT
queryid,
any(fingerprint) AS fingerprint,
groupUniqArray(d_server) AS d_servers,
groupUniqArray(d_database) AS d_databases,
groupUniqArray(d_schema) AS d_schemas,
groupUniqArray(d_username) AS d_usernames,
groupUniqArray(d_client_host) AS d_client_hosts,

MIN(period_start) AS first_seen,
MAX(period_start) AS last_seen,

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
SUM(m_no_good_index_used_sum) AS m_no_good_index_used_sum
FROM metrics
WHERE period_start > :from AND period_start < :to AND queryid = :digest
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
GROUP BY queryid;
`
