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

func TestPostgresExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndTLS", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-update", `{"postgres_exporter": {"agent_id": "test-agent-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentPostgresExporterCommand{
				AgentID:           "test-agent-update",
				Enable:            pointer.ToBool(true),
				Username:          pointer.ToString("newuser"),
				Password:          pointer.ToString("newpass"),
				TLS:               pointer.ToBool(true),
				TLSSkipVerify:     pointer.ToBool(false),
				DisableCollectors: []string{"locks", "replication"},
				ExposeExporter:    pointer.ToBool(true),
				PushMetrics:       pointer.ToBool(false),
				LogLevelNoFatalChangeFlags: flags.LogLevelNoFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"environment": "test", "team": "backend"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"postgres_exporter": {
					"enable": true,
					"username": "newuser",
					"password": "newpass",
					"tls": true,
					"tls_skip_verify": false,
					"disable_collectors": ["locks", "replication"],
					"expose_exporter": true,
					"enable_push_metrics": false,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "test",
							"team": "backend"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-disable", `{"postgres_exporter": {"agent_id": "test-agent-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentPostgresExporterCommand{
				AgentID:     "test-agent-disable",
				Enable:      pointer.ToBool(false),
				PushMetrics: pointer.ToBool(true),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"postgres_exporter": {
					"enable": false,
					"enable_push_metrics": true,
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
			"postgres_exporter": {
				"agent_id": "test-agent-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"service_id": "postgres-service-456",
				"username": "testuser",
				"listen_port": 9187,
				"tls": true,
				"tls_skip_verify": true,
				"expose_exporter": true,
				"push_metrics_enabled": true,
				"disabled": false,
				"custom_labels": {
					"env": "prod",
					"team": "db"
				},
				"process_exec_path": "/usr/bin/postgres_exporter",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "postgres-exporter", "test-agent-all-flags",
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--agent-password=agentpass",
			"--tls",
			"--tls-skip-verify",
			"--disable-collectors=locks,replication",
			"--expose-exporter",
			"--push-metrics",
			"--log-level=info",
			"--custom-labels=env=prod,team=db",
		}

		var cmd ChangeAgentPostgresExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"postgres_exporter": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"agent_password": "agentpass",
				"tls": true,
				"tls_skip_verify": true,
				"disable_collectors": ["locks", "replication"],
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
		expectedOutput := `PostgreSQL Exporter agent configuration updated.
Agent ID              : test-agent-all-flags
PMM-Agent ID          : pmm-agent-123
Service ID            : postgres-service-456
Username              : testuser
Listen port           : 9187
TLS enabled           : true
Skip TLS verification : true
Push metrics enabled  : true
Expose exporter       : true

Disabled              : false
Custom labels         : env=prod, team=db
Process exec path     : /usr/bin/postgres_exporter
Log level             : info
Configuration changes applied:
  - updated custom labels
  - enabled agent
  - updated username
  - updated password
  - updated agent password
  - enabled TLS
  - enabled TLS skip verification
  - updated disabled collectors: [locks replication]
  - enabled expose exporter
  - enabled push metrics
  - changed log level to info
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentPostgresExporterCommand{
			AgentID: "invalid-agent",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-minimal", `{"postgres_exporter": {"agent_id": "test-agent-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "postgres-exporter", "test-agent-minimal", "--enable"}

		var cmd ChangeAgentPostgresExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		expectedJSON := `{
			"postgres_exporter": {
				"enable": true,
				"disable_collectors": null
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "postgres-exporter"}

			var cmd ChangeAgentPostgresExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "postgres-exporter", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentPostgresExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
