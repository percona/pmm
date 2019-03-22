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
	"github.com/google/uuid"
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// makeAgent converts database row to Inventory API Agent.
func (as *AgentsService) makeAgent(q *reform.Querier, row *models.Agent) (inventorypb.Agent, error) {
	labels, err := row.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch row.AgentType {
	case models.PMMAgentType:
		return &inventorypb.PMMAgent{
			AgentId:      row.AgentID,
			RunsOnNodeId: pointer.GetString(row.RunsOnNodeID),
			Connected:    as.r.IsConnected(row.AgentID),
			CustomLabels: labels,
		}, nil

	case models.NodeExporterType:
		return &inventorypb.NodeExporter{
			AgentId:      row.AgentID,
			PmmAgentId:   pointer.GetString(row.PMMAgentID),
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
			AgentId:    row.AgentID,
			PmmAgentId: pointer.GetString(row.PMMAgentID),
			ServiceId:  services[0].ServiceID,
			Username:   pointer.GetString(row.Username),
			Password:   pointer.GetString(row.Password),
			Status:     inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			ListenPort: uint32(pointer.GetUint16(row.ListenPort)),
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
			AgentId:    row.AgentID,
			PmmAgentId: pointer.GetString(row.PMMAgentID),
			ServiceId:  services[0].ServiceID,
			Username:   pointer.GetString(row.Username),
			Password:   pointer.GetString(row.Password),
			Status:     inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
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
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[row.Status]),
			ListenPort:   uint32(pointer.GetUint16(row.ListenPort)),
			CustomLabels: labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Agent type %s", row.AgentType))
	}
}

func get(q *reform.Querier, id string) (*models.Agent, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Agent ID.")
	}

	row := &models.Agent{AgentID: id}
	switch err := q.Reload(row); err {
	case nil:
		return row, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Agent with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

func checkUniqueID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Agent ID")
	}

	row := &models.Agent{AgentID: id}
	switch err := q.Reload(row); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Agent with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
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
			var structs []reform.Struct
			structs, err = tx.Querier.SelectAllFrom(models.AgentTable, "ORDER BY agent_id")
			err = errors.Wrap(err, "failed to select Agents")
			agents = make([]*models.Agent, len(structs))
			for i, s := range structs {
				agents[i] = s.(*models.Agent)
			}
		}
		if err != nil {
			return err
		}

		// TODO That loop makes len(agents) SELECTs, that can be slow. Optimize when needed.
		res = make([]inventorypb.Agent, len(agents))
		for i, row := range agents {
			agent, err := as.makeAgent(tx.Querier, row)
			if err != nil {
				return err
			}
			res[i] = agent
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
		row, err := get(tx.Querier, id)
		if err != nil {
			return err
		}
		res, err = as.makeAgent(tx.Querier, row)
		return err
	})
	return res, e
}

// AddPMMAgent inserts pmm-agent Agent with given parameters.
func (as *AgentsService) AddPMMAgent(ctx context.Context, nodeID string) (*inventorypb.PMMAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	var res *inventorypb.PMMAgent
	e := as.db.InTransaction(func(tx *reform.TX) error {
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		ns := NewNodesService(as.r)
		if _, err := ns.Get(ctx, as.db.Querier, nodeID); err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:      id,
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: pointer.ToStringOrNil(nodeID),
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		agent, err := as.makeAgent(tx.Querier, row)
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
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		pmmAgent, err := get(tx.Querier, req.PmmAgentId)
		if err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:    id,
			AgentType:  models.NodeExporterType,
			PMMAgentID: &req.PmmAgentId,
		}
		if err := row.SetCustomLabels(req.CustomLabels); err != nil {
			return err
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		err = tx.Insert(&models.AgentNode{
			AgentID: row.AgentID,
			NodeID:  pointer.GetString(pmmAgent.RunsOnNodeID),
		})
		if err != nil {
			return errors.WithStack(err)
		}

		agent, err := as.makeAgent(tx.Querier, row)
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
		row, err := get(tx.Querier, req.AgentId)
		if err != nil {
			return err
		}

		if req.GetEnabled() {
			row.Disabled = false
		}
		if req.GetDisabled() {
			row.Disabled = true
		}

		if req.RemoveCustomLabels {
			if err = row.SetCustomLabels(nil); err != nil {
				return err
			}
		}
		if len(req.CustomLabels) != 0 {
			if err = row.SetCustomLabels(req.CustomLabels); err != nil {
				return err
			}
		}

		if err = tx.Update(row); err != nil {
			return errors.WithStack(err)
		}

		agent, err := as.makeAgent(tx.Querier, row)
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
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, q *reform.Querier, req *inventorypb.AddMySQLdExporterRequest) (*inventorypb.MySQLdExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.MySQLdExporter
	id := "/agent_id/" + uuid.New().String()
	if err := checkUniqueID(q, id); err != nil {
		return nil, err
	}

	ns := NewNodesService(as.r)
	ss := NewServicesService(as.r, ns)
	if _, err := ss.Get(ctx, q, req.ServiceId); err != nil {
		return nil, err
	}

	row := &models.Agent{
		AgentID:    id,
		AgentType:  models.MySQLdExporterType,
		PMMAgentID: &req.PmmAgentId,
		Username:   pointer.ToStringOrNil(req.Username),
		Password:   pointer.ToStringOrNil(req.Password),
	}
	if err := row.SetCustomLabels(req.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err := q.Insert(&models.AgentService{
		AgentID:   row.AgentID,
		ServiceID: req.ServiceId,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	agent, err := as.makeAgent(q, row)
	if err != nil {
		return nil, err
	}

	res = agent.(*inventorypb.MySQLdExporter)

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
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
func (as *AgentsService) AddMongoDBExporter(ctx context.Context, q *reform.Querier, req *inventorypb.AddMongoDBExporterRequest) (*inventorypb.MongoDBExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.MongoDBExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		ns := NewNodesService(as.r)
		ss := NewServicesService(as.r, ns)
		if _, err := ss.Get(ctx, q, req.ServiceId); err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:    id,
			AgentType:  models.MongoDBExporterType,
			PMMAgentID: &req.PmmAgentId,
			Username:   pointer.ToStringOrNil(req.Username),
			Password:   pointer.ToStringOrNil(req.Password),
		}
		if err := row.SetCustomLabels(req.CustomLabels); err != nil {
			return err
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		err := tx.Insert(&models.AgentService{
			AgentID:   row.AgentID,
			ServiceID: req.ServiceId,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		agent, err := as.makeAgent(tx.Querier, row)
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

// AddQANMySQLPerfSchemaAgent adds MySQL PerfSchema QAN Agent.
//nolint:lll
func (as *AgentsService) AddQANMySQLPerfSchemaAgent(ctx context.Context, q *reform.Querier, req *inventorypb.AddQANMySQLPerfSchemaAgentRequest) (*inventorypb.QANMySQLPerfSchemaAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.QANMySQLPerfSchemaAgent
	id := "/agent_id/" + uuid.New().String()
	if err := checkUniqueID(q, id); err != nil {
		return nil, err
	}

	ns := NewNodesService(as.r)
	ss := NewServicesService(as.r, ns)
	if _, err := ss.Get(ctx, q, req.ServiceId); err != nil {
		return nil, err
	}

	row := &models.Agent{
		AgentID:    id,
		AgentType:  models.QANMySQLPerfSchemaAgentType,
		PMMAgentID: &req.PmmAgentId,
		Username:   pointer.ToStringOrNil(req.Username),
		Password:   pointer.ToStringOrNil(req.Password),
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err := q.Insert(&models.AgentService{
		AgentID:   row.AgentID,
		ServiceID: req.ServiceId,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	agent, err := as.makeAgent(q, row)
	if err != nil {
		return nil, err
	}
	res = agent.(*inventorypb.QANMySQLPerfSchemaAgent)

	as.r.SendSetStateRequest(ctx, req.PmmAgentId)
	return res, err
}

// AddPostgresExporter inserts postgres_exporter Agent with given parameters.
func (as *AgentsService) AddPostgresExporter(ctx context.Context, q *reform.Querier, req *inventorypb.AddPostgresExporterRequest) (*inventorypb.PostgresExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416

	var res *inventorypb.PostgresExporter
	e := as.db.InTransaction(func(tx *reform.TX) error {
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		ns := NewNodesService(as.r)
		ss := NewServicesService(as.r, ns)
		if _, err := ss.Get(ctx, q, req.ServiceId); err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:    id,
			AgentType:  models.PostgresExporterType,
			PMMAgentID: &req.PmmAgentId,
			Username:   pointer.ToStringOrNil(req.Username),
			Password:   pointer.ToStringOrNil(req.Password),
		}
		if err := row.SetCustomLabels(req.CustomLabels); err != nil {
			return err
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		err := tx.Insert(&models.AgentService{
			AgentID:   row.AgentID,
			ServiceID: req.ServiceId,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		agent, err := as.makeAgent(tx.Querier, row)
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

// Remove deletes Agent by ID.
func (as *AgentsService) Remove(ctx context.Context, id string) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	return as.db.InTransaction(func(tx *reform.TX) error {
		row, err := get(tx.Querier, id)
		if err != nil {
			return err
		}

		if _, err = tx.DeleteFrom(models.AgentServiceView, "WHERE agent_id = "+tx.Placeholder(1), id); err != nil { //nolint:gosec
			return errors.WithStack(err)
		}
		if _, err = tx.DeleteFrom(models.AgentNodeView, "WHERE agent_id = "+tx.Placeholder(1), id); err != nil { //nolint:gosec
			return errors.WithStack(err)
		}

		if err = tx.Delete(row); err != nil {
			return errors.WithStack(err)
		}

		if pointer.GetString(row.PMMAgentID) != "" {
			as.r.SendSetStateRequest(ctx, pointer.GetString(row.PMMAgentID))
		}

		if row.AgentType == models.PMMAgentType {
			as.r.Kick(ctx, id)
		}

		return nil
	})
}
