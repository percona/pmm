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
			params:      map[string]any{"DisableInternalDB": true, "DisableSupervisor": false, "DisableInternalClickhouse": false, "PassivePMM": false},
			file:        "pmm-db_disabled",
		},
		{
			description: "enable internal postgresql db",
			params:      map[string]any{"DisableInternalDB": false, "DisableSupervisor": false, "DisableInternalClickhouse": false, "PassivePMM": false},
			file:        "pmm-db_enabled",
		},
		{
			description: "passive pmm",
			params:      map[string]any{"DisableInternalDB": true, "DisableSupervisor": false, "DisableInternalClickhouse": false, "PassivePMM": true},
			file:        "pmm-passive",
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
