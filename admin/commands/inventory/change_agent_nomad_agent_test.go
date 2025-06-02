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
)

func TestNomadAgentChangeAgent(t *testing.T) {
	t.Parallel()

	t.Run("CoreFunctionality", func(t *testing.T) {
		t.Parallel()

		t.Run("EnableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-nomad-enable", `{"nomad_agent": {"agent_id": "test-agent-nomad-enable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentNomadAgentCommand{
				AgentID: "test-agent-nomad-enable",
				Enable:  pointer.ToBool(true),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"nomad_agent": {
					"enable": true
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)

			// Verify change tracking
			output := result.String()
			assert.Contains(t, output, "enabled agent")
		})

		t.Run("DisableAgent", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-nomad-disable", `{"nomad_agent": {"agent_id": "test-agent-nomad-disable"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentNomadAgentCommand{
				AgentID: "test-agent-nomad-disable",
				Enable:  pointer.ToBool(false),
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			expectedJSON := `{
				"nomad_agent": {
					"enable": false
				}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)

			// Verify change tracking
			output := result.String()
			assert.Contains(t, output, "disabled agent")
		})

		t.Run("NoChangeWhenNoFlags", func(t *testing.T) {
			var capturedRequestBody string
			_, cleanup := setupChangeAgentTestServer(t, "test-agent-nomad-nochange", `{"nomad_agent": {"agent_id": "test-agent-nomad-nochange"}}`, &capturedRequestBody)
			defer cleanup()

			cmd := &ChangeAgentNomadAgentCommand{
				AgentID: "test-agent-nomad-nochange",
				// No Enable flag set - should send empty body except for agent ID
			}

			result, err := cmd.RunCmd()
			require.NoError(t, err)
			assert.NotNil(t, result)

			// Should have empty nomad_agent object (just the wrapper)
			expectedJSON := `{
				"nomad_agent": {}
			}`
			assert.JSONEq(t, expectedJSON, capturedRequestBody)

			// Should not show any configuration changes
			output := result.String()
			assert.NotContains(t, output, "Configuration changes applied:")
		})
	})

	t.Run("ComprehensiveKongParsingAndOutput", func(t *testing.T) {
		var capturedRequestBody string
		// Mock a comprehensive API response with all fields populated
		mockResponse := `{
			"nomad_agent": {
				"agent_id": "test-agent-nomad-comprehensive",
				"pmm_agent_id": "pmm-agent-123",
				"listen_port": 9090,
				"disabled": false,
				"process_exec_path": "/usr/bin/nomad"
			}
		}`
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-nomad-comprehensive", mockResponse, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "nomad-agent", "test-agent-nomad-comprehensive", "--enable"}

		var cmd ChangeAgentNomadAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Test API request JSON
		expectedJSON := `{
			"nomad_agent": {
				"enable": true
			}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)

		// Test output format with all fields
		output := result.String()
		expectedOutput := `Nomad Agent configuration updated.
Agent ID              : test-agent-nomad-comprehensive
PMM-Agent ID          : pmm-agent-123
Listen port           : 9090

Disabled              : false
Process exec path     : /usr/bin/nomad
Configuration changes applied:
  - enabled agent
`
		assert.Equal(t, expectedOutput, output)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		t.Parallel()

		_, cleanup := setupChangeAgentTestServer(t, "invalid-agent-nomad", `{"error": "Agent not found", "code": 404, "message": "Agent not found"}`, nil)
		defer cleanup()

		cmd := &ChangeAgentNomadAgentCommand{
			AgentID: "invalid-agent-nomad",
			Enable:  pointer.ToBool(true),
		}

		result, err := cmd.RunCmd()
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("KongParsingWithMinimalFlags", func(t *testing.T) {
		var capturedRequestBody string
		_, cleanup := setupChangeAgentTestServer(t, "test-agent-nomad-minimal", `{"nomad_agent": {"agent_id": "test-agent-nomad-minimal"}}`, &capturedRequestBody)
		defer cleanup()

		cli := []string{"change-agent", "nomad-agent", "test-agent-nomad-minimal"}

		var cmd ChangeAgentNomadAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse(cli[2:])
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should not contain enable field when not set
		expectedJSON := `{
			"nomad_agent": {}
		}`
		assert.JSONEq(t, expectedJSON, capturedRequestBody)
	})

	t.Run("KongParsingErrorCases", func(t *testing.T) {
		t.Parallel()

		t.Run("MissingRequiredArgument", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "nomad-agent"}

			var cmd ChangeAgentNomadAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "agent-id")
		})

		t.Run("InvalidBooleanValue", func(t *testing.T) {
			t.Parallel()

			cli := []string{"change-agent", "nomad-agent", "test-agent-id", "--enable=maybe"}

			var cmd ChangeAgentNomadAgentCommand
			parser, err := kong.New(&cmd)
			require.NoError(t, err)

			_, err = parser.Parse(cli[2:])
			assert.Error(t, err)
			assert.Contains(t, strings.ToLower(err.Error()), "enable")
		})
	})
}
