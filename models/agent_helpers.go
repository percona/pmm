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

package models

import (
	"fmt"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

// AgentFindByID finds agent by ID.
func AgentFindByID(q *reform.Querier, id string) (*Agent, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Agent ID.")
	}

	row := &Agent{AgentID: id}
	switch err := q.Reload(row); err {
	case nil:
		return row, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Agent with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

// AgentFindAll finds all agents.
func AgentFindAll(q *reform.Querier) ([]*Agent, error) {
	var structs []reform.Struct
	structs, err := q.SelectAllFrom(AgentTable, "ORDER BY agent_id")
	err = errors.Wrap(err, "failed to select Agents")
	agents := make([]*Agent, len(structs))
	for i, s := range structs {
		agents[i] = s.(*Agent)
	}
	return agents, err
}

func agentNewID(q *reform.Querier) (string, error) {
	id := "/agent_id/" + uuid.New().String()
	row := &Agent{AgentID: id}
	switch err := q.Reload(row); err {
	case nil:
		return "", status.Errorf(codes.AlreadyExists, "Agent with ID %q already exists.", id)
	case reform.ErrNoRows:
		return id, nil
	default:
		return "", errors.WithStack(err)
	}
}

// AgentAddPmmAgent creates PMMAgent.
func AgentAddPmmAgent(q *reform.Querier, runsOnNodeID string, customLabels map[string]string) (*Agent, error) {
	id, err := agentNewID(q)
	if err != nil {
		return nil, err
	}

	if _, err := FindNodeByID(q, runsOnNodeID); err != nil {
		return nil, err
	}

	row := &Agent{
		AgentID:      id,
		AgentType:    PMMAgentType,
		RunsOnNodeID: &runsOnNodeID,
	}
	if err := row.SetCustomLabels(customLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// AgentAddNodeExporter creates NodeExporter agent.
func AgentAddNodeExporter(q *reform.Querier, pmmAgentID string, customLabels map[string]string) (*Agent, error) {
	id, err := agentNewID(q)
	if err != nil {
		return nil, err
	}

	pmmAgent, err := AgentFindByID(q, pmmAgentID)
	if err != nil {
		return nil, err
	}

	row := &Agent{
		AgentID:    id,
		AgentType:  NodeExporterType,
		PMMAgentID: &pmmAgentID,
	}
	if err := row.SetCustomLabels(customLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err = q.Insert(&AgentNode{
		AgentID: row.AgentID,
		NodeID:  pointer.GetString(pmmAgent.RunsOnNodeID),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return row, nil
}

// ChangeCommonExporterParams describe common change params for exporters.
type ChangeCommonExporterParams struct {
	AgentID            string
	CustomLabels       map[string]string
	Disabled           bool
	RemoveCustomLabels bool
}

// AgentChangeExporter changes common params for given agent.
func AgentChangeExporter(q *reform.Querier, params *ChangeCommonExporterParams) (*Agent, error) {
	row, err := AgentFindByID(q, params.AgentID)
	if err != nil {
		return nil, err
	}

	row.Disabled = params.Disabled

	if params.RemoveCustomLabels {
		if err = row.SetCustomLabels(nil); err != nil {
			return nil, err
		}
	}
	if len(params.CustomLabels) != 0 {
		if err = row.SetCustomLabels(params.CustomLabels); err != nil {
			return nil, err
		}
	}

	if err = q.Update(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// AddExporterAgentParams params for add common exporter.
type AddExporterAgentParams struct {
	PMMAgentID   string
	ServiceID    string
	Username     string
	Password     string
	CustomLabels map[string]string
}

// AgentAddExporter adds exporter with given type.
func AgentAddExporter(q *reform.Querier, agentType AgentType, params *AddExporterAgentParams) (*Agent, error) {
	id, err := agentNewID(q)
	if err != nil {
		return nil, err
	}

	if _, err := FindServiceByID(q, params.ServiceID); err != nil {
		return nil, err
	}

	row := &Agent{
		AgentID:    id,
		AgentType:  agentType,
		PMMAgentID: &params.PMMAgentID,
		Username:   pointer.ToStringOrNil(params.Username),
		Password:   pointer.ToStringOrNil(params.Password),
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	err = q.Insert(&AgentService{
		AgentID:   row.AgentID,
		ServiceID: params.ServiceID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// AgentsForNode returns all Agents providing insights for given Node.
func AgentsForNode(q *reform.Querier, nodeID string) ([]*Agent, error) {
	structs, err := q.FindAllFrom(AgentNodeView, "node_id", nodeID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agent IDs")
	}

	agentIDs := make([]interface{}, len(structs))
	for i, s := range structs {
		agentIDs[i] = s.(*AgentNode).AgentID
	}
	if len(agentIDs) == 0 {
		return []*Agent{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(agentIDs)), ", ")
	tail := fmt.Sprintf("WHERE agent_id IN (%s) ORDER BY agent_id", p) //nolint:gosec
	structs, err = q.SelectAllFrom(AgentTable, tail, agentIDs...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}
	return res, nil
}

// AgentsRunningByPMMAgent returns all Agents running by PMMAgent.
func AgentsRunningByPMMAgent(q *reform.Querier, pmmAgentID string) ([]*Agent, error) {
	tail := fmt.Sprintf("WHERE pmm_agent_id = %s ORDER BY agent_id", q.Placeholder(1)) //nolint:gosec
	structs, err := q.SelectAllFrom(AgentTable, tail, pmmAgentID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}
	return res, nil
}

// AgentsForService returns all Agents providing insights for given Service.
func AgentsForService(q *reform.Querier, serviceID string) ([]*Agent, error) {
	structs, err := q.FindAllFrom(AgentServiceView, "service_id", serviceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agent IDs")
	}

	agentIDs := make([]interface{}, len(structs))
	for i, s := range structs {
		agentIDs[i] = s.(*AgentService).AgentID
	}
	if len(agentIDs) == 0 {
		return []*Agent{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(agentIDs)), ", ")
	tail := fmt.Sprintf("WHERE agent_id IN (%s) ORDER BY agent_id", p) //nolint:gosec
	structs, err = q.SelectAllFrom(AgentTable, tail, agentIDs...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}
	return res, nil
}

// PMMAgentsForChangedNode returns pmm-agents IDs that are affected
// by the change of the Node with given ID.
// It may return (nil, nil) if no such pmm-agents are found.
// It returns wrapped reform.ErrNoRows if Service with given ID is not found.
func PMMAgentsForChangedNode(q *reform.Querier, nodeID string) ([]string, error) {
	// TODO Real code.
	// Returning all pmm-agents is currently safe, but not optimal for large number of Agents.
	_ = nodeID

	structs, err := q.SelectAllFrom(AgentTable, "ORDER BY agent_id")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	var res []string
	for _, str := range structs {
		row := str.(*Agent)
		if row.AgentType == PMMAgentType {
			res = append(res, row.AgentID)
		}
	}
	return res, nil
}

// PMMAgentsForChangedService returns pmm-agents IDs that are affected
// by the change of the Service with given ID.
// It may return (nil, nil) if no such pmm-agents are found.
// It returns wrapped reform.ErrNoRows if Service with given ID is not found.
func PMMAgentsForChangedService(q *reform.Querier, serviceID string) ([]string, error) {
	// TODO Real code. We need to returns IDs of pmm-agents that:
	// * run Agents providing insights for this Service;
	// * run Agents providing insights for Node that hosts this Service.
	// Returning all pmm-agents is currently safe, but not optimal for large number of Agents.
	_ = serviceID

	structs, err := q.SelectAllFrom(AgentTable, "ORDER BY agent_id")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}

	var res []string
	for _, str := range structs {
		row := str.(*Agent)
		if row.AgentType == PMMAgentType {
			res = append(res, row.AgentID)
		}
	}
	return res, nil
}

// AgentRemove removes Agent by ID.
func AgentRemove(q *reform.Querier, id string) (*Agent, error) {
	row, err := AgentFindByID(q, id)
	if err != nil {
		return nil, err
	}

	if _, err = q.DeleteFrom(AgentServiceView, "WHERE agent_id = "+q.Placeholder(1), id); err != nil { //nolint:gosec
		return nil, errors.WithStack(err)
	}
	if _, err = q.DeleteFrom(AgentNodeView, "WHERE agent_id = "+q.Placeholder(1), id); err != nil { //nolint:gosec
		return nil, errors.WithStack(err)
	}

	if err = q.Delete(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}
