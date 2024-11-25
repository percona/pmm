// Copyright (C) 2023 Percona LLC
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
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// ListAgents returns a filtered list of Agents.
func (s *ManagementService) ListAgents(ctx context.Context, req *managementv1.ListAgentsRequest) (*managementv1.ListAgentsResponse, error) {
	var err error
	err = s.validateListAgentRequest(req)
	if err != nil {
		return nil, err
	}

	var agents []*managementv1.UniversalAgent

	if req.ServiceId != "" {
		agents, err = s.listAgentsByServiceID(ctx, req.ServiceId)
	} else {
		agents, err = s.listAgentsByNodeID(req.NodeId)
	}
	if err != nil {
		return nil, err
	}

	return &managementv1.ListAgentsResponse{Agents: agents}, nil
}

// listAgentsByServiceID returns a list of Agents filtered by ServiceID.
func (s *ManagementService) listAgentsByServiceID(ctx context.Context, serviceID string) ([]*managementv1.UniversalAgent, error) {
	var agents []*models.Agent
	var service *models.Service

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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

	var res []*managementv1.UniversalAgent

	for _, agent := range agents {
		if IsNodeAgent(agent, service) || IsVMAgent(agent, service) || IsServiceAgent(agent, service) {
			ag, err := s.agentToAPI(agent)
			if err != nil {
				return nil, err
			}
			res = append(res, ag)
		}
	}

	return res, nil
}

// listAgentsByNodeID returns a list of Agents filtered by NodeID.
func (s *ManagementService) listAgentsByNodeID(nodeID string) ([]*managementv1.UniversalAgent, error) {
	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{})
	if err != nil {
		return nil, err
	}

	var res []*managementv1.UniversalAgent

	for _, agent := range agents {
		if pointer.GetString(agent.NodeID) == nodeID || pointer.GetString(agent.RunsOnNodeID) == nodeID {
			ag, err := s.agentToAPI(agent)
			if err != nil {
				return nil, err
			}
			res = append(res, ag)
		}
	}

	return res, nil
}

func (s *ManagementService) agentToAPI(agent *models.Agent) (*managementv1.UniversalAgent, error) {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	ua := &managementv1.UniversalAgent{
		AgentId:                        agent.AgentID,
		AgentType:                      string(agent.AgentType),
		AwsAccessKey:                   agent.AWSOptions.AWSAccessKey,
		CreatedAt:                      timestamppb.New(agent.CreatedAt),
		CustomLabels:                   labels,
		Disabled:                       agent.Disabled,
		DisabledCollectors:             agent.ExporterOptions.DisabledCollectors,
		IsConnected:                    s.r.IsConnected(agent.AgentID),
		IsAgentPasswordSet:             pointer.GetString(agent.AgentPassword) != "",
		IsAwsSecretKeySet:              agent.AWSOptions.AWSAccessKey != "",
		IsPasswordSet:                  pointer.GetString(agent.Password) != "",
		ListenPort:                     uint32(pointer.GetUint16(agent.ListenPort)),
		LogLevel:                       inventoryv1.LogLevelAPIValue(agent.LogLevel),
		MaxQueryLength:                 agent.QANOptions.MaxQueryLength,
		MaxQueryLogSize:                agent.QANOptions.MaxQueryLogSize,
		MetricsPath:                    agent.ExporterOptions.MetricsPath,
		MetricsScheme:                  agent.ExporterOptions.MetricsScheme,
		NodeId:                         pointer.GetString(agent.NodeID),
		PmmAgentId:                     pointer.GetString(agent.PMMAgentID),
		ProcessExecPath:                pointer.GetString(agent.ProcessExecPath),
		PushMetrics:                    agent.ExporterOptions.PushMetrics,
		ExposeExporter:                 agent.ExporterOptions.ExposeExporter,
		QueryExamplesDisabled:          agent.QANOptions.QueryExamplesDisabled,
		CommentsParsingDisabled:        agent.QANOptions.CommentsParsingDisabled,
		RdsBasicMetricsDisabled:        agent.AWSOptions.RDSBasicMetricsDisabled,
		RdsEnhancedMetricsDisabled:     agent.AWSOptions.RDSEnhancedMetricsDisabled,
		RunsOnNodeId:                   pointer.GetString(agent.RunsOnNodeID),
		ServiceId:                      pointer.GetString(agent.ServiceID),
		Status:                         agent.Status,
		TableCount:                     pointer.GetInt32(agent.MySQLOptions.TableCount),
		TableCountTablestatsGroupLimit: agent.MySQLOptions.TableCountTablestatsGroupLimit,
		Tls:                            agent.TLS,
		TlsSkipVerify:                  agent.TLSSkipVerify,
		Username:                       pointer.GetString(agent.Username),
		UpdatedAt:                      timestamppb.New(agent.UpdatedAt),
		Version:                        pointer.GetString(agent.Version),
	}

	if agent.AzureOptions != nil {
		ua.AzureOptions = &managementv1.UniversalAgent_AzureOptions{
			ClientId:          agent.AzureOptions.ClientID,
			IsClientSecretSet: agent.AzureOptions.ClientSecret != "",
			TenantId:          agent.AzureOptions.TenantID,
			SubscriptionId:    agent.AzureOptions.SubscriptionID,
			ResourceGroup:     agent.AzureOptions.ResourceGroup,
		}
	}

	if agent.MySQLOptions != nil {
		ua.MysqlOptions = &managementv1.UniversalAgent_MySQLOptions{
			IsTlsKeySet: agent.MySQLOptions.TLSKey != "",
		}
	}

	if agent.PostgreSQLOptions != nil {
		ua.PostgresqlOptions = &managementv1.UniversalAgent_PostgreSQLOptions{
			IsSslKeySet:            agent.PostgreSQLOptions.SSLKey != "",
			AutoDiscoveryLimit:     agent.PostgreSQLOptions.AutoDiscoveryLimit,
			MaxExporterConnections: agent.PostgreSQLOptions.MaxExporterConnections,
		}
	}

	if agent.MongoDBOptions != nil {
		ua.MongoDbOptions = &managementv1.UniversalAgent_MongoDBOptions{
			AuthenticationMechanism:            agent.MongoDBOptions.AuthenticationMechanism,
			AuthenticationDatabase:             agent.MongoDBOptions.AuthenticationDatabase,
			CollectionsLimit:                   agent.MongoDBOptions.CollectionsLimit,
			EnableAllCollectors:                agent.MongoDBOptions.EnableAllCollectors,
			StatsCollections:                   agent.MongoDBOptions.StatsCollections,
			IsTlsCertificateKeySet:             agent.MongoDBOptions.TLSCertificateKey != "",
			IsTlsCertificateKeyFilePasswordSet: agent.MongoDBOptions.TLSCertificateKeyFilePassword != "",
		}
	}

	return ua, nil
}

func (s *ManagementService) validateListAgentRequest(req *managementv1.ListAgentsRequest) error {
	if req.ServiceId == "" && req.NodeId == "" {
		return status.Error(codes.InvalidArgument, "Either service_id or node_id is expected.")
	}

	if req.ServiceId != "" && req.NodeId != "" {
		return status.Error(codes.InvalidArgument, "Either service_id or node_id is expected, not both.")
	}

	return nil
}

// ListAgentVersions returns a list of agents with their update recommendations (update severity).
func (s *ManagementService) ListAgentVersions(ctx context.Context, _ *managementv1.ListAgentVersionsRequest) (*managementv1.ListAgentVersionsResponse, error) {
	var versions []*managementv1.AgentVersions

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		agentType := models.PMMAgentType

		agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{AgentType: &agentType})
		if err != nil {
			return err
		}

		nodes, err := models.FindNodes(s.db.Querier, models.NodeFilters{})
		if err != nil {
			return err
		}

		nodeNames := make(map[string]*string, len(nodes))
		for _, node := range nodes {
			nodeNames[node.NodeID] = pointer.ToString(node.NodeName)
		}

		serverVersion, err := version.Parse(version.PMMVersion)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not parse the server version: %s", version.PMMVersion))
		}

		for _, agent := range agents {
			if agent.Disabled {
				continue
			}
			nodeName, ok := nodeNames[pointer.GetString(agent.RunsOnNodeID)]
			if !ok {
				s.l.Warnf("node not found for agent %s", agent.AgentID)
				continue
			}

			agentVersion, err := version.Parse(pointer.GetString(agent.Version))
			if err != nil {
				// We don't want to fail the whole request if we can't parse the agent version.
				s.l.Warnf(errors.Wrap(err, fmt.Sprintf("could not parse the client version %s for agent %s", pointer.GetString(agent.Version), agent.AgentID)).Error())
				continue
			}

			var severity managementv1.UpdateSeverity
			switch {
			case agentVersion.Major < serverVersion.Major:
				severity = managementv1.UpdateSeverity_UPDATE_SEVERITY_CRITICAL
			case agentVersion.Less(serverVersion):
				severity = managementv1.UpdateSeverity_UPDATE_SEVERITY_REQUIRED
			case serverVersion.Less(agentVersion):
				severity = managementv1.UpdateSeverity_UPDATE_SEVERITY_UNSUPPORTED
			default:
				severity = managementv1.UpdateSeverity_UPDATE_SEVERITY_UP_TO_DATE
			}

			versions = append(versions, &managementv1.AgentVersions{
				AgentId:  agent.AgentID,
				Version:  pointer.GetString(agent.Version),
				NodeName: pointer.GetString(nodeName),
				Severity: severity,
			})
		}

		return nil
	})

	if errTX != nil {
		return nil, errTX
	}

	return &managementv1.ListAgentVersionsResponse{
		AgentVersions: versions,
	}, nil
}
