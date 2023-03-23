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
func (s *AgentService) ListAgents(ctx context.Context, req *managementpb.ListAgentRequest) (*managementpb.ListAgentResponse, error) {
	serviceID := req.GetServiceId()

	var agents []*models.Agent
	var service *models.Service

	e := s.db.InTransaction(func(tx *reform.TX) error {
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

	if e != nil {
		return nil, e
	}

	var svcAgents []*managementpb.GenericAgent

	for _, agent := range agents {
		// agent is not an exporter, but it runs on the same node as the service (p.e. pmm-agent)
		if agent.ServiceID == nil && pointer.GetString(agent.RunsOnNodeID) == service.NodeID {
			ag, err := s.toAPIAgent(agent)
			if err != nil {
				return nil, err
			}
			svcAgents = append(svcAgents, ag)
		}

		// vmagent that runs on the same node as the service
		if pointer.GetString(agent.NodeID) == service.NodeID && agent.AgentType == models.VMAgentType {
			ag, err := s.toAPIAgent(agent)
			if err != nil {
				return nil, err
			}
			svcAgents = append(svcAgents, ag)
		}

		// agent is an exporter for this service
		if agent.ServiceID != nil && pointer.GetString(agent.ServiceID) == serviceID {
			ag, err := s.toAPIAgent(agent)
			if err != nil {
				return nil, err
			}
			svcAgents = append(svcAgents, ag)
		}
	}

	return &managementpb.ListAgentResponse{Agents: svcAgents}, nil
}

func (s *AgentService) toAPIAgent(agent *models.Agent) (*managementpb.GenericAgent, error) {
	const pass = "**********"

	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	ga := &managementpb.GenericAgent{
		AgentId:                        agent.AgentID,
		AgentType:                      string(agent.AgentType),
		AwsAccessKey:                   pointer.GetString(agent.AWSAccessKey),
		CreatedAt:                      agent.CreatedAt.Unix() * 1000,
		CustomLabels:                   labels,
		Disabled:                       agent.Disabled,
		DisabledCollectors:             agent.DisabledCollectors,
		IsConnected:                    s.r.IsConnected(agent.AgentID),
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
		UpdatedAt:                      agent.UpdatedAt.Unix() * 1000,
		Version:                        pointer.GetString(agent.Version),
	}

	// Indicate that there is actually a secret, but don't disclose it.
	if agent.AgentPassword != nil {
		ga.AgentPassword = pass
	}
	if agent.AWSSecretKey != nil {
		ga.AwsSecretKey = pass
	}
	if agent.Password != nil {
		ga.Password = pass
	}

	return ga, nil
}
