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
// Receiver only: OTLP (for agents), transform (pmm_source from node_name), batch, ClickHouse exporter.
// All log collection (including server logs) is done by pmm-agent using DB-driven presets.
// Endpoint must be the ClickHouse address with port (e.g. "127.0.0.1:9000"); it will be prefixed with "tcp://".
func buildOtelCollectorConfigYAML(endpoint, username, password string, retentionDays int) string {
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}
	if username == "" {
		username = "default"
	}
	if retentionDays <= 0 {
		retentionDays = 7
	}
	chEndpoint := "tcp://" + endpoint
	// Exporter ttl uses Go time.Duration: only ns, us, ms, s, m, h are valid (no "d" for days).
	ttl := fmt.Sprintf("%dh", retentionDays*24)

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
exporters:
  clickhouse:
    endpoint: ` + chEndpoint + `
    database: otel
    username: ` + quoteYAMLString(username) + `
    password: ` + quoteYAMLString(password) + `
    logs_table_name: logs
    create_schema: false
    ttl: ` + ttl + `
    timeout: 5s
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, transform, batch]
      exporters: [clickhouse]
`
}

// quoteYAMLString doubles single quotes for YAML single-quoted style when needed.
func quoteYAMLString(s string) string {
	if s == "" || (!strings.Contains(s, "'") && !strings.Contains(s, "\n")) {
		return "'" + s + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
