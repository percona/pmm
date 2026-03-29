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

	"github.com/stretchr/testify/require"
)

func TestBuildOtelCollectorConfigYAML(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90)

	require.Contains(t, yaml, "receivers:")
	require.Contains(t, yaml, "processors:")
	require.Contains(t, yaml, "exporters:")
	require.Contains(t, yaml, "service:")

	require.Contains(t, yaml, "otlp:")
	require.Contains(t, yaml, "endpoint: 0.0.0.0:4317")
	require.Contains(t, yaml, "endpoint: 0.0.0.0:4318")

	require.Contains(t, yaml, "clickhouse/logs:")
	require.Contains(t, yaml, "clickhouse/traces:")
	require.Contains(t, yaml, "clickhouse/metrics:")
	require.Contains(t, yaml, "logs_table_name: logs")
	require.Contains(t, yaml, "traces_table_name: otel_traces")
	require.Contains(t, yaml, "otel_metrics_sum")
	require.Contains(t, yaml, "create_schema: false")

	require.Contains(t, yaml, "pipelines:")
	require.Contains(t, yaml, "exporters: [clickhouse/logs]")
	require.Contains(t, yaml, "exporters: [clickhouse/traces]")
	require.Contains(t, yaml, "exporters: [clickhouse/metrics]")
	require.Contains(t, yaml, "ttl: 168h", "logs and traces 7d")
	require.Contains(t, yaml, "ttl: 2160h", "metrics 90d")
}

func TestBuildOtelCollectorConfigYAML_CustomParams(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("ch-host:9000", "myuser", "mypass", 14, 3, 30)

	require.Contains(t, yaml, "endpoint: tcp://ch-host:9000")
	require.Contains(t, yaml, "username: 'myuser'")
	require.Contains(t, yaml, "password: 'mypass'")
	require.True(t, strings.Contains(yaml, "ttl: 336h") || strings.Contains(yaml, "ttl: 72h"))
}

func TestBuildOtelCollectorConfigYAML_Defaults(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("", "", "", 0, 0, 0)

	require.Contains(t, yaml, "endpoint: tcp://127.0.0.1:9000")
	require.Contains(t, yaml, "username: 'default'")
}

func TestQuoteYAMLString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "'simple'", quoteYAMLString("simple"))
	require.Equal(t, "''''", quoteYAMLString("'"))
	require.Equal(t, "'a''b'", quoteYAMLString("a'b"))
}

func TestBuildOtelCollectorConfigYAML_ValidYAMLStructure(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90)

	require.Contains(t, yaml, "receivers:")
	require.Contains(t, yaml, "processors:")
	require.Contains(t, yaml, "exporters:")
	require.Contains(t, yaml, "service:")
}
