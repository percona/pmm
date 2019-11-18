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

func checkUniqueAgentID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Agent ID")
	}

	agent := &Agent{AgentID: id}
	switch err := q.Reload(agent); err {
	case nil:
		return status.Errorf(codes.AlreadyExists, "Agent with ID %q already exists.", id)
	case reform.ErrNoRows:
		return nil
	default:
		return errors.WithStack(err)
	}
}

// FindAllAgents returns all Agents.
func FindAllAgents(q *reform.Querier) ([]*Agent, error) {
	structs, err := q.SelectAllFrom(AgentTable, "ORDER BY agent_id")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	agents := make([]*Agent, len(structs))
	for i, s := range structs {
		agents[i] = s.(*Agent)
	}

	return agents, nil
}

// FindAgentByID finds Agent by ID.
func FindAgentByID(q *reform.Querier, id string) (*Agent, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Agent ID.")
	}

	agent := &Agent{AgentID: id}
	switch err := q.Reload(agent); err {
	case nil:
		return agent, nil
	case reform.ErrNoRows:
		return nil, status.Errorf(codes.NotFound, "Agent with ID %q not found.", id)
	default:
		return nil, errors.WithStack(err)
	}
}

func findAgentsByIDs(q *reform.Querier, ids []interface{}) ([]*Agent, error) {
	if len(ids) == 0 {
		return []*Agent{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE agent_id IN (%s) ORDER BY agent_id", p) //nolint:gosec
	structs, err := q.SelectAllFrom(AgentTable, tail, ids...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}
	return res, nil
}

// FindAgentsForNode returns all Agents providing insights for given Node.
func FindAgentsForNode(q *reform.Querier, nodeID string) ([]*Agent, error) {
	if _, err := FindNodeByID(q, nodeID); err != nil {
		return nil, err
	}

	structs, err := q.FindAllFrom(AgentTable, "node_id", nodeID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}

	return res, nil
}

// FindAgentsForService returns all Agents providing insights for given Service.
func FindAgentsForService(q *reform.Querier, serviceID string) ([]*Agent, error) {
	if _, err := FindServiceByID(q, serviceID); err != nil {
		return nil, err
	}

	structs, err := q.FindAllFrom(AgentTable, "service_id", serviceID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}

	return res, nil
}

// FindAgentsRunningByPMMAgent returns all Agents running by PMMAgent.
func FindAgentsRunningByPMMAgent(q *reform.Querier, pmmAgentID string) ([]*Agent, error) {
	if _, err := FindAgentByID(q, pmmAgentID); err != nil {
		return nil, err
	}

	structs, err := q.SelectAllFrom(AgentTable, "WHERE pmm_agent_id = $1 ORDER BY agent_id", pmmAgentID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]*Agent, len(structs))
	for i, s := range structs {
		res[i] = s.(*Agent)
	}
	return res, nil
}

// FindPMMAgentsRunningOnNode gets pmm-agents for node where it runs.
func FindPMMAgentsRunningOnNode(q *reform.Querier, nodeID string) ([]*Agent, error) {
	structs, err := q.SelectAllFrom(AgentTable, "WHERE runs_on_node_id = $1 AND agent_type = $2", nodeID, PMMAgentType)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get agents by runs_on_node_id, %s", nodeID)
	}

	var res []*Agent
	for _, str := range structs {
		row := str.(*Agent)
		res = append(res, row)
	}

	if len(res) == 0 {
		return nil, status.Errorf(codes.NotFound, "Couldn't found any pmm-agents by NodeID")
	}

	return res, nil
}

// FindPMMAgentsForService gets pmm-agents for service.
func FindPMMAgentsForService(q *reform.Querier, serviceID string) ([]*Agent, error) {
	_, err := q.SelectOneFrom(ServiceTable, "WHERE service_id = $1", serviceID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get services by service_id, %s", serviceID)
	}

	// First, find agents with serviceID.
	allAgents, err := q.SelectAllFrom(AgentTable, "WHERE service_id = $1", serviceID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get all agents for service %s", serviceID)
	}
	pmmAgentIDs := make([]interface{}, len(allAgents))
	for _, str := range allAgents {
		row := str.(*Agent)
		if row.PMMAgentID != nil {
			for _, a := range pmmAgentIDs {
				if a == *row.PMMAgentID {
					break
				}
				pmmAgentIDs = append(pmmAgentIDs, *row.PMMAgentID)
			}
		}
	}

	// Last, find all pmm-agents.
	ph := strings.Join(q.Placeholders(1, len(pmmAgentIDs)), ", ")
	atail := fmt.Sprintf("WHERE agent_id IN (%s) AND agent_type = '%s'", ph, PMMAgentType) //nolint:gosec
	pmmAgentRecords, err := q.SelectAllFrom(AgentTable, atail, pmmAgentIDs...)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get pmm-agents for service %s", serviceID)
	}
	var res []*Agent
	for _, str := range pmmAgentRecords {
		row := str.(*Agent)
		res = append(res, row)
	}

	return res, nil
}

// createPMMAgentWithID creates PMMAgent with given ID.
func createPMMAgentWithID(q *reform.Querier, id, runsOnNodeID string, customLabels map[string]string) (*Agent, error) {
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	if _, err := FindNodeByID(q, runsOnNodeID); err != nil {
		return nil, err
	}

	// TODO https://jira.percona.com/browse/PMM-4496
	// Check that Node is not remote.

	agent := &Agent{
		AgentID:      id,
		AgentType:    PMMAgentType,
		RunsOnNodeID: &runsOnNodeID,
	}
	if err := agent.SetCustomLabels(customLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(agent); err != nil {
		return nil, errors.WithStack(err)
	}

	return agent, nil
}

// CreatePMMAgent creates PMMAgent.
func CreatePMMAgent(q *reform.Querier, runsOnNodeID string, customLabels map[string]string) (*Agent, error) {
	id := "/agent_id/" + uuid.New().String()
	return createPMMAgentWithID(q, id, runsOnNodeID, customLabels)
}

// CreateNodeExporter creates NodeExporter.
func CreateNodeExporter(q *reform.Querier, pmmAgentID string, customLabels map[string]string) (*Agent, error) {
	// TODO merge into CreateAgent

	id := "/agent_id/" + uuid.New().String()
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	pmmAgent, err := FindAgentByID(q, pmmAgentID)
	if err != nil {
		return nil, err
	}

	row := &Agent{
		AgentID:    id,
		AgentType:  NodeExporterType,
		PMMAgentID: &pmmAgentID,
		NodeID:     pmmAgent.RunsOnNodeID,
	}
	if err := row.SetCustomLabels(customLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// CreateAgentParams params for add common exporter.
type CreateAgentParams struct {
	PMMAgentID            string
	ServiceID             string
	Username              string
	Password              string
	CustomLabels          map[string]string
	TLS                   bool
	TLSSkipVerify         bool
	QueryExamplesDisabled bool
	MaxQueryLogSize       int64
}

// CreateAgent creates Agent with given type.
func CreateAgent(q *reform.Querier, agentType AgentType, params *CreateAgentParams) (*Agent, error) {
	id := "/agent_id/" + uuid.New().String()
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	if _, err := FindAgentByID(q, params.PMMAgentID); err != nil {
		return nil, err
	}

	if _, err := FindServiceByID(q, params.ServiceID); err != nil {
		return nil, err
	}

	row := &Agent{
		AgentID:               id,
		AgentType:             agentType,
		PMMAgentID:            &params.PMMAgentID,
		Username:              pointer.ToStringOrNil(params.Username),
		Password:              pointer.ToStringOrNil(params.Password),
		TLS:                   params.TLS,
		TLSSkipVerify:         params.TLSSkipVerify,
		QueryExamplesDisabled: params.QueryExamplesDisabled,
		MaxQueryLogSize:       params.MaxQueryLogSize,
		ServiceID:             pointer.ToStringOrNil(params.ServiceID),
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// ChangeCommonAgentParams contains parameters that can be changed for all Agents.
type ChangeCommonAgentParams struct {
	Disabled           *bool // true - disable, false - enable, nil - do not change
	CustomLabels       map[string]string
	RemoveCustomLabels bool
}

// ChangeAgent changes common parameters for given Agent.
func ChangeAgent(q *reform.Querier, agentID string, params *ChangeCommonAgentParams) (*Agent, error) {
	row, err := FindAgentByID(q, agentID)
	if err != nil {
		return nil, err
	}

	if params.Disabled != nil {
		if *params.Disabled {
			row.Disabled = true
		} else {
			row.Disabled = false
		}
	}

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

// RemoveAgent removes Agent by ID.
func RemoveAgent(q *reform.Querier, id string, mode RemoveMode) (*Agent, error) {
	a, err := FindAgentByID(q, id)
	if err != nil {
		return nil, err
	}

	structs, err := q.SelectAllFrom(AgentTable, "WHERE pmm_agent_id = $1", id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select Agents")
	}
	if len(structs) != 0 {
		switch mode {
		case RemoveRestrict:
			return nil, status.Errorf(codes.FailedPrecondition, "pmm-agent with ID %q has agents.", id)
		case RemoveCascade:
			for _, str := range structs {
				agentID := str.(*Agent).AgentID
				if _, err = RemoveAgent(q, agentID, RemoveRestrict); err != nil {
					return nil, err
				}
			}
		default:
			panic(fmt.Errorf("unhandled RemoveMode %v", mode))
		}
	}

	if err = q.Delete(a); err != nil {
		return nil, errors.Wrap(q.Delete(a), "failed to delete Agent")
	}

	return a, nil
}
