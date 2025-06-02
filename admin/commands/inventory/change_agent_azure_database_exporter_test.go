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
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
)

func TestAzureDatabaseExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-azure-update", `{"azure_database_exporter": {"agent_id": "test-agent-azure-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentAzureDatabaseExporterCommand{
				AgentID:             "test-agent-azure-update",
				Enable:              pointer.ToBool(true),
				AzureClientID:       pointer.ToString("12345678-1234-1234-1234-123456789012"),
				AzureClientSecret:   pointer.ToString("secret123"),
				AzureTenantID:       pointer.ToString("87654321-4321-4321-4321-210987654321"),
				AzureSubscriptionID: pointer.ToString("11111111-2222-3333-4444-555555555555"),
				AzureResourceGroup:  pointer.ToString("pmm-rg"),
				PushMetrics:         pointer.ToBool(true),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"environment": "production", "cloud": "azure"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"azure_database_exporter": {
					"enable": true,
					"azure_client_id": "12345678-1234-1234-1234-123456789012",
					"azure_client_secret": "secret123",
					"azure_tenant_id": "87654321-4321-4321-4321-210987654321",
					"azure_subscription_id": "11111111-2222-3333-4444-555555555555",
					"azure_resource_group": "pmm-rg",
					"enable_push_metrics": true,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "production",
							"cloud": "azure"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-azure-disable", `{"azure_database_exporter": {"agent_id": "test-agent-azure-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentAzureDatabaseExporterCommand{
				AgentID:     "test-agent-azure-disable",
				Enable:      pointer.ToBool(false),
				PushMetrics: pointer.ToBool(false),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"azure_database_exporter": {
					"enable": false,
					"enable_push_metrics": false
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"azure_database_exporter": {
				"agent_id": "test-agent-azure-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"node_id": "node-456",
				"azure_database_subscription_id": "test-subscription-id",
				"azure_database_resource_type": "mysql",
				"listen_port": 9090,
				"push_metrics_enabled": true,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "azure"
				},
				"process_exec_path": "/usr/bin/azure_metrics_exporter",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-azure-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "azure-database-exporter", "test-agent-azure-all-flags",
			"--enable",
			"--azure-client-id=test-client-id",
			"--azure-client-secret=test-secret",
			"--azure-tenant-id=test-tenant-id",
			"--azure-subscription-id=test-subscription-id",
			"--azure-resource-group=test-rg",
			"--push-metrics",
			"--log-level=info",
			"--custom-labels=env=test,team=azure",
		}

		var cmd ChangeAgentAzureDatabaseExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"azure_database_exporter": {
				"enable": true,
				"azure_client_id": "test-client-id",
				"azure_client_secret": "test-secret",
				"azure_tenant_id": "test-tenant-id",
				"azure_subscription_id": "test-subscription-id",
				"azure_resource_group": "test-rg",
				"enable_push_metrics": true,
				"log_level": "LOG_LEVEL_INFO",
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "azure"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `Azure Database Exporter agent configuration updated.
Agent ID                    : test-agent-azure-all-flags
PMM-Agent ID                : pmm-agent-123
Node ID                     : node-456
Azure Subscription ID      : test-subscription-id
Azure Resource Type        : mysql
Listen port                 : 9090
Push metrics enabled        : true

Disabled                    : false
Custom labels               : env=test, team=azure
Process exec path           : /usr/bin/azure_metrics_exporter
Log level                   : info
Configuration changes applied:
  - enabled agent
  - updated azure_client_id
  - updated azure_client_secret
  - updated azure_tenant_id
  - updated azure_subscription_id
  - updated azure_resource_group
  - enabled push metrics
  - changed log level to info
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-azure", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentAzureDatabaseExporterCommand{
			AgentID: "invalid-agent-azure",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-azure-minimal", `{"azure_database_exporter": {"agent_id": "test-agent-azure-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "azure-database-exporter", "test-agent-azure-minimal"}

		var cmd ChangeAgentAzureDatabaseExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty azure_database_exporter object when no flags are set
		expectedJSON := `{
			"azure_database_exporter": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingWithEnableOnly", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-azure-enable-only", `{"azure_database_exporter": {"agent_id": "test-agent-azure-enable-only"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "azure-database-exporter", "test-agent-azure-enable-only", "--enable"}

		var cmd ChangeAgentAzureDatabaseExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should only include enable field when only enable is set
		expectedJSON := `{
			"azure_database_exporter": {
				"enable": true
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "azure-database-exporter"}

			var cmd ChangeAgentAzureDatabaseExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "azure-database-exporter", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentAzureDatabaseExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
