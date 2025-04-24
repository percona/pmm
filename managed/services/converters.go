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
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/common"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

// ToAPINode converts Node database model to API model.
func ToAPINode(node *models.Node) (inventoryv1.Node, error) { //nolint:ireturn
	labels, err := node.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch node.NodeType {
	case models.GenericNodeType:
		return &inventoryv1.GenericNode{
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
		return &inventoryv1.ContainerNode{
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
		return &inventoryv1.RemoteNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			NodeModel:    node.NodeModel,
			Region:       pointer.GetString(node.Region),
			Az:           node.AZ,
			CustomLabels: labels,
			Address:      node.Address,
		}, nil

	case models.RemoteRDSNodeType:
		return &inventoryv1.RemoteRDSNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			NodeModel:    node.NodeModel,
			Region:       pointer.GetString(node.Region),
			Az:           node.AZ,
			CustomLabels: labels,
			Address:      node.Address,
		}, nil

	case models.RemoteAzureDatabaseNodeType:
		return &inventoryv1.RemoteAzureDatabaseNode{
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
func ToAPIService(service *models.Service) (inventoryv1.Service, error) { //nolint:ireturn
	labels, err := service.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	switch service.ServiceType {
	case models.MySQLServiceType:
		return &inventoryv1.MySQLService{
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
		return &inventoryv1.MongoDBService{
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
		return &inventoryv1.PostgreSQLService{
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
		return &inventoryv1.ProxySQLService{
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
		return &inventoryv1.HAProxyService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.ExternalServiceType:
		return &inventoryv1.ExternalService{
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
func ToAPIAgent(q *reform.Querier, agent *models.Agent) (inventoryv1.Agent, error) { //nolint:ireturn,maintidx
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
		return &inventoryv1.PMMAgent{
			AgentId:         agent.AgentID,
			RunsOnNodeId:    pointer.GetString(agent.RunsOnNodeID),
			CustomLabels:    labels,
			ProcessExecPath: processExecPath,
		}, nil

	case models.NodeExporterType:
		return &inventoryv1.NodeExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			Disabled:           agent.Disabled,
			Status:             inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			PushMetricsEnabled: agent.ExporterOptions.PushMetrics,
			DisabledCollectors: agent.ExporterOptions.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventoryv1.LogLevelAPIValue(agent.LogLevel),
			ExposeExporter:     agent.ExporterOptions.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}, nil

	case models.MySQLdExporterType:
		return &inventoryv1.MySQLdExporter{
			AgentId:                   agent.AgentID,
			PmmAgentId:                pointer.GetString(agent.PMMAgentID),
			ServiceId:                 serviceID,
			Username:                  pointer.GetString(agent.Username),
			Disabled:                  agent.Disabled,
			Status:                    inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:                uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:              labels,
			Tls:                       agent.TLS,
			TlsSkipVerify:             agent.TLSSkipVerify,
			TablestatsGroupTableLimit: agent.MySQLOptions.TableCountTablestatsGroupLimit,
			TablestatsGroupDisabled:   !agent.IsMySQLTablestatsGroupEnabled(),
			TableCount:                pointer.GetInt32(agent.MySQLOptions.TableCount),
			PushMetricsEnabled:        agent.ExporterOptions.PushMetrics,
			DisabledCollectors:        agent.ExporterOptions.DisabledCollectors,
			ProcessExecPath:           processExecPath,
			LogLevel:                  inventoryv1.LogLevelAPIValue(agent.LogLevel),
			ExposeExporter:            agent.ExporterOptions.ExposeExporter,
			MetricsResolutions:        ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}, nil

	case models.MongoDBExporterType:
		exporter := &inventoryv1.MongoDBExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			ServiceId:          serviceID,
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Status:             inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			Tls:                agent.TLS,
			TlsSkipVerify:      agent.TLSSkipVerify,
			PushMetricsEnabled: agent.ExporterOptions.PushMetrics,
			DisabledCollectors: agent.ExporterOptions.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventoryv1.LogLevelAPIValue(agent.LogLevel),
			ExposeExporter:     agent.ExporterOptions.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}

		exporter.StatsCollections = agent.MongoDBOptions.StatsCollections
		exporter.CollectionsLimit = agent.MongoDBOptions.CollectionsLimit
		exporter.EnableAllCollectors = agent.MongoDBOptions.EnableAllCollectors

		return exporter, nil

	case models.PostgresExporterType:
		exporter := &inventoryv1.PostgresExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			ServiceId:          serviceID,
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Status:             inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			Tls:                agent.TLS,
			TlsSkipVerify:      agent.TLSSkipVerify,
			PushMetricsEnabled: agent.ExporterOptions.PushMetrics,
			DisabledCollectors: agent.ExporterOptions.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventoryv1.LogLevelAPIValue(agent.LogLevel),
			ExposeExporter:     agent.ExporterOptions.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}

		exporter.AutoDiscoveryLimit = pointer.GetInt32(agent.PostgreSQLOptions.AutoDiscoveryLimit)
		exporter.MaxExporterConnections = agent.PostgreSQLOptions.MaxExporterConnections

		return exporter, nil
	case models.QANMySQLPerfSchemaAgentType:
		return &inventoryv1.QANMySQLPerfSchemaAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			MaxQueryLength:         agent.QANOptions.MaxQueryLength,
			QueryExamplesDisabled:  agent.QANOptions.QueryExamplesDisabled,
			DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventoryv1.LogLevelAPIValue(agent.LogLevel),
		}, nil

	case models.QANMySQLSlowlogAgentType:
		return &inventoryv1.QANMySQLSlowlogAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			QueryExamplesDisabled:  agent.QANOptions.QueryExamplesDisabled,
			DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
			MaxSlowlogFileSize:     agent.QANOptions.MaxQueryLogSize,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventoryv1.LogLevelAPIValue(agent.LogLevel),
		}, nil

	case models.QANMongoDBProfilerAgentType:
		return &inventoryv1.QANMongoDBProfilerAgent{
			AgentId:         agent.AgentID,
			PmmAgentId:      pointer.GetString(agent.PMMAgentID),
			ServiceId:       serviceID,
			Username:        pointer.GetString(agent.Username),
			Disabled:        agent.Disabled,
			Status:          inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			CustomLabels:    labels,
			Tls:             agent.TLS,
			TlsSkipVerify:   agent.TLSSkipVerify,
			MaxQueryLength:  agent.QANOptions.MaxQueryLength,
			ProcessExecPath: processExecPath,
			LogLevel:        inventoryv1.LogLevelAPIValue(agent.LogLevel),
			// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-4650
		}, nil

	case models.QANMongoDBMongologAgentType:
		return &inventoryv1.QANMongoDBMongologAgent{
			AgentId:         agent.AgentID,
			PmmAgentId:      pointer.GetString(agent.PMMAgentID),
			ServiceId:       serviceID,
			Username:        pointer.GetString(agent.Username),
			Disabled:        agent.Disabled,
			Status:          inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			CustomLabels:    labels,
			Tls:             agent.TLS,
			TlsSkipVerify:   agent.TLSSkipVerify,
			MaxQueryLength:  agent.QANOptions.MaxQueryLength,
			ProcessExecPath: processExecPath,
			LogLevel:        inventoryv1.LogLevelAPIValue(agent.LogLevel),
			// TODO QueryExamplesDisabled https://jira.percona.com/browse/PMM-4650
		}, nil

	case models.ProxySQLExporterType:
		return &inventoryv1.ProxySQLExporter{
			AgentId:            agent.AgentID,
			PmmAgentId:         pointer.GetString(agent.PMMAgentID),
			ServiceId:          serviceID,
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Status:             inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			Tls:                agent.TLS,
			TlsSkipVerify:      agent.TLSSkipVerify,
			PushMetricsEnabled: agent.ExporterOptions.PushMetrics,
			DisabledCollectors: agent.ExporterOptions.DisabledCollectors,
			ProcessExecPath:    processExecPath,
			LogLevel:           inventoryv1.LogLevelAPIValue(agent.LogLevel),
			ExposeExporter:     agent.ExporterOptions.ExposeExporter,
			MetricsResolutions: ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}, nil

	case models.QANPostgreSQLPgStatementsAgentType:
		return &inventoryv1.QANPostgreSQLPgStatementsAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			MaxQueryLength:         agent.QANOptions.MaxQueryLength,
			DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventoryv1.LogLevelAPIValue(agent.LogLevel),
		}, nil

	case models.QANPostgreSQLPgStatMonitorAgentType:
		return &inventoryv1.QANPostgreSQLPgStatMonitorAgent{
			AgentId:                agent.AgentID,
			PmmAgentId:             pointer.GetString(agent.PMMAgentID),
			ServiceId:              serviceID,
			Username:               pointer.GetString(agent.Username),
			Disabled:               agent.Disabled,
			Status:                 inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			CustomLabels:           labels,
			MaxQueryLength:         agent.QANOptions.MaxQueryLength,
			Tls:                    agent.TLS,
			TlsSkipVerify:          agent.TLSSkipVerify,
			QueryExamplesDisabled:  agent.QANOptions.QueryExamplesDisabled,
			DisableCommentsParsing: agent.QANOptions.CommentsParsingDisabled,
			ProcessExecPath:        processExecPath,
			LogLevel:               inventoryv1.LogLevelAPIValue(agent.LogLevel),
		}, nil

	case models.RDSExporterType:
		return &inventoryv1.RDSExporter{
			AgentId:                 agent.AgentID,
			PmmAgentId:              pointer.GetString(agent.PMMAgentID),
			NodeId:                  nodeID,
			Disabled:                agent.Disabled,
			AwsAccessKey:            agent.AWSOptions.AWSAccessKey,
			Status:                  inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:              uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:            labels,
			BasicMetricsDisabled:    agent.AWSOptions.RDSBasicMetricsDisabled,
			EnhancedMetricsDisabled: agent.AWSOptions.RDSEnhancedMetricsDisabled,
			PushMetricsEnabled:      agent.ExporterOptions.PushMetrics,
			ProcessExecPath:         processExecPath,
			LogLevel:                inventoryv1.LogLevelAPIValue(agent.LogLevel),
			MetricsResolutions:      ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}, nil

	case models.ExternalExporterType:
		if agent.RunsOnNodeID == nil && agent.PMMAgentID != nil {
			pmmAgent, err := models.FindAgentByID(q, *agent.PMMAgentID)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot find pmm_agent by id: %s, for external_exporter id: %s without node_id", *agent.PMMAgentID, agent.AgentID)
			}
			agent.RunsOnNodeID = pmmAgent.RunsOnNodeID
		}
		return &inventoryv1.ExternalExporter{
			AgentId:            agent.AgentID,
			RunsOnNodeId:       pointer.GetString(agent.RunsOnNodeID),
			ServiceId:          pointer.GetString(agent.ServiceID),
			Username:           pointer.GetString(agent.Username),
			Disabled:           agent.Disabled,
			Scheme:             agent.ExporterOptions.MetricsScheme,
			MetricsPath:        agent.ExporterOptions.MetricsPath,
			ListenPort:         uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:       labels,
			PushMetricsEnabled: agent.ExporterOptions.PushMetrics,
			ProcessExecPath:    processExecPath,
			MetricsResolutions: ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}, nil

	case models.AzureDatabaseExporterType:
		return &inventoryv1.AzureDatabaseExporter{
			AgentId:                     agent.AgentID,
			PmmAgentId:                  pointer.GetString(agent.PMMAgentID),
			NodeId:                      nodeID,
			Disabled:                    agent.Disabled,
			AzureDatabaseSubscriptionId: agent.AzureOptions.SubscriptionID,
			Status:                      inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ListenPort:                  uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels:                labels,
			ProcessExecPath:             processExecPath,
			LogLevel:                    inventoryv1.LogLevelAPIValue(agent.LogLevel),
			MetricsResolutions:          ConvertMetricsResolutions(agent.ExporterOptions.MetricsResolutions),
		}, nil

	case models.VMAgentType:
		return &inventoryv1.VMAgent{
			AgentId:         agent.AgentID,
			PmmAgentId:      pointer.GetString(agent.PMMAgentID),
			Status:          inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
			ProcessExecPath: processExecPath,
			ListenPort:      uint32(pointer.GetUint16(agent.ListenPort)),
		}, nil

	case models.NomadAgentType:
		return &inventoryv1.NomadAgent{
			AgentId:         agent.AgentID,
			PmmAgentId:      pointer.GetString(agent.PMMAgentID),
			Disabled:        agent.Disabled,
			Status:          inventoryv1.AgentStatus(inventoryv1.AgentStatus_value[agent.Status]),
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
func SpecifyLogLevel(variant, minLogLevel inventoryv1.LogLevel) string {
	if variant == inventoryv1.LogLevel_LOG_LEVEL_UNSPECIFIED {
		return ""
	}

	// downgrade instead of return API error
	if variant < minLogLevel {
		variant = minLogLevel
	}

	return strings.ToLower(strings.TrimPrefix(variant.String(), "LOG_LEVEL_"))
}

// nodeTypes maps protobuf types to their string types.
var nodeTypes = map[inventoryv1.NodeType]models.NodeType{
	inventoryv1.NodeType_NODE_TYPE_GENERIC_NODE:               models.GenericNodeType,
	inventoryv1.NodeType_NODE_TYPE_CONTAINER_NODE:             models.ContainerNodeType,
	inventoryv1.NodeType_NODE_TYPE_REMOTE_NODE:                models.RemoteNodeType,
	inventoryv1.NodeType_NODE_TYPE_REMOTE_RDS_NODE:            models.RemoteRDSNodeType,
	inventoryv1.NodeType_NODE_TYPE_REMOTE_AZURE_DATABASE_NODE: models.RemoteAzureDatabaseNodeType,
}

// ProtoToModelNodeType converts a NodeType from protobuf to model.
func ProtoToModelNodeType(nodeType inventoryv1.NodeType) *models.NodeType {
	if nodeType == inventoryv1.NodeType_NODE_TYPE_UNSPECIFIED {
		return nil
	}
	result := nodeTypes[nodeType]
	return &result
}

// ServiceTypes maps protobuf types to their string types.
var ServiceTypes = map[inventoryv1.ServiceType]models.ServiceType{
	inventoryv1.ServiceType_SERVICE_TYPE_MYSQL_SERVICE:      models.MySQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_MONGODB_SERVICE:    models.MongoDBServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_POSTGRESQL_SERVICE: models.PostgreSQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_PROXYSQL_SERVICE:   models.ProxySQLServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_HAPROXY_SERVICE:    models.HAProxyServiceType,
	inventoryv1.ServiceType_SERVICE_TYPE_EXTERNAL_SERVICE:   models.ExternalServiceType,
}

// ProtoToModelServiceType converts a ServiceType from protobuf to model.
func ProtoToModelServiceType(serviceType inventoryv1.ServiceType) *models.ServiceType {
	if serviceType == inventoryv1.ServiceType_SERVICE_TYPE_UNSPECIFIED {
		return nil
	}
	result := ServiceTypes[serviceType]
	return &result
}
