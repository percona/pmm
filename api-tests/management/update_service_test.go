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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// addMongoDBForUpdate adds a MongoDB Service with an exporter and a profiler QAN Agent and returns its ID.
func addMongoDBForUpdate(t *testing.T) string {
	t.Helper()

	nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
		NodeName: pmmapitests.TestString(t, "node-for-update"),
		NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
	})

	res, err := client.Default.ManagementService.AddService(&mservice.AddServiceParams{
		Context: pmmapitests.Context,
		Body: mservice.AddServiceBody{
			Mongodb: &mservice.AddServiceParamsBodyMongodb{
				NodeID:              nodeID,
				PMMAgentID:          pmmAgentID,
				ServiceName:         pmmapitests.TestString(t, "mongodb-for-update"),
				Address:             "10.10.10.10",
				Port:                27017,
				Username:            "admin",
				Password:            "admin_pass",
				Environment:         "prod",
				CustomLabels:        map[string]string{"team": "db"},
				QANMongodbProfiler:  true,
				EnableAllCollectors: true,
				CollectionsLimit:    -1,
				SkipConnectionCheck: true,
			},
		},
	})
	require.NoError(t, err)

	serviceID := res.Payload.Mongodb.Service.ServiceID
	t.Cleanup(func() { pmmapitests.RemoveServices(t, serviceID) })

	return serviceID
}

func listServiceAgents(t *testing.T, serviceID string) *agents.ListAgentsOKBody {
	t.Helper()

	res, err := inventoryClient.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		Context:   pmmapitests.Context,
		ServiceID: new(serviceID),
	})
	require.NoError(t, err)

	return res.Payload
}

func updateMongoDB(serviceID string, body *mservice.UpdateServiceParamsBodyMongodb, dryRun bool) (*mservice.UpdateServiceOK, error) {
	return client.Default.ManagementService.UpdateService(&mservice.UpdateServiceParams{
		Context:   pmmapitests.Context,
		ServiceID: serviceID,
		Body:      mservice.UpdateServiceBody{Mongodb: body, DryRun: dryRun},
	})
}

func TestUpdateService(t *testing.T) {
	t.Parallel()

	t.Run("ChangesOnlyPassedFieldsInPlace", func(t *testing.T) {
		t.Parallel()

		serviceID := addMongoDBForUpdate(t)
		before := listServiceAgents(t, serviceID)

		res, err := updateMongoDB(serviceID, &mservice.UpdateServiceParamsBodyMongodb{
			CollectionsLimit:  new(int32(0)),
			DisableCollectors: &mservice.UpdateServiceParamsBodyMongodbDisableCollectors{Values: []string{"collstats"}},
		}, false)
		require.NoError(t, err)
		assert.Equal(t, serviceID, res.Payload.Before.ServiceID)
		assert.Equal(t, serviceID, res.Payload.After.ServiceID)

		after := listServiceAgents(t, serviceID)
		require.Len(t, after.MongodbExporter, 1)
		exporter := after.MongodbExporter[0]
		assert.Equal(t, before.MongodbExporter[0].AgentID, exporter.AgentID)
		assert.Equal(t, int32(0), exporter.CollectionsLimit)
		assert.Equal(t, []string{"collstats"}, exporter.DisabledCollectors)
		assert.True(t, exporter.EnableAllCollectors)
		assert.Equal(t, "admin", exporter.Username)
		require.Len(t, after.QANMongodbProfilerAgent, 1)
		assert.Equal(t, before.QANMongodbProfilerAgent[0].AgentID, after.QANMongodbProfilerAgent[0].AgentID)
	})

	// An empty list or map must reach the server as present but empty, so that it clears the stored value.
	t.Run("EmptyValuesClearOverTheGateway", func(t *testing.T) {
		t.Parallel()

		serviceID := addMongoDBForUpdate(t)

		_, err := updateMongoDB(serviceID, &mservice.UpdateServiceParamsBodyMongodb{
			DisableCollectors: &mservice.UpdateServiceParamsBodyMongodbDisableCollectors{Values: []string{"collstats"}},
		}, false)
		require.NoError(t, err)

		_, err = updateMongoDB(serviceID, &mservice.UpdateServiceParamsBodyMongodb{
			DisableCollectors:   &mservice.UpdateServiceParamsBodyMongodbDisableCollectors{Values: []string{}},
			CustomLabels:        &mservice.UpdateServiceParamsBodyMongodbCustomLabels{},
			EnableAllCollectors: new(false),
		}, false)
		require.NoError(t, err)

		exporter := listServiceAgents(t, serviceID).MongodbExporter[0]
		assert.Empty(t, exporter.DisabledCollectors)
		assert.False(t, exporter.EnableAllCollectors)

		service, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			Context:   pmmapitests.Context,
			ServiceID: serviceID,
		})
		require.NoError(t, err)
		assert.Empty(t, service.Payload.Mongodb.CustomLabels)
		assert.Equal(t, "prod", service.Payload.Mongodb.Environment)
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Parallel()

		serviceID := addMongoDBForUpdate(t)

		res, err := updateMongoDB(serviceID, &mservice.UpdateServiceParamsBodyMongodb{Environment: new("staging")}, true)
		require.NoError(t, err)
		assert.Equal(t, "prod", res.Payload.Before.Environment)
		assert.Equal(t, "staging", res.Payload.After.Environment)

		service, err := inventoryClient.Default.ServicesService.GetService(&services.GetServiceParams{
			Context:   pmmapitests.Context,
			ServiceID: serviceID,
		})
		require.NoError(t, err)
		assert.Equal(t, "prod", service.Payload.Mongodb.Environment)
	})

	t.Run("SwitchesQuerySource", func(t *testing.T) {
		t.Parallel()

		serviceID := addMongoDBForUpdate(t)

		_, err := updateMongoDB(serviceID, &mservice.UpdateServiceParamsBodyMongodb{
			QANMongodbProfiler:  new(false),
			QANMongodbMongolog:  new(true),
			SkipConnectionCheck: true,
		}, false)
		require.NoError(t, err)

		after := listServiceAgents(t, serviceID)
		assert.Empty(t, after.QANMongodbProfilerAgent)
		require.Len(t, after.QANMongodbMongologAgent, 1)
		assert.Equal(t, "admin", after.QANMongodbMongologAgent[0].Username)
	})

	t.Run("WrongServiceType", func(t *testing.T) {
		t.Parallel()

		serviceID := addMongoDBForUpdate(t)

		_, err := client.Default.ManagementService.UpdateService(&mservice.UpdateServiceParams{
			Context:   pmmapitests.Context,
			ServiceID: serviceID,
			Body: mservice.UpdateServiceBody{
				Mysql: &mservice.UpdateServiceParamsBodyMysql{Environment: new("qa")},
			},
		})
		pmmapitests.AssertAPIErrorf(t, err, http.StatusBadRequest, codes.InvalidArgument, "is a mongodb Service, not mysql.")
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		_, err := updateMongoDB(pmmapitests.TestString(t, "no-such-service"), &mservice.UpdateServiceParamsBodyMongodb{}, false)
		pmmapitests.AssertAPIErrorf(t, err, http.StatusNotFound, codes.NotFound, "")
	})
}
