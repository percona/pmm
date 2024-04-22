// Copyright (C) 2024 Percona LLC
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

package models

import (
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode(t *testing.T) {
	t.Run("UnifiedLabels", func(t *testing.T) {
		node := &Node{
			NodeID:       "node_id",
			Region:       pointer.ToString("hidden"),
			AZ:           "removed",
			CustomLabels: []byte(`{"region": "region1", "az": "  "}`),
		}
		actual, err := node.UnifiedLabels()
		require.NoError(t, err)
		expected := map[string]string{
			"node_id": "node_id",
			"region":  "region1",
		}
		assert.Equal(t, expected, actual)
	})
}
