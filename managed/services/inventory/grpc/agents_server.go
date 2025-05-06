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

// Package grpc contains inventory gRPC API implementation.
package grpc

import (
	"context"
	"fmt"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
)

type agentsServer struct {
	s *inventory.AgentsService

	inventoryv1.UnimplementedAgentsServiceServer
}

// NewAgentsServer returns Inventory API handler for managing Agents.
func NewAgentsServer(s *inventory.AgentsService) inventoryv1.AgentsServiceServer { //nolint:ireturn
	return &agentsServer{s: s}
}

var agentTypes = map[inventoryv1.AgentType]models.AgentType{
	inventoryv1.AgentType_AGENT_TYPE_PMM_AGENT:                          models.PMMAgentType,
	inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER:                      models.NodeExporterType,
	inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER:                    models.MySQLdExporterType,
	inventoryv1.AgentType_AGENT_TYPE_MONGODB_EXPORTER:                   models.MongoDBExporterType,
	inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER:                  models.PostgresExporterType,
	inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER:                    models.ValkeyExporterType,
	inventoryv1.AgentType_AGENT_TYPE_PROXYSQL_EXPORTER:                  models.ProxySQLExporterType,
	inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT:         models.QANMySQLPerfSchemaAgentType,
	inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_SLOWLOG_AGENT:            models.QANMySQLSlowlogAgentType,
	inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT:         models.QANMongoDBProfilerAgentType,
	inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT:  models.QANPostgreSQLPgStatementsAgentType,
	inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATMONITOR_AGENT: models.QANPostgreSQLPgStatMonitorAgentType,
	inventoryv1.AgentType_AGENT_TYPE_RDS_EXPORTER:                       models.RDSExporterType,
	inventoryv1.AgentType_AGENT_TYPE_EXTERNAL_EXPORTER:                  models.ExternalExporterType,
	inventoryv1.AgentType_AGENT_TYPE_AZURE_DATABASE_EXPORTER:            models.AzureDatabaseExporterType,
	inventoryv1.AgentType_AGENT_TYPE_VM_AGENT:                           models.VMAgentType,
	inventoryv1.AgentType_AGENT_TYPE_NOMAD_AGENT:                        models.NomadAgentType,
}

func agentType(req *inventoryv1.ListAgentsRequest) *models.AgentType {
	if req.AgentType == inventoryv1.AgentType_AGENT_TYPE_UNSPECIFIED {
		return nil
	}
	agentType := agentTypes[req.AgentType]
	return &agentType
}

// ListAgents returns a list of Agents for a given filters.
func (s *agentsServer) ListAgents(ctx context.Context, req *inventoryv1.ListAgentsRequest) (*inventoryv1.ListAgentsResponse, error) {
	filters := models.AgentFilters{
		PMMAgentID: req.GetPmmAgentId(),
		NodeID:     req.GetNodeId(),
		ServiceID:  req.GetServiceId(),
		AgentType:  agentType(req),
	}
	agents, err := s.s.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ListAgentsResponse{}
	for _, agent := range agents {
		switch agent := agent.(type) {
		case *inventoryv1.PMMAgent:
			res.PmmAgent = append(res.PmmAgent, agent)
		case *inventoryv1.NodeExporter:
			res.NodeExporter = append(res.NodeExporter, agent)
		case *inventoryv1.MySQLdExporter:
			res.MysqldExporter = append(res.MysqldExporter, agent)
		case *inventoryv1.MongoDBExporter:
			res.MongodbExporter = append(res.MongodbExporter, agent)
		case *inventoryv1.QANMySQLPerfSchemaAgent:
			res.QanMysqlPerfschemaAgent = append(res.QanMysqlPerfschemaAgent, agent)
		case *inventoryv1.QANMySQLSlowlogAgent:
			res.QanMysqlSlowlogAgent = append(res.QanMysqlSlowlogAgent, agent)
		case *inventoryv1.PostgresExporter:
			res.PostgresExporter = append(res.PostgresExporter, agent)
		case *inventoryv1.ValkeyExporter:
			res.ValkeyExporter = append(res.ValkeyExporter, agent)
		case *inventoryv1.QANMongoDBProfilerAgent:
			res.QanMongodbProfilerAgent = append(res.QanMongodbProfilerAgent, agent)
		case *inventoryv1.ProxySQLExporter:
			res.ProxysqlExporter = append(res.ProxysqlExporter, agent)
		case *inventoryv1.QANPostgreSQLPgStatementsAgent:
			res.QanPostgresqlPgstatementsAgent = append(res.QanPostgresqlPgstatementsAgent, agent)
		case *inventoryv1.QANPostgreSQLPgStatMonitorAgent:
			res.QanPostgresqlPgstatmonitorAgent = append(res.QanPostgresqlPgstatmonitorAgent, agent)
		case *inventoryv1.RDSExporter:
			res.RdsExporter = append(res.RdsExporter, agent)
		case *inventoryv1.ExternalExporter:
			res.ExternalExporter = append(res.ExternalExporter, agent)
		case *inventoryv1.AzureDatabaseExporter:
			res.AzureDatabaseExporter = append(res.AzureDatabaseExporter, agent)
		case *inventoryv1.VMAgent:
			res.VmAgent = append(res.VmAgent, agent)
		case *inventoryv1.NomadAgent:
			res.NomadAgent = append(res.NomadAgent, agent)
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil
}

// GetAgent returns a single Agent by ID.
func (s *agentsServer) GetAgent(ctx context.Context, req *inventoryv1.GetAgentRequest) (*inventoryv1.GetAgentResponse, error) {
	agent, err := s.s.Get(ctx, req.GetAgentId())
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.GetAgentResponse{}
	switch agent := agent.(type) {
	case *inventoryv1.PMMAgent:
		res.Agent = &inventoryv1.GetAgentResponse_PmmAgent{PmmAgent: agent}
	case *inventoryv1.NodeExporter:
		res.Agent = &inventoryv1.GetAgentResponse_NodeExporter{NodeExporter: agent}
	case *inventoryv1.MySQLdExporter:
		res.Agent = &inventoryv1.GetAgentResponse_MysqldExporter{MysqldExporter: agent}
	case *inventoryv1.MongoDBExporter:
		res.Agent = &inventoryv1.GetAgentResponse_MongodbExporter{MongodbExporter: agent}
	case *inventoryv1.QANMySQLPerfSchemaAgent:
		res.Agent = &inventoryv1.GetAgentResponse_QanMysqlPerfschemaAgent{QanMysqlPerfschemaAgent: agent}
	case *inventoryv1.QANMySQLSlowlogAgent:
		res.Agent = &inventoryv1.GetAgentResponse_QanMysqlSlowlogAgent{QanMysqlSlowlogAgent: agent}
	case *inventoryv1.PostgresExporter:
		res.Agent = &inventoryv1.GetAgentResponse_PostgresExporter{PostgresExporter: agent}
	case *inventoryv1.ValkeyExporter:
		res.Agent = &inventoryv1.GetAgentResponse_ValkeyExporter{ValkeyExporter: agent}
	case *inventoryv1.QANMongoDBProfilerAgent:
		res.Agent = &inventoryv1.GetAgentResponse_QanMongodbProfilerAgent{QanMongodbProfilerAgent: agent}
	case *inventoryv1.ProxySQLExporter:
		res.Agent = &inventoryv1.GetAgentResponse_ProxysqlExporter{ProxysqlExporter: agent}
	case *inventoryv1.QANPostgreSQLPgStatementsAgent:
		res.Agent = &inventoryv1.GetAgentResponse_QanPostgresqlPgstatementsAgent{QanPostgresqlPgstatementsAgent: agent}
	case *inventoryv1.QANPostgreSQLPgStatMonitorAgent:
		res.Agent = &inventoryv1.GetAgentResponse_QanPostgresqlPgstatmonitorAgent{QanPostgresqlPgstatmonitorAgent: agent}
	case *inventoryv1.RDSExporter:
		res.Agent = &inventoryv1.GetAgentResponse_RdsExporter{RdsExporter: agent}
	case *inventoryv1.ExternalExporter:
		res.Agent = &inventoryv1.GetAgentResponse_ExternalExporter{ExternalExporter: agent}
	case *inventoryv1.AzureDatabaseExporter:
		res.Agent = &inventoryv1.GetAgentResponse_AzureDatabaseExporter{AzureDatabaseExporter: agent}
	case *inventoryv1.VMAgent:
		// skip it, fix later if needed.
	case *inventoryv1.NomadAgent:
		res.Agent = &inventoryv1.GetAgentResponse_NomadAgent{NomadAgent: agent}
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil
}

// GetAgentLogs returns Agent logs by ID.
func (s *agentsServer) GetAgentLogs(ctx context.Context, req *inventoryv1.GetAgentLogsRequest) (*inventoryv1.GetAgentLogsResponse, error) {
	logs, agentConfigLogLinesCount, err := s.s.Logs(ctx, req.GetAgentId(), req.GetLimit())
	if err != nil {
		return nil, err
	}

	return &inventoryv1.GetAgentLogsResponse{
		Logs:                     logs,
		AgentConfigLogLinesCount: agentConfigLogLinesCount,
	}, nil
}

// AddAgent adds an Agent.
func (s *agentsServer) AddAgent(ctx context.Context, req *inventoryv1.AddAgentRequest) (*inventoryv1.AddAgentResponse, error) {
	switch req.Agent.(type) {
	case *inventoryv1.AddAgentRequest_PmmAgent:
		return s.s.AddPMMAgent(ctx, req.GetPmmAgent())
	case *inventoryv1.AddAgentRequest_NodeExporter:
		return s.s.AddNodeExporter(ctx, req.GetNodeExporter())
	case *inventoryv1.AddAgentRequest_MysqldExporter:
		return s.s.AddMySQLdExporter(ctx, req.GetMysqldExporter())
	case *inventoryv1.AddAgentRequest_MongodbExporter:
		return s.s.AddMongoDBExporter(ctx, req.GetMongodbExporter())
	case *inventoryv1.AddAgentRequest_PostgresExporter:
		return s.s.AddPostgresExporter(ctx, req.GetPostgresExporter())
	case *inventoryv1.AddAgentRequest_ValkeyExporter:
		return nil, fmt.Errorf("Valkey Exporter is not supported yet")
	case *inventoryv1.AddAgentRequest_ProxysqlExporter:
		return s.s.AddProxySQLExporter(ctx, req.GetProxysqlExporter())
	case *inventoryv1.AddAgentRequest_RdsExporter:
		return s.s.AddRDSExporter(ctx, req.GetRdsExporter())
	case *inventoryv1.AddAgentRequest_ExternalExporter:
		return s.s.AddExternalExporter(ctx, req.GetExternalExporter())
	case *inventoryv1.AddAgentRequest_AzureDatabaseExporter:
		return s.s.AddAzureDatabaseExporter(ctx, req.GetAzureDatabaseExporter())
	case *inventoryv1.AddAgentRequest_QanMysqlPerfschemaAgent:
		return s.s.AddQANMySQLPerfSchemaAgent(ctx, req.GetQanMysqlPerfschemaAgent())
	case *inventoryv1.AddAgentRequest_QanMysqlSlowlogAgent:
		return s.s.AddQANMySQLSlowlogAgent(ctx, req.GetQanMysqlSlowlogAgent())
	case *inventoryv1.AddAgentRequest_QanMongodbProfilerAgent:
		return s.s.AddQANMongoDBProfilerAgent(ctx, req.GetQanMongodbProfilerAgent())
	case *inventoryv1.AddAgentRequest_QanPostgresqlPgstatementsAgent:
		return s.s.AddQANPostgreSQLPgStatementsAgent(ctx, req.GetQanPostgresqlPgstatementsAgent())
	case *inventoryv1.AddAgentRequest_QanPostgresqlPgstatmonitorAgent:
		return s.s.AddQANPostgreSQLPgStatMonitorAgent(ctx, req.GetQanPostgresqlPgstatmonitorAgent())
	default:
		return nil, fmt.Errorf("invalid request %v", req.Agent)
	}
}

// ChangeAgent allows to change some Agent attributes.
func (s *agentsServer) ChangeAgent(ctx context.Context, req *inventoryv1.ChangeAgentRequest) (*inventoryv1.ChangeAgentResponse, error) {
	agentID := req.GetAgentId() //nolint:typecheck

	switch req.Agent.(type) {
	case *inventoryv1.ChangeAgentRequest_NodeExporter:
		return s.s.ChangeNodeExporter(ctx, agentID, req.GetNodeExporter())
	case *inventoryv1.ChangeAgentRequest_MysqldExporter:
		return s.s.ChangeMySQLdExporter(ctx, agentID, req.GetMysqldExporter())
	case *inventoryv1.ChangeAgentRequest_MongodbExporter:
		return s.s.ChangeMongoDBExporter(ctx, agentID, req.GetMongodbExporter())
	case *inventoryv1.ChangeAgentRequest_PostgresExporter:
		return s.s.ChangePostgresExporter(ctx, agentID, req.GetPostgresExporter())
	case *inventoryv1.ChangeAgentRequest_ValkeyExporter:
		return nil, fmt.Errorf("Valkey Exporter is not supported yet")
	case *inventoryv1.ChangeAgentRequest_ProxysqlExporter:
		return s.s.ChangeProxySQLExporter(ctx, agentID, req.GetProxysqlExporter())
	case *inventoryv1.ChangeAgentRequest_RdsExporter:
		return s.s.ChangeRDSExporter(ctx, agentID, req.GetRdsExporter())
	case *inventoryv1.ChangeAgentRequest_ExternalExporter:
		return s.s.ChangeExternalExporter(ctx, agentID, req.GetExternalExporter())
	case *inventoryv1.ChangeAgentRequest_AzureDatabaseExporter:
		return s.s.ChangeAzureDatabaseExporter(ctx, agentID, req.GetAzureDatabaseExporter())
	case *inventoryv1.ChangeAgentRequest_QanMysqlPerfschemaAgent:
		return s.s.ChangeQANMySQLPerfSchemaAgent(ctx, agentID, req.GetQanMysqlPerfschemaAgent())
	case *inventoryv1.ChangeAgentRequest_QanMysqlSlowlogAgent:
		return s.s.ChangeQANMySQLSlowlogAgent(ctx, agentID, req.GetQanMysqlSlowlogAgent())
	case *inventoryv1.ChangeAgentRequest_QanMongodbProfilerAgent:
		return s.s.ChangeQANMongoDBProfilerAgent(ctx, agentID, req.GetQanMongodbProfilerAgent())
	case *inventoryv1.ChangeAgentRequest_QanPostgresqlPgstatementsAgent:
		return s.s.ChangeQANPostgreSQLPgStatementsAgent(ctx, agentID, req.GetQanPostgresqlPgstatementsAgent())
	case *inventoryv1.ChangeAgentRequest_QanPostgresqlPgstatmonitorAgent:
		return s.s.ChangeQANPostgreSQLPgStatMonitorAgent(ctx, agentID, req.GetQanPostgresqlPgstatmonitorAgent())
	case *inventoryv1.ChangeAgentRequest_NomadAgent:
		return s.s.ChangeNomadAgent(ctx, agentID, req.GetNomadAgent())
	default:
		return nil, fmt.Errorf("invalid request %v", req.Agent)
	}
}

// RemoveAgent removes the Agent.
func (s *agentsServer) RemoveAgent(ctx context.Context, req *inventoryv1.RemoveAgentRequest) (*inventoryv1.RemoveAgentResponse, error) {
	if err := s.s.Remove(ctx, req.GetAgentId(), req.GetForce()); err != nil {
		return nil, err
	}

	return &inventoryv1.RemoveAgentResponse{}, nil
}
