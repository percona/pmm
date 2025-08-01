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
	"github.com/stretchr/testify/assert"

	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

func TestAddAgentMysqldExporter(t *testing.T) {
	t.Run("TablestatEnabled", func(t *testing.T) {
		res := &addAgentMysqldExporterResult{
			Agent: &agents.AddAgentOKBodyMysqldExporter{
				AgentID:    "1",
				PMMAgentID: "2",
				Username:   "username",
				ServiceID:  "1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),

				TablestatsGroupTableLimit: 1000,
				TablestatsGroupDisabled:   false,
			},
			TableCount: 500,
		}
		expected := strings.TrimSpace(`
Mysqld Exporter added.
Agent ID              : 1
PMM-Agent ID          : 2
Service ID            : 1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
Extra DSN params      : map[]

Tablestat collectors  : enabled (the limit is 1000, the actual table count is 500).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatEnabledNoLimit", func(t *testing.T) {
		res := &addAgentMysqldExporterResult{
			Agent: &agents.AddAgentOKBodyMysqldExporter{
				AgentID:    "1",
				PMMAgentID: "2",
				Username:   "username",
				ServiceID:  "1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),

				TablestatsGroupTableLimit: 0,
				TablestatsGroupDisabled:   false,
			},
			TableCount: 2000,
		}
		expected := strings.TrimSpace(`
Mysqld Exporter added.
Agent ID              : 1
PMM-Agent ID          : 2
Service ID            : 1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
Extra DSN params      : map[]

Tablestat collectors  : enabled (the table count limit is not set).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatEnabledUnknown", func(t *testing.T) {
		res := &addAgentMysqldExporterResult{
			Agent: &agents.AddAgentOKBodyMysqldExporter{
				AgentID:    "1",
				PMMAgentID: "2",
				Username:   "username",
				ServiceID:  "1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),

				TablestatsGroupTableLimit: 1000,
				TablestatsGroupDisabled:   false,
			},
			TableCount: 0,
		}
		expected := strings.TrimSpace(`
Mysqld Exporter added.
Agent ID              : 1
PMM-Agent ID          : 2
Service ID            : 1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
Extra DSN params      : map[]

Tablestat collectors  : enabled (the limit is 1000, the actual table count is unknown).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatDisabled", func(t *testing.T) {
		res := &addAgentMysqldExporterResult{
			Agent: &agents.AddAgentOKBodyMysqldExporter{
				AgentID:    "1",
				PMMAgentID: "2",
				Username:   "username",
				ServiceID:  "1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),

				TablestatsGroupTableLimit: 1000,
				TablestatsGroupDisabled:   true,
			},
			TableCount: 2000,
		}
		expected := strings.TrimSpace(`
Mysqld Exporter added.
Agent ID              : 1
PMM-Agent ID          : 2
Service ID            : 1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
Extra DSN params      : map[]

Tablestat collectors  : disabled (the limit is 1000, the actual table count is 2000).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("TablestatDisabledAlways", func(t *testing.T) {
		res := &addAgentMysqldExporterResult{
			Agent: &agents.AddAgentOKBodyMysqldExporter{
				AgentID:    "1",
				PMMAgentID: "2",
				Username:   "username",
				ServiceID:  "1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),

				TablestatsGroupTableLimit: -1,
				TablestatsGroupDisabled:   true,
			},
			TableCount: 2000,
		}
		expected := strings.TrimSpace(`
Mysqld Exporter added.
Agent ID              : 1
PMM-Agent ID          : 2
Service ID            : 1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
Extra DSN params      : map[]

Tablestat collectors  : disabled (always).
		`)
		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})

	t.Run("with allowCleartextPasswords DSN param", func(t *testing.T) {
		res := &addAgentMysqldExporterResult{
			Agent: &agents.AddAgentOKBodyMysqldExporter{
				AgentID:    "1",
				PMMAgentID: "2",
				Username:   "username",
				ServiceID:  "1",
				ListenPort: 42001,
				Status:     pointer.ToString("RUNNING"),

				ExtraDsnParams: map[string]string{
					"allowCleartextPasswords": "1",
				},
			},
			TableCount: 0,
		}
		expected := strings.TrimSpace(`
Mysqld Exporter added.
Agent ID              : 1
PMM-Agent ID          : 2
Service ID            : 1
Username              : username
Listen port           : 42001
TLS enabled           : false
Skip TLS verification : false

Status                : RUNNING
Disabled              : false
Custom labels         : map[]
Extra DSN params      : map[allowCleartextPasswords:1]

Tablestat collectors  : enabled (the table count limit is not set).
		`)

		assert.Equal(t, expected, strings.TrimSpace(res.String()))
	})
}
