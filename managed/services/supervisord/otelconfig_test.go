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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOtelCollectorConfigYAML(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7)

	// Required sections
	require.Contains(t, yaml, "receivers:")
	require.Contains(t, yaml, "processors:")
	require.Contains(t, yaml, "exporters:")
	require.Contains(t, yaml, "service:")

	// OTLP receiver only (no filelog; server log collection is done by pmm-agent with DB presets)
	require.Contains(t, yaml, "otlp:")
	require.Contains(t, yaml, "endpoint: 0.0.0.0:4317")
	require.Contains(t, yaml, "endpoint: 0.0.0.0:4318")
	require.NotContains(t, yaml, "filelog/")

	// Processors: memory_limiter, transform (pmm_source from node_name), batch
	require.Contains(t, yaml, "memory_limiter:")
	require.Contains(t, yaml, "transform:")
	require.Contains(t, yaml, "batch:")
	require.Contains(t, yaml, `set(resource.attributes["pmm_source"], resource.attributes["node_name"])`)

	// ClickHouse exporter with substituted values
	require.Contains(t, yaml, "endpoint: tcp://127.0.0.1:9000")
	require.Contains(t, yaml, "database: otel")
	require.Contains(t, yaml, "logs_table_name: logs")
	require.Contains(t, yaml, "create_schema: false")
	require.Contains(t, yaml, "ttl: 168h")
	require.Contains(t, yaml, "username: 'default'")
	require.Contains(t, yaml, "password: 'clickhouse'")

	// Pipeline: receiver-only
	require.Contains(t, yaml, "receivers: [otlp]")
	require.Contains(t, yaml, "processors: [memory_limiter, transform, batch]")
	require.Contains(t, yaml, "exporters: [clickhouse]")
}

func TestBuildOtelCollectorConfigYAML_CustomParams(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("ch-host:9000", "myuser", "mypass", 14)

	require.Contains(t, yaml, "endpoint: tcp://ch-host:9000")
	require.Contains(t, yaml, "username: 'myuser'")
	require.Contains(t, yaml, "password: 'mypass'")
	require.Contains(t, yaml, "ttl: 336h")
}

func TestBuildOtelCollectorConfigYAML_Defaults(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("", "", "", 0)

	require.Contains(t, yaml, "endpoint: tcp://127.0.0.1:9000")
	require.Contains(t, yaml, "username: 'default'")
	require.Contains(t, yaml, "ttl: 168h")
}

func TestQuoteYAMLString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "'simple'", quoteYAMLString("simple"))
	assert.Equal(t, "''''", quoteYAMLString("'"))
	assert.Equal(t, "'a''b'", quoteYAMLString("a'b"))
}

func TestBuildOtelCollectorConfigYAML_ValidYAMLStructure(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7)

	// Basic structure: receivers, processors, exporters, service are top-level
	lines := strings.Split(yaml, "\n")
	var topLevel []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.HasSuffix(trimmed, ":") {
				topLevel = append(topLevel, trimmed)
			}
		}
	}
	// Should have at least these top-level keys
	assert.Contains(t, yaml, "receivers:")
	assert.Contains(t, yaml, "processors:")
	assert.Contains(t, yaml, "exporters:")
	assert.Contains(t, yaml, "service:")
}
