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

package otel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestIndentYAML(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, IndentYAML("", "  "))
	})
	t.Run("single_line", func(t *testing.T) {
		assert.Equal(t, "    - type: foo\n", IndentYAML("- type: foo", "    "))
	})
	t.Run("multi_line", func(t *testing.T) {
		block := "- type: key_value_parser\n  parse_from: body\n"
		got := IndentYAML(block, "      ")
		assert.True(t, strings.HasPrefix(got, "      - type: key_value_parser"))
		assert.Contains(t, got, "        parse_from: body")
	})
}

func TestBuildServerOtelConfigYAML(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	t.Run("config_includes_otlp_and_structure", func(t *testing.T) {
		// Migrations populate log_parser_presets, so we may get full config (with filelog) or receiver-only; either is valid.
		yaml, err := BuildServerOtelConfigYAML(db.Querier, "127.0.0.1:9000", "default", "clickhouse", 7)
		require.NoError(t, err)
		assert.Contains(t, yaml, "receivers:")
		assert.Contains(t, yaml, "otlp:")
		assert.Contains(t, yaml, "endpoint: 0.0.0.0:4317")
		assert.Contains(t, yaml, "processors:")
		assert.Contains(t, yaml, "memory_limiter:")
		assert.Contains(t, yaml, "transform:")
		assert.Contains(t, yaml, "exporters:")
		assert.Contains(t, yaml, "clickhouse:")
		// Logs pipeline must include otlp (either "[otlp]" or "[..., otlp]").
		assert.Contains(t, yaml, ", otlp]")
	})

	t.Run("full_config_with_presets", func(t *testing.T) {
		// Release the first DB so testdb.Open can DROP/CREATE the same database.
		_ = sqlDB.Close()
		// Use DB with fixtures so log_parser_presets has rows.
		sqlDB2 := testdb.Open(t, models.SetupFixtures, nil)
		db2 := reform.NewDB(sqlDB2, postgresql.Dialect, nil)
		t.Cleanup(func() { _ = sqlDB2.Close() })

		yaml, err := BuildServerOtelConfigYAML(db2.Querier, "127.0.0.1:9000", "ch", "secret", 14)
		require.NoError(t, err)
		assert.Contains(t, yaml, "receivers:")
		assert.Contains(t, yaml, "otlp:")
		assert.Contains(t, yaml, "filelog/server_")
		assert.Contains(t, yaml, "start_at: beginning")
		assert.Contains(t, yaml, "include:")
		assert.Contains(t, yaml, "/srv/logs/")
		assert.Contains(t, yaml, "operators:")
		assert.Contains(t, yaml, "processors:")
		assert.Contains(t, yaml, "clickhouse:")
		assert.Contains(t, yaml, "receivers: [")
		assert.Contains(t, yaml, ", otlp]")
	})
}
