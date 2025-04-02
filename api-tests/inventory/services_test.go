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
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/inventory/v1/types"
)

func TestServices(t *testing.T) {
	t.Parallel()
	t.Run("List", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		remoteNodeOKBody := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for services test"))
		remoteNodeID := remoteNodeOKBody.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, remoteNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "Some MySQL Service"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		remoteService := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      remoteNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "Some MySQL Service on remote Node"),
			},
		})
		remoteServiceID := remoteService.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, remoteServiceID)

		postgreSQLService := addService(t, services.AddServiceBody{
			Postgresql: &services.AddServiceParamsBodyPostgresql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        5432,
				ServiceName: pmmapitests.TestString(t, "Some MySQL Service on remote Node"),
			},
		})
		postgreSQLServiceID := postgreSQLService.Postgresql.ServiceID
		defer pmmapitests.RemoveServices(t, postgreSQLServiceID)

		externalService := addService(t, services.AddServiceBody{
			External: &services.AddServiceParamsBodyExternal{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "Some External Service on remote Node"),
				Group:       "rabbitmq",
			},
		})
		externalServiceID := externalService.External.ServiceID
		defer pmmapitests.RemoveServices(t, externalServiceID)

		haProxyService := addService(t, services.AddServiceBody{
			Haproxy: &services.AddServiceParamsBodyHaproxy{
				NodeID:      genericNodeID,
				ServiceName: pmmapitests.TestString(t, "Some External Service on remote Node"),
			},
		})
		haProxyServiceID := haProxyService.Haproxy.ServiceID
		defer pmmapitests.RemoveServices(t, haProxyServiceID)

		res, err := client.Default.ServicesService.ListServices(&services.ListServicesParams{Context: pmmapitests.Context})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.Payload.Mysql, "There should be at least one MySQL service")
		assert.NotEmpty(t, res.Payload.Postgresql, "There should be at least one PostgreSQL service")
		assertMySQLServiceExists(t, res, serviceID)
		assertMySQLServiceExists(t, res, remoteServiceID)
		assertPostgreSQLServiceExists(t, res, postgreSQLServiceID)
		assertExternalServiceExists(t, res, externalServiceID)
		assertHAProxyServiceExists(t, res, haProxyServiceID)

		// Filter by node ID.
		res, err = client.Default.ServicesService.ListServices(&services.ListServicesParams{
			NodeID:      pointer.ToString(genericNodeID),
			ServiceType: nil,
			Context:     pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.Payload.Mysql, "There should be at least one MySQL service")
		assert.NotEmpty(t, res.Payload.Postgresql, "There should be at least one PostgreSQL service")
		assertMySQLServiceExists(t, res, serviceID)
		assertMySQLServiceNotExist(t, res, remoteServiceID)
		assertPostgreSQLServiceExists(t, res, postgreSQLServiceID)
		assertExternalServiceExists(t, res, externalServiceID)
		assertHAProxyServiceExists(t, res, haProxyServiceID)

		// Filter by service type.
		res, err = client.Default.ServicesService.ListServices(&services.ListServicesParams{
			ServiceType: pointer.ToString(types.ServiceTypePostgreSQLService),
			Context:     pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.Payload.Postgresql, "There should be at least one PostgreSQL service")
		assertMySQLServiceNotExist(t, res, serviceID)
		assertMySQLServiceNotExist(t, res, remoteServiceID)
		assertExternalServiceNotExist(t, res, externalServiceID)
		assertHAProxyServiceNotExist(t, res, haProxyServiceID)
		assertPostgreSQLServiceExists(t, res, postgreSQLServiceID)
	})

	t.Run("FilterList", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		remoteNodeOKBody := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node to check services filter"))
		remoteNodeID := remoteNodeOKBody.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, remoteNodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      genericNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "Some MySQL Service for filters test"),
			},
		})
		serviceID := service.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)

		remoteService := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      remoteNodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "Some MySQL Service on remote Node for filters test"),
			},
		})
		remoteServiceID := remoteService.Mysql.ServiceID
		defer pmmapitests.RemoveServices(t, remoteServiceID)

		res, err := client.Default.ServicesService.ListServices(&services.ListServicesParams{
			NodeID:  pointer.ToString(remoteNodeID),
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotEmpty(t, res.Payload.Mysql, "There should be at least one node")
		assertMySQLServiceNotExist(t, res, serviceID)
		assertMySQLServiceExists(t, res, remoteServiceID)
	})
}

func TestGetService(t *testing.T) {
	t.Parallel()
	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()

		params := &services.GetServiceParams{
			ServiceID: "pmm-not-found",
			Context:   pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.GetService(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID \"pmm-not-found\" not found.")
		assert.Nil(t, res)
	})

	t.Run("EmptyServiceID", func(t *testing.T) {
		t.Parallel()

		params := &services.GetServiceParams{
			ServiceID: "",
			Context:   pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.GetService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid GetServiceRequest.ServiceId: value length must be at least 1 runes")
		assert.Nil(t, res)
	})
}

func TestRemoveService(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for agents list"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      nodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
			},
		})
		serviceID := service.Mysql.ServiceID

		params := &services.RemoveServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.RemoveService(params)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Has agents", func(t *testing.T) {
		t.Parallel()

		node := pmmapitests.AddRemoteNode(t, pmmapitests.TestString(t, "Remote node for agents list"))
		nodeID := node.Remote.NodeID
		defer pmmapitests.RemoveNodes(t, nodeID)

		service := addService(t, services.AddServiceBody{
			Mysql: &services.AddServiceParamsBodyMysql{
				NodeID:      nodeID,
				Address:     "localhost",
				Port:        3306,
				ServiceName: pmmapitests.TestString(t, "MySQL Service for agent"),
			},
		})
		serviceID := service.Mysql.ServiceID

		pmmAgent := pmmapitests.AddPMMAgent(t, nodeID)
		pmmAgentID := pmmAgent.PMMAgent.AgentID
		defer pmmapitests.RemoveAgents(t, pmmAgentID)

		_ = addAgent(t, agents.AddAgentBody{
			MysqldExporter: &agents.AddAgentParamsBodyMysqldExporter{
				ServiceID:  serviceID,
				Username:   "username",
				Password:   "password",
				PMMAgentID: pmmAgentID,

				SkipConnectionCheck: true,
			},
		})

		params := &services.RemoveServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.RemoveService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `Service with ID %q has agents.`, serviceID)
		assert.Nil(t, res)

		// Remove with force flag.
		params = &services.RemoveServiceParams{
			ServiceID: serviceID,
			Force:     pointer.ToBool(true),
			Context:   pmmapitests.Context,
		}
		res, err = client.Default.ServicesService.RemoveService(params)
		require.NoError(t, err)
		assert.NotNil(t, res)

		// Check that the service and agents are removed.
		getServiceResp, err := client.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, getServiceResp)

		listAgentsOK, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
			ServiceID: pointer.ToString(serviceID),
			Context:   pmmapitests.Context,
		})
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, "Service with ID %q not found.", serviceID)
		assert.Nil(t, listAgentsOK)
	})

	t.Run("Not-exist service", func(t *testing.T) {
		t.Parallel()
		serviceID := "not-exist-service-id"

		params := &services.RemoveServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.RemoveService(params)
		pmmapitests.AssertAPIErrorf(t, err, 404, codes.NotFound, `Service with ID %q not found.`, serviceID)
		assert.Nil(t, res)
	})

	t.Run("Empty params", func(t *testing.T) {
		t.Parallel()
		removeResp, err := client.Default.ServicesService.RemoveService(&services.RemoveServiceParams{
			Context: context.Background(),
		})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid RemoveServiceRequest.ServiceId: value length must be at least 1 runes")
		assert.Nil(t, removeResp)
	})
}

func TestMySQLService(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Basic MySQL Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        3306,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.Mysql.ServiceID
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				Mysql: &services.AddServiceOKBodyMysql{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					Address:      "localhost",
					Port:         3306,
					ServiceName:  serviceName,
					CustomLabels: map[string]string{},
				},
			},
		}, res)
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check if the service saved in pmm-managed.
		serviceRes, err := client.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceRes)
		assert.Equal(t, &services.GetServiceOK{
			Payload: &services.GetServiceOKBody{
				Mysql: &services.GetServiceOKBodyMysql{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					Address:      "localhost",
					Port:         3306,
					ServiceName:  serviceName,
					CustomLabels: map[string]string{},
				},
			},
		}, serviceRes)

		// Check duplicates.
		params = &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					Address:     "127.0.0.1",
					Port:        3336,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err = client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Service with name %q already exists.", serviceName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      "",
					Address:     "localhost",
					Port:        3306,
					ServiceName: pmmapitests.TestString(t, "MySQL Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLServiceParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})

	t.Run("AddEmptyPort", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					ServiceName: pmmapitests.TestString(t, "MySQL Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})

	t.Run("AddAddressSocketConflict", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        3306,
					Socket:      "/var/run/mysqld/mysqld.sock",
					ServiceName: pmmapitests.TestString(t, "MySQL Service with address and socket conflict"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})

	t.Run("AddPortWithNoAddress", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "MySQL Service with port and socket"),
					Port:        3306,
					Socket:      "/var/run/mysqld/mysqld.sock",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and port cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})

	t.Run("AddEpmtyAddressAndSocket", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "MySQL Service with empty address and socket"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})

	t.Run("AddServiceNameEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mysql: &services.AddServiceParamsBodyMysql{
					NodeID:      genericNodeID,
					ServiceName: "",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMySQLServiceParams.ServiceName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mysql.ServiceID)
		}
	})
}

func TestMongoDBService(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Basic Mongo Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					ServiceName: serviceName,
					Address:     "localhost",
					Port:        27017,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.Mongodb.ServiceID
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				Mongodb: &services.AddServiceOKBodyMongodb{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					ServiceName:  serviceName,
					Address:      "localhost",
					Port:         27017,
					CustomLabels: map[string]string{},
				},
			},
		}, res)
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check if the service saved in pmm-managed.
		serviceRes, err := client.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		require.NotNil(t, serviceRes)
		assert.Equal(t, &services.GetServiceOK{
			Payload: &services.GetServiceOKBody{
				Mongodb: &services.GetServiceOKBodyMongodb{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					ServiceName:  serviceName,
					Address:      "localhost",
					Port:         27017,
					CustomLabels: map[string]string{},
				},
			},
		}, serviceRes)

		// Check duplicates.
		params = &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					ServiceName: serviceName,
					Address:     "localhost",
					Port:        27017,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err = client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Service with name %q already exists.", serviceName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mongodb.ServiceID)
		}
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      "",
					ServiceName: pmmapitests.TestString(t, "MongoDB Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMongoDBServiceParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mongodb.ServiceID)
		}
	})

	t.Run("AddServiceNameEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					ServiceName: "",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddMongoDBServiceParams.ServiceName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mongodb.ServiceID)
		}
	})

	t.Run("AddAddressSocketConflict", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        27017,
					Socket:      "/tmp/mongodb-27017.sock",
					ServiceName: pmmapitests.TestString(t, "MongoDB Service with address and socket conflict"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mongodb.ServiceID)
		}
	})

	t.Run("AddPortWithNoAddress", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "MongoDB Service with port and socket"),
					Port:        27017,
					Socket:      "/tmp/mongodb-27017.sock",
				},
			},
			Context: pmmapitests.Context,
		}

		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and port cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mongodb.ServiceID)
		}
	})

	t.Run("AddEpmtyAddressAndSocket", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "MongoDB Service with empty address and socket"),
				},
			},
			Context: pmmapitests.Context,
		}

		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Mongodb.ServiceID)
		}
	})

	t.Run("Socket", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		defer pmmapitests.RemoveNodes(t, genericNodeID)
		require.NotEmpty(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Mongo with Socket Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Mongodb: &services.AddServiceParamsBodyMongodb{
					NodeID:      genericNodeID,
					ServiceName: serviceName,
					Socket:      "/tmp/mongodb-27017.sock",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.Mongodb.ServiceID
		defer pmmapitests.RemoveServices(t, serviceID)
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				Mongodb: &services.AddServiceOKBodyMongodb{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					ServiceName:  serviceName,
					Socket:       "/tmp/mongodb-27017.sock",
					CustomLabels: map[string]string{},
				},
			},
		}, res)
	})
}

func TestPostgreSQLService(t *testing.T) {
	t.Parallel()
	const defaultPostgresDBName = "postgres"

	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Basic PostgreSQL Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        5432,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.Postgresql.ServiceID
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				Postgresql: &services.AddServiceOKBodyPostgresql{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					Address:      "localhost",
					DatabaseName: defaultPostgresDBName,
					Port:         5432,
					ServiceName:  serviceName,
					CustomLabels: map[string]string{},
				},
			},
		}, res)
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check if the service saved in pmm-managed.
		serviceRes, err := client.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceRes)
		assert.Equal(t, &services.GetServiceOK{
			Payload: &services.GetServiceOKBody{
				Postgresql: &services.GetServiceOKBodyPostgresql{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					Address:      "localhost",
					DatabaseName: defaultPostgresDBName,
					Port:         5432,
					ServiceName:  serviceName,
					CustomLabels: map[string]string{},
				},
			},
		}, serviceRes)

		// Check duplicates.
		params = &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					Address:     "127.0.0.1",
					Port:        3336,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err = client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Service with name %q already exists.", serviceName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      "",
					Address:     "localhost",
					Port:        5432,
					ServiceName: pmmapitests.TestString(t, "PostgreSQL Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgreSQLServiceParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})

	t.Run("AddEmptyPort", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					ServiceName: pmmapitests.TestString(t, "PostgreSQL Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})

	t.Run("AddServiceNameEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					ServiceName: "",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddPostgreSQLServiceParams.ServiceName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})

	t.Run("AddAddressSocketConflict", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        5432,
					Socket:      "/var/run/postgresql",
					ServiceName: pmmapitests.TestString(t, "PostgreSQL Service with address and socket conflict"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})

	t.Run("AddPortWithNoAddress", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "PostgreSQL Service with port and socket"),
					Port:        5432,
					Socket:      "/var/run/postgresql",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and port cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})

	t.Run("AddEmptyAddressAndSocket", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Postgresql: &services.AddServiceParamsBodyPostgresql{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "PostgreSQL Service with empty address and socket"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Postgresql.ServiceID)
		}
	})
}

func TestProxySQLService(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Basic ProxySQL Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        5432,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.Proxysql.ServiceID
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				Proxysql: &services.AddServiceOKBodyProxysql{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					Address:      "localhost",
					Port:         5432,
					ServiceName:  serviceName,
					CustomLabels: map[string]string{},
				},
			},
		}, res)
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check if the service saved in pmm-managed.
		serviceRes, err := client.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceRes)
		assert.Equal(t, &services.GetServiceOK{
			Payload: &services.GetServiceOKBody{
				Proxysql: &services.GetServiceOKBodyProxysql{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					Address:      "localhost",
					Port:         5432,
					ServiceName:  serviceName,
					CustomLabels: map[string]string{},
				},
			},
		}, serviceRes)

		// Check duplicates.
		params = &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					Address:     "127.0.0.1",
					Port:        3336,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err = client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Service with name %q already exists.", serviceName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      "",
					Address:     "localhost",
					Port:        5432,
					ServiceName: pmmapitests.TestString(t, "ProxySQL Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddProxySQLServiceParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})

	t.Run("AddEmptyPort", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					ServiceName: pmmapitests.TestString(t, "ProxySQL Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Port is expected to be passed along with the host address.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})

	t.Run("AddAddressSocketConflict", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					Address:     "localhost",
					Port:        6032,
					Socket:      "/tmp/proxysql_admin.sock",
					ServiceName: pmmapitests.TestString(t, "ProxySQL Service with address and socket conflict"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and address cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})

	t.Run("AddPortWithNoAddress", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "ProxySQL Service with port and socket"),
					Port:        6032,
					Socket:      "/tmp/proxysql_admin.sock",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Socket and port cannot be specified together.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})

	t.Run("AddEpmtyAddressAndSocket", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					ServiceName: pmmapitests.TestString(t, "ProxySQL Service with empty address and socket"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Neither socket nor address passed.")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})

	t.Run("AddServiceNameEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				Proxysql: &services.AddServiceParamsBodyProxysql{
					NodeID:      genericNodeID,
					ServiceName: "",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddProxySQLServiceParams.ServiceName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.Proxysql.ServiceID)
		}
	})
}

func TestExternalService(t *testing.T) {
	t.Parallel()
	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		containsExternalWithGroup := func(items []*services.ListServicesOKBodyExternalItems0, expectedGroup string) func() bool {
			return func() bool {
				for _, ext := range items {
					if ext.Group == expectedGroup {
						return true
					}
				}
				return false
			}
		}

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Basic External Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				External: &services.AddServiceParamsBodyExternal{
					NodeID:      genericNodeID,
					ServiceName: serviceName,
					Group:       "redis",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.External.ServiceID
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				External: &services.AddServiceOKBodyExternal{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					ServiceName:  serviceName,
					Group:        "redis",
					CustomLabels: map[string]string{},
				},
			},
		}, res)
		defer pmmapitests.RemoveServices(t, serviceID)

		// Check if the service saved in pmm-managed.
		serviceRes, err := client.Default.ServicesService.GetService(&services.GetServiceParams{
			ServiceID: serviceID,
			Context:   pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, serviceRes)
		assert.Equal(t, &services.GetServiceOK{
			Payload: &services.GetServiceOKBody{
				External: &services.GetServiceOKBodyExternal{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					ServiceName:  serviceName,
					Group:        "redis",
					CustomLabels: map[string]string{},
				},
			},
		}, serviceRes)

		// Filter services by external group.
		servicesList, err := client.Default.ServicesService.ListServices(&services.ListServicesParams{
			ExternalGroup: pointer.ToString("redis"),
			Context:       pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, servicesList)
		assert.Empty(t, servicesList.Payload.Mysql)
		assert.Empty(t, servicesList.Payload.Mongodb)
		assert.Empty(t, servicesList.Payload.Postgresql)
		assert.Empty(t, servicesList.Payload.Proxysql)
		assert.Len(t, servicesList.Payload.External, 1)
		assert.Conditionf(t, containsExternalWithGroup(servicesList.Payload.External, "redis"), "list does not contain external group %s", "redis")

		// Filter services by a non-existing external group.
		emptyServicesList, err := client.Default.ServicesService.ListServices(&services.ListServicesParams{
			ExternalGroup: pointer.ToString("non-existing-external-group"),
			Context:       pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, emptyServicesList)
		assert.Empty(t, emptyServicesList.Payload.Mysql)
		assert.Empty(t, emptyServicesList.Payload.Mongodb)
		assert.Empty(t, emptyServicesList.Payload.Postgresql)
		assert.Empty(t, emptyServicesList.Payload.Proxysql)
		assert.Empty(t, emptyServicesList.Payload.External)

		//  List services with out filter by external group.
		noFilterServicesList, err := client.Default.ServicesService.ListServices(&services.ListServicesParams{
			ExternalGroup: pointer.ToString(""),
			Context:       pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.NotNil(t, noFilterServicesList)
		assert.GreaterOrEqual(t, len(noFilterServicesList.Payload.Mysql), 0)
		assert.GreaterOrEqual(t, len(noFilterServicesList.Payload.Mongodb), 0)
		assert.NotEmpty(t, noFilterServicesList.Payload.Postgresql)
		assert.GreaterOrEqual(t, len(noFilterServicesList.Payload.Proxysql), 0)
		assert.NotEmpty(t, noFilterServicesList.Payload.External)
		assert.Conditionf(t, containsExternalWithGroup(noFilterServicesList.Payload.External, "redis"), "list does not contain external group %s", "redis")

		// Check duplicates.
		params = &services.AddServiceParams{
			Body: services.AddServiceBody{
				External: &services.AddServiceParamsBodyExternal{
					NodeID:      genericNodeID,
					ServiceName: serviceName,
					Group:       "redis",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err = client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 409, codes.AlreadyExists, "Service with name %q already exists.", serviceName)
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.External.ServiceID)
		}
	})

	t.Run("AddNodeIDEmpty", func(t *testing.T) {
		t.Parallel()

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				External: &services.AddServiceParamsBodyExternal{
					NodeID:      "",
					ServiceName: pmmapitests.TestString(t, "External Service with empty node id"),
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddExternalServiceParams.NodeId: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.External.ServiceID)
		}
	})

	t.Run("AddServiceNameEmpty", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				External: &services.AddServiceParamsBodyExternal{
					NodeID:      genericNodeID,
					ServiceName: "",
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddExternalServiceParams.ServiceName: value length must be at least 1 runes")
		if !assert.Nil(t, res) {
			pmmapitests.RemoveServices(t, res.Payload.External.ServiceID)
		}
	})

	t.Run("AddServiceWithOutGroup", func(t *testing.T) {
		t.Parallel()

		genericNodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "")).NodeID
		require.NotEmpty(t, genericNodeID)
		defer pmmapitests.RemoveNodes(t, genericNodeID)

		serviceName := pmmapitests.TestString(t, "Basic External Service")
		params := &services.AddServiceParams{
			Body: services.AddServiceBody{
				External: &services.AddServiceParamsBodyExternal{
					NodeID:      genericNodeID,
					ServiceName: serviceName,
				},
			},
			Context: pmmapitests.Context,
		}
		res, err := client.Default.ServicesService.AddService(params)
		require.NoError(t, err)
		require.NotNil(t, res)
		serviceID := res.Payload.External.ServiceID
		assert.Equal(t, &services.AddServiceOK{
			Payload: &services.AddServiceOKBody{
				External: &services.AddServiceOKBodyExternal{
					ServiceID:    serviceID,
					NodeID:       genericNodeID,
					ServiceName:  serviceName,
					Group:        "external",
					CustomLabels: map[string]string{},
				},
			},
		}, res)
		defer pmmapitests.RemoveServices(t, serviceID)
	})
}
