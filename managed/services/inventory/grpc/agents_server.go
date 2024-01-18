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
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil
}

// GetAgent returns a single Agent by ID.
func (s *agentsServer) GetAgent(ctx context.Context, req *inventoryv1.GetAgentRequest) (*inventoryv1.GetAgentResponse, error) {
	agent, err := s.s.Get(ctx, req.AgentId)
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
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil
}

// GetAgentLogs returns Agent logs by ID.
func (s *agentsServer) GetAgentLogs(ctx context.Context, req *inventoryv1.GetAgentLogsRequest) (*inventoryv1.GetAgentLogsResponse, error) {
	logs, agentConfigLogLinesCount, err := s.s.Logs(ctx, req.AgentId, req.Limit)
	if err != nil {
		return nil, err
	}

	return &inventoryv1.GetAgentLogsResponse{
		Logs:                     logs,
		AgentConfigLogLinesCount: agentConfigLogLinesCount,
	}, nil
}

// AddPMMAgent adds pmm-agent Agent.
func (s *agentsServer) addPMMAgent(ctx context.Context, params *inventoryv1.AddPMMAgentParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddPMMAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_PmmAgent{
			PmmAgent: agent,
		},
	}
	return res, nil
}

// AddAgent adds an Agent.
func (s *agentsServer) AddAgent(ctx context.Context, req *inventoryv1.AddAgentRequest) (*inventoryv1.AddAgentResponse, error) {
	switch req.Exporter.(type) {
	case *inventoryv1.AddAgentRequest_PmmAgent:
		return s.addPMMAgent(ctx, req.GetPmmAgent())
	case *inventoryv1.AddAgentRequest_NodeExporter:
		return s.addNodeExporter(ctx, req.GetNodeExporter())
	case *inventoryv1.AddAgentRequest_MysqldExporter:
		return s.addMySQLdExporter(ctx, req.GetMysqldExporter())
	case *inventoryv1.AddAgentRequest_MongodbExporter:
		return s.addMongoDBExporter(ctx, req.GetMongodbExporter())
	case *inventoryv1.AddAgentRequest_PostgresExporter:
		return s.addPostgresExporter(ctx, req.GetPostgresExporter())
	case *inventoryv1.AddAgentRequest_ProxysqlExporter:
		return s.addProxySQLExporter(ctx, req.GetProxysqlExporter())
	case *inventoryv1.AddAgentRequest_RdsExporter:
		return s.addRDSExporter(ctx, req.GetRdsExporter())
	case *inventoryv1.AddAgentRequest_ExternalExporter:
		return s.addExternalExporter(ctx, req.GetExternalExporter())
	case *inventoryv1.AddAgentRequest_AzureDatabaseExporter:
		return s.addAzureDatabaseExporter(ctx, req.GetAzureDatabaseExporter())
	case *inventoryv1.AddAgentRequest_QanMysqlPerfschemaAgent:
		return s.addQANMySQLPerfSchemaAgent(ctx, req.GetQanMysqlPerfschemaAgent())
	case *inventoryv1.AddAgentRequest_QanMysqlSlowlogAgent:
		return s.addQANMySQLSlowlogAgent(ctx, req.GetQanMysqlSlowlogAgent())
	case *inventoryv1.AddAgentRequest_QanMongodbProfilerAgent:
		return s.addQANMongoDBProfilerAgent(ctx, req.GetQanMongodbProfilerAgent())
	case *inventoryv1.AddAgentRequest_QanPostgresqlPgstatementsAgent:
		return s.addQANPostgreSQLPgStatementsAgent(ctx, req.GetQanPostgresqlPgstatementsAgent())
	case *inventoryv1.AddAgentRequest_QanPostgresqlPgstatmonitorAgent:
		return s.addQANPostgreSQLPgStatMonitorAgent(ctx, req.GetQanPostgresqlPgstatmonitorAgent())
	default:
		return nil, fmt.Errorf("invalid request %v", req.Exporter)
	}
}

// ChangeAgent allows to change some Agent attributes.
func (s *agentsServer) ChangeAgent(ctx context.Context, req *inventoryv1.ChangeAgentRequest) (*inventoryv1.ChangeAgentResponse, error) {
	switch req.Agent.(type) {
	case *inventoryv1.ChangeAgentRequest_NodeExporter:
		return s.changeNodeExporter(ctx, req.GetNodeExporter())
	case *inventoryv1.ChangeAgentRequest_MysqldExporter:
		return s.changeMySQLdExporter(ctx, req.GetMysqldExporter())
	case *inventoryv1.ChangeAgentRequest_MongodbExporter:
		return s.changeMongoDBExporter(ctx, req.GetMongodbExporter())
	case *inventoryv1.ChangeAgentRequest_PostgresExporter:
		return s.changePostgresExporter(ctx, req.GetPostgresExporter())
	case *inventoryv1.ChangeAgentRequest_ProxysqlExporter:
		return s.changeProxySQLExporter(ctx, req.GetProxysqlExporter())
	case *inventoryv1.ChangeAgentRequest_RdsExporter:
		return s.changeRDSExporter(ctx, req.GetRdsExporter())
	case *inventoryv1.ChangeAgentRequest_ExternalExporter:
		return s.changeExternalExporter(ctx, req.GetExternalExporter())
	case *inventoryv1.ChangeAgentRequest_AzureDatabaseExporter:
		return s.changeAzureDatabaseExporter(ctx, req.GetAzureDatabaseExporter())
	case *inventoryv1.ChangeAgentRequest_QanMysqlPerfschemaAgent:
		return s.changeQANMySQLPerfSchemaAgent(ctx, req.GetQanMysqlPerfschemaAgent())
	case *inventoryv1.ChangeAgentRequest_QanMysqlSlowlogAgent:
		return s.changeQANMySQLSlowlogAgent(ctx, req.GetQanMysqlSlowlogAgent())
	case *inventoryv1.ChangeAgentRequest_QanMongodbProfilerAgent:
		return s.changeQANMongoDBProfilerAgent(ctx, req.GetQanMongodbProfilerAgent())
	case *inventoryv1.ChangeAgentRequest_QanPostgresqlPgstatementsAgent:
		return s.changeQANPostgreSQLPgStatementsAgent(ctx, req.GetQanPostgresqlPgstatementsAgent())
	case *inventoryv1.ChangeAgentRequest_QanPostgresqlPgstatmonitorAgent:
		return s.changeQANPostgreSQLPgStatMonitorAgent(ctx, req.GetQanPostgresqlPgstatmonitorAgent())
	default:
		return nil, fmt.Errorf("invalid request %v", req.Agent)
	}
}

// addNodeExporter adds node_exporter Agent.
func (s *agentsServer) addNodeExporter(ctx context.Context, params *inventoryv1.AddNodeExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddNodeExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_NodeExporter{
			NodeExporter: agent,
		},
	}
	return res, nil
}

// ChangeNodeExporter changes disabled flag and custom labels of node_exporter Agent.
func (s *agentsServer) changeNodeExporter(ctx context.Context, params *inventoryv1.ChangeNodeExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeNodeExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_NodeExporter{
			NodeExporter: agent,
		},
	}
	return res, nil
}

// addMySQLdExporter adds mysqld_exporter Agent.
func (s *agentsServer) addMySQLdExporter(ctx context.Context, params *inventoryv1.AddMySQLdExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, tableCount, err := s.s.AddMySQLdExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_MysqldExporter{
			MysqldExporter: agent,
		},
		TableCount: tableCount,
	}
	return res, nil
}

// ChangeMySQLdExporter changes disabled flag and custom labels of mysqld_exporter Agent.
func (s *agentsServer) changeMySQLdExporter(ctx context.Context, params *inventoryv1.ChangeMySQLdExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeMySQLdExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_MysqldExporter{
			MysqldExporter: agent,
		},
	}
	return res, nil
}

// addMongoDBExporter adds mongodb_exporter Agent.
func (s *agentsServer) addMongoDBExporter(ctx context.Context, params *inventoryv1.AddMongoDBExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddMongoDBExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_MongodbExporter{
			MongodbExporter: agent,
		},
	}
	return res, nil
}

// ChangeMongoDBExporter changes disabled flag and custom labels of mongo_exporter Agent.
func (s *agentsServer) changeMongoDBExporter(ctx context.Context, params *inventoryv1.ChangeMongoDBExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeMongoDBExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_MongodbExporter{
			MongodbExporter: agent,
		},
	}
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (s *agentsServer) addQANMySQLPerfSchemaAgent(ctx context.Context, params *inventoryv1.AddQANMySQLPerfSchemaAgentParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddQANMySQLPerfSchemaAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_QanMysqlPerfschemaAgent{
			QanMysqlPerfschemaAgent: agent,
		},
	}
	return res, nil
}

// ChangeQANMySQLPerfSchemaAgent changes disabled flag and custom labels of MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (s *agentsServer) changeQANMySQLPerfSchemaAgent(ctx context.Context, params *inventoryv1.ChangeQANMySQLPerfSchemaAgentParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeQANMySQLPerfSchemaAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMysqlPerfschemaAgent{
			QanMysqlPerfschemaAgent: agent,
		},
	}
	return res, nil
}

// AddQANMySQLSlowlogAgent adds MySQL Slowlog QAN Agent.
//
//nolint:lll
func (s *agentsServer) addQANMySQLSlowlogAgent(ctx context.Context, params *inventoryv1.AddQANMySQLSlowlogAgentParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddQANMySQLSlowlogAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_QanMysqlSlowlogAgent{
			QanMysqlSlowlogAgent: agent,
		},
	}
	return res, nil
}

// ChangeQANMySQLSlowlogAgent changes disabled flag and custom labels of MySQL Slowlog QAN Agent.
//
//nolint:lll
func (s *agentsServer) changeQANMySQLSlowlogAgent(ctx context.Context, params *inventoryv1.ChangeQANMySQLSlowlogAgentParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeQANMySQLSlowlogAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMysqlSlowlogAgent{
			QanMysqlSlowlogAgent: agent,
		},
	}
	return res, nil
}

// addPostgresExporter adds postgres_exporter Agent.
func (s *agentsServer) addPostgresExporter(ctx context.Context, params *inventoryv1.AddPostgresExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddPostgresExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_PostgresExporter{
			PostgresExporter: agent,
		},
	}
	return res, nil
}

// ChangePostgresExporter changes disabled flag and custom labels of postgres_exporter Agent.
func (s *agentsServer) changePostgresExporter(ctx context.Context, params *inventoryv1.ChangePostgresExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangePostgresExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_PostgresExporter{
			PostgresExporter: agent,
		},
	}
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
//
//nolint:lll
func (s *agentsServer) addQANMongoDBProfilerAgent(ctx context.Context, params *inventoryv1.AddQANMongoDBProfilerAgentParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddQANMongoDBProfilerAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: agent,
		},
	}
	return res, nil
}

// ChangeQANMongoDBProfilerAgent changes disabled flag and custom labels of MongoDB Profiler QAN Agent.
//
//nolint:lll
func (s *agentsServer) changeQANMongoDBProfilerAgent(ctx context.Context, params *inventoryv1.ChangeQANMongoDBProfilerAgentParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeQANMongoDBProfilerAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: agent,
		},
	}
	return res, nil
}

// addProxySQLExporter adds proxysql_exporter Agent.
func (s *agentsServer) addProxySQLExporter(ctx context.Context, params *inventoryv1.AddProxySQLExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddProxySQLExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_ProxysqlExporter{
			ProxysqlExporter: agent,
		},
	}
	return res, nil
}

// ChangeProxySQLExporter changes disabled flag and custom labels of proxysql_exporter Agent.
func (s *agentsServer) changeProxySQLExporter(ctx context.Context, params *inventoryv1.ChangeProxySQLExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeProxySQLExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ProxysqlExporter{
			ProxysqlExporter: agent,
		},
	}
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
func (s *agentsServer) addQANPostgreSQLPgStatementsAgent(ctx context.Context, params *inventoryv1.AddQANPostgreSQLPgStatementsAgentParams) (*inventoryv1.AddAgentResponse, error) { //nolint:lll
	agent, err := s.s.AddQANPostgreSQLPgStatementsAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: agent,
		},
	}
	return res, nil
}

// ChangeQANPostgreSQLPgStatementsAgent changes disabled flag and custom labels of PostgreSQL Pg stat statements QAN Agent.
func (s *agentsServer) changeQANPostgreSQLPgStatementsAgent(ctx context.Context, params *inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	agent, err := s.s.ChangeQANPostgreSQLPgStatementsAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: agent,
		},
	}
	return res, nil
}

// AddQANPostgreSQLPgStatMonitorAgent adds PostgreSQL Pg stat monitor QAN Agent.
func (s *agentsServer) addQANPostgreSQLPgStatMonitorAgent(ctx context.Context, params *inventoryv1.AddQANPostgreSQLPgStatMonitorAgentParams) (*inventoryv1.AddAgentResponse, error) { //nolint:lll
	agent, err := s.s.AddQANPostgreSQLPgStatMonitorAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_QanPostgresqlPgstatmonitorAgent{
			QanPostgresqlPgstatmonitorAgent: agent,
		},
	}
	return res, nil
}

// ChangeQANPostgreSQLPgStatMonitorAgent changes disabled flag and custom labels of PostgreSQL Pg stat monitor QAN Agent.
func (s *agentsServer) changeQANPostgreSQLPgStatMonitorAgent(ctx context.Context, params *inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	agent, err := s.s.ChangeQANPostgreSQLPgStatMonitorAgent(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatmonitorAgent{
			QanPostgresqlPgstatmonitorAgent: agent,
		},
	}
	return res, nil
}

// addRDSExporter adds rds_exporter Agent.
func (s *agentsServer) addRDSExporter(ctx context.Context, params *inventoryv1.AddRDSExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddRDSExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_RdsExporter{
			RdsExporter: agent,
		},
	}
	return res, nil
}

// ChangeRDSExporter changes disabled flag and custom labels of rds_exporter Agent.
func (s *agentsServer) changeRDSExporter(ctx context.Context, params *inventoryv1.ChangeRDSExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeRDSExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_RdsExporter{
			RdsExporter: agent,
		},
	}
	return res, nil
}

// addExternalExporter adds external_exporter Agent.
func (s *agentsServer) addExternalExporter(ctx context.Context, params *inventoryv1.AddExternalExporterParams) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddExternalExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_ExternalExporter{
			ExternalExporter: agent,
		},
	}
	return res, nil
}

func (s *agentsServer) changeExternalExporter(ctx context.Context, params *inventoryv1.ChangeExternalExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeExternalExporter(params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ExternalExporter{
			ExternalExporter: agent,
		},
	}
	return res, nil
}

// addAzureDatabaseExporter adds azure_database_exporter Agent.
func (s *agentsServer) addAzureDatabaseExporter(
	ctx context.Context,
	params *inventoryv1.AddAzureDatabaseExporterParams,
) (*inventoryv1.AddAgentResponse, error) {
	agent, err := s.s.AddAzureDatabaseExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddAgentResponse{
		Exporter: &inventoryv1.AddAgentResponse_AzureDatabaseExporter{
			AzureDatabaseExporter: agent,
		},
	}
	return res, nil
}

// ChangeAzureDatabaseExporter changes disabled flag and custom labels of azure_database_exporter Agent.
func (s *agentsServer) changeAzureDatabaseExporter(
	ctx context.Context,
	params *inventoryv1.ChangeAzureDatabaseExporterParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	agent, err := s.s.ChangeAzureDatabaseExporter(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_AzureDatabaseExporter{
			AzureDatabaseExporter: agent,
		},
	}
	return res, nil
}

// RemoveAgent removes Agent.
func (s *agentsServer) RemoveAgent(ctx context.Context, req *inventoryv1.RemoveAgentRequest) (*inventoryv1.RemoveAgentResponse, error) {
	if err := s.s.Remove(ctx, req.AgentId, req.Force); err != nil {
		return nil, err
	}

	return &inventoryv1.RemoveAgentResponse{}, nil
}
