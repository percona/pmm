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

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	agentv1beta1 "github.com/percona/pmm/api/managementpb/agent"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/encryption"
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
func (s *AgentService) ListAgents(ctx context.Context, req *agentv1beta1.ListAgentRequest) (*agentv1beta1.ListAgentResponse, error) {
	var err error
	err = s.validateListAgentRequest(req)
	if err != nil {
		return nil, err
	}

	var agents []*agentv1beta1.UniversalAgent

	if req.ServiceId != "" {
		agents, err = s.listAgentsByServiceID(ctx, req.ServiceId)
	} else {
		agents, err = s.listAgentsByNodeID(req.NodeId)
	}
	if err != nil {
		return nil, err
	}

	return &agentv1beta1.ListAgentResponse{Agents: agents}, nil
}

// listAgentsByServiceID returns a list of Agents filtered by ServiceID.
func (s *AgentService) listAgentsByServiceID(ctx context.Context, serviceID string) ([]*agentv1beta1.UniversalAgent, error) {
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

	var res []*agentv1beta1.UniversalAgent

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
func (s *AgentService) listAgentsByNodeID(nodeID string) ([]*agentv1beta1.UniversalAgent, error) {
	agents, err := models.FindAgents(s.db.Querier, models.AgentFilters{})
	if err != nil {
		return nil, err
	}

	var res []*agentv1beta1.UniversalAgent

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

func (s *AgentService) agentToAPI(agent *models.Agent) (*agentv1beta1.UniversalAgent, error) {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	decryptedUsername, err := encryption.Decrypt(pointer.GetString(agent.Username))
	if err != nil {
		logrus.Warningf("Encryption: %v", err)
	}
	decryptedPassword, err := encryption.Decrypt(pointer.GetString(agent.Password))
	if err != nil {
		logrus.Warningf("Encryption: %v", err)
	}
	decryptedAWSAccessKey, err := encryption.Decrypt(pointer.GetString(agent.AWSAccessKey))
	if err != nil {
		logrus.Warningf("Encryption: %v", err)
	}
	decryptedAgentPassword, err := encryption.Decrypt(pointer.GetString(agent.AgentPassword))
	if err != nil {
		logrus.Warningf("Encryption: %v", err)
	}
	decryptedAWSSecretKey, err := encryption.Decrypt(pointer.GetString(agent.AWSSecretKey))
	if err != nil {
		logrus.Warningf("Encryption: %v", err)
	}

	ua := &agentv1beta1.UniversalAgent{
		AgentId:                        agent.AgentID,
		AgentType:                      string(agent.AgentType),
		AwsAccessKey:                   decryptedAWSAccessKey,
		CreatedAt:                      timestamppb.New(agent.CreatedAt),
		CustomLabels:                   labels,
		Disabled:                       agent.Disabled,
		DisabledCollectors:             agent.DisabledCollectors,
		IsConnected:                    s.r.IsConnected(agent.AgentID),
		IsAgentPasswordSet:             decryptedAgentPassword != "",
		IsAwsSecretKeySet:              decryptedAWSSecretKey != "",
		IsPasswordSet:                  decryptedPassword != "",
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
		ExposeExporter:                 agent.ExposeExporter,
		QueryExamplesDisabled:          agent.QueryExamplesDisabled,
		CommentsParsingDisabled:        agent.CommentsParsingDisabled,
		RdsBasicMetricsDisabled:        agent.RDSBasicMetricsDisabled,
		RdsEnhancedMetricsDisabled:     agent.RDSEnhancedMetricsDisabled,
		RunsOnNodeId:                   pointer.GetString(agent.RunsOnNodeID),
		ServiceId:                      pointer.GetString(agent.ServiceID),
		Status:                         agent.Status,
		TableCount:                     pointer.GetInt32(agent.TableCount),
		TableCountTablestatsGroupLimit: agent.TableCountTablestatsGroupLimit,
		Tls:                            agent.TLS,
		TlsSkipVerify:                  agent.TLSSkipVerify,
		Username:                       decryptedUsername,
		UpdatedAt:                      timestamppb.New(agent.UpdatedAt),
		Version:                        pointer.GetString(agent.Version),
	}

	if agent.AzureOptions != nil {
		decryptedClientID, err := encryption.Decrypt(agent.AzureOptions.ClientID)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}
		decryptedClientSecret, err := encryption.Decrypt(agent.AzureOptions.ClientSecret)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}
		decryptedTenantID, err := encryption.Decrypt(agent.AzureOptions.TenantID)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}
		decryptedSubscriptionID, err := encryption.Decrypt(agent.AzureOptions.SubscriptionID)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}

		ua.AzureOptions = &agentv1beta1.UniversalAgent_AzureOptions{
			ClientId:          decryptedClientID,
			IsClientSecretSet: decryptedClientSecret != "",
			TenantId:          decryptedTenantID,
			SubscriptionId:    decryptedSubscriptionID,
			ResourceGroup:     agent.AzureOptions.ResourceGroup,
		}
	}

	if agent.MySQLOptions != nil {
		decryptedTLSKey, err := encryption.Decrypt(agent.MySQLOptions.TLSKey)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}

		ua.MysqlOptions = &agentv1beta1.UniversalAgent_MySQLOptions{
			IsTlsKeySet: decryptedTLSKey != "",
		}
	}

	if agent.PostgreSQLOptions != nil {
		decryptedSSLKey, err := encryption.Decrypt(agent.PostgreSQLOptions.SSLKey)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}

		ua.PostgresqlOptions = &agentv1beta1.UniversalAgent_PostgreSQLOptions{
			IsSslKeySet:            decryptedSSLKey != "",
			AutoDiscoveryLimit:     agent.PostgreSQLOptions.AutoDiscoveryLimit,
			MaxExporterConnections: agent.PostgreSQLOptions.MaxExporterConnections,
		}
	}

	if agent.MongoDBOptions != nil {
		decryptedTLSCertificateKey, err := encryption.Decrypt(agent.MongoDBOptions.TLSCertificateKey)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}
		decryptedTLSCertificateKeyFilePassword, err := encryption.Decrypt(agent.MongoDBOptions.TLSCertificateKeyFilePassword)
		if err != nil {
			logrus.Warningf("Encryption: %v", err)
		}

		ua.MongoDbOptions = &agentv1beta1.UniversalAgent_MongoDBOptions{
			AuthenticationMechanism:            agent.MongoDBOptions.AuthenticationMechanism,
			AuthenticationDatabase:             agent.MongoDBOptions.AuthenticationDatabase,
			CollectionsLimit:                   agent.MongoDBOptions.CollectionsLimit,
			EnableAllCollectors:                agent.MongoDBOptions.EnableAllCollectors,
			StatsCollections:                   agent.MongoDBOptions.StatsCollections,
			IsTlsCertificateKeySet:             decryptedTLSCertificateKey != "",
			IsTlsCertificateKeyFilePasswordSet: decryptedTLSCertificateKeyFilePassword != "",
		}
	}

	return ua, nil
}

func (s *AgentService) validateListAgentRequest(req *agentv1beta1.ListAgentRequest) error {
	if req.ServiceId == "" && req.NodeId == "" {
		return status.Error(codes.InvalidArgument, "Either service_id or node_id is expected.")
	}

	if req.ServiceId != "" && req.NodeId != "" {
		return status.Error(codes.InvalidArgument, "Either service_id or node_id is expected, not both.")
	}

	return nil
}
