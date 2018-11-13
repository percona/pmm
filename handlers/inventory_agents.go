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

	api "github.com/percona/pmm/api/inventory"
)

type AgentsServer struct {
}

func (s *AgentsServer) ListAgents(ctx context.Context, req *api.ListAgentsRequest) (*api.ListAgentsResponse, error) {
	panic("not implemented")
}

func (s *AgentsServer) GetAgent(ctx context.Context, req *api.GetAgentRequest) (*api.GetAgentResponse, error) {
	panic("not implemented")
}

func (s *AgentsServer) AddMySQLdExporterAgent(ctx context.Context, req *api.AddMySQLdExporterAgentRequest) (*api.AddMySQLdExporterAgentResponse, error) {
	panic("not implemented")
}

func (s *AgentsServer) RemoveAgent(ctx context.Context, req *api.RemoveAgentRequest) (*api.RemoveAgentResponse, error) {
	panic("not implemented")
}

// check interfaces
var (
	_ api.AgentsServer = (*AgentsServer)(nil)
)
