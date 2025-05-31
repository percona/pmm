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

package management

import (
	"regexp"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/managed/models"
)

var IDRegex = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsNodeAgent checks if agent runs on the same node as the service (e.g. pmm-agent).
func IsNodeAgent(agent *models.Agent, service *models.Service) bool {
	return agent.ServiceID == nil && pointer.GetString(agent.RunsOnNodeID) == service.NodeID
}

// IsVMAgent checks if the agent is an vmagent and if it relates to a particular service.
func IsVMAgent(agent *models.Agent, service *models.Service) bool {
	return pointer.GetString(agent.NodeID) == service.NodeID && agent.AgentType == models.VMAgentType
}

// IsServiceAgent checks if the agent relates to a particular service.
func IsServiceAgent(agent *models.Agent, service *models.Service) bool {
	return agent.ServiceID != nil && pointer.GetString(agent.ServiceID) == service.ServiceID
}

// LooksLikeID checks if a string contains a UUID substring in it.
func LooksLikeID(serviceID string) bool {
	return IDRegex.MatchString(serviceID)
}
