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

func TestQANMySQLPerfSchemaAgentChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-perfschema-update", `{"qan_mysql_perfschema_agent": {"agent_id": "test-agent-qan-perfschema-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMySQLPerfSchemaAgentCommand{
				AgentID:              "test-agent-qan-perfschema-update",
				Enable:               pointer.ToBool(true),
				Username:             pointer.ToString("mysql_user"),
				Password:             pointer.ToString("mysql_pass"),
				TLS:                  pointer.ToBool(true),
				TLSSkipVerify:        pointer.ToBool(false),
				MaxQueryLength:       pointer.ToInt32(2048),
				DisableQueryExamples: pointer.ToBool(false),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CommentsParsingChangeFlags: flags.CommentsParsingChangeFlags{
					CommentsParsing: pointer.ToString("off"),
				},
				CustomLabels: &map[string]string{"service": "mysql", "role": "primary"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mysql_perfschema_agent": {
					"enable": true,
					"username": "mysql_user",
					"password": "mysql_pass",
					"tls": true,
					"tls_skip_verify": false,
					"max_query_length": 2048,
					"disable_query_examples": false,
					"disable_comments_parsing": true,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"service": "mysql",
							"role": "primary"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-perfschema-disable", `{"qan_mysql_perfschema_agent": {"agent_id": "test-agent-qan-perfschema-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMySQLPerfSchemaAgentCommand{
				AgentID:              "test-agent-qan-perfschema-disable",
				Enable:               pointer.ToBool(false),
				DisableQueryExamples: pointer.ToBool(true),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mysql_perfschema_agent": {
					"enable": false,
					"disable_query_examples": true
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-qan-perfschema", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentQANMySQLPerfSchemaAgentCommand{
			AgentID: "invalid-agent-qan-perfschema",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"qan_mysql_perfschema_agent": {
				"agent_id": "test-agent-qan-perfschema-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "mysql-service-456",
				"username": "testuser",
				"tls": true,
				"tls_skip_verify": true,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "qan"
				},
				"process_exec_path": "/usr/bin/mysql",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-perfschema-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "qan-mysql-perfschema-agent", "test-agent-qan-perfschema-flags",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--tls",
			"--tls-skip-verify",
			"--max-query-length=1024",
			"--disable-query-examples",
			"--comments-parsing=off",
			"--log-level=info",
			"--custom-labels=env=test,team=qan",
		}

		var cmd ChangeAgentQANMySQLPerfSchemaAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"qan_mysql_perfschema_agent": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"tls": true,
				"tls_skip_verify": true,
				"max_query_length": 1024,
				"disable_query_examples": true,
				"disable_comments_parsing": true,
				"log_level": "LOG_LEVEL_INFO",
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "qan"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `QAN MySQL PerfSchema agent configuration updated.
Agent ID              : test-agent-qan-perfschema-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : mysql-service-456
Username              : testuser
TLS enabled           : true
Skip TLS verification : true

Disabled              : false
Custom labels         : env=test, team=qan
Process exec path     : /usr/bin/mysql
Log level             : info
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - enabled TLS
  - enabled TLS skip verification
  - changed max query length to 1024
  - disabled query examples
  - disabled comments parsing
  - changed log level to info
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-perfschema-minimal", `{"qan_mysql_perfschema_agent": {"agent_id": "test-agent-qan-perfschema-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "qan-mysql-perfschema-agent", "test-agent-qan-perfschema-minimal"}

		var cmd ChangeAgentQANMySQLPerfSchemaAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty qan_mysql_perfschema_agent object when no flags are set
		expectedJSON := `{
			"qan_mysql_perfschema_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mysql-perfschema-agent"}

			var cmd ChangeAgentQANMySQLPerfSchemaAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mysql-perfschema-agent", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentQANMySQLPerfSchemaAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
