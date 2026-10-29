// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package agents

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/otel"
	"github.com/percona/pmm/managed/utils/testdb"
)

// TestOtelCollectorOperatorsYAMLIndent verifies filelog operators are indented under
// "operators:" the same way as managed/otel.BuildServerLogs (flush-left DB YAML).
func TestOtelCollectorOperatorsYAMLIndent(t *testing.T) {
	raw := "- type: regex_parser\n  regex: '.*'\n  parse_from: body"
	merged := "    operators:\n" + otel.IndentYAML(raw, "      ")
	assert.Contains(t, merged, "    operators:\n")
	lines := strings.Split(strings.TrimSuffix(merged, "\n"), "\n")
	opIdx := -1
	for i, ln := range lines {
		if strings.HasPrefix(ln, "    operators:") {
			opIdx = i
			break
		}
	}
	require.NotEqual(t, -1, opIdx, "operators block missing")
	if opIdx+1 < len(lines) {
		first := lines[opIdx+1]
		assert.True(t, strings.HasPrefix(first, "      - type:"),
			"first list item must be nested under operators: got %q", first)
		assert.Greater(t, len(first), len("    operators:"),
			"list item must be indented deeper than operators key")
	}
}

func TestOtelCollectorConfig_PresetTag(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	q := reform.NewDB(sqlDB, postgresql.Dialect, nil).Querier

	row := &models.Agent{AgentID: "otel-1", AgentType: models.OtelCollectorType}
	require.NoError(t, row.SetCustomLabels(map[string]string{
		"log_sources": `[{"path":"/var/log/pg.log","preset":"postgres"},{"path":"/var/log/app.log","preset":"raw"}]`,
	}))

	cfg := otelCollectorConfig(row, q).TextFiles["otelconfig"]
	receiversYAML, _, found := strings.Cut(cfg, "\nprocessors:")
	require.True(t, found, cfg)
	var parsed struct {
		Receivers map[string]map[string]any `yaml:"receivers"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(receiversYAML), &parsed))

	tag := func(preset string) map[string]any {
		return map[string]any{"type": "add", "field": `attributes["log.preset"]`, "value": preset, "on_error": "send_quiet"}
	}
	pg, ok := parsed.Receivers["filelog/preset_postgres"]["operators"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(pg), 1, "preset operators come first")
	assert.Equal(t, tag("postgres"), pg[len(pg)-1])
	assert.Equal(t, []any{tag("raw")}, parsed.Receivers["filelog/preset_raw"]["operators"])
}
