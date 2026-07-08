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
	"testing"

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

// RegisterNode registers a node using the provided parameters.
// It returns the registered node's ID and the associated PMM Agent's ID.
// The function also ensures that the registered node is cleaned up after the test by using t.Cleanup.
func RegisterNode(t *testing.T, body mservice.RegisterNodeBody) (string, string) {
	t.Helper()

	require.NotNil(t, body.NodeType)
	params := &mservice.RegisterNodeParams{
		Context: t.Context(),
		Body:    body,
	}

	registerOK, err := client.Default.ManagementService.RegisterNode(params)
	require.NoError(t, err)
	require.NotNil(t, registerOK)
	require.NotNil(t, registerOK.Payload)
	require.NotNil(t, registerOK.Payload.PMMAgent)
	require.NotEmpty(t, registerOK.Payload.PMMAgent.AgentID)

	var nodeID string
	switch *body.NodeType {
	case mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE:
		require.NotNil(t, registerOK.Payload.GenericNode)
		nodeID = registerOK.Payload.GenericNode.NodeID
	case mservice.RegisterNodeBodyNodeTypeNODETYPECONTAINERNODE:
		require.NotNil(t, registerOK.Payload.ContainerNode)
		nodeID = registerOK.Payload.ContainerNode.NodeID
	}
	require.NotEmpty(t, nodeID)
	t.Cleanup(func() {
		pmmapitests.UnregisterNodes(t, nodeID)
	})

	return nodeID, registerOK.Payload.PMMAgent.AgentID
}

func assertNodeExporterCreated(t *testing.T, pmmAgentID string) (string, bool) {
	t.Helper()

	listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		PMMAgentID: new(pmmAgentID),
		Context:    pmmapitests.Context,
	})
	require.NoError(t, err)
	require.Len(t, listAgentsOK.Payload.NodeExporter, 1)
	nodeExporterAgentID := listAgentsOK.Payload.NodeExporter[0].AgentID
	asserted := assert.Equal(t, agents.ListAgentsOKBodyNodeExporterItems0{
		PMMAgentID:         pmmAgentID,
		AgentID:            nodeExporterAgentID,
		PushMetricsEnabled: true,
		Status:             &AgentStatusUnknown,
		CustomLabels:       make(map[string]string),
		DisabledCollectors: make([]string, 0),
		LogLevel:           new("LOG_LEVEL_UNSPECIFIED"),
	}, *listAgentsOK.Payload.NodeExporter[0])
	return nodeExporterAgentID, asserted
}

func assertPMMAgentCreated(t *testing.T, nodeID string, pmmAgentID string) {
	t.Helper()

	agentOK, err := inventoryClient.Default.AgentsService.GetAgent(&agents.GetAgentParams{
		AgentID: pmmAgentID,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, agents.GetAgentOKBody{
		PMMAgent: &agents.GetAgentOKBodyPMMAgent{
			AgentID:      pmmAgentID,
			RunsOnNodeID: nodeID,
			CustomLabels: map[string]string{},
		},
	}, *agentOK.Payload)
}

func assertNodeCreated(t *testing.T, nodeID string, expectedResult nodes.GetNodeOKBody) {
	t.Helper()

	nodeOK, err := inventoryClient.Default.NodesService.GetNode(&nodes.GetNodeParams{
		NodeID:  nodeID,
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.Equal(t, expectedResult, *nodeOK.Payload)
}

// RemovePMMAgentWithSubAgents removes a PMM Agent along with its sub-agents.
func RemovePMMAgentWithSubAgents(t *testing.T, pmmAgentID string) {
	t.Helper()

	listAgentsOK, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		PMMAgentID: new(pmmAgentID),
		Context:    context.Background(),
	})
	if err == nil {
		removeAllAgentsInList(t, listAgentsOK)
		pmmapitests.RemoveAgents(t, pmmAgentID)
	}
}

func removeAllAgentsInList(t *testing.T, listAgentsOK *agents.ListAgentsOK) {
	t.Helper()

	require.NotNil(t, listAgentsOK)
	require.NotNil(t, listAgentsOK.Payload)

	agentIDs := make([]string, 0,
		len(listAgentsOK.Payload.NodeExporter)+
			len(listAgentsOK.Payload.PMMAgent)+
			len(listAgentsOK.Payload.PostgresExporter)+
			len(listAgentsOK.Payload.MysqldExporter)+
			len(listAgentsOK.Payload.ProxysqlExporter)+
			len(listAgentsOK.Payload.QANMysqlPerfschemaAgent)+
			len(listAgentsOK.Payload.MongodbExporter)+
			len(listAgentsOK.Payload.QANMongodbProfilerAgent)+
			len(listAgentsOK.Payload.QANMongodbMongologAgent)+
			len(listAgentsOK.Payload.QANMysqlSlowlogAgent)+
			len(listAgentsOK.Payload.QANPostgresqlPgstatementsAgent)+
			len(listAgentsOK.Payload.ExternalExporter)+
			len(listAgentsOK.Payload.VMAgent)+
			len(listAgentsOK.Payload.RtaMongodbAgent))
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
	for _, agent := range listAgentsOK.Payload.QANMongodbMongologAgent {
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
	for _, agent := range listAgentsOK.Payload.RtaMongodbAgent {
		agentIDs = append(agentIDs, agent.AgentID)
	}

	pmmapitests.RemoveAgents(t, agentIDs...)
}
