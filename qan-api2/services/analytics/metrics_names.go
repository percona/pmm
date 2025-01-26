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
	"context"

	qanpb "github.com/percona/pmm/api/qan/v1"
)

// metricsNames is a map of metrics names and keys.
var metricsNames = map[string]string{
	"load":                    "Load",
	"count":                   "Count",
	"latency":                 "Latency",
	"query_time":              "Query Time",
	"lock_time":               "Lock Time",
	"rows_sent":               "Rows Sent",
	"rows_examined":           "Rows Examined",
	"rows_affected":           "Rows Affected",
	"rows_read":               "Rows Read",
	"merge_passes":            "Merge Passes",
	"innodb_io_r_ops":         "Innodb IO R Ops",
	"innodb_io_r_bytes":       "Innodb IO R Bytes",
	"innodb_io_r_wait":        "Innodb IO R Wait",
	"innodb_rec_lock_wait":    "Innodb Rec Lock Wait",
	"innodb_queue_wait":       "Innodb Queue Wait",
	"innodb_pages_distinct":   "Innodb Pages Distinct",
	"query_length":            "Query Length",
	"bytes_sent":              "Bytes Sent",
	"tmp_tables":              "Tmp Tables",
	"tmp_disk_tables":         "Tmp Disk Tables",
	"tmp_table_sizes":         "Tmp Table Sizes",
	"qc_hit":                  "Query Cache Hit",
	"full_scan":               "Full Scan",
	"full_join":               "Full Join",
	"tmp_table":               "Tmp Table",
	"tmp_table_on_disk":       "Tmp Table on Disk",
	"filesort":                "Filesort",
	"filesort_on_disk":        "Filesort on Disk",
	"select_full_range_join":  "Select Full Range Join",
	"select_range":            "Select Range",
	"select_range_check":      "Select Range Check",
	"sort_range":              "Sort Range",
	"sort_rows":               "Sort Rows",
	"sort_scan":               "Sort Scan",
	"no_index_used":           "No Index Used",
	"no_good_index_used":      "No Good Index Used",
	"docs_returned":           "Docs Returned",
	"response_length":         "Response Length",
	"docs_scanned":            "Docs Scanned",
	"m_shared_blks_hit":       "Shared blocks cache hits",
	"m_shared_blks_read":      "Shared blocks read",
	"m_shared_blks_dirtied":   "Shared blocks dirtied",
	"m_shared_blks_written":   "Shared blocks written",
	"m_local_blks_hit":        "Local blocks cache hits",
	"m_local_blks_read":       "Local blocks read",
	"m_local_blks_dirtied":    "Local blocks dirtied",
	"m_local_blks_written":    "Local blocks written",
	"m_temp_blks_read":        "Temp blocks read",
	"m_temp_blks_written":     "Temp blocks written",
	"m_blk_read_time":         "Time the statement spent reading blocks [deprecated]",
	"m_blk_write_time":        "Time the statement spent writing blocks [deprecated]",
	"m_shared_blk_read_time":  "Time the statement spent reading shared blocks",
	"m_shared_blk_write_time": "Time the statement spent writing shared blocks",
	"m_local_blk_read_time":   "Time the statement spent reading local_blocks",
	"m_local_blk_write_time":  "Time the statement spent writing local_blocks",
	"m_cpu_user_time":         "Total time user spent in query",
	"m_cpu_sys_time":          "Total time system spent in query",
	"m_plans_calls":           "Total number of planned calls",
	"m_wal_records":           "Total number of WAL (Write-ahead logging) records",
	"m_wal_fpi":               "Total number of FPI (full page images) in WAL (Write-ahead logging) records",
	"m_wal_bytes":             "Total bytes of WAL (Write-ahead logging) records",
	"m_plan_time":             "Total plan time spent in query",
	"cmd_type":                "Type of SQL command used in the query",
	"top_queryid":             "Top parent query ID",
	"top_query":               "Top query plain text",
	"application_name":        "Name provided by pg_stat_monitor",
	"planid":                  "Plan ID for query",
	"plan_summary":            "Plan summary from MongoDB collection system.profile",
}

// GetMetricsNames implements rpc to get list of available metrics.
func (s *Service) GetMetricsNames(_ context.Context, _ *qanpb.GetMetricsNamesRequest) (*qanpb.GetMetricsNamesResponse, error) { //nolint:unparam
	return &qanpb.GetMetricsNamesResponse{Data: metricsNames}, nil
}
