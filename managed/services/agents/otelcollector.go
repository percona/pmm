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
	"fmt"
	"strings"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

const logFilePathsLabel = "log_file_paths"

// otelCollectorConfig returns desired configuration of otel-collector process for log collection (OTLP to PMM Server).
func otelCollectorConfig(row *models.Agent) *agentv1.SetStateRequest_AgentProcess {
	args := []string{
		"--config={{ .TextFiles.otelconfig }}",
	}

	labels, _ := row.GetCustomLabels()
	userPaths := labels[logFilePathsLabel]
	var receivers []string
	if userPaths != "" {
		receivers = append(receivers, "filelog/custom")
	}
	receivers = append(receivers, "otlp")
	receiversYaml := "[" + strings.Join(receivers, ", ") + "]"

	configYaml := `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
`
	if userPaths != "" {
		paths := strings.Split(userPaths, ",")
		for i, p := range paths {
			paths[i] = strings.TrimSpace(p)
		}
		var quoted []string
		for _, p := range paths {
			if p != "" {
				quoted = append(quoted, fmt.Sprintf("%q", p))
			}
		}
		if len(quoted) != 0 {
			configYaml += fmt.Sprintf("  filelog/custom:\n    include: [%s]\n    start_at: end\n", strings.Join(quoted, ", "))
		}
	}
	configYaml += `processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 128
  batch:
    timeout: 2s
    send_batch_size: 10000

exporters:
  otlp:
    endpoint: '{{ .server_otlp_url }}'
    headers:
      "Authorization": "Basic {{ .server_auth_b64 }}"
    tls:
      insecure_skip_verify: "{{ .server_insecure }}"

service:
  pipelines:
    logs:
      receivers: ` + receiversYaml + `
      processors: [memory_limiter, batch]
      exporters: [otlp]
`

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
