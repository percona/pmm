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

func TestQANPostgreSQLPgStatMonitorAgentChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndSettings", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-pgstatmon-update", `{"qan_postgresql_pgstatmonitor_agent": {"agent_id": "test-agent-qan-pgstatmon-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand{
				AgentID:              "test-agent-qan-pgstatmon-update",
				Enable:               pointer.ToBool(true),
				Username:             pointer.ToString("postgres_user"),
				Password:             pointer.ToString("postgres_pass"),
				TLS:                  pointer.ToBool(true),
				TLSSkipVerify:        pointer.ToBool(false),
				MaxQueryLength:       pointer.ToInt32(4096),
				DisableQueryExamples: pointer.ToBool(false),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CommentsParsingChangeFlags: flags.CommentsParsingChangeFlags{
					CommentsParsing: pointer.ToString("off"),
				},
				CustomLabels: &map[string]string{"environment": "production", "service": "postgresql"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"qan_postgresql_pgstatmonitor_agent": {
					"enable": true,
					"username": "postgres_user",
					"password": "postgres_pass",
					"tls": true,
					"tls_skip_verify": false,
					"max_query_length": 4096,
					"disable_query_examples": false,
					"disable_comments_parsing": true,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "production",
							"service": "postgresql"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-pgstatmon-disable", `{"qan_postgresql_pgstatmonitor_agent": {"agent_id": "test-agent-qan-pgstatmon-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand{
				AgentID:              "test-agent-qan-pgstatmon-disable",
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
				"qan_postgresql_pgstatmonitor_agent": {
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
			"qan_postgresql_pgstatmonitor_agent": {
				"agent_id": "test-agent-qan-pgstatmon-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "postgres-service-456",
				"username": "testuser",
				"tls": true,
				"tls_skip_verify": true,
				"max_query_length": 1024,
				"query_examples_disabled": true,
				"disable_comments_parsing": true,
				"disabled": false,
				"custom_labels": {
					"env": "test",
					"team": "qan"
				},
				"process_exec_path": "/usr/bin/postgres",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-pgstatmon-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "qan-postgresql-pgstatmonitor-agent", "test-agent-qan-pgstatmon-all-flags",
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

		var cmd ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"qan_postgresql_pgstatmonitor_agent": {
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
		expectedOutput := `QAN PostgreSQL PgStatMonitor agent configuration updated.
Agent ID                     : test-agent-qan-pgstatmon-all-flags
PMM-Agent ID                 : pmm-agent-123
Service ID                   : postgres-service-456
Username                     : testuser
TLS enabled                  : true
Skip TLS verification        : true
Max query length             : 1024
Query examples disabled      : true
Disable comments parsing     : true

Disabled                     : false
Custom labels                : env=test, team=qan
Process exec path            : /usr/bin/postgres
Log level                    : info
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

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-qan-pgstatmon", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand{
			AgentID: "invalid-agent-qan-pgstatmon",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-qan-pgstatmon-minimal", `{"qan_postgresql_pgstatmonitor_agent": {"agent_id": "test-agent-qan-pgstatmon-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "qan-postgresql-pgstatmonitor-agent", "test-agent-qan-pgstatmon-minimal"}

		var cmd ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have empty qan_postgresql_pgstatmonitor_agent object when no flags are set
		expectedJSON := `{
			"qan_postgresql_pgstatmonitor_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-postgresql-pgstatmonitor-agent"}

			var cmd ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "qan-postgresql-pgstatmonitor-agent", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentQANPostgreSQLPgStatMonitorAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
