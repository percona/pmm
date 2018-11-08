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

package handlers

import (
	"context"

	"github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/agents"
)

type AgentsServer struct {
	Store  *agents.Store
	Agents map[uint32]*agents.Conn
}

func (s *AgentsServer) ListAgents(ctx context.Context, req *inventory.ListAgentsRequest) (*inventory.ListAgentsResponse, error) {
	panic("not implemented")
}

func (s *AgentsServer) GetAgent(ctx context.Context, req *inventory.GetAgentRequest) (*inventory.GetAgentResponse, error) {
	panic("not implemented")
}

func (s *AgentsServer) AddMySQLdExporterAgent(ctx context.Context, req *inventory.AddMySQLdExporterAgentRequest) (*inventory.AddMySQLdExporterAgentResponse, error) {
	return s.Store.AddMySQLdExporter(req), nil
}

func (s *AgentsServer) RemoveAgent(ctx context.Context, req *inventory.RemoveAgentRequest) (*inventory.RemoveAgentResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ inventory.AgentsServer = (*AgentsServer)(nil)
)
