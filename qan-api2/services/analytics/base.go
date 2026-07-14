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

package analytics

import (
	qanpb "github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/qan-api2/models"
)

// Service implements gRPC service to communicate with QAN-APP.
type Service struct {
	rm models.Reporter
	mm models.Metrics

	qanpb.UnimplementedProfileServer
	qanpb.UnimplementedFiltersServer
	qanpb.UnimplementedObjectDetailsServer
	qanpb.UnimplementedMetricsNamesServer
}

// NewService create new insstance of Service.
func NewService(rm models.Reporter, mm models.Metrics) *Service {
	return &Service{rm: rm, mm: mm}
}

var standartDimensions = map[string]struct{}{
	"queryid":          {},
	"service_name":     {},
	"database":         {},
	"schema":           {},
	"username":         {},
	"client_host":      {},
	"cmd_type":         {},
	"application_name": {},
	"top_queryid":      {},
	"planid":           {},
}

var sumColumnNames = map[string]struct{}{
	"qc_hit":                 {},
	"full_scan":              {},
	"full_join":              {},
	"tmp_table":              {},
	"tmp_table_on_disk":      {},
	"filesort":               {},
	"filesort_on_disk":       {},
	"select_full_range_join": {},
	"select_range":           {},
	"select_range_check":     {},
	"sort_range":             {},
	"sort_rows":              {},
	"sort_scan":              {},
	"no_index_used":          {},
	"no_good_index_used":     {},
	"shared_blks_hit":        {},
	"shared_blks_read":       {},
	"shared_blks_dirtied":    {},
	"shared_blks_written":    {},
	"local_blks_hit":         {},
	"local_blks_read":        {},
	"local_blks_dirtied":     {},
	"local_blks_written":     {},
	"temp_blks_read":         {},
	"temp_blks_written":      {},
	"blk_read_time":          {},
	"blk_write_time":         {},
	"shared_blk_read_time":   {},
	"shared_blk_write_time":  {},
	"local_blk_read_time":    {},
	"local_blk_write_time":   {},
	"cpu_user_time":          {},
	"cpu_sys_time":           {},
	"plans_calls":            {},
	"wal_records":            {},
	"wal_fpi":                {},
	"plan_time":              {},
	"wal_bytes":              {},
}

func isBoolMetric(name string) bool {
	_, ok := sumColumnNames[name]
	return ok
}

var specialColumnNames = map[string]struct{}{
	"load":                      {},
	"num_queries":               {},
	"num_queries_with_errors":   {},
	"num_queries_with_warnings": {},
}

func isSpecialMetric(name string) bool {
	_, ok := specialColumnNames[name]
	return ok
}

var commonColumnNames = map[string]struct{}{
	"query_time":            {},
	"lock_time":             {},
	"rows_sent":             {},
	"rows_examined":         {},
	"rows_affected":         {},
	"rows_read":             {},
	"merge_passes":          {},
	"innodb_io_r_ops":       {},
	"innodb_io_r_bytes":     {},
	"innodb_io_r_wait":      {},
	"innodb_rec_lock_wait":  {},
	"innodb_queue_wait":     {},
	"innodb_pages_distinct": {},
	"query_length":          {},
	"bytes_sent":            {},
	"tmp_tables":            {},
	"tmp_disk_tables":       {},
	"tmp_table_sizes":       {},
	"docs_returned":         {},
	"response_length":       {},
	"docs_scanned":          {},
}

func isCommonMetric(name string) bool {
	_, ok := commonColumnNames[name]
	return ok
}

func interfaceToFloat32(unk interface{}) float32 {
	switch i := unk.(type) {
	case float64:
		return float32(i)
	case float32:
		return i
	case int64:
		return float32(i)
	default:
		return float32(0)
	}
}

func interfaceToString(unk interface{}) string {
	switch i := unk.(type) {
	case string:
		return i
	default:
		return ""
	}
}

func isDimension(name string) bool {
	dimensionColumnNames := map[string]struct{}{
		// Main dimensions
		"queryid":      {},
		"service_name": {},
		"database":     {},
		"schema":       {},
		"username":     {},
		"client_host":  {},
		// Standard labels
		"replication_set":  {},
		"cluster":          {},
		"service_type":     {},
		"service_id":       {},
		"environment":      {},
		"az":               {},
		"region":           {},
		"node_model":       {},
		"node_id":          {},
		"node_name":        {},
		"node_type":        {},
		"machine_id":       {},
		"container_name":   {},
		"container_id":     {},
		"cmd_type":         {},
		"application_name": {},
		"top_queryid":      {},
		"planid":           {},
	}

	_, ok := dimensionColumnNames[name]
	return ok
}

// isTimeMetric checks if a metric in the time metrics group.
func isTimeMetric(name string) bool {
	timeColumnNames := map[string]struct{}{
		"query_time":            {},
		"lock_time":             {},
		"innodb_io_r_wait":      {},
		"innodb_rec_lock_wait":  {},
		"innodb_queue_wait":     {},
		"blk_read_time":         {},
		"blk_write_time":        {},
		"shared_blk_read_time":  {},
		"shared_blk_write_time": {},
		"local_blk_read_time":   {},
		"local_blk_write_time":  {},
		"cpu_user_time":         {},
		"cpu_sys_time":          {},
		"plan_time":             {},
	}

	_, ok := timeColumnNames[name]
	return ok
}
