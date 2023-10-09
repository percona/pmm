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

package supervisord

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavePMMConfig(t *testing.T) {
	t.Parallel()
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	tests := []struct {
		description string
		params      map[string]any
		file        string
	}{
		{
			description: "disable internal postgresql db",
			params:      map[string]any{"DisableInternalDB": true, "DisableSupervisor": false, "DisableInternalClickhouse": false},
			file:        "pmm-db_disabled",
		},
		{
			description: "enable internal postgresql db",
			params:      map[string]any{"DisableInternalDB": false, "DisableSupervisor": false, "DisableInternalClickhouse": false},
			file:        "pmm-db_enabled",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			expected, err := os.ReadFile(filepath.Join(configDir, test.file+".ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := marshalConfig(test.params)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}
