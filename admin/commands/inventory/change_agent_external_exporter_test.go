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
)

func TestExternalExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateConfigurationAndMetrics", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-external-update", `{"external_exporter": {"agent_id": "test-agent-external-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentExternalExporterCommand{
				AgentID:       "test-agent-external-update",
				Enable:        pointer.ToBool(true),
				Username:      pointer.ToString("external_user"),
				ListenPort:    pointer.ToInt64(9104),
				MetricsScheme: pointer.ToString("https"),
				MetricsPath:   pointer.ToString("/custom/metrics"),
				PushMetrics:   pointer.ToBool(true),
				CustomLabels:  &map[string]string{"service": "external", "environment": "production"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"external_exporter": {
					"enable": true,
					"username": "external_user",
					"listen_port": 9104,
					"scheme": "https",
					"metrics_path": "/custom/metrics",
					"enable_push_metrics": true,
					"custom_labels": {
						"values": {
							"service": "external",
							"environment": "production"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-external-disable", `{"external_exporter": {"agent_id": "test-agent-external-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentExternalExporterCommand{
				AgentID:     "test-agent-external-disable",
				Enable:      pointer.ToBool(false),
				PushMetrics: pointer.ToBool(false),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"external_exporter": {
					"enable": false,
					"enable_push_metrics": false
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("UpdateOnlyMetricsConfiguration", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-external-metrics", `{"external_exporter": {"agent_id": "test-agent-external-metrics"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentExternalExporterCommand{
				AgentID:       "test-agent-external-metrics",
				ListenPort:    pointer.ToInt64(8080),
				MetricsScheme: pointer.ToString("http"),
				MetricsPath:   pointer.ToString("/metrics"),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"external_exporter": {
					"listen_port": 8080,
					"scheme": "http",
					"metrics_path": "/metrics"
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"external_exporter": {
				"agent_id": "test-agent-external-all-flags",
				"runs_on_node_id": "node-123",
				"service_id": "external-service-456",
				"username": "exporter_user",
				"scheme": "https",
				"metrics_path": "/custom/metrics",
				"listen_port": 9105,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "external"
				}
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-external-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "external-exporter", "test-agent-external-all-flags",
			"--enable",
			"--username=exporter_user",
			"--listen-port=9105",
			"--metrics-scheme=https",
			"--metrics-path=/custom/metrics",
			"--push-metrics",
			"--custom-labels=env=test,team=external",
		}

		var cmd ChangeAgentExternalExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"external_exporter": {
				"enable": true,
				"username": "exporter_user",
				"listen_port": 9105,
				"scheme": "https",
				"metrics_path": "/custom/metrics",
				"enable_push_metrics": true,
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "external"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `External Exporter agent configuration updated.
Agent ID        : test-agent-external-all-flags
Runs on node ID : node-123
Service ID      : external-service-456
Username        : exporter_user
Scheme          : https
Metrics path    : /custom/metrics
Listen port     : 9105

Disabled        : false
Custom labels   : env=test, team=external
Configuration changes applied:
  - enabled agent
  - updated username
  - changed listen port to 9105
  - changed metrics scheme to https
  - changed metrics path to /custom/metrics
  - enabled push metrics
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-external", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentExternalExporterCommand{
			AgentID: "invalid-agent-external",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-external-minimal", `{"external_exporter": {"agent_id": "test-agent-external-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "external-exporter", "test-agent-external-minimal"}

		var cmd ChangeAgentExternalExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty external_exporter object when no flags are set
		expectedJSON := `{
			"external_exporter": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingMetricsConfigurationOnly", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-external-metrics-only", `{"external_exporter": {"agent_id": "test-agent-external-metrics-only"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "external-exporter", "test-agent-external-metrics-only",
			"--listen-port=9106",
			"--metrics-scheme=http",
			"--metrics-path=/health",
		}

		var cmd ChangeAgentExternalExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"external_exporter": {
				"listen_port": 9106,
				"scheme": "http",
				"metrics_path": "/health"
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "external-exporter"}

			var cmd ChangeAgentExternalExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidPortNumber", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "external-exporter", "test-agent-id", "--listen-port=invalid"}

			var cmd ChangeAgentExternalExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "listen-port")
		})
	})
}
