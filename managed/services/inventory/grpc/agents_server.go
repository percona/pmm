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
func (s *agentsServer) AddPMMAgent(ctx context.Context, req *inventoryv1.AddPMMAgentRequest) (*inventoryv1.AddPMMAgentResponse, error) {
	agent, err := s.s.AddPMMAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddPMMAgentResponse{
		PmmAgent: agent,
	}
	return res, nil
}

// AddAgent adds an Agent.
func (s *agentsServer) AddAgent(ctx context.Context, req *inventoryv1.AddAgentRequest) (*inventoryv1.AddAgentResponse, error) {
	switch req.Exporter.(type) {
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
	default:
		return nil, fmt.Errorf("invalid request %v", req.Exporter)
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
func (s *agentsServer) ChangeNodeExporter(ctx context.Context, req *inventoryv1.ChangeNodeExporterRequest) (*inventoryv1.ChangeNodeExporterResponse, error) {
	agent, err := s.s.ChangeNodeExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeNodeExporterResponse{
		NodeExporter: agent,
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
func (s *agentsServer) ChangeMySQLdExporter(ctx context.Context, req *inventoryv1.ChangeMySQLdExporterRequest) (*inventoryv1.ChangeMySQLdExporterResponse, error) {
	agent, err := s.s.ChangeMySQLdExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeMySQLdExporterResponse{
		MysqldExporter: agent,
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
func (s *agentsServer) ChangeMongoDBExporter(ctx context.Context, req *inventoryv1.ChangeMongoDBExporterRequest) (*inventoryv1.ChangeMongoDBExporterResponse, error) {
	agent, err := s.s.ChangeMongoDBExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeMongoDBExporterResponse{
		MongodbExporter: agent,
	}
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (s *agentsServer) AddQANMySQLPerfSchemaAgent(ctx context.Context, req *inventoryv1.AddQANMySQLPerfSchemaAgentRequest) (*inventoryv1.AddQANMySQLPerfSchemaAgentResponse, error) {
	agent, err := s.s.AddQANMySQLPerfSchemaAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddQANMySQLPerfSchemaAgentResponse{
		QanMysqlPerfschemaAgent: agent,
	}
	return res, nil
}

// ChangeQANMySQLPerfSchemaAgent changes disabled flag and custom labels of MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (s *agentsServer) ChangeQANMySQLPerfSchemaAgent(ctx context.Context, req *inventoryv1.ChangeQANMySQLPerfSchemaAgentRequest) (*inventoryv1.ChangeQANMySQLPerfSchemaAgentResponse, error) {
	agent, err := s.s.ChangeQANMySQLPerfSchemaAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeQANMySQLPerfSchemaAgentResponse{
		QanMysqlPerfschemaAgent: agent,
	}
	return res, nil
}

// AddQANMySQLSlowlogAgent adds MySQL Slowlog QAN Agent.
//
//nolint:lll
func (s *agentsServer) AddQANMySQLSlowlogAgent(ctx context.Context, req *inventoryv1.AddQANMySQLSlowlogAgentRequest) (*inventoryv1.AddQANMySQLSlowlogAgentResponse, error) {
	agent, err := s.s.AddQANMySQLSlowlogAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddQANMySQLSlowlogAgentResponse{
		QanMysqlSlowlogAgent: agent,
	}
	return res, nil
}

// ChangeQANMySQLSlowlogAgent changes disabled flag and custom labels of MySQL Slowlog QAN Agent.
//
//nolint:lll
func (s *agentsServer) ChangeQANMySQLSlowlogAgent(ctx context.Context, req *inventoryv1.ChangeQANMySQLSlowlogAgentRequest) (*inventoryv1.ChangeQANMySQLSlowlogAgentResponse, error) {
	agent, err := s.s.ChangeQANMySQLSlowlogAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeQANMySQLSlowlogAgentResponse{
		QanMysqlSlowlogAgent: agent,
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
func (s *agentsServer) ChangePostgresExporter(ctx context.Context, req *inventoryv1.ChangePostgresExporterRequest) (*inventoryv1.ChangePostgresExporterResponse, error) {
	agent, err := s.s.ChangePostgresExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangePostgresExporterResponse{
		PostgresExporter: agent,
	}
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
//
//nolint:lll
func (s *agentsServer) AddQANMongoDBProfilerAgent(ctx context.Context, req *inventoryv1.AddQANMongoDBProfilerAgentRequest) (*inventoryv1.AddQANMongoDBProfilerAgentResponse, error) {
	agent, err := s.s.AddQANMongoDBProfilerAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddQANMongoDBProfilerAgentResponse{
		QanMongodbProfilerAgent: agent,
	}
	return res, nil
}

// ChangeQANMongoDBProfilerAgent changes disabled flag and custom labels of MongoDB Profiler QAN Agent.
//
//nolint:lll
func (s *agentsServer) ChangeQANMongoDBProfilerAgent(ctx context.Context, req *inventoryv1.ChangeQANMongoDBProfilerAgentRequest) (*inventoryv1.ChangeQANMongoDBProfilerAgentResponse, error) {
	agent, err := s.s.ChangeQANMongoDBProfilerAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeQANMongoDBProfilerAgentResponse{
		QanMongodbProfilerAgent: agent,
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
func (s *agentsServer) ChangeProxySQLExporter(ctx context.Context, req *inventoryv1.ChangeProxySQLExporterRequest) (*inventoryv1.ChangeProxySQLExporterResponse, error) {
	agent, err := s.s.ChangeProxySQLExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeProxySQLExporterResponse{
		ProxysqlExporter: agent,
	}
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
func (s *agentsServer) AddQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventoryv1.AddQANPostgreSQLPgStatementsAgentRequest) (*inventoryv1.AddQANPostgreSQLPgStatementsAgentResponse, error) { //nolint:lll
	agent, err := s.s.AddQANPostgreSQLPgStatementsAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddQANPostgreSQLPgStatementsAgentResponse{
		QanPostgresqlPgstatementsAgent: agent,
	}
	return res, nil
}

// ChangeQANPostgreSQLPgStatementsAgent changes disabled flag and custom labels of PostgreSQL Pg stat statements QAN Agent.
func (s *agentsServer) ChangeQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventoryv1.ChangeQANPostgreSQLPgStatementsAgentRequest) (*inventoryv1.ChangeQANPostgreSQLPgStatementsAgentResponse, error) { //nolint:lll
	agent, err := s.s.ChangeQANPostgreSQLPgStatementsAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeQANPostgreSQLPgStatementsAgentResponse{
		QanPostgresqlPgstatementsAgent: agent,
	}
	return res, nil
}

// AddQANPostgreSQLPgStatMonitorAgent adds PostgreSQL Pg stat monitor QAN Agent.
func (s *agentsServer) AddQANPostgreSQLPgStatMonitorAgent(ctx context.Context, req *inventoryv1.AddQANPostgreSQLPgStatMonitorAgentRequest) (*inventoryv1.AddQANPostgreSQLPgStatMonitorAgentResponse, error) { //nolint:lll
	agent, err := s.s.AddQANPostgreSQLPgStatMonitorAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.AddQANPostgreSQLPgStatMonitorAgentResponse{
		QanPostgresqlPgstatmonitorAgent: agent,
	}
	return res, nil
}

// ChangeQANPostgreSQLPgStatMonitorAgent changes disabled flag and custom labels of PostgreSQL Pg stat monitor QAN Agent.
func (s *agentsServer) ChangeQANPostgreSQLPgStatMonitorAgent(ctx context.Context, req *inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentRequest) (*inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentResponse, error) { //nolint:lll
	agent, err := s.s.ChangeQANPostgreSQLPgStatMonitorAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentResponse{
		QanPostgresqlPgstatmonitorAgent: agent,
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
func (s *agentsServer) ChangeRDSExporter(ctx context.Context, req *inventoryv1.ChangeRDSExporterRequest) (*inventoryv1.ChangeRDSExporterResponse, error) {
	agent, err := s.s.ChangeRDSExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeRDSExporterResponse{
		RdsExporter: agent,
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

func (s *agentsServer) ChangeExternalExporter(ctx context.Context, req *inventoryv1.ChangeExternalExporterRequest) (*inventoryv1.ChangeExternalExporterResponse, error) {
	agent, err := s.s.ChangeExternalExporter(req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeExternalExporterResponse{
		ExternalExporter: agent,
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
func (s *agentsServer) ChangeAzureDatabaseExporter(
	ctx context.Context,
	req *inventoryv1.ChangeAzureDatabaseExporterRequest,
) (*inventoryv1.ChangeAzureDatabaseExporterResponse, error) {
	agent, err := s.s.ChangeAzureDatabaseExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventoryv1.ChangeAzureDatabaseExporterResponse{
		AzureDatabaseExporter: agent,
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
