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

package management

import (
	"context"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/models"
)

// AgentService represents service for working with agents.
type AgentService struct {
	db    *reform.DB
	r     agentsRegistry
	state agentsStateUpdater
	vmdb  prometheusService

	managementpb.UnimplementedAgentServer
}

// NewServiceService creates ServiceService instance.
func NewAgentService(db *reform.DB, r agentsRegistry, state agentsStateUpdater, vmdb prometheusService) *AgentService {
	return &AgentService{
		db:    db,
		r:     r,
		state: state,
		vmdb:  vmdb,
	}
}

// ListAgents returns a filtered list of Agents.
//
//nolint:unparam
func (s *AgentService) ListAgents(ctx context.Context, req *managementpb.ListAgentRequest) (*managementpb.ListAgentResponse, error) {
	if req.ServiceId == "" {
		return nil, status.Error(codes.InvalidArgument, "service_id is required")
	}

	serviceID := req.GetServiceId()

	var agents []*models.Agent
	var nodes []*models.Node

	e := s.db.InTransaction(func(tx *reform.TX) error {
		var err error

		agents, err = models.FindAgents(tx.Querier, models.AgentFilters{})
		if err != nil {
			return err
		}

		nodes, err = models.FindNodes(tx.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		return nil
	})

	if e != nil {
		return nil, e
	}

	svcAgents := []*managementpb.GenericAgent{}

	for _, agent := range agents {
		// case #1: agent is an exporter for this service
		if agent.ServiceID != nil && pointer.GetString(agent.ServiceID) == serviceID {
			ag, err := s.toAPIAgent(agent)
			if err != nil {
				return nil, err
			}
			svcAgents = append(svcAgents, ag)
		}

		for _, node := range nodes {
			// case #2: the agent runs on the same node as the service
			// case #3: the agent runs externally, i.e. runs_on_node_id is set
			// NOTE: conditions 1 and 2-3 are mutually exclusive
			if pointer.GetString(agent.NodeID) == node.NodeID || pointer.GetString(agent.RunsOnNodeID) == node.NodeID {
				ag, err := s.toAPIAgent(agent)
				if err != nil {
					return nil, err
				}
				svcAgents = append(svcAgents, ag)
			}
		}
	}

	res := &managementpb.ListAgentResponse{
		Agents: svcAgents,
	}

	return res, nil
}

func (s *AgentService) toAPIAgent(agent *models.Agent) (*managementpb.GenericAgent, error) {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	return &managementpb.GenericAgent{
		AgentId:                   agent.AgentID,
		AgentType:                 string(agent.AgentType),
		CustomLabels:              labels,
		Disabled:                  agent.Disabled,
		DisabledCollectors:        agent.DisabledCollectors,
		IsConnected:               s.r.IsConnected(agent.AgentID),
		LogLevel:                  pointer.GetString(agent.LogLevel),
		ListenPort:                uint32(pointer.GetUint16(agent.ListenPort)),
		PmmAgentId:                pointer.GetString(agent.PMMAgentID),
		ProcessExecPath:           pointer.GetString(agent.ProcessExecPath),
		PushMetricsEnabled:        agent.PushMetrics,
		RunsOnNodeId:              pointer.GetString(agent.RunsOnNodeID),
		Status:                    agent.Status,
		Tls:                       agent.TLS,
		TlsSkipVerify:             agent.TLSSkipVerify,
		TablestatsGroupTableLimit: agent.TableCountTablestatsGroupLimit,
		Username:                  pointer.GetString(agent.Username),
		Version:                   pointer.GetString(agent.Version),
	}, nil
}
