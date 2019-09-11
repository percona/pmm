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

package inventory

import (
	"context"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"
)

// AgentsService works with inventory API Agents.
type AgentsService struct {
	r  agentsRegistry
	db *reform.DB
}

// NewAgentsService creates new AgentsService
func NewAgentsService(db *reform.DB, r agentsRegistry) *AgentsService {
	return &AgentsService{
		r:  r,
		db: db,
	}
}

func toInventoryAgent(q *reform.Querier, row *models.Agent, registry agentsRegistry) (inventorypb.Agent, error) {
	agent, err := services.ToAPIAgent(q, row)
	if err != nil {
		return nil, err
	}

	if row.AgentType == models.PMMAgentType {
		agent.(*inventorypb.PMMAgent).Connected = registry.IsConnected(row.AgentID)
	}
	return agent, nil
}

// changeAgent changes common parameters for given Agent.
func (as *AgentsService) changeAgent(agentID string, common *inventorypb.ChangeCommonAgentParams) (inventorypb.Agent, error) {
	var agent inventorypb.Agent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonAgentParams{
			CustomLabels:       common.CustomLabels,
			RemoveCustomLabels: common.RemoveCustomLabels,
		}
		if common.GetEnabled() {
			params.Disabled = pointer.ToBool(false)
		}
		if common.GetDisabled() {
			params.Disabled = pointer.ToBool(true)
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

// AgentFilters represents filters for agents list.
type AgentFilters struct {
	// Return only Agents started by this pmm-agent.
	PMMAgentID string
	// Return only Agents that provide insights for that Node.
	NodeID string
	// Return only Agents that provide insights for that Service.
	ServiceID string
}

// List selects all Agents in a stable order for a given service.
//nolint:unparam
func (as *AgentsService) List(ctx context.Context, filters AgentFilters) ([]inventorypb.Agent, error) {
	var res []inventorypb.Agent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		var agents []*models.Agent
		var err error
		switch {
		case filters.PMMAgentID != "":
			agents, err = models.FindAgentsRunningByPMMAgent(tx.Querier, filters.PMMAgentID)
		case filters.NodeID != "":
			agents, err = models.FindAgentsForNode(tx.Querier, filters.NodeID)
		case filters.ServiceID != "":
			agents, err = models.FindAgentsForService(tx.Querier, filters.ServiceID)
		default:
			agents, err = models.FindAllAgents(tx.Querier)
		}
		if err != nil {
			return err
		}

		// TODO That loop makes len(agents) SELECTs, that can be slow. Optimize when needed.
		res = make([]inventorypb.Agent, len(agents))
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
//nolint:unparam
func (as *AgentsService) Get(ctx context.Context, id string) (inventorypb.Agent, error) {
	var res inventorypb.Agent
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

// AddPMMAgent inserts pmm-agent Agent with given parameters.
//nolint:unparam
func (as *AgentsService) AddPMMAgent(ctx context.Context, req *inventorypb.AddPMMAgentRequest) (*inventorypb.PMMAgent, error) {
	var res *inventorypb.PMMAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		row, err := models.CreatePMMAgent(tx.Querier, req.RunsOnNodeId, req.CustomLabels)
		if err != nil {
			return err
		}

		agent, err := toInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.PMMAgent)
		return nil
	})
	return res, e
}

// AddNodeExporter inserts node_exporter Agent with given parameters.
func (as *AgentsService) AddNodeExporter(ctx context.Context, req *inventorypb.AddNodeExporterRequest) (*inventorypb.NodeExporter, error) {
	var res *inventorypb.NodeExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		row, err := models.CreateNodeExporter(tx.Querier, req.PmmAgentId, req.CustomLabels)
		if err != nil {
			return err
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.NodeExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, nil
}

// ChangeNodeExporter updates node_exporter Agent with given parameters.
func (as *AgentsService) ChangeNodeExporter(ctx context.Context, req *inventorypb.ChangeNodeExporterRequest) (*inventorypb.NodeExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.NodeExporter)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddMySQLdExporter inserts mysqld_exporter Agent with given parameters.
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, req *inventorypb.AddMySQLdExporterRequest) (*inventorypb.MySQLdExporter, error) {
	var res *inventorypb.MySQLdExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
		}
		row, err := models.CreateAgent(tx.Querier, models.MySQLdExporterType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.MySQLdExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, nil
}

// ChangeMySQLdExporter updates mysqld_exporter Agent with given parameters.
func (as *AgentsService) ChangeMySQLdExporter(ctx context.Context, req *inventorypb.ChangeMySQLdExporterRequest) (*inventorypb.MySQLdExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.MySQLdExporter)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddMongoDBExporter inserts mongodb_exporter Agent with given parameters.
func (as *AgentsService) AddMongoDBExporter(ctx context.Context, req *inventorypb.AddMongoDBExporterRequest) (*inventorypb.MongoDBExporter, error) {
	var res *inventorypb.MongoDBExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
		}
		row, err := models.CreateAgent(tx.Querier, models.MongoDBExporterType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.MongoDBExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, nil
}

// ChangeMongoDBExporter updates mongo_exporter Agent with given parameters.
func (as *AgentsService) ChangeMongoDBExporter(ctx context.Context, req *inventorypb.ChangeMongoDBExporterRequest) (*inventorypb.MongoDBExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.MongoDBExporter)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//nolint:lll,unused
func (as *AgentsService) AddQANMySQLPerfSchemaAgent(ctx context.Context, req *inventorypb.AddQANMySQLPerfSchemaAgentRequest) (*inventorypb.QANMySQLPerfSchemaAgent, error) {
	var res *inventorypb.QANMySQLPerfSchemaAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
		}
		row, err := models.CreateAgent(tx.Querier, models.QANMySQLPerfSchemaAgentType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.QANMySQLPerfSchemaAgent)
		return nil
	})
	if e != nil {
		return res, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, e
}

// ChangeQANMySQLPerfSchemaAgent updates MySQL PerfSchema QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLPerfSchemaAgent(ctx context.Context, req *inventorypb.ChangeQANMySQLPerfSchemaAgentRequest) (*inventorypb.QANMySQLPerfSchemaAgent, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.QANMySQLPerfSchemaAgent)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMySQLSlowlogAgent adds MySQL Slowlog QAN Agent.
//nolint:lll,unused
func (as *AgentsService) AddQANMySQLSlowlogAgent(ctx context.Context, req *inventorypb.AddQANMySQLSlowlogAgentRequest) (*inventorypb.QANMySQLSlowlogAgent, error) {
	var res *inventorypb.QANMySQLSlowlogAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
		}
		row, err := models.CreateAgent(tx.Querier, models.QANMySQLSlowlogAgentType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.QANMySQLSlowlogAgent)
		return nil
	})
	if e != nil {
		return res, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, e
}

// ChangeQANMySQLSlowlogAgent updates MySQL Slowlog QAN Agent with given parameters.
func (as *AgentsService) ChangeQANMySQLSlowlogAgent(ctx context.Context, req *inventorypb.ChangeQANMySQLSlowlogAgentRequest) (*inventorypb.QANMySQLSlowlogAgent, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.QANMySQLSlowlogAgent)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddPostgresExporter inserts postgres_exporter Agent with given parameters.
func (as *AgentsService) AddPostgresExporter(ctx context.Context, req *inventorypb.AddPostgresExporterRequest) (*inventorypb.PostgresExporter, error) {
	var res *inventorypb.PostgresExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:    req.PmmAgentId,
			ServiceID:     req.ServiceId,
			Username:      req.Username,
			Password:      req.Password,
			CustomLabels:  req.CustomLabels,
			TLS:           req.Tls,
			TLSSkipVerify: req.TlsSkipVerify,
		}
		row, err := models.CreateAgent(tx.Querier, models.PostgresExporterType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.PostgresExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, nil
}

// ChangePostgresExporter updates postgres_exporter Agent with given parameters.
func (as *AgentsService) ChangePostgresExporter(ctx context.Context, req *inventorypb.ChangePostgresExporterRequest) (*inventorypb.PostgresExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.PostgresExporter)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
//nolint:lll,unused
func (as *AgentsService) AddQANMongoDBProfilerAgent(ctx context.Context, req *inventorypb.AddQANMongoDBProfilerAgentRequest) (*inventorypb.QANMongoDBProfilerAgent, error) {
	var res *inventorypb.QANMongoDBProfilerAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
		}
		row, err := models.CreateAgent(tx.Querier, models.QANMongoDBProfilerAgentType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.QANMongoDBProfilerAgent)
		return nil
	})
	if e != nil {
		return res, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, e
}

// ChangeQANMongoDBProfilerAgent updates MongoDB Profiler QAN Agent with given parameters.
//nolint:lll,dupl
func (as *AgentsService) ChangeQANMongoDBProfilerAgent(ctx context.Context, req *inventorypb.ChangeQANMongoDBProfilerAgentRequest) (*inventorypb.QANMongoDBProfilerAgent, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.QANMongoDBProfilerAgent)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddProxySQLExporter inserts proxysql_exporter Agent with given parameters.
func (as *AgentsService) AddProxySQLExporter(ctx context.Context, req *inventorypb.AddProxySQLExporterRequest) (*inventorypb.ProxySQLExporter, error) {
	var res *inventorypb.ProxySQLExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
		}
		row, err := models.CreateAgent(tx.Querier, models.ProxySQLExporterType, params)
		if err != nil {
			return err
		}
		if !req.SkipConnectionCheck {
			service, err := models.FindServiceByID(tx.Querier, req.ServiceId)
			if err != nil {
				return err
			}

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.ProxySQLExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, nil
}

// ChangeProxySQLExporter updates proxysql_exporter Agent with given parameters.
func (as *AgentsService) ChangeProxySQLExporter(ctx context.Context, req *inventorypb.ChangeProxySQLExporterRequest) (*inventorypb.ProxySQLExporter, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.ProxySQLExporter)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANPostgreSQLPgStatementsAgent adds PostgreSQL Pg stat statements QAN Agent.
//nolint:lll,unused
func (as *AgentsService) AddQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventorypb.AddQANPostgreSQLPgStatementsAgentRequest) (*inventorypb.QANPostgreSQLPgStatementsAgent, error) {
	var res *inventorypb.QANPostgreSQLPgStatementsAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.CreateAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
			// TODO TLS
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

			if err = as.r.CheckConnectionToService(ctx, service, row); err != nil {
				return err
			}
		}

		agent, err := services.ToAPIAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.QANPostgreSQLPgStatementsAgent)
		return nil
	})
	if e != nil {
		return res, e
	}

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, e
}

// ChangeQANPostgreSQLPgStatementsAgent updates PostgreSQL Pg stat statements QAN Agent with given parameters.
func (as *AgentsService) ChangeQANPostgreSQLPgStatementsAgent(ctx context.Context, req *inventorypb.ChangeQANPostgreSQLPgStatementsAgentRequest) (*inventorypb.QANPostgreSQLPgStatementsAgent, error) {
	agent, err := as.changeAgent(req.AgentId, req.Common)
	if err != nil {
		return nil, err
	}

	res := agent.(*inventorypb.QANPostgreSQLPgStatementsAgent)
	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
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
		as.r.SendSetStateRequest(ctx, pmmAgentID)
	}

	if removedAgent.AgentType == models.PMMAgentType {
		as.r.Kick(ctx, id)
	}

	return nil
}
