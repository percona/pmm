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

// AgentService implements many-to-many relationship between Agents and Services.
//reform:agent_services
type AgentService struct {
	AgentID   uint32    `reform:"agent_id"`
	ServiceID uint32    `reform:"service_id"`
	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (as *AgentService) BeforeInsert() error {
	now := time.Now().Truncate(time.Microsecond).UTC()
	as.CreatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (as *AgentService) BeforeUpdate() error {
	panic("AgentService should not be updated")
}

// AfterFind implements reform.AfterFinder interface.
func (as *AgentService) AfterFind() error {
	as.CreatedAt = as.CreatedAt.UTC()
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*AgentService)(nil)
	_ reform.BeforeUpdater  = (*AgentService)(nil)
	_ reform.AfterFinder    = (*AgentService)(nil)
)

// AgentsForServiceID returns agents providing insights for a given service.
func AgentsForServiceID(q *reform.Querier, serviceID uint32) ([]Agent, error) {
	agentServices, err := q.SelectAllFrom(AgentServiceView, "WHERE service_id = ?", serviceID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	agentIDs := make([]interface{}, len(agentServices))
	for i, str := range agentServices {
		agentIDs[i] = str.(*AgentService).AgentID
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
