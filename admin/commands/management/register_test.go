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

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/managementpb/json/client/node"
)

func TestRegisterResult(t *testing.T) {
	tests := []struct {
		name   string
		result registerResult
		want   string
	}{
		{
			name: "Success",
			result: registerResult{
				PMMAgent: &node.RegisterNodeOKBodyPMMAgent{
					AgentID:      "/agent_id/new_id",
					RunsOnNodeID: "/node_id/second_id",
				},
				Warning: "",
			},
			want: `pmm-agent registered.
pmm-agent ID: /agent_id/new_id
Node ID     : /node_id/second_id
`,
		},
		{
			name: "With warning",
			result: registerResult{
				PMMAgent: &node.RegisterNodeOKBodyPMMAgent{
					AgentID:      "/agent_id/warning",
					RunsOnNodeID: "/node_id/warning_node",
				},
				Warning: "Couldn't create Admin API Key",
			},
			want: `pmm-agent registered.
pmm-agent ID: /agent_id/warning
Node ID     : /node_id/warning_node

Warning: Couldn't create Admin API Key
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.result.String(), "String()")
		})
	}
}
