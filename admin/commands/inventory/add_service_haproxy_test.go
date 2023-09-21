// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package inventory

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/inventorypb/json/client/services"
)

func TestAddServiceHAProxy(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		res := &addServiceHAProxyResult{
			Service: &services.AddHAProxyServiceOKBodyHaproxy{
				ServiceID:      "/service_id/1",
				ServiceName:    "ClickHouse Service",
				NodeID:         "/node_id/1",
				Environment:    "environment",
				Cluster:        "clickhouse-cluster",
				ReplicationSet: "clickhouse-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `HAProxy Service added.
Service ID     : /service_id/1
Service name   : ClickHouse Service
Node ID        : /node_id/1
Environment    : environment
Cluster name   : clickhouse-cluster
Replication set: clickhouse-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})
}
