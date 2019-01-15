package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"

	collectorpb "github.com/Percona-Lab/qan-api/api/collector"
)

const insertSQL = `
  INSERT INTO queries
  (
	digest,
	digest_text,
	db_server,
	db_schema,
	db_username,
	client_host,
	labels.key,
	labels.value,
	agent_uuid,
	period_start,
	period_length,
	example,
	example_format,
	is_truncated,
	example_type,
	example_metrics,
	num_query_with_warnings,
	warnings.code,
	warnings.count,
	num_query_with_errors,
	errors.code,
	errors.count,
	num_queries,
	m_query_time_cnt,
	m_query_time_sum,
	m_query_time_min,
	m_query_time_max,
	m_query_time_p99,
	m_query_time_hg,
	m_lock_time_cnt,
	m_lock_time_sum,
	m_lock_time_min,
	m_lock_time_max,
	m_lock_time_p99,
	m_lock_time_hg,
	m_rows_sent_cnt,
	m_rows_sent_sum,
	m_rows_sent_min,
	m_rows_sent_max,
	m_rows_sent_p99,
	m_rows_sent_hg,
	m_rows_examined_cnt,
	m_rows_examined_sum,
	m_rows_examined_min,
	m_rows_examined_max,
	m_rows_examined_p99,
	m_rows_examined_hg,
	m_rows_affected_cnt,
	m_rows_affected_sum,
	m_rows_affected_min,
	m_rows_affected_max,
	m_rows_affected_p99,
	m_rows_affected_hg,
	m_rows_read_cnt,
	m_rows_read_sum,
	m_rows_read_min,
	m_rows_read_max,
	m_rows_read_p99,
	m_rows_read_hg,
	m_merge_passes_cnt,
	m_merge_passes_sum,
	m_merge_passes_min,
	m_merge_passes_max,
	m_merge_passes_p99,
	m_merge_passes_hg,
	m_innodb_io_r_ops_cnt,
	m_innodb_io_r_ops_sum,
	m_innodb_io_r_ops_min,
	m_innodb_io_r_ops_max,
	m_innodb_io_r_ops_p99,
	m_innodb_io_r_ops_hg,
	m_innodb_io_r_bytes_cnt,
	m_innodb_io_r_bytes_sum,
	m_innodb_io_r_bytes_min,
	m_innodb_io_r_bytes_max,
	m_innodb_io_r_bytes_p99,
	m_innodb_io_r_bytes_hg,
	m_innodb_io_r_wait_cnt,
	m_innodb_io_r_wait_sum,
	m_innodb_io_r_wait_min,
	m_innodb_io_r_wait_max,
	m_innodb_io_r_wait_p99,
	m_innodb_io_r_wait_hg,
	m_innodb_rec_lock_wait_cnt,
	m_innodb_rec_lock_wait_sum,
	m_innodb_rec_lock_wait_min,
	m_innodb_rec_lock_wait_max,
	m_innodb_rec_lock_wait_p99,
	m_innodb_rec_lock_wait_hg,
	m_innodb_queue_wait_cnt,
	m_innodb_queue_wait_sum,
	m_innodb_queue_wait_min,
	m_innodb_queue_wait_max,
	m_innodb_queue_wait_p99,
	m_innodb_queue_wait_hg,
	m_innodb_pages_distinct_cnt,
	m_innodb_pages_distinct_sum,
	m_innodb_pages_distinct_min,
	m_innodb_pages_distinct_max,
	m_innodb_pages_distinct_p99,
	m_innodb_pages_distinct_hg,
	m_query_length_cnt,
	m_query_length_sum,
	m_query_length_min,
	m_query_length_max,
	m_query_length_p99,
	m_query_length_hg,
	m_bytes_sent_cnt,
	m_bytes_sent_sum,
	m_bytes_sent_min,
	m_bytes_sent_max,
	m_bytes_sent_p99,
	m_bytes_sent_hg,
	m_tmp_tables_cnt,
	m_tmp_tables_sum,
	m_tmp_tables_min,
	m_tmp_tables_max,
	m_tmp_tables_p99,
	m_tmp_tables_hg,
	m_tmp_disk_tables_cnt,
	m_tmp_disk_tables_sum,
	m_tmp_disk_tables_min,
	m_tmp_disk_tables_max,
	m_tmp_disk_tables_p99,
	m_tmp_disk_tables_hg,
	m_tmp_table_sizes_cnt,
	m_tmp_table_sizes_sum,
	m_tmp_table_sizes_min,
	m_tmp_table_sizes_max,
	m_tmp_table_sizes_p99,
	m_tmp_table_sizes_hg,
	m_qc_hit_sum,
	m_full_scan_sum,
	m_full_join_sum,
	m_tmp_table_sum,
	m_tmp_table_on_disk_sum,
	m_filesort_sum,
	m_filesort_on_disk_sum,
	m_select_full_range_join_sum,
	m_select_range_sum,
	m_select_range_check_sum,
	m_sort_range_sum,
	m_sort_rows_sum,
	m_sort_scan_sum,
	m_no_index_used_sum,
	m_no_good_index_used_sum,
	grpstr,
	grpint,
	labint.key,
	labint.value
   )
  VALUES (
	:digest,
	:digest_text,
	:db_server,
	:db_schema,
	:db_username,
	:client_host,
	:labels_key,
	:labels_value,
	:agent_uuid,
	:period_start_ts,
	:period_length,
	:example,
	CAST( :example_format_s AS Enum8('EXAMPLE' = 0, 'DIGEST' = 1)) AS example_format,
	:is_truncated,
	CAST( :example_type_s AS Enum8('RANDOM' = 0, 'SLOWEST' = 1, 'FASTEST' = 2, 'WITH_ERROR' = 3)) AS example_type,
	:example_metrics,
	:num_query_with_warnings,
	:warnings_code,
	:warnings_count,
	:num_query_with_errors,
	:errors_code,
	:errors_count,
	:num_queries,
	:m_query_time_cnt,
	:m_query_time_sum,
	:m_query_time_min,
	:m_query_time_max,
	:m_query_time_p99,
	:m_query_time_hg,
	:m_lock_time_cnt,
	:m_lock_time_sum,
	:m_lock_time_min,
	:m_lock_time_max,
	:m_lock_time_p99,
	:m_lock_time_hg,
	:m_rows_sent_cnt,
	:m_rows_sent_sum,
	:m_rows_sent_min,
	:m_rows_sent_max,
	:m_rows_sent_p99,
	:m_rows_sent_hg,
	:m_rows_examined_cnt,
	:m_rows_examined_sum,
	:m_rows_examined_min,
	:m_rows_examined_max,
	:m_rows_examined_p99,
	:m_rows_examined_hg,
	:m_rows_affected_cnt,
	:m_rows_affected_sum,
	:m_rows_affected_min,
	:m_rows_affected_max,
	:m_rows_affected_p99,
	:m_rows_affected_hg,
	:m_rows_read_cnt,
	:m_rows_read_sum,
	:m_rows_read_min,
	:m_rows_read_max,
	:m_rows_read_p99,
	:m_rows_read_hg,
	:m_merge_passes_cnt,
	:m_merge_passes_sum,
	:m_merge_passes_min,
	:m_merge_passes_max,
	:m_merge_passes_p99,
	:m_merge_passes_hg,
	:m_innodb_io_r_ops_cnt,
	:m_innodb_io_r_ops_sum,
	:m_innodb_io_r_ops_min,
	:m_innodb_io_r_ops_max,
	:m_innodb_io_r_ops_p99,
	:m_innodb_io_r_ops_hg,
	:m_innodb_io_r_bytes_cnt,
	:m_innodb_io_r_bytes_sum,
	:m_innodb_io_r_bytes_min,
	:m_innodb_io_r_bytes_max,
	:m_innodb_io_r_bytes_p99,
	:m_innodb_io_r_bytes_hg,
	:m_innodb_io_r_wait_cnt,
	:m_innodb_io_r_wait_sum,
	:m_innodb_io_r_wait_min,
	:m_innodb_io_r_wait_max,
	:m_innodb_io_r_wait_p99,
	:m_innodb_io_r_wait_hg,
	:m_innodb_rec_lock_wait_cnt,
	:m_innodb_rec_lock_wait_sum,
	:m_innodb_rec_lock_wait_min,
	:m_innodb_rec_lock_wait_max,
	:m_innodb_rec_lock_wait_p99,
	:m_innodb_rec_lock_wait_hg,
	:m_innodb_queue_wait_cnt,
	:m_innodb_queue_wait_sum,
	:m_innodb_queue_wait_min,
	:m_innodb_queue_wait_max,
	:m_innodb_queue_wait_p99,
	:m_innodb_queue_wait_hg,
	:m_innodb_pages_distinct_cnt,
	:m_innodb_pages_distinct_sum,
	:m_innodb_pages_distinct_min,
	:m_innodb_pages_distinct_max,
	:m_innodb_pages_distinct_p99,
	:m_innodb_pages_distinct_hg,
	:m_query_length_cnt,
	:m_query_length_sum,
	:m_query_length_min,
	:m_query_length_max,
	:m_query_length_p99,
	:m_query_length_hg,
	:m_bytes_sent_cnt,
	:m_bytes_sent_sum,
	:m_bytes_sent_min,
	:m_bytes_sent_max,
	:m_bytes_sent_p99,
	:m_bytes_sent_hg,
	:m_tmp_tables_cnt,
	:m_tmp_tables_sum,
	:m_tmp_tables_min,
	:m_tmp_tables_max,
	:m_tmp_tables_p99,
	:m_tmp_tables_hg,
	:m_tmp_disk_tables_cnt,
	:m_tmp_disk_tables_sum,
	:m_tmp_disk_tables_min,
	:m_tmp_disk_tables_max,
	:m_tmp_disk_tables_p99,
	:m_tmp_disk_tables_hg,
	:m_tmp_table_sizes_cnt,
	:m_tmp_table_sizes_sum,
	:m_tmp_table_sizes_min,
	:m_tmp_table_sizes_max,
	:m_tmp_table_sizes_p99,
	:m_tmp_table_sizes_hg,
	:m_qc_hit_sum,
	:m_full_scan_sum,
	:m_full_join_sum,
	:m_tmp_table_sum,
	:m_tmp_table_on_disk_sum,
	:m_filesort_sum,
	:m_filesort_on_disk_sum,
	:m_select_full_range_join_sum,
	:m_select_range_sum,
	:m_select_range_check_sum,
	:m_sort_range_sum,
	:m_sort_rows_sum,
	:m_sort_scan_sum,
	:m_no_index_used_sum,
	:m_no_good_index_used_sum,
	:grpstr,
	:grpint,
	:labint_key
	:labint_value
  )
`

// QueryClass implements models to store query classes
type QueryClass struct {
	db *sqlx.DB
}

// NewQueryClass initialize QueryClass with db instance.
func NewQueryClass(db *sqlx.DB) QueryClass {
	return QueryClass{db: db}
}

// QueryClassExtended  extends proto QueryClass to store converted data into db.
type QueryClassExtended struct {
	PeriodStart   time.Time `json:"period_start_ts"`
	ExampleType   string    `json:"example_type_s"`
	ExampleFormat string    `json:"example_format_s"`
	LabelsKey     []string  `json:"labels_key"`
	LabelsValues  []string  `json:"labels_value"`
	WarningsCode  []string  `json:"warnings_code"`
	WarningsCount []uint64  `json:"warnings_count"`
	ErrorsCode    []string  `json:"errors_code"`
	ErrorsCount   []uint64  `json:"errors_count"`
	LabintKey     []uint32  `json:"labint_key"`
	LabintValue   []uint32  `json:"labint_value"`
	*collectorpb.QueryClass
}

// Save store cquery classes received from agent into db.
func (qc *QueryClass) Save(agentMsg *collectorpb.AgentMessage) error {

	if len(agentMsg.QueryClass) == 0 {
		return errors.New("Nothing to save - no query classes")
	}

	// TODO: find solution with better performance
	qc.db.Mapper = reflectx.NewMapperTagFunc("json", strings.ToUpper, func(value string) string {
		if strings.Contains(value, ",") {
			return strings.Split(value, ",")[0]
		}
		return value
	})
	tx, err := qc.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err.Error())
	}
	stmt, err := tx.PrepareNamed(insertSQL)
	if err != nil {
		return fmt.Errorf("prepare named: %s", err.Error())
	}
	var errs error
	for _, qc := range agentMsg.QueryClass {
		lk, lv := MapToArrsStrStr(qc.Labels)
		wk, wv := MapToArrsStrInt(qc.Warnings)
		ek, ev := MapToArrsStrInt(qc.Errors)
		labintk, labintv := MapToArrsIntInt(qc.Labint)
		q := QueryClassExtended{
			time.Unix(qc.GetPeriodStart(), 0),
			qc.GetExampleType().String(),
			qc.GetExampleFormat().String(),
			lk,
			lv,
			wk,
			wv,
			ek,
			ev,
			labintk,
			labintv,
			qc,
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

// MapToArrsStrInt converts map into two lists.
func MapToArrsStrInt(m map[string]uint64) (keys []string, values []uint64) {
	keys = []string{}
	values = []uint64{}
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}

// MapToArrsIntInt converts map into two lists.
func MapToArrsIntInt(m map[uint32]uint32) (keys []uint32, values []uint32) {
	keys = []uint32{}
	values = []uint32{}
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return keys, values
}
