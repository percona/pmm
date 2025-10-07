// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package inventory

import (
	"os"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

func TestQANToggle(t *testing.T) {
	envVar, exists := os.LookupEnv("PMM_ENABLE_INTERNAL_PG_QAN")
	// List agents to find the internal PostgreSQL QAN agent
	listAgentsRes, err := client.Default.AgentsService.ListAgents(&agents.ListAgentsParams{
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)

	// Find the internal PostgreSQL QAN agent (the one attached to pmm-server-postgresql service)
	var internalPgQANAgent *agents.ListAgentsOKBodyQANPostgresqlPgstatementsAgentItems0
	for _, agent := range listAgentsRes.Payload.QANPostgresqlPgstatementsAgent {
		if agent.PMMAgentID == "pmm-server" {
			internalPgQANAgent = agent
			break
		}
	}

	// Skip test if the internal PostgreSQL QAN agent doesn't exist
	if internalPgQANAgent == nil {
		t.Skip("Internal PostgreSQL QAN agent not found")
		return
	}

	agentID := internalPgQANAgent.AgentID
	originalDisabledState := internalPgQANAgent.Disabled

	// Restore original state at the end
	defer func() {
		_, _ = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
			AgentID: agentID,
			Body: agents.ChangeAgentBody{
				QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
					Enable: pointer.ToBool(!originalDisabledState),
				},
			},
			Context: pmmapitests.Context,
		})
	}()
	if exists && envVar != "" {
		t.Run("FailWhenEnvVarSet", func(t *testing.T) {
			// When PMM_ENABLE_INTERNAL_PG_QAN is set, trying to toggle should fail
			// Try to enable it (opposite of current state)
			_, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						Enable: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})

			// Expect a FailedPrecondition error
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition,
				"QAN for PMM's internal PostgreSQL server is set to 1 via the PMM_ENABLE_INTERNAL_PG_QAN environment variable.")

			// Try to disable it as well
			_, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						Enable: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})

			// Expect a FailedPrecondition error
			pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition,
				"QAN for PMM's internal PostgreSQL server is set to 1 via the PMM_ENABLE_INTERNAL_PG_QAN environment variable.")
		})
	} else {
		t.Run("SucceedWhenEnvVarNotSet", func(t *testing.T) {
			// Try to enable QAN - should succeed
			changeRes, err := client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						Enable: pointer.ToBool(true),
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			require.NotNil(t, changeRes)
			assert.False(t, changeRes.Payload.QANPostgresqlPgstatementsAgent.Disabled)

			// Verify the setting was persisted
			getRes, err := client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.False(t, getRes.Payload.QANPostgresqlPgstatementsAgent.Disabled)

			// Try to disable QAN - should also succeed
			changeRes, err = client.Default.AgentsService.ChangeAgent(&agents.ChangeAgentParams{
				AgentID: agentID,
				Body: agents.ChangeAgentBody{
					QANPostgresqlPgstatementsAgent: &agents.ChangeAgentParamsBodyQANPostgresqlPgstatementsAgent{
						Enable: pointer.ToBool(false),
					},
				},
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			require.NotNil(t, changeRes)
			assert.True(t, changeRes.Payload.QANPostgresqlPgstatementsAgent.Disabled)

			// Verify the setting was persisted
			getRes, err = client.Default.AgentsService.GetAgent(&agents.GetAgentParams{
				AgentID: agentID,
				Context: pmmapitests.Context,
			})
			require.NoError(t, err)
			assert.True(t, getRes.Payload.QANPostgresqlPgstatementsAgent.Disabled)
		})
	}
}
