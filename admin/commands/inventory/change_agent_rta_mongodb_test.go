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
	"time"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/pkg/flags"
)

func TestChangeAgentRTAMongoDBAgentCommand(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			cleanup := setupChangeAgentTestServer(t, "test-agent-rta-mongodb-update", `{"rta_mongodb_agent": {"agent_id": "test-agent-rta-mongodb-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentRTAMongoDBAgentCommand{
				AgentID:                       "test-agent-rta-mongodb-update",
				Enable:                        new(true),
				Username:                      new("mongodb_user"),
				Password:                      new("mongodb_pass"),
				TLS:                           new(true),
				TLSSkipVerify:                 new(false),
				TLSCertificateKeyFilePassword: new("cert_password"),
				AuthenticationMechanism:       new("SCRAM-SHA-256"),
				CollectInterval:               new(3 * time.Second),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: new(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"service": "mongodb", "environment": "production"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"rta_mongodb_agent": {
					"enable": true,
					"username": "mongodb_user",
					"password": "mongodb_pass",
					"tls": true,
					"tls_skip_verify": false,
					"tls_certificate_key_file_password": "cert_password",
					"authentication_mechanism": "SCRAM-SHA-256",
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"service": "mongodb",
							"environment": "production"
						}
					},
					"rta_options": {
						"collect_interval": "3s"
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			cleanup := setupChangeAgentTestServer(t, "test-agent-rta-mongodb-disable", `{"rta_mongodb_agent": {"agent_id": "test-agent-rta-mongodb-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentRTAMongoDBAgentCommand{
				AgentID: "test-agent-rta-mongodb-disable",
				Enable:  new(false),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"rta_mongodb_agent": {
					"enable": false
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		mockResponse := `{
			"rta_mongodb_agent": {
				"agent_id": "test-agent-rta-mongodb-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "mongodb-service-456",
				"username": "testuser",
				"tls": true,
				"tls_skip_verify": true,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "rta"
				},
				"rta_options": {
					"collect_interval": "5s"
				},
				"log_level": "LOG_LEVEL_INFO"
			}
		}`

		cleanup := setupChangeAgentTestServer(t, "test-agent-rta-mongodb-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "rta-mongodb-agent", "test-agent-rta-mongodb-all-flags",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--tls",
			"--tls-skip-verify",
			"--tls-certificate-key-file-password=certpass",
			"--authentication-mechanism=MONGODB-CR",
			"--collect-interval=5s",
			"--log-level=info",
			"--custom-labels=env=test,team=rta",
		}

		var cmd ChangeAgentRTAMongoDBAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"rta_mongodb_agent": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"tls": true,
				"tls_skip_verify": true,
				"tls_certificate_key_file_password": "certpass",
				"authentication_mechanism": "MONGODB-CR",
				"log_level": "LOG_LEVEL_INFO",
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "rta"
					}
				},
				"rta_options": {
					"collect_interval": "5s"
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		output := result.String()
		expectedOutput := `Real-Time Analytics MongoDB agent configuration updated.
Agent ID              : test-agent-rta-mongodb-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : mongodb-service-456
Username              : testuser
TLS enabled           : true
Skip TLS verification : true

Disabled              : false
Custom labels         : env=test, team=rta
Collect interval      : 5s
Log level             : info
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - enabled TLS
  - enabled TLS skip verification
  - updated TLS certificate key password
  - changed authentication mechanism to MONGODB-CR
  - changed log level to info
  - updated custom labels
  - changed collect interval to 5s
`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		cleanup := setupChangeAgentTestServer(t, "invalid-agent-rta-mongodb", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentRTAMongoDBAgentCommand{
			AgentID: "invalid-agent-rta-mongodb",
			Enable:  new(true),
		}

		result, err := cmd.RunCmd()
		require.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		cleanup := setupChangeAgentTestServer(t, "test-agent-rta-mongodb-minimal", `{"rta_mongodb_agent": {"agent_id": "test-agent-rta-mongodb-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "rta-mongodb-agent", "test-agent-rta-mongodb-minimal"}

		var cmd ChangeAgentRTAMongoDBAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"rta_mongodb_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "rta-mongodb-agent"}

			var cmd ChangeAgentRTAMongoDBAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "rta-mongodb-agent", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentRTAMongoDBAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			require.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
