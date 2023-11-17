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

	services "github.com/percona/pmm/api/inventorypb/v1/json/client/services_service"
)

func TestAddServiceMongoDB(t *testing.T) {
	t.Run("Address and port", func(t *testing.T) {
		res := &addServiceMongoDBResult{
			Service: &services.AddMongoDBServiceOKBodyMongodb{
				ServiceID:      "/service_id/1",
				ServiceName:    "MongoDB Service",
				NodeID:         "/node_id/1",
				Address:        "127.0.0.1",
				Port:           27017,
				Environment:    "environment",
				Cluster:        "mongodb-cluster",
				ReplicationSet: "mongodb-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `MongoDB Service added.
Service ID     : /service_id/1
Service name   : MongoDB Service
Node ID        : /node_id/1
Address        : 127.0.0.1
Port           : 27017
Environment    : environment
Cluster name   : mongodb-cluster
Replication set: mongodb-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})

	t.Run("Socket", func(t *testing.T) {
		res := &addServiceMongoDBResult{
			Service: &services.AddMongoDBServiceOKBodyMongodb{
				ServiceID:      "/service_id/1",
				ServiceName:    "MongoDB Socket Service",
				NodeID:         "/node_id/1",
				Socket:         "/tmp/mongodb-27017.sock",
				Environment:    "environment",
				Cluster:        "mongodb-cluster",
				ReplicationSet: "mongodb-replication-set",
				CustomLabels:   map[string]string{"key": "value", "foo": "bar"},
			},
		}
		expected := `MongoDB Service added.
Service ID     : /service_id/1
Service name   : MongoDB Socket Service
Node ID        : /node_id/1
Socket         : /tmp/mongodb-27017.sock
Environment    : environment
Cluster name   : mongodb-cluster
Replication set: mongodb-replication-set
Custom labels  : map[foo:bar key:value]
`
		require.Equal(t, expected, res.String())
	})
}
