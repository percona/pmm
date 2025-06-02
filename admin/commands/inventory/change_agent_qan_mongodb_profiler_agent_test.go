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

func TestQANMongoDBProfilerAgentChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongodb-update", `{"qan_mongodb_profiler_agent": {"agent_id": "test-agent-qan-mongodb-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMongoDBProfilerAgentCommand{
				AgentID:                       "test-agent-qan-mongodb-update",
				Enable:                        pointer.ToBool(true),
				Username:                      pointer.ToString("mongodb_user"),
				Password:                      pointer.ToString("mongodb_pass"),
				TLS:                           pointer.ToBool(true),
				TLSSkipVerify:                 pointer.ToBool(false),
				TLSCertificateKeyFilePassword: pointer.ToString("cert_password"),
				AuthenticationMechanism:       pointer.ToString("SCRAM-SHA-256"),
				AuthenticationDatabase:        pointer.ToString("admin"),
				MaxQueryLength:                pointer.ToInt32(2048),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"service": "mongodb", "environment": "production"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mongodb_profiler_agent": {
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
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongodb-disable", `{"qan_mongodb_profiler_agent": {"agent_id": "test-agent-qan-mongodb-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANMongoDBProfilerAgentCommand{
				AgentID:        "test-agent-qan-mongodb-disable",
				Enable:         pointer.ToBool(false),
				MaxQueryLength: pointer.ToInt32(1024),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_mongodb_profiler_agent": {
					"enable": false,
					"max_query_length": 1024
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"qan_mongodb_profiler_agent": {
				"agent_id": "test-agent-qan-mongodb-all-flags",
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
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongodb-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "qan-mongodb-profiler-agent", "test-agent-qan-mongodb-all-flags",
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

		var cmd ChangeAgentQANMongoDBProfilerAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"qan_mongodb_profiler_agent": {
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

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `QAN MongoDB Profiler agent configuration updated.
Agent ID              : test-agent-qan-mongodb-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : mongodb-service-456
Username              : testuser
TLS enabled           : true
Skip TLS verification : true
Max query length      : 1024

Disabled              : false
Custom labels         : env=test, team=qan
Process exec path     : /usr/bin/pmm-agent
Log level             : info
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - enabled TLS
  - enabled TLS skip verification
  - updated TLS certificate key password
  - changed authentication mechanism to MONGODB-CR
  - changed authentication database to testdb
  - changed max query length to 1024
  - changed log level to info
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-qan-mongodb", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentQANMongoDBProfilerAgentCommand{
			AgentID: "invalid-agent-qan-mongodb",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-mongodb-minimal", `{"qan_mongodb_profiler_agent": {"agent_id": "test-agent-qan-mongodb-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "qan-mongodb-profiler-agent", "test-agent-qan-mongodb-minimal"}

		var cmd ChangeAgentQANMongoDBProfilerAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty qan_mongodb_profiler_agent object when no flags are set
		expectedJSON := `{
			"qan_mongodb_profiler_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mongodb-profiler-agent"}

			var cmd ChangeAgentQANMongoDBProfilerAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-mongodb-profiler-agent", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentQANMongoDBProfilerAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
