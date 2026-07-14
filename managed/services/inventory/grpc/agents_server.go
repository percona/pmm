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

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/inventory"
)

type agentsServer struct {
	s *inventory.AgentsService

	inventorypb.UnimplementedAgentsServer
}

// NewAgentsServer returns Inventory API handler for managing Agents.
func NewAgentsServer(s *inventory.AgentsService) inventorypb.AgentsServer { //nolint:ireturn
	return &agentsServer{s: s}
}

var agentTypes = map[inventorypb.AgentType]models.AgentType{
	inventorypb.AgentType_PMM_AGENT:                          models.PMMAgentType,
	inventorypb.AgentType_NODE_EXPORTER:                      models.NodeExporterType,
	inventorypb.AgentType_MYSQLD_EXPORTER:                    models.MySQLdExporterType,
	inventorypb.AgentType_MONGODB_EXPORTER:                   models.MongoDBExporterType,
	inventorypb.AgentType_POSTGRES_EXPORTER:                  models.PostgresExporterType,
	inventorypb.AgentType_PROXYSQL_EXPORTER:                  models.ProxySQLExporterType,
	inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT:         models.QANMySQLPerfSchemaAgentType,
	inventorypb.AgentType_QAN_MYSQL_SLOWLOG_AGENT:            models.QANMySQLSlowlogAgentType,
	inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT:         models.QANMongoDBProfilerAgentType,
	inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT:  models.QANPostgreSQLPgStatementsAgentType,
	inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT: models.QANPostgreSQLPgStatMonitorAgentType,
	inventorypb.AgentType_RDS_EXPORTER:                       models.RDSExporterType,
	inventorypb.AgentType_EXTERNAL_EXPORTER:                  models.ExternalExporterType,
	inventorypb.AgentType_AZURE_DATABASE_EXPORTER:            models.AzureDatabaseExporterType,
	inventorypb.AgentType_VM_AGENT:                           models.VMAgentType,
}

func agentType(req *inventorypb.ListAgentsRequest) *models.AgentType {
	if req.AgentType == inventorypb.AgentType_AGENT_TYPE_INVALID {
		return nil
	}
	agentType := agentTypes[req.AgentType]
	return &agentType
}

// ListAgents returns a list of Agents for a given filters.
func (s *agentsServer) ListAgents(ctx context.Context, req *inventorypb.ListAgentsRequest) (*inventorypb.ListAgentsResponse, error) {
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

	res := &inventorypb.ListAgentsResponse{}
	for _, agent := range agents {
		switch agent := agent.(type) {
		case *inventorypb.PMMAgent:
			res.PmmAgent = append(res.PmmAgent, agent)
		case *inventorypb.NodeExporter:
			res.NodeExporter = append(res.NodeExporter, agent)
		case *inventorypb.MySQLdExporter:
			res.MysqldExporter = append(res.MysqldExporter, agent)
		case *inventorypb.MongoDBExporter:
			res.MongodbExporter = append(res.MongodbExporter, agent)
		case *inventorypb.QANMySQLPerfSchemaAgent:
			res.QanMysqlPerfschemaAgent = append(res.QanMysqlPerfschemaAgent, agent)
		case *inventorypb.QANMySQLSlowlogAgent:
			res.QanMysqlSlowlogAgent = append(res.QanMysqlSlowlogAgent, agent)
		case *inventorypb.PostgresExporter:
			res.PostgresExporter = append(res.PostgresExporter, agent)
		case *inventorypb.QANMongoDBProfilerAgent:
			res.QanMongodbProfilerAgent = append(res.QanMongodbProfilerAgent, agent)
		case *inventorypb.ProxySQLExporter:
			res.ProxysqlExporter = append(res.ProxysqlExporter, agent)
		case *inventorypb.QANPostgreSQLPgStatementsAgent:
			res.QanPostgresqlPgstatementsAgent = append(res.QanPostgresqlPgstatementsAgent, agent)
		case *inventorypb.QANPostgreSQLPgStatMonitorAgent:
			res.QanPostgresqlPgstatmonitorAgent = append(res.QanPostgresqlPgstatmonitorAgent, agent)
		case *inventorypb.RDSExporter:
			res.RdsExporter = append(res.RdsExporter, agent)
		case *inventorypb.ExternalExporter:
			res.ExternalExporter = append(res.ExternalExporter, agent)
		case *inventorypb.AzureDatabaseExporter:
			res.AzureDatabaseExporter = append(res.AzureDatabaseExporter, agent)
		case *inventorypb.VMAgent:
			res.VmAgent = append(res.VmAgent, agent)
		default:
			panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
		}
	}
	return res, nil
}

// GetAgent returns a single Agent by ID.
func (s *agentsServer) GetAgent(ctx context.Context, req *inventorypb.GetAgentRequest) (*inventorypb.GetAgentResponse, error) {
	agent, err := s.s.Get(ctx, req.AgentId)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.GetAgentResponse{}
	switch agent := agent.(type) {
	case *inventorypb.PMMAgent:
		res.Agent = &inventorypb.GetAgentResponse_PmmAgent{PmmAgent: agent}
	case *inventorypb.NodeExporter:
		res.Agent = &inventorypb.GetAgentResponse_NodeExporter{NodeExporter: agent}
	case *inventorypb.MySQLdExporter:
		res.Agent = &inventorypb.GetAgentResponse_MysqldExporter{MysqldExporter: agent}
	case *inventorypb.MongoDBExporter:
		res.Agent = &inventorypb.GetAgentResponse_MongodbExporter{MongodbExporter: agent}
	case *inventorypb.QANMySQLPerfSchemaAgent:
		res.Agent = &inventorypb.GetAgentResponse_QanMysqlPerfschemaAgent{QanMysqlPerfschemaAgent: agent}
	case *inventorypb.QANMySQLSlowlogAgent:
		res.Agent = &inventorypb.GetAgentResponse_QanMysqlSlowlogAgent{QanMysqlSlowlogAgent: agent}
	case *inventorypb.PostgresExporter:
		res.Agent = &inventorypb.GetAgentResponse_PostgresExporter{PostgresExporter: agent}
	case *inventorypb.QANMongoDBProfilerAgent:
		res.Agent = &inventorypb.GetAgentResponse_QanMongodbProfilerAgent{QanMongodbProfilerAgent: agent}
	case *inventorypb.ProxySQLExporter:
		res.Agent = &inventorypb.GetAgentResponse_ProxysqlExporter{ProxysqlExporter: agent}
	case *inventorypb.QANPostgreSQLPgStatementsAgent:
		res.Agent = &inventorypb.GetAgentResponse_QanPostgresqlPgstatementsAgent{QanPostgresqlPgstatementsAgent: agent}
	case *inventorypb.QANPostgreSQLPgStatMonitorAgent:
		res.Agent = &inventorypb.GetAgentResponse_QanPostgresqlPgstatmonitorAgent{QanPostgresqlPgstatmonitorAgent: agent}
	case *inventorypb.RDSExporter:
		res.Agent = &inventorypb.GetAgentResponse_RdsExporter{RdsExporter: agent}
	case *inventorypb.ExternalExporter:
		res.Agent = &inventorypb.GetAgentResponse_ExternalExporter{ExternalExporter: agent}
	case *inventorypb.AzureDatabaseExporter:
		res.Agent = &inventorypb.GetAgentResponse_AzureDatabaseExporter{AzureDatabaseExporter: agent}
	case *inventorypb.VMAgent:
		// skip it, fix later if needed.
	default:
		panic(fmt.Errorf("unhandled inventory Agent type %T", agent))
	}
	return res, nil
}

// GetAgentLogs returns Agent logs by ID.
func (s *agentsServer) GetAgentLogs(ctx context.Context, req *inventorypb.GetAgentLogsRequest) (*inventorypb.GetAgentLogsResponse, error) {
	logs, agentConfigLogLinesCount, err := s.s.Logs(ctx, req.AgentId, req.Limit)
	if err != nil {
		return nil, err
	}

	return &inventorypb.GetAgentLogsResponse{
		Logs:                     logs,
		AgentConfigLogLinesCount: agentConfigLogLinesCount,
	}, nil
}

// AddPMMAgent adds pmm-agent Agent.
func (s *agentsServer) AddPMMAgent(ctx context.Context, req *inventorypb.AddPMMAgentRequest) (*inventorypb.AddPMMAgentResponse, error) {
	agent, err := s.s.AddPMMAgent(ctx, req)
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
	agent, err := s.s.AddNodeExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddNodeExporterResponse{
		NodeExporter: agent,
	}
	return res, nil
}

// ChangeNodeExporter changes disabled flag and custom labels of node_exporter Agent.
func (s *agentsServer) ChangeNodeExporter(ctx context.Context, req *inventorypb.ChangeNodeExporterRequest) (*inventorypb.ChangeNodeExporterResponse, error) {
	agent, err := s.s.ChangeNodeExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeNodeExporterResponse{
		NodeExporter: agent,
	}
	return res, nil
}

// AddMySQLdExporter adds mysqld_exporter Agent.
func (s *agentsServer) AddMySQLdExporter(ctx context.Context, req *inventorypb.AddMySQLdExporterRequest) (*inventorypb.AddMySQLdExporterResponse, error) {
	agent, tableCount, err := s.s.AddMySQLdExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMySQLdExporterResponse{
		MysqldExporter: agent,
		TableCount:     tableCount,
	}
	return res, nil
}

// ChangeMySQLdExporter changes disabled flag and custom labels of mysqld_exporter Agent.
func (s *agentsServer) ChangeMySQLdExporter(ctx context.Context, req *inventorypb.ChangeMySQLdExporterRequest) (*inventorypb.ChangeMySQLdExporterResponse, error) {
	agent, err := s.s.ChangeMySQLdExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeMySQLdExporterResponse{
		MysqldExporter: agent,
	}
	return res, nil
}

// AddMongoDBExporter adds mongodb_exporter Agent.
func (s *agentsServer) AddMongoDBExporter(ctx context.Context, req *inventorypb.AddMongoDBExporterRequest) (*inventorypb.AddMongoDBExporterResponse, error) {
	agent, err := s.s.AddMongoDBExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddMongoDBExporterResponse{
		MongodbExporter: agent,
	}
	return res, nil
}

// ChangeMongoDBExporter changes disabled flag and custom labels of mongo_exporter Agent.
func (s *agentsServer) ChangeMongoDBExporter(ctx context.Context, req *inventorypb.ChangeMongoDBExporterRequest) (*inventorypb.ChangeMongoDBExporterResponse, error) {
	agent, err := s.s.ChangeMongoDBExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeMongoDBExporterResponse{
		MongodbExporter: agent,
	}
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (s *agentsServer) AddQANMySQLPerfSchemaAgent(ctx context.Context, req *inventorypb.AddQANMySQLPerfSchemaAgentRequest) (*inventorypb.AddQANMySQLPerfSchemaAgentResponse, error) {
	agent, err := s.s.AddQANMySQLPerfSchemaAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddQANMySQLPerfSchemaAgentResponse{
		QanMysqlPerfschemaAgent: agent,
	}
	return res, nil
}

// ChangeQANMySQLPerfSchemaAgent changes disabled flag and custom labels of MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (s *agentsServer) ChangeQANMySQLPerfSchemaAgent(ctx context.Context, req *inventorypb.ChangeQANMySQLPerfSchemaAgentRequest) (*inventorypb.ChangeQANMySQLPerfSchemaAgentResponse, error) {
	agent, err := s.s.ChangeQANMySQLPerfSchemaAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeQANMySQLPerfSchemaAgentResponse{
		QanMysqlPerfschemaAgent: agent,
	}
	return res, nil
}

// AddQANMySQLSlowlogAgent adds MySQL Slowlog QAN Agent.
//
//nolint:lll
func (s *agentsServer) AddQANMySQLSlowlogAgent(ctx context.Context, req *inventorypb.AddQANMySQLSlowlogAgentRequest) (*inventorypb.AddQANMySQLSlowlogAgentResponse, error) {
	agent, err := s.s.AddQANMySQLSlowlogAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddQANMySQLSlowlogAgentResponse{
		QanMysqlSlowlogAgent: agent,
	}
	return res, nil
}

// ChangeQANMySQLSlowlogAgent changes disabled flag and custom labels of MySQL Slowlog QAN Agent.
//
//nolint:lll
func (s *agentsServer) ChangeQANMySQLSlowlogAgent(ctx context.Context, req *inventorypb.ChangeQANMySQLSlowlogAgentRequest) (*inventorypb.ChangeQANMySQLSlowlogAgentResponse, error) {
	agent, err := s.s.ChangeQANMySQLSlowlogAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeQANMySQLSlowlogAgentResponse{
		QanMysqlSlowlogAgent: agent,
	}
	return res, nil
}

// AddPostgresExporter adds postgres_exporter Agent.
func (s *agentsServer) AddPostgresExporter(ctx context.Context, req *inventorypb.AddPostgresExporterRequest) (*inventorypb.AddPostgresExporterResponse, error) {
	agent, err := s.s.AddPostgresExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddPostgresExporterResponse{
		PostgresExporter: agent,
	}
	return res, nil
}

// ChangePostgresExporter changes disabled flag and custom labels of postgres_exporter Agent.
func (s *agentsServer) ChangePostgresExporter(ctx context.Context, req *inventorypb.ChangePostgresExporterRequest) (*inventorypb.ChangePostgresExporterResponse, error) {
	agent, err := s.s.ChangePostgresExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangePostgresExporterResponse{
		PostgresExporter: agent,
	}
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
//
//nolint:lll
func (s *agentsServer) AddQANMongoDBProfilerAgent(ctx context.Context, req *inventorypb.AddQANMongoDBProfilerAgentRequest) (*inventorypb.AddQANMongoDBProfilerAgentResponse, error) {
	agent, err := s.s.AddQANMongoDBProfilerAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddQANMongoDBProfilerAgentResponse{
		QanMongodbProfilerAgent: agent,
	}
	return res, nil
}

// ChangeQANMongoDBProfilerAgent changes disabled flag and custom labels of MongoDB Profiler QAN Agent.
//
//nolint:lll
func (s *agentsServer) ChangeQANMongoDBProfilerAgent(ctx context.Context, req *inventorypb.ChangeQANMongoDBProfilerAgentRequest) (*inventorypb.ChangeQANMongoDBProfilerAgentResponse, error) {
	agent, err := s.s.ChangeQANMongoDBProfilerAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeQANMongoDBProfilerAgentResponse{
		QanMongodbProfilerAgent: agent,
	}
	return res, nil
}

// AddProxySQLExporter adds proxysql_exporter Agent.
func (s *agentsServer) AddProxySQLExporter(ctx context.Context, req *inventorypb.AddProxySQLExporterRequest) (*inventorypb.AddProxySQLExporterResponse, error) {
	agent, err := s.s.AddProxySQLExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddProxySQLExporterResponse{
		ProxysqlExporter: agent,
	}
	return res, nil
}

// ChangeProxySQLExporter changes disabled flag and custom labels of proxysql_exporter Agent.
func (s *agentsServer) ChangeProxySQLExporter(ctx context.Context, req *inventorypb.ChangeProxySQLExporterRequest) (*inventorypb.ChangeProxySQLExporterResponse, error) {
	agent, err := s.s.ChangeProxySQLExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeProxySQLExporterResponse{
		ProxysqlExporter: agent,
	}
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
func (s *agentsServer) AddQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventorypb.AddQANPostgreSQLPgStatementsAgentRequest) (*inventorypb.AddQANPostgreSQLPgStatementsAgentResponse, error) { //nolint:lll
	agent, err := s.s.AddQANPostgreSQLPgStatementsAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddQANPostgreSQLPgStatementsAgentResponse{
		QanPostgresqlPgstatementsAgent: agent,
	}
	return res, nil
}

// ChangeQANPostgreSQLPgStatementsAgent changes disabled flag and custom labels of PostgreSQL Pg stat statements QAN Agent.
func (s *agentsServer) ChangeQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventorypb.ChangeQANPostgreSQLPgStatementsAgentRequest) (*inventorypb.ChangeQANPostgreSQLPgStatementsAgentResponse, error) { //nolint:lll
	agent, err := s.s.ChangeQANPostgreSQLPgStatementsAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeQANPostgreSQLPgStatementsAgentResponse{
		QanPostgresqlPgstatementsAgent: agent,
	}
	return res, nil
}

// AddQANPostgreSQLPgStatMonitorAgent adds PostgreSQL Pg stat monitor QAN Agent.
func (s *agentsServer) AddQANPostgreSQLPgStatMonitorAgent(ctx context.Context, req *inventorypb.AddQANPostgreSQLPgStatMonitorAgentRequest) (*inventorypb.AddQANPostgreSQLPgStatMonitorAgentResponse, error) { //nolint:lll
	agent, err := s.s.AddQANPostgreSQLPgStatMonitorAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddQANPostgreSQLPgStatMonitorAgentResponse{
		QanPostgresqlPgstatmonitorAgent: agent,
	}
	return res, nil
}

// ChangeQANPostgreSQLPgStatMonitorAgent changes disabled flag and custom labels of PostgreSQL Pg stat monitor QAN Agent.
func (s *agentsServer) ChangeQANPostgreSQLPgStatMonitorAgent(ctx context.Context, req *inventorypb.ChangeQANPostgreSQLPgStatMonitorAgentRequest) (*inventorypb.ChangeQANPostgreSQLPgStatMonitorAgentResponse, error) { //nolint:lll
	agent, err := s.s.ChangeQANPostgreSQLPgStatMonitorAgent(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeQANPostgreSQLPgStatMonitorAgentResponse{
		QanPostgresqlPgstatmonitorAgent: agent,
	}
	return res, nil
}

// AddRDSExporter adds rds_exporter Agent.
func (s *agentsServer) AddRDSExporter(ctx context.Context, req *inventorypb.AddRDSExporterRequest) (*inventorypb.AddRDSExporterResponse, error) {
	agent, err := s.s.AddRDSExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddRDSExporterResponse{
		RdsExporter: agent,
	}
	return res, nil
}

// ChangeRDSExporter changes disabled flag and custom labels of rds_exporter Agent.
func (s *agentsServer) ChangeRDSExporter(ctx context.Context, req *inventorypb.ChangeRDSExporterRequest) (*inventorypb.ChangeRDSExporterResponse, error) {
	agent, err := s.s.ChangeRDSExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeRDSExporterResponse{
		RdsExporter: agent,
	}
	return res, nil
}

func (s *agentsServer) AddExternalExporter(ctx context.Context, req *inventorypb.AddExternalExporterRequest) (*inventorypb.AddExternalExporterResponse, error) {
	agent, err := s.s.AddExternalExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddExternalExporterResponse{
		ExternalExporter: agent,
	}
	return res, nil
}

func (s *agentsServer) ChangeExternalExporter(_ context.Context, req *inventorypb.ChangeExternalExporterRequest) (*inventorypb.ChangeExternalExporterResponse, error) {
	agent, err := s.s.ChangeExternalExporter(req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeExternalExporterResponse{
		ExternalExporter: agent,
	}
	return res, nil
}

// AddAzureDatabaseExporter adds azure_database_exporter Agent.
func (s *agentsServer) AddAzureDatabaseExporter(
	ctx context.Context,
	req *inventorypb.AddAzureDatabaseExporterRequest,
) (*inventorypb.AddAzureDatabaseExporterResponse, error) {
	agent, err := s.s.AddAzureDatabaseExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.AddAzureDatabaseExporterResponse{
		AzureDatabaseExporter: agent,
	}
	return res, nil
}

// ChangeAzureDatabaseExporter changes disabled flag and custom labels of azure_database_exporter Agent.
func (s *agentsServer) ChangeAzureDatabaseExporter(
	ctx context.Context,
	req *inventorypb.ChangeAzureDatabaseExporterRequest,
) (*inventorypb.ChangeAzureDatabaseExporterResponse, error) {
	agent, err := s.s.ChangeAzureDatabaseExporter(ctx, req)
	if err != nil {
		return nil, err
	}

	res := &inventorypb.ChangeAzureDatabaseExporterResponse{
		AzureDatabaseExporter: agent,
	}
	return res, nil
}

// RemoveAgent removes Agent.
func (s *agentsServer) RemoveAgent(ctx context.Context, req *inventorypb.RemoveAgentRequest) (*inventorypb.RemoveAgentResponse, error) {
	if err := s.s.Remove(ctx, req.AgentId, req.Force); err != nil {
		return nil, err
	}

	return &inventorypb.RemoveAgentResponse{}, nil
}
