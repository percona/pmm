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

package models

import (
	"time"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

//go:generate reform

//reform:agent_nodes
type AgentNode struct {
	AgentID   uint32    `reform:"agent_id"`
	NodeID    uint32    `reform:"node_id"`
	CreatedAt time.Time `reform:"created_at"`
}

// AgentsForNodeID returns agents providing insights for a given node.
func AgentsForNodeID(q *reform.Querier, nodeID uint32) ([]Agent, error) {
	agentNodes, err := q.SelectAllFrom(AgentNodeView, "WHERE node_id = ?", nodeID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	agentIDs := make([]interface{}, len(agentNodes))
	for i, str := range agentNodes {
		agentIDs[i] = str.(*AgentNode).AgentID
	}

	if len(agentIDs) == 0 {
		return []Agent{}, nil
	}

	structs, err := q.FindAllFrom(AgentTable, "id", agentIDs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	agents := make([]Agent, len(structs))
	for i, str := range structs {
		agents[i] = *str.(*Agent)
	}
	return agents, nil
}
