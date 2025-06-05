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
	"strings"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/common"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
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
func NewAgentsService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService, cc connectionChecker, sib serviceInfoBroker, a agentService) *AgentsService { //nolint:lll
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
		agent.(*inventoryv1.PMMAgent).Connected = registry.IsConnected(row.AgentID) //nolint:forcetypeassert
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

	pmmAgentID, err := models.ExtractPmmAgentID(agent)
	if err != nil {
		return nil, 0, err
	}

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
		agent = aa.(*inventoryv1.PMMAgent) //nolint:forcetypeassert
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
		agent = aa.(*inventoryv1.NodeExporter) //nolint:forcetypeassert
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

	nodeExporter := agent.(*inventoryv1.NodeExporter) //nolint:forcetypeassert
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
	var row *models.Agent
	var agent *inventoryv1.MySQLdExporter

	mysqlOptions := models.MySQLOptionsFromRequest(p)
	mysqlOptions.TableCountTablestatsGroupLimit = p.TablestatsGroupTableLimit
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			},
			MySQLOptions: mysqlOptions,
			LogLevel:     services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
		}
		var err error
		row, err = models.CreateAgent(tx.Querier, models.MySQLdExporterType, params)
		if err != nil {
			return err
		}

		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}

			if err = as.sib.GetInfoFromService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_MysqldExporter{
			MysqldExporter: agent,
		},
	}

	return res, nil
}

// ChangeMySQLdExporter updates mysqld_exporter Agent with given parameters.
func (as *AgentsService) ChangeMySQLdExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeMySQLdExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		AgentPassword: p.AgentPassword,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	mysqldExporter := agent.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert
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
	var agent *inventoryv1.MongoDBExporter
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:     p.PmmAgentId,
			ServiceID:      p.ServiceId,
			Username:       p.Username,
			Password:       p.Password,
			AgentPassword:  p.AgentPassword,
			CustomLabels:   p.CustomLabels,
			TLS:            p.Tls,
			TLSSkipVerify:  p.TlsSkipVerify,
			MongoDBOptions: models.MongoDBOptionsFromRequest(p),
			ExporterOptions: models.ExporterOptions{
				PushMetrics:        p.PushMetrics,
				DisabledCollectors: p.DisableCollectors,
			},
			LogLevel: services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.MongoDBExporterType, params)
		if err != nil {
			return err
		}

		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}

			if err = as.sib.GetInfoFromService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_MongodbExporter{
			MongodbExporter: agent,
		},
	}

	return res, nil
}

// ChangeMongoDBExporter updates mongo_exporter Agent with given parameters.
func (as *AgentsService) ChangeMongoDBExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeMongoDBExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		AgentPassword: p.AgentPassword,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	mongodbExporter := agent.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert
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
	var agent *inventoryv1.QANMySQLPerfSchemaAgent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			MySQLOptions: models.MySQLOptionsFromRequest(p),
			LogLevel:     services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, params)
		if err != nil {
			return err
		}
		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMysqlPerfschemaAgent{
			QanMysqlPerfschemaAgent: agent,
		},
	}

	return res, e
}

// ChangeQANMySQLPerfSchemaAgent updates MySQL PerfSchema QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLPerfSchemaAgent(ctx context.Context, agentID string, p *inventoryv1.ChangeQANMySQLPerfSchemaAgentParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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

	qanAgent := agent.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
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
	var agent *inventoryv1.QANMySQLSlowlogAgent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		// tweak according to API docs
		maxSlowlogFileSize := p.MaxSlowlogFileSize
		if maxSlowlogFileSize < 0 {
			maxSlowlogFileSize = 0
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
				MaxQueryLogSize:         maxSlowlogFileSize,
			},
			MySQLOptions: models.MySQLOptionsFromRequest(p),
			LogLevel:     services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.QANMySQLSlowlogAgentType, params)
		if err != nil {
			return err
		}
		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.QANMySQLSlowlogAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMysqlSlowlogAgent{
			QanMysqlSlowlogAgent: agent,
		},
	}

	return res, e
}

// ChangeQANMySQLSlowlogAgent updates MySQL Slowlog QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLSlowlogAgent(ctx context.Context, agentID string, p *inventoryv1.ChangeQANMySQLSlowlogAgentParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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

	qanAgent := agent.(*inventoryv1.QANMySQLSlowlogAgent) //nolint:forcetypeassert
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
	var agent *inventoryv1.PostgresExporter
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			},
			PostgreSQLOptions: models.PostgreSQLOptionsFromRequest(p),
			LogLevel:          services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
		}
		row, err := models.CreateAgent(tx.Querier, models.PostgresExporterType, params)
		if err != nil {
			return err
		}

		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}

			if err = as.sib.GetInfoFromService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_PostgresExporter{
			PostgresExporter: agent,
		},
	}

	return res, nil
}

// ChangePostgresExporter updates postgres_exporter Agent with given parameters.
func (as *AgentsService) ChangePostgresExporter(ctx context.Context, agentID string, p *inventoryv1.ChangePostgresExporterParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		AgentPassword: p.AgentPassword,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	postgresExporter := agent.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, postgresExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_PostgresExporter{
			PostgresExporter: postgresExporter,
		},
	}
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
func (as *AgentsService) AddQANMongoDBProfilerAgent(ctx context.Context, p *inventoryv1.AddQANMongoDBProfilerAgentParams) (*inventoryv1.AddAgentResponse, error) {
	var agent *inventoryv1.QANMongoDBProfilerAgent

	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			MongoDBOptions: models.MongoDBOptionsFromRequest(p),
			LogLevel:       services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.QANMongoDBProfilerAgentType, params)
		if err != nil {
			return err
		}
		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: agent,
		},
	}

	return res, e
}

// ChangeQANMongoDBProfilerAgent updates MongoDB Profiler QAN Agent with given parameters.
//
//nolint:lll,dupl
func (as *AgentsService) ChangeQANMongoDBProfilerAgent(ctx context.Context, agentID string, p *inventoryv1.ChangeQANMongoDBProfilerAgentParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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

	mongodbProfilerAgent := agent.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, mongodbProfilerAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: mongodbProfilerAgent,
		},
	}
	return res, nil
}

// AddProxySQLExporter inserts proxysql_exporter Agent with given parameters.
func (as *AgentsService) AddProxySQLExporter(ctx context.Context, p *inventoryv1.AddProxySQLExporterParams) (*inventoryv1.AddAgentResponse, error) {
	var agent *inventoryv1.ProxySQLExporter
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			},
			LogLevel: services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.ProxySQLExporterType, params)
		if err != nil {
			return err
		}

		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}

			if err = as.sib.GetInfoFromService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.ProxySQLExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_ProxysqlExporter{
			ProxysqlExporter: agent,
		},
	}

	return res, nil
}

// ChangeProxySQLExporter updates proxysql_exporter Agent with given parameters.
func (as *AgentsService) ChangeProxySQLExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeProxySQLExporterParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		AgentPassword: p.AgentPassword,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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

	proxysqlExporter := agent.(*inventoryv1.ProxySQLExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, proxysqlExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ProxysqlExporter{
			ProxysqlExporter: proxysqlExporter,
		},
	}
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
func (as *AgentsService) AddQANPostgreSQLPgStatementsAgent(ctx context.Context, p *inventoryv1.AddQANPostgreSQLPgStatementsAgentParams) (*inventoryv1.AddAgentResponse, error) { //nolint:lll
	var agent *inventoryv1.QANPostgreSQLPgStatementsAgent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			PostgreSQLOptions: models.PostgreSQLOptionsFromRequest(p),
			LogLevel:          services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatementsAgentType, params)
		if err != nil {
			return err
		}
		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: agent,
		},
	}

	return res, e
}

// ChangeQANPostgreSQLPgStatementsAgent updates PostgreSQL Pg stat statements QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatementsAgent(ctx context.Context, agentID string, p *inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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

	pgStatementsAgent := agent.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, pgStatementsAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: pgStatementsAgent,
		},
	}
	return res, nil
}

// AddQANPostgreSQLPgStatMonitorAgent adds PostgreSQL Pg stat monitor QAN Agent.
func (as *AgentsService) AddQANPostgreSQLPgStatMonitorAgent(ctx context.Context, p *inventoryv1.AddQANPostgreSQLPgStatMonitorAgentParams) (*inventoryv1.AddAgentResponse, error) { //nolint:lll
	var agent *inventoryv1.QANPostgreSQLPgStatMonitorAgent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
			PostgreSQLOptions: models.PostgreSQLOptionsFromRequest(p),
			LogLevel:          services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatMonitorAgentType, params)
		if err != nil {
			return err
		}
		if !p.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, p.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_QanPostgresqlPgstatmonitorAgent{
			QanPostgresqlPgstatmonitorAgent: agent,
		},
	}

	return res, e
}

// ChangeQANPostgreSQLPgStatMonitorAgent updates PostgreSQL Pg stat monitor QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatMonitorAgent(ctx context.Context, agentID string, p *inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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

	pgStatMonitorAgent := agent.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent) //nolint:forcetypeassert
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
	var agent *inventoryv1.RDSExporter
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
		}
		row, err := models.CreateAgent(tx.Querier, models.RDSExporterType, params)
		if err != nil {
			return err
		}

		// TODO check connection to AWS: https://jira.percona.com/browse/PMM-5024
		// if !p.SkipConnectionCheck {
		// 	...
		// }

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.RDSExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_RdsExporter{
			RdsExporter: agent,
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

	rdsExporter := agent.(*inventoryv1.RDSExporter) //nolint:forcetypeassert
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
			RunsOnNodeID: p.RunsOnNodeId,
			ServiceID:    p.ServiceId,
			Username:     p.Username,
			Password:     p.Password,
			Scheme:       p.Scheme,
			MetricsPath:  p.MetricsPath,
			ListenPort:   p.ListenPort,
			CustomLabels: p.CustomLabels,
			PushMetrics:  p.PushMetrics,
		}
		row, err := models.CreateExternalExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.ExternalExporter) //nolint:forcetypeassert
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
func (as *AgentsService) ChangeExternalExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeExternalExporterParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:      p.Enable,
		Username:     p.Username,
		ListenPort:   p.ListenPort,
		CustomLabels: convertCustomLabels(p.CustomLabels),
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

	externalExporter := agent.(*inventoryv1.ExternalExporter) //nolint:forcetypeassert
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
	var agent *inventoryv1.AzureDatabaseExporter

	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   p.PmmAgentId,
			NodeID:       p.NodeId,
			CustomLabels: p.CustomLabels,
			ExporterOptions: models.ExporterOptions{
				PushMetrics: p.PushMetrics,
			},
			AzureOptions: models.AzureOptionsFromRequest(p),
			LogLevel:     services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.AzureDatabaseExporterType, params)
		if err != nil {
			return err
		}

		aa, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		agent = aa.(*inventoryv1.AzureDatabaseExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	res := &inventoryv1.AddAgentResponse{
		Agent: &inventoryv1.AddAgentResponse_AzureDatabaseExporter{
			AzureDatabaseExporter: agent,
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

	azureDatabaseExporter := agent.(*inventoryv1.AzureDatabaseExporter) //nolint:forcetypeassert
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
	}

	agent, err := as.executeAgentChange(ctx, agentID, changeParams)
	if err != nil {
		return nil, err
	}

	nomadAgent := agent.(*inventoryv1.NomadAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, nomadAgent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_NomadAgent{
			NomadAgent: nomadAgent,
		},
	}
	return res, nil
}

// ChangeValkeyExporter updates valkey_exporter Agent with given parameters.
func (as *AgentsService) ChangeValkeyExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeValkeyExporterParams) (*inventoryv1.ChangeAgentResponse, error) {
	// Convert protobuf parameters to model parameters
	params := &models.ChangeAgentParams{
		Enabled:       p.Enable,
		Username:      p.Username,
		Password:      p.Password,
		TLS:           p.Tls,
		TLSSkipVerify: p.TlsSkipVerify,
		AgentPassword: p.AgentPassword,
		CustomLabels:  convertCustomLabels(p.CustomLabels),
		LogLevel:      convertLogLevel(p.LogLevel),
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
	}

	agent, err := as.executeAgentChange(ctx, agentID, params)
	if err != nil {
		return nil, err
	}

	valkeyExporter := agent.(*inventoryv1.ValkeyExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, valkeyExporter.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ValkeyExporter{
			ValkeyExporter: valkeyExporter,
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

// Helper function to convert custom labels from protobuf to model format
func convertCustomLabels(customLabels *common.StringMap) *map[string]string {
	if customLabels != nil {
		return &customLabels.Values
	}
	return nil
}

// Helper function to convert log level from protobuf to model format
func convertLogLevel(logLevel *inventoryv1.LogLevel) *string {
	if logLevel != nil {
		// Convert from "LOG_LEVEL_DEBUG" to "debug"
		fullName := logLevel.String()
		if strings.HasPrefix(fullName, "LOG_LEVEL_") {
			simplified := strings.ToLower(strings.TrimPrefix(fullName, "LOG_LEVEL_"))
			return &simplified
		}
		return &fullName
	}
	return nil
}

// Helper function to convert metrics resolutions from protobuf to model format
func convertMetricsResolutions(mrs *common.MetricsResolutions) *models.ChangeMetricsResolutionsParams {
	if mrs == nil {
		return nil
	}

	result := &models.ChangeMetricsResolutionsParams{}
	if hr := mrs.GetHr(); hr != nil {
		result.HR = pointer.ToDuration(hr.AsDuration())
	}
	if mr := mrs.GetMr(); mr != nil {
		result.MR = pointer.ToDuration(mr.AsDuration())
	}
	if lr := mrs.GetLr(); lr != nil {
		result.LR = pointer.ToDuration(lr.AsDuration())
	}
	return result
}

// Helper function to execute agent change and build response
func (as *AgentsService) executeAgentChange(ctx context.Context, agentID string, params *models.ChangeAgentParams) (inventoryv1.Agent, error) {
	var agent inventoryv1.Agent
	err := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		row, err := models.ChangeAgent(tx.Querier, agentID, params)
		if err != nil {
			return err
		}

		agent, err = toInventoryAgent(tx.Querier, row, as.r)
		return err
	})
	return agent, err
}
