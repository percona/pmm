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

		agentMetrics := []Metric{
			{
				labels: []string{"A1", string(models.PMMAgentType), "S1", "N1", "NN1", "PA1", strconv.Itoa(1), "V1"},
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
			pmm_managed_inventory_agents{agent_id="A1",agent_type="pmm-agent",disabled="1",node_id="N1",node_name="NN1",pmm_agent_id="PA1",service_id="S1",version="V1"} 1
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
}
