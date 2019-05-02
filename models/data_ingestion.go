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
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/percona/pmm/api/qanpb"
	"github.com/pkg/errors"
)

const insertSQL = `
  INSERT INTO metrics
  (
    queryid,
    d_server,
    d_database,
    d_schema,
    d_username,
    d_client_host,
    replication_set,
    cluster,
    service_type,
    environment,
    az,
    region,
    node_model,
    container_name,
    labels.key,
    labels.value,
    agent_uuid,
    metrics_source,
    period_start,
    period_length,
    fingerprint,
    example,
    example_format,
    is_truncated,
    example_type,
    example_metrics,
    num_queries_with_warnings,
    warnings.code,
    warnings.count,
    num_queries_with_errors,
    errors.code,
    errors.count,
    num_queries,
    m_query_time_cnt,
    m_query_time_sum,
    m_query_time_min,
    m_query_time_max,
    m_query_time_p99,
    m_lock_time_cnt,
    m_lock_time_sum,
    m_lock_time_min,
    m_lock_time_max,
    m_lock_time_p99,
    m_rows_sent_cnt,
    m_rows_sent_sum,
    m_rows_sent_min,
    m_rows_sent_max,
    m_rows_sent_p99,
    m_rows_examined_cnt,
    m_rows_examined_sum,
    m_rows_examined_min,
    m_rows_examined_max,
    m_rows_examined_p99,
    m_rows_affected_cnt,
    m_rows_affected_sum,
    m_rows_affected_min,
    m_rows_affected_max,
    m_rows_affected_p99,
    m_rows_read_cnt,
    m_rows_read_sum,
    m_rows_read_min,
    m_rows_read_max,
    m_rows_read_p99,
    m_merge_passes_cnt,
    m_merge_passes_sum,
    m_merge_passes_min,
    m_merge_passes_max,
    m_merge_passes_p99,
    m_innodb_io_r_ops_cnt,
    m_innodb_io_r_ops_sum,
    m_innodb_io_r_ops_min,
    m_innodb_io_r_ops_max,
    m_innodb_io_r_ops_p99,
    m_innodb_io_r_bytes_cnt,
    m_innodb_io_r_bytes_sum,
    m_innodb_io_r_bytes_min,
    m_innodb_io_r_bytes_max,
    m_innodb_io_r_bytes_p99,
    m_innodb_io_r_wait_cnt,
    m_innodb_io_r_wait_sum,
    m_innodb_io_r_wait_min,
    m_innodb_io_r_wait_max,
    m_innodb_io_r_wait_p99,
    m_innodb_rec_lock_wait_cnt,
    m_innodb_rec_lock_wait_sum,
    m_innodb_rec_lock_wait_min,
    m_innodb_rec_lock_wait_max,
    m_innodb_rec_lock_wait_p99,
    m_innodb_queue_wait_cnt,
    m_innodb_queue_wait_sum,
    m_innodb_queue_wait_min,
    m_innodb_queue_wait_max,
    m_innodb_queue_wait_p99,
    m_innodb_pages_distinct_cnt,
    m_innodb_pages_distinct_sum,
    m_innodb_pages_distinct_min,
    m_innodb_pages_distinct_max,
    m_innodb_pages_distinct_p99,
    m_query_length_cnt,
    m_query_length_sum,
    m_query_length_min,
    m_query_length_max,
    m_query_length_p99,
    m_bytes_sent_cnt,
    m_bytes_sent_sum,
    m_bytes_sent_min,
    m_bytes_sent_max,
    m_bytes_sent_p99,
    m_tmp_tables_cnt,
    m_tmp_tables_sum,
    m_tmp_tables_min,
    m_tmp_tables_max,
    m_tmp_tables_p99,
    m_tmp_disk_tables_cnt,
    m_tmp_disk_tables_sum,
    m_tmp_disk_tables_min,
    m_tmp_disk_tables_max,
    m_tmp_disk_tables_p99,
    m_tmp_table_sizes_cnt,
    m_tmp_table_sizes_sum,
    m_tmp_table_sizes_min,
    m_tmp_table_sizes_max,
    m_tmp_table_sizes_p99,
    m_qc_hit_cnt,
    m_qc_hit_sum,
    m_full_scan_cnt,
    m_full_scan_sum,
    m_full_join_cnt,
    m_full_join_sum,
    m_tmp_table_cnt,
    m_tmp_table_sum,
    m_tmp_table_on_disk_cnt,
    m_tmp_table_on_disk_sum,
    m_filesort_cnt,
    m_filesort_sum,
    m_filesort_on_disk_cnt,
    m_filesort_on_disk_sum,
    m_select_full_range_join_cnt,
    m_select_full_range_join_sum,
    m_select_range_cnt,
    m_select_range_sum,
    m_select_range_check_cnt,
    m_select_range_check_sum,
    m_sort_range_cnt,
    m_sort_range_sum,
    m_sort_rows_cnt,
    m_sort_rows_sum,
    m_sort_scan_cnt,
    m_sort_scan_sum,
    m_no_index_used_cnt,
    m_no_index_used_sum,
    m_no_good_index_used_cnt,
    m_no_good_index_used_sum,
    m_docs_returned_cnt,
    m_docs_returned_sum,
    m_docs_returned_min,
    m_docs_returned_max,
    m_docs_returned_p99,
    m_response_length_cnt,
    m_response_length_sum,
    m_response_length_min,
    m_response_length_max,
    m_response_length_p99,
    m_docs_scanned_cnt,
    m_docs_scanned_sum,
    m_docs_scanned_min,
    m_docs_scanned_max,
    m_docs_scanned_p99
   )
  VALUES (
    :queryid,
    :service_name,
    :d_database,
    :d_schema,
    :d_username,
    :d_client_host,
    :replication_set,
    :cluster,
    :service_type,
    :environment,
    :az,
    :region,
    :node_model,
    :container_name,
    :labels_key,
    :labels_value,
    :agent_id,
    CAST( :metrics_source_s AS Enum8('METRICS_SOURCE_INVALID' = 0, 'MYSQL_SLOWLOG' = 1, 'MYSQL_PERFSCHEMA' = 2, 'MONGODB_PROFILER' = 3)) AS metrics_source,
    :period_start_ts,
    :period_length_secs,
    :fingerprint,
    :example,
    CAST( :example_format_s AS Enum8('EXAMPLE' = 0, 'FINGERPRINT' = 1)) AS example_format,
    :is_query_truncated,
    CAST( :example_type_s AS Enum8('RANDOM' = 0, 'SLOWEST' = 1, 'FASTEST' = 2, 'WITH_ERROR' = 3)) AS example_type,
    :example_metrics,
    :num_queries_with_warnings,
    :warnings_code,
    :warnings_count,
    :num_queries_with_errors,
    :errors_code,
    :errors_count,
    :num_queries,
    :m_query_time_cnt,
    :m_query_time_sum,
    :m_query_time_min,
    :m_query_time_max,
    :m_query_time_p99,
    :m_lock_time_cnt,
    :m_lock_time_sum,
    :m_lock_time_min,
    :m_lock_time_max,
    :m_lock_time_p99,
    :m_rows_sent_cnt,
    :m_rows_sent_sum,
    :m_rows_sent_min,
    :m_rows_sent_max,
    :m_rows_sent_p99,
    :m_rows_examined_cnt,
    :m_rows_examined_sum,
    :m_rows_examined_min,
    :m_rows_examined_max,
    :m_rows_examined_p99,
    :m_rows_affected_cnt,
    :m_rows_affected_sum,
    :m_rows_affected_min,
    :m_rows_affected_max,
    :m_rows_affected_p99,
    :m_rows_read_cnt,
    :m_rows_read_sum,
    :m_rows_read_min,
    :m_rows_read_max,
    :m_rows_read_p99,
    :m_merge_passes_cnt,
    :m_merge_passes_sum,
    :m_merge_passes_min,
    :m_merge_passes_max,
    :m_merge_passes_p99,
    :m_innodb_io_r_ops_cnt,
    :m_innodb_io_r_ops_sum,
    :m_innodb_io_r_ops_min,
    :m_innodb_io_r_ops_max,
    :m_innodb_io_r_ops_p99,
    :m_innodb_io_r_bytes_cnt,
    :m_innodb_io_r_bytes_sum,
    :m_innodb_io_r_bytes_min,
    :m_innodb_io_r_bytes_max,
    :m_innodb_io_r_bytes_p99,
    :m_innodb_io_r_wait_cnt,
    :m_innodb_io_r_wait_sum,
    :m_innodb_io_r_wait_min,
    :m_innodb_io_r_wait_max,
    :m_innodb_io_r_wait_p99,
    :m_innodb_rec_lock_wait_cnt,
    :m_innodb_rec_lock_wait_sum,
    :m_innodb_rec_lock_wait_min,
    :m_innodb_rec_lock_wait_max,
    :m_innodb_rec_lock_wait_p99,
    :m_innodb_queue_wait_cnt,
    :m_innodb_queue_wait_sum,
    :m_innodb_queue_wait_min,
    :m_innodb_queue_wait_max,
    :m_innodb_queue_wait_p99,
    :m_innodb_pages_distinct_cnt,
    :m_innodb_pages_distinct_sum,
    :m_innodb_pages_distinct_min,
    :m_innodb_pages_distinct_max,
    :m_innodb_pages_distinct_p99,
    :m_query_length_cnt,
    :m_query_length_sum,
    :m_query_length_min,
    :m_query_length_max,
    :m_query_length_p99,
    :m_bytes_sent_cnt,
    :m_bytes_sent_sum,
    :m_bytes_sent_min,
    :m_bytes_sent_max,
    :m_bytes_sent_p99,
    :m_tmp_tables_cnt,
    :m_tmp_tables_sum,
    :m_tmp_tables_min,
    :m_tmp_tables_max,
    :m_tmp_tables_p99,
    :m_tmp_disk_tables_cnt,
    :m_tmp_disk_tables_sum,
    :m_tmp_disk_tables_min,
    :m_tmp_disk_tables_max,
    :m_tmp_disk_tables_p99,
    :m_tmp_table_sizes_cnt,
    :m_tmp_table_sizes_sum,
    :m_tmp_table_sizes_min,
    :m_tmp_table_sizes_max,
    :m_tmp_table_sizes_p99,
    :m_qc_hit_cnt,
    :m_qc_hit_sum,
    :m_full_scan_cnt,
    :m_full_scan_sum,
    :m_full_join_cnt,
    :m_full_join_sum,
    :m_tmp_table_cnt,
    :m_tmp_table_sum,
    :m_tmp_table_on_disk_cnt,
    :m_tmp_table_on_disk_sum,
    :m_filesort_cnt,
    :m_filesort_sum,
    :m_filesort_on_disk_cnt,
    :m_filesort_on_disk_sum,
    :m_select_full_range_join_cnt,
    :m_select_full_range_join_sum,
    :m_select_range_cnt,
    :m_select_range_sum,
    :m_select_range_check_cnt,
    :m_select_range_check_sum,
    :m_sort_range_cnt,
    :m_sort_range_sum,
    :m_sort_rows_cnt,
    :m_sort_rows_sum,
    :m_sort_scan_cnt,
    :m_sort_scan_sum,
    :m_no_index_used_cnt,
    :m_no_index_used_sum,
    :m_no_good_index_used_cnt,
    :m_no_good_index_used_sum,
    :m_docs_returned_cnt,
    :m_docs_returned_sum,
    :m_docs_returned_min,
    :m_docs_returned_max,
    :m_docs_returned_p99,
    :m_response_length_cnt,
    :m_response_length_sum,
    :m_response_length_min,
    :m_response_length_max,
    :m_response_length_p99,
    :m_docs_scanned_cnt,
    :m_docs_scanned_sum,
    :m_docs_scanned_min,
    :m_docs_scanned_max,
    :m_docs_scanned_p99
  )
`

// MetricsBucket implements models to store metrics bucket
type MetricsBucket struct {
	db *sqlx.DB
}

// NewMetricsBucket initialize MetricsBucket with db instance.
func NewMetricsBucket(db *sqlx.DB) MetricsBucket {
	return MetricsBucket{db: db}
}

// MetricsBucketExtended  extends proto MetricsBucket to store converted data into db.
type MetricsBucketExtended struct {
	PeriodStart      time.Time `json:"period_start_ts"`
	MetricsSource    string    `json:"metrics_source_s"`
	ExampleType      string    `json:"example_type_s"`
	ExampleFormat    string    `json:"example_format_s"`
	LabelsKey        []string  `json:"labels_key"`
	LabelsValues     []string  `json:"labels_value"`
	WarningsCode     []uint64  `json:"warnings_code"`
	WarningsCount    []uint64  `json:"warnings_count"`
	ErrorsCode       []uint64  `json:"errors_code"`
	ErrorsCount      []uint64  `json:"errors_count"`
	IsQueryTruncated uint8     `json:"is_query_truncated"` // uint32 -> uint8
	*qanpb.MetricsBucket
}

// Save store metrics bucket received from agent into db.
func (mb *MetricsBucket) Save(agentMsg *qanpb.CollectRequest) error {
	if len(agentMsg.MetricsBucket) == 0 {
		return errors.New("Nothing to save - no metrics buckets")
	}

	// TODO: find solution with better performance
	mb.db.Mapper = reflectx.NewMapperTagFunc("json", strings.ToUpper, func(value string) string {
		if strings.Contains(value, ",") {
			return strings.Split(value, ",")[0]
		}
		return value
	})
	tx, err := mb.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err.Error())
	}
	stmt, err := tx.PrepareNamed(insertSQL)
	if err != nil {
		return fmt.Errorf("prepare named: %s", err.Error())
	}
	var errs error
	for _, mb := range agentMsg.MetricsBucket {
		lk, lv := MapToArrsStrStr(mb.Labels)
		wk, wv := MapToArrsIntInt(mb.Warnings)
		ek, ev := MapToArrsIntInt(mb.Errors)

		var truncated uint8
		if mb.IsTruncated {
			truncated = 1
		}
		q := MetricsBucketExtended{
			time.Unix(int64(mb.GetPeriodStartUnixSecs()), 0).UTC(),
			mb.GetMetricsSource().String(),
			mb.GetExampleType().String(),
			mb.GetExampleFormat().String(),
			lk,
			lv,
			wk,
			wv,
			ek,
			ev,
			truncated,
			mb,
		}

		_, err := stmt.Exec(q)

		if err != nil {
			errs = fmt.Errorf("%v; execute: %v;", errs, err)
		}
	}
	if errs != nil {
		return errs
	}
	err = stmt.Close()
	if err != nil {
		return fmt.Errorf("close statement: %s", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("transaction commit %s", err.Error())
	}
	return nil
}

// MapToArrsStrStr converts map into two lists.
func MapToArrsStrStr(m map[string]string) (keys []string, values []string) {
	keys = []string{}
	values = []string{}
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}

// MapToArrsIntInt converts map into two lists.
func MapToArrsIntInt(m map[uint64]uint64) (keys []uint64, values []uint64) {
	keys = []uint64{}
	values = []uint64{}
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}
