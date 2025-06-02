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

func TestRDSExporterChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("UpdateCredentialsAndMetrics", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-rds-update", `{"rds_exporter": {"agent_id": "test-agent-rds-update"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentRDSExporterCommand{
				AgentID:                "test-agent-rds-update",
				Enable:                 pointer.ToBool(true),
				AWSAccessKey:           pointer.ToString("AKIAIOSFODNN7EXAMPLE"),
				AWSSecretKey:           pointer.ToString("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"),
				DisableBasicMetrics:    pointer.ToBool(false),
				DisableEnhancedMetrics: pointer.ToBool(true),
				PushMetrics:            pointer.ToBool(true),
				LogLevelFatalChangeFlags: flags.LogLevelFatalChangeFlags{
					LogLevel: pointer.To(flags.LogLevel("debug")),
				},
				CustomLabels: &map[string]string{"environment": "production", "region": "us-west-2"},
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"rds_exporter": {
					"enable": true,
					"aws_access_key": "AKIAIOSFODNN7EXAMPLE",
					"aws_secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					"disable_basic_metrics": false,
					"disable_enhanced_metrics": true,
					"enable_push_metrics": true,
					"log_level": "LOG_LEVEL_DEBUG",
					"custom_labels": {
						"values": {
							"environment": "production",
							"region": "us-west-2"
						}
					}
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-rds-disable", `{"rds_exporter": {"agent_id": "test-agent-rds-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentRDSExporterCommand{
				AgentID:             "test-agent-rds-disable",
				Enable:              pointer.ToBool(false),
				DisableBasicMetrics: pointer.ToBool(true),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"rds_exporter": {
					"enable": false,
					"disable_basic_metrics": true
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)
		})
	})

	t.Run("ComprehensiveAllFieldsValidation", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"rds_exporter": {
				"agent_id": "test-agent-rds-all-flags",
				"pmm_agent_id": "pmm-agent-123",
				"node_id": "node-456",
				"listen_port": 9042,
				"push_metrics_enabled": true,
				"disabled": false,
				"basic_metrics_disabled": true,
				"enhanced_metrics_disabled": true,
				"custom_labels": {
					"env": "test",
					"team": "devops"
				},
				"process_exec_path": "/usr/bin/rds_exporter",
				"log_level": "LOG_LEVEL_INFO"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-rds-all-flags", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "rds-exporter", "test-agent-rds-all-flags",
			"--enable",
			"--aws-access-key=AKIATEST123",
			"--aws-secret-key=secretkey123",
			"--disable-basic-metrics",
			"--disable-enhanced-metrics",
			"--push-metrics",
			"--log-level=info",
			"--custom-labels=env=test,team=devops",
		}

		var cmd ChangeAgentRDSExporterCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"rds_exporter": {
				"enable": true,
				"aws_access_key": "AKIATEST123",
				"aws_secret_key": "secretkey123",
				"disable_basic_metrics": true,
				"disable_enhanced_metrics": true,
				"enable_push_metrics": true,
				"log_level": "LOG_LEVEL_INFO",
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "devops"
					}
				}
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()

		// Expected complete output with all fields and changes
		expectedOutput := `RDS Exporter agent configuration updated.
Agent ID                   : test-agent-rds-all-flags
PMM-Agent ID               : pmm-agent-123
Node ID                    : node-456
Listen port                : 9042
Push metrics enabled       : true

Disabled                   : false
Basic metrics disabled     : true
Enhanced metrics disabled  : true
Custom labels              : env=test, team=devops
Process exec path          : /usr/bin/rds_exporter
Log level                  : info
Configuration changes applied:
  - enabled agent
  - updated AWS access key
  - updated AWS secret key
  - disabled basic metrics
  - disabled enhanced metrics
  - enabled push metrics
  - changed log level to info
  - updated custom labels
`

		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-rds", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentRDSExporterCommand{
			AgentID: "invalid-agent-rds",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-rds-minimal", `{"rds_exporter": {"agent_id": "test-agent-rds-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "rds-exporter", "test-agent-rds-minimal", "--enable"}

		var cmd ChangeAgentRDSExporterCommand
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

			cli := []string{"change-agent", "rds-exporter"}

			var cmd ChangeAgentRDSExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidLogLevel", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "rds-exporter", "test-agent-id", "--log-level=invalid"}

			var cmd ChangeAgentRDSExporterCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "log-level")
		})
	})
}
