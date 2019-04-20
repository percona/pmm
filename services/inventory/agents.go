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
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// AgentsService works with inventory API Agents.
type AgentsService struct {
	r  registry
	db *reform.DB
}

// NewAgentsService creates new AgentsService
func NewAgentsService(db *reform.DB, r registry) *AgentsService {
	return &AgentsService{
		r:  r,
		db: db,
	}
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
			agents, err = models.AgentsRunningByPMMAgent(tx.Querier, filters.PMMAgentID)
		case filters.NodeID != "":
			agents, err = models.AgentsForNode(tx.Querier, filters.NodeID)
		case filters.ServiceID != "":
			agents, err = models.AgentsForService(tx.Querier, filters.ServiceID)
		default:
			agents, err = models.AgentFindAll(tx.Querier)
		}
		if err != nil {
			return err
		}

		res, err = ToInventoryAgents(agents, tx.Querier, as.r)
		if err != nil {
			return err
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
		row, err := models.AgentFindByID(tx.Querier, id)
		if err != nil {
			return err
		}

		res, err = ToInventoryAgent(tx.Querier, row, as.r)
		return err
	})
	return res, e
}

// AddPMMAgent inserts pmm-agent Agent with given parameters.
//nolint:unparam
func (as *AgentsService) AddPMMAgent(ctx context.Context, req *inventorypb.AddPMMAgentRequest) (*inventorypb.PMMAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	var res *inventorypb.PMMAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		row, err := models.AgentAddPmmAgent(tx.Querier, req.RunsOnNodeId, req.CustomLabels)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.NodeExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		row, err := models.AgentAddNodeExporter(tx.Querier, req.PmmAgentId, req.CustomLabels)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	var res *inventorypb.NodeExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonExporterParams{
			AgentID:            req.AgentId,
			CustomLabels:       req.CustomLabels,
			RemoveCustomLabels: req.RemoveCustomLabels,
		}
		if req.GetEnabled() {
			params.Disabled = false
		}
		if req.GetDisabled() {
			params.Disabled = true
		}
		row, err := models.AgentChangeExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.NodeExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddMySQLdExporter inserts mysqld_exporter Agent with given parameters.
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, req *inventorypb.AddMySQLdExporterRequest) (*inventorypb.MySQLdExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.MySQLdExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.AddExporterAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
		}
		row, err := models.AgentAddExporter(tx.Querier, models.MySQLdExporterType, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	var res *inventorypb.MySQLdExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonExporterParams{
			AgentID:            req.AgentId,
			CustomLabels:       req.CustomLabels,
			RemoveCustomLabels: req.RemoveCustomLabels,
		}
		if req.GetEnabled() {
			params.Disabled = false
		}
		if req.GetDisabled() {
			params.Disabled = true
		}
		row, err := models.AgentChangeExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.MySQLdExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

/*
// SetDisabled enables or disables Agent by ID.
func (as *AgentsService) SetDisabled(ctx context.Context, id string, disabled bool) error {
	row, _, err := as.get(ctx, id)
	if err != nil {
		return err
	}

	row.Disabled = disabled
	err = as.q.Update(row)
	return errors.WithStack(err)
}
*/

// AddMongoDBExporter inserts mongodb_exporter Agent with given parameters.
func (as *AgentsService) AddMongoDBExporter(ctx context.Context, req *inventorypb.AddMongoDBExporterRequest) (*inventorypb.MongoDBExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.MongoDBExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.AddExporterAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
		}
		row, err := models.AgentAddExporter(tx.Querier, models.MongoDBExporterType, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	var res *inventorypb.MongoDBExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonExporterParams{
			AgentID:            req.AgentId,
			CustomLabels:       req.CustomLabels,
			RemoveCustomLabels: req.RemoveCustomLabels,
		}
		if req.GetEnabled() {
			params.Disabled = false
		}
		if req.GetDisabled() {
			params.Disabled = true
		}
		row, err := models.AgentChangeExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.MongoDBExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//nolint:lll,unused
func (as *AgentsService) AddQANMySQLPerfSchemaAgent(ctx context.Context, req *inventorypb.AddQANMySQLPerfSchemaAgentRequest) (*inventorypb.QANMySQLPerfSchemaAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.QANMySQLPerfSchemaAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.AddExporterAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
		}
		row, err := models.AgentAddExporter(tx.Querier, models.QANMySQLPerfSchemaAgentType, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	var res *inventorypb.QANMySQLPerfSchemaAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonExporterParams{
			AgentID:            req.AgentId,
			CustomLabels:       req.CustomLabels,
			RemoveCustomLabels: req.RemoveCustomLabels,
		}
		if req.GetEnabled() {
			params.Disabled = false
		}
		if req.GetDisabled() {
			params.Disabled = true
		}
		row, err := models.AgentChangeExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.QANMySQLPerfSchemaAgent)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddPostgresExporter inserts postgres_exporter Agent with given parameters.
func (as *AgentsService) AddPostgresExporter(ctx context.Context, req *inventorypb.AddPostgresExporterRequest) (*inventorypb.PostgresExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.PostgresExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.AddExporterAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
		}
		row, err := models.AgentAddExporter(tx.Querier, models.PostgresExporterType, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	var res *inventorypb.PostgresExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonExporterParams{
			AgentID:            req.AgentId,
			CustomLabels:       req.CustomLabels,
			RemoveCustomLabels: req.RemoveCustomLabels,
		}
		if req.GetEnabled() {
			params.Disabled = false
		}
		if req.GetDisabled() {
			params.Disabled = true
		}
		row, err := models.AgentChangeExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.PostgresExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// AddQANMongoDBProfilerAgent adds MongoDB Profiler QAN Agent.
//nolint:lll,unused
func (as *AgentsService) AddQANMongoDBProfilerAgent(ctx context.Context, req *inventorypb.AddQANMongoDBProfilerAgentRequest) (*inventorypb.QANMongoDBProfilerAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.QANMongoDBProfilerAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.AddExporterAgentParams{
			PMMAgentID:   req.PmmAgentId,
			ServiceID:    req.ServiceId,
			Username:     req.Username,
			Password:     req.Password,
			CustomLabels: req.CustomLabels,
		}
		row, err := models.AgentAddExporter(tx.Querier, models.QANMongoDBProfilerAgentType, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
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
	var res *inventorypb.QANMongoDBProfilerAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		params := &models.ChangeCommonExporterParams{
			AgentID:            req.AgentId,
			CustomLabels:       req.CustomLabels,
			RemoveCustomLabels: req.RemoveCustomLabels,
		}
		if req.GetEnabled() {
			params.Disabled = false
		}
		if req.GetDisabled() {
			params.Disabled = true
		}
		row, err := models.AgentChangeExporter(tx.Querier, params)
		if err != nil {
			return err
		}

		agent, err := ToInventoryAgent(tx.Querier, row, as.r)
		if err != nil {
			return err
		}
		res = agent.(*inventorypb.QANMongoDBProfilerAgent)
		return nil
	})
	if e != nil {
		return nil, e
	}

	as.r.SendSetStateRequest(ctx, res.PmmAgentId)
	return res, nil
}

// Remove deletes Agent by ID.
func (as *AgentsService) Remove(ctx context.Context, id string) error {
	var removedAgent *models.Agent
	err := as.db.InTransaction(func(tx *reform.TX) error {
		var err error
		removedAgent, err = models.AgentRemove(tx.Querier, id)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if pmmAgentID := pointer.GetString(removedAgent.PMMAgentID); pmmAgentID != "" {
		as.r.SendSetStateRequest(ctx, pmmAgentID)
	}

	if removedAgent.AgentType == models.PMMAgentType {
		as.r.Kick(ctx, id)
	}

	return nil
}

// AgentConnectionChecker provides interface for check pmm agent connection status.
type AgentConnectionChecker interface {
	IsConnected(pmmAgentID string) bool
}

// ToInventoryAgent converts database row to Inventory API Agent.
func ToInventoryAgent(q *reform.Querier, row *models.Agent, connChecker AgentConnectionChecker) (inventorypb.Agent, error) {
	labels, err := row.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch row.AgentType {
	case models.PMMAgentType:
		return &inventorypb.PMMAgent{
			AgentId:      row.AgentID,
			RunsOnNodeId: pointer.GetString(row.RunsOnNodeID),
			Connected:    connChecker.IsConnected(row.AgentID),
			CustomLabels: labels,
		}, nil

	case models.NodeExporterType:
		return &inventorypb.NodeExporter{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
			Disabled:     row.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			ListenPort:   uint32(pointer.GetUint16(row.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.MySQLdExporterType:
		services, err := models.ServicesForAgent(q, row.AgentID)
		if err != nil {
			return nil, err
		}
		if len(services) != 1 {
			return nil, errors.Errorf("expected exactly one Service, got %d", len(services))
		}

		return &inventorypb.MySQLdExporter{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
			ServiceId:    services[0].ServiceID,
			Username:     pointer.GetString(row.Username),
			Password:     pointer.GetString(row.Password),
			Disabled:     row.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			ListenPort:   uint32(pointer.GetUint16(row.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.MongoDBExporterType:
		services, err := models.ServicesForAgent(q, row.AgentID)
		if err != nil {
			return nil, err
		}
		if len(services) != 1 {
			return nil, errors.Errorf("expected exactly one Service, got %d", len(services))
		}

		return &inventorypb.MongoDBExporter{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
			ServiceId:    services[0].ServiceID,
			Username:     pointer.GetString(row.Username),
			Password:     pointer.GetString(row.Password),
			Disabled:     row.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			ListenPort:   uint32(pointer.GetUint16(row.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.QANMySQLPerfSchemaAgentType:
		services, err := models.ServicesForAgent(q, row.AgentID)
		if err != nil {
			return nil, err
		}
		if len(services) != 1 {
			return nil, errors.Errorf("expected exactly one Service, got %d", len(services))
		}

		return &inventorypb.QANMySQLPerfSchemaAgent{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
			ServiceId:    services[0].ServiceID,
			Username:     pointer.GetString(row.Username),
			Password:     pointer.GetString(row.Password),
			Disabled:     row.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			CustomLabels: labels,
		}, nil

	case models.QANMongoDBProfilerAgentType:
		services, err := models.ServicesForAgent(q, row.AgentID)
		if err != nil {
			return nil, err
		}
		if len(services) != 1 {
			return nil, errors.Errorf("expected exactly one Service, got %d", len(services))
		}

		return &inventorypb.QANMongoDBProfilerAgent{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
			ServiceId:    services[0].ServiceID,
			Username:     pointer.GetString(row.Username),
			Password:     pointer.GetString(row.Password),
			Disabled:     row.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			CustomLabels: labels,
		}, nil

	case models.PostgresExporterType:
		services, err := models.ServicesForAgent(q, row.AgentID)
		if err != nil {
			return nil, err
		}
		if len(services) != 1 {
			return nil, errors.Errorf("expected exactly one Service, got %d", len(services))
		}

		return &inventorypb.PostgresExporter{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
			ServiceId:    services[0].ServiceID,
			Username:     pointer.GetString(row.Username),
			Password:     pointer.GetString(row.Password),
			Disabled:     row.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			ListenPort:   uint32(pointer.GetUint16(row.ListenPort)),
			CustomLabels: labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Agent type %s", row.AgentType))
	}
}

// ToInventoryAgents converts database rows to Inventory API Agents.
func ToInventoryAgents(agents []*models.Agent, q *reform.Querier, connChecker AgentConnectionChecker) ([]inventorypb.Agent, error) {
	// TODO That loop makes len(agents) SELECTs, that can be slow. Optimize when needed.
	res := make([]inventorypb.Agent, len(agents))
	for i, row := range agents {
		agent, err := ToInventoryAgent(q, row, connChecker)
		if err != nil {
			return res, err
		}
		res[i] = agent
	}
	return res, nil
}
