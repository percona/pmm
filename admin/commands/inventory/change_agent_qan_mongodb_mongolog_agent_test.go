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

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
)

func TestQANMongoDBMongologAgentChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongolog-update", `{"qan_mongodb_mongolog_agent": {"agent_id": "test-agent-qan-mongolog-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMongoDBMongologAgentCommand{
				AgentID:                       "test-agent-qan-mongolog-update",
				Enable:                        new(true),
				Username:                      new("mongodb_user"),
				Password:                      new("mongodb_pass"),
				TLS:                           new(true),
				TLSSkipVerify:                 new(false),
				TLSCertificateKeyFilePassword: new("cert_password"),
				AuthenticationMechanism:       new("SCRAM-SHA-256"),
				AuthenticationDatabase:        new("admin"),
				MaxQueryLength:                new(int32(2048)),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: new(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"service": "mongodb", "environment": "production"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mongodb_mongolog_agent": {
					"enable": true,
					"username": "mongodb_user",
					"password": "mongodb_pass",
					"tls": true,
					"tls_skip_verify": false,
					"tls_certificate_key_file_password": "cert_password",
					"authentication_mechanism": "SCRAM-SHA-256",
					"authentication_database": "admin",
					"max_query_length": 2048,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"service": "mongodb",
							"environment": "production"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongolog-disable", `{"qan_mongodb_mongolog_agent": {"agent_id": "test-agent-qan-mongolog-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMongoDBMongologAgentCommand{
				AgentID:        "test-agent-qan-mongolog-disable",
				Enable:         new(false),
				MaxQueryLength: new(int32(1024)),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mongodb_mongolog_agent": {
					"enable": false,
					"max_query_length": 1024
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		mockResponse := `{
			"qan_mongodb_mongolog_agent": {
				"agent_id": "test-agent-qan-mongolog-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "mongodb-service-456",
				"username": "testuser",
				"tls": true,
				"tls_skip_verify": true,
				"max_query_length": 1024,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "qan"
				},
				"process_exec_path": "/usr/bin/pmm-agent",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`

		cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongolog-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "qan-mongodb-mongolog-agent", "test-agent-qan-mongolog-all-flags",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--tls",
			"--tls-skip-verify",
			"--tls-certificate-key-file-password=certpass",
			"--authentication-mechanism=MONGODB-CR",
			"--authentication-database=testdb",
			"--max-query-length=1024",
			"--log-level=info",
			"--custom-labels=env=test,team=qan",
		}

		var cmd ChangeAgentQANMongoDBMongologAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"qan_mongodb_mongolog_agent": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"tls": true,
				"tls_skip_verify": true,
				"tls_certificate_key_file_password": "certpass",
				"authentication_mechanism": "MONGODB-CR",
				"authentication_database": "testdb",
				"max_query_length": 1024,
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

		output := result.String()
		expectedOutput := `QAN MongoDB Mongolog agent configuration updated.
Agent ID              : test-agent-qan-mongolog-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : mongodb-service-456
Username              : testuser
TLS enabled           : true
Skip TLS verification : true
Max query length      : 1024

Disabled              : false
Custom labels         : env=test, team=qan
Log level             : info
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - enabled TLS
  - enabled TLS skip verification
  - updated TLS certificate key password
  - changed max query length to 1024
  - changed authentication mechanism to MONGODB-CR
  - changed authentication database to testdb
  - changed log level to info
  - updated custom labels
`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		cleanup := setupChangeAgentTestServer(t, "invalid-agent-qan-mongolog", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentQANMongoDBMongologAgentCommand{
			AgentID: "invalid-agent-qan-mongolog",
			Enable:  new(true),
		}

		result, err := cmd.RunCmd()
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongolog-minimal", `{"qan_mongodb_mongolog_agent": {"agent_id": "test-agent-qan-mongolog-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "qan-mongodb-mongolog-agent", "test-agent-qan-mongolog-minimal"}

		var cmd ChangeAgentQANMongoDBMongologAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"qan_mongodb_mongolog_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mongodb-mongolog-agent"}

			var cmd ChangeAgentQANMongoDBMongologAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mongodb-mongolog-agent", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentQANMongoDBMongologAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
