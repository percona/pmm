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

package inventory

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestInventory(t *testing.T) {
	iapi := new(mockInventoryAPI)
	inventoryCollector := NewInventory(iapi)

	agentMetrics := []Metric{
		{
			labels: []string{"A1", string(models.PMMAgentType), "S1", "N1", "PA1", strconv.Itoa(1), "V1"},
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

	t.Run("Check real metrics", func(t *testing.T) {
		_, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:7773/debug/metrics", nil)

		require.NoError(t, err)
		rw := httptest.NewRecorder()

		resp := rw.Result()
		defer resp.Body.Close() //nolint:gosec
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, b)
	})

	t.Run("Check mocked metrics", func(t *testing.T) {
		iapi.On("GetAgentDataForMetrics", mock.Anything).Return(agentMetrics, nil)
		iapi.On("GetNodeDataForMetrics", mock.Anything).Return(nodeMetrics, nil)
		iapi.On("GetServiceDataForMetrics", mock.Anything).Return(serviceMetrics, nil)

		const aMetadata = `
			# HELP pmm_managed_inventory_agents The current information about agent
			# TYPE pmm_managed_inventory_agents gauge
		`

		aExpected := `
			pmm_managed_inventory_agents{agent_id="A1",agent_type="pmm-agent",disabled="1",node_id="N1",pmm_agent_id="PA1",service_id="S1",version="V1"} 1
		`

		const nMetadata = `
			# HELP pmm_managed_inventory_nodes The current information about node
        	# TYPE pmm_managed_inventory_nodes gauge
		`

		nExpected := `
        	pmm_managed_inventory_nodes{container_name="C1",node_id="N1",node_name="N1",node_type="generic"} 1
		`

		const sMetadata = `
			# HELP pmm_managed_inventory_services The current information about service
			# TYPE pmm_managed_inventory_services gauge
		`

		sExpected := `
			pmm_managed_inventory_services{node_id="N1",service_id="C1",service_type="proxysql"} 1
		`

		if err := testutil.CollectAndCompare(inventoryCollector, strings.NewReader(aMetadata+aExpected), "pmm_managed_inventory_agents"); err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		if err := testutil.CollectAndCompare(inventoryCollector, strings.NewReader(nMetadata+nExpected), "pmm_managed_inventory_nodes"); err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		if err := testutil.CollectAndCompare(inventoryCollector, strings.NewReader(sMetadata+sExpected), "pmm_managed_inventory_services"); err != nil {
			t.Errorf("Unexpected collecting result:\n%s", err)
		}

		iapi.AssertExpectations(t)
	})
}
