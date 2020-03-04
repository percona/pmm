// pmm-admin
// Copyright 2019 Percona LLC
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

package commands

import (
	"strings"
	"testing"

	"github.com/percona/pmm/api/inventorypb/types"
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
					{ServiceType: types.ServiceTypeMySQLService, ServiceID: "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a", ServiceName: "mysql-service"},
				},
				Agents: []listResultAgent{
					{AgentType: types.AgentTypeMySQLdExporter, AgentID: "/agent_id/8b732ac3-8256-40b0-a98b-0fd5fa9a1140", ServiceID: "/service_id/4ff49c41-80a1-4030-bc02-cd76e3b0b84a", Status: "RUNNING"},
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
