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

package management

import (
	"fmt"
	"os"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

func TestRDSDiscovery(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		accessKey, secretKey := os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET_KEY")
		if accessKey == "" || secretKey == "" {
			// TODO remove skip once secrets are added
			t.Skip("Environment variables AWS_ACCESS_KEY / AWS_SECRET_KEY are not defined, skipping test")
		}

		params := &mservice.DiscoverRDSParams{
			Body: mservice.DiscoverRDSBody{
				AWSAccessKey: accessKey,
				AWSSecretKey: secretKey,
			},
			Context: pmmapitests.Context,
		}
		discoverOK, err := client.Default.ManagementService.DiscoverRDS(params)
		require.NoError(t, err)
		require.NotNil(t, discoverOK.Payload)
		assert.NotEmpty(t, discoverOK.Payload.RDSInstances)

		// TODO Better tests: https://jira.percona.com/browse/PMM-4896
	})
}

func TestAddRds(t *testing.T) {
	t.Run("BasicAddRDS", func(t *testing.T) {
		params := &mservice.AddServiceParams{
			Body: mservice.AddServiceBody{
				RDS: &mservice.AddServiceParamsBodyRDS{
					Region:                    "region",
					Az:                        "az",
					InstanceID:                "d752f1a9-31c9-4b8c-bb2d-d26bc000001",
					NodeModel:                 "some-model",
					Address:                   "some.example.rds",
					Port:                      3306,
					Engine:                    pointer.ToString("DISCOVER_RDS_ENGINE_MYSQL"),
					NodeName:                  "some-node-name-000001",
					ServiceName:               "test-add-rds-service000001",
					Environment:               "some-env",
					Cluster:                   "cluster-01",
					ReplicationSet:            "rs-01",
					Username:                  "some-username",
					Password:                  "some-password",
					AWSAccessKey:              "my-aws-access-key",
					AWSSecretKey:              "my-aws-secret-key",
					RDSExporter:               true,
					QANMysqlPerfschema:        true,
					CustomLabels:              make(map[string]string),
					SkipConnectionCheck:       true,
					TLS:                       false,
					TLSSkipVerify:             false,
					DisableQueryExamples:      false,
					TablestatsGroupTableLimit: 2000,
					DisableBasicMetrics:       true,
					DisableEnhancedMetrics:    true,
				},
			},
			Context: pmmapitests.Context,
		}
		addRDSOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addRDSOK.Payload)

		body := addRDSOK.Payload
		assert.True(t, body.RDS.RDSExporter.BasicMetricsDisabled)
		assert.True(t, body.RDS.RDSExporter.EnhancedMetricsDisabled)

		pmmapitests.RemoveAgents(t, body.RDS.MysqldExporter.AgentID)
		pmmapitests.RemoveAgents(t, body.RDS.QANMysqlPerfschema.AgentID)
		pmmapitests.RemoveServices(t, body.RDS.Mysql.ServiceID)

		agentID := body.RDS.RDSExporter.AgentID
		nodeID := body.RDS.Mysql.NodeID
		_, err = inventoryClient.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: agentID,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf(`Agent with ID "%s" not found.`, agentID))

		_, err = inventoryClient.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  nodeID,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf(`Node with ID "%s" not found.`, nodeID))
	})

	t.Run("AddRDSPostgres", func(t *testing.T) {
		params := &mservice.AddServiceParams{
			Body: mservice.AddServiceBody{
				RDS: &mservice.AddServiceParamsBodyRDS{
					Region:                    "region",
					Az:                        "az",
					InstanceID:                "d752f1a9-31c9-4b8c-bb2d-d26bc000009",
					NodeModel:                 "some-model",
					Address:                   "some.example.rds",
					Port:                      5432,
					Engine:                    pointer.ToString("DISCOVER_RDS_ENGINE_POSTGRESQL"),
					NodeName:                  "some-node-name-000009",
					ServiceName:               "test-add-rds-service000009",
					Environment:               "some-env",
					Cluster:                   "cluster-01",
					ReplicationSet:            "rs-01",
					Username:                  "some-username",
					Password:                  "some-password",
					AWSAccessKey:              "my-aws-access-key",
					AWSSecretKey:              "my-aws-secret-key",
					RDSExporter:               true,
					CustomLabels:              make(map[string]string),
					SkipConnectionCheck:       true,
					TLS:                       false,
					TLSSkipVerify:             false,
					TablestatsGroupTableLimit: 2000,
					DisableBasicMetrics:       true,
					DisableEnhancedMetrics:    true,
					QANPostgresqlPgstatements: true,
				},
			},
			Context: pmmapitests.Context,
		}
		addRDSOK, err := client.Default.ManagementService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, addRDSOK.Payload)

		body := addRDSOK.Payload
		assert.True(t, body.RDS.RDSExporter.BasicMetricsDisabled)
		assert.True(t, body.RDS.RDSExporter.EnhancedMetricsDisabled)

		pmmapitests.RemoveAgents(t, body.RDS.PostgresqlExporter.AgentID)
		pmmapitests.RemoveAgents(t, body.RDS.QANPostgresqlPgstatements.AgentID)
		pmmapitests.RemoveServices(t, body.RDS.Postgresql.ServiceID)

		_, err = inventoryClient.Default.AgentsService.GetAgent(&agents.GetAgentParams{
			AgentID: body.RDS.RDSExporter.AgentID,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf(`Agent with ID "%s" not found.`, body.RDS.RDSExporter.AgentID))

		_, err = inventoryClient.Default.NodesService.GetNode(&nodes.GetNodeParams{
			NodeID:  body.RDS.Postgresql.NodeID,
			Context: pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, fmt.Sprintf(`Node with ID "%s" not found.`, body.RDS.Postgresql.NodeID))
	})
}
