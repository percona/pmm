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
// It includes OTLP receiver (for agents), filelog receivers for server logs (nginx, grafana,
// pmm-managed, pmm-agent, postgres), transform and batch processors, and ClickHouse exporter.
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

	// YAML structure aligned with PoC PR #4230 (dev/otel/config.yml): OTLP + filelog receivers,
	// memory_limiter, transform, batch processors, clickhouse exporter. Log paths match PMM server layout.
	return `# Managed by pmm-managed. DO NOT EDIT.
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
  # nginx.log: we only parse logfmt access lines (time="...", host=..., status=...) for now.
  # Nginx internal lines (e.g. 2026/03/05 [warn] ..., nginx: [alert] ...) in the same file are not parsed.
  filelog/nginx_access:
    include: [/srv/logs/nginx.log]
    operators:
      - type: key_value_parser
        parse_from: body
        parse_to: attributes
        pair_delimiter: " "
        key_value_delimiter: "="
      - type: time_parser
        parse_from: attributes.time
        layout: '2006-01-02T15:04:05Z07:00'
        layout_type: gotime
      - type: add
        field: attributes.level
        value: 'EXPR(int(attributes.status) >= 500 ? "error" : (int(attributes.status) >= 400 ? "warn" : "info"))'
      - type: severity_parser
        parse_from: attributes.level
        preset: none
        mapping:
          info: info
          warn: warn
          error: error
  filelog/nginx_error:
    include: [/srv/logs/nginx-error.log]
    operators:
      - type: regex_parser
        regex: '^(?P<timestamp>\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(?P<level>\w+)\] (?P<pid>\d+)#(?P<tid>\d+): (?P<message>.*?)(?:, client: (?P<client>[^,]+))?(?:, server: (?P<server>[^,]+))?(?:, request: "(?P<request>[^"]*)")?(?:, host: "(?P<host>[^"]*)")?.*'
        parse_from: body
        parse_to: attributes
      - type: time_parser
        parse_from: attributes.timestamp
        layout: '2006/01/02 15:04:05'
        layout_type: gotime
      - type: severity_parser
        parse_from: attributes.level
        preset: none
        mapping:
          debug: debug
          info: info
          notice: info
          warn: warn
          error: error
          crit: fatal
          alert: fatal
          emerg: fatal
  filelog/grafana:
    include: [/srv/logs/grafana.log]
    operators:
      - type: key_value_parser
        parse_from: body
        parse_to: attributes
        pair_delimiter: " "
        key_value_delimiter: "="
      - type: time_parser
        parse_from: attributes.t
        layout: '2006-01-02T15:04:05.000000000Z07:00'
        layout_type: gotime
        on_error: drop
      - type: severity_parser
        parse_from: attributes.level
        preset: none
        mapping:
          debug: debug
          info: info
          warn: warn
          error: error
      - type: move
        from: attributes.msg
        to: body
  filelog/pmm_managed:
    include: [/srv/logs/pmm-managed.log]
    operators:
      - type: key_value_parser
        parse_from: body
        parse_to: attributes
        pair_delimiter: " "
        key_value_delimiter: "="
      - type: time_parser
        parse_from: attributes.time
        layout: '2006-01-02T15:04:05.000Z07:00'
        layout_type: gotime
      - type: severity_parser
        parse_from: attributes.level
        preset: none
        mapping:
          debug: debug
          info: info
          warning: warn
          warn: warn
          error: error
          fatal: fatal
          panic: fatal
  filelog/pmm_agent:
    include: [/srv/logs/pmm-agent.log]
    operators:
      - type: key_value_parser
        parse_from: body
        parse_to: attributes
        pair_delimiter: " "
        key_value_delimiter: "="
      - type: time_parser
        parse_from: attributes.time
        layout: '2006-01-02T15:04:05.000Z07:00'
        layout_type: gotime
      - type: severity_parser
        parse_from: attributes.level
        preset: none
        mapping:
          debug: debug
          info: info
          warning: warn
          warn: warn
          error: error
          fatal: fatal
          panic: fatal
  filelog/postgres:
    include: [/srv/logs/postgresql14.log]
    start_at: end
    operators:
      - type: regex_parser
        regex: '^(?P<timestamp>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d+) UTC \[(?P<pid>\d+)\] (?P<level>\w+):\s*(?P<message>.*)$'
        parse_from: body
        parse_to: attributes
      - type: time_parser
        parse_from: attributes.timestamp
        layout: '2006-01-02 15:04:05.000 UTC'
        layout_type: gotime
      - type: severity_parser
        parse_from: attributes.level
        preset: none
        mapping:
          debug: debug
          info: info
          notice: info
          warning: warn
          warn: warn
          error: error
          fatal: fatal
          panic: fatal
          LOG: info
          STATEMENT: info
      - type: move
        from: attributes.message
        to: body
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
  transform:
    error_mode: ignore
    log_statements:
      - set(log.attributes["log_file"], log.attributes["log.file.name"]) where log.attributes["log.file.name"] != nil
      - delete_key(log.attributes, "log.file.name") where log.attributes["log.file.name"] != nil
      - delete_key(log.attributes, "level") where log.attributes["level"] != nil
      - set(resource.attributes["service.name"], "nginx") where log.attributes["log_file"] == "nginx.log" or log.attributes["log_file"] == "nginx-access.log"
      - set(resource.attributes["service.version"], "1.20.1") where log.attributes["log_file"] == "nginx.log" or log.attributes["log_file"] == "nginx-access.log"
      - set(resource.attributes["service.name"], "grafana") where log.attributes["log_file"] == "grafana.log"
      - set(resource.attributes["service.version"], "11.6.1") where log.attributes["log_file"] == "grafana.log"
      - set(resource.attributes["service.name"], "pmm-managed") where log.attributes["log_file"] == "pmm-managed.log"
      - set(resource.attributes["service.version"], "3.3.1") where log.attributes["log_file"] == "pmm-managed.log"
      - set(resource.attributes["service.name"], "pmm-agent") where log.attributes["log_file"] == "pmm-agent.log"
      - set(resource.attributes["service.version"], "2.42.0") where log.attributes["log_file"] == "pmm-agent.log"
      - set(resource.attributes["service.name"], "postgres") where log.attributes["log_file"] == "postgresql14.log"
      - set(resource.attributes["service.version"], "14") where log.attributes["log_file"] == "postgresql14.log"
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
      receivers: [otlp, filelog/nginx_access, filelog/nginx_error, filelog/grafana, filelog/pmm_managed, filelog/pmm_agent, filelog/postgres]
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
