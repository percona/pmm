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

// AgentFilters represents filters for agents list.
type AgentFilters struct {
	// Return only Agents started by this pmm-agent.
	PMMAgentID string
	// Return only Agents that provide insights for that Node.
	NodeID string
	// Return only Agents that provide insights for that Service.
	ServiceID string
	// Return Agents with provided type.
	AgentType *AgentType
}

// FindAgents returns Agents by filters.
func FindAgents(q *reform.Querier, filters AgentFilters) ([]*Agent, error) {
	var conditions []string
	var args []interface{}
	idx := 1
	if filters.PMMAgentID != "" {
		if _, err := FindAgentByID(q, filters.PMMAgentID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("pmm_agent_id = %s", q.Placeholder(idx)))
		args = append(args, filters.PMMAgentID)
		idx++
	}
	if filters.NodeID != "" {
		if _, err := FindNodeByID(q, filters.NodeID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("node_id = %s", q.Placeholder(idx)))
		args = append(args, filters.NodeID)
		idx++
	}
	if filters.ServiceID != "" {
		if _, err := FindServiceByID(q, filters.ServiceID); err != nil {
			return nil, err
		}
		conditions = append(conditions, fmt.Sprintf("service_id = %s", q.Placeholder(idx)))
		args = append(args, filters.ServiceID)
		idx++
	}
	if filters.AgentType != nil {
		conditions = append(conditions, fmt.Sprintf("agent_type = %s", q.Placeholder(idx)))
		args = append(args, *filters.AgentType)
	}

	var whereClause string
	if len(conditions) != 0 {
		whereClause = fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND "))
	}
	structs, err := q.SelectAllFrom(AgentTable, fmt.Sprintf("%s ORDER BY agent_id", whereClause), args...)
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

// FindAgentsByIDs finds Agents by IDs.
func FindAgentsByIDs(q *reform.Querier, ids []string) ([]*Agent, error) {
	if len(ids) == 0 {
		return []*Agent{}, nil
	}

	p := strings.Join(q.Placeholders(1, len(ids)), ", ")
	tail := fmt.Sprintf("WHERE agent_id IN (%s) ORDER BY agent_id", p) //nolint:gosec
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	structs, err := q.SelectAllFrom(AgentTable, tail, args...)
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

	res := make([]*Agent, 0, len(structs))
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
	atail := fmt.Sprintf("WHERE agent_id IN (%s) AND agent_type = '%s' ORDER BY agent_id", ph, PMMAgentType) //nolint:gosec
	pmmAgentRecords, err := q.SelectAllFrom(AgentTable, atail, pmmAgentIDs...)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Couldn't get pmm-agents for service %s", serviceID)
	}
	res := make([]*Agent, 0, len(pmmAgentRecords))
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

// CreateExternalExporterParams params for add external exporter.
type CreateExternalExporterParams struct {
	RunsOnNodeID string
	ServiceID    string
	Username     string
	Password     string
	Scheme       string
	MetricsPath  string
	ListenPort   uint32
	CustomLabels map[string]string
}

// CreateExternalExporter creates ExternalExporter.
func CreateExternalExporter(q *reform.Querier, params *CreateExternalExporterParams) (*Agent, error) {
	if !(params.ListenPort > 0 && params.ListenPort < 65536) {
		return nil, status.Errorf(codes.InvalidArgument, "Listen port should be between 1 and 65535.")
	}
	id := "/agent_id/" + uuid.New().String()
	if err := checkUniqueAgentID(q, id); err != nil {
		return nil, err
	}

	if _, err := FindNodeByID(q, params.RunsOnNodeID); err != nil {
		return nil, err
	}
	if _, err := FindServiceByID(q, params.ServiceID); err != nil {
		return nil, err
	}

	scheme := params.Scheme
	if scheme == "" {
		scheme = "http"
	}
	metricsPath := params.MetricsPath
	if metricsPath == "" {
		metricsPath = "/metrics"
	}
	row := &Agent{
		AgentID:       id,
		AgentType:     ExternalExporterType,
		RunsOnNodeID:  &params.RunsOnNodeID,
		ServiceID:     pointer.ToStringOrNil(params.ServiceID),
		Username:      pointer.ToStringOrNil(params.Username),
		Password:      pointer.ToStringOrNil(params.Password),
		MetricsScheme: &scheme,
		MetricsPath:   &metricsPath,
		ListenPort:    pointer.ToUint16(uint16(params.ListenPort)),
	}
	if err := row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}
	if err := q.Insert(row); err != nil {
		return nil, errors.WithStack(err)
	}

	return row, nil
}

// CreateAgentParams params for add common exporter.
type CreateAgentParams struct {
	PMMAgentID                     string
	NodeID                         string
	ServiceID                      string
	Username                       string
	Password                       string
	CustomLabels                   map[string]string
	TLS                            bool
	TLSSkipVerify                  bool
	TableCountTablestatsGroupLimit int32
	QueryExamplesDisabled          bool
	MaxQueryLogSize                int64
	AWSAccessKey                   string
	AWSSecretKey                   string
	RDSBasicMetricsDisabled        bool
	RDSEnhancedMetricsDisabled     bool
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

	if params.NodeID != "" {
		if _, err := FindNodeByID(q, params.NodeID); err != nil {
			return nil, err
		}
	}
	if params.ServiceID != "" {
		if _, err := FindServiceByID(q, params.ServiceID); err != nil {
			return nil, err
		}
	}

	row := &Agent{
		AgentID:                        id,
		AgentType:                      agentType,
		PMMAgentID:                     &params.PMMAgentID,
		ServiceID:                      pointer.ToStringOrNil(params.ServiceID),
		NodeID:                         pointer.ToStringOrNil(params.NodeID),
		Username:                       pointer.ToStringOrNil(params.Username),
		Password:                       pointer.ToStringOrNil(params.Password),
		TLS:                            params.TLS,
		TLSSkipVerify:                  params.TLSSkipVerify,
		TableCountTablestatsGroupLimit: params.TableCountTablestatsGroupLimit,
		QueryExamplesDisabled:          params.QueryExamplesDisabled,
		MaxQueryLogSize:                params.MaxQueryLogSize,
		AWSAccessKey:                   pointer.ToStringOrNil(params.AWSAccessKey),
		AWSSecretKey:                   pointer.ToStringOrNil(params.AWSSecretKey),
		RDSBasicMetricsDisabled:        params.RDSBasicMetricsDisabled,
		RDSEnhancedMetricsDisabled:     params.RDSEnhancedMetricsDisabled,
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

	if id == PMMServerAgentID {
		return nil, status.Error(codes.PermissionDenied, "pmm-agent on PMM Server can't be removed.")
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
