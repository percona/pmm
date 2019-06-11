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

// Package services implements business logic of pmm-managed.
package services

import (
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// ToAPINode converts Node database model to API model.
func ToAPINode(node *models.Node) (inventorypb.Node, error) {
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
			CustomLabels: labels,
		}, nil

	case models.RemoteAmazonRDSNodeType:
		return &inventorypb.RemoteAmazonRDSNode{
			NodeId:       node.NodeID,
			NodeName:     node.NodeName,
			Instance:     node.Address,
			Region:       pointer.GetString(node.Region),
			CustomLabels: labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Node type %s", node.NodeType))
	}
}

// ToAPIService converts Service database model to API model.
func ToAPIService(service *models.Service) (inventorypb.Service, error) {
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
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	case models.PostgreSQLServiceType:
		return &inventorypb.PostgreSQLService{
			ServiceId:      service.ServiceID,
			ServiceName:    service.ServiceName,
			NodeId:         service.NodeID,
			Address:        pointer.GetString(service.Address),
			Port:           uint32(pointer.GetUint16(service.Port)),
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
			Environment:    service.Environment,
			Cluster:        service.Cluster,
			ReplicationSet: service.ReplicationSet,
			CustomLabels:   labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Service type %s", service.ServiceType))
	}
}

// ToAPIAgent converts Agent database model to API model.
func ToAPIAgent(q *reform.Querier, agent *models.Agent) (inventorypb.Agent, error) {
	labels, err := agent.GetCustomLabels()
	if err != nil {
		return nil, err
	}

	// agents without services

	switch agent.AgentType {
	case models.PMMAgentType:
		return &inventorypb.PMMAgent{
			AgentId:      agent.AgentID,
			RunsOnNodeId: pointer.GetString(agent.RunsOnNodeID),
			CustomLabels: labels,
		}, nil

	case models.NodeExporterType:
		return &inventorypb.NodeExporter{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:   uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels: labels,
		}, nil
	}

	// agents with exactly one service
	services, err := models.ServicesForAgent(q, agent.AgentID)
	if err != nil {
		return nil, err
	}
	if len(services) != 1 {
		return nil, errors.Errorf("expected exactly one Service, got %d", len(services))
	}
	serviceID := services[0].ServiceID

	switch agent.AgentType {
	case models.MySQLdExporterType:
		return &inventorypb.MySQLdExporter{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:   uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.MongoDBExporterType:
		return &inventorypb.MongoDBExporter{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:   uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.PostgresExporterType:
		return &inventorypb.PostgresExporter{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:   uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels: labels,
		}, nil

	case models.QANMySQLPerfSchemaAgentType:
		return &inventorypb.QANMySQLPerfSchemaAgent{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels: labels,
		}, nil

	case models.QANMySQLSlowlogAgentType:
		return &inventorypb.QANMySQLSlowlogAgent{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels: labels,
		}, nil

	case models.QANMongoDBProfilerAgentType:
		return &inventorypb.QANMongoDBProfilerAgent{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			CustomLabels: labels,
		}, nil

	case models.ProxySQLExporterType:
		return &inventorypb.ProxySQLExporter{
			AgentId:      agent.AgentID,
			PmmAgentId:   pointer.GetString(agent.PMMAgentID),
			ServiceId:    serviceID,
			Username:     pointer.GetString(agent.Username),
			Password:     pointer.GetString(agent.Password),
			Disabled:     agent.Disabled,
			Status:       inventorypb.AgentStatus(inventorypb.AgentStatus_value[agent.Status]),
			ListenPort:   uint32(pointer.GetUint16(agent.ListenPort)),
			CustomLabels: labels,
		}, nil

	default:
		panic(fmt.Errorf("unhandled Agent type %s", agent.AgentType))
	}
}
