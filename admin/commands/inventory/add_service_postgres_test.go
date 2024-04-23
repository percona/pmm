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

func TestAddServicePostgreSQL(t *testing.T) {
	t.Run("Address and port", func(t *testing.T) {
		res := &addServicePostgreSQLResult{
			Service: &services.AddServiceOKBodyPostgresql{
				ServiceID:      "1",
				ServiceName:    "PostgreSQL Service",
				NodeID:         "1",
				Address:        "127.0.0.1",
				Port:           5432,
				Environment:    "environment",
				Cluster:        "postgresql-cluster",
				ReplicationSet: "postgresql-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `PostgreSQL Service added.
Service ID     : 1
Service name   : PostgreSQL Service
Node ID        : 1
Address        : 127.0.0.1
Port           : 5432
Environment    : environment
Cluster name   : postgresql-cluster
Replication set: postgresql-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})

	t.Run("Socket", func(t *testing.T) {
		res := &addServicePostgreSQLResult{
			Service: &services.AddServiceOKBodyPostgresql{
				ServiceID:      "1",
				ServiceName:    "PostgreSQL Socket Service",
				NodeID:         "1",
				Socket:         "/var/run/postgresql",
				Environment:    "environment",
				Cluster:        "postgresql-cluster",
				ReplicationSet: "postgresql-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `PostgreSQL Service added.
Service ID     : 1
Service name   : PostgreSQL Socket Service
Node ID        : 1
Socket         : /var/run/postgresql
Environment    : environment
Cluster name   : postgresql-cluster
Replication set: postgresql-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})
}
