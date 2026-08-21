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

	"github.com/stretchr/testify/assert"

	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

// The server omits RTA options entirely when none are set - ToAPIRTAOptions
// returns nil for empty options - and RenderTemplate panics on a nil pointer
// dereference, so both result templates have to tolerate the field's absence.
func TestRTAMySQLAgentResultTemplates(t *testing.T) {
	t.Parallel()

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		t.Run("WithCollectInterval", func(t *testing.T) {
			t.Parallel()

			res := &addAgentRTAMySQLAgentResult{
				Agent: &agents.AddAgentOKBodyRtaMysqlAgent{
					AgentID:  "agent-id",
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
					RtaOptions: &agents.AddAgentOKBodyRtaMysqlAgentRtaOptions{
						CollectInterval: "5s",
					},
				},
			}

			out := res.String()
			assert.Contains(t, out, "Collect interval      : 5s")
		})

		t.Run("WithoutRTAOptions", func(t *testing.T) {
			t.Parallel()

			res := &addAgentRTAMySQLAgentResult{
				Agent: &agents.AddAgentOKBodyRtaMysqlAgent{
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

	t.Run("Change", func(t *testing.T) {
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

			out := res.String()
			assert.Contains(t, out, "Collect interval      : 10s")
		})

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
