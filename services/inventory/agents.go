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
	api "github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// AgentsService works with inventory API Agents.
type AgentsService struct {
	r registry
}

func NewAgentsService(r registry) *AgentsService {
	return &AgentsService{
		r: r,
	}
}

// makeAgent converts database row to Inventory API Agent.
func (as *AgentsService) makeAgent(q *reform.Querier, row *models.Agent) (api.Agent, error) {
	labels, err := row.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch row.AgentType {
	case models.PMMAgentType:
		return &api.PMMAgent{
			AgentId:      row.AgentID,
			NodeId:       row.RunsOnNodeID,
			Connected:    as.r.IsConnected(row.AgentID),
			CustomLabels: labels,
		}, nil

	case models.NodeExporterType:
		return &api.NodeExporter{
			AgentId:      row.AgentID,
			NodeId:       row.RunsOnNodeID,
			Status:       api.AgentStatus(api.AgentStatus_value[row.Status]),
			ListenPort:   uint32(pointer.GetUint16(row.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.MySQLdExporterType:
		services, err := models.ServicesForAgent(q, row.AgentID)
		if err != nil {
			return nil, err
		}
		if len(services) != 1 {
			return nil, errors.Errorf("expected exactly one Services, got %d", len(services))
		}

		return &api.MySQLdExporter{
			AgentId:      row.AgentID,
			RunsOnNodeId: row.RunsOnNodeID,
			ServiceId:    services[0].ServiceID,
			Username:     pointer.GetString(row.Username),
			Password:     pointer.GetString(row.Password),
			Status:       api.AgentStatus(api.AgentStatus_value[row.Status]),
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
	// Return only Agents running on that Node.
	RunsOnNodeID string
	// Return only Agents that provide insights for that Node.
	NodeID string
	// Return only Agents that provide insights for that Service.
	ServiceID string
}

// List selects all Agents in a stable order for a given service.
func (as *AgentsService) List(ctx context.Context, db *reform.DB, filters AgentFilters) ([]api.Agent, error) {
	var res []api.Agent
	e := db.InTransaction(func(tx *reform.TX) error {
		var agents []*models.Agent
		var err error
		switch {
		case filters.RunsOnNodeID != "":
			agents, err = models.AgentsRunningOnNode(tx.Querier, filters.RunsOnNodeID)
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
		res = make([]api.Agent, len(agents))
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
func (as *AgentsService) Get(ctx context.Context, db *reform.DB, id string) (api.Agent, error) {
	var res api.Agent
	e := db.InTransaction(func(tx *reform.TX) error {
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
func (as *AgentsService) AddPMMAgent(ctx context.Context, db *reform.DB, nodeID string) (*api.PMMAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	var res *api.PMMAgent
	e := db.InTransaction(func(tx *reform.TX) error {
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		ns := NewNodesService(tx.Querier, as.r)
		if _, err := ns.get(ctx, nodeID); err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:      id,
			AgentType:    models.PMMAgentType,
			RunsOnNodeID: nodeID,
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		err := tx.Insert(&models.AgentNode{
			AgentID: row.AgentID,
			NodeID:  nodeID,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		agent, err := as.makeAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*api.PMMAgent)
		return nil
	})
	return res, e
}

// AddNodeExporter inserts node_exporter Agent with given parameters.
func (as *AgentsService) AddNodeExporter(ctx context.Context, db *reform.DB, req *api.AddNodeExporterRequest) (*api.NodeExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	var res *api.NodeExporter
	var pmmAgentID string
	e := db.InTransaction(func(tx *reform.TX) error {
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		ns := NewNodesService(tx.Querier, as.r)
		if _, err := ns.get(ctx, req.NodeId); err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:      id,
			AgentType:    models.NodeExporterType,
			RunsOnNodeID: req.NodeId,
		}
		if err := row.SetCustomLabels(req.CustomLabels); err != nil {
			return err
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		err := tx.Insert(&models.AgentNode{
			AgentID: row.AgentID,
			NodeID:  req.NodeId,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		pmmAgentID, err = models.PMMAgentForAgent(tx.Querier, row.AgentID)
		if err != nil {
			return err
		}

		agent, err := as.makeAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*api.NodeExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	if pmmAgentID != "" {
		as.r.SendSetStateRequest(ctx, pmmAgentID)
	}
	return res, nil
}

// AddMySQLdExporter inserts mysqld_exporter Agent with given parameters.
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, db *reform.DB, req *api.AddMySQLdExporterRequest) (*api.MySQLdExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	var res *api.MySQLdExporter
	var pmmAgentID string
	e := db.InTransaction(func(tx *reform.TX) error {
		id := "/agent_id/" + uuid.New().String()
		if err := checkUniqueID(tx.Querier, id); err != nil {
			return err
		}

		ns := NewNodesService(tx.Querier, as.r)
		if _, err := ns.get(ctx, req.RunsOnNodeId); err != nil {
			return err
		}

		ss := NewServicesService(tx.Querier, as.r)
		if _, err := ss.get(ctx, req.ServiceId); err != nil {
			return err
		}

		row := &models.Agent{
			AgentID:      id,
			AgentType:    models.MySQLdExporterType,
			RunsOnNodeID: req.RunsOnNodeId,
			Username:     pointer.ToStringOrNil(req.Username),
			Password:     pointer.ToStringOrNil(req.Password),
		}
		if err := row.SetCustomLabels(req.CustomLabels); err != nil {
			return err
		}
		if err := tx.Insert(row); err != nil {
			return errors.WithStack(err)
		}

		err := tx.Insert(&models.AgentNode{
			AgentID: row.AgentID,
			NodeID:  req.RunsOnNodeId,
		})
		if err != nil {
			return errors.WithStack(err)
		}
		err = tx.Insert(&models.AgentService{
			AgentID:   row.AgentID,
			ServiceID: req.ServiceId,
		})
		if err != nil {
			return errors.WithStack(err)
		}

		pmmAgentID, err = models.PMMAgentForAgent(tx.Querier, row.AgentID)
		if err != nil {
			return err
		}

		agent, err := as.makeAgent(tx.Querier, row)
		if err != nil {
			return err
		}
		res = agent.(*api.MySQLdExporter)
		return nil
	})
	if e != nil {
		return nil, e
	}

	if pmmAgentID != "" {
		as.r.SendSetStateRequest(ctx, pmmAgentID)
	}
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

// Remove deletes Agent by ID.
func (as *AgentsService) Remove(ctx context.Context, db *reform.DB, id string) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	return db.InTransaction(func(tx *reform.TX) error {
		row, err := get(tx.Querier, id)
		if err != nil {
			return err
		}

		pmmAgentID, err := models.PMMAgentForAgent(tx.Querier, row.AgentID)
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

		if pmmAgentID != "" {
			as.r.SendSetStateRequest(ctx, pmmAgentID)
		}

		if row.AgentType == models.PMMAgentType {
			as.r.Kick(ctx, id)
		}

		return nil
	})
}
