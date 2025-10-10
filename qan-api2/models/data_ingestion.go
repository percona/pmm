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
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	qanpb "github.com/percona/pmm/api/qanpb"
)

const (
	prometheusNamespace = "qan_api2"
	prometheusSubsystem = "data_ingestion"

	requestsCap     = 100
	batchTimeout    = 500 * time.Millisecond
	batchErrorDelay = time.Second
)

//nolint:lll
const insertSQL = `
  INSERT INTO metrics
  (
    queryid,
    explain_fingerprint,
    placeholders_count,
    service_name,
    database,
    schema,
    tables,
    username,
    client_host,
    replication_set,
    cluster,
    service_type,
    service_id,
    environment,
    az,
    region,
    node_model,
    node_id,
    node_name,
    node_type,
    machine_id,
    container_name,
    container_id,
    labels.key,
    labels.value,
    agent_id,
    agent_type,
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
    m_docs_scanned_p99,
    m_shared_blks_hit_cnt,
    m_shared_blks_hit_sum,
    m_shared_blks_read_cnt,
    m_shared_blks_read_sum,
    m_shared_blks_dirtied_cnt,
    m_shared_blks_dirtied_sum,
    m_shared_blks_written_cnt,
    m_shared_blks_written_sum,
    m_local_blks_hit_cnt,
    m_local_blks_hit_sum,
    m_local_blks_read_cnt,
    m_local_blks_read_sum,
    m_local_blks_dirtied_cnt,
    m_local_blks_dirtied_sum,
    m_local_blks_written_cnt,
    m_local_blks_written_sum,
    m_temp_blks_read_cnt,
    m_temp_blks_read_sum,
    m_temp_blks_written_cnt,
    m_temp_blks_written_sum,
    m_shared_blk_read_time_cnt,
    m_shared_blk_read_time_sum,
    m_shared_blk_write_time_cnt,
    m_shared_blk_write_time_sum,
    m_local_blk_read_time_cnt,
    m_local_blk_read_time_sum,
    m_local_blk_write_time_cnt,
    m_local_blk_write_time_sum,
    m_cpu_user_time_cnt,
    m_cpu_user_time_sum,
    m_cpu_sys_time_cnt,
    m_cpu_sys_time_sum,
    m_plans_calls_sum,
    m_plans_calls_cnt,
    m_wal_records_sum,
    m_wal_records_cnt,
    m_wal_fpi_sum,
    m_wal_fpi_cnt,
    m_wal_bytes_sum,
    m_wal_bytes_cnt,
    m_plan_time_cnt,
    m_plan_time_sum,
    m_plan_time_min,
    m_plan_time_max,
    cmd_type,
    top_queryid,
    top_query,
    application_name,
    planid,
    query_plan,
    histogram_items
   )
  VALUES (
    :queryid,
    :explain_fingerprint,
    :placeholders_count,
    :service_name,
    :database,
    :schema,
    :tables,
    :username,
    :client_host,
    :replication_set,
    :cluster,
    :service_type,
    :service_id,
    :environment,
    :az,
    :region,
    :node_model,
    :node_id,
    :node_name,
    :node_type,
    :machine_id,
    :container_name,
    :container_id,
    :labels_key,
    :labels_value,
    :agent_id,
    CAST( :agent_type_s AS Enum8('qan-agent-type-invalid'=0, 'qan-mysql-perfschema-agent'=1, 'qan-mysql-slowlog-agent'=2, 'qan-mongodb-profiler-agent'=3, 'qan-postgresql-pgstatements-agent'=4, 'qan-postgresql-pgstatmonitor-agent'=5)) AS agent_type,
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
    :m_docs_scanned_p99,
    :m_shared_blks_hit_cnt,
    :m_shared_blks_hit_sum,
    :m_shared_blks_read_cnt,
    :m_shared_blks_read_sum,
    :m_shared_blks_dirtied_cnt,
    :m_shared_blks_dirtied_sum,
    :m_shared_blks_written_cnt,
    :m_shared_blks_written_sum,
    :m_local_blks_hit_cnt,
    :m_local_blks_hit_sum,
    :m_local_blks_read_cnt,
    :m_local_blks_read_sum,
    :m_local_blks_dirtied_cnt,
    :m_local_blks_dirtied_sum,
    :m_local_blks_written_cnt,
    :m_local_blks_written_sum,
    :m_temp_blks_read_cnt,
    :m_temp_blks_read_sum,
    :m_temp_blks_written_cnt,
    :m_temp_blks_written_sum,
    :m_shared_blk_read_time_cnt,
    :m_shared_blk_read_time_sum,
    :m_shared_blk_write_time_cnt,
    :m_shared_blk_write_time_sum,
    :m_local_blk_read_time_cnt,
    :m_local_blk_read_time_sum,
    :m_local_blk_write_time_cnt,
    :m_local_blk_write_time_sum,
    :m_cpu_user_time_cnt,
    :m_cpu_user_time_sum,
    :m_cpu_sys_time_cnt,
    :m_cpu_sys_time_sum,
    :m_plans_calls_sum,
    :m_plans_calls_cnt,
    :m_wal_records_sum,
    :m_wal_records_cnt,
    :m_wal_fpi_sum,
    :m_wal_fpi_cnt,
    :m_wal_bytes_sum,
    :m_wal_bytes_cnt,
    :m_plan_time_cnt, 
    :m_plan_time_sum,
    :m_plan_time_min,
    :m_plan_time_max,
    :cmd_type,
    :top_queryid,
    :top_query,
    :application_name,
    :planid,
    :query_plan,
    :histogram_items
  )
`

// MetricsBucketExtended extends proto MetricsBucket to store converted data into db.
type MetricsBucketExtended struct {
	PeriodStart      time.Time `json:"period_start_ts"`
	AgentType        string    `json:"agent_type_s"`
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

// MetricsBucket implements models to store metrics bucket.
type MetricsBucket struct {
	db         *sqlx.DB
	l          *logrus.Entry
	requestsCh chan *qanpb.CollectRequest

	mBucketsPerBatch  *prometheus.SummaryVec
	mBatchSaveSeconds *prometheus.SummaryVec
	mRequestsLen      prometheus.GaugeFunc
	mRequestsCap      prometheus.GaugeFunc
}

// NewMetricsBucket initialize MetricsBucket with db instance.
func NewMetricsBucket(db *sqlx.DB) *MetricsBucket {
	requestsCh := make(chan *qanpb.CollectRequest, requestsCap)

	mb := &MetricsBucket{
		db:         db,
		l:          logrus.WithField("component", "data_ingestion"),
		requestsCh: requestsCh,

		mBucketsPerBatch: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "buckets_per_batch",
			Help:       "Number of metric buckets per ClickHouse batch.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"}),
		mBatchSaveSeconds: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  prometheusNamespace,
			Subsystem:  prometheusSubsystem,
			Name:       "batch_save_seconds",
			Help:       "Batch save duration.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"error"}),
		mRequestsLen: prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "requests_len",
			Help:      "Enqueued Collect requests.",
		}, func() float64 { return float64(len(requestsCh)) }),
		mRequestsCap: prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: prometheusNamespace,
			Subsystem: prometheusSubsystem,
			Name:      "requests_cap",
			Help:      "Maximum number of Collect requests that can be enqueued.",
		}, func() float64 { return float64(cap(requestsCh)) }),
	}

	// initialize metrics with labels
	mb.mBucketsPerBatch.WithLabelValues("1")
	mb.mBatchSaveSeconds.WithLabelValues("1")

	return mb
}

// Describe implements prometheus.Collector interface.
func (mb *MetricsBucket) Describe(ch chan<- *prometheus.Desc) {
	mb.mBucketsPerBatch.Describe(ch)
	mb.mBatchSaveSeconds.Describe(ch)
	mb.mRequestsLen.Describe(ch)
	mb.mRequestsCap.Describe(ch)
}

// Collect implements prometheus.Collector interface.
func (mb *MetricsBucket) Collect(ch chan<- prometheus.Metric) {
	mb.mBucketsPerBatch.Collect(ch)
	mb.mBatchSaveSeconds.Collect(ch)
	mb.mRequestsLen.Collect(ch)
	mb.mRequestsCap.Collect(ch)
}

// Run stores incoming data until context is canceled.
// It exits when all data is stored.
func (mb *MetricsBucket) Run(ctx context.Context) {
	go func() {
		<-ctx.Done()
		mb.l.Warn("Closing requests channel.")
		close(mb.requestsCh)
	}()

	for ctx.Err() == nil {
		if err := mb.insertBatch(batchTimeout); err != nil {
			time.Sleep(batchErrorDelay)
		}
	}

	// insert one last final batch
	_ = mb.insertBatch(0)
}

func (mb *MetricsBucket) insertBatch(timeout time.Duration) error {
	// wait for first request before doing anything, ignore timeout
	req, ok := <-mb.requestsCh
	if !ok {
		mb.l.Warn("Requests channel closed, nothing to store.")
		return nil
	}

	var err error
	var buckets int
	start := time.Now()
	defer func() {
		d := time.Since(start)

		var e string
		if err == nil {
			e = "0"
			mb.l.Infof("Saved %d buckets in %s.", buckets, d)
		} else {
			e = "1"
			mb.l.Errorf("Failed to save %d buckets in %s: %s.", buckets, d, err)
		}

		mb.mBucketsPerBatch.WithLabelValues(e).Observe(float64(buckets))
		mb.mBatchSaveSeconds.WithLabelValues(e).Observe(d.Seconds())
	}()

	// begin "transaction" and commit or rollback it on exit
	var tx *sqlx.Tx
	if tx, err = mb.db.Beginx(); err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer func() {
		if err == nil {
			if err = tx.Commit(); err != nil {
				err = errors.Wrap(err, "failed to commit transaction")
			}
		} else {
			_ = tx.Rollback()
		}
	}()

	// prepare INSERT statement and close it on exit
	var stmt *sqlx.NamedStmt
	if stmt, err = tx.PrepareNamed(insertSQL); err != nil {
		return errors.Wrap(err, "failed to prepare statement")
	}
	defer func() {
		if e := stmt.Close(); e != nil && err == nil {
			err = errors.Wrap(e, "failed to close statement")
		}
	}()

	// limit only by time, not by batch size, because large batches already handled by the driver
	// ("block_size" query parameter)
	var timeoutCh <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timeoutCh = t.C
	}

	for {
		// INSERT buckets from current request
		for _, metricsBucket := range req.MetricsBucket {
			buckets++

			lk, lv := mapToArrsStrStr(metricsBucket.Labels)
			wk, wv := mapToArrsIntInt(metricsBucket.Warnings)
			ek, ev := mapToArrsIntInt(metricsBucket.Errors)

			var truncated uint8
			if metricsBucket.IsTruncated {
				truncated = 1
			}

			q := MetricsBucketExtended{
				time.Unix(int64(metricsBucket.GetPeriodStartUnixSecs()), 0).UTC(),
				agentTypeToClickHouseEnum(metricsBucket.GetAgentType()),
				metricsBucket.GetExampleType().String(),
				// TODO should we remove this field since it's deprecated?
				metricsBucket.GetExampleFormat().String(), //nolint:staticcheck
				lk,
				lv,
				wk,
				wv,
				ek,
				ev,
				truncated,
				metricsBucket,
			}

			if _, err = stmt.Exec(q); err != nil {
				return errors.Wrap(err, "failed to exec")
			}
		}

		// wait for the next request or exit on timer
		select {
		case req, ok = <-mb.requestsCh:
			if !ok {
				mb.l.Warn("Requests channel closed, exiting.")
				return nil
			}
		case <-timeoutCh:
			return nil
		}
	}
}

// Save store metrics bucket received from agent into db.
func (mb *MetricsBucket) Save(agentMsg *qanpb.CollectRequest) error { //nolint:unparam
	if len(agentMsg.MetricsBucket) == 0 {
		mb.l.Warnf("Nothing to save - no metrics buckets.")
		return nil
	}

	mb.requestsCh <- agentMsg
	return nil
}

// mapToArrsStrStr converts map into two lists.
func mapToArrsStrStr(m map[string]string) ([]string, []string) {
	keys := make([]string, 0, len(m))
	values := make([]string, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}

	return keys, values
}

// mapToArrsIntInt converts map into two lists.
func mapToArrsIntInt(m map[uint64]uint64) ([]uint64, []uint64) {
	keys := make([]uint64, 0, len(m))
	values := make([]uint64, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}

	return keys, values
}

// check interfaces.
var (
	_ prometheus.Collector = (*MetricsBucket)(nil)
)
