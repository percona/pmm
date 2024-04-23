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

func TestAddServiceMySQL(t *testing.T) {
	t.Run("Address and port", func(t *testing.T) {
		res := &addServiceMySQLResult{
			Service: &services.AddServiceOKBodyMysql{
				ServiceID:      "/service_id/1",
				ServiceName:    "MySQL Service",
				NodeID:         "1",
				Address:        "127.0.0.1",
				Port:           3306,
				Environment:    "environment",
				Cluster:        "mysql-cluster",
				ReplicationSet: "mysql-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `MySQL Service added.
Service ID     : /service_id/1
Service name   : MySQL Service
Node ID        : 1
Address        : 127.0.0.1
Port           : 3306
Environment    : environment
Cluster name   : mysql-cluster
Replication set: mysql-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})

	t.Run("Socket", func(t *testing.T) {
		res := &addServiceMySQLResult{
			Service: &services.AddServiceOKBodyMysql{
				ServiceID:      "/service_id/1",
				ServiceName:    "MySQL Socket Service",
				NodeID:         "1",
				Socket:         "/path/to/socket",
				Environment:    "environment",
				Cluster:        "mysql-cluster",
				ReplicationSet: "mysql-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `MySQL Service added.
Service ID     : /service_id/1
Service name   : MySQL Socket Service
Node ID        : 1
Socket         : /path/to/socket
Environment    : environment
Cluster name   : mysql-cluster
Replication set: mysql-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})
}
