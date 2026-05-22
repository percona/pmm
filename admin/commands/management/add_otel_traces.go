// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package management

import (
	"github.com/percona/pmm/admin/commands"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

// AddOtelTracesCommand records trace-ingestion intent via labels; OTLP receivers are already enabled on the collector.
type AddOtelTracesCommand struct {
	PMMAgentID   string            `help:"PMM Agent ID (default: local pmm-agent)"`
	CustomLabels map[string]string `mapsep:"," help:"Extra custom labels merged with defaults"`
}

// RunCmd implements trace-policy label upsert.
func (cmd *AddOtelTracesCommand) RunCmd() (commands.Result, error) {
	pmmAgentID, err := resolvePMMAgentIDForOtel(cmd.PMMAgentID)
	if err != nil {
		return nil, err
	}

	rows, err := ensureAtMostOneOtelCollector(pmmAgentID)
	if err != nil {
		return nil, err
	}

	custom := *commands.ParseKeyValuePair(&cmd.CustomLabels)
	if custom["pmm_otlp_traces_enabled"] == "" {
		custom["pmm_otlp_traces_enabled"] = "v1"
	}

	if len(rows) == 0 {
		return addOtelCollectorAPI(&agents.AddAgentParamsBodyOtelCollector{
			PMMAgentID:   pmmAgentID,
			CustomLabels: custom,
		})
	}

	return changeOtelCollectorAPI(rows[0].AgentID, &agents.ChangeAgentParamsBodyOtelCollector{
		MergeLabels: custom,
	})
}
