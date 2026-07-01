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

package supervisord

import (
	"fmt"
	"strings"
)

// buildOtelCollectorConfigYAML returns the server-side otel-collector config YAML.
// OTLP receives logs, traces, and metrics; separate ClickHouse exporter instances apply TTL per signal.
// LogRetentionDays, spanRetentionDays, metricsRetentionDays control exporter TTL (table-level TTL also set in PMM DDL).
func buildOtelCollectorConfigYAML(endpoint, username, password string, logRetentionDays, spanRetentionDays, metricsRetentionDays int) string {
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}
	if username == "" {
		username = "default"
	}
	if logRetentionDays <= 0 {
		logRetentionDays = 7
	}
	if spanRetentionDays <= 0 {
		spanRetentionDays = 7
	}
	if metricsRetentionDays <= 0 {
		metricsRetentionDays = 90
	}
	chEndpoint := "tcp://" + endpoint
	logsTTL := fmt.Sprintf("%dh", logRetentionDays*24)        //nolint:mnd
	tracesTTL := fmt.Sprintf("%dh", spanRetentionDays*24)     //nolint:mnd
	metricsTTL := fmt.Sprintf("%dh", metricsRetentionDays*24) //nolint:mnd

	// Exporter blocks: component type is "clickhouse"; multiple instances use type/name keys (e.g. clickhouse/logs).
	// Underscore-only keys like "clickhouse_logs" are invalid — the loader treats them as unknown types.
	// create_schema false — PMM-managed applies DDL (otel.logs, otel.otel_traces, otel.otel_metrics_sum, ...).
	exporters := fmt.Sprintf(`exporters:
  clickhouse/logs:
    endpoint: %s
    database: otel
    username: %s
    password: %s
    logs_table_name: logs
    create_schema: false
    ttl: %s
    timeout: 5s
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
  clickhouse/traces:
    endpoint: %s
    database: otel
    username: %s
    password: %s
    traces_table_name: otel_traces
    create_schema: false
    ttl: %s
    timeout: 5s
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
  clickhouse/metrics:
    endpoint: %s
    database: otel
    username: %s
    password: %s
    create_schema: false
    ttl: %s
    timeout: 5s
    metrics_tables:
      sum:
        name: otel_metrics_sum
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
`, chEndpoint, quoteYAMLString(username), quoteYAMLString(password), logsTTL,
		chEndpoint, quoteYAMLString(username), quoteYAMLString(password), tracesTTL,
		chEndpoint, quoteYAMLString(username), quoteYAMLString(password), metricsTTL)

	return `# Managed by pmm-managed. DO NOT EDIT.
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
  transform:
    error_mode: ignore
    log_statements:
      - set(resource.attributes["pmm_source"], resource.attributes["node_name"]) where resource.attributes["node_name"] != nil
  batch:
    timeout: 1s
    send_batch_size: 1024
    send_batch_max_size: 2048
` + exporters + `
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, transform, batch]
      exporters: [clickhouse/logs]
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [clickhouse/traces]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [clickhouse/metrics]
`
}

// quoteYAMLString doubles single quotes for YAML single-quoted style when needed.
func quoteYAMLString(s string) string {
	if s == "" || (!strings.Contains(s, "'") && !strings.Contains(s, "\n")) {
		return "'" + s + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
