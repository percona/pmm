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
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

func TestChangeAgentRTAMySQLAgentCommand(t *testing.T) {
	t.Parallel()

	t.Run("AllFlags", func(t *testing.T) {
		t.Parallel()

		const agentID = "test-agent-rta-mysql-all-flags"
		var capturedRequestBody string
		cleanup := setupChangeAgentTestServer(t, agentID,
			`{"rta_mysql_agent": {"agent_id": "`+agentID+`"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "rta-mysql-agent", agentID,
			"--enable",
			"--username=testuser",
			"--password=testpass",
			"--tls",
			"--tls-skip-verify",
			// 1m is deliberate: time.Duration.String() renders it "1m0s", which the server rejects.
			"--collect-interval=1m",
			"--log-level=info",
			"--custom-labels=env=test,team=rta",
			"--skip-connection-check",
		}

		var cmd ChangeAgentRTAMySQLAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		assert.JSONEq(t, `{
			"rta_mysql_agent": {
				"enable": true,
				"username": "testuser",
				"password": "testpass",
				"tls": true,
				"tls_skip_verify": true,
				"log_level": "LOG_LEVEL_INFO",
				"skip_connection_check": true,
				"custom_labels": {
					"values": {
						"env": "test",
						"team": "rta"
					}
				},
				"rta_options": {
					"collect_interval": "60s"
				}
			}
		}`, capturedRequestBody)
	})

	// Only the flags the user passed may appear in the request; anything else
	// would overwrite a field the user did not mean to change.
	t.Run("OnlyProvidedFlagsAreSent", func(t *testing.T) {
		t.Parallel()

		const agentID = "test-agent-rta-mysql-partial"
		var capturedRequestBody string
		cleanup := setupChangeAgentTestServer(t, agentID,
			`{"rta_mysql_agent": {"agent_id": "`+agentID+`"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{
			"change-agent", "rta-mysql-agent", agentID,
			"--username=only-user",
		}

		var cmd ChangeAgentRTAMySQLAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		_, err = cmd.RunCmd()
		require.NoError(t, err)

		assert.JSONEq(t, `{"rta_mysql_agent": {"username": "only-user"}}`, capturedRequestBody)
	})

	t.Run("MissingTLSFileIsReported", func(t *testing.T) {
		t.Parallel()

		var cmd ChangeAgentRTAMySQLAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse([]string{
			"test-agent-rta-mysql-missing-file",
			"--tls-ca-file=/non/existent/ca.pem",
		})
		require.NoError(t, err)

		_, err = cmd.RunCmd()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "/non/existent/ca.pem")
	})

	t.Run("ResultTemplate", func(t *testing.T) {
		t.Parallel()

		t.Run("WithCollectInterval", func(t *testing.T) {
			t.Parallel()

			res := &changeAgentRTAMySQLAgentResult{
				Agent: &agents.ChangeAgentOKBodyRtaMysqlAgent{
					AgentID:  "agent-id",
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
					RtaOptions: &agents.ChangeAgentOKBodyRtaMysqlAgentRtaOptions{
						CollectInterval: "10s",
					},
				},
			}

			assert.Contains(t, res.String(), "Collect interval      : 10s")
		})

		// ToAPIRTAOptions returns nil for empty options, and RenderTemplate
		// panics on a nil dereference, so the field's absence has to render.
		t.Run("WithoutRTAOptions", func(t *testing.T) {
			t.Parallel()

			res := &changeAgentRTAMySQLAgentResult{
				Agent: &agents.ChangeAgentOKBodyRtaMysqlAgent{
					AgentID:  "agent-id",
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			}

			out := res.String()
			assert.NotContains(t, out, "Collect interval")
			assert.Contains(t, out, "Agent ID              : agent-id")
			assert.Contains(t, out, "Log level")
		})
	})
}
