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

package agents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"gopkg.in/reform.v1"
)

const (
	logFilePathsLabel = "log_file_paths"
	logSourcesLabel   = "log_sources"
	presetRaw         = "raw"
)

// logSourceEntry matches the JSON stored in custom_labels["log_sources"].
type logSourceEntry struct {
	Path   string `json:"path"`
	Preset string `json:"preset"`
}

// getLogSourcesFromAgent returns path+preset pairs from agent custom_labels.
// Prefers log_sources JSON; falls back to log_file_paths with preset "raw".
func getLogSourcesFromAgent(row *models.Agent) ([]logSourceEntry, error) {
	labels, err := row.GetCustomLabels()
	if err != nil {
		return nil, err
	}
	if s := labels[logSourcesLabel]; s != "" {
		var entries []logSourceEntry
		if err := json.Unmarshal([]byte(s), &entries); err != nil {
			return nil, err
		}
		return entries, nil
	}
	// Legacy: log_file_paths as comma-separated paths with preset "raw".
	if s := labels[logFilePathsLabel]; s != "" {
		var entries []logSourceEntry
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				entries = append(entries, logSourceEntry{Path: p, Preset: presetRaw})
			}
		}
		return entries, nil
	}
	return nil, nil
}

// sanitizePresetName returns a safe receiver id suffix (alphanumeric and underscore only).
func sanitizePresetName(name string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(name, "_")
}

// otelCollectorConfig returns desired configuration of otel-collector process for log collection (OTLP to PMM Server).
func otelCollectorConfig(row *models.Agent, q *reform.Querier) *agentv1.SetStateRequest_AgentProcess {
	args := []string{
		"--config={{ .TextFiles.otelconfig }}",
	}

	sources, err := getLogSourcesFromAgent(row)
	if err != nil || len(sources) == 0 {
		// No log sources or parse error: OTLP-only config.
		configYaml := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
` + baseOtelConfigYaml([]string{"otlp"})
		tdp := models.TemplateDelimsPair()
		return &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_OTEL_COLLECTOR,
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
			Args:               args,
			TextFiles: map[string]string{
				"otelconfig": configYaml,
			},
		}
	}

	// Group paths by preset.
	byPreset := make(map[string][]string)
	for _, e := range sources {
		preset := e.Preset
		if preset == "" {
			preset = presetRaw
		}
		byPreset[preset] = append(byPreset[preset], e.Path)
	}

	// Load preset operator YAML from DB for non-raw presets.
	presetYAML := make(map[string]string)
	for name := range byPreset {
		if name == presetRaw {
			continue
		}
		p, err := models.FindLogParserPresetByName(q, name)
		if err != nil || p == nil {
			continue
		}
		presetYAML[name] = p.OperatorYAML
	}

	var receivers []string
	configYaml := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
`
	for preset, paths := range byPreset {
		if len(paths) == 0 {
			continue
		}
		receiverID := "filelog/preset_" + sanitizePresetName(preset)
		receivers = append(receivers, receiverID)
		var quoted []string
		for _, p := range paths {
			if p != "" {
				quoted = append(quoted, fmt.Sprintf("%q", p))
			}
		}
		if len(quoted) == 0 {
			continue
		}
		configYaml += fmt.Sprintf("  %s:\n    include: [%s]\n    start_at: end\n", receiverID, strings.Join(quoted, ", "))
		if preset != presetRaw {
			if yaml, ok := presetYAML[preset]; ok && yaml != "" {
				configYaml += "    operators:\n" + yaml + "\n"
			}
		}
	}
	sort.Strings(receivers)
	receivers = append(receivers, "otlp")
	configYaml += baseOtelConfigYaml(receivers)

	tdp := models.TemplateDelimsPair()
	return &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_OTEL_COLLECTOR,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles: map[string]string{
			"otelconfig": configYaml,
		},
	}
}

// baseOtelConfigYaml returns processors, exporters, and service.pipelines with the given receivers.
func baseOtelConfigYaml(receivers []string) string {
	receiversYaml := "[" + strings.Join(receivers, ", ") + "]"
	return `processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 128
  batch:
    timeout: 2s
    send_batch_size: 10000

exporters:
  otlp_http:
    endpoint: '{{ .server_otlp_url }}'
    headers:
      "Authorization": "Basic {{ .server_auth_b64 }}"
    tls:
      insecure_skip_verify: {{ .server_insecure }}

service:
  pipelines:
    logs:
      receivers: ` + receiversYaml + `
      processors: [memory_limiter, batch]
      exporters: [otlp_http]
`
}
