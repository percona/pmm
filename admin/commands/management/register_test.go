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
