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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/kong"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
)

func TestNodeExporterChangeAgent(t *testing.T) {
	t.Run("CoreFunctionality", func(t *testing.T) {
		agentID := "test-agent-id"
		responseJSON := `{
			"node_exporter": {
				"agent_id": "test-agent-id",
				"pmm_agent_id": "test-pmm-agent-id", 
				"disabled": false,
				"listen_port": 9100,
				"process_exec_path": "",
				"log_level": "LOG_LEVEL_INFO",
				"push_metrics_enabled": true,
				"expose_exporter": false,
				"custom_labels": {
					"env": "test",
					"team": "devops"
				}
			}
		}`

		_, cleanup := setupChangeAgentTestServer(t, agentID, responseJSON, nil)
		defer cleanup()

		t.Run("EnableAgent", func(t *testing.T) {
			customLabels := map[string]string{"env": "production", "region": "us-west"}
			logLevel := flags.LogLevel("info")
			cmd := &ChangeAgentNodeExporterCommand{
				AgentID:      agentID,
				Enable:       pointer.ToBool(true),
				CustomLabels: &customLabels,
				PushMetrics:  pointer.ToBool(true),
			}
			cmd.LogLevel = &logLevel

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			require.NotNil(t, result)

			changeResult := result.(*changeAgentNodeExporterResult)
			assert.Equal(t, agentID, changeResult.Agent.AgentID)

			expectedChanges := []string{
				"enabled agent",
				"enabled push metrics",
				"changed log level to info",
				"updated custom labels",
			}
			assert.ElementsMatch(t, expectedChanges, changeResult.Changes)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			cmd := &ChangeAgentNodeExporterCommand{
				AgentID:        agentID,
				Enable:         pointer.ToBool(false),
				ExposeExporter: pointer.ToBool(true),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			require.NotNil(t, result)

			changeResult := result.(*changeAgentNodeExporterResult)
			expectedChanges := []string{
				"disabled agent",
				"enabled expose exporter",
			}
			assert.ElementsMatch(t, expectedChanges, changeResult.Changes)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"node_exporter": {
				"agent_id": "test-agent-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"listen_port": 9100,
				"push_metrics_enabled": true,
				"expose_exporter": false,
				"disabled": false,
				"custom_labels": {
					"env": "production",
					"team": "devops"
				},
				"process_exec_path": "/usr/bin/node_exporter",
				"log_level": "LOG_LEVEL_DEBUG"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		// Use Kong to parse CLI arguments into command struct
		var cmd ChangeAgentNodeExporterCommand
		parser := kong.Must(&cmd)

		args := []string{
			"test-agent-all-flags",
			"--enable",
			"--push-metrics",
			"--expose-exporter=false",
			"--disable-collectors=cpu,memory,filesystem",
			"--custom-labels=env=production,team=devops",
			"--log-level=debug",
		}

		ctx, err := parser.Parse(args)
		require.NoError(t, err)
		require.NotNil(t, ctx)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		require.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"node_exporter": {
				"enable": true,
				"enable_push_metrics": true,
				"expose_exporter": false,
				"log_level": "LOG_LEVEL_DEBUG",
				"disable_collectors": ["cpu", "memory", "filesystem"],
				"custom_labels": {
					"values": {
						"env": "production",
						"team": "devops"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `Node Exporter agent configuration updated.
Agent ID              : test-agent-all-flags
PMM-Agent ID          : pmm-agent-123
Listen port           : 9100
Push metrics enabled  : true
Expose exporter       : false

Disabled              : false
Custom labels         : env=production, team=devops
Process exec path     : /usr/bin/node_exporter
Log level             : debug
Configuration changes applied:
  - enabled agent
  - enabled push metrics
  - disabled expose exporter
  - updated disabled collectors: [cpu memory filesystem]
  - changed log level to debug
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		agentID := "invalid-id"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/v1/inventory/agents/"+agentID, r.URL.Path)

			w.WriteHeader(http.StatusNotFound)
			response := `{"code": 5, "error": "Agent not found", "message": "Agent with ID 'invalid-id' not found."}`
			_, err := w.Write([]byte(response))
			require.NoError(t, err)
		}))
		defer server.Close()

		// Setup client to use test server
		serverURL, _ := url.Parse(server.URL)
		originalClient := client.Default
		transport := httptransport.New(serverURL.Host, serverURL.Path, []string{serverURL.Scheme})
		client.Default = client.New(transport, nil)
		defer func() { client.Default = originalClient }()

		cmd := &ChangeAgentNodeExporterCommand{
			AgentID: agentID,
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		agentID := "test-agent-no-flags"
		var capturedRequestBody string

		_, cleanup := setupChangeAgentTestServer(t, agentID, "", &capturedRequestBody)
		defer cleanup()

		// Use Kong to parse minimal CLI arguments (only required AgentID)
		var cmd ChangeAgentNodeExporterCommand
		parser := kong.Must(&cmd)

		args := []string{agentID} // Only the required argument

		ctx, err := parser.Parse(args)
		require.NoError(t, err)
		require.NotNil(t, ctx)

		// Verify Kong parsed correctly with default/nil values
		assert.Equal(t, agentID, cmd.AgentID)
		assert.Nil(t, cmd.Enable)
		assert.Nil(t, cmd.PushMetrics)
		assert.Nil(t, cmd.ExposeExporter)
		assert.Nil(t, cmd.CustomLabels)
		assert.Empty(t, cmd.DisableCollectors)
		assert.Nil(t, cmd.LogLevel) // Now nil instead of default "warn"

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify request body only contains disable_collectors (no omitempty) but not log_level (nil = omitted)
		expectedJSON := `{
			"node_exporter": {
				"disable_collectors": null
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Run("MissingRequiredArgument", func(t *testing.T) {
			var cmd ChangeAgentNodeExporterCommand
			parser := kong.Must(&cmd)

			// Missing required AgentID argument
			_, err := parser.Parse([]string{"--enable"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			var cmd ChangeAgentNodeExporterCommand
			parser := kong.Must(&cmd)

			_, err := parser.Parse([]string{"test-id", "--log-level=invalid"})
			assert.Error(t, err)
		})
	})
}
