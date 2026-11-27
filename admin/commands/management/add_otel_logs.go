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
	"errors"

	"github.com/percona/pmm/admin/commands"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

// AddOtelLogsCommand adds or merges log sources (and optional labels) on the single otel_collector agent.
type AddOtelLogsCommand struct {
	PMMAgentID   string            `help:"PMM Agent ID (default: local pmm-agent)"`
	LogFilePaths []string          `name:"log-file-paths" help:"Log file paths. Used with --parser-preset or as raw if --log-sources not set."`
	LogSources   string            `name:"log-sources" help:"Comma-separated path:preset pairs (e.g. /var/log/mysql/error.log:mysql_error,/var/log/messages:syslog_mysql_systemd). Overrides --log-file-paths."`                 //nolint:lll
	ParserPreset string            `name:"parser-preset" help:"Preset for all --log-file-paths when --log-sources is not set (built-ins include mysql_error, syslog_mysql_systemd, nginx_access, raw, …; UI: Settings → OTEL)."` //nolint:lll
	CustomLabels map[string]string `mapsep:"," help:"Custom labels merged into the collector agent"`
}

// RunCmd implements OTEL log source upsert.
func (cmd *AddOtelLogsCommand) RunCmd() (commands.Result, error) {
	pmmAgentID, err := resolvePMMAgentIDForOtel(cmd.PMMAgentID)
	if err != nil {
		return nil, err
	}

	rows, err := ensureAtMostOneOtelCollector(pmmAgentID)
	if err != nil {
		return nil, err
	}

	srcItems := appendLogSourcesFromCLI(cmd.LogSources, cmd.LogFilePaths, cmd.ParserPreset)

	custom := commands.ParseKeyValuePair(&cmd.CustomLabels)

	if len(rows) == 0 {
		body := &agents.AddAgentParamsBodyOtelCollector{
			PMMAgentID:   pmmAgentID,
			CustomLabels: *custom,
		}
		if len(srcItems) > 0 {
			body.LogSources = srcItems
		}
		return addOtelCollectorAPI(body)
	}

	chBody := &agents.ChangeAgentParamsBodyOtelCollector{
		MergeLabels: *custom,
	}
	if len(srcItems) > 0 {
		chBody.AddLogSources = toChangeAddLogSources(srcItems)
	}
	if len(chBody.MergeLabels) == 0 && len(chBody.AddLogSources) == 0 {
		return nil, errors.New("nothing to change: set --log-sources, --log-file-paths, or --custom-labels")
	}
	return changeOtelCollectorAPI(rows[0].AgentID, chBody)
}
