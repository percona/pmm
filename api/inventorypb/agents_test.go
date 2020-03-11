package inventorypb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/inventorypb/types"
)

// This test ensures that the AgentTypeNames map that holds the human readable agent type
// names is up to date with the types defined in AgentType_name by the proto definition
// by calling the AgentTypeName function using the values from protobuf and it expects the
// result is a non-empty string, meaning that the AgentTypeNames list matches the proto
// definitions
func TestAgentTypes(t *testing.T) {
	for _, val := range AgentType_name {
		if strings.HasSuffix(val, "INVALID") {
			continue
		}
		assert.NotEmpty(t, types.AgentTypeName(val))
	}
}
