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

package models

import (
	"github.com/AlekSi/pointer"
	"gopkg.in/reform.v1"
)

// maxAgentParentDepth bounds the walk up the pmm_agent_id chain. The chain is one level
// deep in practice; the bound only stops a cycle in corrupted data from looping forever.
const maxAgentParentDepth = 8

// FindNodeIDForService returns the Node a Service belongs to.
func FindNodeIDForService(q *reform.Querier, serviceID string) (string, error) {
	service, err := FindServiceByID(q, serviceID)
	if err != nil {
		return "", err
	}

	return service.NodeID, nil
}

// FindNodeIDForAgent returns the Node an Agent belongs to: the node it runs on, the node it
// monitors, the node of the service it monitors, or the node of its parent pmm-agent.
// It returns an empty string for agents tied to no node at all, such as an external exporter.
func FindNodeIDForAgent(q *reform.Querier, agentID string) (string, error) {
	for range maxAgentParentDepth {
		agent, err := FindAgentByID(q, agentID)
		if err != nil {
			return "", err
		}

		switch {
		case agent.RunsOnNodeID != nil:
			return pointer.GetString(agent.RunsOnNodeID), nil
		case agent.NodeID != nil:
			return pointer.GetString(agent.NodeID), nil
		case agent.ServiceID != nil:
			return FindNodeIDForService(q, pointer.GetString(agent.ServiceID))
		case agent.PMMAgentID != nil:
			agentID = pointer.GetString(agent.PMMAgentID)
		default:
			return "", nil
		}
	}

	return "", nil
}
