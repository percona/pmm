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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	agentv1beta1 "github.com/percona/pmm/api/managementpb/agent"
	"github.com/percona/pmm/managed/models"
)

// AgentService represents service for working with agents.
type AgentService struct {
	db *reform.DB
	r  agentsRegistry

	agentv1beta1.UnimplementedAgentServer
}

// NewAgentService creates AgentService instance.
func NewAgentService(db *reform.DB, r agentsRegistry) *AgentService {
	return &AgentService{
		db: db,
		r:  r,
	}
}

// ListAgents returns a filtered list of Agents.
//
//nolint:unparam
func (s *AgentService) ListAgents(ctx context.Context, req *agentv1beta1.ListAgentRequest) (*agentv1beta1.ListAgentResponse, error) {
	serviceID := req.ServiceId

	var agents []*models.Agent
	var service *models.Service

	// TODO: provide a higher level of data consistency guarantee by using a locking mechanism.
	errTX := s.db.InTransaction(func(tx *reform.TX) error {
		var err error

		agents, err = models.FindAgents(tx.Querier, models.AgentFilters{})
		if err != nil {
			return err
		}

		service, err = models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	var svcAgents []*agentv1beta1.UniversalAgent

	for _, agent := range agents {
		if IsNodeAgent(agent, service) || IsVMAgent(agent, service) || IsServiceAgent(agent, service) {
			ag, err := s.agentToAPI(agent)
			if err != nil {
				return nil, err
			}
			svcAgents = append(svcAgents, ag)
		}
	}

	return &agentv1beta1.ListAgentResponse{Agents: svcAgents}, nil
}

func (s *AgentService) agentToAPI(agent *models.Agent) (*agentv1beta1.UniversalAgent, error) {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	return &agentv1beta1.UniversalAgent{
		AgentId:                        agent.AgentID,
		AgentType:                      string(agent.AgentType),
		AwsAccessKey:                   pointer.GetString(agent.AWSAccessKey),
		CreatedAt:                      timestamppb.New(agent.CreatedAt),
		CustomLabels:                   labels,
		Disabled:                       agent.Disabled,
		DisabledCollectors:             agent.DisabledCollectors,
		IsConnected:                    s.r.IsConnected(agent.AgentID),
		IsAgentPasswordSet:             agent.AgentPassword != nil,
		IsAwsSecretKeySet:              agent.AWSSecretKey != nil,
		IsPasswordSet:                  agent.Password != nil,
		ListenPort:                     uint32(pointer.GetUint16(agent.ListenPort)),
		LogLevel:                       pointer.GetString(agent.LogLevel),
		MaxQueryLength:                 agent.MaxQueryLength,
		MaxQueryLogSize:                agent.MaxQueryLogSize,
		MetricsPath:                    pointer.GetString(agent.MetricsPath),
		MetricsScheme:                  pointer.GetString(agent.MetricsScheme),
		NodeId:                         pointer.GetString(agent.NodeID),
		PmmAgentId:                     pointer.GetString(agent.PMMAgentID),
		ProcessExecPath:                pointer.GetString(agent.ProcessExecPath),
		PushMetrics:                    agent.PushMetrics,
		QueryExamplesDisabled:          agent.QueryExamplesDisabled,
		RdsBasicMetricsDisabled:        agent.RDSBasicMetricsDisabled,
		RdsEnhancedMetricsDisabled:     agent.RDSEnhancedMetricsDisabled,
		RunsOnNodeId:                   pointer.GetString(agent.RunsOnNodeID),
		ServiceId:                      pointer.GetString(agent.ServiceID),
		Status:                         agent.Status,
		TableCount:                     pointer.GetInt32(agent.TableCount),
		TableCountTablestatsGroupLimit: agent.TableCountTablestatsGroupLimit,
		Tls:                            agent.TLS,
		TlsSkipVerify:                  agent.TLSSkipVerify,
		Username:                       pointer.GetString(agent.Username),
		UpdatedAt:                      timestamppb.New(agent.UpdatedAt),
		Version:                        pointer.GetString(agent.Version),
	}, nil
}
