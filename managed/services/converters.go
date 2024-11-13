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

// Package services implements business logic of pmm-managed.
package services

import (
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/common"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/managed/models"
)

// ToAPINode converts Node database model to API model.
func ToAPINode(node *models.Node) (inventorypb.Node, error) { //nolint:ireturn
	labels, err := node.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch node.NodeType {
	case models.GenericNodeType:
		return &inventorypb.GenericNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			MachineId:    pointer.GetString(node.MachineID),
			Distro:       node.Distro,
			NodeModel:    node.NodeModel,
			Region:       pointer.GetString(node.Region),
			Az:           node.AZ,
			CustomLabels: labels,
			Address:      node.Address,
		}, nil

	case models.ContainerNodeType:
		return &inventorypb.ContainerNode{
			NodeId:        node.NodeID,
			NodeName:      node.NodeName,
			MachineId:     pointer.GetString(node.MachineID),
			ContainerId:   pointer.GetString(node.ContainerID),
			ContainerName: pointer.GetString(node.ContainerName),
			NodeModel:     node.NodeModel,
			Region:        pointer.GetString(node.Region),
			Az:            node.AZ,
			CustomLabels:  labels,
			Address:       node.Address,
		}, nil

	case models.RemoteNodeType:
		return &inventorypb.RemoteNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			NodeModel:    node.NodeModel,
			Region:       pointer.GetString(node.Region),
			Az:           node.AZ,
			CustomLabels: labels,
			Address:      node.Address,
		}, nil

	case models.RemoteRDSNodeType:
		return &inventorypb.RemoteRDSNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			NodeModel:    node.NodeModel,
			Region:       pointer.GetString(node.Region),
			Az:           node.AZ,
			CustomLabels: labels,
			Address:      node.Address,
		}, nil

	case models.RemoteAzureDatabaseNodeType:
		return &inventorypb.RemoteAzureDatabaseNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			NodeModel:    node.NodeModel,
			Region:       pointer.GetString(node.Region),
			Az:           node.AZ,
			CustomLabels: labels,
			Address:      node.Address,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Node type %s", node.NodeType))
	}
}

// ToAPIService converts Service database model to API model.
func ToAPIService(service *models.Service) (inventorypb.Service, error) { //nolint:ireturn
	labels, err := service.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch service.ServiceType {
	case models.MySQLServiceType:
		return &inventorypb.MySQLService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Address:        pointer.GetString(service.Address),
			Port:           uint32(pointer.GetUint16(service.Port)),
			Socket:         pointer.GetString(service.Socket),
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.MongoDBServiceType:
		return &inventorypb.MongoDBService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Address:        pointer.GetString(service.Address),
			Port:           uint32(pointer.GetUint16(service.Port)),
			Socket:         pointer.GetString(service.Socket),
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.PostgreSQLServiceType:
		return &inventorypb.PostgreSQLService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			DatabaseName:   service.DatabaseName,
			NodeId:         service.NodeID,
			Address:        pointer.GetString(service.Address),
			Port:           uint32(pointer.GetUint16(service.Port)),
			Socket:         pointer.GetString(service.Socket),
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.ProxySQLServiceType:
		return &inventorypb.ProxySQLService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Address:        pointer.GetString(service.Address),
			Port:           uint32(pointer.GetUint16(service.Port)),
			Socket:         pointer.GetString(service.Socket),
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.HAProxyServiceType:
		return &inventorypb.HAProxyService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.ExternalServiceType:
		return &inventorypb.ExternalService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
			Group:          service.ExternalGroup,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Service type %s", service.ServiceType))
	}
}

// ToAPIAgent converts Agent database model to API model.
func ToAPIAgent(q *reform.Querier, agent *models.Agent) (inventorypb.Agent, error) { //nolint:ireturn,maintidx
	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	var nodeID, serviceID string
	if agent.NodeID != nil {
		node, err := models.FindNodeByID(q, *agent.NodeID)
		if err != nil {
			return nil, err
		}
		nodeID = node.NodeID
	}
	if agent.ServiceID != nil {
		service, err := models.FindServiceByID(q, *agent.ServiceID)
		if err != nil {
			return nil, err
		}
		serviceID = service.ServiceID
	}
	processExecPath := pointer.GetString(agent.ProcessExecPath)
	switch agent.AgentType {
	case models.PMMAgentType:
		return &inventorypb.PMMAgent{
			AgentId:         agent.AgentID,
			RunsOnNodeId:    pointer.GetString(agent.RunsOnNodeID),
			CustomLabels:    labels,
			ProcessExecPath: processExecPath,
		}, nil

	case models.NodeExporterType:
		return &inventorypb.NodeExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			Disabled:           agent.Disabled,
			Status:             inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			PushMetricsEnabled: agent.PushMetrics,
			DisabledCollectors: agent.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			ExposeExporter:     agent.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.MetricsResolutions),
		}, nil

	case models.MySQLdExporterType:
		return &inventorypb.MySQLdExporter{
			AgentId:                   agent.AgentID,
			PmmAgentId:                pointer.GetString(agent.PMMAgentID),
			ServiceId:                 serviceID,
			Username:                  pointer.GetString(agent.Username),
			Disabled:                  agent.Disabled,
			Status:                    inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:                uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:              labels,
			Tls:                       agent.TLS,
			TlsSkipVerify:             agent.TLSSkipVerify,
			TablestatsGroupTableLimit: agent.TableCountTablestatsGroupLimit,
			TablestatsGroupDisabled:   !agent.IsMySQLTablestatsGroupEnabled(),
			PushMetricsEnabled:        agent.PushMetrics,
			DisabledCollectors:        agent.DisabledCollectors,
			ProcessExecPath:           processExecPath,
			LogLevel:                  inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			ExposeExporter:            agent.ExposeExporter,
			MetricsResolutions:        ConvertMetricsResolutions(agent.MetricsResolutions),
		}, nil

	case models.MongoDBExporterType:
		exporter := &inventorypb.MongoDBExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			ServiceId:          serviceID,
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Status:             inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			Tls:                agent.TLS,
			TlsSkipVerify:      agent.TLSSkipVerify,
			PushMetricsEnabled: agent.PushMetrics,
			DisabledCollectors: agent.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			ExposeExporter:     agent.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.MetricsResolutions),
		}
		if agent.MongoDBOptions != nil {
			exporter.StatsCollections = agent.MongoDBOptions.StatsCollections
			exporter.CollectionsLimit = agent.MongoDBOptions.CollectionsLimit
			exporter.EnableAllCollectors = agent.MongoDBOptions.EnableAllCollectors
		}
		return exporter, nil

	case models.PostgresExporterType:
		exporter := &inventorypb.PostgresExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			ServiceId:          serviceID,
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Status:             inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			Tls:                agent.TLS,
			TlsSkipVerify:      agent.TLSSkipVerify,
			PushMetricsEnabled: agent.PushMetrics,
			DisabledCollectors: agent.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			ExposeExporter:     agent.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.MetricsResolutions),
		}
		if agent.PostgreSQLOptions != nil {
			exporter.AutoDiscoveryLimit = agent.PostgreSQLOptions.AutoDiscoveryLimit
			exporter.MaxExporterConnections = agent.PostgreSQLOptions.MaxExporterConnections
		}
		return exporter, nil
	case models.QANMySQLPerfSchemaAgentType:
		return &inventorypb.QANMySQLPerfSchemaAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			MaxQueryLength:         agent.MaxQueryLength,
			QueryExamplesDisabled:  agent.QueryExamplesDisabled,
			DisableCommentsParsing: agent.CommentsParsingDisabled,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
		}, nil

	case models.QANMySQLSlowlogAgentType:
		return &inventorypb.QANMySQLSlowlogAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			QueryExamplesDisabled:  agent.QueryExamplesDisabled,
			DisableCommentsParsing: agent.CommentsParsingDisabled,
			MaxSlowlogFileSize:     agent.MaxQueryLogSize,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
		}, nil

	case models.QANMongoDBProfilerAgentType:
		return &inventorypb.QANMongoDBProfilerAgent{
			AgentId:         agent.AgentID,
			PmmAgentId:      pointer.GetString(agent.PMMAgentID),
			ServiceId:       serviceID,
			Username:        pointer.GetString(agent.Username),
			Disabled:        agent.Disabled,
			Status:          inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels:    labels,
			Tls:             agent.TLS,
			TlsSkipVerify:   agent.TLSSkipVerify,
			MaxQueryLength:  agent.MaxQueryLength,
			ProcessExecPath: processExecPath,
			LogLevel:        inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-4650
		}, nil

	case models.ProxySQLExporterType:
		return &inventorypb.ProxySQLExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			ServiceId:          serviceID,
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Status:             inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			Tls:                agent.TLS,
			TlsSkipVerify:      agent.TLSSkipVerify,
			PushMetricsEnabled: agent.PushMetrics,
			DisabledCollectors: agent.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			ExposeExporter:     agent.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.MetricsResolutions),
		}, nil

	case models.QANPostgreSQLPgStatementsAgentType:
		return &inventorypb.QANPostgreSQLPgStatementsAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			MaxQueryLength:         agent.MaxQueryLength,
			DisableCommentsParsing: agent.CommentsParsingDisabled,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
		}, nil

	case models.QANPostgreSQLPgStatMonitorAgentType:
		return &inventorypb.QANPostgreSQLPgStatMonitorAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			MaxQueryLength:         agent.MaxQueryLength,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			QueryExamplesDisabled:  agent.QueryExamplesDisabled,
			DisableCommentsParsing: agent.CommentsParsingDisabled,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
		}, nil

	case models.RDSExporterType:
		return &inventorypb.RDSExporter{
			AgentId:                 agent.AgentID,
			PmmAgentId:              pointer.GetString(agent.PMMAgentID),
			NodeId:                  nodeID,
			Disabled:                agent.Disabled,
			AwsAccessKey:            pointer.GetString(agent.AWSAccessKey),
			Status:                  inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:              uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:            labels,
			BasicMetricsDisabled:    agent.RDSBasicMetricsDisabled,
			EnhancedMetricsDisabled: agent.RDSEnhancedMetricsDisabled,
			PushMetricsEnabled:      agent.PushMetrics,
			ProcessExecPath:         processExecPath,
			LogLevel:                inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			MetricsResolutions:      ConvertMetricsResolutions(agent.MetricsResolutions),
		}, nil

	case models.ExternalExporterType:
		if agent.RunsOnNodeID == nil && agent.PMMAgentID != nil {
			pmmAgent, err := models.FindAgentByID(q, *agent.PMMAgentID)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot find pmm_agent by id: %s, for external_exporter id: %s without node_id", *agent.PMMAgentID, agent.AgentID)
			}
			agent.RunsOnNodeID = pmmAgent.RunsOnNodeID
		}
		return &inventorypb.ExternalExporter{
			AgentId:            agent.AgentID,
			RunsOnNodeId:       pointer.GetString(agent.RunsOnNodeID),
			ServiceId:          pointer.GetString(agent.ServiceID),
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Scheme:             pointer.GetString(agent.MetricsScheme),
			MetricsPath:        pointer.GetString(agent.MetricsPath),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			PushMetricsEnabled: agent.PushMetrics,
			ProcessExecPath:    processExecPath,
			MetricsResolutions: ConvertMetricsResolutions(agent.MetricsResolutions),
		}, nil

	case models.AzureDatabaseExporterType:
		return &inventorypb.AzureDatabaseExporter{
			AgentId:                     agent.AgentID,
			PmmAgentId:                  pointer.GetString(agent.PMMAgentID),
			NodeId:                      nodeID,
			Disabled:                    agent.Disabled,
			AzureDatabaseSubscriptionId: agent.AzureOptions.SubscriptionID,
			Status:                      inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:                  uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:                labels,
			ProcessExecPath:             processExecPath,
			LogLevel:                    inventorypb.LogLevel(inventorypb.LogLevel_value[pointer.GetString(agent.LogLevel)]),
			MetricsResolutions:          ConvertMetricsResolutions(agent.MetricsResolutions),
		}, nil

	case models.VMAgentType:
		return &inventorypb.VMAgent{
			AgentId:         agent.AgentID,
			PmmAgentId:      pointer.GetString(agent.PMMAgentID),
			Status:          inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ProcessExecPath: processExecPath,
			ListenPort:      uint32(pointer.GetUint16(agent.ListenPort)),
		}, nil

	default:
		panic(fmt.Errorf("unhandled Agent type %s", agent.AgentType))
	}
}

// ConvertMetricsResolutions converts MetricsResolutions from model to API.
func ConvertMetricsResolutions(resolutions *models.MetricsResolutions) *common.MetricsResolutions {
	if resolutions == nil {
		return nil
	}
	var res common.MetricsResolutions
	if resolutions.HR != 0 {
		res.Hr = durationpb.New(resolutions.HR)
	}
	if resolutions.MR != 0 {
		res.Mr = durationpb.New(resolutions.MR)
	}
	if resolutions.LR != 0 {
		res.Lr = durationpb.New(resolutions.LR)
	}
	return &res
}

// SpecifyLogLevel - convert proto enum to string
// mysqld_exporter, node_exporter and postgres_exporter don't support --log.level=fatal.
func SpecifyLogLevel(variant, minLogLevel inventorypb.LogLevel) string {
	if variant == inventorypb.LogLevel_auto {
		return ""
	}

	// downgrade instead of return API error
	if variant < minLogLevel {
		return minLogLevel.String()
	}

	return variant.String()
}

// nodeTypes maps protobuf types to their string types.
var nodeTypes = map[inventorypb.NodeType]models.NodeType{
	inventorypb.NodeType_GENERIC_NODE:               models.GenericNodeType,
	inventorypb.NodeType_CONTAINER_NODE:             models.ContainerNodeType,
	inventorypb.NodeType_REMOTE_NODE:                models.RemoteNodeType,
	inventorypb.NodeType_REMOTE_RDS_NODE:            models.RemoteRDSNodeType,
	inventorypb.NodeType_REMOTE_AZURE_DATABASE_NODE: models.RemoteAzureDatabaseNodeType,
}

// ProtoToModelNodeType converts a NodeType from protobuf to model.
func ProtoToModelNodeType(nodeType inventorypb.NodeType) *models.NodeType {
	if nodeType == inventorypb.NodeType_NODE_TYPE_INVALID {
		return nil
	}
	result := nodeTypes[nodeType]
	return &result
}

// ServiceTypes maps protobuf types to their string types.
var ServiceTypes = map[inventorypb.ServiceType]models.ServiceType{
	inventorypb.ServiceType_MYSQL_SERVICE:      models.MySQLServiceType,
	inventorypb.ServiceType_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventorypb.ServiceType_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventorypb.ServiceType_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventorypb.ServiceType_HAPROXY_SERVICE:    models.HAProxyServiceType,
	inventorypb.ServiceType_EXTERNAL_SERVICE:   models.ExternalServiceType,
}

// ProtoToModelServiceType converts a ServiceType from protobuf to model.
func ProtoToModelServiceType(serviceType inventorypb.ServiceType) *models.ServiceType {
	if serviceType == inventorypb.ServiceType_SERVICE_TYPE_INVALID {
		return nil
	}
	result := ServiceTypes[serviceType]
	return &result
}
