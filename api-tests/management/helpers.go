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

// Package management contains management API tests.
package management

import (
	"context"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// AgentStatusUnknown means agent is not connected and we don't know anything about its status.
var AgentStatusUnknown = inventoryv1.AgentStatus_name[int32(inventoryv1.AgentStatus_AGENT_STATUS_UNKNOWN)]

// RegisterGenericNode registers a generic node using the provided parameters.
func RegisterGenericNode(t pmmapitests.TestingT, body mservice.RegisterNodeBody) (string, string) {
	t.Helper()
	params := mservice.RegisterNodeParams{
		Context: pmmapitests.Context,
		Body:    body,
	}
	registerOK, err := client.Default.ManagementService.RegisterNode(&params)
	require.NoError(t, err)
	require.NotNil(t, registerOK)
	require.NotNil(t, registerOK.Payload.PMMAgent)
	require.NotNil(t, registerOK.Payload.PMMAgent.AgentID)
	require.NotNil(t, registerOK.Payload.GenericNode)
	require.NotNil(t, registerOK.Payload.GenericNode.NodeID)

	return registerOK.Payload.GenericNode.NodeID, registerOK.Payload.PMMAgent.AgentID
}

func registerContainerNode(t pmmapitests.TestingT, body mservice.RegisterNodeBody) (string, string) {
	t.Helper()

	params := mservice.RegisterNodeParams{
		Context: pmmapitests.Context,
		Body:    body,
	}
	registerOK, err := client.Default.ManagementService.RegisterNode(&params)
	require.NoError(t, err)
	require.NotNil(t, registerOK)
	require.NotNil(t, registerOK.Payload.PMMAgent)
	require.NotNil(t, registerOK.Payload.PMMAgent.AgentID)
	require.NotNil(t, registerOK.Payload.ContainerNode)
	require.NotNil(t, registerOK.Payload.ContainerNode.NodeID)

	return registerOK.Payload.ContainerNode.NodeID, registerOK.Payload.PMMAgent.AgentID
}

func assertNodeExporterCreated(t pmmapitests.TestingT, pmmAgentID string) (string, bool) {
	t.Helper()

	listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		PMMAgentID: pointer.ToString(pmmAgentID),
		Context:    pmmapitests.Context,
	})
	assert.NoError(t, err)
	require.Len(t, listAgentsOK.Payload.NodeExporter, 1)
	nodeExporterAgentID := listAgentsOK.Payload.NodeExporter[0].AgentID
	asserted := assert.Equal(t, agents.ListAgentsOKBodyNodeExporterItems0{
		PMMAgentID:         pmmAgentID,
		AgentID:            nodeExporterAgentID,
		PushMetricsEnabled: true,
		Status:             &AgentStatusUnknown,
		CustomLabels:       make(map[string]string),
		DisabledCollectors: make([]string, 0),
		LogLevel:           pointer.ToString("LOG_LEVEL_UNSPECIFIED"),
	}, *listAgentsOK.Payload.NodeExporter[0])
	return nodeExporterAgentID, asserted
}

func assertPMMAgentCreated(t pmmapitests.TestingT, nodeID string, pmmAgentID string) {
	t.Helper()

	agentOK, err := inventoryClient.Default.AgentsService.GetAgent(&agents.GetAgentParams{
		AgentID: pmmAgentID,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	assert.Equal(t, agents.GetAgentOKBody{
		PMMAgent: &agents.GetAgentOKBodyPMMAgent{
			AgentID:      pmmAgentID,
			RunsOnNodeID: nodeID,
			CustomLabels: map[string]string{},
		},
	}, *agentOK.Payload)
}

func assertNodeCreated(t pmmapitests.TestingT, nodeID string, expectedResult nodes.GetNodeOKBody) {
	t.Helper()

	nodeOK, err := inventoryClient.Default.NodesService.GetNode(&nodes.GetNodeParams{
		NodeID:  nodeID,
		Context: pmmapitests.Context,
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, *nodeOK.Payload)
}

// RemovePMMAgentWithSubAgents removes a PMM Agent along with its sub-agents.
func RemovePMMAgentWithSubAgents(t pmmapitests.TestingT, pmmAgentID string) {
	t.Helper()

	listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		PMMAgentID: pointer.ToString(pmmAgentID),
		Context:    context.Background(),
	})
	assert.NoError(t, err)
	removeAllAgentsInList(t, listAgentsOK)
	pmmapitests.RemoveAgents(t, pmmAgentID)
}

func removeServiceAgents(t pmmapitests.TestingT, serviceID string) {
	t.Helper()

	listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		ServiceID: pointer.ToString(serviceID),
		Context:   context.Background(),
	})
	assert.NoError(t, err)
	removeAllAgentsInList(t, listAgentsOK)
}

func removeAllAgentsInList(t pmmapitests.TestingT, listAgentsOK *agents.ListAgentsOK) {
	t.Helper()

	require.NotNil(t, listAgentsOK)
	require.NotNil(t, listAgentsOK.Payload)

	var agentIDs []string //nolint:prealloc
	for _, agent := range listAgentsOK.Payload.NodeExporter {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.PMMAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.PostgresExporter {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.MysqldExporter {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.ProxysqlExporter {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.QANMysqlPerfschemaAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.MongodbExporter {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.QANMongodbProfilerAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.QANMysqlSlowlogAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.QANPostgresqlPgstatementsAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.ExternalExporter {
		agentIDs = append(agentIDs, agent.AgentID)
	}
	for _, agent := range listAgentsOK.Payload.VMAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}

	pmmapitests.RemoveAgents(t, agentIDs...)
}
