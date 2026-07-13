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

// Package inventory contains inventory API implementation.
package inventory

import (
	"context"
	"os"
	"strings"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/common"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/duration"
	"github.com/percona/pmm/managed/utils/env"
	"github.com/percona/pmm/utils/logger"
)

// AgentsService works with inventory API Agents.
type AgentsService struct {
	r     agentsRegistry
	a     agentService
	state agentsStateUpdater
	vmdb  prometheusService
	db    *reform.DB
	cc    connectionChecker
	sib   serviceInfoBroker
}

// NewAgentsService creates new AgentsService.
func NewAgentsService(
	db *reform.DB,
	r agentsRegistry,
	state agentsStateUpdater,
	vmdb prometheusService,
	cc connectionChecker,
	sib serviceInfoBroker,
	a agentService,
) *AgentsService {
	return &AgentsService{
		r:     r,
		a:     a,
		state: state,
		vmdb:  vmdb,
		db:    db,
		cc:    cc,
		sib:   sib,
	}
}

func toInventoryAgent(q *reform.Querier, row *models.Agent, registry agentsRegistry) (inventoryv1.Agent, error) { //nolint:ireturn
	agent, err := services.ToAPIAgent(q, row)
	if err != nil {
		return nil, err
	}

	if row.AgentType == models.PMMAgentType {
		pmmAgent, ok := agent.(*inventoryv1.PMMAgent)
		if !ok {
			return nil, unexpectedAgentTypeError(agent)
		}
		pmmAgent.Connected = registry.IsConnected(row.AgentID)
	}
	return agent, nil
}

// List selects all Agents in a stable order for a given service.
func (as *AgentsService) List(ctx context.Context, filters models.AgentFilters) ([]inventoryv1.Agent, error) {
	var res []inventoryv1.Agent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		got := 0
		if filters.PMMAgentID != "" {
			got++
		}
		if filters.NodeID != "" {
			got++
		}
		if filters.ServiceID != "" {
			got++
		}
		if got > 1 {
			return status.Errorf(codes.InvalidArgument, "expected at most one param: pmm_agent_id, node_id or service_id")
		}
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}
		filters.IgnoreNomad = !settings.IsNomadEnabled()

		agents, err := models.FindAgents(tx.Querier, filters)
		if err != nil {
			return err
		}

		// TODO That loop makes len(agents) SELECTs, that can be slow. Optimize when needed.
		res = make([]inventoryv1.Agent, len(agents))
		for i, a := range agents {
			res[i], err = toInventoryAgent(tx.Querier, a, as.r)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return res, e
}

// Get selects a single Agent by ID.
func (as *AgentsService) Get(ctx context.Context, id string) (inventoryv1.Agent, error) { //nolint:ireturn
	var res inventoryv1.Agent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		row, err := models.FindAgentByID(tx.Querier, id)
		if err != nil {
			return err
		}

		res, err = toInventoryAgent(tx.Querier, row, as.r)
		return err
	})
	return res, e
}

// Logs by Agent ID.
func (as *AgentsService) Logs(ctx context.Context, id string, limit uint32) ([]string, uint32, error) {
	agent, err := models.FindAgentByID(as.db.Querier, id)
	if err != nil {
		return nil, 0, err
	}

	pmmAgentID := models.ExtractPmmAgentID(agent)
	return as.a.Logs(ctx, pmmAgentID, id, limit)
}

// AddPMMAgent inserts pmm-agent Agent with given parameters.
func (as *AgentsService) AddPMMAgent(ctx context.Context, p *inventoryv1.AddPMMAgentParams) (*inventoryv1.AddAgentResponse, error) {
	var agent *inventoryv1.PMMAgent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		row, err := models.CreatePMMAgent(tx.Querier, p.RunsOnNodeId, p.CustomLabels)
		if err != nil {
			return err
		}

		aa, err := toInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		pmmAgent, ok := aa.(*inventoryv1.PMMAgent)
		if !ok {
			return unexpectedAgentTypeError(aa)
		}
		agent = pmmAgent
		return nil
	})

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_PmmAgent{
			PmmAgent: agent,
		},
	}

	return res, e
}

// AddNodeExporter inserts node_exporter Agent with given parameters.
func (as *AgentsService) AddNodeExporter(ctx context.Context, p *inventoryv1.AddNodeExporterParams) (*inventoryv1.AddAgentResponse, error) {
	var agent *inventoryv1.NodeExporter
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		row, err := models.CreateNodeExporter(tx.Querier, p.PmmAgentId, p.CustomLabels, p.PushMetrics, p.ExposeExporter,
			p.DisableCollectors, nil, services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR))
		if err != nil {
			return err
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		nodeExporter, ok := aa.(*inventoryv1.NodeExporter)
		if !ok {
			return unexpectedAgentTypeError(aa)
		}
		agent = nodeExporter
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_NodeExporter{
			NodeExporter: agent,
		},
	}

	return res, nil
}

// ChangeNodeExporter updates node_exporter Agent with given parameters.
func (as *AgentsService) ChangeNodeExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeNodeExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:      p.Enable,
		CustomLabels: convertCustomLabels(p.CustomLabels),
		LogLevel:     convertLogLevel(p.LogLevel),
		// Always skip the connection check for node_exporter
		SkipConnectionCheck: true,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		DisabledCollectors: p.DisableCollectors,
		ExposeExporter:     p.ExposeExporter,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	nodeExporter, ok := agent.(*inventoryv1.NodeExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, nodeExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_NodeExporter{
			NodeExporter: nodeExporter,
		},
	}

	return res, nil
}

// AddMySQLdExporter inserts mysqld_exporter Agent with given parameters and returns it and an actual table count.
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, p *inventoryv1.AddMySQLdExporterParams) (*inventoryv1.AddAgentResponse, error) {
	mysqlOptions, err := models.MySQLOptionsFromRequest(p)
	if err != nil {
		return nil, err
	}
	mysqlOptions.TableCountTablestatsGroupLimit = p.TablestatsGroupTableLimit

	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		AgentPassword: p.AgentPassword,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		ExporterOptions: models.ExporterOptions{
			PushMetrics:        p.PushMetrics,
			DisabledCollectors: p.DisableCollectors,
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		MySQLOptions:        mysqlOptions,
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.MySQLdExporterType, params, true)
	if err != nil {
		return nil, err
	}

	mysqldExporter, ok := agent.(*inventoryv1.MySQLdExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_MysqldExporter{
			MysqldExporter: mysqldExporter,
		},
	}

	return res, nil
}

// ChangeMySQLdExporter updates mysqld_exporter Agent with given parameters.
func (as *AgentsService) ChangeMySQLdExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeMySQLdExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		AgentPassword:       p.AgentPassword,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set MySQLOptions
	params.MySQLOptions = &models.ChangeMySQLOptions{
		TLSCa:                          p.TlsCa,
		TLSCert:                        p.TlsCert,
		TLSKey:                         p.TlsKey,
		TableCountTablestatsGroupLimit: p.TablestatsGroupTableLimit,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		DisabledCollectors: p.DisableCollectors,
		ExposeExporter:     p.ExposeExporter,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
		ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	mysqldExporter, ok := agent.(*inventoryv1.MySQLdExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, mysqldExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_MysqldExporter{
			MysqldExporter: mysqldExporter,
		},
	}

	return res, nil
}

// AddMongoDBExporter inserts mongodb_exporter Agent with given parameters.
func (as *AgentsService) AddMongoDBExporter(ctx context.Context, p *inventoryv1.AddMongoDBExporterParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:               p.PmmAgentId,
		ServiceID:                p.ServiceId,
		Username:                 p.Username,
		Password:                 p.Password,
		AgentPassword:            p.AgentPassword,
		CustomLabels:             p.CustomLabels,
		EnvironmentVariableNames: p.GetEnvironmentVariableNames(),
		TLS:                      p.Tls,
		TLSSkipVerify:            p.TlsSkipVerify,
		MongoDBOptions:           models.MongoDBOptionsFromRequest(p),
		ExporterOptions: models.ExporterOptions{
			PushMetrics:        p.PushMetrics,
			DisabledCollectors: p.DisableCollectors,
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.MongoDBExporterType, params, true)
	if err != nil {
		return nil, err
	}

	mongodbExporter, ok := agent.(*inventoryv1.MongoDBExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_MongodbExporter{
			MongodbExporter: mongodbExporter,
		},
	}

	return res, nil
}

// ChangeMongoDBExporter updates mongo_exporter Agent with given parameters.
func (as *AgentsService) ChangeMongoDBExporter(
	ctx context.Context,
	agentID string,
	p *inventoryv1.ChangeMongoDBExporterParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		AgentPassword:       p.AgentPassword,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set MongoDBOptions
	params.MongoDBOptions = &models.ChangeMongoDBOptions{
		TLSCertificateKey:             p.TlsCertificateKey,
		TLSCertificateKeyFilePassword: p.TlsCertificateKeyFilePassword,
		TLSCa:                         p.TlsCa,
		AuthenticationMechanism:       p.AuthenticationMechanism,
		AuthenticationDatabase:        p.AuthenticationDatabase,
		StatsCollections:              p.StatsCollections,
		CollectionsLimit:              p.CollectionsLimit,
		EnableAllCollectors:           p.EnableAllCollectors,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		DisabledCollectors: p.DisableCollectors,
		ExposeExporter:     p.ExposeExporter,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
		ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	mongodbExporter, ok := agent.(*inventoryv1.MongoDBExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, mongodbExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_MongodbExporter{
			MongodbExporter: mongodbExporter,
		},
	}

	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
func (as *AgentsService) AddQANMySQLPerfSchemaAgent(ctx context.Context, p *inventoryv1.AddQANMySQLPerfSchemaAgentParams) (*inventoryv1.AddAgentResponse, error) {
	mysqlOptions, err := models.MySQLOptionsFromRequest(p)
	if err != nil {
		return nil, err
	}

	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		QANOptions: models.QANOptions{
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
		},
		MySQLOptions:        mysqlOptions,
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.QANMySQLPerfSchemaAgentType, params, false)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANMySQLPerfSchemaAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMysqlPerfschemaAgent{
			QanMysqlPerfschemaAgent: qanAgent,
		},
	}

	return res, nil
}

// ChangeQANMySQLPerfSchemaAgent updates MySQL PerfSchema QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLPerfSchemaAgent(
	ctx context.Context,
	agentID string,
	p *inventoryv1.ChangeQANMySQLPerfSchemaAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set QANOptions
	params.QANOptions = &models.ChangeQANOptions{
		MaxQueryLength:          p.MaxQueryLength,
		QueryExamplesDisabled:   p.DisableQueryExamples,
		CommentsParsingDisabled: p.DisableCommentsParsing,
	}

	// Set MySQLOptions
	params.MySQLOptions = &models.ChangeMySQLOptions{
		TLSCa:   p.TlsCa,
		TLSCert: p.TlsCert,
		TLSKey:  p.TlsKey,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANMySQLPerfSchemaAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, qanAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMysqlPerfschemaAgent{
			QanMysqlPerfschemaAgent: qanAgent,
		},
	}
	return res, nil
}

// AddQANMySQLSlowlogAgent adds MySQL Slowlog QAN Agent.
func (as *AgentsService) AddQANMySQLSlowlogAgent(ctx context.Context, p *inventoryv1.AddQANMySQLSlowlogAgentParams) (*inventoryv1.AddAgentResponse, error) {
	mysqlOptions, err := models.MySQLOptionsFromRequest(p)
	if err != nil {
		return nil, err
	}

	// tweak according to API docs
	maxSlowlogFileSize := max(p.MaxSlowlogFileSize, 0)

	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		QANOptions: models.QANOptions{
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
			MaxQueryLogSize:         maxSlowlogFileSize,
		},
		MySQLOptions:        mysqlOptions,
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.QANMySQLSlowlogAgentType, params, false)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANMySQLSlowlogAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMysqlSlowlogAgent{
			QanMysqlSlowlogAgent: qanAgent,
		},
	}

	return res, nil
}

// ChangeQANMySQLSlowlogAgent updates MySQL Slowlog QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLSlowlogAgent(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeQANMySQLSlowlogAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set QANOptions
	params.QANOptions = &models.ChangeQANOptions{
		MaxQueryLength:          p.MaxQueryLength,
		QueryExamplesDisabled:   p.DisableQueryExamples,
		CommentsParsingDisabled: p.DisableCommentsParsing,
		MaxQueryLogSize:         p.MaxSlowlogFileSize,
	}

	// Set MySQLOptions
	params.MySQLOptions = &models.ChangeMySQLOptions{
		TLSCa:   p.TlsCa,
		TLSCert: p.TlsCert,
		TLSKey:  p.TlsKey,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANMySQLSlowlogAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, qanAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMysqlSlowlogAgent{
			QanMysqlSlowlogAgent: qanAgent,
		},
	}
	return res, nil
}

// AddPostgresExporter inserts postgres_exporter Agent with given parameters.
func (as *AgentsService) AddPostgresExporter(ctx context.Context, p *inventoryv1.AddPostgresExporterParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		AgentPassword: p.AgentPassword,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		ExporterOptions: models.ExporterOptions{
			PushMetrics:        p.PushMetrics,
			DisabledCollectors: p.DisableCollectors,
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		PostgreSQLOptions:   models.PostgreSQLOptionsFromRequest(p),
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.PostgresExporterType, params, true)
	if err != nil {
		return nil, err
	}

	postgresExporter, ok := agent.(*inventoryv1.PostgresExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_PostgresExporter{
			PostgresExporter: postgresExporter,
		},
	}

	return res, nil
}

// ChangePostgresExporter updates postgres_exporter Agent with given parameters.
func (as *AgentsService) ChangePostgresExporter(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangePostgresExporterParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		AgentPassword:       p.AgentPassword,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set PostgreSQLOptions
	params.PostgreSQLOptions = &models.ChangePostgreSQLOptions{
		MaxExporterConnections: p.MaxExporterConnections,
		AutoDiscoveryLimit:     p.AutoDiscoveryLimit,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		DisabledCollectors: p.DisableCollectors,
		ExposeExporter:     p.ExposeExporter,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
		ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	postgresExporter, ok := agent.(*inventoryv1.PostgresExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, postgresExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_PostgresExporter{
			PostgresExporter: postgresExporter,
		},
	}
	return res, nil
}

// AddValkeyExporter adds a valkey exporter with the given parameters.
func (as *AgentsService) AddValkeyExporter(ctx context.Context, p *inventoryv1.AddValkeyExporterParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		AgentPassword: p.AgentPassword,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		LogLevel:      services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
		ExporterOptions: models.ExporterOptions{
			PushMetrics:       p.PushMetrics,
			ExposeExporter:    p.ExposeExporter,
			ConnectionTimeout: duration.OptionalFromProto(p.ConnectionTimeout),
		},
		ValkeyOptions:       models.ValkeyOptionsFromRequest(p),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.ValkeyExporterType, params, true)
	if err != nil {
		return nil, err
	}

	valkeyExporter, ok := agent.(*inventoryv1.ValkeyExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_ValkeyExporter{
			ValkeyExporter: valkeyExporter,
		},
	}

	return res, nil
}

// ChangeValkeyExporter updates valkey_exporter Agent with given parameters.
func (as *AgentsService) ChangeValkeyExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeValkeyExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		AgentPassword:       p.AgentPassword,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set ValkeyOptions
	params.ValkeyOptions = &models.ChangeValkeyOptions{
		SSLCa:   p.TlsCa,
		SSLCert: p.TlsCert,
		SSLKey:  p.TlsKey,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		DisabledCollectors: p.DisableCollectors,
		ExposeExporter:     p.ExposeExporter,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
		ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	valkeyExporter, ok := agent.(*inventoryv1.ValkeyExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, valkeyExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ValkeyExporter{
			ValkeyExporter: valkeyExporter,
		},
	}
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
func (as *AgentsService) AddQANMongoDBProfilerAgent(ctx context.Context, p *inventoryv1.AddQANMongoDBProfilerAgentParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		QANOptions: models.QANOptions{
			MaxQueryLength: p.MaxQueryLength,
			// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-4650 - done, but not included in params.
		},
		MongoDBOptions:      models.MongoDBOptionsFromRequest(p),
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.QANMongoDBProfilerAgentType, params, false)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANMongoDBProfilerAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: qanAgent,
		},
	}

	return res, nil
}

// ChangeQANMongoDBProfilerAgent updates MongoDB Profiler QAN Agent with given parameters.
//
//nolint:dupl
func (as *AgentsService) ChangeQANMongoDBProfilerAgent(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeQANMongoDBProfilerAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set QANOptions
	params.QANOptions = &models.ChangeQANOptions{
		MaxQueryLength: p.MaxQueryLength,
	}

	// Set MongoDBOptions
	params.MongoDBOptions = &models.ChangeMongoDBOptions{
		TLSCertificateKey:             p.TlsCertificateKey,
		TLSCertificateKeyFilePassword: p.TlsCertificateKeyFilePassword,
		TLSCa:                         p.TlsCa,
		AuthenticationMechanism:       p.AuthenticationMechanism,
		AuthenticationDatabase:        p.AuthenticationDatabase,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	mongodbProfilerAgent, ok := agent.(*inventoryv1.QANMongoDBProfilerAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, mongodbProfilerAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: mongodbProfilerAgent,
		},
	}
	return res, nil
}

// AddQANMongoDBMongologAgent adds MongoDB Mongolog QAN Agent.
func (as *AgentsService) AddQANMongoDBMongologAgent(ctx context.Context, p *inventoryv1.AddQANMongoDBMongologAgentParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		QANOptions: models.QANOptions{
			MaxQueryLength: p.MaxQueryLength,
			// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-4650 - done, but not included in params.
		},
		MongoDBOptions:      models.MongoDBOptionsFromRequest(p),
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.QANMongoDBMongologAgentType, params, false)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANMongoDBMongologAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMongodbMongologAgent{
			QanMongodbMongologAgent: qanAgent,
		},
	}

	return res, nil
}

// ChangeQANMongoDBMongologAgent updates MongoDB Mongolog QAN Agent with given parameters.
//
//nolint:dupl
func (as *AgentsService) ChangeQANMongoDBMongologAgent(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeQANMongoDBMongologAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set QANOptions
	params.QANOptions = &models.ChangeQANOptions{
		MaxQueryLength: p.MaxQueryLength,
	}

	// Set MongoDBOptions
	params.MongoDBOptions = &models.ChangeMongoDBOptions{
		TLSCertificateKey:             p.TlsCertificateKey,
		TLSCertificateKeyFilePassword: p.TlsCertificateKeyFilePassword,
		TLSCa:                         p.TlsCa,
		AuthenticationMechanism:       p.AuthenticationMechanism,
		AuthenticationDatabase:        p.AuthenticationDatabase,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	mongodbMongologAgent, ok := agent.(*inventoryv1.QANMongoDBMongologAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, mongodbMongologAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMongodbMongologAgent{
			QanMongodbMongologAgent: mongodbMongologAgent,
		},
	}
	return res, nil
}

// AddProxySQLExporter inserts proxysql_exporter Agent with given parameters.
func (as *AgentsService) AddProxySQLExporter(ctx context.Context, p *inventoryv1.AddProxySQLExporterParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		AgentPassword: p.AgentPassword,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		ExporterOptions: models.ExporterOptions{
			PushMetrics:        p.PushMetrics,
			DisabledCollectors: p.DisableCollectors,
			ExposeExporter:     p.ExposeExporter,
			ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
		},
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.ProxySQLExporterType, params, true)
	if err != nil {
		return nil, err
	}

	proxysqlExporter, ok := agent.(*inventoryv1.ProxySQLExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_ProxysqlExporter{
			ProxysqlExporter: proxysqlExporter,
		},
	}

	return res, nil
}

// ChangeProxySQLExporter updates proxysql_exporter Agent with given parameters.
func (as *AgentsService) ChangeProxySQLExporter(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeProxySQLExporterParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		AgentPassword:       p.AgentPassword,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		DisabledCollectors: p.DisableCollectors,
		ExposeExporter:     p.ExposeExporter,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
		ConnectionTimeout:  duration.OptionalFromProto(p.ConnectionTimeout),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	proxysqlExporter, ok := agent.(*inventoryv1.ProxySQLExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, proxysqlExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ProxysqlExporter{
			ProxysqlExporter: proxysqlExporter,
		},
	}
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
func (as *AgentsService) AddQANPostgreSQLPgStatementsAgent(
	ctx context.Context,
	p *inventoryv1.AddQANPostgreSQLPgStatementsAgentParams,
) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		QANOptions: models.QANOptions{
			MaxQueryLength:          p.MaxQueryLength,
			CommentsParsingDisabled: p.DisableCommentsParsing,
		},
		PostgreSQLOptions:   models.PostgreSQLOptionsFromRequest(p),
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.QANPostgreSQLPgStatementsAgentType, params, false)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANPostgreSQLPgStatementsAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: qanAgent,
		},
	}

	return res, nil
}

// ChangeQANPostgreSQLPgStatementsAgent updates PostgreSQL Pg stat statements QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatementsAgent(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set QANOptions
	params.QANOptions = &models.ChangeQANOptions{
		MaxQueryLength:          p.MaxQueryLength,
		CommentsParsingDisabled: p.DisableCommentsParsing,
	}

	// Set PostgreSQLOptions
	params.PostgreSQLOptions = &models.ChangePostgreSQLOptions{
		SSLCa:   p.TlsCa,
		SSLCert: p.TlsCert,
		SSLKey:  p.TlsKey,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}
	// Check if we're trying to modify the internal PostgreSQL QAN agent and if the environment variable is set
	envVar, exists := os.LookupEnv(env.EnableInternalPgQAN)
	if exists && envVar != "" {
		a, err := models.FindAgentByID(as.db.Querier, agentID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "agent with ID %q not found", agentID)
		}
		if pointer.GetString(a.PMMAgentID) == models.PMMServerAgentID {
			return nil, status.Errorf(
				codes.FailedPrecondition,
				"QAN for PMM's internal PostgreSQL server is set to %s via an environment variable.",
				envVar,
			)
		}
	}

	pgStatementsAgent, ok := agent.(*inventoryv1.QANPostgreSQLPgStatementsAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, pgStatementsAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: pgStatementsAgent,
		},
	}
	return res, nil
}

// AddQANPostgreSQLPgStatMonitorAgent adds PostgreSQL Pg stat monitor QAN Agent.
func (as *AgentsService) AddQANPostgreSQLPgStatMonitorAgent(
	ctx context.Context,
	p *inventoryv1.AddQANPostgreSQLPgStatMonitorAgentParams,
) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		QANOptions: models.QANOptions{
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
		},
		PostgreSQLOptions:   models.PostgreSQLOptionsFromRequest(p),
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	agent, err := as.executeAgentAdd(ctx, models.QANPostgreSQLPgStatMonitorAgentType, params, false)
	if err != nil {
		return nil, err
	}

	qanAgent, ok := agent.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanPostgresqlPgstatmonitorAgent{
			QanPostgresqlPgstatmonitorAgent: qanAgent,
		},
	}

	return res, nil
}

// ChangeQANPostgreSQLPgStatMonitorAgent updates PostgreSQL Pg stat monitor QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatMonitorAgent(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		Password:            p.Password,
		TLS:                 p.Tls,
		TLSSkipVerify:       p.TlsSkipVerify,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		LogLevel:            convertLogLevel(p.LogLevel),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set QANOptions
	params.QANOptions = &models.ChangeQANOptions{
		MaxQueryLength:          p.MaxQueryLength,
		QueryExamplesDisabled:   p.DisableQueryExamples,
		CommentsParsingDisabled: p.DisableCommentsParsing,
	}

	// Set PostgreSQLOptions
	params.PostgreSQLOptions = &models.ChangePostgreSQLOptions{
		SSLCa:   p.TlsCa,
		SSLCert: p.TlsCert,
		SSLKey:  p.TlsKey,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	pgStatMonitorAgent, ok := agent.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, pgStatMonitorAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatmonitorAgent{
			QanPostgresqlPgstatmonitorAgent: pgStatMonitorAgent,
		},
	}
	return res, nil
}

// AddRDSExporter inserts rds_exporter Agent with given parameters.
func (as *AgentsService) AddRDSExporter(ctx context.Context, p *inventoryv1.AddRDSExporterParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:   p.PmmAgentId,
		NodeID:       p.NodeId,
		CustomLabels: p.CustomLabels,
		ExporterOptions: models.ExporterOptions{
			PushMetrics: p.PushMetrics,
		},
		AWSOptions: models.AWSOptions{
			AWSAccessKey:               p.AwsAccessKey,
			AWSSecretKey:               p.AwsSecretKey,
			RDSBasicMetricsDisabled:    p.DisableBasicMetrics,
			RDSEnhancedMetricsDisabled: p.DisableEnhancedMetrics,
		},
		LogLevel: services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		// Always skip the connection check for RDS (it uses a NodeID, not a ServiceID).
		// TODO check connection to AWS: https://jira.percona.com/browse/PMM-5024
		SkipConnectionCheck: true,
	}

	agent, err := as.executeAgentAdd(ctx, models.RDSExporterType, params, false)
	if err != nil {
		return nil, err
	}

	rdsExporter, ok := agent.(*inventoryv1.RDSExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_RdsExporter{
			RdsExporter: rdsExporter,
		},
	}

	return res, nil
}

// ChangeRDSExporter updates rds_exporter Agent with given parameters.
func (as *AgentsService) ChangeRDSExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeRDSExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:      p.Enable,
		CustomLabels: convertCustomLabels(p.CustomLabels),
		LogLevel:     convertLogLevel(p.LogLevel),
		// Always skip the connection check for RDS
		SkipConnectionCheck: true,
	}

	// Set AWSOptions
	params.AWSOptions = &models.ChangeAWSOptions{
		AWSAccessKey:               p.AwsAccessKey,
		AWSSecretKey:               p.AwsSecretKey,
		RDSBasicMetricsDisabled:    p.DisableBasicMetrics,
		RDSEnhancedMetricsDisabled: p.DisableEnhancedMetrics,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	rdsExporter, ok := agent.(*inventoryv1.RDSExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, rdsExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_RdsExporter{
			RdsExporter: rdsExporter,
		},
	}
	return res, nil
}

// AddExternalExporter inserts external-exporter Agent with given parameters.
func (as *AgentsService) AddExternalExporter(ctx context.Context, p *inventoryv1.AddExternalExporterParams) (*inventoryv1.AddAgentResponse, error) {
	var (
		agent      *inventoryv1.ExternalExporter
		PMMAgentID *string
	)
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		params := &models.CreateExternalExporterParams{
			RunsOnNodeID:  p.RunsOnNodeId,
			ServiceID:     p.ServiceId,
			Username:      p.Username,
			Password:      p.Password,
			Scheme:        p.Scheme,
			MetricsPath:   p.MetricsPath,
			ListenPort:    p.ListenPort,
			CustomLabels:  p.CustomLabels,
			PushMetrics:   p.PushMetrics,
			TLSSkipVerify: p.TlsSkipVerify,
		}
		row, err := models.CreateExternalExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		externalExporter, ok := aa.(*inventoryv1.ExternalExporter)
		if !ok {
			return unexpectedAgentTypeError(aa)
		}
		agent = externalExporter
		PMMAgentID = row.PMMAgentID
		return nil
	})
	if e != nil {
		return nil, e
	}

	if PMMAgentID != nil {
		as.state.RequestStateUpdate(ctx, *PMMAgentID)
	} else {
		// It's required to regenerate victoriametrics config file.
		as.vmdb.RequestConfigurationUpdate()
	}

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_ExternalExporter{
			ExternalExporter: agent,
		},
	}

	return res, nil
}

// ChangeExternalExporter updates external-exporter Agent with given parameters.
func (as *AgentsService) ChangeExternalExporter(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeExternalExporterParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:             p.Enable,
		Username:            p.Username,
		ListenPort:          p.ListenPort,
		CustomLabels:        convertCustomLabels(p.CustomLabels),
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsScheme:      p.Scheme,
		MetricsPath:        p.MetricsPath,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	// It's required to regenerate victoriametrics config file.
	as.vmdb.RequestConfigurationUpdate()

	externalExporter, ok := agent.(*inventoryv1.ExternalExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, externalExporter.RunsOnNodeId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ExternalExporter{
			ExternalExporter: externalExporter,
		},
	}
	return res, nil
}

// AddAzureDatabaseExporter inserts azure_exporter Agent with given parameters.
func (as *AgentsService) AddAzureDatabaseExporter(ctx context.Context, p *inventoryv1.AddAzureDatabaseExporterParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:   p.PmmAgentId,
		NodeID:       p.NodeId,
		CustomLabels: p.CustomLabels,
		ExporterOptions: models.ExporterOptions{
			PushMetrics: p.PushMetrics,
		},
		AzureOptions: models.AzureOptionsFromRequest(p),
		LogLevel:     services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		// Always skip the connection check for Azure
		SkipConnectionCheck: true,
	}

	agent, err := as.executeAgentAdd(ctx, models.AzureDatabaseExporterType, params, false)
	if err != nil {
		return nil, err
	}

	azureDatabaseExporter, ok := agent.(*inventoryv1.AzureDatabaseExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_AzureDatabaseExporter{
			AzureDatabaseExporter: azureDatabaseExporter,
		},
	}

	return res, nil
}

// ChangeAzureDatabaseExporter updates azure_exporter Agent with given parameters.
func (as *AgentsService) ChangeAzureDatabaseExporter(
	ctx context.Context,
	agentID string,
	p *inventoryv1.ChangeAzureDatabaseExporterParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:      p.Enable,
		CustomLabels: convertCustomLabels(p.CustomLabels),
		LogLevel:     convertLogLevel(p.LogLevel),
		// Always skip the connection check for Azure
		SkipConnectionCheck: true,
	}

	// Set AzureOptions
	params.AzureOptions = &models.ChangeAzureOptions{
		SubscriptionID: p.AzureSubscriptionId,
		ClientID:       p.AzureClientId,
		ClientSecret:   p.AzureClientSecret,
		TenantID:       p.AzureTenantId,
		ResourceGroup:  p.AzureResourceGroup,
	}

	// Set ExporterOptions
	params.ExporterOptions = &models.ChangeExporterOptions{
		PushMetrics:        p.EnablePushMetrics,
		MetricsResolutions: convertMetricsResolutions(p.MetricsResolutions),
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	azureDatabaseExporter, ok := agent.(*inventoryv1.AzureDatabaseExporter)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, azureDatabaseExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_AzureDatabaseExporter{
			AzureDatabaseExporter: azureDatabaseExporter,
		},
	}
	return res, nil
}

// ChangeNomadAgent updates Nomad Agent with given parameters.
func (as *AgentsService) ChangeNomadAgent(ctx context.Context, agentID string, params *inventoryv1.ChangeNomadAgentParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	changeParams := &models.ChangeAgentParams{
		Enabled: params.Enable,
		// Always skip the connection check for Nomad
		SkipConnectionCheck: true,
	}

	agent, err := as.executeAgentChange(ctx, agentID, changeParams)
	if err != nil {
		return nil, err
	}

	nomadAgent, ok := agent.(*inventoryv1.NomadAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, nomadAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_NomadAgent{
			NomadAgent: nomadAgent,
		},
	}
	return res, nil
}

// AddRTAMongoDBAgent adds MongoDB Real-Time Analytics Agent.
func (as *AgentsService) AddRTAMongoDBAgent(ctx context.Context, p *inventoryv1.AddRTAMongoDBAgentParams) (*inventoryv1.AddAgentResponse, error) {
	params := &models.CreateAgentParams{
		PMMAgentID:    p.PmmAgentId,
		ServiceID:     p.ServiceId,
		Username:      p.Username,
		Password:      p.Password,
		CustomLabels:  p.CustomLabels,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		MongoDBOptions: models.MongoDBOptions{
			TLSCertificateKey:             p.GetTlsCertificateKey(),
			TLSCertificateKeyFilePassword: p.GetTlsCertificateKeyFilePassword(),
			TLSCa:                         p.GetTlsCa(),
			AuthenticationMechanism:       p.GetAuthenticationMechanism(),
		},
		LogLevel:            services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		SkipConnectionCheck: p.SkipConnectionCheck,
	}

	// Set RTA options if provided
	if p.RtaOptions != nil {
		params.RTAOptions = *models.RTAOptionsFromRequest(p.RtaOptions)
	}

	agent, err := as.executeAgentAdd(ctx, models.RTAMongoDBAgentType, params, true)
	if err != nil {
		return nil, err
	}

	rtaMongoDBAgent, ok := agent.(*inventoryv1.RTAMongoDBAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(agent)
	}
	as.state.RequestStateUpdate(ctx, p.PmmAgentId)

	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_RtaMongodbAgent{
			RtaMongodbAgent: rtaMongoDBAgent,
		},
	}

	return res, nil
}

// ChangeRTAMongoDBAgent updates MongoDB Real-Time Analytics Agent with given parameters.
func (as *AgentsService) ChangeRTAMongoDBAgent(
	ctx context.Context, agentID string,
	p *inventoryv1.ChangeRTAMongoDBAgentParams,
) (*inventoryv1.ChangeAgentResponse, error) {
	changeParams := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		LogLevel:      convertLogLevel(p.LogLevel),
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		MongoDBOptions: &models.ChangeMongoDBOptions{
			TLSCertificateKey:             p.TlsCertificateKey,
			TLSCertificateKeyFilePassword: p.TlsCertificateKeyFilePassword,
			TLSCa:                         p.TlsCa,
			AuthenticationMechanism:       p.AuthenticationMechanism,
		},
		SkipConnectionCheck: p.GetSkipConnectionCheck(),
	}

	// Set RTA options if provided
	if p.RtaOptions != nil {
		changeParams.RTAOptions = models.RTAOptionsFromRequest(p.RtaOptions)
	}

	ag, err := as.executeAgentChange(ctx, agentID, changeParams)
	if err != nil {
		return nil, err
	}

	agent, ok := ag.(*inventoryv1.RTAMongoDBAgent)
	if !ok {
		return nil, unexpectedAgentTypeError(ag)
	}
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_RtaMongodbAgent{
			RtaMongodbAgent: agent,
		},
	}

	return res, nil
}

// Remove removes Agent, and sends state update to pmm-agent, or kicks it.
func (as *AgentsService) Remove(ctx context.Context, id string, force bool) error {
	var removedAgent *models.Agent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		mode := models.RemoveRestrict
		if force {
			mode = models.RemoveCascade
		}
		removedAgent, err = models.RemoveAgent(tx.Querier, id, mode)
		return err
	})
	if e != nil {
		return e
	}

	if pmmAgentID := pointer.GetString(removedAgent.PMMAgentID); pmmAgentID != "" {
		as.state.RequestStateUpdate(ctx, pmmAgentID)
	} else {
		// It's required to regenerate victoriametrics config file for the agents which aren't run by pmm-agent.
		as.vmdb.RequestConfigurationUpdate()
	}

	if removedAgent.AgentType == models.PMMAgentType {
		logger.Get(ctx).Infof("pmm-agent with ID %s will be kicked because it was removed.", id)
		as.r.Kick(ctx, id)
	}

	return nil
}

// unexpectedAgentTypeError returns error for when a type assertion on the agent fails.
func unexpectedAgentTypeError(agent inventoryv1.Agent) error {
	return status.Errorf(codes.Internal, "unexpected agent type %T", agent)
}

// Helper function to convert custom labels from protobuf to model format.
func convertCustomLabels(customLabels *common.StringMap) *map[string]string {
	if customLabels != nil {
		return &customLabels.Values
	}

	return nil
}

// Helper function to convert log level from protobuf to model format.
func convertLogLevel(logLevel *inventoryv1.LogLevel) *string {
	if logLevel != nil {
		// Convert from "LOG_LEVEL_DEBUG" to "debug"
		fullName := logLevel.String()
		if after, ok := strings.CutPrefix(fullName, "LOG_LEVEL_"); ok {
			return new(strings.ToLower(after))
		}

		return &fullName
	}

	return nil
}

// Helper function to convert metrics resolutions from protobuf to model format.
func convertMetricsResolutions(mrs *common.MetricsResolutions) *models.ChangeMetricsResolutionsParams {
	if mrs == nil {
		return nil
	}

	result := &models.ChangeMetricsResolutionsParams{}
	if hr := mrs.GetHr(); hr != nil {
		result.HR = new(hr.AsDuration())
	}

	if mr := mrs.GetMr(); mr != nil {
		result.MR = new(mr.AsDuration())
	}
	if lr := mrs.GetLr(); lr != nil {
		result.LR = new(lr.AsDuration())
	}

	return result
}

// Helper function to execute agent change and build response.
func (as *AgentsService) executeAgentChange(ctx context.Context, agentID string, params *models.ChangeAgentParams) (inventoryv1.Agent, error) { //nolint:ireturn
	var agent inventoryv1.Agent

	err := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		updatedAgent, err := models.ChangeAgent(tx.Querier, agentID, params)
		if err != nil {
			return err
		}

		if !params.SkipConnectionCheck && params.AffectsConnection() && updatedAgent.ServiceID != nil {
			service, err := models.FindServiceByID(tx.Querier, pointer.GetString(updatedAgent.ServiceID))
			if err != nil {
				return err
			}

			err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, updatedAgent)
			if err != nil {
				return err
			}
		}

		agent, err = toInventoryAgent(tx.Querier, updatedAgent, as.r)

		return err
	})

	return agent, err
}

// executeAgentAdd creates an agent and returns the agent.
func (as *AgentsService) executeAgentAdd(ctx context.Context, agentType models.AgentType, params *models.CreateAgentParams, getServiceInfo bool) (inventoryv1.Agent, error) { //nolint:ireturn,lll
	var agent inventoryv1.Agent

	err := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		row, err := models.CreateAgent(tx.Querier, agentType, params)
		if err != nil {
			return err
		}

		if !params.SkipConnectionCheck && params.ServiceID != "" {
			service, err := models.FindServiceByID(tx.Querier, params.ServiceID)
			if err != nil {
				return err
			}

			err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row)
			if err != nil {
				return err
			}

			if getServiceInfo {
				err = as.sib.GetInfoFromService(ctx, tx.Querier, service, row)
				if err != nil {
					return err
				}
			}
		}

		agent, err = services.ToAPIAgent(tx.Querier, row)

		return err
	})

	return agent, err
}
