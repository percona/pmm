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
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/services/inventory"
)

type agentsServer struct {
	s  *inventory.AgentsService
	db *reform.DB
}

// NewAgentsServer returns Inventory API handler for managing Agents.
func NewAgentsServer(s *inventory.AgentsService, db *reform.DB) api.AgentsServer {
	return &agentsServer{
		s:  s,
		db: db,
	}
}

// ListAgents returns a list of Agents for a given filters.
func (s *agentsServer) ListAgents(ctx context.Context, req *api.ListAgentsRequest) (*api.ListAgentsResponse, error) {
	filters := inventory.AgentFilters{
		RunsOnNodeID: req.GetRunsOnNodeId(),
		NodeID:       req.GetNodeId(),
		ServiceID:    req.GetServiceId(),
	}
	agents, err := s.s.List(ctx, s.db, filters)
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
		case *api.MongoDBExporter:
			res.MongodbExporter = append(res.MongodbExporter, agent)
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil
}

// GetAgent returns a single Agent by ID.
func (s *agentsServer) GetAgent(ctx context.Context, req *api.GetAgentRequest) (*api.GetAgentResponse, error) {
	agent, err := s.s.Get(ctx, s.db, req.AgentId)
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
	case *api.MongoDBExporter:
		res.Agent = &api.GetAgentResponse_MongodbExporter{MongodbExporter: agent}
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil

}

// AddPMMAgent adds pmm-agent Agent.
func (s *agentsServer) AddPMMAgent(ctx context.Context, req *api.AddPMMAgentRequest) (*api.AddPMMAgentResponse, error) {
	agent, err := s.s.AddPMMAgent(ctx, s.db, req.NodeId)
	if err != nil {
		return nil, err
	}

	res := &api.AddPMMAgentResponse{
		PmmAgent: agent,
	}
	return res, nil
}

// AddNodeExporter adds node_exporter Agent.
func (s *agentsServer) AddNodeExporter(ctx context.Context, req *api.AddNodeExporterRequest) (*api.AddNodeExporterResponse, error) {
	agent, err := s.s.AddNodeExporter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &api.AddNodeExporterResponse{
		NodeExporter: agent,
	}
	return res, nil
}

// AddMySQLdExporter adds mysqld_exporter Agent.
func (s *agentsServer) AddMySQLdExporter(ctx context.Context, req *api.AddMySQLdExporterRequest) (*api.AddMySQLdExporterResponse, error) {
	agent, err := s.s.AddMySQLdExporter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &api.AddMySQLdExporterResponse{
		MysqldExporter: agent,
	}
	return res, nil
}

// AddRDSExporter adds rds_exporter Agent.
func (s *agentsServer) AddRDSExporter(ctx context.Context, req *api.AddRDSExporterRequest) (*api.AddRDSExporterResponse, error) {
	panic("not implemented yet")
}

// AddExternalExporter adds external Agent.
func (s *agentsServer) AddExternalExporter(ctx context.Context, req *api.AddExternalExporterRequest) (*api.AddExternalExporterResponse, error) {
	panic("not implemented yet")
}

// AddMongoDBExporter adds mongodb_exporter Agent.
func (s *agentsServer) AddMongoDBExporter(ctx context.Context, req *api.AddMongoDBExporterRequest) (*api.AddMongoDBExporterResponse, error) {
	agent, err := s.s.AddMongoDBExporter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &api.AddMongoDBExporterResponse{
		MongodbExporter: agent,
	}
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
func (s *agentsServer) AddQANMySQLPerfSchemaAgent(ctx context.Context, req *api.AddQANMySQLPerfSchemaAgentRequest) (*api.AddQANMySQLPerfSchemaAgentResponse, error) {
	panic("not implemented yet")
}

// RemoveAgent removes Agent.
func (s *agentsServer) RemoveAgent(ctx context.Context, req *api.RemoveAgentRequest) (*api.RemoveAgentResponse, error) {
	if err := s.s.Remove(ctx, s.db, req.AgentId); err != nil {
		return nil, err
	}

	return new(api.RemoveAgentResponse), nil
}
