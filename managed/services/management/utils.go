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
	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/managed/models"
)

// IsNonExporterAgent checks if agent is not an exporter and if it runs on the same node as the service (p.e. pmm-agent).
func IsNonExporterAgent(agent *models.Agent, service *models.Service) bool {
	return agent.ServiceID == nil && pointer.GetString(agent.RunsOnNodeID) == service.NodeID
}

// IsVMAgent checks if the agent is an vmagent and if it runs on the same node as the service.
func IsVMAgent(agent *models.Agent, service *models.Service) bool {
	return pointer.GetString(agent.NodeID) == service.NodeID && agent.AgentType == models.VMAgentType
}

// IsExporterAgent checks if agent is an exporter and if it runs on the same node as the service.
func IsExporterAgent(agent *models.Agent, service *models.Service) bool {
	return agent.ServiceID != nil && pointer.GetString(agent.ServiceID) == service.ServiceID
}
