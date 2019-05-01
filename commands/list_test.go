// pmm-admin
// Copyright (C) 2018 Percona LLC
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

package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListResultString(t *testing.T) {
	tests := []struct {
		name       string
		listResult listResult
		expected   string
	}{
		{
			name: "filled",
			listResult: listResult{
				Services: []listResultService{
					{ServiceType: "MySQL", ServiceID: "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a", ServiceName: "mysql-service"},
				},
				Agents: []listResultAgent{
					{AgentType: "mysqld_exporter", AgentID: "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140", ServiceID: "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a", Status: "RUNNING"},
				},
			},
			expected: strings.TrimSpace(`
Service type  Service name         Address and port  Service ID
MySQL         mysql-service                          /service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a

Agent type                  Status     Agent ID                                        Service ID
mysqld_exporter             RUNNING    /agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140  /service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a
`),
		},
		{
			name:       "empty",
			listResult: listResult{},
			expected: strings.TrimSpace(`
Service type  Service name         Address and port  Service ID

Agent type                  Status     Agent ID                                        Service ID
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := strings.TrimSpace(tt.listResult.String())
			assert.Equal(t, tt.expected, actual)
		})
	}
}
