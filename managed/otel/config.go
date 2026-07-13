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

package otel

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// IndentYAML prefixes each line of block with indent so it can be nested under a YAML key
// (e.g. filelog receiver operators under "operators:").
func IndentYAML(block, indent string) string {
	if block == "" {
		return ""
	}
	lines := strings.Split(strings.TrimSuffix(block, "\n"), "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n") + "\n"
}

// sanitizePresetName returns a safe receiver id suffix (alphanumeric and underscore only).
func sanitizePresetName(name string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(name, "_")
}

func quoteYAMLString(s string) string {
	if s == "" || (!strings.Contains(s, "'") && !strings.Contains(s, "\n")) {
		return "'" + s + "'"
	}
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// BuildServerOtelConfigYAML builds the server-side otel-collector config YAML with filelog
// receivers (using presets from log_parser_presets) and OTLP receiver. Filelog uses start_at: beginning.
// If no presets can be loaded, returns receiver-only config (OTLP + processors + ClickHouse).
func BuildServerOtelConfigYAML(q *reform.Querier, endpoint, username, password string, retentionDays int) (string, error) {
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}
	if username == "" {
		username = "default"
	}
	if password == "" {
		password = "clickhouse"
	}
	if retentionDays <= 0 {
		retentionDays = 7
	}
	chEndpoint := "tcp://" + endpoint
	ttl := fmt.Sprintf("%dh", retentionDays*24) //nolint:mnd

	processorsBlock := `  memory_limiter:
    check_interval: 1s
    limit_mib: 512
  resource/add_server_node:
    attributes:
      - key: node_name
        value: pmm-server
        action: insert
  transform:
    error_mode: ignore
    log_statements:
      - set(resource.attributes["pmm_source"], resource.attributes["node_name"]) where resource.attributes["node_name"] != nil
  batch:
    timeout: 1s
    send_batch_size: 1024
    send_batch_max_size: 2048
`

	exportersBlock := `exporters:
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
`

	// Build filelog receivers from DefaultServerOtelLogSources and DB presets.
	var filelogReceivers []string
	var receiversYaml strings.Builder
	receiversYaml.WriteString("receivers:\n")
	receiversYaml.WriteString("  otlp:\n    protocols:\n      grpc:\n        endpoint: 0.0.0.0:4317\n      http:\n        endpoint: 0.0.0.0:4318\n")

	for _, src := range DefaultServerOtelLogSources {
		preset, err := models.FindLogParserPresetByName(q, src.Preset)
		if err != nil {
			return "", err
		}
		if preset == nil {
			logrus.WithField("preset", src.Preset).Debugf("Skipping server log source %q: preset not found", src.Path)
			continue
		}
		// Use path-derived ID to avoid duplicate YAML keys when multiple sources share the same preset (e.g. pmm_agent for pmm-agent.log, qan-api2.log, vmproxy.log).
		stem := strings.TrimSuffix(filepath.Base(src.Path), filepath.Ext(src.Path))
		receiverID := "filelog/server_" + sanitizePresetName(stem)
		filelogReceivers = append(filelogReceivers, receiverID)
		receiversYaml.WriteString("  " + receiverID + ":\n")
		receiversYaml.WriteString("    include: [" + fmt.Sprintf("%q", src.Path) + "]\n")
		receiversYaml.WriteString("    start_at: beginning\n")
		if preset.OperatorYAML != "" {
			receiversYaml.WriteString("    operators:\n")
			receiversYaml.WriteString(IndentYAML(preset.OperatorYAML, "      "))
		}
	}

	if len(filelogReceivers) == 0 {
		// Fallback: receiver-only config (OTLP only).
		logrus.Warn("No server log presets loaded; using receiver-only OTEL config")
		pipelineReceivers := "[otlp]"
		return `# Managed by pmm-managed. DO NOT EDIT.
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
processors:
` + processorsBlock + exportersBlock + `
service:
  pipelines:
    logs:
      receivers: ` + pipelineReceivers + `
      processors: [memory_limiter, resource/add_server_node, transform, batch]
      exporters: [clickhouse]
`, nil
	}

	// Full config: filelog + OTLP.
	pipelineReceivers := "[" + strings.Join(filelogReceivers, ", ") + ", otlp]"
	return `# Managed by pmm-managed. DO NOT EDIT.
` + receiversYaml.String() + `
processors:
` + processorsBlock + exportersBlock + `
service:
  pipelines:
    logs:
      receivers: ` + pipelineReceivers + `
      processors: [memory_limiter, resource/add_server_node, transform, batch]
      exporters: [clickhouse]
`, nil
}
