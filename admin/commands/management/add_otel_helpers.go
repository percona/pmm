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
	"strings"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

var otelUpsertResultT = commands.ParseTemplate(`
OTEL Collector {{ .Verb }}.
Agent ID     : {{ .AgentID }}
PMM-Agent ID : {{ .PMMAgentID }}
Status       : {{ .Status }}
Disabled     : {{ .Disabled }}
`)

// otelUpsertResult is printed after add or change otel_collector.
type otelUpsertResult struct {
	Verb       string
	AgentID    string
	PMMAgentID string
	Status     string
	Disabled   bool
}

func (res *otelUpsertResult) Result() {}

func (res *otelUpsertResult) String() string {
	return commands.RenderTemplate(otelUpsertResultT, res)
}

func otelStatusString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func resolvePMMAgentIDForOtel(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return "", err
	}
	return status.AgentID, nil
}

func listOtelCollectorsForPMMAgent(pmmAgentID string) ([]*agents.ListAgentsOKBodyOtelCollectorItems0, error) {
	at := types.AgentTypeOtelCollector
	params := agents.NewListAgentsParams().
		WithContext(commands.Ctx).
		WithPMMAgentID(&pmmAgentID).
		WithAgentType(&at)
	resp, err := client.Default.AgentsService.ListAgents(params)
	if err != nil {
		return nil, err
	}
	if resp.Payload == nil {
		return nil, nil
	}
	return resp.Payload.OtelCollector, nil
}

func ensureAtMostOneOtelCollector(pmmAgentID string) ([]*agents.ListAgentsOKBodyOtelCollectorItems0, error) {
	rows, err := listOtelCollectorsForPMMAgent(pmmAgentID)
	if err != nil {
		return nil, err
	}
	if len(rows) > 1 {
		return nil, errors.New("multiple otel_collector agents exist for this pmm-agent; remove extras with `pmm-admin inventory remove agent` until only one remains")
	}
	return rows, nil
}

func addOtelCollectorAPI(body *agents.AddAgentParamsBodyOtelCollector) (*otelUpsertResult, error) {
	params := &agents.AddAgentParams{
		Body: agents.AddAgentBody{
			OtelCollector: body,
		},
		Context: commands.Ctx,
	}
	resp, err := client.Default.AgentsService.AddAgent(params)
	if err != nil {
		return nil, err
	}
	a := resp.Payload.OtelCollector
	return &otelUpsertResult{
		Verb:       "created",
		AgentID:    a.AgentID,
		PMMAgentID: a.PMMAgentID,
		Status:     otelStatusString(a.Status),
		Disabled:   a.Disabled,
	}, nil
}

func changeOtelCollectorAPI(agentID string, body *agents.ChangeAgentParamsBodyOtelCollector) (*otelUpsertResult, error) {
	params := agents.NewChangeAgentParams().
		WithContext(commands.Ctx).
		WithAgentID(agentID).
		WithBody(agents.ChangeAgentBody{
			OtelCollector: body,
		})
	resp, err := client.Default.AgentsService.ChangeAgent(params)
	if err != nil {
		return nil, err
	}
	a := resp.Payload.OtelCollector
	return &otelUpsertResult{
		Verb:       "updated",
		AgentID:    a.AgentID,
		PMMAgentID: a.PMMAgentID,
		Status:     otelStatusString(a.Status),
		Disabled:   a.Disabled,
	}, nil
}

// appendLogSourcesFromCLI parses --log-sources and --log-file-paths / --parser-preset into swagger log source items.
func appendLogSourcesFromCLI(logSources string, logFilePaths []string, parserPreset string) []*agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0 {
	var out []*agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0
	if logSources != "" {
		for pair := range strings.SplitSeq(logSources, ",") {
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}
			path := pair
			preset := "raw"
			if before, after, ok := strings.Cut(pair, ":"); ok {
				path = strings.TrimSpace(before)
				preset = strings.TrimSpace(after)
				if preset == "" {
					preset = "raw"
				}
			}
			if path != "" {
				out = append(out, &agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0{
					Path:   path,
					Preset: preset,
				})
			}
		}
		return out
	}
	if len(logFilePaths) == 0 {
		return nil
	}
	preset := parserPreset
	if preset == "" {
		preset = "raw"
	}
	for _, p := range logFilePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, &agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0{
			Path:   p,
			Preset: preset,
		})
	}
	return out
}

func toChangeAddLogSources(in []*agents.AddAgentParamsBodyOtelCollectorLogSourcesItems0) []*agents.ChangeAgentParamsBodyOtelCollectorAddLogSourcesItems0 {
	if len(in) == 0 {
		return nil
	}
	out := make([]*agents.ChangeAgentParamsBodyOtelCollectorAddLogSourcesItems0, 0, len(in))
	for _, e := range in {
		if e == nil {
			continue
		}
		out = append(out, &agents.ChangeAgentParamsBodyOtelCollectorAddLogSourcesItems0{
			Path:   e.Path,
			Preset: e.Preset,
		})
	}
	return out
}
