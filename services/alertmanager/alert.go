// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package alertmanager

import (
	"fmt"

	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/percona/pmm-managed/models"
)

// Severity defines alert severity.
type Severity string

// severities
const (
	Error   = Severity("error")
	Warning = Severity("warning")
	Info    = Severity("info")
)

// AlertParams defines alert parameters.
type AlertParams struct {
	Name        string
	Summary     string
	Description string
	Severity    Severity

	Node    *models.Node
	Service *models.Service
	Agent   *models.Agent
}

// validate checks parameters and fills defaults.
func (ap *AlertParams) validate() error {
	if ap.Name == "" {
		return errors.New("empty Name")
	}
	if ap.Summary == "" {
		return errors.New("empty Summary")
	}
	if ap.Description == "" {
		return errors.New("empty Description")
	}

	if ap.Severity == "" {
		ap.Severity = Info
	}

	return nil
}

// makeAlert makes alert from given parameters.
func makeAlert(params *AlertParams) (*ammodels.PostableAlert, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	labels, err := models.MergeLabels(params.Node, params.Service, params.Agent)
	if err != nil {
		return nil, err
	}

	labels[model.AlertNameLabel] = params.Name
	labels["severity"] = string(params.Severity)
	labels["stt_check"] = "1"

	return &ammodels.PostableAlert{
		Alert: ammodels.Alert{
			// GeneratorURL: "TODO",
			Labels: labels,
		},

		// StartsAt and EndAt can't be added there without changes in registry

		Annotations: map[string]string{
			"summary":     params.Summary,
			"description": params.Description,
		},
	}, nil
}

// makeAlertPMMAgentNotConnected makes pmm_agent_not_connected alert.
func makeAlertPMMAgentNotConnected(agent *models.Agent, node *models.Node) (string, *ammodels.PostableAlert, error) {
	name := "pmm_agent_not_connected"
	alert, err := makeAlert(&AlertParams{
		Name:        name,
		Summary:     "pmm-agent is not connected to PMM Server",
		Description: fmt.Sprintf("Node name: %s", node.NodeName),
		Severity:    Warning,

		Node:  node,
		Agent: agent,
	})
	if err != nil {
		return "", nil, err
	}
	return name, alert, nil
}

// makeAlertPMMAgentIsOutdated makes pmm_agent_outdated alert.
func makeAlertPMMAgentIsOutdated(agent *models.Agent, node *models.Node, serverVersion string) (string, *ammodels.PostableAlert, error) {
	name := "pmm_agent_outdated"
	alert, err := makeAlert(&AlertParams{
		Name:    name,
		Summary: "pmm-agent is outdated",
		Description: fmt.Sprintf(
			"Node name: %s\npmm-agent version: %s\nPMM Server version: %s",
			node.NodeName, *agent.Version, serverVersion,
		),
		Severity: Info,

		Node:  node,
		Agent: agent,
	})
	if err != nil {
		return "", nil, err
	}
	return name, alert, nil
}
