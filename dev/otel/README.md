# Logging Functionality in Percona Monitoring and Management (PMM)

## Goals
While the overarching goal it provide a robust logging system that allows users to monitor and troubleshoot their database environments effectively, the specific requirements for the logging functionality in PMM are as follows:
- ability to collect logs from various sources and in various formats, including logs generated within PMM
- integrate with PMM and provide an interface for viewing anf querying the logs
- support log retention policies to manage disk space
- support alerting based on log events and notify users via available channels (e.g., email, slack, etc.)
- support a developer-friendly architecture that allows for easy extension and modification
- ability to scale with the growing needs of users and handle large volumes of log data efficiently
- stay true to open source principles

## Overview
Percona Monitoring and Management (PMM) provides a robust logging system that allows users to monitor and troubleshoot their database environments effectively. This document outlines the logging functionality available in PMM, including how to configure, view, and manage logs.

## Architecture
PMM's logging architecture is designed to extract logs produced by various components, be they internal or external to PMM. The logs are collected, processed and then persisted to facilitate easy searching and filtering, making it easier for users to identify issues and monitor system health.

### Architecture Diagrams
These architecture diagrams illustrate the flow of logs from various sources through the OpenTelemetry Collector to the PMM server, where they are stored in ClickHouse and visualized in Grafana.

![PMM Logging Architecture](/dev/otel/doc/otel-collector.png)
![PMM Log Collection Diagram](/dev/otel/doc/PMM%20Log%20Collection%20Diagram.jpg)
![PMM Logging Diagram - Internal Storage](/dev/otel/doc/PMM%20Logging%20Diagram%20-%20Internal%20Storage.jpg)
![PMM Logging Diagram - External Storage](/dev/otel/doc/PMM%20Logging%20Diagram%20-%20External%20Storage.jpg)

###  Logging Components
PMM's logging functionality consists of several key components:
- **PMM Server**: The central component that collects and stores logs from various sources.
- **ClickHouse**: The underlying storage system where logs are stored, which can be local or remote.
- **Open Telemetry (Otel) Collector**: Collector agents installed on systems that gather, process and send logs to PMM server.
- **Grafana**: A user interface component that allows users to view and search logs persisted to PMM.
- **Clickhouse Datasource**: A Grafana-authorded ClickHouse datasource used to visualize and query logs.

## Features
- **Centralized Logging**: Collects logs from multiple sources into a single system for easier management and analysis.
- **Log Levels**: Supports multiple log levels (debug, info, warn, error, fatal) to allow the user to filter through log severity.
- **Log Retention**: Defined the log lifetime duration and automatically drops stale records to save on disk space.
- **Scalability**: Built on ClickHouse, which can handle large volumes of log data efficiently.
- **Integration with Grafana**: Provides a powerful visualization layer for logs, enabling users to create custom dashboards and alerts.
- **Flexible Configuration**: Allows users to customize log collection, processing, and storage according to their needs.
- **Alerting Capabilities**: Integrates with Grafana's alerting system to notify users of critical log events.
- **Search and Filter**: Provides a user-friendly interface for searching and filtering logs, making it accessible even for non-technical users.
- **OpenTelemetry Compliance**: Follows OpenTelemetry standards, ensuring compatibility with a wide range of logging tools and services.
- **Dev friendly**: The architecture is designed to be easily extendable, allowing developers to add new log sources or modify existing configurations without significant overhead. The change-deploy-test cycle is streamlined, enabling rapid iteration and testing of new log sources or configurations in a local dev environment.


## Database Schema
PMM uses a structured schema to store logs in ClickHouse. The schema follows the OpenTelemetry recommendations for log data, ensuring compatibility and ease of use. You can find the schema definition in the PMM documentation or read more about it in the [OpenTelemetry documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/clickhouseexporter/example/default_ddl/logs.sql).

The database creation is managed by ClickHouse's `startup_scripts` functionality, which automatically initializes the database and tables when PMM boots up. The schema includes fields for log messages, timestamps, severity levels, and other relevant metadata.

The table to store logs is named `logs`, and it is created in the `otel` database. The schema is automatically created by the collector when PMM server starts, ensuring that the necessary structure is in place for log storage.

The schema is optimized for efficient querying and indexing, allowing users to quickly retrieve logs based on various criteria. The table is partitioned by date to improve performance and manageability. It heavily utilizes ClickHouse's compression codecs to reduce storage requirements while maintaining fast access to log data.

The following SQL statement creates the `logs` table in the `otel` database:

```plaintext
CREATE TABLE otel.logs
(
    `Timestamp` DateTime64(9) CODEC(Delta(8), ZSTD(1)),
    `TimestampTime` DateTime DEFAULT toDateTime(Timestamp),
    `TraceId` String CODEC(ZSTD(1)),
    `SpanId` String CODEC(ZSTD(1)),
    `TraceFlags` UInt8,
    `SeverityText` LowCardinality(String) CODEC(ZSTD(1)),
    `SeverityNumber` UInt8,
    `ServiceName` LowCardinality(String) CODEC(ZSTD(1)),
    `Body` String CODEC(ZSTD(1)),
    `ResourceSchemaUrl` LowCardinality(String) CODEC(ZSTD(1)),
    `ResourceAttributes` Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    `ScopeSchemaUrl` LowCardinality(String) CODEC(ZSTD(1)),
    `ScopeName` String CODEC(ZSTD(1)),
    `ScopeVersion` LowCardinality(String) CODEC(ZSTD(1)),
    `ScopeAttributes` Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    `LogAttributes` Map(LowCardinality(String), String) CODEC(ZSTD(1)),
    INDEX idx_trace_id TraceId TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_res_attr_key mapKeys(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_res_attr_value mapValues(ResourceAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_scope_attr_key mapKeys(ScopeAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_scope_attr_value mapValues(ScopeAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_log_attr_key mapKeys(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_log_attr_value mapValues(LogAttributes) TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_body Body TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 8
)
ENGINE = MergeTree
PARTITION BY toDate(TimestampTime)
PRIMARY KEY (ServiceName, TimestampTime)
ORDER BY (ServiceName, TimestampTime, Timestamp)
TTL TimestampTime + toIntervalDay(3)
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1
```

## Logging Configuration

PMM's logging configuration is managed through a [YAML file](/dev/otel/config.yml) mounted as `/etc/otel/config.yaml` to `otel-collector` container. This file allows users to customize various aspects of logging, including log file locations, severity levels, output formats, in-flight transformations and more.

To read about the configuration options, refer to the [OpenTelemetry Collector Configuration](https://opentelemetry.io/docs/collector/configuration/) documentation. The configuration file is structured in a way that allows users to define receivers, processors, exporters, and other components that control how logs are collected, processed, and stored.

The list of available recievers, processors, exporters, etc can be found in the [builder-config.yaml](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/cmd/otelcontribcol/builder-config.yaml)

### Log Levels
PMM supports several log levels, which can be set in the configuration file:
- `debug`: Detailed information, typically of interest only when diagnosing problems.
- `info`: General information about the system's operation.
- `warn`: Events that may signal about a potential issue.
- `error`: Error events that might still allow the application to continue running.
- `fatal`: Severe errors that cause premature termination of the application.

It is a good practice to set the log level to `info` or `warn` in production environments to avoid excessive logging, while `debug` can be used during development or troubleshooting.

For detailed information about how these levels map to OpenTelemetry severity numbers, see the [OpenTelemetry Severity Numbers](#opentelemetry-severity-numbers) section below.


### Example Configuration
```yaml
logging:
  level: info
  file: /var/log/pmm-server.log
```

## Viewing Logs

### Accessing Logs
Logs can be accessed directly from the log file specified in the configuration. You can use standard command-line tools like `cat`, `less`, or `tail` to view the logs.
```bash
tail -f /var/log/pmm-server.log
```

## Log Management

### Log Retention
To manage log retention, PMM .

### Exporting Logs
PMM allows exporting logs to external systems for long-term storage or further analysis. You can configure the Otel exporter to send logs to a remote ClickHouse instance or other supported backends.

### Integration with Grafana
PMM integrates with Grafana to provide a user-friendly interface for viewing and analyzing logs. You can create custom dashboards to visualize log data, set up alerts based on log events, and use Grafana's powerful search capabilities to filter logs.

## Troubleshooting

### Raw database queries
You can run raw SQL queries against the ClickHouse database to retrieve logs. This is useful for advanced users who want to perform custom queries or analyze logs in detail. For example, you can run the following query to retrieve logs from the last 24 hours:

```sql
SELECT *
FROM otel.logs
WHERE TimestampTime >= now() - INTERVAL 1 DAY
ORDER BY TimestampTime DESC
```

To connect to ClickHouse, you can run `docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse -d otel`. This will leverage the ClickHouse client available from the PMM server container.

### Common Issues
- **Logs Not Appearing**: Ensure that the logging configuration is correctly set up and that the PMM server has permission to write to the specified log file.
- **Log Levels Not Working**: Verify that the log level is correctly set in the configuration file and that otel-collector has been restarted after making changes.
- **Log Rotation Issues**: Log rotation is the reponsibility of the underlying system where the logs get sourced from. OpenTelemetry Collector does not handle log rotation itself. Ensure that your system's log rotation settings are correctly configured to manage log file sizes and retention.

## Conclusion
PMM's logging functionality provides a comprehensive solution for monitoring and managing database environments. By leveraging the structured logging architecture, users can effectively troubleshoot issues, monitor system health, and gain insights into their database operations. Proper configuration and management of logs are essential for maintaining an efficient and reliable monitoring system.

## References
- [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/index.html)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [ClickHouse Documentation](https://clickhouse.com/docs/en/)
- [Grafana Documentation](https://grafana.com/docs/)

## Additional Notes

### OpenTelemetry Severity Numbers

PMM follows the **OpenTelemetry specification** for log severity levels, which uses the syslog RFC 5424 standard. The severity numbers may appear sparse, but this is intentional and provides several benefits:

#### Severity Level Mapping

| Level Name | Severity Number Range | Actual Number Used | Syslog Level | Description |
|------------|----------------------|-------------------|--------------|-------------|
| `TRACE` | 1-4 | - | - | Finest-grained debug info |
| `DEBUG` | 5-8 | 5 | Debug (7) | Debug information |
| `INFO` | 9-12 | 9 | Informational (6) | General information |
| `WARN` | 13-16 | 13 | Warning (4) | Warning conditions |
| `ERROR` | 17-20 | 17 | Error (3) | Error conditions |
| `FATAL` | 21-24 | 21 | Critical/Alert/Emergency (2/1/0) | System unusable |

#### Why Sparse Numbering?

1. **Granularity**: Each severity level gets a range of 4 numbers, allowing for sub-levels (e.g., INFO1=9, INFO2=10, INFO3=11, INFO4=12)
2. **Backward Compatibility**: Matches the well-established syslog standard used for decades
3. **Future Extensions**: Leaves room for new severity levels between existing ones
4. **Industry Standard**: Compatible with most logging systems (rsyslog, journald, etc.)
5. **Interoperability**: Works seamlessly with monitoring tools like Grafana and Prometheus

#### HTTP Status Code to Severity Mapping

In PMM's web server log processing, HTTP status codes are automatically mapped to appropriate severity levels:

- **2xx Status Codes** → `INFO` (SeverityNumber: 9) - Successful requests (e.g., 200, 201, 202, 301, 303, etc.)
- **4xx Status Codes** → `WARN` (SeverityNumber: 13) - Client errors (e.g., 404, 403)
- **5xx Status Codes** → `ERROR` (SeverityNumber: 17) - Server errors (e.g., 500, 502, 503)

This mapping ensures that log severity accurately reflects the nature of each request and helps with monitoring and alerting.

### Comparison with Other Systems
Previously, we have explored other logging systems like Grafana Loki and VictoriaLogs. While these systems offer similar functionalities, the current solution using OpenTelemetry Collector and ClickHouse provides several advantages:

- **Disk Usage**: ClickHouse's columnar storage is very efficient in storing large volumes of log data and comsumes minimal disk space compared to other systems.
- **Performance**: ClickHouse is optimized for high-performance queries, making it suitable for real
- **Scalability**: The architecture is designed to handle high log volumes efficiently, making it suitable for large-scale deployments.
- **Flexibility**: OpenTelemetry's modular architecture allows for easy integration with various log sources and sinks, enabling users to customize their logging setup according to their needs.
- **Rich Ecosystem**: The combination of OpenTelemetry, ClickHouse, and Grafana provides a powerful ecosystem for log management, visualization, and alerting, leveraging the strengths of each component

#### Comparison Table
It's important to note that all solutions have their own strengths and weaknesses, and the choice of logging system should be based on specific use cases and requirements. Below is a comparison table summarizing the key features of OpenTelemtry-based logging solution against Grafana Loki and VictoriaLogs:

| Feature                | PMM (OpenTelemetry + ClickHouse) | Grafana Loki | VictoriaLogs |
|------------------------|----------------------------------|--------------|--------------|
| Disk Usage             | Low                              | High         | Moderate     |
| Performance            | High                             | Low          | Moderate     |
| Scalability            | High (designed for large volumes)| Moderate     | Moderate     |
| HA Support             | Yes (ClickHouse replication)     | Yes          | Yes          |
| Extensibility          | High                             | Moderate     | Moderate     |
| Ecosystem              | Rich                             | Moderate     | Low          |
| Log Retention          | Yes (via TTL in ClickHouse)      | Yes          | Yes          |
| Log Visualization      | Yes                              | Yes          | Yes          |
| Log Search & Filtering | Yes                              | Yes          | Yes          |
| Alerting Integration   | Yes                              | Yes          | Yes          |
| Query language         | SQL (ClickHouse)                 | LogQL        | LogsQL       |
| Licensing              | Apache 2.0                       | AGPLv3       | Apache 2.0   |
| External Dependencies  | OpenTelemetry Collector          | Loki, Log collector | VictoriaLogs, Log collector |
| Community Support      | Strong                           | Strong       | Moderate     |
| Documentation          | Comprehensive                    | Moderate     | Good         |
| Requires Build         | No                               | Yes          | Yes          |
