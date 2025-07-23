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

func TestMongodbExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndTLS", func(t *testing.T) {
			t.Parallel()

			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-id", `{"mongodb_exporter": {"agent_id": "test-agent-id"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentMongodbExporterCommand{
				AgentID:                 "test-agent-id",
				Enable:                  pointer.ToBool(true),
				Username:                pointer.ToString("newuser"),
				Password:                pointer.ToString("newpass"),
				TLS:                     pointer.ToBool(true),
				TLSSkipVerify:           pointer.ToBool(false),
				AuthenticationMechanism: pointer.ToString("SCRAM-SHA-256"),
				AuthenticationDatabase:  pointer.ToString("admin"),
				StatsCollections:        pointer.ToString("collection1,collection2"),
				CollectionsLimit:        pointer.ToInt32(100),
				DisableCollectors:       []string{"general_stats", "index_stats"},
				ExposeExporter:          pointer.ToBool(true),
				PushMetrics:             pointer.ToBool(false),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"environment": "test", "team": "backend"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"mongodb_exporter": {
					"enable": true,
					"username": "newuser",
					"password": "newpass",
					"tls": true,
					"tls_skip_verify": false,
					"authentication_mechanism": "SCRAM-SHA-256",
					"authentication_database": "admin",
					"stats_collections": ["collection1", "collection2"],
					"collections_limit": 100,
					"disable_collectors": ["general_stats", "index_stats"],
					"expose_exporter": true,
					"enable_push_metrics": false,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "test",
							"team": "backend"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgentWithCollectors", func(t *testing.T) {
			t.Parallel()

			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-id", `{"mongodb_exporter": {"agent_id": "test-agent-id"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentMongodbExporterCommand{
				AgentID:           "test-agent-id",
				Enable:            pointer.ToBool(false),
				DisableCollectors: []string{"general_stats"},
				PushMetrics:       pointer.ToBool(true),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"mongodb_exporter": {
					"enable": false,
					"disable_collectors": ["general_stats"],
					"enable_push_metrics": true,
					"stats_collections": null
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"mongodb_exporter": {
				"agent_id": "test-agent-id",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "mongodb-service-456",
				"username": "testuser",
				"listen_port": 9216,
				"tls": true,
				"tls_skip_verify": true,
				"push_metrics_enabled": true,
				"expose_exporter": true,
				"disabled": false,
				"custom_labels": {
					"env": "prod",
					"team": "db"
				},
				"process_exec_path": "/usr/bin/mongodb_exporter",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-id", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "mongodb-exporter", "test-agent-id",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--agent-password=agentpass",
			"--tls",
			"--tls-skip-verify",
			"--authentication-mechanism=SCRAM-SHA-1",
			"--authentication-database=testdb",
			"--stats-collections=col1,col2",
			"--collections-limit=50",
			"--disable-collectors=general_stats,index_stats",
			"--expose-exporter",
			"--push-metrics",
			"--log-level=info",
			"--custom-labels=env=prod,team=db",
		}

		var cmd ChangeAgentMongodbExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"mongodb_exporter": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"agent_password": "agentpass",
				"tls": true,
				"tls_skip_verify": true,
				"authentication_mechanism": "SCRAM-SHA-1",
				"authentication_database": "testdb",
				"stats_collections": ["col1", "col2"],
				"collections_limit": 50,
				"disable_collectors": ["general_stats", "index_stats"],
				"expose_exporter": true,
				"enable_push_metrics": true,
				"log_level": "LOG_LEVEL_INFO",
				"custom_labels": {
					"values": {
						"env": "prod",
						"team": "db"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `MongoDB Exporter agent configuration updated.
Agent ID              : test-agent-id
PMM-Agent ID          : pmm-agent-123
Service ID            : mongodb-service-456
Username              : testuser
Listen port           : 9216
TLS enabled           : true
Skip TLS verification : true
Push metrics enabled  : true
Expose exporter       : true

Disabled              : false
Custom labels         : env=prod, team=db
Process exec path     : /usr/bin/mongodb_exporter
Log level             : info
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - updated agent password
  - enabled TLS
  - enabled TLS skip verification
  - changed authentication mechanism to SCRAM-SHA-1
  - changed authentication database to testdb
  - updated stats collections: col1,col2
  - changed collections limit to 50
  - updated disabled collectors: [general_stats index_stats]
  - enabled expose exporter
  - enabled push metrics
  - changed log level to info
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentMongodbExporterCommand{
			AgentID: "invalid-agent",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		t.Parallel()

		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-id", `{"mongodb_exporter": {"agent_id": "test-agent-id"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "mongodb-exporter", "test-agent-id", "--enable"}

		var cmd ChangeAgentMongodbExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"mongodb_exporter": {
				"enable": true,
				"disable_collectors": null,
				"stats_collections": null
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "mongodb-exporter"}

			var cmd ChangeAgentMongodbExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "mongodb-exporter", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentMongodbExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
