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

	"gopkg.in/reform.v1"
)

//go:generate reform

// AgentNode implements many-to-many relationship between Agents and Nodes.
//reform:agent_nodes
type AgentNode struct {
	AgentID   string    `reform:"agent_id"`
	NodeID    string    `reform:"node_id"`
	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
//nolint:unparam
func (s *AgentNode) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
//nolint:unparam
func (s *AgentNode) BeforeUpdate() error {
	panic("AgentNode should not be updated")
}

// AfterFind implements reform.AfterFinder interface.
//nolint:unparam
func (s *AgentNode) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	return nil
}

// check interfaces
var (
	_ reform.BeforeInserter = (*AgentNode)(nil)
	_ reform.BeforeUpdater  = (*AgentNode)(nil)
	_ reform.AfterFinder    = (*AgentNode)(nil)
)
