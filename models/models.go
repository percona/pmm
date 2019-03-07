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

// Package models contains generated Reform records and helpers.
package models

import (
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// Now returns current time with database precision.
var Now = func() time.Time {
	return time.Now().Truncate(time.Second).UTC()
}

// PMMAgentsForChangedNode returns pmm-agents IDs that are affected
// by the change of the Node with given ID.
// It may return (nil, nil) if no such pmm-agents are found.
// It returns wrapped reform.ErrNoRows if Service with given ID is not found.
func PMMAgentsForChangedNode(q *reform.Querier, nodeID string) ([]string, error) {
	// TODO Real code.
	// Returning all pmm-agents is currently safe, but not optimal for large number of Agents.
	_ = nodeID

	structs, err := q.SelectAllFrom(AgentTable, "ORDER BY agent_id")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	var res []string
	for _, str := range structs {
		row := str.(*Agent)
		if row.AgentType == PMMAgentType {
			res = append(res, row.AgentID)
		}
	}
	return res, nil
}

// PMMAgentsForChangedService returns pmm-agents IDs that are affected
// by the change of the Service with given ID.
// It may return (nil, nil) if no such pmm-agents are found.
// It returns wrapped reform.ErrNoRows if Service with given ID is not found.
func PMMAgentsForChangedService(q *reform.Querier, serviceID string) ([]string, error) {
	// TODO Real code. We need to returns IDs of pmm-agents that:
	// * run Agents providing insights for this Service;
	// * run Agents providing insights for Node that hosts this Service.
	// Returning all pmm-agents is currently safe, but not optimal for large number of Agents.
	_ = serviceID

	structs, err := q.SelectAllFrom(AgentTable, "ORDER BY agent_id")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	var res []string
	for _, str := range structs {
		row := str.(*Agent)
		if row.AgentType == PMMAgentType {
			res = append(res, row.AgentID)
		}
	}
	return res, nil
}
