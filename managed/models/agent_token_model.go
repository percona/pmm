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

import "time"

//go:generate go tool reform

// AgentToken is a credential pmm-managed issued for a Node's pmm-agent.
//
// Agent credentials used to be Grafana service accounts, which meant enrolling a node
// required Grafana Org Admin and did not work at all where Grafana has no local users, as
// with LDAP plus disable_initial_admin_creation. These are PMM's own: Grafana never sees
// them and never needs to.
//
// Only TokenHash is stored. The token itself is returned once, when it is created.
//
//reform:agent_tokens
type AgentToken struct {
	TokenHash string    `reform:"token_hash,pk"`
	NodeID    string    `reform:"node_id"`
	CreatedAt time.Time `reform:"created_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (t *AgentToken) BeforeInsert() error { //nolint:unparam
	t.CreatedAt = Now()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (t *AgentToken) AfterFind() error { //nolint:unparam
	t.CreatedAt = t.CreatedAt.UTC()
	return nil
}
