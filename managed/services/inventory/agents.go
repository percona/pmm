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

// changeAgent changes common parameters for given Agent.
func (as *AgentsService) changeAgent(agentID string, common *inventoryv1.ChangeCommonAgentParams) (inventoryv1.Agent, error) { //nolint:ireturn
	var agent inventoryv1.Agent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonAgentParams{
			CustomLabels:       common.CustomLabels,
			RemoveCustomLabels: common.RemoveCustomLabels,
		}

		got := 0
		if common.Enable {
			got++
			params.Disabled = pointer.ToBool(false)
		}
		if common.Disable {
			got++
			params.Disabled = pointer.ToBool(true)
		}
		if got > 1 {
			return status.Errorf(codes.InvalidArgument, "expected at most one param: enable or disable")
		}
		got = 0
		if common.EnablePushMetrics {
			got++
			params.DisablePushMetrics = pointer.ToBool(false)
		}
		if common.DisablePushMetrics {
			got++
			params.DisablePushMetrics = pointer.ToBool(true)
		}
		if got > 1 {
			return status.Errorf(codes.InvalidArgument, "expected one of  param: enable_push_metrics or disable_push_metrics")
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
//
//nolint:unparam
func (as *AgentsService) List(ctx context.Context, filters models.AgentFilters) ([]inventoryv1.Agent, error) {
	var res []inventoryv1.Agent
	e := as.db.InTransaction(func(tx *reform.TX) error {
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
func (as *AgentsService) Get(ctx context.Context, id string) (inventoryv1.Agent, error) { //nolint:ireturn,unparam
	var res inventoryv1.Agent
	e := as.db.InTransaction(func(tx *reform.TX) error {
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
//
//nolint:unparam
func (as *AgentsService) AddPMMAgent(ctx context.Context, p *inventoryv1.AddPMMAgentParams) (*inventoryv1.PMMAgent, error) {
	var res *inventoryv1.PMMAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		row, err := models.CreatePMMAgent(tx.Querier, p.RunsOnNodeId, p.CustomLabels)
		if err != nil {
			return err
		}

		agent, err := toInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.PMMAgent) //nolint:forcetypeassert
		return nil
	})
	return res, e
}

// AddNodeExporter inserts node_exporter Agent with given parameters.
func (as *AgentsService) AddNodeExporter(ctx context.Context, p *inventoryv1.AddNodeExporterParams) (*inventoryv1.NodeExporter, error) {
	var res *inventoryv1.NodeExporter
	e := as.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		row, err := models.CreateNodeExporter(tx.Querier, p.PmmAgentId, p.CustomLabels, p.PushMetrics, p.ExposeExporter,
			p.DisableCollectors, nil, services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR))
		if err != nil {
			return err
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.NodeExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, nil
}

// ChangeNodeExporter updates node_exporter Agent with given parameters.
func (as *AgentsService) ChangeNodeExporter(ctx context.Context, req *inventoryv1.ChangeNodeExporterParams) (*inventoryv1.NodeExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.NodeExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddMySQLdExporter inserts mysqld_exporter Agent with given parameters and returns it and an actual table count.
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, p *inventoryv1.AddMySQLdExporterParams) (*inventoryv1.MySQLdExporter, int32, error) { //nolint:unparam
	var row *models.Agent
	var res *inventoryv1.MySQLdExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:                     p.PmmAgentId,
			ServiceID:                      p.ServiceId,
			Username:                       p.Username,
			Password:                       p.Password,
			AgentPassword:                  p.AgentPassword,
			CustomLabels:                   p.CustomLabels,
			TLS:                            p.Tls,
			TLSSkipVerify:                  p.TlsSkipVerify,
			MySQLOptions:                   models.MySQLOptionsFromRequest(p),
			TableCountTablestatsGroupLimit: p.TablestatsGroupTableLimit,
			PushMetrics:                    p.PushMetrics,
			DisableCollectors:              p.DisableCollectors,
			LogLevel:                       services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_ERROR),
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, 0, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, pointer.GetInt32(row.TableCount), nil
}

// ChangeMySQLdExporter updates mysqld_exporter Agent with given parameters.
func (as *AgentsService) ChangeMySQLdExporter(ctx context.Context, req *inventoryv1.ChangeMySQLdExporterParams) (*inventoryv1.MySQLdExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.MySQLdExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddMongoDBExporter inserts mongodb_exporter Agent with given parameters.
func (as *AgentsService) AddMongoDBExporter(ctx context.Context, p *inventoryv1.AddMongoDBExporterParams) (*inventoryv1.MongoDBExporter, error) {
	var res *inventoryv1.MongoDBExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:        p.PmmAgentId,
			ServiceID:         p.ServiceId,
			Username:          p.Username,
			Password:          p.Password,
			AgentPassword:     p.AgentPassword,
			CustomLabels:      p.CustomLabels,
			TLS:               p.Tls,
			TLSSkipVerify:     p.TlsSkipVerify,
			MongoDBOptions:    models.MongoDBOptionsFromRequest(p),
			PushMetrics:       p.PushMetrics,
			DisableCollectors: p.DisableCollectors,
			LogLevel:          services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, nil
}

// ChangeMongoDBExporter updates mongo_exporter Agent with given parameters.
func (as *AgentsService) ChangeMongoDBExporter(ctx context.Context, req *inventoryv1.ChangeMongoDBExporterParams) (*inventoryv1.MongoDBExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.MongoDBExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//
//nolint:lll
func (as *AgentsService) AddQANMySQLPerfSchemaAgent(ctx context.Context, p *inventoryv1.AddQANMySQLPerfSchemaAgentParams) (*inventoryv1.QANMySQLPerfSchemaAgent, error) {
	var res *inventoryv1.QANMySQLPerfSchemaAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:              p.PmmAgentId,
			ServiceID:               p.ServiceId,
			Username:                p.Username,
			Password:                p.Password,
			CustomLabels:            p.CustomLabels,
			TLS:                     p.Tls,
			TLSSkipVerify:           p.TlsSkipVerify,
			MySQLOptions:            models.MySQLOptionsFromRequest(p),
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
			LogLevel:                services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return res, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, e
}

// ChangeQANMySQLPerfSchemaAgent updates MySQL PerfSchema QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLPerfSchemaAgent(ctx context.Context, req *inventoryv1.ChangeQANMySQLPerfSchemaAgentParams) (*inventoryv1.QANMySQLPerfSchemaAgent, error) { //nolint:lll
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.QANMySQLPerfSchemaAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMySQLSlowlogAgent adds MySQL Slowlog QAN Agent.
func (as *AgentsService) AddQANMySQLSlowlogAgent(ctx context.Context, p *inventoryv1.AddQANMySQLSlowlogAgentParams) (*inventoryv1.QANMySQLSlowlogAgent, error) {
	var res *inventoryv1.QANMySQLSlowlogAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		// tweak according to API docs
		maxSlowlogFileSize := p.MaxSlowlogFileSize
		if maxSlowlogFileSize < 0 {
			maxSlowlogFileSize = 0
		}

		params := &models.CreateAgentParams{
			PMMAgentID:              p.PmmAgentId,
			ServiceID:               p.ServiceId,
			Username:                p.Username,
			Password:                p.Password,
			CustomLabels:            p.CustomLabels,
			TLS:                     p.Tls,
			TLSSkipVerify:           p.TlsSkipVerify,
			MySQLOptions:            models.MySQLOptionsFromRequest(p),
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
			MaxQueryLogSize:         maxSlowlogFileSize,
			LogLevel:                services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.QANMySQLSlowlogAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return res, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, e
}

// ChangeQANMySQLSlowlogAgent updates MySQL Slowlog QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLSlowlogAgent(ctx context.Context, req *inventoryv1.ChangeQANMySQLSlowlogAgentParams) (*inventoryv1.QANMySQLSlowlogAgent, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.QANMySQLSlowlogAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddPostgresExporter inserts postgres_exporter Agent with given parameters.
func (as *AgentsService) AddPostgresExporter(ctx context.Context, p *inventoryv1.AddPostgresExporterParams) (*inventoryv1.PostgresExporter, error) {
	var res *inventoryv1.PostgresExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:        p.PmmAgentId,
			ServiceID:         p.ServiceId,
			Username:          p.Username,
			Password:          p.Password,
			AgentPassword:     p.AgentPassword,
			CustomLabels:      p.CustomLabels,
			TLS:               p.Tls,
			TLSSkipVerify:     p.TlsSkipVerify,
			PushMetrics:       p.PushMetrics,
			DisableCollectors: p.DisableCollectors,
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, nil
}

// ChangePostgresExporter updates postgres_exporter Agent with given parameters.
func (as *AgentsService) ChangePostgresExporter(ctx context.Context, req *inventoryv1.ChangePostgresExporterParams) (*inventoryv1.PostgresExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.PostgresExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
//
//nolint:lll
func (as *AgentsService) AddQANMongoDBProfilerAgent(ctx context.Context, p *inventoryv1.AddQANMongoDBProfilerAgentParams) (*inventoryv1.QANMongoDBProfilerAgent, error) {
	var res *inventoryv1.QANMongoDBProfilerAgent

	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:     p.PmmAgentId,
			ServiceID:      p.ServiceId,
			Username:       p.Username,
			Password:       p.Password,
			CustomLabels:   p.CustomLabels,
			TLS:            p.Tls,
			TLSSkipVerify:  p.TlsSkipVerify,
			MongoDBOptions: models.MongoDBOptionsFromRequest(p),
			MaxQueryLength: p.MaxQueryLength,
			LogLevel:       services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
			// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-4650
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return res, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, e
}

// ChangeQANMongoDBProfilerAgent updates MongoDB Profiler QAN Agent with given parameters.
//
//nolint:lll,dupl
func (as *AgentsService) ChangeQANMongoDBProfilerAgent(ctx context.Context, req *inventoryv1.ChangeQANMongoDBProfilerAgentParams) (*inventoryv1.QANMongoDBProfilerAgent, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.QANMongoDBProfilerAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddProxySQLExporter inserts proxysql_exporter Agent with given parameters.
func (as *AgentsService) AddProxySQLExporter(ctx context.Context, p *inventoryv1.AddProxySQLExporterParams) (*inventoryv1.ProxySQLExporter, error) {
	var res *inventoryv1.ProxySQLExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:        p.PmmAgentId,
			ServiceID:         p.ServiceId,
			Username:          p.Username,
			Password:          p.Password,
			AgentPassword:     p.AgentPassword,
			CustomLabels:      p.CustomLabels,
			TLS:               p.Tls,
			TLSSkipVerify:     p.TlsSkipVerify,
			PushMetrics:       p.PushMetrics,
			DisableCollectors: p.DisableCollectors,
			LogLevel:          services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.ProxySQLExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, nil
}

// ChangeProxySQLExporter updates proxysql_exporter Agent with given parameters.
func (as *AgentsService) ChangeProxySQLExporter(ctx context.Context, req *inventoryv1.ChangeProxySQLExporterParams) (*inventoryv1.ProxySQLExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.ProxySQLExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
//
//nolint:lll
func (as *AgentsService) AddQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventoryv1.AddQANPostgreSQLPgStatementsAgentParams) (*inventoryv1.QANPostgreSQLPgStatementsAgent, error) {
	var res *inventoryv1.QANPostgreSQLPgStatementsAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:              req.PmmAgentId,
			ServiceID:               req.ServiceId,
			Username:                req.Username,
			Password:                req.Password,
			CustomLabels:            req.CustomLabels,
			MaxQueryLength:          req.MaxQueryLength,
			CommentsParsingDisabled: req.DisableCommentsParsing,
			TLS:                     req.Tls,
			TLSSkipVerify:           req.TlsSkipVerify,
			PostgreSQLOptions:       models.PostgreSQLOptionsFromRequest(req),
			LogLevel:                services.SpecifyLogLevel(req.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.QANPostgreSQLPgStatementsAgentType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.cc.CheckConnectionToService(ctx, tx.Querier, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return res, e
	}

	as.state.RequestStateUpdate(ctx, req.PmmAgentId)
	return res, e
}

// ChangeQANPostgreSQLPgStatementsAgent updates PostgreSQL Pg stat statements QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventoryv1.ChangeQANPostgreSQLPgStatementsAgentParams) (*inventoryv1.QANPostgreSQLPgStatementsAgent, error) { //nolint:lll
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.QANPostgreSQLPgStatementsAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANPostgreSQLPgStatMonitorAgent adds PostgreSQL Pg stat monitor QAN Agent.
//
//nolint:lll
func (as *AgentsService) AddQANPostgreSQLPgStatMonitorAgent(ctx context.Context, p *inventoryv1.AddQANPostgreSQLPgStatMonitorAgentParams) (*inventoryv1.QANPostgreSQLPgStatMonitorAgent, error) {
	var res *inventoryv1.QANPostgreSQLPgStatMonitorAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:              p.PmmAgentId,
			ServiceID:               p.ServiceId,
			Username:                p.Username,
			Password:                p.Password,
			MaxQueryLength:          p.MaxQueryLength,
			QueryExamplesDisabled:   p.DisableQueryExamples,
			CommentsParsingDisabled: p.DisableCommentsParsing,
			CustomLabels:            p.CustomLabels,
			TLS:                     p.Tls,
			TLSSkipVerify:           p.TlsSkipVerify,
			PostgreSQLOptions:       models.PostgreSQLOptionsFromRequest(p),
			LogLevel:                services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return res, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, e
}

// ChangeQANPostgreSQLPgStatMonitorAgent updates PostgreSQL Pg stat monitor QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatMonitorAgent(ctx context.Context, req *inventoryv1.ChangeQANPostgreSQLPgStatMonitorAgentParams) (*inventoryv1.QANPostgreSQLPgStatMonitorAgent, error) { //nolint:lll
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.QANPostgreSQLPgStatMonitorAgent) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddRDSExporter inserts rds_exporter Agent with given parameters.
func (as *AgentsService) AddRDSExporter(ctx context.Context, p *inventoryv1.AddRDSExporterParams) (*inventoryv1.RDSExporter, error) {
	var res *inventoryv1.RDSExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:                 p.PmmAgentId,
			NodeID:                     p.NodeId,
			AWSAccessKey:               p.AwsAccessKey,
			AWSSecretKey:               p.AwsSecretKey,
			CustomLabels:               p.CustomLabels,
			RDSBasicMetricsDisabled:    p.DisableBasicMetrics,
			RDSEnhancedMetricsDisabled: p.DisableEnhancedMetrics,
			PushMetrics:                p.PushMetrics,
			LogLevel:                   services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.RDSExporterType, params)
		if err != nil {
			return err
		}

		// TODO check connection to AWS: https://jira.percona.com/browse/PMM-5024
		// if !p.SkipConnectionCheck {
		// 	...
		// }

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.RDSExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, nil
}

// ChangeRDSExporter updates rds_exporter Agent with given parameters.
func (as *AgentsService) ChangeRDSExporter(ctx context.Context, req *inventoryv1.ChangeRDSExporterParams) (*inventoryv1.RDSExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.RDSExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// AddExternalExporter inserts external-exporter Agent with given parameters.
func (as *AgentsService) AddExternalExporter(ctx context.Context, p *inventoryv1.AddExternalExporterParams) (*inventoryv1.ExternalExporter, error) {
	var (
		res        *inventoryv1.ExternalExporter
		PMMAgentID *string
	)
	e := as.db.InTransaction(func(tx *reform.TX) error {
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

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.ExternalExporter) //nolint:forcetypeassert
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

	return res, nil
}

// ChangeExternalExporter updates external-exporter Agent with given parameters.
func (as *AgentsService) ChangeExternalExporter(req *inventoryv1.ChangeExternalExporterParams) (*inventoryv1.ExternalExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	// It's required to regenerate victoriametrics config file.
	as.vmdb.RequestConfigurationUpdate()

	res := agent.(*inventoryv1.ExternalExporter) //nolint:forceTypeAssert
	return res, nil
}

// AddAzureDatabaseExporter inserts azure_exporter Agent with given parameters.
func (as *AgentsService) AddAzureDatabaseExporter(ctx context.Context, p *inventoryv1.AddAzureDatabaseExporterParams) (*inventoryv1.AzureDatabaseExporter, error) {
	var res *inventoryv1.AzureDatabaseExporter

	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   p.PmmAgentId,
			NodeID:       p.NodeId,
			AzureOptions: models.AzureOptionsFromRequest(p),
			CustomLabels: p.CustomLabels,
			PushMetrics:  p.PushMetrics,
			LogLevel:     services.SpecifyLogLevel(p.LogLevel, inventoryv1.LogLevel_LOG_LEVEL_FATAL),
		}
		row, err := models.CreateAgent(tx.Querier, models.AzureDatabaseExporterType, params)
		if err != nil {
			return err
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventoryv1.AzureDatabaseExporter) //nolint:forcetypeassert
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.state.RequestStateUpdate(ctx, p.PmmAgentId)
	return res, nil
}

// ChangeAzureDatabaseExporter updates azure_exporter Agent with given parameters.
func (as *AgentsService) ChangeAzureDatabaseExporter(
	ctx context.Context,
	req *inventoryv1.ChangeAzureDatabaseExporterParams,
) (*inventoryv1.AzureDatabaseExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventoryv1.AzureDatabaseExporter) //nolint:forcetypeassert
	as.state.RequestStateUpdate(ctx, res.PmmAgentId)
	return res, nil
}

// Remove removes Agent, and sends state update to pmm-agent, or kicks it.
func (as *AgentsService) Remove(ctx context.Context, id string, force bool) error {
	var removedAgent *models.Agent
	e := as.db.InTransaction(func(tx *reform.TX) error {
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
		logger.Get(ctx).Infof("pmm-agent with ID %q will be kicked because it was removed.", id)
		as.r.Kick(ctx, id)
	}

	return nil
}
