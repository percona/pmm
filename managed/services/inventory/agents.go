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

type commonAgentParams struct {
	Enable             *bool
	EnablePushMetrics  *bool
	CustomLabels       *common.StringMap
	MetricsResolutions *common.MetricsResolutions
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

// changeAgent changes common parameters for given Agent.
func (as *AgentsService) changeAgent(ctx context.Context, agentID string, common *commonAgentParams) (inventoryv1.Agent, error) { //nolint:ireturn
	var agent inventoryv1.Agent
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		params := &models.ChangeCommonAgentParams{
			Enabled:           common.Enable,
			EnablePushMetrics: common.EnablePushMetrics,
		}
		if common.CustomLabels != nil {
			params.CustomLabels = &common.CustomLabels.Values
		}

		if mrs := common.MetricsResolutions; mrs != nil {
			if hr := mrs.GetHr(); hr != nil {
				params.MetricsResolutions.HR = pointer.ToDuration(hr.AsDuration())
			}

			if mr := mrs.GetMr(); mr != nil {
				params.MetricsResolutions.MR = pointer.ToDuration(mr.AsDuration())
			}

			if lr := mrs.GetLr(); lr != nil {
				params.MetricsResolutions.LR = pointer.ToDuration(lr.AsDuration())
			}
		}

		row, err := models.ChangeAgent(tx.Querier, agentID, params)
		if err != nil {
			return err
		}
		agent, err = toInventoryAgent(tx.Querier, row, as.r)
		return err
	})
	return agent, e
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.NodeExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_NodeExporter{
			NodeExporter: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_MysqldExporter{
			MysqldExporter: agent,
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
func (as *AgentsService) ChangeMongoDBExporter(ctx context.Context, agentID string, p *inventoryv1.ChangeMongoDBExporterParams) (*inventoryv1.ChangeAgentResponse, error) { //nolint:lll
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_MongodbExporter{
			MongodbExporter: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMysqlPerfschemaAgent{
			QanMysqlPerfschemaAgent: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.QANMySQLSlowlogAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMysqlSlowlogAgent{
			QanMysqlSlowlogAgent: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_PostgresExporter{
			PostgresExporter: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanMongodbProfilerAgent{
			QanMongodbProfilerAgent: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.ProxySQLExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ProxysqlExporter{
			ProxysqlExporter: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatementsAgent{
			QanPostgresqlPgstatementsAgent: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_QanPostgresqlPgstatmonitorAgent{
			QanPostgresqlPgstatmonitorAgent: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.RDSExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_RdsExporter{
			RdsExporter: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	// It's required to regenerate victoriametrics config file.
	as.vmdb.RequestConfigurationUpdate()

	agent := ag.(*inventoryv1.ExternalExporter) //nolint:forceTypeAssert

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_ExternalExporter{
			ExternalExporter: agent,
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
	common := &commonAgentParams{
		Enable:             p.Enable,
		EnablePushMetrics:  p.EnablePushMetrics,
		CustomLabels:       p.CustomLabels,
		MetricsResolutions: p.MetricsResolutions,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}

	agent := ag.(*inventoryv1.AzureDatabaseExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)

	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_AzureDatabaseExporter{
			AzureDatabaseExporter: agent,
		},
	}
	return res, nil
}

// ChangeNomadAgent updates Nomad Agent with given parameters.
func (as *AgentsService) ChangeNomadAgent(ctx context.Context, agentID string, params *inventoryv1.ChangeNomadAgentParams) (*inventoryv1.ChangeAgentResponse, error) {
	common := &commonAgentParams{
		Enable: params.Enable,
	}
	ag, err := as.changeAgent(ctx, agentID, common)
	if err != nil {
		return nil, err
	}
	agent := ag.(*inventoryv1.NomadAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, agent.PmmAgentId)
	res := &inventoryv1.ChangeAgentResponse{
		Agent: &inventoryv1.ChangeAgentResponse_NomadAgent{
			NomadAgent: agent,
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
