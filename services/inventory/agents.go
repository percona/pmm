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
	q *reform.Querier
	r registry
}

func NewAgentsService(q *reform.Querier, r registry) *AgentsService {
	return &AgentsService{
		q: q,
		r: r,
	}
}

// makeAgent converts database row to Inventory API Agent.
func (as *AgentsService) makeAgent(ctx context.Context, row *models.Agent) (api.Agent, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	switch row.AgentType {
	case models.PMMAgentType:
		return &api.PMMAgent{
			AgentId:   row.AgentID,
			NodeId:    row.RunsOnNodeID,
			Connected: as.r.IsConnected(row.AgentID),
		}, nil

	case models.NodeExporterType:
		return &api.NodeExporter{
			AgentId:    row.AgentID,
			NodeId:     row.RunsOnNodeID,
			Status:     api.AgentStatus(api.AgentStatus_value[row.Status]),
			ListenPort: uint32(pointer.GetUint16(row.ListenPort)),
		}, nil

	case models.MySQLdExporterType:
		services, err := models.ServicesForAgent(as.q, row.AgentID)
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
		}, nil

	default:
		panic(fmt.Errorf("unhandled Agent type %s", row.AgentType))
	}
}

func (as *AgentsService) get(ctx context.Context, id string) (*models.Agent, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Agent ID.")
	}

	row := &models.Agent{AgentID: id}
	switch err := as.q.Reload(row); err {
	case nil:
		return row, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Agent with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

func (as *AgentsService) checkUniqueID(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if id == "" {
		panic("empty Agent ID")
	}

	row := &models.Agent{AgentID: id}
	switch err := as.q.Reload(row); err {
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
func (as *AgentsService) List(ctx context.Context, filters AgentFilters) ([]api.Agent, error) {
	var agents []*models.Agent
	var err error
	switch {
	case filters.RunsOnNodeID != "":
		agents, err = models.AgentsRunningOnNode(as.q, filters.RunsOnNodeID)
	case filters.NodeID != "":
		agents, err = models.AgentsForNode(as.q, filters.NodeID)
	case filters.ServiceID != "":
		agents, err = models.AgentsForService(as.q, filters.ServiceID)
	default:
		var structs []reform.Struct
		structs, err = as.q.SelectAllFrom(models.AgentTable, "ORDER BY agent_id")
		err = errors.Wrap(err, "failed to select Agents")
		agents = make([]*models.Agent, len(structs))
		for i, s := range structs {
			agents[i] = s.(*models.Agent)
		}
	}
	if err != nil {
		return nil, err
	}

	// TODO That loop makes len(agents) SELECTs, that can be slow. Optimize when needed.
	res := make([]api.Agent, len(agents))
	for i, row := range agents {
		agent, err := as.makeAgent(ctx, row)
		if err != nil {
			return nil, err
		}
		res[i] = agent
	}
	return res, nil
}

// Get selects a single Agent by ID.
func (as *AgentsService) Get(ctx context.Context, id string) (api.Agent, error) {
	row, err := as.get(ctx, id)
	if err != nil {
		return nil, err
	}
	return as.makeAgent(ctx, row)
}

// AddPMMAgent inserts pmm-agent Agent with given parameters.
func (as *AgentsService) AddPMMAgent(ctx context.Context, nodeID string) (*api.PMMAgent, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	id := "/agent_id/" + uuid.New().String()
	if err := as.checkUniqueID(ctx, id); err != nil {
		return nil, err
	}

	ns := NewNodesService(as.q, as.r)
	if _, err := ns.get(ctx, nodeID); err != nil {
		return nil, err
	}

	row := &models.Agent{
		AgentID:      id,
		AgentType:    models.PMMAgentType,
		RunsOnNodeID: nodeID,
	}
	if err := as.q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err := as.q.Insert(&models.AgentNode{
		AgentID: row.AgentID,
		NodeID:  nodeID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	agent, err := as.makeAgent(ctx, row)
	if err != nil {
		return nil, err
	}
	return agent.(*api.PMMAgent), nil
}

// AddNodeExporter inserts node_exporter Agent with given parameters.
func (as *AgentsService) AddNodeExporter(ctx context.Context, nodeID string) (*api.NodeExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	id := "/agent_id/" + uuid.New().String()
	if err := as.checkUniqueID(ctx, id); err != nil {
		return nil, err
	}

	ns := NewNodesService(as.q, as.r)
	if _, err := ns.get(ctx, nodeID); err != nil {
		return nil, err
	}

	row := &models.Agent{
		AgentID:      id,
		AgentType:    models.NodeExporterType,
		RunsOnNodeID: nodeID,
	}
	if err := as.q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err := as.q.Insert(&models.AgentNode{
		AgentID: row.AgentID,
		NodeID:  nodeID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// send new state to pmm-agents
	agents, err := models.AgentsRunningOnNode(as.q, nodeID)
	if err != nil {
		return nil, err
	}
	for _, agent := range agents {
		if agent.AgentType != models.PMMAgentType {
			continue
		}
		as.r.SendSetStateRequest(ctx, agent.AgentID)
	}

	agent, err := as.makeAgent(ctx, row)
	if err != nil {
		return nil, err
	}
	return agent.(*api.NodeExporter), nil
}

// AddMySQLdExporter inserts mysqld_exporter Agent with given parameters.
func (as *AgentsService) AddMySQLdExporter(ctx context.Context, nodeID string, serviceID string, username, password *string) (*api.MySQLdExporter, error) {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// TODO Check runs-on Node: it must be BM, VM, DC (i.e. not remote, AWS RDS, etc.)

	id := "/agent_id/" + uuid.New().String()
	if err := as.checkUniqueID(ctx, id); err != nil {
		return nil, err
	}

	ns := NewNodesService(as.q, as.r)
	if _, err := ns.get(ctx, nodeID); err != nil {
		return nil, err
	}

	ss := NewServicesService(as.q, as.r)
	if _, err := ss.get(ctx, serviceID); err != nil {
		return nil, err
	}

	row := &models.Agent{
		AgentID:      id,
		AgentType:    models.MySQLdExporterType,
		RunsOnNodeID: nodeID,
		Username:     username,
		Password:     password,
	}
	if err := as.q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err := as.q.Insert(&models.AgentNode{
		AgentID: row.AgentID,
		NodeID:  nodeID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = as.q.Insert(&models.AgentService{
		AgentID:   row.AgentID,
		ServiceID: serviceID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// send new state to pmm-agents
	agents, err := models.AgentsRunningOnNode(as.q, nodeID)
	if err != nil {
		return nil, err
	}
	for _, agent := range agents {
		if agent.AgentType != models.PMMAgentType {
			continue
		}
		as.r.SendSetStateRequest(ctx, agent.AgentID)
	}

	agent, err := as.makeAgent(ctx, row)
	if err != nil {
		return nil, err
	}
	return agent.(*api.MySQLdExporter), nil
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
func (as *AgentsService) Remove(ctx context.Context, id string) error {
	// TODO Decide about validation. https://jira.percona.com/browse/PMM-1416
	// ID is not 0.

	row, err := as.get(ctx, id)
	if err != nil {
		return err
	}

	if _, err = as.q.DeleteFrom(models.AgentServiceView, "WHERE agent_id = "+as.q.Placeholder(1), id); err != nil { //nolint:gosec
		return errors.WithStack(err)
	}
	if _, err = as.q.DeleteFrom(models.AgentNodeView, "WHERE agent_id = "+as.q.Placeholder(1), id); err != nil { //nolint:gosec
		return errors.WithStack(err)
	}

	if err = as.q.Delete(row); err != nil {
		return errors.WithStack(err)
	}

	// TODO as.r.SendSetStateRequest(ctx, proper ID)

	if row.AgentType == models.PMMAgentType {
		as.r.Kick(ctx, id)
	}

	return nil
}
