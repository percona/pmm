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

	"github.com/AlekSi/pointer"
	api "github.com/percona/pmm/api/inventory"

	"github.com/percona/pmm-managed/services/inventory"
)

// AgentsServer handles Inventory API requests to manage Agents.
type AgentsServer struct {
	Agents *inventory.AgentsService
}

// ListAgents returns a list of Agents for a given filters.
func (s *AgentsServer) ListAgents(ctx context.Context, req *api.ListAgentsRequest) (*api.ListAgentsResponse, error) {
	filters := inventory.AgentFilters{
		RunsOnNodeID: req.GetRunsOnNodeId(),
		NodeID:       req.GetNodeId(),
		ServiceID:    req.GetServiceId(),
	}
	agents, err := s.Agents.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := new(api.ListAgentsResponse)
	for _, agent := range agents {
		switch agent := agent.(type) {
		case *api.PMMAgent:
			res.PmmAgent = append(res.PmmAgent, agent)
		case *api.NodeExporter:
			res.NodeExporter = append(res.NodeExporter, agent)
		case *api.MySQLdExporter:
			res.MysqldExporter = append(res.MysqldExporter, agent)
		case *api.RDSExporter:
			res.RdsExporter = append(res.RdsExporter, agent)
		case *api.ExternalExporter:
			res.ExternalExporter = append(res.ExternalExporter, agent)
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil
}

// GetAgent returns a single Agent by ID.
func (s *AgentsServer) GetAgent(ctx context.Context, req *api.GetAgentRequest) (*api.GetAgentResponse, error) {
	agent, err := s.Agents.Get(ctx, req.AgentId)
	if err != nil {
		return nil, err
	}

	res := new(api.GetAgentResponse)
	switch agent := agent.(type) {
	case *api.PMMAgent:
		res.Agent = &api.GetAgentResponse_PmmAgent{PmmAgent: agent}
	case *api.NodeExporter:
		res.Agent = &api.GetAgentResponse_NodeExporter{NodeExporter: agent}
	case *api.MySQLdExporter:
		res.Agent = &api.GetAgentResponse_MysqldExporter{MysqldExporter: agent}
	case *api.RDSExporter:
		res.Agent = &api.GetAgentResponse_RdsExporter{RdsExporter: agent}
	case *api.ExternalExporter:
		res.Agent = &api.GetAgentResponse_ExternalExporter{ExternalExporter: agent}
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil

}

// AddPMMAgent adds pmm-agent Agent.
func (s *AgentsServer) AddPMMAgent(ctx context.Context, req *api.AddPMMAgentRequest) (*api.AddPMMAgentResponse, error) {
	agent, err := s.Agents.AddPMMAgent(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}

	res := &api.AddPMMAgentResponse{
		PmmAgent: agent,
	}
	return res, nil
}

// AddNodeExporter adds node_exporter Agent.
func (s *AgentsServer) AddNodeExporter(ctx context.Context, req *api.AddNodeExporterRequest) (*api.AddNodeExporterResponse, error) {
	agent, err := s.Agents.AddNodeExporter(ctx, req.NodeId)
	if err != nil {
		return nil, err
	}

	res := &api.AddNodeExporterResponse{
		NodeExporter: agent,
	}
	return res, nil
}

// AddMySQLdExporter adds mysqld_exporter Agent.
func (s *AgentsServer) AddMySQLdExporter(ctx context.Context, req *api.AddMySQLdExporterRequest) (*api.AddMySQLdExporterResponse, error) {
	username := pointer.ToStringOrNil(req.Username)
	password := pointer.ToStringOrNil(req.Password)
	agent, err := s.Agents.AddMySQLdExporter(ctx, req.RunsOnNodeId, req.ServiceId, username, password)
	if err != nil {
		return nil, err
	}

	res := &api.AddMySQLdExporterResponse{
		MysqldExporter: agent,
	}
	return res, nil
}

// AddRDSExporter adds rds_exporter Agent.
func (s *AgentsServer) AddRDSExporter(ctx context.Context, req *api.AddRDSExporterRequest) (*api.AddRDSExporterResponse, error) {
	panic("not implemented yet")
}

// AddExternalExporter adds external Agent.
func (s *AgentsServer) AddExternalExporter(ctx context.Context, req *api.AddExternalExporterRequest) (*api.AddExternalExporterResponse, error) {
	panic("not implemented yet")
}

// RemoveAgent removes Agent.
func (s *AgentsServer) RemoveAgent(ctx context.Context, req *api.RemoveAgentRequest) (*api.RemoveAgentResponse, error) {
	if err := s.Agents.Remove(ctx, req.AgentId); err != nil {
		return nil, err
	}

	return new(api.RemoveAgentResponse), nil
}

// check interfaces
var (
	_ api.AgentsServer = (*AgentsServer)(nil)
)
