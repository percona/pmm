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
	"encoding/json"
	"errors"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
)

const updateNodeHelp = `Changes the labels of the Node in place: the Node, its Agents and its Services keep their IDs and
history, and the metrics of the Node and of the Services on it carry the new labels from the next scrape.
A Service with an environment of its own keeps it; the Node environment applies to the others.
--environment also replaces an "environment" custom label, which would otherwise override it.

Only the flags you pass are changed; everything else stays as it is. Use an empty value, e.g. --region=
or --custom-labels=, to clear a label.`

// UpdateNodeCommand is used by Kong for CLI flags and commands.
type UpdateNodeCommand struct {
	typedFlags

	NodeID       string            `help:"ID of the Node to update (default: the Node of this pmm-agent)"`
	Environment  string            `help:"Node environment"`
	NodeModel    string            `help:"Node model"`
	Region       string            `help:"Node region"`
	Az           string            `help:"Node availability zone"`
	CustomLabels map[string]string `mapsep:"," help:"Custom user-assigned labels, replacing all of them"`
	DryRun       bool              `help:"Show what would change, without changing anything"`
}

// Help returns the detailed help of the command.
func (cmd *UpdateNodeCommand) Help() string {
	return updateNodeHelp
}

// nodeLabels holds the labels of a Node of any type.
type nodeLabels struct {
	NodeID       string            `json:"node_id"`
	NodeName     string            `json:"node_name"`
	Environment  string            `json:"environment"`
	NodeModel    string            `json:"node_model"`
	Region       string            `json:"region"`
	Az           string            `json:"az"`
	CustomLabels map[string]string `json:"custom_labels"`
}

// nodeLabelsOf returns the labels of the Node in a GetNode or ChangeNode response. The response holds the
// Node under the name of its type, and every type names its labels alike.
func nodeLabelsOf(payload any) (*nodeLabels, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var byType map[string]*nodeLabels
	err = json.Unmarshal(b, &byType)
	if err != nil {
		return nil, err
	}

	for _, node := range byType {
		if node != nil {
			return node, nil
		}
	}

	return nil, errors.New("PMM Server returned no Node")
}

// settings returns the labels of the Node as the flags that set them.
func (n *nodeLabels) settings() []setting {
	return []setting{
		{Flag: "--environment", Value: n.Environment},
		{Flag: "--node-model", Value: n.NodeModel},
		{Flag: "--region", Value: n.Region},
		{Flag: "--az", Value: n.Az},
		{Flag: "--custom-labels", Value: labelsSettingValue(n.CustomLabels)},
	}
}

var updateNodeResultT = commands.ParseTemplate(`
{{ .Title }}
Node ID  : {{ .NodeID }}
Node name: {{ .NodeName }}

{{ if .Changes }}{{ if .DryRun }}Settings the update would change:{{ else }}Changed settings:{{ end }}
{{- range .Changes }}
  {{ . }}
{{- end }}
{{- else }}No settings changed.{{ end }}

Current settings:
{{- range .Settings }}
  {{ . }}
{{- end }}
`)

type updateNodeResult struct {
	NodeID   string          `json:"node_id"`
	NodeName string          `json:"node_name"`
	DryRun   bool            `json:"dry_run"`
	Changed  bool            `json:"changed"`
	Changes  []settingChange `json:"changes"`
	Settings []setting       `json:"settings"`
}

func (res *updateNodeResult) Result() {}

// Title returns the first line of the text output.
func (res *updateNodeResult) Title() string {
	if res.DryRun {
		return "Node not changed (dry run)."
	}

	return "Node updated."
}

func (res *updateNodeResult) String() string {
	return commands.RenderTemplate(updateNodeResultT, res)
}

// RunCmd runs the command for UpdateNodeCommand.
func (cmd *UpdateNodeCommand) RunCmd() (commands.Result, error) {
	nodeID := cmd.NodeID
	if nodeID == "" {
		status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
		if err != nil {
			return nil, err
		}
		nodeID = status.NodeID
	}

	current, err := inventoryClient.Default.NodesService.GetNode(&nodes.GetNodeParams{
		NodeID:  nodeID,
		Context: commands.Ctx,
	})
	if err != nil {
		return nil, err
	}
	before, err := nodeLabelsOf(current.Payload)
	if err != nil {
		return nil, err
	}

	body := nodes.ChangeNodeBody{
		Environment: cmd.str("environment", cmd.Environment),
		NodeModel:   cmd.str("node-model", cmd.NodeModel),
		Region:      cmd.str("region", cmd.Region),
		Az:          cmd.str("az", cmd.Az),
		DryRun:      cmd.DryRun,
	}
	if labels := cmd.labels("custom-labels", cmd.CustomLabels); labels != nil {
		body.CustomLabels = &nodes.ChangeNodeParamsBodyCustomLabels{Values: labels}
	}

	resp, err := inventoryClient.Default.NodesService.ChangeNode(&nodes.ChangeNodeParams{
		NodeID:  nodeID,
		Body:    body,
		Context: commands.Ctx,
	})
	if err != nil {
		return nil, err
	}
	after, err := nodeLabelsOf(resp.Payload)
	if err != nil {
		return nil, err
	}

	res := &updateNodeResult{
		NodeID:   after.NodeID,
		NodeName: after.NodeName,
		DryRun:   cmd.DryRun,
		Settings: after.settings(),
	}
	beforeSettings := before.settings()
	for i, s := range res.Settings {
		if old := beforeSettings[i].Value; old != s.Value {
			res.Changes = append(res.Changes, settingChange{Flag: s.Flag, Before: old, After: s.Value})
		}
	}
	res.Changed = len(res.Changes) != 0

	return res, nil
}
