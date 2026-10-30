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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

const (
	// The Node and the pmm-agent of PMM Server. Unlike a registered pmm-agent that never connected,
	// this pmm-agent is connected and has a version, which credentials stored in Vault require.
	pmmServerID = "pmm-server"

	vaultPath = "secret/data/pmm/api-tests"

	vaultStoreError = "Failed to store the credentials in Vault: "
)

// storedInVault reports whether the credentials a request gave were stored in Vault. Only a PMM Server whose
// own pmm-agent has Vault configured stores them; anywhere else the request is refused, changing nothing.
func storedInVault(t *testing.T, err error) bool {
	t.Helper()

	if err == nil {
		return true
	}
	pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, vaultStoreError)
	return false
}

// removeAddedService removes the Service an AddService response describes, whichever its type is.
func removeAddedService(t *testing.T, res *mservice.AddServiceOK) {
	t.Helper()

	for _, f := range reflect.ValueOf(res.Payload).Elem().Fields() {
		if f.Kind() != reflect.Pointer || f.IsNil() {
			continue
		}
		for _, name := range []string{"Service", "Mysql", "Postgresql"} {
			service := f.Elem().FieldByName(name)
			if service.IsValid() && !service.IsNil() {
				t.Cleanup(func() { pmmapitests.RemoveServices(t, service.Elem().FieldByName("ServiceID").String()) })
			}
		}
		node := f.Elem().FieldByName("Node")
		if node.IsValid() && !node.IsNil() {
			t.Cleanup(func() { pmmapitests.RemoveNodes(t, node.Elem().FieldByName("NodeID").String()) })
		}
	}
}

func addService(t *testing.T, body mservice.AddServiceBody) (*mservice.AddServiceOK, error) {
	t.Helper()

	return client.Default.ManagementService.AddService(&mservice.AddServiceParams{Body: body, Context: pmmapitests.Context})
}

// assertAgentsInVault checks that every Agent of the Service has the Vault path and no stored credentials.
func assertAgentsInVault(t *testing.T, serviceID string, expected int) {
	t.Helper()

	res, err := client.Default.ManagementService.ListAgents(&mservice.ListAgentsParams{ServiceID: &serviceID, Context: pmmapitests.Context})
	require.NoError(t, err)

	var found int
	for _, agent := range res.Payload.Agents {
		if agent.AgentType == "pmm-agent" {
			continue
		}
		found++
		assert.Equal(t, vaultPath, agent.VaultPath, agent.AgentType)
		assert.False(t, agent.IsPasswordSet, agent.AgentType)
		assert.Empty(t, agent.Username, agent.AgentType)
	}
	assert.Equal(t, expected, found)
}

func TestAddServicesWithCredentialsInVault(t *testing.T) {
	t.Parallel()

	t.Run("MySQL", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, mservice.AddServiceBody{Mysql: &mservice.AddServiceParamsBodyMysql{
			NodeID:              pmmServerID,
			PMMAgentID:          pmmServerID,
			ServiceName:         pmmapitests.TestString(t, "vault-mysql"),
			Address:             "10.10.10.10",
			Port:                3306,
			VaultPath:           vaultPath,
			QANMysqlPerfschema:  true,
			QANMysqlSlowlog:     true,
			SkipConnectionCheck: true,
		}})
		require.NoError(t, err)
		body := res.Payload.Mysql
		t.Cleanup(func() { pmmapitests.RemoveServices(t, body.Service.ServiceID) })

		assert.Equal(t, vaultPath, body.MysqldExporter.VaultPath)
		assert.Equal(t, vaultPath, body.QANMysqlPerfschema.VaultPath)
		assert.Equal(t, vaultPath, body.QANMysqlSlowlog.VaultPath)
		assertAgentsInVault(t, body.Service.ServiceID, 3)
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, mservice.AddServiceBody{Postgresql: &mservice.AddServiceParamsBodyPostgresql{
			NodeID:                          pmmServerID,
			PMMAgentID:                      pmmServerID,
			ServiceName:                     pmmapitests.TestString(t, "vault-postgresql"),
			Address:                         "10.10.10.10",
			Port:                            5432,
			VaultPath:                       vaultPath,
			QANPostgresqlPgstatementsAgent:  true,
			QANPostgresqlPgstatmonitorAgent: true,
			SkipConnectionCheck:             true,
		}})
		require.NoError(t, err)
		body := res.Payload.Postgresql
		t.Cleanup(func() { pmmapitests.RemoveServices(t, body.Service.ServiceID) })

		assert.Equal(t, vaultPath, body.PostgresExporter.VaultPath)
		assert.Equal(t, vaultPath, body.QANPostgresqlPgstatementsAgent.VaultPath)
		assert.Equal(t, vaultPath, body.QANPostgresqlPgstatmonitorAgent.VaultPath)
		assertAgentsInVault(t, body.Service.ServiceID, 3)
	})

	t.Run("MongoDB", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, mservice.AddServiceBody{Mongodb: &mservice.AddServiceParamsBodyMongodb{
			NodeID:              pmmServerID,
			PMMAgentID:          pmmServerID,
			ServiceName:         pmmapitests.TestString(t, "vault-mongodb"),
			Address:             "10.10.10.10",
			Port:                27017,
			VaultPath:           vaultPath,
			QANMongodbProfiler:  true,
			QANMongodbMongolog:  true,
			RtaMongodbAgent:     true,
			SkipConnectionCheck: true,
		}})
		require.NoError(t, err)
		body := res.Payload.Mongodb
		t.Cleanup(func() { pmmapitests.RemoveServices(t, body.Service.ServiceID) })

		assert.Equal(t, vaultPath, body.MongodbExporter.VaultPath)
		assert.Equal(t, vaultPath, body.QANMongodbProfiler.VaultPath)
		assert.Equal(t, vaultPath, body.QANMongodbMongolog.VaultPath)
		assert.Equal(t, vaultPath, body.RtaMongodbAgent.VaultPath)
		assertAgentsInVault(t, body.Service.ServiceID, 4)
	})

	t.Run("ProxySQL", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, mservice.AddServiceBody{Proxysql: &mservice.AddServiceParamsBodyProxysql{
			NodeID:              pmmServerID,
			PMMAgentID:          pmmServerID,
			ServiceName:         pmmapitests.TestString(t, "vault-proxysql"),
			Address:             "10.10.10.10",
			Port:                6032,
			VaultPath:           vaultPath,
			SkipConnectionCheck: true,
		}})
		require.NoError(t, err)
		body := res.Payload.Proxysql
		t.Cleanup(func() { pmmapitests.RemoveServices(t, body.Service.ServiceID) })

		assert.Equal(t, vaultPath, body.ProxysqlExporter.VaultPath)
		assertAgentsInVault(t, body.Service.ServiceID, 1)
	})

	t.Run("Valkey", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, mservice.AddServiceBody{Valkey: &mservice.AddServiceParamsBodyValkey{
			NodeID:              pmmServerID,
			PMMAgentID:          pmmServerID,
			ServiceName:         pmmapitests.TestString(t, "vault-valkey"),
			Address:             "10.10.10.10",
			Port:                6379,
			VaultPath:           vaultPath,
			SkipConnectionCheck: true,
		}})
		require.NoError(t, err)
		body := res.Payload.Valkey
		t.Cleanup(func() { pmmapitests.RemoveServices(t, body.Service.ServiceID) })

		assert.Equal(t, vaultPath, body.ValkeyExporter.VaultPath)
		assertAgentsInVault(t, body.Service.ServiceID, 1)
	})
}

func TestAddServicesWithPasswordAndVaultPath(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]func(t *testing.T, password, vaultPath string) mservice.AddServiceBody{
		"MySQL": func(t *testing.T, password, vaultPath string) mservice.AddServiceBody {
			t.Helper()
			return mservice.AddServiceBody{Mysql: &mservice.AddServiceParamsBodyMysql{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-mysql"),
				Address: "10.10.10.10", Port: 3306, Username: "username", Password: password, VaultPath: vaultPath, SkipConnectionCheck: true,
			}}
		},
		"PostgreSQL": func(t *testing.T, password, vaultPath string) mservice.AddServiceBody {
			t.Helper()
			return mservice.AddServiceBody{Postgresql: &mservice.AddServiceParamsBodyPostgresql{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-postgresql"),
				Address: "10.10.10.10", Port: 5432, Username: "username", Password: password, VaultPath: vaultPath, SkipConnectionCheck: true,
			}}
		},
		"MongoDB": func(t *testing.T, password, vaultPath string) mservice.AddServiceBody {
			t.Helper()
			return mservice.AddServiceBody{Mongodb: &mservice.AddServiceParamsBodyMongodb{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-mongodb"),
				Address: "10.10.10.10", Port: 27017, Username: "username", Password: password, VaultPath: vaultPath, SkipConnectionCheck: true,
			}}
		},
		"ProxySQL": func(t *testing.T, password, vaultPath string) mservice.AddServiceBody {
			t.Helper()
			return mservice.AddServiceBody{Proxysql: &mservice.AddServiceParamsBodyProxysql{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-proxysql"),
				Address: "10.10.10.10", Port: 6032, Username: "username", Password: password, VaultPath: vaultPath, SkipConnectionCheck: true,
			}}
		},
		"Valkey": func(t *testing.T, password, vaultPath string) mservice.AddServiceBody {
			t.Helper()
			return mservice.AddServiceBody{Valkey: &mservice.AddServiceParamsBodyValkey{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-valkey"),
				Address: "10.10.10.10", Port: 6379, Username: "username", Password: password, VaultPath: vaultPath, SkipConnectionCheck: true,
			}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The password goes to Vault, and PMM Server stores the path only.
			res, err := addService(t, body(t, "password", vaultPath))
			if storedInVault(t, err) {
				removeAddedService(t, res)
			} else {
				assert.Nil(t, res)
			}

			res, err = addService(t, body(t, "", "secret/../pmm"))
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "vault path 'secret/../pmm' has an empty or relative segment")
			assert.Nil(t, res)
		})
	}

	t.Run("UsernameRequiredWithoutVault", func(t *testing.T) {
		t.Parallel()

		for message, body := range map[string]mservice.AddServiceBody{
			"invalid AddMySQLServiceParams.Username: value length must be at least 1 runes": {Mysql: &mservice.AddServiceParamsBodyMysql{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-mysql"),
				Address: "10.10.10.10", Port: 3306, Password: "password", SkipConnectionCheck: true,
			}},
			"invalid AddPostgreSQLServiceParams.Username: value length must be at least 1 runes": {Postgresql: &mservice.AddServiceParamsBodyPostgresql{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-postgresql"),
				Address: "10.10.10.10", Port: 5432, Password: "password", SkipConnectionCheck: true,
			}},
			"invalid AddProxySQLServiceParams.Username: value length must be at least 1 runes": {Proxysql: &mservice.AddServiceParamsBodyProxysql{
				NodeID: pmmServerID, PMMAgentID: pmmServerID, ServiceName: pmmapitests.TestString(t, "vault-proxysql"),
				Address: "10.10.10.10", Port: 6032, Password: "password", SkipConnectionCheck: true,
			}},
		} {
			res, err := addService(t, body)
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, message)
			assert.Nil(t, res)
		}
	})

	t.Run("PMMAgentWithoutVersion", func(t *testing.T) {
		t.Parallel()

		// A registered pmm-agent that never connected may be too old to read credentials from Vault.
		nodeID, pmmAgentID := RegisterNode(t, mservice.RegisterNodeBody{
			NodeName: pmmapitests.TestString(t, "vault-node"),
			NodeType: new(mservice.RegisterNodeBodyNodeTypeNODETYPEGENERICNODE),
		})

		res, err := addService(t, mservice.AddServiceBody{Mysql: &mservice.AddServiceParamsBodyMysql{
			NodeID:              nodeID,
			PMMAgentID:          pmmAgentID,
			ServiceName:         pmmapitests.TestString(t, "vault-mysql"),
			Address:             "10.10.10.10",
			Port:                3306,
			VaultPath:           vaultPath,
			SkipConnectionCheck: true,
		}})
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, fmt.Sprintf("pmm agent %q has no version info", pmmAgentID))
		assert.Nil(t, res)
	})
}

func TestAddRDSWithCredentialsInVault(t *testing.T) {
	t.Parallel()

	rdsBody := func(t *testing.T, engine, username, password, vaultPath string) mservice.AddServiceBody {
		t.Helper()

		return mservice.AddServiceBody{RDS: &mservice.AddServiceParamsBodyRDS{
			Region:                    pmmapitests.TestString(t, "region"),
			Az:                        "az",
			InstanceID:                pmmapitests.TestString(t, "vault-instance"),
			Address:                   pmmapitests.TestString(t, "vault.example.rds"),
			Port:                      3306,
			Engine:                    new(engine),
			NodeName:                  pmmapitests.TestString(t, "vault-rds-node"),
			ServiceName:               pmmapitests.TestString(t, "vault-rds-service"),
			Username:                  username,
			Password:                  password,
			VaultPath:                 vaultPath,
			AWSAccessKey:              "my-aws-access-key",
			AWSSecretKey:              "my-aws-secret-key",
			RDSExporter:               true,
			QANMysqlPerfschema:        true,
			QANPostgresqlPgstatements: true,
			SkipConnectionCheck:       true,
		}}
	}

	t.Run("MySQL", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, rdsBody(t, "DISCOVER_RDS_ENGINE_MYSQL", "", "", vaultPath))
		require.NoError(t, err)
		body := res.Payload.RDS
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, body.Mysql.ServiceID)
			pmmapitests.RemoveNodes(t, body.Node.NodeID)
		})

		assert.Equal(t, vaultPath, body.MysqldExporter.VaultPath)
		assert.Equal(t, vaultPath, body.QANMysqlPerfschema.VaultPath)
		assertAgentsInVault(t, body.Mysql.ServiceID, 2)
	})

	t.Run("PostgreSQL", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, rdsBody(t, "DISCOVER_RDS_ENGINE_POSTGRESQL", "", "", vaultPath))
		require.NoError(t, err)
		body := res.Payload.RDS
		t.Cleanup(func() {
			pmmapitests.RemoveServices(t, body.Postgresql.ServiceID)
			pmmapitests.RemoveNodes(t, body.Node.NodeID)
		})

		assert.Equal(t, vaultPath, body.PostgresqlExporter.VaultPath)
		assert.Equal(t, vaultPath, body.QANPostgresqlPgstatements.VaultPath)
		assertAgentsInVault(t, body.Postgresql.ServiceID, 2)
	})

	t.Run("Errors", func(t *testing.T) {
		t.Parallel()

		res, err := addService(t, rdsBody(t, "DISCOVER_RDS_ENGINE_MYSQL", "username", "password", vaultPath))
		if storedInVault(t, err) {
			removeAddedService(t, res)
		} else {
			assert.Nil(t, res)
		}

		res, err = addService(t, rdsBody(t, "DISCOVER_RDS_ENGINE_MYSQL", "", "password", ""))
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddRDSServiceParams.Username: value length must be at least 1 runes")
		assert.Nil(t, res)
	})
}

func TestAddAzureDatabaseWithCredentialsInVault(t *testing.T) {
	t.Parallel()

	serviceName := pmmapitests.TestString(t, "vault-azure-service")
	request := func(vaultPath string) *mservice.AddAzureDatabaseParams {
		return &mservice.AddAzureDatabaseParams{
			Body: mservice.AddAzureDatabaseBody{
				Region:              "region",
				Az:                  "az",
				AzureClientID:       "azure-client-id",
				AzureClientSecret:   "azure-client-secret",
				AzureTenantID:       "azure-tenant-id",
				AzureSubscriptionID: "azure-subscription-id",
				AzureResourceGroup:  "azure-resource-group",
				InstanceID:          pmmapitests.TestString(t, "vault-azure-instance"),
				Address:             pmmapitests.TestString(t, "vault.mysql.database.azure.com"),
				Port:                3306,
				VaultPath:           vaultPath,
				NodeName:            pmmapitests.TestString(t, "vault-azure-node"),
				ServiceName:         serviceName,
				Type:                new("DISCOVER_AZURE_DATABASE_TYPE_MYSQL"),
				SkipConnectionCheck: true,
			},
			Context: pmmapitests.Context,
		}
	}

	_, err := client.Default.ManagementService.AddAzureDatabase(request(""))
	pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "invalid AddAzureDatabaseRequest.Username: value length must be at least 1 runes")

	settings, err := serverClient.Default.ServerService.GetSettings(&server.GetSettingsParams{Context: pmmapitests.Context})
	require.NoError(t, err)

	// A Vault path stands in for the username, so the request gets as far as the Azure settings.
	if !settings.Payload.Settings.AzurediscoverEnabled {
		// pmm-managed answers its "azure monitoring is disabled" error as an internal one.
		_, err = client.Default.ManagementService.AddAzureDatabase(request(vaultPath))
		pmmapitests.AssertAPIErrorf(t, err, 500, codes.Internal, "Internal server error.")
		return
	}

	_, err = client.Default.ManagementService.AddAzureDatabase(request(vaultPath))
	require.NoError(t, err)

	list, err := inventoryClient.Default.ServicesService.ListServices(&services.ListServicesParams{Context: pmmapitests.Context})
	require.NoError(t, err)
	var serviceID, nodeID string
	for _, s := range list.Payload.Mysql {
		if s.ServiceName == serviceName {
			serviceID, nodeID = s.ServiceID, s.NodeID
		}
	}
	require.NotEmpty(t, serviceID)
	t.Cleanup(func() {
		pmmapitests.RemoveServices(t, serviceID)
		pmmapitests.RemoveNodes(t, nodeID)
	})

	assertAgentsInVault(t, serviceID, 1)
}
