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
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
)

const (
	// The pmm-agent of PMM Server. Unlike a pmm-agent added through the API, it is connected and has
	// a version, which credentials stored in Vault require.
	pmmServerAgentID = "pmm-server"

	vaultPath      = "secret/data/pmm/api-tests"
	otherVaultPath = "secret/data/pmm/api-tests-other"
)

// vaultAgentType is an Agent type that connects to a database, so its credentials can be stored in Vault.
type vaultAgentType struct {
	// field is the name of the Agent type in the request and response bodies.
	field string
	// service is the name of the type of the Service it monitors in the services request body.
	service string
	// params is the name of its Add message, as validation errors give it.
	params string
	// usernameRequired is whether it needs a username when the credentials are not stored in Vault.
	usernameRequired bool
}

var vaultAgentTypes = []vaultAgentType{ //nolint:gochecknoglobals
	{field: "MysqldExporter", service: "Mysql", params: "AddMySQLdExporterParams", usernameRequired: true},
	{field: "QANMysqlPerfschemaAgent", service: "Mysql", params: "AddQANMySQLPerfSchemaAgentParams", usernameRequired: true},
	{field: "QANMysqlSlowlogAgent", service: "Mysql", params: "AddQANMySQLSlowlogAgentParams", usernameRequired: true},
	{field: "MongodbExporter", service: "Mongodb", params: "AddMongoDBExporterParams"},
	{field: "QANMongodbProfilerAgent", service: "Mongodb", params: "AddQANMongoDBProfilerAgentParams"},
	{field: "QANMongodbMongologAgent", service: "Mongodb", params: "AddQANMongoDBMongologAgentParams"},
	{field: "RtaMongodbAgent", service: "Mongodb", params: "AddRTAMongoDBAgentParams"},
	{field: "PostgresExporter", service: "Postgresql", params: "AddPostgresExporterParams", usernameRequired: true},
	{field: "QANPostgresqlPgstatementsAgent", service: "Postgresql", params: "AddQANPostgreSQLPgStatementsAgentParams", usernameRequired: true},
	{field: "QANPostgresqlPgstatmonitorAgent", service: "Postgresql", params: "AddQANPostgreSQLPgStatMonitorAgentParams", usernameRequired: true},
	{field: "ProxysqlExporter", service: "Proxysql", params: "AddProxySQLExporterParams", usernameRequired: true},
	{field: "ValkeyExporter", service: "Valkey", params: "AddValkeyExporterParams", usernameRequired: true},
}

// setOneOf sets the field of body, a pointer to a request body, to a new message with the given fields.
// The request and response bodies have one field per Agent or Service type, all with the same fields
// of their own, so the tests below cover every type the same way.
func setOneOf(t *testing.T, body any, field string, values map[string]any) {
	t.Helper()

	f := reflect.ValueOf(body).Elem().FieldByName(field)
	require.True(t, f.IsValid(), "%T has no field %s", body, field)

	msg := reflect.New(f.Type().Elem())
	for name, value := range values {
		mf := msg.Elem().FieldByName(name)
		require.True(t, mf.IsValid(), "%s has no field %s", msg.Type(), name)

		v := reflect.ValueOf(value)
		if mf.Kind() == reflect.Pointer && v.Kind() != reflect.Pointer {
			p := reflect.New(v.Type())
			p.Elem().Set(v)
			v = p
		}
		mf.Set(v)
	}
	f.Set(msg)
}

// oneOfField returns the field name of the message in the field of body, a pointer to a response body.
func oneOfField(t *testing.T, body any, field, name string) any {
	t.Helper()

	f := reflect.ValueOf(body).Elem().FieldByName(field)
	require.True(t, f.IsValid(), "%T has no field %s", body, field)
	require.False(t, f.IsNil(), "%T has no %s", body, field)

	return f.Elem().FieldByName(name).Interface()
}

// addVaultTestService adds a Service of the given type, on a Node of its own, for the Agents of a test.
func addVaultTestService(t *testing.T, serviceType string) string {
	t.Helper()

	nodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Vault node")).NodeID

	var body services.AddServiceBody
	setOneOf(t, &body, serviceType, map[string]any{
		"NodeID":      nodeID,
		"ServiceName": pmmapitests.TestString(t, "Vault "+serviceType+" service"),
		"Address":     "10.10.10.10",
		"Port":        int64(1234),
	})
	res := pmmapitests.AddService(t, body)

	return oneOfField(t, res, serviceType, "ServiceID").(string)
}

func addAgentRequest(t *testing.T, at vaultAgentType, values map[string]any) *agents.AddAgentParams {
	t.Helper()

	values["SkipConnectionCheck"] = true

	var body agents.AddAgentBody
	setOneOf(t, &body, at.field, values)

	return &agents.AddAgentParams{Body: body, Context: pmmapitests.Context}
}

func changeAgent(t *testing.T, at vaultAgentType, agentID string, values map[string]any) (*agents.ChangeAgentOK, error) {
	t.Helper()

	values["SkipConnectionCheck"] = true

	var body agents.ChangeAgentBody
	setOneOf(t, &body, at.field, values)

	return client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
		AgentID: agentID,
		Body:    body,
		Context: pmmapitests.Context,
	})
}

func TestAgentsWithCredentialsInVault(t *testing.T) {
	t.Parallel()

	for _, at := range vaultAgentTypes {
		t.Run(at.field, func(t *testing.T) {
			t.Parallel()

			serviceID := addVaultTestService(t, at.service)

			t.Run("AddGetListChange", func(t *testing.T) {
				// The username stays out of the request: the Vault secret holds it.
				added := pmmapitests.AddAgent(t, addAgentRequest(t, at, map[string]any{
					"PMMAgentID": pmmServerAgentID,
					"ServiceID":  serviceID,
					"VaultPath":  "/" + vaultPath + "/",
				}).Body)
				agentID := oneOfField(t, added, at.field, "AgentID").(string)
				assert.Equal(t, vaultPath, oneOfField(t, added, at.field, "VaultPath"), "the path is stored without surrounding slashes")
				assert.Empty(t, oneOfField(t, added, at.field, "Username"))

				got, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{AgentID: agentID, Context: pmmapitests.Context})
				require.NoError(t, err)
				assert.Equal(t, vaultPath, oneOfField(t, got.Payload, at.field, "VaultPath"))

				list, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{ServiceID: &serviceID, Context: pmmapitests.Context})
				require.NoError(t, err)
				listed := reflect.ValueOf(list.Payload).Elem().FieldByName(at.field)
				require.Equal(t, 1, listed.Len())
				assert.Equal(t, vaultPath, listed.Index(0).Elem().FieldByName("VaultPath").Interface())

				// A password replaces the Vault path.
				changed, err := changeAgent(t, at, agentID, map[string]any{"Username": "username", "Password": "password"})
				require.NoError(t, err)
				assert.Empty(t, oneOfField(t, changed.Payload, at.field, "VaultPath"))
				assert.Equal(t, "username", oneOfField(t, changed.Payload, at.field, "Username"))

				// A Vault path replaces the password.
				changed, err = changeAgent(t, at, agentID, map[string]any{"VaultPath": otherVaultPath})
				require.NoError(t, err)
				assert.Equal(t, otherVaultPath, oneOfField(t, changed.Payload, at.field, "VaultPath"))

				// A change that does not touch the credentials keeps them.
				changed, err = changeAgent(t, at, agentID, map[string]any{"Enable": false})
				require.NoError(t, err)
				assert.Equal(t, otherVaultPath, oneOfField(t, changed.Payload, at.field, "VaultPath"))

				_, err = changeAgent(t, at, agentID, map[string]any{"Password": "password", "VaultPath": vaultPath})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Password and Vault path are mutually exclusive: the password is read from Vault.")

				_, err = changeAgent(t, at, agentID, map[string]any{"VaultPath": "secret/../pmm"})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "vault path 'secret/../pmm' has an empty or relative segment")

				// An empty Vault path stops reading the credentials from Vault.
				changed, err = changeAgent(t, at, agentID, map[string]any{"VaultPath": new("")})
				require.NoError(t, err)
				assert.Empty(t, oneOfField(t, changed.Payload, at.field, "VaultPath"))
			})

			t.Run("AddWithPassword", func(t *testing.T) {
				res, err := client.Default.AgentsService.AddAgent(addAgentRequest(t, at, map[string]any{
					"PMMAgentID": pmmServerAgentID,
					"ServiceID":  serviceID,
					"Username":   "username",
					"Password":   "password",
					"VaultPath":  vaultPath,
				}))
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, "Password and Vault path are mutually exclusive: the password is read from Vault.")
				assert.Nil(t, res)
			})

			t.Run("AddWithInvalidPath", func(t *testing.T) {
				for path, message := range map[string]string{
					"/":                   "vault path must not be empty",
					"secret/../pmm":       "vault path 'secret/../pmm' has an empty or relative segment",
					"secret//pmm":         "vault path 'secret//pmm' has an empty or relative segment",
					"secret/data/pmm?x=1": "vault path 'secret/data/pmm?x=1' must not contain whitespace, '?' or '#'",
				} {
					res, err := client.Default.AgentsService.AddAgent(addAgentRequest(t, at, map[string]any{
						"PMMAgentID": pmmServerAgentID,
						"ServiceID":  serviceID,
						"VaultPath":  path,
					}))
					pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, message)
					assert.Nil(t, res, path)
				}
			})

			t.Run("AddWithoutUsername", func(t *testing.T) {
				if !at.usernameRequired {
					t.Skip("The username is optional for this Agent type.")
				}

				res, err := client.Default.AgentsService.AddAgent(addAgentRequest(t, at, map[string]any{
					"PMMAgentID": pmmServerAgentID,
					"ServiceID":  serviceID,
					"Password":   "password",
				}))
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.InvalidArgument, fmt.Sprintf("invalid %s.Username: value length must be at least 1 runes", at.params))
				assert.Nil(t, res)
			})

			t.Run("PMMAgentWithoutVersion", func(t *testing.T) {
				// It never connected, so it may be too old to read credentials from Vault.
				nodeID := pmmapitests.AddGenericNode(t, pmmapitests.TestString(t, "Vault node of an old pmm-agent")).NodeID
				pmmAgentID := pmmapitests.AddPMMAgent(t, nodeID).AgentID

				res, err := client.Default.AgentsService.AddAgent(addAgentRequest(t, at, map[string]any{
					"PMMAgentID": pmmAgentID,
					"ServiceID":  serviceID,
					"VaultPath":  vaultPath,
				}))
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, fmt.Sprintf("pmm agent %q has no version info", pmmAgentID))
				assert.Nil(t, res)

				// Its Agents are not affected otherwise, but cannot be moved to Vault.
				added := pmmapitests.AddAgent(t, addAgentRequest(t, at, map[string]any{
					"PMMAgentID": pmmAgentID,
					"ServiceID":  serviceID,
					"Username":   "username",
					"Password":   "password",
				}).Body)
				agentID := oneOfField(t, added, at.field, "AgentID").(string)

				_, err = changeAgent(t, at, agentID, map[string]any{"VaultPath": vaultPath})
				pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, fmt.Sprintf("pmm agent %q has no version info", pmmAgentID))

				changed, err := changeAgent(t, at, agentID, map[string]any{"Enable": false})
				require.NoError(t, err)
				assert.Empty(t, oneOfField(t, changed.Payload, at.field, "VaultPath"))
			})
		})
	}
}
