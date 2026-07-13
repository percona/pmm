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

	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/otel"
)

const (
	logFilePathsLabel = "log_file_paths"
	logSourcesLabel   = "log_sources"
	presetRaw         = "raw"
)

var presetNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

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
		err := json.Unmarshal([]byte(s), &entries)
		if err != nil {
			return nil, err
		}
		return entries, nil
	}
	// Legacy: log_file_paths as comma-separated paths with preset "raw".
	if s := labels[logFilePathsLabel]; s != "" {
		var entries []logSourceEntry
		for p := range strings.SplitSeq(s, ",") {
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
	return presetNameSanitizer.ReplaceAllString(name, "_")
}

// otelResourceAttributes returns resource attributes (agent_id, node_id, service_name, etc.) for the OTEL collector
// so that logs in ClickHouse can be correlated with the same labels as VictoriaMetrics.
// If node or service lookup fails, returns agent-only labels so logs still have at least agent_id and agent_type.
func otelResourceAttributes(row *models.Agent, q *reform.Querier) (map[string]string, error) {
	var node *models.Node
	var service *models.Service

	if nodeID := pointer.GetString(row.NodeID); nodeID != "" {
		n, err := models.FindNodeByID(q, nodeID)
		if err == nil {
			node = n
		}
	}

	if serviceID := pointer.GetString(row.ServiceID); serviceID != "" {
		s, err := models.FindServiceByID(q, serviceID)
		if err == nil {
			service = s
		}
	}

	labels, err := models.MergeLabels(node, service, row)
	if err != nil {
		return nil, err
	}

	// Match scrape config: instance = agent_id for filtering/grouping.
	labels["instance"] = row.AgentID

	return labels, nil
}

// otelCollectorConfig returns desired configuration of otel-collector process for log collection (OTLP to PMM Server).
func otelCollectorConfig(row *models.Agent, q *reform.Querier) *agentv1.SetStateRequest_AgentProcess {
	args := []string{
		"--config={{ .TextFiles.otelconfig }}",
	}

	resourceAttrs, err := otelResourceAttributes(row, q)
	if err != nil {
		resourceAttrs = nil
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
` + baseOtelConfigYaml([]string{"otlp"}, []string{"otlp"}, resourceAttrs)
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
	var configYamlSb179 strings.Builder
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
		fmt.Fprintf(&configYamlSb179, "  %s:\n    include: [%s]\n    start_at: end\n", receiverID, strings.Join(quoted, ", "))
		if preset != presetRaw {
			if yaml, ok := presetYAML[preset]; ok && yaml != "" {
				configYamlSb179.WriteString("    operators:\n" + otel.IndentYAML(yaml, "      "))
			}
		}
	}
	configYaml += configYamlSb179.String()
	sort.Strings(receivers)
	receivers = append(receivers, "otlp")
	configYaml += baseOtelConfigYaml(receivers, []string{"otlp"}, resourceAttrs)

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

// quoteYAMLAttrValue returns a YAML-safe quoted value for resource processor attributes.
func quoteYAMLAttrValue(v string) string {
	if v == "" {
		return `""`
	}
	// Use double quotes and escape backslash and double quote.
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range v {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// baseOtelConfigYaml returns processors, exporters, and service.pipelines.
// LogPipelineReceivers lists receivers for the logs pipeline (OTLP plus any filelog/* receivers).
// TracesMetricsReceivers lists receivers for traces and metrics only — must not include filelog,
// which emits logs only and cannot be wired into traces or metrics pipelines.
// If resourceAttrs is non-nil and non-empty, a resource processor is added to set PMM context (agent_id, node_id, etc.)
// so logs in ClickHouse match VictoriaMetrics labels.
func baseOtelConfigYaml(logPipelineReceivers, tracesMetricsReceivers []string, resourceAttrs map[string]string) string {
	logReceiversYaml := "[" + strings.Join(logPipelineReceivers, ", ") + "]"
	tracesMetricsReceiversYaml := "[" + strings.Join(tracesMetricsReceivers, ", ") + "]"

	processorsBlock := `  memory_limiter:
    check_interval: 1s
    limit_mib: 128
  batch:
    timeout: 2s
    send_batch_size: 10000
`
	pipelineProcessors := "[memory_limiter, batch]"

	if len(resourceAttrs) != 0 {
		// Resource processor adds agent_id, node_id, service_name, etc. to every log record.
		processorsBlock = `  resource/add_pmm_context:
    attributes:
`
		keys := make([]string, 0, len(resourceAttrs))
		for k := range resourceAttrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var processorsBlockSb268 strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&processorsBlockSb268, "      - key: %s\n        value: %s\n        action: upsert\n", k, quoteYAMLAttrValue(resourceAttrs[k]))
		}
		processorsBlock += processorsBlockSb268.String()
		processorsBlock += `  memory_limiter:
    check_interval: 1s
    limit_mib: 128
  batch:
    timeout: 2s
    send_batch_size: 10000
`
		pipelineProcessors = "[resource/add_pmm_context, memory_limiter, batch]"
	}

	return `processors:
` + processorsBlock + `
exporters:
  otlp_http:
    endpoint: '{{ .server_otlp_url }}'
    headers:
      "Authorization": "Basic {{ .server_auth_b64 }}"
    tls:
      insecure_skip_verify: {{ .server_insecure }}

service:
  telemetry:
    metrics:
      level: none
  pipelines:
    logs:
      receivers: ` + logReceiversYaml + `
      processors: ` + pipelineProcessors + `
      exporters: [otlp_http]
    traces:
      receivers: ` + tracesMetricsReceiversYaml + `
      processors: ` + pipelineProcessors + `
      exporters: [otlp_http]
    metrics:
      receivers: ` + tracesMetricsReceiversYaml + `
      processors: ` + pipelineProcessors + `
      exporters: [otlp_http]
`
}
