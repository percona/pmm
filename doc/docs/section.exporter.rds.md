# Amazon RDS Exporter (rds_exporter)

The Amazon RDS exporter makes the Amazon Cloudwatch metrics available to PMM. PMM uses this exporter to obtain metrics from any Amazon RDS node that you choose to monitor.

## Metrics

The Amazon RDS exporter has two types of metrics: basic and advanced. To be able to use advanced metrics, make sure to set the Enable Enhanced Monitoring option in the settings of your Amazon RDS DB instance.

![](_images/amazon-rds.modify-db-instance.2.png)

| rds_exporter Metric                                            | Amazon Cloudwatch Metric | Units | Type of Metric |
| -------------------                                            | ------------------------ | ----- | -------------- |
| aws_rds_bin_log_disk_usage_average                             | BinLogDiskUsage | Bytes | Basic |
| aws_rds_cpu_credit_balance_average                             | CPUCreditBalance | Credits (vCPU-minutes) | Basic |
| aws_rds_cpu_credit_usage_average                               | CPUCreditUsage | Credits (vCPU-minutes) | Basic |
| aws_rds_database_connections_average                           | DatabaseConnections | Count | Basic |
| aws_rds_disk_queue_depth_average                               | DiskQueueDepth | Count | Basic |
| aws_rds_network_receive_throughput_average                     | NetworkReceiveThroughput | Bytes per second | Basic |
| aws_rds_network_transmit_throughput_average                    | NetworkTransmitThroughput | Bytes per second | Basic |
| aws_rds_read_iops_average                                      | ReadIOPS | Count per second | Basic |
| aws_rds_read_latency_average                                   | ReadLatency | Seconds | Basic |
| aws_rds_read_throughput_average                                | ReadThroughput | Bytes per second | Basic |
| aws_rds_swap_usage_average                                     | SwapUsage | Bytes | Basic |
| aws_rds_write_iops_average                                     | WriteIOPS | Count per second | Basic |
| aws_rds_write_latency_average                                  | WriteLatency | Seconds | Basic |
| aws_rds_write_throughput_average                               | WriteThroughput | Bytes per second | Basic |
| node_cpu_average                                               | CPUUtilization | Percentage | Enhanced |
| node_filesystem_free                                           | FreeStorageSpace | Bytes | Enhanced |
| node_memory_Cached                                             | FreeableMemory | Bytes | Enhanced |
| rds_exporter_erroneous_requests                                | No corresponding Amazon Cloudwatch metric. The number of erroneous API requests made to CloudWatch. | Count | Enhanced |
| rds_exporter_requests_total                                    | No corresponding Amazon Cloudwatch metric. API requests made to Amazon Cloudwatch | Count | Enhanced |
| rds_exporter_scrape_duration_seconds                           | No corresponding Amazon Cloudwatch metric. The time that the current RDS scrape took. | Seconds | Enhanced |
| rds_latency                                                    | No corresponding Amazon Cloudwatch metric. The difference between the current time and timestamp in the metric itself. | Seconds | Enhanced |
| node_cp_average                                                | CPUUtilization | Percentage | Enhanced |
| node_load1                                                     | No corresponding Amazon Cloudwatch metric. The number of processes requesting CPU time over the last minute. | Count | Enhanced |
| node_memory_Active                                             | No corresponding Amazon Cloudwatch metric. The amount of assigned memory. | Kilobytes | Enhanced |
| node_memory_Buffers                                            | No corresponding Amazon Cloudwatch metric. The amount of memory used for buffering I/O requests prior to writing to the storage device | Kilobytes | Enhanced |
| node_memory_Cached                                             | No corresponding Amazon Cloudwatch metric. The amount of memory used for caching file system–based I/O. | Kilobytes | Enhanced |
| node_memory_Inactive                                           | No corresponding Amazon Cloudwatch metric. The amount of least-frequently used memory pages. | Kilobytes | Enhanced |
| node_memory_Mapped                                             | No corresponding Amazon Cloudwatch metric. The total amount of file-system contents that is memory mapped inside a process address space. | Kilobytes | Enhanced |
| node_memory_MemFree                                            | No corresponding Amazon Cloudwatch metric. The amount of unassigned memory. | Kilobytes | Enhanced |
| node_memory_MemTotal                                           | No corresponding Amazon Cloudwatch metric. The total amount of memory. | Kilobytes | Enhanced |
| node_memory_PageTables                                         | No corresponding Amazon Cloudwatch metric. The amount of memory used by page tables | Kilobytes | Enhanced |
| node_memory_Slab                                               | The amount of reusable kernel data structures | Kilobytes | Enhanced |
| node_memory_SwapFree                                           | No corresponding Amazon Cloudwatch metric. The total amount of swap memory free. | Kilobytes | Enhanced |
| node_memory_SwapTotal                                          | No corresponding Amazon Cloudwatch metric. The total amount of swap memory available. | Kilobytes | Enhanced |
| node_memory_nr_dirty                                           | No corresponding Amazon Cloudwatch metric. The amount of memory pages in RAM that have been modified but not written to their related data block in storage, | Kilobytes | Enhanced |
| node_procs_blocked                                             | No corresponding Amazon Cloudwatch metric. The number of tasks that are blocked. | Count | Enhanced |
| node_procs_running.                                            | No corresponding Amazon Cloudwatch metric.The number of tasks that are running. | Count | Enhanced |
| node_vmstat_pswpin                                             | No corresponding Amazon Cloudwatch metric. The number of kilobytes the system has swapped in from disk per second (disk reads). | Kilobytes | Enhanced |
| node_vmstat_pswpout                                            | No corresponding Amazon Cloudwatch metric. The number of kilobytes the system has swapped out to disk per second (disk writes). | Kilobytes | Enhanced |
| rds_exporter_erroneous_requests (Enhanced rds_exporter metric) | No corresponding Amazon Cloudwatch metric. The number of erroneous API request made to Amazon Cloudwatch. | Count | Enhanced |
| rds_exporter_requests_total                                    | No corresponding Amazon Cloudwatch metric. API requests made to Amazon Cloudwatch | Count | Enhanced |
| rds_exporter_scrape_duration_seconds                           | No corresponding Amazon Cloudwatch metric. The amount of time that this RDS scrape took. | Seconds | Enhanced |
| rds_latency                                                    | No corresponding Amazon Cloudwatch metric. The difference between the current time and timestamp in the metric itself. | Seconds | Enhanced |
| rdsosmetrics_General_numVCPUs                                  | No corresponding Amazon Cloudwatch metric. The number of virtual CPUs for the DB instance. | Count | Enhanced |
| rdsosmetrics_General_version                                   | No corresponding Amazon Cloudwatch metric. The version of the OS metrics stream JSON format. | Version number | Enhanced |
| rdsosmetrics_diskIO_await                                      | No corresponding Amazon Cloudwatch metric. The number of milliseconds required to respond to requests, including queue time and service time. This metric is not available for Amazon Aurora. | Milliseconds | Enhanced |
| rdsosmetrics_diskIO_tps                                        | No corresponding Amazon Cloudwatch metric. The number of I/O transactions per second. This metric is not available for Amazon Aurora. | Count per Second | Enhanced |
| rdsosmetrics_fileSys_maxFiles                                  | No corresponding Amazon Cloudwatch metric. The maximum number of files that can be created for the file system. | Count | Enhanced |
| rdsosmetrics_fileSys_usedFilePercent                           | No corresponding Amazon Cloudwatch metric. The percentage of available files in use. | Percentage | Enhanced |
| rdsosmetrics_loadAverageMinute_fifteen                         | No corresponding Amazon Cloudwatch metric. The number of processes requesting CPU time over the last 15 minutes. | Count | Enhanced |
| rdsosmetrics_loadAverageMinute_five                            | No corresponding Amazon Cloudwatch metric. The number of processes requesting CPU time over the last 5 minutes. | Count | Enhanced |
| rdsosmetrics_memory_hugePagesFree                              | No corresponding Amazon Cloudwatch metric. The number of free huge pages. Huge pages are a feature of the Linux kernel. | Count | Enhanced |
| rdsosmetrics_memory_hugePagesRsvd                              | No corresponding Amazon Cloudwatch metric. The number of committed huge pages. | Count | Enhanced |
| rdsosmetrics_memory_hugePagesSize                              | No corresponding Amazon Cloudwatch metric. The size for each huge pages unit. | Count | Enhanced |
| rdsosmetrics_memory_hugePagesSurp                              | No corresponding Amazon Cloudwatch metric. The number of available surplus huge pages over the total. | Count | Enhanced |
| rdsosmetrics_memory_hugePagesTotal                             | No corresponding Amazon Cloudwatch metric. The total number of huge pages for the system. | Count | Enhanced |
| rdsosmetrics_memory_writeback                                  | No corresponding Amazon Cloudwatch metric. The amount of dirty pages in RAM that are still being written to the backing storage. | Count | Enhanced |
| rdsosmetrics_processList_cpuUsedPc                             | No corresponding Amazon Cloudwatch metric. The percentage of CPU used by the process. | Percentage | Enhanced |
| rdsosmetrics_processList_id                                    | No corresponding Amazon Cloudwatch metric. The identifier of the process. | Process ID | Enhanced |
| rdsosmetrics_processList_parentID                              | No corresponding Amazon Cloudwatch metric. The process identifier for the parent process of the process. | Process ID | Enhanced |
| rdsosmetrics_processList_rss                                   | No corresponding Amazon Cloudwatch metric. The amount of RAM allocated to the process | Kilobytes | Enhanced |
| rdsosmetrics_processList_tgid                                  | No corresponding Amazon Cloudwatch metric. The thread group identifier, which is a number representing the process ID to which a thread belongs. This identifier is used to group threads from the same process. | Identifier | Enhanced |
| rdsosmetrics_processList_vss                                   | No corresponding Amazon Cloudwatch metric. The amount of virtual memory allocated to the process | Kilobytes | Enhanced |
| rdsosmetrics_swap_cached                                       | No corresponding Amazon Cloudwatch metric. The amount of swap memory used as cache memory. | Kilobytes | Enhanced |
| rdsosmetrics_tasks_sleeping                                    | No corresponding Amazon Cloudwatch metric. The number of tasks that are sleeping. | Count | Enhanced |
| rdsosmetrics_tasks_stopped                                     | No corresponding Amazon Cloudwatch metric. The number of tasks that are stopped. | Count | Enhanced |
| rdsosmetrics_tasks_total                                       | No corresponding Amazon Cloudwatch metric. The total number of tasks. | Count | Enhanced |
| rdsosmetrics_tasks_zombie                                      | No corresponding Amazon Cloudwatch metric. The number of child tasks that are inactive with an active parent task. | Count | Enhanced |
