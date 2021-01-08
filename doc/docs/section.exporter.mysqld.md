# MySQL Server Exporter (mysqld_exporter)

**mysqld_exporter** is the Prometheus exporter for MySQL metrics. This exporter has three resolutions to group the metrics:

* metrics-hr (metrics with a high resolution) uses the default Prometheus scrape interval
* metrics-mr (metrics with a medium resolution) scrapes every 5 seconds
* metrics-lr (metrics with a low resolution) scrapes every 60 seconds

For example, *metrics-hr* contains very frequently changing values, such as **mysql_global_status_commands_total**.

On the other hand, *metrics-lr* contains infrequently changing values such as **mysql_global_variables_autocommit**.

The following options may be passed to the `mysql:metrics` monitoring service as additional options.

## Collector options

| Name                                                   | MySQL Version | Description |
| ------------------------------------------------------ | ------------- | ----------- |
| collect.auto_increment.columns                         | 5.1           | Collect auto_increment columns and max values from information_schema. |
| collect.binlog_size                                    | 5.1           | Collect the current size of all registered binlog files |
| collect.engine_innodb_status                           | 5.1           | Collect from SHOW ENGINE INNODB STATUS. |
| collect.engine_tokudb_status                           | 5.6           | Collect from SHOW ENGINE TOKUDB STATUS. |
| collect.global_status                                  | 5.1           | Collect from SHOW GLOBAL STATUS (Enabled by default) |
| collect.global_variables                               | 5.1           | Collect from SHOW GLOBAL VARIABLES (Enabled by default) |
| collect.info_schema.clientstats                        | 5.5           | If running with userstat=1, set to true to collect client statistics. |
| collect.info_schema.innodb_metrics                     | 5.6           | Collect metrics from information_schema.innodb_metrics. |
| collect.info_schema.innodb_tablespaces                 | 5.7           | Collect metrics from information_schema.innodb_sys_tablespaces. |
| collect.info_schema.processlist                        | 5.1           | Collect thread state counts from information_schema.processlist. |
| collect.info_schema.processlist.min_time               | 5.1           | Minimum time a thread must be in each state to be counted. (default: 0) |
| collect.info_schema.query_response_time                | 5.5           | Collect query response time distribution if query_response_time_stats is ON. |
| collect.info_schema.tables                             | 5.1           | Collect metrics from information_schema.tables (Enabled by default) |
| collect.info_schema.tables.databases                   | 5.1           | The list of databases to collect table stats for, or ‘\*’ for all. |
| collect.info_schema.tablestats                         | 5.1           | If running with userstat=1, set to true to collect table statistics. |
| collect.info_schema.userstats                          | 5.1           | If running with userstat=1, set to true to collect user statistics. |
| collect.perf_schema.eventsstatements                   | 5.6           | Collect metrics from performance_schema.events_statements_summary_by_digest. |
| collect.perf_schema.eventsstatements.digest_text_limit | 5.6           | Maximum length of the normalized statement text. (default: 120) |
| collect.perf_schema.eventsstatements.limit             | 5.6           | Limit the number of events statements digests by response time. (default: 250) |
| collect.perf_schema.eventsstatements.timelimit         | 5.6           | Limit how old the ‘last_seen’ events statements can be, in seconds. (default: 86400) |
| collect.perf_schema.eventswaits                        | 5.5           | Collect metrics from performance_schema.events_waits_summary_global_by_event_name. |
| collect.perf_schema.file_events                        | 5.6           | Collect metrics from performance_schema.file_summary_by_event_name. |
| collect.perf_schema.file_instances                     | 5.5           | Collect metrics from performance_schema.file_summary_by_instance. |
| collect.perf_schema.indexiowaits                       | 5.6           | Collect metrics from performance_schema.table_io_waits_summary_by_index_usage. |
| collect.perf_schema.tableiowaits                       | 5.6           | Collect metrics from performance_schema.table_io_waits_summary_by_table. |
| collect.perf_schema.tablelocks                         | 5.6           | Collect metrics from performance_schema.table_lock_waits_summary_by_table. |
| collect.slave_status                                   | 5.1           | Collect from SHOW SLAVE STATUS (Enabled by default) |
| collect.heartbeat                                      | 5.1           | Collect from [heartbeat](#heartbeat). |
| collect.heartbeat.database                             | 5.1           | Database from where to collect heartbeat data. (default: heartbeat) |
| collect.heartbeat.table                                | 5.1           | Table from where to collect heartbeat data. (default: heartbeat) |


## General options

| Name               | Description |
| ------------------ | -------------------------------------------------------------------- |
| config.my-cnf      | Path to .my.cnf file to read MySQL credentials from. (default: ~/.my.cnf) |
| log.level          | Logging verbosity (default: info) |
| log_slow_filter    | Add a log_slow_filter to avoid exessive MySQL slow logging.  NOTE: Not supported by Oracle MySQL. |
| web.listen-address | Address to listen on for web interface and telemetry. |
| web.telemetry-path | Path under which to expose metrics. |
| version            | Print the version information. |
