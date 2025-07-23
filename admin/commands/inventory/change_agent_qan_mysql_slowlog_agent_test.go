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

func TestQANMySQLSlowlogAgentChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-slowlog-update", `{"qan_mysql_slowlog_agent": {"agent_id": "test-agent-qan-slowlog-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMySQLSlowlogAgentCommand{
				AgentID:              "test-agent-qan-slowlog-update",
				Enable:               pointer.ToBool(true),
				Username:             pointer.ToString("mysql_user"),
				Password:             pointer.ToString("mysql_pass"),
				TLS:                  pointer.ToBool(true),
				TLSSkipVerify:        pointer.ToBool(false),
				MaxSlowlogFileSize:   pointer.ToString("2GiB"),
				MaxQueryLength:       pointer.ToInt32(2048),
				DisableQueryExamples: pointer.ToBool(true),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("info")),
				},
				CommentsParsingChangeFlags: flags.CommentsParsingChangeFlags{
					CommentsParsing: pointer.ToString("off"),
				},
				CustomLabels: &map[string]string{"service": "mysql", "team": "db"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mysql_slowlog_agent": {
					"enable": true,
					"username": "mysql_user",
					"password": "mysql_pass",
					"tls": true,
					"tls_skip_verify": false,
					"max_slowlog_file_size": "2GiB",
					"max_query_length": 2048,
					"disable_query_examples": true,
					"disable_comments_parsing": true,
					"log_level": "LOG_LEVEL_INFO",
					"custom_labels": {
						"values": {
							"service": "mysql",
							"team": "db"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-slowlog-disable", `{"qan_mysql_slowlog_agent": {"agent_id": "test-agent-qan-slowlog-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMySQLSlowlogAgentCommand{
				AgentID:              "test-agent-qan-slowlog-disable",
				Enable:               pointer.ToBool(false),
				DisableQueryExamples: pointer.ToBool(true),
				CommentsParsingChangeFlags: flags.CommentsParsingChangeFlags{
					CommentsParsing: pointer.ToString("off"),
				},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mysql_slowlog_agent": {
					"enable": false,
					"disable_query_examples": true,
					"disable_comments_parsing": true
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"qan_mysql_slowlog_agent": {
				"agent_id": "test-agent-qan-slowlog-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "mysql-service-456",
				"username": "mysql_user",
				"tls": true,
				"tls_skip_verify": true,
				"max_query_length": 2048,
				"max_slowlog_file_size": "2GiB",
				"disable_query_examples": true,
				"disable_comments_parsing": true,
				"disabled": false,
				"custom_labels": {
					"service": "mysql",
					"team": "db"
				},
				"process_exec_path": "/usr/bin/mysqld",
				"log_level": "LOG_LEVEL_DEBUG"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-slowlog-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "qan-mysql-slowlog-agent", "test-agent-qan-slowlog-all-flags",
			"--enable",
			"--username=mysql_user",
			"--password=mysql_pass",
			"--tls",
			"--tls-skip-verify",
			"--max-query-length=2048",
			"--max-slowlog-file-size=2GiB",
			"--disable-query-examples",
			"--comments-parsing=off",
			"--log-level=debug",
			"--custom-labels=service=mysql,team=db",
		}

		var cmd ChangeAgentQANMySQLSlowlogAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"qan_mysql_slowlog_agent": {
				"enable": true,
				"username": "mysql_user",
				"password": "mysql_pass",
				"tls": true,
				"tls_skip_verify": true,
				"max_query_length": 2048,
				"max_slowlog_file_size": "2GiB",
				"disable_query_examples": true,
				"disable_comments_parsing": true,
				"log_level": "LOG_LEVEL_DEBUG",
				"custom_labels": {
					"values": {
						"service": "mysql",
						"team": "db"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `QAN MySQL SlowLog agent configuration updated.
Agent ID              : test-agent-qan-slowlog-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : mysql-service-456
Username              : mysql_user
TLS enabled           : true
Skip TLS verification : true

Disabled              : false
Custom labels         : service=mysql, team=db
Process exec path     : /usr/bin/mysqld
Log level             : debug
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - enabled TLS
  - enabled TLS skip verification
  - changed max slowlog file size to 2GiB
  - changed max query length to 2048
  - disabled query examples
  - disabled comments parsing
  - changed log level to debug
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-qan-slowlog", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentQANMySQLSlowlogAgentCommand{
			AgentID: "invalid-agent-qan-slowlog",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-slowlog-minimal", `{"qan_mysql_slowlog_agent": {"agent_id": "test-agent-qan-slowlog-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "qan-mysql-slowlog-agent", "test-agent-qan-slowlog-minimal"}

		var cmd ChangeAgentQANMySQLSlowlogAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty qan_mysql_slowlog_agent object when no flags are set
		expectedJSON := `{
			"qan_mysql_slowlog_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingWithLogLevel", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-slowlog-with-log", `{"qan_mysql_slowlog_agent": {"agent_id": "test-agent-qan-slowlog-with-log"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "qan-mysql-slowlog-agent", "test-agent-qan-slowlog-with-log", "--enable", "--log-level=debug"}

		var cmd ChangeAgentQANMySQLSlowlogAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should include both enable and log_level when both are set
		expectedJSON := `{
			"qan_mysql_slowlog_agent": {
				"enable": true,
				"log_level": "LOG_LEVEL_DEBUG"
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mysql-slowlog-agent"}

			var cmd ChangeAgentQANMySQLSlowlogAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mysql-slowlog-agent", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentQANMySQLSlowlogAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
