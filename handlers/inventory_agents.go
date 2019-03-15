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

	inventorypb "github.com/percona/pmm/api/inventory"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/services/inventory"
)

type agentsServer struct {
	s  *inventory.AgentsService
	db *reform.DB
}

// NewAgentsServer returns Inventory API handler for managing Agents.
func NewAgentsServer(s *inventory.AgentsService, db *reform.DB) inventorypb.AgentsServer {
	return &agentsServer{
		s:  s,
		db: db,
	}
}

// ListAgents returns a list of Agents for a given filters.
func (s *agentsServer) ListAgents(ctx context.Context, req *inventorypb.ListAgentsRequest) (*inventorypb.ListAgentsResponse, error) {
	filters := inventory.AgentFilters{
		PMMAgentID: req.GetPmmAgentId(),
		NodeID:     req.GetNodeId(),
		ServiceID:  req.GetServiceId(),
	}
	agents, err := s.s.List(ctx, s.db, filters)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.ListAgentsResponse)
	for _, agent := range agents {
		switch agent := agent.(type) {
		case *inventorypb.PMMAgent:
			res.PmmAgent = append(res.PmmAgent, agent)
		case *inventorypb.NodeExporter:
			res.NodeExporter = append(res.NodeExporter, agent)
		case *inventorypb.MySQLdExporter:
			res.MysqldExporter = append(res.MysqldExporter, agent)
		case *inventorypb.RDSExporter:
			res.RdsExporter = append(res.RdsExporter, agent)
		case *inventorypb.ExternalExporter:
			res.ExternalExporter = append(res.ExternalExporter, agent)
		case *inventorypb.MongoDBExporter:
			res.MongodbExporter = append(res.MongodbExporter, agent)
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil
}

// GetAgent returns a single Agent by ID.
func (s *agentsServer) GetAgent(ctx context.Context, req *inventorypb.GetAgentRequest) (*inventorypb.GetAgentResponse, error) {
	agent, err := s.s.Get(ctx, s.db, req.AgentId)
	if err != nil {
		return nil, err
	}

	res := new(inventorypb.GetAgentResponse)
	switch agent := agent.(type) {
	case *inventorypb.PMMAgent:
		res.Agent = &inventorypb.GetAgentResponse_PmmAgent{PmmAgent: agent}
	case *inventorypb.NodeExporter:
		res.Agent = &inventorypb.GetAgentResponse_NodeExporter{NodeExporter: agent}
	case *inventorypb.MySQLdExporter:
		res.Agent = &inventorypb.GetAgentResponse_MysqldExporter{MysqldExporter: agent}
	case *inventorypb.RDSExporter:
		res.Agent = &inventorypb.GetAgentResponse_RdsExporter{RdsExporter: agent}
	case *inventorypb.ExternalExporter:
		res.Agent = &inventorypb.GetAgentResponse_ExternalExporter{ExternalExporter: agent}
	case *inventorypb.MongoDBExporter:
		res.Agent = &inventorypb.GetAgentResponse_MongodbExporter{MongodbExporter: agent}
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil

}

// AddPMMAgent adds pmm-agent Agent.
func (s *agentsServer) AddPMMAgent(ctx context.Context, req *inventorypb.AddPMMAgentRequest) (*inventorypb.AddPMMAgentResponse, error) {
	agent, err := s.s.AddPMMAgent(ctx, s.db, req.RunsOnNodeId)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddPMMAgentResponse{
		PmmAgent: agent,
	}
	return res, nil
}

// AddNodeExporter adds node_exporter Agent.
func (s *agentsServer) AddNodeExporter(ctx context.Context, req *inventorypb.AddNodeExporterRequest) (*inventorypb.AddNodeExporterResponse, error) {
	agent, err := s.s.AddNodeExporter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddNodeExporterResponse{
		NodeExporter: agent,
	}
	return res, nil
}

// AddMySQLdExporter adds mysqld_exporter Agent.
func (s *agentsServer) AddMySQLdExporter(ctx context.Context, req *inventorypb.AddMySQLdExporterRequest) (*inventorypb.AddMySQLdExporterResponse, error) {
	agent, err := s.s.AddMySQLdExporter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMySQLdExporterResponse{
		MysqldExporter: agent,
	}
	return res, nil
}

// AddRDSExporter adds rds_exporter Agent.
func (s *agentsServer) AddRDSExporter(ctx context.Context, req *inventorypb.AddRDSExporterRequest) (*inventorypb.AddRDSExporterResponse, error) {
	panic("not implemented yet")
}

// AddExternalExporter adds external Agent.
func (s *agentsServer) AddExternalExporter(ctx context.Context, req *inventorypb.AddExternalExporterRequest) (*inventorypb.AddExternalExporterResponse, error) {
	panic("not implemented yet")
}

// AddMongoDBExporter adds mongodb_exporter Agent.
func (s *agentsServer) AddMongoDBExporter(ctx context.Context, req *inventorypb.AddMongoDBExporterRequest) (*inventorypb.AddMongoDBExporterResponse, error) {
	agent, err := s.s.AddMongoDBExporter(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMongoDBExporterResponse{
		MongodbExporter: agent,
	}
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
func (s *agentsServer) AddQANMySQLPerfSchemaAgent(ctx context.Context, req *inventorypb.AddQANMySQLPerfSchemaAgentRequest) (*inventorypb.AddQANMySQLPerfSchemaAgentResponse, error) {
	agent, err := s.s.AddQANMySQLPerfSchemaAgent(ctx, s.db, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddQANMySQLPerfSchemaAgentResponse{
		QanMysqlPerfschemaAgent: agent,
	}
	return res, nil
}

// RemoveAgent removes Agent.
func (s *agentsServer) RemoveAgent(ctx context.Context, req *inventorypb.RemoveAgentRequest) (*inventorypb.RemoveAgentResponse, error) {
	if err := s.s.Remove(ctx, s.db, req.AgentId); err != nil {
		return nil, err
	}

	return new(inventorypb.RemoveAgentResponse), nil
}
