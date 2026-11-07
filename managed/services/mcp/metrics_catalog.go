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

import "sync"

const engineNode = "node"

// metricsEngines are the engines pmm_catalog's metrics catalogue has entries for.
var metricsEngines = []string{engineMySQL, enginePostgreSQL, engineMongoDB, engineNode}

// metricInfo is one metric of the metrics catalogue.
type metricInfo struct {
	Name   string
	Engine string
	// Meaning is one line saying what the metric measures.
	Meaning string
	// Unit is the unit of the value Query returns.
	Unit string
	// Type is counter or gauge.
	Type string
	// Labels are the label names on the metric's samples as the exporter exposes them.
	Labels []string
	// Query is a MetricsQL expression with <service> (or <node>, for node metrics) to replace with a name.
	Query string
}

var metricInfoIndex = sync.OnceValue(func() map[string]metricInfo {
	m := make(map[string]metricInfo, len(metricsCatalog))
	for _, i := range metricsCatalog {
		m[i.Name] = i
	}
	return m
})

// metricInfoByName returns the catalogue entry of a metric.
func metricInfoByName(name string) (metricInfo, bool) {
	i, ok := metricInfoIndex()[name]
	return i, ok
}

// metricsCatalog lists, per engine, the metrics of the dashboards that PMM ships and of the health signals that are
// present in the exporter output captured under testdata/exporters. Counter queries use rate over 5 minutes.
var metricsCatalog = []metricInfo{
	{
		Name: "mysql_global_status_aborted_clients", Engine: engineMySQL,
		Meaning: "Connections closed without a proper client disconnect, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_aborted_clients{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_aborted_connects", Engine: engineMySQL,
		Meaning: "Failed attempts to connect to the server, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_aborted_connects{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_bytes_received", Engine: engineMySQL,
		Meaning: "Bytes received from all clients, per second.",
		Unit:    "Bps", Type: "counter",
		Query: `rate(mysql_global_status_bytes_received{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_bytes_sent", Engine: engineMySQL,
		Meaning: "Bytes sent to all clients, per second.",
		Unit:    "Bps", Type: "counter",
		Query: `rate(mysql_global_status_bytes_sent{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_commands_total", Engine: engineMySQL,
		Meaning: "Statements executed, per second, by statement type.",
		Unit:    "ops", Type: "counter", Labels: []string{"command"},
		Query: `sum by (command) (rate(mysql_global_status_commands_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mysql_global_status_created_tmp_disk_tables", Engine: engineMySQL,
		Meaning: "Internal temporary tables created on disk, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_created_tmp_disk_tables{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_created_tmp_files", Engine: engineMySQL,
		Meaning: "Temporary files created by mysqld, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_created_tmp_files{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_created_tmp_tables", Engine: engineMySQL,
		Meaning: "Internal temporary tables created, in memory or on disk, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_created_tmp_tables{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_handlers_total", Engine: engineMySQL,
		Meaning: "Storage engine handler calls, per second, by handler (read_key, write, delete and so on).",
		Unit:    "ops", Type: "counter", Labels: []string{"handler"},
		Query: `sum by (handler) (rate(mysql_global_status_handlers_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mysql_global_status_innodb_buffer_pool_read_requests", Engine: engineMySQL,
		Meaning: "Logical read requests to the InnoDB buffer pool, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_innodb_buffer_pool_read_requests{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_innodb_buffer_pool_reads", Engine: engineMySQL,
		Meaning: "Logical reads the buffer pool could not serve, so InnoDB read the disk, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_innodb_buffer_pool_reads{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_innodb_num_open_files", Engine: engineMySQL,
		Meaning: "Files InnoDB currently holds open.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_innodb_num_open_files{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_max_used_connections", Engine: engineMySQL,
		Meaning: "Highest number of connections open at once since the server started.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_max_used_connections{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_open_files", Engine: engineMySQL,
		Meaning: "Files currently open.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_open_files{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_open_table_definitions", Engine: engineMySQL,
		Meaning: "Table definitions currently cached.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_open_table_definitions{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_open_tables", Engine: engineMySQL,
		Meaning: "Tables currently open.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_open_tables{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_opened_files", Engine: engineMySQL,
		Meaning: "Files opened, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_opened_files{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_opened_table_definitions", Engine: engineMySQL,
		Meaning: "Table definitions cached, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_opened_table_definitions{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_opened_tables", Engine: engineMySQL,
		Meaning: "Tables opened, per second; a high rate suggests table_open_cache is too small.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_opened_tables{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_queries", Engine: engineMySQL,
		Meaning: "Statements executed per second, including those run inside stored programs: the QPS.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_queries{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_questions", Engine: engineMySQL,
		Meaning: "Statements sent by clients, per second, not counting those run inside stored programs.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_questions{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_select_full_join", Engine: engineMySQL,
		Meaning: "Joins that did a full scan of the first table because they use no index, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_select_full_join{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_select_full_range_join", Engine: engineMySQL,
		Meaning: "Joins that used a range search on a reference table, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_select_full_range_join{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_select_range", Engine: engineMySQL,
		Meaning: "Joins that used a range on the first table, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_select_range{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_select_range_check", Engine: engineMySQL,
		Meaning: "Joins without keys that check for key use after each row, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_select_range_check{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_select_scan", Engine: engineMySQL,
		Meaning: "Joins that did a full scan of the first table, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_select_scan{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_slow_queries", Engine: engineMySQL,
		Meaning: "Statements slower than long_query_time, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_slow_queries{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_sort_merge_passes", Engine: engineMySQL,
		Meaning: "Merge passes the sort algorithm made, per second; a high rate suggests a small sort_buffer_size.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_sort_merge_passes{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_sort_range", Engine: engineMySQL,
		Meaning: "Sorts done with ranges, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_sort_range{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_sort_rows", Engine: engineMySQL,
		Meaning: "Rows sorted, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_sort_rows{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_sort_scan", Engine: engineMySQL,
		Meaning: "Sorts done by scanning the table, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_sort_scan{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_table_locks_immediate", Engine: engineMySQL,
		Meaning: "Table lock requests granted at once, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_table_locks_immediate{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_table_locks_waited", Engine: engineMySQL,
		Meaning: "Table lock requests that had to wait, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_table_locks_waited{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_table_open_cache_hits", Engine: engineMySQL,
		Meaning: "Lookups in the table open cache that hit, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_table_open_cache_hits{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_table_open_cache_misses", Engine: engineMySQL,
		Meaning: "Lookups in the table open cache that missed, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_table_open_cache_misses{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_table_open_cache_overflows", Engine: engineMySQL,
		Meaning: "Table open cache entries evicted because the cache was full, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_table_open_cache_overflows{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_threads_cached", Engine: engineMySQL,
		Meaning: "Threads in the thread cache.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_threads_cached{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_threads_connected", Engine: engineMySQL,
		Meaning: "Currently open client connections.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_threads_connected{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_threads_created", Engine: engineMySQL,
		Meaning: "Threads created to handle connections, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(mysql_global_status_threads_created{service_name="<service>"}[5m])`,
	},
	{
		Name: "mysql_global_status_threads_running", Engine: engineMySQL,
		Meaning: "Threads that are not sleeping.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_status_threads_running{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_status_uptime", Engine: engineMySQL,
		Meaning: "Seconds the server has been up.",
		Unit:    "s", Type: "gauge",
		Query: `mysql_global_status_uptime{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_innodb_buffer_pool_size", Engine: engineMySQL,
		Meaning: "Configured size of the InnoDB buffer pool.",
		Unit:    "bytes", Type: "gauge",
		Query: `mysql_global_variables_innodb_buffer_pool_size{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_innodb_log_buffer_size", Engine: engineMySQL,
		Meaning: "Configured size of the InnoDB log buffer.",
		Unit:    "bytes", Type: "gauge",
		Query: `mysql_global_variables_innodb_log_buffer_size{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_key_buffer_size", Engine: engineMySQL,
		Meaning: "Configured size of the MyISAM key buffer.",
		Unit:    "bytes", Type: "gauge",
		Query: `mysql_global_variables_key_buffer_size{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_max_connections", Engine: engineMySQL,
		Meaning: "Configured maximum number of simultaneous client connections.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_variables_max_connections{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_open_files_limit", Engine: engineMySQL,
		Meaning: "Configured limit on open files.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_variables_open_files_limit{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_table_definition_cache", Engine: engineMySQL,
		Meaning: "Configured number of table definitions the cache holds.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_variables_table_definition_cache{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_table_open_cache", Engine: engineMySQL,
		Meaning: "Configured size of the table open cache.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_variables_table_open_cache{service_name="<service>"}`,
	},
	{
		Name: "mysql_global_variables_thread_cache_size", Engine: engineMySQL,
		Meaning: "Configured number of threads the server caches.",
		Unit:    "short", Type: "gauge",
		Query: `mysql_global_variables_thread_cache_size{service_name="<service>"}`,
	},
	{
		Name: "mysql_info_schema_threads", Engine: engineMySQL,
		Meaning: "Threads in information_schema.processlist, by state.",
		Unit:    "short", Type: "gauge", Labels: []string{"state"},
		Query: `sum by (state) (mysql_info_schema_threads{service_name="<service>"})`,
	},
	{
		Name: "mysql_slave_status_seconds_behind_master", Engine: engineMySQL,
		Meaning: "Seconds the replica's SQL thread is behind the source; only a replica reports it.",
		Unit:    "s", Type: "gauge", Labels: []string{"channel_name", "connection_name", "master_host", "master_uuid"},
		Query: `mysql_slave_status_seconds_behind_master{service_name="<service>"}`,
	},
	{
		Name: "mysql_slave_status_sql_delay", Engine: engineMySQL,
		Meaning: "Configured replication delay (SOURCE_DELAY) in seconds; only a replica reports it.",
		Unit:    "s", Type: "gauge", Labels: []string{"channel_name", "connection_name", "master_host", "master_uuid"},
		Query: `mysql_slave_status_sql_delay{service_name="<service>"}`,
	},
	{
		Name: "mysql_up", Engine: engineMySQL,
		Meaning: "1 when the exporter reached MySQL, 0 when it did not.",
		Unit:    "none", Type: "gauge",
		Query: `mysql_up{service_name="<service>"}`,
	},
	{
		Name: "pg_custom_database_size_custom_bytes", Engine: enginePostgreSQL,
		Meaning: "Size of each database, from PMM's custom query.",
		Unit:    "bytes", Type: "gauge", Labels: []string{"datname", "server"},
		Query: `max by (datname) (pg_custom_database_size_custom_bytes{service_name="<service>"})`,
	},
	{
		Name: "pg_database_wraparound_age_datfrozenxid_seconds", Engine: enginePostgreSQL,
		Meaning: "Age of each database's oldest unfrozen transaction ID, in transactions; near 2 billion means wraparound.",
		Unit:    "short", Type: "gauge", Labels: []string{"datname"},
		Query: `max by (datname) (pg_database_wraparound_age_datfrozenxid_seconds{service_name="<service>"})`,
	},
	{
		Name: "pg_locks_count", Engine: enginePostgreSQL,
		Meaning: "Locks held, by lock mode.",
		Unit:    "short", Type: "gauge", Labels: []string{"datname", "mode"},
		Query: `sum by (mode) (pg_locks_count{service_name="<service>"})`,
	},
	{
		Name: "pg_replication_lag_seconds", Engine: enginePostgreSQL,
		Meaning: "Seconds a standby is behind its primary; only a standby reports it.",
		Unit:    "s", Type: "gauge",
		Query: `pg_replication_lag_seconds{service_name="<service>"}`,
	},
	{
		Name: "pg_settings_autovacuum", Engine: enginePostgreSQL,
		Meaning: "1 when autovacuum is on.",
		Unit:    "none", Type: "gauge", Labels: []string{"server"},
		Query: `pg_settings_autovacuum{service_name="<service>"}`,
	},
	{
		Name: "pg_settings_autovacuum_freeze_max_age", Engine: enginePostgreSQL,
		Meaning: "Configured autovacuum_freeze_max_age, in transactions.",
		Unit:    "short", Type: "gauge", Labels: []string{"server"},
		Query: `pg_settings_autovacuum_freeze_max_age{service_name="<service>"}`,
	},
	{
		Name: "pg_settings_max_connections", Engine: enginePostgreSQL,
		Meaning: "Configured max_connections.",
		Unit:    "short", Type: "gauge", Labels: []string{"server"},
		Query: `pg_settings_max_connections{service_name="<service>"}`,
	},
	{
		Name: "pg_settings_shared_buffers_bytes", Engine: enginePostgreSQL,
		Meaning: "Configured shared_buffers.",
		Unit:    "bytes", Type: "gauge", Labels: []string{"server"},
		Query: `pg_settings_shared_buffers_bytes{service_name="<service>"}`,
	},
	{
		Name: "pg_settings_wal_buffers_bytes", Engine: enginePostgreSQL,
		Meaning: "Configured wal_buffers.",
		Unit:    "bytes", Type: "gauge", Labels: []string{"server"},
		Query: `pg_settings_wal_buffers_bytes{service_name="<service>"}`,
	},
	{
		Name: "pg_settings_work_mem_bytes", Engine: enginePostgreSQL,
		Meaning: "Configured work_mem.",
		Unit:    "bytes", Type: "gauge", Labels: []string{"server"},
		Query: `pg_settings_work_mem_bytes{service_name="<service>"}`,
	},
	{
		Name: "pg_stat_activity_count", Engine: enginePostgreSQL,
		Meaning: "Backends in pg_stat_activity, by state.",
		Unit:    "short", Type: "gauge", Labels: []string{"application_name", "datname", "server", "state", "usename"},
		Query: `sum by (state) (pg_stat_activity_count{service_name="<service>"})`,
	},
	{
		Name: "pg_stat_activity_max_tx_duration", Engine: enginePostgreSQL,
		Meaning: "Longest open transaction in seconds, by backend state.",
		Unit:    "s", Type: "gauge", Labels: []string{"application_name", "datname", "server", "state", "usename"},
		Query: `max by (state) (pg_stat_activity_max_tx_duration{service_name="<service>"})`,
	},
	{
		Name: "pg_stat_bgwriter_buffers_alloc", Engine: enginePostgreSQL,
		Meaning: "Buffers allocated, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_buffers_alloc{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_bgwriter_buffers_backend", Engine: enginePostgreSQL,
		Meaning: "Buffers backends wrote directly, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_buffers_backend{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_bgwriter_buffers_backend_fsync", Engine: enginePostgreSQL,
		Meaning: "Times a backend had to run its own fsync, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_buffers_backend_fsync{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_bgwriter_buffers_checkpoint", Engine: enginePostgreSQL,
		Meaning: "Buffers written by checkpoints, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_buffers_checkpoint{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_bgwriter_buffers_clean", Engine: enginePostgreSQL,
		Meaning: "Buffers written by the background writer, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_buffers_clean{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_bgwriter_checkpoint_sync_time", Engine: enginePostgreSQL,
		Meaning: "Milliseconds checkpoints spent syncing files to disk, per second.",
		Unit:    "ms", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_checkpoint_sync_time{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_bgwriter_checkpoint_write_time", Engine: enginePostgreSQL,
		Meaning: "Milliseconds checkpoints spent writing files to disk, per second.",
		Unit:    "ms", Type: "counter", Labels: []string{"collector", "server"},
		Query: `rate(pg_stat_bgwriter_checkpoint_write_time{service_name="<service>"}[5m])`,
	},
	{
		Name: "pg_stat_database_blk_read_time", Engine: enginePostgreSQL,
		Meaning: "Milliseconds spent reading data file blocks, per second, by database; needs track_io_timing.",
		Unit:    "ms", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_blk_read_time{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_blk_write_time", Engine: enginePostgreSQL,
		Meaning: "Milliseconds spent writing data file blocks, per second, by database; needs track_io_timing.",
		Unit:    "ms", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_blk_write_time{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_blks_hit", Engine: enginePostgreSQL,
		Meaning: "Block reads served from the shared buffer cache, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_blks_hit{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_blks_read", Engine: enginePostgreSQL,
		Meaning: "Block reads that went to disk or the OS cache, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_blks_read{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_conflicts", Engine: enginePostgreSQL,
		Meaning: "Queries canceled by recovery conflicts on a standby, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_conflicts{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_conflicts_confl_bufferpin", Engine: enginePostgreSQL,
		Meaning: "Queries canceled because of a pinned buffer, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname", "server"},
		Query: `sum by (datname) (rate(pg_stat_database_conflicts_confl_bufferpin{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_conflicts_confl_deadlock", Engine: enginePostgreSQL,
		Meaning: "Queries canceled because of a deadlock with recovery, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname", "server"},
		Query: `sum by (datname) (rate(pg_stat_database_conflicts_confl_deadlock{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_conflicts_confl_lock", Engine: enginePostgreSQL,
		Meaning: "Queries canceled because of a lock timeout, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname", "server"},
		Query: `sum by (datname) (rate(pg_stat_database_conflicts_confl_lock{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_conflicts_confl_snapshot", Engine: enginePostgreSQL,
		Meaning: "Queries canceled because of an old snapshot, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname", "server"},
		Query: `sum by (datname) (rate(pg_stat_database_conflicts_confl_snapshot{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_conflicts_confl_tablespace", Engine: enginePostgreSQL,
		Meaning: "Queries canceled because of a dropped tablespace, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname", "server"},
		Query: `sum by (datname) (rate(pg_stat_database_conflicts_confl_tablespace{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_deadlocks", Engine: enginePostgreSQL,
		Meaning: "Deadlocks detected, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_deadlocks{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_temp_bytes", Engine: enginePostgreSQL,
		Meaning: "Bytes written to temporary files by queries, per second, by database.",
		Unit:    "Bps", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_temp_bytes{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_temp_files", Engine: enginePostgreSQL,
		Meaning: "Temporary files queries created, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_temp_files{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_tup_deleted", Engine: enginePostgreSQL,
		Meaning: "Rows deleted, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_tup_deleted{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_tup_fetched", Engine: enginePostgreSQL,
		Meaning: "Rows fetched by queries, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_tup_fetched{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_tup_inserted", Engine: enginePostgreSQL,
		Meaning: "Rows inserted, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_tup_inserted{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_tup_returned", Engine: enginePostgreSQL,
		Meaning: "Rows returned by queries, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_tup_returned{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_tup_updated", Engine: enginePostgreSQL,
		Meaning: "Rows updated, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_tup_updated{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_xact_commit", Engine: enginePostgreSQL,
		Meaning: "Transactions committed, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_xact_commit{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_database_xact_rollback", Engine: enginePostgreSQL,
		Meaning: "Transactions rolled back, per second, by database.",
		Unit:    "ops", Type: "counter", Labels: []string{"datid", "datname"},
		Query: `sum by (datname) (rate(pg_stat_database_xact_rollback{service_name="<service>"}[5m]))`,
	},
	{
		Name: "pg_stat_user_tables_last_autovacuum", Engine: enginePostgreSQL,
		Meaning: "Time of the most recent autovacuum on any user table, as a Unix timestamp.",
		Unit:    "dateTimeAsSystem", Type: "gauge", Labels: []string{"datname", "relname", "schemaname", "server"},
		Query: `max(pg_stat_user_tables_last_autovacuum{service_name="<service>"})`,
	},
	{
		Name: "pg_stat_user_tables_n_dead_tup", Engine: enginePostgreSQL,
		Meaning: "Estimated dead rows of the ten user tables with the most.",
		Unit:    "short", Type: "gauge", Labels: []string{"datname", "relname", "schemaname", "server"},
		Query: `topk(10, max by (relname) (pg_stat_user_tables_n_dead_tup{service_name="<service>"}))`,
	},
	{
		Name: "pg_stat_user_tables_n_live_tup", Engine: enginePostgreSQL,
		Meaning: "Estimated live rows of the ten user tables with the most.",
		Unit:    "short", Type: "gauge", Labels: []string{"datname", "relname", "schemaname", "server"},
		Query: `topk(10, max by (relname) (pg_stat_user_tables_n_live_tup{service_name="<service>"}))`,
	},
	{
		Name: "pg_up", Engine: enginePostgreSQL,
		Meaning: "1 when the exporter reached PostgreSQL, 0 when it did not.",
		Unit:    "none", Type: "gauge", Labels: []string{"collector"},
		Query: `pg_up{service_name="<service>"}`,
	},
	{
		Name: "mongodb_asserts_total", Engine: engineMongoDB,
		Meaning: "Assertions raised, per second, by type (regular, warning, msg, user, rollovers).",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "type"},
		Query: `sum by (type) (rate(mongodb_asserts_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_connections", Engine: engineMongoDB,
		Meaning: "Connections, by state (current, available, active and so on).",
		Unit:    "short", Type: "gauge", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "state"},
		Query: `mongodb_connections{service_name="<service>"}`,
	},
	{
		Name: "mongodb_extra_info_page_faults_total", Engine: engineMongoDB,
		Meaning: "Page faults, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state"},
		Query: `rate(mongodb_extra_info_page_faults_total{service_name="<service>"}[5m])`,
	},
	{
		Name: "mongodb_instance_uptime_seconds", Engine: engineMongoDB,
		Meaning: "Seconds the instance has been up.",
		Unit:    "s", Type: "gauge", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state"},
		Query: `mongodb_instance_uptime_seconds{service_name="<service>"}`,
	},
	{
		Name: "mongodb_mongod_global_lock_current_queue", Engine: engineMongoDB,
		Meaning: "Operations waiting for the global lock, by type (readers, writers).",
		Unit:    "short", Type: "gauge", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "type"},
		Query: `mongodb_mongod_global_lock_current_queue{service_name="<service>"}`,
	},
	{
		Name: "mongodb_mongod_metrics_cursor_open", Engine: engineMongoDB,
		Meaning: "Open cursors, by state (total, noTimeout, pinned).",
		Unit:    "short", Type: "gauge", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "state"},
		Query: `mongodb_mongod_metrics_cursor_open{service_name="<service>"}`,
	},
	{
		Name: "mongodb_mongod_metrics_document_total", Engine: engineMongoDB,
		Meaning: "Documents processed, per second, by operation (returned, inserted, updated, deleted).",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "state"},
		Query: `sum by (state) (rate(mongodb_mongod_metrics_document_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_mongod_metrics_get_last_error_wtime_num_total", Engine: engineMongoDB,
		Meaning: "getLastError calls with a write concern that waited, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state"},
		Query: `rate(mongodb_mongod_metrics_get_last_error_wtime_num_total{service_name="<service>"}[5m])`,
	},
	{
		Name: "mongodb_mongod_metrics_get_last_error_wtime_total_milliseconds", Engine: engineMongoDB,
		Meaning: "Milliseconds getLastError calls waited for the write concern, per second.",
		Unit:    "ms", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state"},
		Query: `rate(mongodb_mongod_metrics_get_last_error_wtime_total_milliseconds{service_name="<service>"}[5m])`,
	},
	{
		Name: "mongodb_mongod_metrics_get_last_error_wtimeouts_total", Engine: engineMongoDB,
		Meaning: "getLastError write concern timeouts, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state"},
		Query: `rate(mongodb_mongod_metrics_get_last_error_wtimeouts_total{service_name="<service>"}[5m])`,
	},
	{
		Name: "mongodb_mongod_metrics_query_executor_total", Engine: engineMongoDB,
		Meaning: "Query executor work, per second, by state (scanned index keys, scanned_objects documents).",
		Unit:    "ops", Type: "counter", Labels: []string{"state"},
		Query: `sum by (state) (rate(mongodb_mongod_metrics_query_executor_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_mongod_metrics_ttl_deleted_documents_total", Engine: engineMongoDB,
		Meaning: "Documents deleted by TTL indexes, per second.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state"},
		Query: `rate(mongodb_mongod_metrics_ttl_deleted_documents_total{service_name="<service>"}[5m])`,
	},
	{
		Name: "mongodb_mongod_op_counters_repl_total", Engine: engineMongoDB,
		Meaning: "Operations replicated to this member, per second, by type.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "type"},
		Query: `sum by (type) (rate(mongodb_mongod_op_counters_repl_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_mongod_op_latencies_latency_total", Engine: engineMongoDB,
		Meaning: "Microseconds spent in operations, per second, by type; divide by the ops counter for the mean latency.",
		Unit:    "µs", Type: "counter", Labels: []string{"type"},
		Query: `sum by (type) (rate(mongodb_mongod_op_latencies_latency_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_mongod_op_latencies_ops_total", Engine: engineMongoDB,
		Meaning: "Operations measured for latency, per second, by type.",
		Unit:    "ops", Type: "counter", Labels: []string{"type"},
		Query: `sum by (type) (rate(mongodb_mongod_op_latencies_ops_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_mongod_replset_member_replication_lag", Engine: engineMongoDB,
		Meaning: "Seconds each replica set member is behind the primary, by member.",
		Unit:    "s", Type: "gauge", Labels: []string{"name", "self", "set", "state"},
		Query: `mongodb_mongod_replset_member_replication_lag{service_name="<service>"}`,
	},
	{
		Name: "mongodb_mongod_replset_my_state", Engine: engineMongoDB,
		Meaning: "Replica set state of this member: 1 PRIMARY, 2 SECONDARY, 7 ARBITER, others transitional.",
		Unit:    "none", Type: "gauge", Labels: []string{"set"},
		Query: `mongodb_mongod_replset_my_state{service_name="<service>"}`,
	},
	{
		Name: "mongodb_op_counters_total", Engine: engineMongoDB,
		Meaning: "Operations executed, per second, by type (insert, query, update, delete, getmore, command): the QPS.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "rs_nm", "rs_state", "type"},
		Query: `sum by (type) (rate(mongodb_op_counters_total{service_name="<service>"}[5m]))`,
	},
	{
		Name: "mongodb_ss_opcounters", Engine: engineMongoDB,
		Meaning: "Operations executed, per second, by type, from serverStatus opcounters.",
		Unit:    "ops", Type: "counter", Labels: []string{"cl_id", "cl_role", "legacy_op_type", "rs_nm", "rs_state"},
		Query: `sum by (legacy_op_type) (rate(mongodb_ss_opcounters{service_name="<service>"}[5m]))`,
	},
	{
		Name: "node_boot_time_seconds", Engine: engineNode,
		Meaning: "Unix time the node booted.",
		Unit:    "s", Type: "gauge",
		Query: `node_boot_time_seconds{node_name="<node>"}`,
	},
	{
		Name: "node_context_switches_total", Engine: engineNode,
		Meaning: "Context switches, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_context_switches_total{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_cpu_seconds_total", Engine: engineNode,
		Meaning: "CPU time used, in CPU-seconds per second (cores busy), by mode (user, system, iowait, idle and so on).",
		Unit:    "short", Type: "counter", Labels: []string{"cpu", "mode"},
		Query: `sum by (mode) (rate(node_cpu_seconds_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_disk_read_time_seconds_total", Engine: engineNode,
		Meaning: "Seconds spent reading, per second, by disk.",
		Unit:    "s", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_disk_read_time_seconds_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_disk_reads_completed_total", Engine: engineNode,
		Meaning: "Reads completed, per second, by disk.",
		Unit:    "ops", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_disk_reads_completed_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_disk_write_time_seconds_total", Engine: engineNode,
		Meaning: "Seconds spent writing, per second, by disk.",
		Unit:    "s", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_disk_write_time_seconds_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_disk_writes_completed_total", Engine: engineNode,
		Meaning: "Writes completed, per second, by disk.",
		Unit:    "ops", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_disk_writes_completed_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_filefd_allocated", Engine: engineNode,
		Meaning: "File descriptors allocated.",
		Unit:    "short", Type: "gauge",
		Query: `node_filefd_allocated{node_name="<node>"}`,
	},
	{
		Name: "node_filefd_maximum", Engine: engineNode,
		Meaning: "Maximum file descriptors.",
		Unit:    "short", Type: "gauge",
		Query: `node_filefd_maximum{node_name="<node>"}`,
	},
	{
		Name: "node_filesystem_free_bytes", Engine: engineNode,
		Meaning: "Free space in bytes by mount point, without in-memory and overlay file systems.",
		Unit:    "bytes", Type: "gauge", Labels: []string{"device", "device_error", "fstype", "mountpoint"},
		Query: `node_filesystem_free_bytes{node_name="<node>",fstype!~"rootfs|selinuxfs|autofs|rpc_pipefs|tmpfs|shm|overlay|squashfs"}`,
	},
	{
		Name: "node_filesystem_size_bytes", Engine: engineNode,
		Meaning: "Size in bytes by mount point, without in-memory and overlay file systems.",
		Unit:    "bytes", Type: "gauge", Labels: []string{"device", "device_error", "fstype", "mountpoint"},
		Query: `node_filesystem_size_bytes{node_name="<node>",fstype!~"rootfs|selinuxfs|autofs|rpc_pipefs|tmpfs|shm|overlay|squashfs"}`,
	},
	{
		Name: "node_forks_total", Engine: engineNode,
		Meaning: "Forks, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_forks_total{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_intr_total", Engine: engineNode,
		Meaning: "Interrupts serviced, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_intr_total{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_load1", Engine: engineNode,
		Meaning: "Load average over one minute.",
		Unit:    "short", Type: "gauge",
		Query: `node_load1{node_name="<node>"}`,
	},
	{
		Name: "node_memory_Buffers_bytes", Engine: engineNode,
		Meaning: "Memory in buffers.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_Buffers_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_Cached_bytes", Engine: engineNode,
		Meaning: "Memory in the page cache.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_Cached_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_MemAvailable_bytes", Engine: engineNode,
		Meaning: "Memory available for new work without swapping.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_MemAvailable_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_MemFree_bytes", Engine: engineNode,
		Meaning: "Memory not in use at all.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_MemFree_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_MemTotal_bytes", Engine: engineNode,
		Meaning: "Total memory.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_MemTotal_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_SReclaimable_bytes", Engine: engineNode,
		Meaning: "Reclaimable kernel slab memory.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_SReclaimable_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_Shmem_bytes", Engine: engineNode,
		Meaning: "Shared memory.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_Shmem_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_SwapFree_bytes", Engine: engineNode,
		Meaning: "Swap not in use.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_SwapFree_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_memory_SwapTotal_bytes", Engine: engineNode,
		Meaning: "Total swap.",
		Unit:    "bytes", Type: "gauge",
		Query: `node_memory_SwapTotal_bytes{node_name="<node>"}`,
	},
	{
		Name: "node_netstat_Tcp_OutSegs", Engine: engineNode,
		Meaning: "TCP segments sent, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_netstat_Tcp_OutSegs{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_netstat_Tcp_RetransSegs", Engine: engineNode,
		Meaning: "TCP segments retransmitted, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_netstat_Tcp_RetransSegs{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_network_receive_bytes_total", Engine: engineNode,
		Meaning: "Bytes received, per second, by interface.",
		Unit:    "Bps", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_network_receive_bytes_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_network_receive_drop_total", Engine: engineNode,
		Meaning: "Received packets dropped, per second, by interface.",
		Unit:    "ops", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_network_receive_drop_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_network_receive_errs_total", Engine: engineNode,
		Meaning: "Receive errors, per second, by interface.",
		Unit:    "ops", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_network_receive_errs_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_network_transmit_bytes_total", Engine: engineNode,
		Meaning: "Bytes sent, per second, by interface.",
		Unit:    "Bps", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_network_transmit_bytes_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_network_transmit_drop_total", Engine: engineNode,
		Meaning: "Sent packets dropped, per second, by interface.",
		Unit:    "ops", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_network_transmit_drop_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_network_transmit_errs_total", Engine: engineNode,
		Meaning: "Transmit errors, per second, by interface.",
		Unit:    "ops", Type: "counter", Labels: []string{"device"},
		Query: `sum by (device) (rate(node_network_transmit_errs_total{node_name="<node>"}[5m]))`,
	},
	{
		Name: "node_procs_blocked", Engine: engineNode,
		Meaning: "Processes blocked waiting for I/O.",
		Unit:    "short", Type: "gauge",
		Query: `node_procs_blocked{node_name="<node>"}`,
	},
	{
		Name: "node_procs_running", Engine: engineNode,
		Meaning: "Processes in the runnable state.",
		Unit:    "short", Type: "gauge",
		Query: `node_procs_running{node_name="<node>"}`,
	},
	{
		Name: "node_time_seconds", Engine: engineNode,
		Meaning: "Node's current Unix time.",
		Unit:    "s", Type: "gauge",
		Query: `node_time_seconds{node_name="<node>"}`,
	},
	{
		Name: "node_vmstat_pgpgin", Engine: engineNode,
		Meaning: "Data paged in from disk, per second, in KiB as the kernel reports it.",
		Unit:    "Bps", Type: "counter",
		Query: `rate(node_vmstat_pgpgin{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_vmstat_pgpgout", Engine: engineNode,
		Meaning: "Data paged out to disk, per second, in KiB as the kernel reports it.",
		Unit:    "Bps", Type: "counter",
		Query: `rate(node_vmstat_pgpgout{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_vmstat_pswpin", Engine: engineNode,
		Meaning: "Pages swapped in, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_vmstat_pswpin{node_name="<node>"}[5m])`,
	},
	{
		Name: "node_vmstat_pswpout", Engine: engineNode,
		Meaning: "Pages swapped out, per second.",
		Unit:    "ops", Type: "counter",
		Query: `rate(node_vmstat_pswpout{node_name="<node>"}[5m])`,
	},
}
