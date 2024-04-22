// Copyright (C) 2024 Percona LLC
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
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/inventorypb/json/client/agents"
)

func TestAddAgentPostgresExporter(t *testing.T) {
	t.Run("TablestatEnabled", func(t *testing.T) {
		res := &addAgentPostgresExporterResult{
			Agent: &agents.AddPostgresExporterOKBodyPostgresExporter{
				AgentID:    "/agent_id/1",
				PMMAgentID: "/agent_id/2",
				Username:   "username",
				ServiceID:  "/service_id/1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),
			},
		}
		expected := strings.TrimSpace(`
Postgres Exporter added.
Agent ID              : /agent_id/1
PMM-Agent ID          : /agent_id/2
Service ID            : /service_id/1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})
}
