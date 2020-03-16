package inventorypb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/inventorypb/types"
)

// This test ensures that the NodetypeNames map that holds the human readable agent type
// names is up to date with the types defined in Nodetype_name by the proto definition
// by calling the NodetypeName function using the values from protobuf and it expects the
// result is a non-empty string, meaning that the NodetypeNames list matches the proto
// definitions
func TestNodeTypes(t *testing.T) {
	for _, val := range NodeType_name {
		if strings.HasSuffix(val, "INVALID") {
			continue
		}
		assert.NotEmpty(t, types.NodeTypeName(val))
	}
}
