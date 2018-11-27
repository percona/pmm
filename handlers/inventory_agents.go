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
	"fmt"

	api "github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/inventory"
)

// AgentsServer handles Inventory API requests to manage Agents.
type AgentsServer struct {
	Agents *inventory.AgentsService
}

// ListAgents returns a list of all Agents.
func (s *AgentsServer) ListAgents(ctx context.Context, req *api.ListAgentsRequest) (*api.ListAgentsResponse, error) {
	agents, err := s.Agents.List(ctx)
	if err != nil {
		return nil, err
	}

	res := new(api.ListAgentsResponse)
	for _, agent := range agents {
		switch agent := agent.(type) {
		case *api.NodeExporter:
			res.NodeExporter = append(res.NodeExporter, agent)
		case *api.MySQLdExporter:
			res.MysqldExporter = append(res.MysqldExporter, agent)
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil

}

// GetAgent returns a single Agent by ID.
func (s *AgentsServer) GetAgent(ctx context.Context, req *api.GetAgentRequest) (*api.GetAgentResponse, error) {
	agent, err := s.Agents.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	res := new(api.GetAgentResponse)
	switch agent := agent.(type) {
	case *api.NodeExporter:
		res.Agent = &api.GetAgentResponse_NodeExporter{NodeExporter: agent}
	case *api.MySQLdExporter:
		res.Agent = &api.GetAgentResponse_MysqldExporter{MysqldExporter: agent}
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil

}

// AddNodeExporterAgent adds node_exporter Agent.
func (s *AgentsServer) AddNodeExporterAgent(ctx context.Context, req *api.AddNodeExporterAgentRequest) (*api.AddNodeExporterAgentResponse, error) {
	panic("not implemented")
}

// AddMySQLdExporterAgent adds mysqld_exporter Agent.
func (s *AgentsServer) AddMySQLdExporterAgent(ctx context.Context, req *api.AddMySQLdExporterAgentRequest) (*api.AddMySQLdExporterAgentResponse, error) {
	panic("not implemented")
}

// StartAgent starts Agent.
func (s *AgentsServer) StartAgent(ctx context.Context, req *api.StartAgentRequest) (*api.StartAgentResponse, error) {
	panic("not implemented")
}

// StopAgent stops Agent.
func (s *AgentsServer) StopAgent(ctx context.Context, req *api.StopAgentRequest) (*api.StopAgentResponse, error) {
	panic("not implemented")
}

// RemoveAgent removes Agent.
func (s *AgentsServer) RemoveAgent(ctx context.Context, req *api.RemoveAgentRequest) (*api.RemoveAgentResponse, error) {
	if err := s.Agents.Remove(ctx, req.Id); err != nil {
		return nil, err
	}

	return new(api.RemoveAgentResponse), nil
}

// check interfaces
var (
	_ api.AgentsServer = (*AgentsServer)(nil)
)
