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

func TestValkeyExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndTLS", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-valkey-update", `{"valkey_exporter": {"agent_id": "test-agent-valkey-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentValkeyExporterCommand{
				AgentID:     "test-agent-valkey-update",
				Enable:      pointer.ToBool(true),
				Username:    pointer.ToString("redis_user"),
				Password:    pointer.ToString("redis_pass"),
				TLS:         pointer.ToBool(true),
				PushMetrics: pointer.ToBool(false),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"environment": "test"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"valkey_exporter": {
					"enable": true,
					"username": "redis_user",
					"password": "redis_pass",
					"tls": true,
					"disable_collectors": null,
					"enable_push_metrics": false,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "test"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-valkey-disable", `{"valkey_exporter": {"agent_id": "test-agent-valkey-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentValkeyExporterCommand{
				AgentID:     "test-agent-valkey-disable",
				Enable:      pointer.ToBool(false),
				PushMetrics: pointer.ToBool(false),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"valkey_exporter": {
					"enable": false,
					"disable_collectors": null,
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
			"valkey_exporter": {
				"agent_id": "test-agent-valkey-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "valkey-service-456",
				"username": "valkey_user",
				"listen_port": 9121,
				"tls": true,
				"tls_skip_verify": true,
				"push_metrics_enabled": true,
				"expose_exporter": true,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "valkey"
				},
				"process_exec_path": "/usr/bin/valkey_exporter",
				"log_level": "LOG_LEVEL_DEBUG"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-valkey-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "valkey-exporter", "test-agent-valkey-all-flags",
			"--enable",
			"--username=valkey_user",
			"--password=valkey_pass",
			"--agent-password=agentpass",
			"--tls",
			"--tls-skip-verify",
			"--disable-collectors=latency,info",
			"--expose-exporter",
			"--push-metrics",
			"--log-level=debug",
			"--custom-labels=env=test,team=valkey",
		}

		var cmd ChangeAgentValkeyExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"valkey_exporter": {
				"enable": true,
				"username": "valkey_user",
				"password": "valkey_pass",
				"agent_password": "agentpass",
				"tls": true,
				"tls_skip_verify": true,
				"disable_collectors": ["latency", "info"],
				"expose_exporter": true,
				"enable_push_metrics": true,
				"log_level": "LOG_LEVEL_DEBUG",
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "valkey"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `Valkey Exporter agent configuration updated.
Agent ID              : test-agent-valkey-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : valkey-service-456
Username              : valkey_user
Listen port           : 9121
TLS enabled           : true
Skip TLS verification : true
Push metrics enabled  : true
Expose exporter       : true

Disabled              : false
Custom labels         : env=test, team=valkey
Process exec path     : /usr/bin/valkey_exporter
Log level             : debug
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - updated agent password
  - enabled TLS
  - enabled TLS skip verification
  - updated disabled collectors: [latency info]
  - enabled expose exporter
  - enabled push metrics
  - changed log level to debug
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-valkey", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentValkeyExporterCommand{
			AgentID: "invalid-agent-valkey",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-valkey-minimal", `{"valkey_exporter": {"agent_id": "test-agent-valkey-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "valkey-exporter", "test-agent-valkey-minimal"}

		var cmd ChangeAgentValkeyExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty valkey_exporter object when no flags are set
		expectedJSON := `{
			"valkey_exporter": {
				"disable_collectors": null
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "valkey-exporter"}

			var cmd ChangeAgentValkeyExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "valkey-exporter", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentValkeyExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
