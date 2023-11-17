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
// definitions.
func TestNodeTypes(t *testing.T) {
	for _, val := range NodeType_name {
		if strings.HasSuffix(val, "UNSPECIFIED") {
			continue
		}
		assert.NotEmpty(t, types.NodeTypeName(val))
	}
}
