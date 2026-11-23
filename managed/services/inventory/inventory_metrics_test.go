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

package inventory

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestNewInventoryMetricsCollector(t *testing.T) {
	t.Run("Metrics returns inventory metrics", func(t *testing.T) {
		client := http.Client{}

		ctx, cancelCtx := context.WithTimeout(t.Context(), 3*time.Second)
		defer cancelCtx()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:7773/debug/metrics", nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, resp.Body.Close())
		})

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotEmpty(t, body)

		assert.Contains(t, string(body), "TYPE pmm_managed_inventory_agents gauge")
		assert.Contains(t, string(body), "TYPE pmm_managed_inventory_nodes gauge")
		assert.Contains(t, string(body), "TYPE pmm_managed_inventory_services gauge")
	})

	t.Run("Collector", func(t *testing.T) {
		metricsMock := &mockInventoryMetrics{}
		metricsMock.Test(t)

		t.Cleanup(func() { metricsMock.AssertExpectations(t) })

		inventoryCollector := NewInventoryMetricsCollector(metricsMock)

		agentMetrics := []Metric{
			{
				labels: []string{"A1", string(models.PMMAgentType), "S1", "SN1", "N1", "NN1", "ENV1", "CL1", "PA1", strconv.Itoa(1), "V1"},
				value:  float64(1),
			},
		}

		nodeMetrics := []Metric{
			{
				labels: []string{"N1", string(models.GenericNodeType), "N1", "C1"},
				value:  float64(1),
			},
		}

		serviceMetrics := []Metric{
			{
				labels: []string{"C1", string(models.ProxySQLServiceType), "N1"},
				value:  float64(1),
			},
		}

		metricsMock.On("GetAgentMetrics", mock.Anything).Return(agentMetrics, nil)
		metricsMock.On("GetNodeMetrics", mock.Anything).Return(nodeMetrics, nil)
		metricsMock.On("GetServiceMetrics", mock.Anything).Return(serviceMetrics, nil)

		const expectedAgentMetrics = `
			# HELP pmm_managed_inventory_agents Inventory Agent
			# TYPE pmm_managed_inventory_agents gauge
			pmm_managed_inventory_agents{agent_id="A1",agent_type="pmm-agent",cluster="CL1",disabled="1",environment="ENV1",node_id="N1",node_name="NN1",pmm_agent_id="PA1",service_id="S1",service_name="SN1",version="V1"} 1
		`

		const expectedNodeMetrics = `
			# HELP pmm_managed_inventory_nodes Inventory Node
			# TYPE pmm_managed_inventory_nodes gauge
			pmm_managed_inventory_nodes{container_name="C1",node_id="N1",node_name="N1",node_type="generic"} 1
		`

		const expectedServiceMetrics = `
			# HELP pmm_managed_inventory_services Inventory Service
			# TYPE pmm_managed_inventory_services gauge
			pmm_managed_inventory_services{node_id="N1",service_id="C1",service_type="proxysql"} 1
		`

		err := testutil.CollectAndCompare(
			inventoryCollector,
			strings.NewReader(expectedAgentMetrics),
			"pmm_managed_inventory_agents",
		)
		if err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		err = testutil.CollectAndCompare(
			inventoryCollector,
			strings.NewReader(expectedNodeMetrics),
			"pmm_managed_inventory_nodes",
		)
		if err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		err = testutil.CollectAndCompare(
			inventoryCollector,
			strings.NewReader(expectedServiceMetrics),
			"pmm_managed_inventory_services",
		)
		if err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}
	})
}

func TestAgentMetric(t *testing.T) {
	t.Parallel()

	// A remotely monitored service: it belongs to one node while its exporter runs on another.
	serviceNode := &models.Node{
		NodeID:       "N-remote",
		NodeName:     "remote-node",
		CustomLabels: []byte(`{"environment": "service-node-env", "cluster": "service-node-cluster"}`),
	}
	runsOnNode := &models.Node{
		NodeID:       "N-server",
		NodeName:     "server-node",
		CustomLabels: []byte(`{"environment": "runs-on-env", "cluster": "runs-on-cluster"}`),
	}
	service := &models.Service{ServiceID: "S1", ServiceName: "mongos-1", NodeID: serviceNode.NodeID}

	pmmAgent := &models.Agent{AgentID: "PA1", AgentType: models.PMMAgentType, RunsOnNodeID: new(runsOnNode.NodeID), Version: new("3.11.0")}
	exporter := &models.Agent{
		AgentID:    "A1",
		AgentType:  models.MongoDBExporterType,
		PMMAgentID: new(pmmAgent.AgentID),
		ServiceID:  new(service.ServiceID),
		Status:     "AGENT_STATUS_RUNNING",
	}
	nodeExporter := &models.Agent{
		AgentID:    "A2",
		AgentType:  models.NodeExporterType,
		PMMAgentID: new(pmmAgent.AgentID),
		NodeID:     new(runsOnNode.NodeID),
		Disabled:   true,
	}

	agents := []*models.Agent{pmmAgent, exporter, nodeExporter}
	nodes := map[string]*models.Node{serviceNode.NodeID: serviceNode, runsOnNode.NodeID: runsOnNode}
	services := map[string]*models.Service{service.ServiceID: service}

	registry := &mockAgentsRegistry{}
	registry.Test(t)
	t.Cleanup(func() { registry.AssertExpectations(t) })
	registry.On("IsConnected", pmmAgent.AgentID).Return(true)

	inventoryMetrics := &InventoryMetrics{registry: registry}
	agentMetrics := make([]Metric, 0, len(agents))
	for _, agent := range agents {
		agentMetrics = append(agentMetrics, inventoryMetrics.agentMetric(t.Context(), agent, agents, nodes, services))
	}

	metricsMock := &mockInventoryMetrics{}
	metricsMock.Test(t)
	metricsMock.On("GetAgentMetrics", mock.Anything).Return(agentMetrics, nil)
	metricsMock.On("GetNodeMetrics", mock.Anything).Return([]Metric{}, nil)
	metricsMock.On("GetServiceMetrics", mock.Anything).Return([]Metric{}, nil)

	// The exporter takes environment and cluster from the node of its service, and node_id and
	// node_name from the node it runs on. The agents without a service have neither label set.
	const expected = `
		# HELP pmm_managed_inventory_agents Inventory Agent
		# TYPE pmm_managed_inventory_agents gauge
		pmm_managed_inventory_agents{agent_id="A1",agent_type="mongodb_exporter",cluster="service-node-cluster",disabled="0",environment="service-node-env",node_id="N-server",node_name="server-node",pmm_agent_id="PA1",service_id="S1",service_name="mongos-1",version=""} 2
		pmm_managed_inventory_agents{agent_id="A2",agent_type="node_exporter",cluster="",disabled="1",environment="",node_id="N-server",node_name="server-node",pmm_agent_id="PA1",service_id="",service_name="",version=""} 0
		pmm_managed_inventory_agents{agent_id="PA1",agent_type="pmm-agent",cluster="",disabled="0",environment="",node_id="N-server",node_name="server-node",pmm_agent_id="",service_id="",service_name="",version="3.11.0"} 1
	`
	err := testutil.CollectAndCompare(NewInventoryMetricsCollector(metricsMock), strings.NewReader(expected), "pmm_managed_inventory_agents")
	require.NoError(t, err)
}

func TestScrapedEnvironmentAndCluster(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		nodeLabels      string
		serviceLabels   string
		environment     string
		cluster         string
		agentLabels     string
		wantEnvironment string
		wantCluster     string
		wantErr         bool
	}{
		{
			name: "nothing set",
		},
		{
			name:            "service values",
			environment:     "ENV1",
			cluster:         "CL1",
			wantEnvironment: "ENV1",
			wantCluster:     "CL1",
		},
		{
			name:            "inherited from the node",
			nodeLabels:      `{"environment": "node-env", "cluster": "node-cluster"}`,
			wantEnvironment: "node-env",
			wantCluster:     "node-cluster",
		},
		{
			name:            "service overrides the node",
			nodeLabels:      `{"environment": "node-env", "cluster": "node-cluster"}`,
			environment:     "ENV1",
			cluster:         "CL1",
			wantEnvironment: "ENV1",
			wantCluster:     "CL1",
		},
		{
			name:            "service custom label overrides the service field",
			environment:     "ENV1",
			cluster:         "CL1",
			serviceLabels:   `{"environment": "custom-env", "cluster": "custom-cluster"}`,
			wantEnvironment: "custom-env",
			wantCluster:     "custom-cluster",
		},
		{
			name:            "agent overrides the service",
			environment:     "ENV1",
			cluster:         "CL1",
			agentLabels:     `{"environment": "agent-env", "cluster": "agent-cluster"}`,
			wantEnvironment: "agent-env",
			wantCluster:     "agent-cluster",
		},
		{
			name:            "unreadable agent labels fall back to the service",
			nodeLabels:      `{"environment": "node-env", "cluster": "node-cluster"}`,
			environment:     "ENV1",
			cluster:         "CL1",
			agentLabels:     "not json",
			wantEnvironment: "ENV1",
			wantCluster:     "CL1",
			wantErr:         true,
		},
		{
			name:            "unreadable node labels fall back to the service",
			nodeLabels:      "not json",
			environment:     "ENV1",
			cluster:         "CL1",
			agentLabels:     `{"environment": "agent-env", "cluster": "agent-cluster"}`,
			wantEnvironment: "ENV1",
			wantCluster:     "CL1",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := &models.Node{NodeID: "N1", CustomLabels: []byte(tt.nodeLabels)}
			service := &models.Service{
				ServiceID:    "S1",
				NodeID:       node.NodeID,
				Environment:  tt.environment,
				Cluster:      tt.cluster,
				CustomLabels: []byte(tt.serviceLabels),
			}
			agent := &models.Agent{AgentID: "A1", AgentType: models.MongoDBExporterType, CustomLabels: []byte(tt.agentLabels)}

			environment, cluster, err := scrapedEnvironmentAndCluster(node, service, agent)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantEnvironment, environment)
			assert.Equal(t, tt.wantCluster, cluster)
		})
	}

	t.Run("no node", func(t *testing.T) {
		t.Parallel()

		service := &models.Service{ServiceID: "S1", Environment: "ENV1", Cluster: "CL1"}
		environment, cluster, err := scrapedEnvironmentAndCluster(nil, service, &models.Agent{AgentID: "A1"})

		require.NoError(t, err)
		assert.Equal(t, "ENV1", environment)
		assert.Equal(t, "CL1", cluster)
	})
}
