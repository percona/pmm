SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'region' AS key, region AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY region UNION ALL SELECT 'region' AS key, region AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY region GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'application_name' AS key, application_name AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY application_name UNION ALL SELECT 'application_name' AS key, application_name AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY application_name GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'schema' AS key, schema AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY schema UNION ALL SELECT 'schema' AS key, schema AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY schema GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'replication_set' AS key, replication_set AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY replication_set UNION ALL SELECT 'replication_set' AS key, replication_set AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY replication_set GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'service_id' AS key, service_id AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY service_id UNION ALL SELECT 'service_id' AS key, service_id AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_id GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'node_model' AS key, node_model AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY node_model UNION ALL SELECT 'node_model' AS key, node_model AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY node_model GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'node_type' AS key, node_type AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY node_type UNION ALL SELECT 'node_type' AS key, node_type AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY node_type GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'service_name' AS key, service_name AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_type IN ('mongodb')) GROUP BY service_name UNION ALL SELECT 'service_name' AS key, service_name AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_name GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'username' AS key, username AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY username UNION ALL SELECT 'username' AS key, username AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY username GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'cluster' AS key, cluster AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY cluster UNION ALL SELECT 'cluster' AS key, cluster AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY cluster GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'service_type' AS key, service_type AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) GROUP BY service_type UNION ALL SELECT 'service_type' AS key, service_type AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_type GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'environment' AS key, environment AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY environment UNION ALL SELECT 'environment' AS key, environment AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY environment GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'az' AS key, az AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY az UNION ALL SELECT 'az' AS key, az AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY az GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'machine_id' AS key, machine_id AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY machine_id UNION ALL SELECT 'machine_id' AS key, machine_id AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY machine_id GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'container_id' AS key, container_id AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY container_id UNION ALL SELECT 'container_id' AS key, container_id AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY container_id GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC


SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'database' AS key, database AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY database UNION ALL SELECT 'database' AS key, database AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY database GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'top_queryid' AS key, top_queryid AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY top_queryid UNION ALL SELECT 'top_queryid' AS key, top_queryid AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY top_queryid GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'planid' AS key, planid AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY planid UNION ALL SELECT 'planid' AS key, planid AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY planid GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'cmd_type' AS key, cmd_type AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY cmd_type UNION ALL SELECT 'cmd_type' AS key, cmd_type AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY cmd_type GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'node_id' AS key, node_id AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY node_id UNION ALL SELECT 'node_id' AS key, node_id AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY node_id GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'node_name' AS key, node_name AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY node_name UNION ALL SELECT 'node_name' AS key, node_name AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY node_name GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'container_name' AS key, container_name AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY container_name UNION ALL SELECT 'container_name' AS key, container_name AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY container_name GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'plan_summary' AS key, plan_summary AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY plan_summary UNION ALL SELECT 'plan_summary' AS key, plan_summary AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY plan_summary GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC
SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'client_host' AS key, client_host AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) AND (service_type IN ('mongodb')) GROUP BY client_host UNION ALL SELECT 'client_host' AS key, client_host AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY client_host GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC

SELECT 'service_type' AS key, service_type AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  AND (service_name IN ('All', 'mysql-svc')) GROUP BY service_type UNION ALL SELECT 'service_type' AS key, service_type AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_type 
startFrom: 1744241330, startTo: 1744284530

SELECT key, value, sum(main_metric_sum) AS main_metric_sum FROM (SELECT 'service_type' AS key, service_type AS value, SUM(m_query_time_sum) AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_type)
UNION ALL 
SELECT 'service_type' AS key, service_type AS value, 0 AS main_metric_sum FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_type ) GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC 

SELECT key, value, sum(main_metric_sum) AS main_metric_sum 
  FROM (
    SELECT 'service_type' AS key, service_type AS value, SUM(m_query_time_sum) AS main_metric_sum 
      FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)  GROUP BY service_type
    UNION ALL 
    SELECT 'service_type' AS key, service_type AS value, 0 AS main_metric_sum 
      FROM pmm.metrics WHERE (period_start >= 1744241330) AND (period_start <= 1744284530) GROUP BY service_type
  )
GROUP BY key, value WITH TOTALS ORDER BY main_metric_sum DESC, value ASC;

SELECT
	DISTINCT ON (labels.key, labels.value)
FROM pmm.metrics
WHERE (period_start >= 1744241330) AND (period_start <= 1744284530)
ORDER BY
	labels.value ASC;

CREATE TABLE pmm.metrics (
  `queryid` LowCardinality(String) COMMENT 'hash of query fingerprint', 
  `service_name` LowCardinality(String) COMMENT 'Name of service (IP or hostname of DB server by default)', 
  `database` LowCardinality(String) COMMENT 'PostgreSQL: database', 
  `schema` LowCardinality(String) COMMENT 'MySQL: database; PostgreSQL: schema', 
  `username` LowCardinality(String) COMMENT 'client user name', 
  `client_host` LowCardinality(String) COMMENT 'client IP or hostname', 
  `replication_set` LowCardinality(String) COMMENT 'Name of replication set', 
  `cluster` LowCardinality(String) COMMENT 'Cluster name', 
  `service_type` LowCardinality(String) COMMENT 'Type of service', 
  `service_id` LowCardinality(String) COMMENT 'Service identifier', 
  `environment` LowCardinality(String) COMMENT 'Environment name', 
  `az` LowCardinality(String) COMMENT 'Availability zone', 
  `region` LowCardinality(String) COMMENT 'Region name', 
  `node_model` LowCardinality(String) COMMENT 'Node model', 
  `node_id` LowCardinality(String) COMMENT 'Node identifier', 
  `node_name` LowCardinality(String) COMMENT 'Node name', 
  `node_type` LowCardinality(String) COMMENT 'Node type', 
  `machine_id` LowCardinality(String) COMMENT 'Machine identifier', 
  `container_name` LowCardinality(String) COMMENT 'Container name', 
  `container_id` LowCardinality(String) COMMENT 'Container identifier', 
  `labels.key` Array(LowCardinality(String)) COMMENT 'Custom labels names', 
  `labels.value` Array(LowCardinality(String)) COMMENT 'Custom labels values', 
  `agent_id` LowCardinality(String) COMMENT 'Identifier of agent that collect and send metrics', 
  `agent_type` Enum8('qan-agent-type-invalid' = 0, 'qan-mysql-perfschema-agent' = 1, 'qan-mysql-slowlog-agent' = 2, 'qan-mongodb-profiler-agent' = 3, 'qan-postgresql-pgstatements-agent' = 4, 'qan-postgresql-pgstatmonitor-agent' = 5) COMMENT 'Agent Type that collects metrics: slowlog, perf schema, etc.', 
  `period_start` DateTime COMMENT 'Time when collection of bucket started', 
  `period_length` UInt32 COMMENT 'Duration of collection bucket', 
  `fingerprint` LowCardinality(String) COMMENT 'mysql digest_text; query without data', 
  `example` String COMMENT 'One of query example from set found in bucket', 
  `is_truncated` UInt8 COMMENT 'Indicates if query examples is too long and was truncated', 
  `example_type` Enum8('EXAMPLE_TYPE_INVALID' = 0, 'RANDOM' = 1, 'SLOWEST' = 2, 'FASTEST' = 3, 'WITH_ERROR' = 4) COMMENT 'Indicates what query example was picked up', 
  `example_metrics` String COMMENT 'Metrics of query example in JSON format.', 
  `num_queries_with_warnings` Float32 COMMENT 'How many queries was with warnings in bucket', 
  `warnings.code` Array(UInt32) COMMENT 'List of warnings', 
  `warnings.count` Array(Float32) COMMENT 'Count of each warnings in bucket', 
  `num_queries_with_errors` Float32 COMMENT 'How many queries was with error in bucket', 
  `errors.code` Array(UInt64) COMMENT 'List of Last_errno', 
  `errors.count` Array(UInt64) COMMENT 'Count of each Last_errno in bucket', 
  `num_queries` Float32 COMMENT 'Amount queries in this bucket', 
  `m_query_time_cnt` Float32 COMMENT 'The statement execution time in seconds was met.', 
  `m_query_time_sum` Float32 COMMENT 'The statement execution time in seconds.', 
  `m_query_time_min` Float32 COMMENT 'Smallest value of query_time in bucket', 
  `m_query_time_max` Float32 COMMENT 'Biggest value of query_time in bucket', 
  `m_query_time_p99` Float32 COMMENT '99 percentile of value of query_time in bucket', 
  `m_lock_time_cnt` Float32, 
  `m_lock_time_sum` Float32 COMMENT 'The time to acquire locks in seconds.', 
  `m_lock_time_min` Float32, 
  `m_lock_time_max` Float32, 
  `m_lock_time_p99` Float32, 
  `m_rows_sent_cnt` Float32, 
  `m_rows_sent_sum` Float32 COMMENT 'The number of rows sent to the client.', 
  `m_rows_sent_min` Float32, 
  `m_rows_sent_max` Float32, 
  `m_rows_sent_p99` Float32, 
  `m_rows_examined_cnt` Float32, 
  `m_rows_examined_sum` Float32 COMMENT 'Number of rows scanned - SELECT.', 
  `m_rows_examined_min` Float32, 
  `m_rows_examined_max` Float32, 
  `m_rows_examined_p99` Float32, 
  `m_rows_affected_cnt` Float32, 
  `m_rows_affected_sum` Float32 COMMENT 'Number of rows changed - UPDATE, DELETE, INSERT.', 
  `m_rows_affected_min` Float32, 
  `m_rows_affected_max` Float32, 
  `m_rows_affected_p99` Float32, 
  `m_rows_read_cnt` Float32, 
  `m_rows_read_sum` Float32 COMMENT 'The number of rows read from tables.', 
  `m_rows_read_min` Float32, 
  `m_rows_read_max` Float32, 
  `m_rows_read_p99` Float32, 
  `m_merge_passes_cnt` Float32, 
  `m_merge_passes_sum` Float32 COMMENT 'The number of merge passes that the sort algorithm has had to do.', 
  `m_merge_passes_min` Float32, 
  `m_merge_passes_max` Float32, 
  `m_merge_passes_p99` Float32, 
  `m_innodb_io_r_ops_cnt` Float32, 
  `m_innodb_io_r_ops_sum` Float32 COMMENT 'Counts the number of page read operations scheduled.', 
  `m_innodb_io_r_ops_min` Float32, 
  `m_innodb_io_r_ops_max` Float32, 
  `m_innodb_io_r_ops_p99` Float32, 
  `m_innodb_io_r_bytes_cnt` Float32, 
  `m_innodb_io_r_bytes_sum` Float32 COMMENT 'Similar to innodb_IO_r_ops, but the unit is bytes.', 
  `m_innodb_io_r_bytes_min` Float32, 
  `m_innodb_io_r_bytes_max` Float32, 
  `m_innodb_io_r_bytes_p99` Float32, 
  `m_innodb_io_r_wait_cnt` Float32, 
  `m_innodb_io_r_wait_sum` Float32 COMMENT 'Shows how long (in seconds) it took InnoDB to actually read the data from storage.', 
  `m_innodb_io_r_wait_min` Float32, 
  `m_innodb_io_r_wait_max` Float32, 
  `m_innodb_io_r_wait_p99` Float32, 
  `m_innodb_rec_lock_wait_cnt` Float32, 
  `m_innodb_rec_lock_wait_sum` Float32 COMMENT 'Shows how long (in seconds) the query waited for row locks.', 
  `m_innodb_rec_lock_wait_min` Float32, 
  `m_innodb_rec_lock_wait_max` Float32, 
  `m_innodb_rec_lock_wait_p99` Float32, 
  `m_innodb_queue_wait_cnt` Float32, 
  `m_innodb_queue_wait_sum` Float32 COMMENT 'Shows how long (in seconds) the query spent either waiting to enter the InnoDB queue or inside that queue waiting for execution.', 
  `m_innodb_queue_wait_min` Float32, 
  `m_innodb_queue_wait_max` Float32, 
  `m_innodb_queue_wait_p99` Float32, 
  `m_innodb_pages_distinct_cnt` Float32, 
  `m_innodb_pages_distinct_sum` Float32 COMMENT 'Counts approximately the number of unique pages the query accessed.', 
  `m_innodb_pages_distinct_min` Float32, 
  `m_innodb_pages_distinct_max` Float32, 
  `m_innodb_pages_distinct_p99` Float32, 
  `m_query_length_cnt` Float32, 
  `m_query_length_sum` Float32 COMMENT 'Shows how long the query is.', 
  `m_query_length_min` Float32, 
  `m_query_length_max` Float32, 
  `m_query_length_p99` Float32, 
  `m_bytes_sent_cnt` Float32, 
  `m_bytes_sent_sum` Float32 COMMENT 'The number of bytes sent to all clients.', 
  `m_bytes_sent_min` Float32, 
  `m_bytes_sent_max` Float32, 
  `m_bytes_sent_p99` Float32, 
  `m_tmp_tables_cnt` Float32, 
  `m_tmp_tables_sum` Float32 COMMENT 'Number of temporary tables created on memory for the query.', 
  `m_tmp_tables_min` Float32, 
  `m_tmp_tables_max` Float32, 
  `m_tmp_tables_p99` Float32, 
  `m_tmp_disk_tables_cnt` Float32, 
  `m_tmp_disk_tables_sum` Float32 COMMENT 'Number of temporary tables created on disk for the query.', 
  `m_tmp_disk_tables_min` Float32, 
  `m_tmp_disk_tables_max` Float32, 
  `m_tmp_disk_tables_p99` Float32, 
  `m_tmp_table_sizes_cnt` Float32, 
  `m_tmp_table_sizes_sum` Float32 COMMENT 'Total Size in bytes for all temporary tables used in the query.', 
  `m_tmp_table_sizes_min` Float32, 
  `m_tmp_table_sizes_max` Float32, 
  `m_tmp_table_sizes_p99` Float32, 
  `m_qc_hit_cnt` Float32, 
  `m_qc_hit_sum` Float32 COMMENT 'Query Cache hits.', 
  `m_full_scan_cnt` Float32, 
  `m_full_scan_sum` Float32 COMMENT 'The query performed a full table scan.', 
  `m_full_join_cnt` Float32, 
  `m_full_join_sum` Float32 COMMENT 'The query performed a full join (a join without indexes).', 
  `m_tmp_table_cnt` Float32, 
  `m_tmp_table_sum` Float32 COMMENT 'The query created an implicit internal temporary table.', 
  `m_tmp_table_on_disk_cnt` Float32, 
  `m_tmp_table_on_disk_sum` Float32 COMMENT 'The querys temporary table was stored on disk.', 
  `m_filesort_cnt` Float32, 
  `m_filesort_sum` Float32 COMMENT 'The query used a filesort.', 
  `m_filesort_on_disk_cnt` Float32, 
  `m_filesort_on_disk_sum` Float32 COMMENT 'The filesort was performed on disk.', 
  `m_select_full_range_join_cnt` Float32, 
  `m_select_full_range_join_sum` Float32 COMMENT 'The number of joins that used a range search on a reference table.', 
  `m_select_range_cnt` Float32, 
  `m_select_range_sum` Float32 COMMENT 'The number of joins that used ranges on the first table.', 
  `m_select_range_check_cnt` Float32, 
  `m_select_range_check_sum` Float32 COMMENT 'The number of joins without keys that check for key usage after each row.', 
  `m_sort_range_cnt` Float32, 
  `m_sort_range_sum` Float32 COMMENT 'The number of sorts that were done using ranges.', 
  `m_sort_rows_cnt` Float32, 
  `m_sort_rows_sum` Float32 COMMENT 'The number of sorted rows.', 
  `m_sort_scan_cnt` Float32, 
  `m_sort_scan_sum` Float32 COMMENT 'The number of sorts that were done by scanning the table.', 
  `m_no_index_used_cnt` Float32, 
  `m_no_index_used_sum` Float32 COMMENT 'The number of queries without index.', 
  `m_no_good_index_used_cnt` Float32, 
  `m_no_good_index_used_sum` Float32 COMMENT 'The number of queries without good index.', 
  `m_docs_returned_cnt` Float32, 
  `m_docs_returned_sum` Float32 COMMENT 'The number of returned documents.', 
  `m_docs_returned_min` Float32, 
  `m_docs_returned_max` Float32, 
  `m_docs_returned_p99` Float32, 
  `m_response_length_cnt` Float32, 
  `m_response_length_sum` Float32 COMMENT 'The response length of the query result in bytes.', 
  `m_response_length_min` Float32, 
  `m_response_length_max` Float32, 
  `m_response_length_p99` Float32, 
  `m_docs_scanned_cnt` Float32, 
  `m_docs_scanned_sum` Float32 COMMENT 'The number of scanned documents.', 
  `m_docs_scanned_min` Float32, 
  `m_docs_scanned_max` Float32, 
  `m_docs_scanned_p99` Float32, 
  `m_shared_blks_hit_cnt` Float32, 
  `m_shared_blks_hit_sum` Float32 COMMENT 'Total number of shared blocks cache hits by the statement', 
  `m_shared_blks_read_cnt` Float32, 
  `m_shared_blks_read_sum` Float32 COMMENT 'Total number of shared blocks read by the statement.', 
  `m_shared_blks_dirtied_cnt` Float32, 
  `m_shared_blks_dirtied_sum` Float32 COMMENT 'Total number of shared blocks dirtied by the statement.', 
  `m_shared_blks_written_cnt` Float32, 
  `m_shared_blks_written_sum` Float32 COMMENT 'Total number of shared blocks written by the statement.', 
  `m_local_blks_hit_cnt` Float32, 
  `m_local_blks_hit_sum` Float32 COMMENT 'Total number of local block cache hits by the statement', 
  `m_local_blks_read_cnt` Float32, 
  `m_local_blks_read_sum` Float32 COMMENT 'Total number of local blocks read by the statement.', 
  `m_local_blks_dirtied_cnt` Float32, 
  `m_local_blks_dirtied_sum` Float32 COMMENT 'Total number of local blocks dirtied by the statement.', 
  `m_local_blks_written_cnt` Float32, 
  `m_local_blks_written_sum` Float32 COMMENT 'Total number of local blocks written by the statement.', 
  `m_temp_blks_read_cnt` Float32, 
  `m_temp_blks_read_sum` Float32 COMMENT 'Total number of temp blocks read by the statement.', 
  `m_temp_blks_written_cnt` Float32, 
  `m_temp_blks_written_sum` Float32 COMMENT 'Total number of temp blocks written by the statement.', 
  `m_shared_blk_read_time_cnt` Float32, 
  `m_shared_blk_read_time_sum` Float32 COMMENT 'Total time the statement spent reading shared blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).', 
  `m_shared_blk_write_time_cnt` Float32, 
  `m_shared_blk_write_time_sum` Float32 COMMENT 'Total time the statement spent writing shared blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).', 
  `tables` Array(String), 
  `m_cpu_user_time_cnt` Float32, 
  `m_cpu_user_time_sum` Float32 COMMENT 'Total time user spent in query', 
  `m_cpu_sys_time_cnt` Float32, 
  `m_cpu_sys_time_sum` Float32 COMMENT 'Total time system spent in query', 
  `m_plans_calls_cnt` Float32, 
  `m_plans_calls_sum` Float32 COMMENT 'Total number of planned calls', 
  `m_wal_records_cnt` Float32, 
  `m_wal_records_sum` Float32 COMMENT 'Total number of WAL (Write-ahead logging) records', 
  `m_wal_fpi_cnt` Float32, 
  `m_wal_fpi_sum` Float32 COMMENT 'Total number of FPI (full page images) in WAL (Write-ahead logging) records', 
  `m_wal_bytes_cnt` Float32, 
  `m_wal_bytes_sum` Float32 COMMENT 'Total bytes of WAL (Write-ahead logging) records', 
  `m_plan_time_cnt` Float32 COMMENT 'Count of plan time.', 
  `m_plan_time_sum` Float32 COMMENT 'Sum of plan time.', 
  `m_plan_time_min` Float32 COMMENT 'Min of plan time.', 
  `m_plan_time_max` Float32 COMMENT 'Max of plan time.', 
  `top_queryid` LowCardinality(String), 
  `application_name` LowCardinality(String), 
  `planid` LowCardinality(String), 
  `cmd_type` LowCardinality(String), 
  `top_query` LowCardinality(String), 
  `query_plan` LowCardinality(String), 
  `histogram_items` Array(String), 
  `explain_fingerprint` String, 
  `placeholders_count` UInt32, 
  `m_local_blk_read_time_cnt` Float32, 
  `m_local_blk_read_time_sum` Float32 COMMENT 'Total time the statement spent reading local blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).', 
  `m_local_blk_write_time_cnt` Float32, 
  `m_local_blk_write_time_sum` Float32 COMMENT 'Total time the statement spent writing local blocks, in milliseconds (if track_io_timing is enabled, otherwise zero).', 
  `plan_summary` LowCardinality(String), 
  `m_docs_examined_cnt` Float32, 
  `m_docs_examined_sum` Float32 COMMENT 'Total number of documents scanned during query execution', 
  `m_docs_examined_min` Float32, 
  `m_docs_examined_max` Float32, 
  `m_docs_examined_p99` Float32, 
  `m_keys_examined_cnt` Float32, 
  `m_keys_examined_sum` Float32 COMMENT 'Total number of index keys scanned during query execution', 
  `m_keys_examined_min` Float32, 
  `m_keys_examined_max` Float32, 
  `m_keys_examined_p99` Float32, 
  `m_locks_global_acquire_count_read_shared_cnt` Float32, 
  `m_locks_global_acquire_count_read_shared_sum` Float32 COMMENT 'Number of times a global read lock was acquired during query execution', 
  `m_locks_global_acquire_count_write_shared_cnt` Float32, 
  `m_locks_global_acquire_count_write_shared_sum` Float32 COMMENT 'Number of times a global write lock was acquired during query execution', 
  `m_locks_database_acquire_count_read_shared_cnt` Float32, 
  `m_locks_database_acquire_count_read_shared_sum` Float32 COMMENT 'Number of times a read lock was acquired at the database level during query execution', 
  `m_locks_database_acquire_wait_count_read_shared_cnt` Float32, 
  `m_locks_database_acquire_wait_count_read_shared_sum` Float32 COMMENT 'Number of times a read lock at the database level was requested but had to wait before being granted', 
  `m_locks_database_time_acquiring_micros_read_shared_cnt` Float32, 
  `m_locks_database_time_acquiring_micros_read_shared_sum` Float32 COMMENT 'Indicates the time, spent acquiring a read lock at the database level during an operation', 
  `m_locks_database_time_acquiring_micros_read_shared_min` Float32, 
  `m_locks_database_time_acquiring_micros_read_shared_max` Float32, 
  `m_locks_database_time_acquiring_micros_read_shared_p99` Float32, 
  `m_locks_collection_acquire_count_read_shared_cnt` Float32, 
  `m_locks_collection_acquire_count_read_shared_sum` Float32 COMMENT 'Number of times a read lock was acquired on a specific collection during operations', 
  `m_storage_bytes_read_cnt` Float32, 
  `m_storage_bytes_read_sum` Float32 COMMENT 'Total number of bytes read from storage during a specific operation', 
  `m_storage_bytes_read_min` Float32, 
  `m_storage_bytes_read_max` Float32, 
  `m_storage_bytes_read_p99` Float32, 
  `m_storage_time_reading_micros_cnt` Float32, 
  `m_storage_time_reading_micros_sum` Float32 COMMENT 'Indicates the time, spent reading data from storage during an operation', 
  `m_storage_time_reading_micros_min` Float32, 
  `m_storage_time_reading_micros_max` Float32, 
  `m_storage_time_reading_micros_p99` Float32 
) ENGINE = MergeTree 
PARTITION BY toYYYYMMDD(period_start) 
ORDER BY (queryid, service_name, database, schema, username, client_host, period_start) 
SETTINGS index_granularity = 8192;


SELECT DISTINCT labels.key, labels.value 
FROM pmm.metrics 
WHERE (period_start >= 1744316725) AND (period_start <= 1744359925) 
  AND match(service_type, 'mysql|mongodb') 
  AND environment = 'dev' 
  AND NOT match(environment, 'prod') 
  AND az != 'us-east-1' 
ORDER BY labels.value ASC;


SELECT 
  queryid AS dimension,
  any(database) as database_name,
  any(fingerprint) AS fingerprint,
  SUM(num_queries) AS num_queries,
  SUM(m_query_time_cnt) AS m_query_time_cnt,
  SUM(m_query_time_sum) AS m_query_time_sum,
  MIN(m_query_time_min) AS m_query_time_min,
  MAX(m_query_time_max) AS m_query_time_max,
  AVG(m_query_time_p99) AS m_query_time_p99,
  m_query_time_sum/num_queries AS m_query_time_avg,
  m_query_time_sum / 21600 AS load,
  count(DISTINCT dimension) AS total_rows 
FROM metrics 
WHERE period_start >= 1744802362
  AND period_start <= 1744823962
  AND service_type IN ( 'All', 'postgresql' ) 
  AND (
    match(service_type, 'postgresql') 
    OR (
      match(service_type, 'mysql|mongodb') 
      AND match(environment, 'dev|prod') 
      AND NOT (hasAny(labels.key, ['source']) AND arrayExists(x -> match(x, 'slowlog'), labels.value))
    )
  ) 
GROUP BY queryid WITH TOTALS
ORDER BY m_query_time_sum DESC
LIMIT 0, 25
-- args: [1744802362 1744823962 0 25]  