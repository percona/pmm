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

	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

func TestAddServiceProxySQL(t *testing.T) {
	t.Run("Address and port", func(t *testing.T) {
		res := &addServiceProxySQLResult{
			Service: &services.AddServiceOKBodyProxysql{
				ServiceID:      "/service_id/1",
				ServiceName:    "ProxySQL Service",
				NodeID:         "/node_id/1",
				Address:        "127.0.0.1",
				Port:           6032,
				Environment:    "environment",
				Cluster:        "proxysql-cluster",
				ReplicationSet: "proxysql-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `ProxySQL Service added.
Service ID     : /service_id/1
Service name   : ProxySQL Service
Node ID        : /node_id/1
Address        : 127.0.0.1
Port           : 6032
Environment    : environment
Cluster name   : proxysql-cluster
Replication set: proxysql-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})

	t.Run("Socket", func(t *testing.T) {
		res := &addServiceProxySQLResult{
			Service: &services.AddServiceOKBodyProxysql{
				ServiceID:      "/service_id/1",
				ServiceName:    "ProxySQL Socket Service",
				NodeID:         "/node_id/1",
				Socket:         "/tmp/proxysql_admin.sock",
				Environment:    "environment",
				Cluster:        "proxysql-cluster",
				ReplicationSet: "proxysql-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `ProxySQL Service added.
Service ID     : /service_id/1
Service name   : ProxySQL Socket Service
Node ID        : /node_id/1
Socket         : /tmp/proxysql_admin.sock
Environment    : environment
Cluster name   : proxysql-cluster
Replication set: proxysql-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})
}
