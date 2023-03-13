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
	err := s.validateRequest(req)
	if err != nil {
		return nil, err
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
			svcAgents = append(svcAgents, &managementpb.GenericAgent{
				AgentId:     agent.AgentID,
				AgentType:   string(agent.AgentType),
				Status:      agent.Status,
				IsConnected: s.r.IsConnected(agent.AgentID),
			})
		}

		for _, node := range nodes {
			// case #2: the agent runs on the same node as the service
			// case #3: the agent runs externally, i.e. runs_on_node_id is set
			if pointer.GetString(agent.NodeID) == node.NodeID || pointer.GetString(agent.RunsOnNodeID) == node.NodeID {
				svcAgents = append(svcAgents, &managementpb.GenericAgent{
					AgentId:     agent.AgentID,
					AgentType:   string(agent.AgentType),
					Status:      agent.Status,
					IsConnected: s.r.IsConnected(agent.AgentID),
				})
			}
		}
	}

	res := &managementpb.ListAgentResponse{
		Agents: svcAgents,
	}

	return res, nil
}

func (s *AgentService) validateRequest(request *managementpb.ListAgentRequest) error {
	if request.ServiceId == "" {
		return status.Error(codes.InvalidArgument, "service_id expected")
	}
	return nil
}
