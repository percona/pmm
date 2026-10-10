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

package mcp

import (
	"slices"
	"strings"
)

// qanColumn is one QAN metric that pmm_top_queries can return.
type qanColumn struct {
	name string
	// engineMySQL, enginePostgreSQL, engineMongoDB, or "" for every engine.
	engine string
	// avg is set when qan-api2 reports the metric as a per-call average; otherwise it is a sum.
	avg bool
}

// qanColumns mirrors qan-api2's column maps, which drop an unknown name without an
// error; TestQANColumnsMatchQANAPI2 keeps the two in step (PMM-15529).
var qanColumns = []qanColumn{
	{"load", "", false},
	{"num_queries", "", false},
	{"num_queries_with_errors", "", false},
	{"num_queries_with_warnings", "", false},
	{"query_time", "", true},
	{"lock_time", "", true},
	{"rows_sent", "", true},
	{"rows_examined", "", true},
	{"rows_affected", "", true},
	{"rows_read", "", true},

	{"merge_passes", engineMySQL, true},
	{"innodb_io_r_ops", engineMySQL, true},
	{"innodb_io_r_bytes", engineMySQL, true},
	{"innodb_io_r_wait", engineMySQL, true},
	{"innodb_rec_lock_wait", engineMySQL, true},
	{"innodb_queue_wait", engineMySQL, true},
	{"innodb_pages_distinct", engineMySQL, true},
	{"query_length", engineMySQL, true},
	{"bytes_sent", engineMySQL, true},
	{"tmp_tables", engineMySQL, true},
	{"tmp_disk_tables", engineMySQL, true},
	{"tmp_table_sizes", engineMySQL, true},
	{"qc_hit", engineMySQL, false},
	{"full_scan", engineMySQL, false},
	{"full_join", engineMySQL, false},
	{"tmp_table", engineMySQL, false},
	{"tmp_table_on_disk", engineMySQL, false},
	{"filesort", engineMySQL, false},
	{"filesort_on_disk", engineMySQL, false},
	{"select_full_range_join", engineMySQL, false},
	{"select_range", engineMySQL, false},
	{"select_range_check", engineMySQL, false},
	{"sort_range", engineMySQL, false},
	{"sort_rows", engineMySQL, false},
	{"sort_scan", engineMySQL, false},
	{"no_index_used", engineMySQL, false},
	{"no_good_index_used", engineMySQL, false},

	{"shared_blks_hit", enginePostgreSQL, false},
	{"shared_blks_read", enginePostgreSQL, false},
	{"shared_blks_dirtied", enginePostgreSQL, false},
	{"shared_blks_written", enginePostgreSQL, false},
	{"local_blks_hit", enginePostgreSQL, false},
	{"local_blks_read", enginePostgreSQL, false},
	{"local_blks_dirtied", enginePostgreSQL, false},
	{"local_blks_written", enginePostgreSQL, false},
	{"temp_blks_read", enginePostgreSQL, false},
	{"temp_blks_written", enginePostgreSQL, false},
	{"blk_read_time", enginePostgreSQL, false},
	{"blk_write_time", enginePostgreSQL, false},
	{"shared_blk_read_time", enginePostgreSQL, false},
	{"shared_blk_write_time", enginePostgreSQL, false},
	{"local_blk_read_time", enginePostgreSQL, false},
	{"local_blk_write_time", enginePostgreSQL, false},
	{"cpu_user_time", enginePostgreSQL, false},
	{"cpu_sys_time", enginePostgreSQL, false},
	{"plans_calls", enginePostgreSQL, false},
	{"plan_time", enginePostgreSQL, false},
	{"wal_records", enginePostgreSQL, false},
	{"wal_fpi", enginePostgreSQL, false},
	{"wal_bytes", enginePostgreSQL, false},
	{"wal_buffers_full", enginePostgreSQL, false},
	{"parallel_workers_to_launch", enginePostgreSQL, false},
	{"parallel_workers_launched", enginePostgreSQL, false},

	{"docs_returned", engineMongoDB, true},
	{"response_length", engineMongoDB, true},
	{"docs_scanned", engineMongoDB, true},
	{"docs_examined", engineMongoDB, true},
	{"keys_examined", engineMongoDB, true},
	{"storage_bytes_read", engineMongoDB, true},
	{"storage_time_reading_micros", engineMongoDB, true},
	{"locks_database_time_acquiring_micros_read_shared", engineMongoDB, true},
	{"locks_global_acquire_count_read_shared", engineMongoDB, false},
	{"locks_global_acquire_count_write_shared", engineMongoDB, false},
	{"locks_database_acquire_count_read_shared", engineMongoDB, false},
	{"locks_database_acquire_wait_count_read_shared", engineMongoDB, false},
	{"locks_collection_acquire_count_read_shared", engineMongoDB, false},
}

// qanColumnByName returns the catalogue entry for a column name.
func qanColumnByName(name string) (qanColumn, bool) {
	i := slices.IndexFunc(qanColumns, func(c qanColumn) bool { return c.name == name })
	if i < 0 {
		return qanColumn{}, false
	}
	return qanColumns[i], true
}

// qanColumnsText lists the catalogue's names by engine, for the tool description.
func qanColumnsText() string {
	groups := []struct{ label, engine string }{{"all engines", ""}, {"MySQL", engineMySQL}, {"PostgreSQL", enginePostgreSQL}, {"MongoDB", engineMongoDB}}
	parts := make([]string, 0, len(groups))
	for _, g := range groups {
		var names []string
		for _, c := range qanColumns {
			if c.engine == g.engine {
				names = append(names, c.name)
			}
		}
		parts = append(parts, g.label+": "+strings.Join(names, ", "))
	}
	return strings.Join(parts, "; ")
}
