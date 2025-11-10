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
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestNewInventoryMetricsCollector(t *testing.T) {
	t.Run("Metrics returns inventory metrics", func(t *testing.T) {
		client := http.Client{}

		ctx, cancelCtx := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelCtx()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:7773/debug/metrics", nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

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

		// Agent labels: agent_id, agent_type, az, cluster, container_id, container_name, disabled, environment, external_group, machine_id, node_id, node_model, node_name, node_type, pmm_agent_id, region, replication_set, service_id, service_name, service_type, version
		agentMetrics := []Metric{
			{
				labels: []string{"A1", string(models.PMMAgentType), "", "", "", "", "1", "", "", "", "N1", "", "NN1", "", "PA1", "", "", "S1", "", "", "V1"},
				value:  float64(1),
			},
		}

		// Node labels: az, container_id, container_name, machine_id, node_id, node_model, node_name, node_type, region
		nodeMetrics := []Metric{
			{
				labels: []string{"AZ1", "CID1", "C1", "M1", "N1", "NM1", "N1", string(models.GenericNodeType), "R1"},
				value:  float64(1),
			},
		}

		// Service labels: az, cluster, container_id, container_name, environment, external_group, machine_id, node_id, node_model, node_name, node_type, region, replication_set, service_id, service_name, service_type
		serviceMetrics := []Metric{
			{
				labels: []string{"AZ1", "CL1", "CID1", "C1", "ENV1", "EG1", "M1", "N1", "NM1", "N1", string(models.GenericNodeType), "R1", "RS1", "C1", "SN1", string(models.ProxySQLServiceType)},
				value:  float64(1),
			},
		}

		metricsMock.On("GetAgentMetrics", mock.Anything).Return(agentMetrics, nil)
		metricsMock.On("GetNodeMetrics", mock.Anything).Return(nodeMetrics, nil)
		metricsMock.On("GetServiceMetrics", mock.Anything).Return(serviceMetrics, nil)

		const expectedAgentMetrics = `
			# HELP pmm_managed_inventory_agents Inventory Agent
			# TYPE pmm_managed_inventory_agents gauge
			pmm_managed_inventory_agents{agent_id="A1",agent_type="pmm-agent",az="",cluster="",container_id="",container_name="",disabled="1",environment="",external_group="",machine_id="",node_id="N1",node_model="",node_name="NN1",node_type="",pmm_agent_id="PA1",region="",replication_set="",service_id="S1",service_name="",service_type="",version="V1"} 1
		`

		const expectedNodeMetrics = `
			# HELP pmm_managed_inventory_nodes Inventory Node
			# TYPE pmm_managed_inventory_nodes gauge
			pmm_managed_inventory_nodes{az="AZ1",container_id="CID1",container_name="C1",machine_id="M1",node_id="N1",node_model="NM1",node_name="N1",node_type="generic",region="R1"} 1
		`

		const expectedServiceMetrics = `
			# HELP pmm_managed_inventory_services Inventory Service
			# TYPE pmm_managed_inventory_services gauge
			pmm_managed_inventory_services{az="AZ1",cluster="CL1",container_id="CID1",container_name="C1",environment="ENV1",external_group="EG1",machine_id="M1",node_id="N1",node_model="NM1",node_name="N1",node_type="generic",region="R1",replication_set="RS1",service_id="C1",service_name="SN1",service_type="proxysql"} 1
		`

		if err := testutil.CollectAndCompare(
			inventoryCollector,
			strings.NewReader(expectedAgentMetrics),
			"pmm_managed_inventory_agents"); err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		if err := testutil.CollectAndCompare(
			inventoryCollector,
			strings.NewReader(expectedNodeMetrics),
			"pmm_managed_inventory_nodes"); err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		if err := testutil.CollectAndCompare(
			inventoryCollector,
			strings.NewReader(expectedServiceMetrics),
			"pmm_managed_inventory_services"); err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}
	})

	t.Run("Agent metrics include node and service labels", func(t *testing.T) {
		uuid.SetRand(&tests.IDReader{})
		defer uuid.SetRand(nil)

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		defer func() {
			require.NoError(t, sqlDB.Close())
		}()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Create a simple mock registry
		registry := &mockAgentsRegistry{}
		registry.Test(t)
		registry.On("IsConnected", mock.Anything).Return(false)

		// Create node with labels
		var node *models.Node
		err := db.InTransaction(func(tx *reform.TX) error {
			var err error
			node, err = models.CreateNode(tx.Querier, models.GenericNodeType, &models.CreateNodeParams{
				NodeName:  "TestNode",
				Address:   "127.0.0.1",
				MachineID: pointer.ToString("machine-123"),
				NodeModel: "TestModel",
				Region:    pointer.ToString("us-east-1"),
				AZ:        "us-east-1a",
			})
			return err
		})
		require.NoError(t, err)

		// Create service with labels
		var service *models.Service
		err = db.InTransaction(func(tx *reform.TX) error {
			var err error
			service, err = models.AddNewService(tx.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName:    "TestService",
				NodeID:         node.NodeID,
				Address:        pointer.ToString("127.0.0.1"),
				Port:           pointer.ToUint16(3306),
				Environment:    "production",
				Cluster:        "test-cluster",
				ReplicationSet: "rs-1",
			})
			return err
		})
		require.NoError(t, err)

		// Create PMM agent
		var pmmAgent *models.Agent
		err = db.InTransaction(func(tx *reform.TX) error {
			var err error
			pmmAgent, err = models.CreatePMMAgent(tx.Querier, node.NodeID, nil)
			return err
		})
		require.NoError(t, err)

		// Create MySQL exporter agent
		var agent *models.Agent
		err = db.InTransaction(func(tx *reform.TX) error {
			var err error
			agent, err = models.CreateAgent(tx.Querier, models.MySQLdExporterType, &models.CreateAgentParams{
				PMMAgentID: pmmAgent.AgentID,
				ServiceID:  service.ServiceID,
			})
			if err != nil {
				return err
			}
			// Set version after creation
			agent.Version = pointer.ToString("v1.0.0")
			return tx.Update(agent)
		})
		require.NoError(t, err)

		// Create InventoryMetrics with real database
		metrics := NewInventoryMetrics(db, registry)

		// Collect metrics
		ctx := context.Background()
		agentMetrics, err := metrics.GetAgentMetrics(ctx)
		require.NoError(t, err)
		require.Len(t, agentMetrics, 2) // PMM agent + MySQL exporter

		// Find the MySQL exporter agent metric
		var mysqlAgentMetric *Metric
		for i := range agentMetrics {
			// agent_id is the first label
			if len(agentMetrics[i].labels) > 0 && agentMetrics[i].labels[0] == agent.AgentID {
				mysqlAgentMetric = &agentMetrics[i]
				break
			}
		}
		require.NotNil(t, mysqlAgentMetric, "MySQL exporter agent metric should be found")

		// Verify node labels are present in agent metrics
		// Labels are in order: agent_id, agent_type, az, cluster, container_id, container_name, disabled, environment, external_group, machine_id, node_id, node_model, node_name, node_type, pmm_agent_id, region, replication_set, service_id, service_name, service_type, version
		labelMap := make(map[string]string)
		labelNames := agentLabelNames
		for i, name := range labelNames {
			if i < len(mysqlAgentMetric.labels) {
				labelMap[name] = mysqlAgentMetric.labels[i]
			}
		}

		// Verify node labels
		assert.Equal(t, node.NodeID, labelMap["node_id"], "Agent metrics should contain node_id")
		assert.Equal(t, node.NodeName, labelMap["node_name"], "Agent metrics should contain node_name")
		assert.Equal(t, string(node.NodeType), labelMap["node_type"], "Agent metrics should contain node_type")
		assert.Equal(t, node.AZ, labelMap["az"], "Agent metrics should contain az (node)")
		assert.Equal(t, pointer.GetString(node.Region), labelMap["region"], "Agent metrics should contain region (node)")
		assert.Equal(t, pointer.GetString(node.MachineID), labelMap["machine_id"], "Agent metrics should contain machine_id (node)")

		// Verify service labels
		assert.Equal(t, service.ServiceID, labelMap["service_id"], "Agent metrics should contain service_id")
		assert.Equal(t, service.ServiceName, labelMap["service_name"], "Agent metrics should contain service_name")
		assert.Equal(t, string(service.ServiceType), labelMap["service_type"], "Agent metrics should contain service_type")
		assert.Equal(t, service.Cluster, labelMap["cluster"], "Agent metrics should contain cluster (service)")
		assert.Equal(t, service.Environment, labelMap["environment"], "Agent metrics should contain environment (service)")
		assert.Equal(t, service.ReplicationSet, labelMap["replication_set"], "Agent metrics should contain replication_set (service)")

		// Verify agent-specific labels
		assert.Equal(t, agent.AgentID, labelMap["agent_id"], "Agent metrics should contain agent_id")
		assert.Equal(t, string(agent.AgentType), labelMap["agent_type"], "Agent metrics should contain agent_type")
		assert.Equal(t, pointer.GetString(agent.PMMAgentID), labelMap["pmm_agent_id"], "Agent metrics should contain pmm_agent_id")
		assert.Equal(t, pointer.GetString(agent.Version), labelMap["version"], "Agent metrics should contain version")
	})

	t.Run("Service metrics include node labels", func(t *testing.T) {
		uuid.SetRand(&tests.IDReader{})
		defer uuid.SetRand(nil)

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		defer func() {
			require.NoError(t, sqlDB.Close())
		}()
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		// Create a simple mock registry
		registry := &mockAgentsRegistry{}
		registry.Test(t)

		// Create node with labels
		var node *models.Node
		err := db.InTransaction(func(tx *reform.TX) error {
			var err error
			node, err = models.CreateNode(tx.Querier, models.GenericNodeType, &models.CreateNodeParams{
				NodeName:      "TestNode",
				Address:       "127.0.0.1",
				MachineID:     pointer.ToString("machine-456"),
				NodeModel:     "TestModel",
				Region:        pointer.ToString("us-west-2"),
				AZ:            "us-west-2a",
				ContainerID:   pointer.ToString("container-123"),
				ContainerName: pointer.ToString("test-container"),
			})
			return err
		})
		require.NoError(t, err)

		// Create service with labels
		var service *models.Service
		err = db.InTransaction(func(tx *reform.TX) error {
			var err error
			service, err = models.AddNewService(tx.Querier, models.MySQLServiceType, &models.AddDBMSServiceParams{
				ServiceName:    "TestService",
				NodeID:         node.NodeID,
				Address:        pointer.ToString("127.0.0.1"),
				Port:           pointer.ToUint16(3306),
				Environment:    "staging",
				Cluster:        "test-cluster-2",
				ReplicationSet: "rs-2",
			})
			return err
		})
		require.NoError(t, err)

		// Create InventoryMetrics with real database
		metrics := NewInventoryMetrics(db, registry)

		// Collect metrics
		ctx := context.Background()
		serviceMetrics, err := metrics.GetServiceMetrics(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(serviceMetrics), 1, "Should have at least one service metric")

		// Find our test service metric
		var testServiceMetric *Metric
		for i := range serviceMetrics {
			// service_id is the 14th label (index 13) in serviceLabelNames
			if len(serviceMetrics[i].labels) > 13 && serviceMetrics[i].labels[13] == service.ServiceID {
				testServiceMetric = &serviceMetrics[i]
				break
			}
		}
		require.NotNil(t, testServiceMetric, "Test service metric should be found")

		// Verify node labels are present in service metrics
		// Labels are in order: az, cluster, container_id, container_name, environment, external_group, machine_id, node_id, node_model, node_name, node_type, region, replication_set, service_id, service_name, service_type
		labelMap := make(map[string]string)
		labelNames := serviceLabelNames
		for i, name := range labelNames {
			if i < len(testServiceMetric.labels) {
				labelMap[name] = testServiceMetric.labels[i]
			}
		}

		// Verify node labels are present
		assert.Equal(t, node.NodeID, labelMap["node_id"], "Service metrics should contain node_id")
		assert.Equal(t, node.NodeName, labelMap["node_name"], "Service metrics should contain node_name")
		assert.Equal(t, string(node.NodeType), labelMap["node_type"], "Service metrics should contain node_type")
		assert.Equal(t, node.AZ, labelMap["az"], "Service metrics should contain az (node)")
		assert.Equal(t, pointer.GetString(node.Region), labelMap["region"], "Service metrics should contain region (node)")
		assert.Equal(t, pointer.GetString(node.MachineID), labelMap["machine_id"], "Service metrics should contain machine_id (node)")
		assert.Equal(t, pointer.GetString(node.ContainerID), labelMap["container_id"], "Service metrics should contain container_id (node)")
		assert.Equal(t, pointer.GetString(node.ContainerName), labelMap["container_name"], "Service metrics should contain container_name (node)")
		assert.Equal(t, node.NodeModel, labelMap["node_model"], "Service metrics should contain node_model (node)")

		// Verify service labels
		assert.Equal(t, service.ServiceID, labelMap["service_id"], "Service metrics should contain service_id")
		assert.Equal(t, service.ServiceName, labelMap["service_name"], "Service metrics should contain service_name")
		assert.Equal(t, string(service.ServiceType), labelMap["service_type"], "Service metrics should contain service_type")
		assert.Equal(t, service.Cluster, labelMap["cluster"], "Service metrics should contain cluster")
		assert.Equal(t, service.Environment, labelMap["environment"], "Service metrics should contain environment")
		assert.Equal(t, service.ReplicationSet, labelMap["replication_set"], "Service metrics should contain replication_set")
	})
}
