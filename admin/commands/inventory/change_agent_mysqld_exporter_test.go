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
	"strings"
	"testing"

	"github.com/AlekSi/pointer"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alecthomas/kong"
	"github.com/percona/pmm/admin/pkg/flags"
	"github.com/percona/pmm/api/inventory/v1/json/client"
)

func TestMysqldExporterChangeAgent(t *testing.T) {
	t.Run("CoreFunctionality", func(t *testing.T) {
		agentID := "test-mysqld-agent-id"
		responseJSON := `{
			"mysqld_exporter": {
				"agent_id": "test-mysqld-agent-id",
				"pmm_agent_id": "test-pmm-agent-id",
				"service_id": "test-service-id",
				"username": "test-user",
				"disabled": false,
				"listen_port": 9104,
				"process_exec_path": "",
				"log_level": "LOG_LEVEL_INFO",
				"tls": true,
				"tls_skip_verify": false,
				"push_metrics_enabled": true,
				"expose_exporter": true,
				"custom_labels": {
					"env": "production",
					"database": "mysql"
				},
				"tablestats_group_table_limit": 1000
			}
		}`

		_, cleanup := setupChangeAgentTestServer(t, agentID, responseJSON, nil)
		defer cleanup()

		t.Run("UpdateCredentialsAndTLS", func(t *testing.T) {
			customLabels := map[string]string{"env": "staging", "version": "8.0"}
			cmd := &ChangeAgentMysqldExporterCommand{
				AgentID:                   agentID,
				Enable:                    pointer.ToBool(true),
				Username:                  pointer.ToString("new-user"),
				Password:                  pointer.ToString("new-password"),
				TLS:                       pointer.ToBool(true),
				TLSSkipVerify:             pointer.ToBool(false),
				CustomLabels:              &customLabels,
				PushMetrics:               pointer.ToBool(true),
				ExposeExporter:            pointer.ToBool(true),
				TablestatsGroupTableLimit: pointer.ToInt32(2000),
			}
			logLevel := flags.LogLevel("debug")
			cmd.LogLevel = &logLevel

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			require.NotNil(t, result)

			changeResult := result.(*changeAgentMysqldExporterResult)
			assert.Equal(t, agentID, changeResult.Agent.AgentID)

			expectedChanges := []string{
				"enabled agent",
				"updated username",
				"updated password",
				"enabled TLS",
				"disabled TLS skip verification",
				"enabled push metrics",
				"enabled expose exporter",
				"changed log level to debug",
				"updated custom labels",
				"changed tablestats group table limit to 2000",
			}
			assert.ElementsMatch(t, expectedChanges, changeResult.Changes)
		})

		t.Run("DisableAgentWithCollectors", func(t *testing.T) {
			cmd := &ChangeAgentMysqldExporterCommand{
				AgentID:           agentID,
				Enable:            pointer.ToBool(false),
				DisableCollectors: []string{"info_schema.innodb_metrics", "info_schema.processlist"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			require.NotNil(t, result)

			changeResult := result.(*changeAgentMysqldExporterResult)
			assert.Contains(t, changeResult.Changes, "disabled agent")
			assert.Len(t, changeResult.Changes, 2) // Should have both "disabled agent" and the collectors change
			// Check that there's a change about collectors
			foundCollectorsChange := false
			for _, change := range changeResult.Changes {
				if strings.Contains(change, "disabled collectors") {
					foundCollectorsChange = true
					break
				}
			}
			assert.True(t, foundCollectorsChange, "Should have a change about disabled collectors")
		})
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		agentID := "invalid-id"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/v1/inventory/agents/"+agentID, r.URL.Path)

			w.WriteHeader(http.StatusNotFound)
			response := `{"code": 5, "error": "Agent not found"}`
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

		cmd := &ChangeAgentMysqldExporterCommand{
			AgentID: agentID,
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("TLSFilesValidation", func(t *testing.T) {
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-id", "", nil)
		defer cleanup()

		t.Run("NonExistentTLSFiles", func(t *testing.T) {
			cmd := &ChangeAgentMysqldExporterCommand{
				AgentID:     "test-agent-id",
				TLSCaFile:   pointer.ToString("/non/existent/ca.pem"),
				TLSCertFile: pointer.ToString("/non/existent/cert.pem"),
				TLSKeyFile:  pointer.ToString("/non/existent/key.pem"),
			}

			result, err := cmd.RunCmd()
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "failed to read TLS")
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"mysqld_exporter": {
				"agent_id": "test-mysqld-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "mysql-service-456",
				"username": "testuser",
				"listen_port": 9104,
				"tls": true,
				"tls_skip_verify": false,
				"push_metrics_enabled": true,
				"expose_exporter": true,
				"disabled": false,
				"custom_labels": {
					"env": "production",
					"db": "mysql"
				},
				"process_exec_path": "/usr/bin/mysqld_exporter",
				"log_level": "LOG_LEVEL_DEBUG"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-mysqld-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		// Use Kong to parse CLI arguments into command struct
		var cmd ChangeAgentMysqldExporterCommand
		parser := kong.Must(&cmd)

		args := []string{
			"test-mysqld-all-flags",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--agent-password=agentpass",
			"--tls",
			"--tls-skip-verify=false",
			"--tablestats-group-table-limit=5000",
			"--disable-collectors=info_schema.innodb_metrics,info_schema.processlist",
			"--expose-exporter",
			"--push-metrics",
			"--custom-labels=env=production,db=mysql",
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
			"mysqld_exporter": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"agent_password": "agentpass",
				"tls": true,
				"tls_skip_verify": false,
				"tablestats_group_table_limit": 5000,
				"disable_collectors": ["info_schema.innodb_metrics", "info_schema.processlist"],
				"expose_exporter": true,
				"enable_push_metrics": true,
				"log_level": "LOG_LEVEL_DEBUG",
				"custom_labels": {
					"values": {
						"db": "mysql",
						"env": "production"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `MySQL Exporter agent configuration updated.
Agent ID                     : test-mysqld-all-flags
PMM-Agent ID                 : pmm-agent-123
Service ID                   : mysql-service-456
Username                     : testuser
Listen port                  : 9104
TLS enabled                  : true
Skip TLS verification        : false
Push metrics enabled         : true
Expose exporter              : true

Disabled                     : false
Custom labels                : db=mysql, env=production
Process exec path            : /usr/bin/mysqld_exporter
Log level                    : debug
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - updated agent password
  - enabled TLS
  - disabled TLS skip verification
  - changed tablestats group table limit to 5000
  - updated disabled collectors: [info_schema.innodb_metrics info_schema.processlist]
  - enabled expose exporter
  - enabled push metrics
  - changed log level to debug
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		agentID := "test-mysqld-minimal"
		var capturedRequestBody string

		_, cleanup := setupChangeAgentTestServer(t, agentID, "", &capturedRequestBody)
		defer cleanup()

		// Use Kong to parse minimal CLI arguments (only required AgentID)
		var cmd ChangeAgentMysqldExporterCommand
		parser := kong.Must(&cmd)

		args := []string{agentID} // Only the required argument

		ctx, err := parser.Parse(args)
		require.NoError(t, err)
		require.NotNil(t, ctx)

		// Verify Kong parsed correctly with default/nil values
		assert.Equal(t, agentID, cmd.AgentID)
		assert.Nil(t, cmd.Enable)
		assert.Nil(t, cmd.Username)
		assert.Nil(t, cmd.Password)
		assert.Nil(t, cmd.AgentPassword)
		assert.Nil(t, cmd.TLS)
		assert.Nil(t, cmd.TLSSkipVerify)
		assert.Nil(t, cmd.TablestatsGroupTableLimit)
		assert.Empty(t, cmd.DisableCollectors)
		assert.Nil(t, cmd.ExposeExporter)
		assert.Nil(t, cmd.PushMetrics)
		assert.Nil(t, cmd.CustomLabels)
		assert.Nil(t, cmd.LogLevel) // No default value anymore

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify request body does NOT contain optional fields (except those that serialize or have defaults)
		expectedJSON := `{
			"mysqld_exporter": {
				"disable_collectors": null
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Run("MissingRequiredArgument", func(t *testing.T) {
			var cmd ChangeAgentMysqldExporterCommand
			parser := kong.Must(&cmd)

			// Missing required AgentID argument
			_, err := parser.Parse([]string{"--enable"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			var cmd ChangeAgentMysqldExporterCommand
			parser := kong.Must(&cmd)

			_, err := parser.Parse([]string{"test-id", "--log-level=invalid"})
			assert.Error(t, err)
		})
	})
}
