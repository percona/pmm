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

func TestProxySQLExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndTLS", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-proxysql-update", `{"proxysql_exporter": {"agent_id": "test-agent-proxysql-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentProxysqlExporterCommand{
				AgentID:           "test-agent-proxysql-update",
				Enable:            pointer.ToBool(true),
				Username:          pointer.ToString("proxysql_user"),
				Password:          pointer.ToString("proxysql_pass"),
				AgentPassword:     pointer.ToString("agent_pass"),
				TLS:               pointer.ToBool(true),
				TLSSkipVerify:     pointer.ToBool(false),
				DisableCollectors: []string{"connection_pool", "stats_mysql_global"},
				ExposeExporter:    pointer.ToBool(true),
				PushMetrics:       pointer.ToBool(true),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"environment": "staging", "service": "proxysql"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"proxysql_exporter": {
					"enable": true,
					"username": "proxysql_user",
					"password": "proxysql_pass",
					"agent_password": "agent_pass",
					"tls": true,
					"tls_skip_verify": false,
					"disable_collectors": ["connection_pool", "stats_mysql_global"],
					"expose_exporter": true,
					"enable_push_metrics": true,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "staging",
							"service": "proxysql"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-proxysql-disable", `{"proxysql_exporter": {"agent_id": "test-agent-proxysql-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentProxysqlExporterCommand{
				AgentID:        "test-agent-proxysql-disable",
				Enable:         pointer.ToBool(false),
				ExposeExporter: pointer.ToBool(false),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"proxysql_exporter": {
					"enable": false,
					"expose_exporter": false,
					"disable_collectors": null
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"proxysql_exporter": {
				"agent_id": "test-agent-proxysql-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "proxysql-service-456",
				"username": "testuser",
				"listen_port": 42004,
				"tls": true,
				"tls_skip_verify": true,
				"push_metrics_enabled": true,
				"expose_exporter": true,
				"disabled": false,
				"custom_labels": {
					"env": "prod",
					"team": "db"
				},
				"process_exec_path": "/usr/bin/proxysql_exporter",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-proxysql-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "proxysql-exporter", "test-agent-proxysql-all-flags",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--agent-password=agentpass",
			"--tls",
			"--tls-skip-verify",
			"--disable-collectors=stats_mysql_commands,stats_mysql_connection_pool",
			"--expose-exporter",
			"--push-metrics",
			"--log-level=info",
			"--custom-labels=env=prod,team=db",
		}

		var cmd ChangeAgentProxysqlExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"proxysql_exporter": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"agent_password": "agentpass",
				"tls": true,
				"tls_skip_verify": true,
				"disable_collectors": ["stats_mysql_commands", "stats_mysql_connection_pool"],
				"expose_exporter": true,
				"enable_push_metrics": true,
				"log_level": "LOG_LEVEL_INFO",
				"custom_labels": {
					"values": {
						"env": "prod",
						"team": "db"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `ProxySQL Exporter agent configuration updated.
Agent ID              : test-agent-proxysql-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : proxysql-service-456
Username              : testuser
Listen port           : 42004
TLS enabled           : true
Skip TLS verification : true
Push metrics enabled  : true
Expose exporter       : true

Disabled              : false
Custom labels         : env=prod, team=db
Process exec path     : /usr/bin/proxysql_exporter
Log level             : info
Configuration changes applied:
  - enabled agent
  - updated username
  - updated password
  - updated agent password
  - enabled TLS
  - enabled TLS skip verification
  - updated disabled collectors: [stats_mysql_commands stats_mysql_connection_pool]
  - enabled expose exporter
  - enabled push metrics
  - changed log level to info
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-proxysql", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentProxysqlExporterCommand{
			AgentID: "invalid-agent-proxysql",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-proxysql-minimal", `{"proxysql_exporter": {"agent_id": "test-agent-proxysql-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "proxysql-exporter", "test-agent-proxysql-minimal", "--enable"}

		var cmd ChangeAgentProxysqlExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should only include enable field, no log_level when not set
		assert.Contains(t, capturedRequestBody, `"enable":true`)
		assert.NotContains(t, capturedRequestBody, `"log_level"`)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "proxysql-exporter"}

			var cmd ChangeAgentProxysqlExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "proxysql-exporter", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentProxysqlExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
