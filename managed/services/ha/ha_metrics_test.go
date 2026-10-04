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

package ha

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestHAMetricsCollectorNodeInfo(t *testing.T) {
	t.Parallel()

	t.Run("namespace known", func(t *testing.T) {
		t.Parallel()

		c := NewHAMetricsCollector(New(&models.HAParams{
			Enabled:   true,
			NodeID:    "pmm-ha-1",
			Namespace: "pmm",
		}))

		expected := `
			# HELP pmm_ha_node_info Always 1, labelled with the Kubernetes namespace this PMM replica runs in. Lets a dashboard scope cluster-wide kube-state-metrics series down to PMM's own namespace.
			# TYPE pmm_ha_node_info gauge
			pmm_ha_node_info{namespace="pmm",node_id="pmm-ha-1"} 1
		`
		err := testutil.CollectAndCompare(c, strings.NewReader(expected), "pmm_ha_node_info")
		require.NoError(t, err)
	})

	t.Run("namespace unknown", func(t *testing.T) {
		t.Parallel()

		c := NewHAMetricsCollector(New(&models.HAParams{
			Enabled: true,
			NodeID:  "pmm-ha-1",
		}))

		assert.Equal(t, 0, testutil.CollectAndCount(c, "pmm_ha_node_info"))
		assert.Equal(t, 1, testutil.CollectAndCount(c, "pmm_ha_up"))
	})

	t.Run("HA disabled", func(t *testing.T) {
		t.Parallel()

		c := NewHAMetricsCollector(New(&models.HAParams{
			NodeID:    "pmm-ha-1",
			Namespace: "pmm",
		}))

		assert.Equal(t, 0, testutil.CollectAndCount(c))
	})
}
