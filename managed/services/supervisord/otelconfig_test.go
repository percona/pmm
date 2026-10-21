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
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/otel"
)

func TestBuildOtelCollectorConfigYAML(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90, otel.ServerLogs{})

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

	yaml := buildOtelCollectorConfigYAML("ch-host:9000", "myuser", "mypass", 14, 3, 30, otel.ServerLogs{})

	require.Contains(t, yaml, "endpoint: tcp://ch-host:9000")
	require.Contains(t, yaml, "username: 'myuser'")
	require.Contains(t, yaml, "password: 'mypass'")
	require.True(t, strings.Contains(yaml, "ttl: 336h") || strings.Contains(yaml, "ttl: 72h"))
}

func TestBuildOtelCollectorConfigYAML_Defaults(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("", "", "", 0, 0, 0, otel.ServerLogs{})

	require.Contains(t, yaml, "endpoint: tcp://127.0.0.1:9000")
	require.Contains(t, yaml, "username: 'default'")
}

func TestBuildOtelCollectorConfigYAML_ServiceName(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90, otel.ServerLogs{})

	fromService := `set(resource.attributes["service.name"], resource.attributes["service_name"]) where resource.attributes["service.name"] == nil and resource.attributes["service_name"] != nil`
	fromNode := `set(resource.attributes["service.name"], resource.attributes["node_name"]) where resource.attributes["service.name"] == nil and resource.attributes["node_name"] != nil`
	require.Contains(t, yaml, fromService)
	require.Contains(t, yaml, fromNode)
	require.Less(t, strings.Index(yaml, fromService), strings.Index(yaml, fromNode), "service_name wins over node_name")
}

func TestBuildOtelCollectorConfigYAML_PostgresHeaders(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90, otel.ServerLogs{})
	assert.Contains(t, yaml, `replace_pattern(log.body, "(?m)^\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2}:\\d{2}(?:\\.\\d+)? \\S+ \\[\\d+\\] (?:\\S+@\\S+ )?(DETAIL|HINT|CONTEXT|STATEMENT|QUERY|LOCATION):", "$$1:") where log.attributes["log.preset"] == "postgres"`)

	// OTTL's replace_pattern uses Go's regexp, so this is what the collector does to a joined entry.
	body := "execute <unnamed>: SELECT pg_database_size($1)\n" +
		"2026-10-17 21:59:31.311 GMT [349071] DETAIL:  Parameters: $1 = 'shop'\n" +
		"2026-10-17 21:59:31.311 GMT [349071] pmm@postgres HINT:  See the server log.\n" +
		"\t\tFROM pg_database 2026-10-17 12:00:00 GMT [1] LOG:  not a header"
	assert.Equal(t, "execute <unnamed>: SELECT pg_database_size($1)\n"+
		"DETAIL:  Parameters: $1 = 'shop'\n"+
		"HINT:  See the server log.\n"+
		"\t\tFROM pg_database 2026-10-17 12:00:00 GMT [1] LOG:  not a header",
		regexp.MustCompile(postgresHeaderPattern).ReplaceAllString(body, "$1:"))
}

type renderedOtelConfig struct {
	Extensions map[string]map[string]any `yaml:"extensions"`
	Receivers  map[string]any            `yaml:"receivers"`
	Service    struct {
		Extensions []string `yaml:"extensions"`
		Pipelines  map[string]struct {
			Receivers []string `yaml:"receivers"`
		} `yaml:"pipelines"`
	} `yaml:"service"`
}

func TestBuildOtelCollectorConfigYAML_ServerLogs(t *testing.T) {
	t.Parallel()

	t.Run("with receivers", func(t *testing.T) {
		t.Parallel()
		serverLogs := otel.ServerLogs{
			ReceiversYAML: "  filelog/srv_x:\n    include: ['/srv/logs/x.log']\n",
			ReceiverIDs:   []string{"filelog/srv_x"},
		}
		var cfg renderedOtelConfig
		require.NoError(t, yaml.Unmarshal([]byte(buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90, serverLogs)), &cfg))

		assert.Equal(t, []string{"otlp", "filelog/srv_x"}, cfg.Service.Pipelines["logs"].Receivers)
		assert.Equal(t, []string{"otlp"}, cfg.Service.Pipelines["traces"].Receivers)
		assert.Contains(t, cfg.Receivers, "filelog/srv_x")
		assert.Contains(t, cfg.Receivers, "otlp")
		assert.Equal(t, []string{otel.ServerLogStorage}, cfg.Service.Extensions)
		assert.Equal(t, "/srv/otelcol/storage", cfg.Extensions[otel.ServerLogStorage]["directory"])
	})

	t.Run("without receivers", func(t *testing.T) {
		t.Parallel()
		var cfg renderedOtelConfig
		require.NoError(t, yaml.Unmarshal([]byte(buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90, otel.ServerLogs{})), &cfg))

		assert.Equal(t, []string{"otlp"}, cfg.Service.Pipelines["logs"].Receivers)
		assert.Equal(t, []string{otel.ServerLogStorage}, cfg.Service.Extensions)
	})
}

func TestWriteOtelCollectorConfig_Unchanged(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("a: 1\n"), 0o600))

	written, err := writeOtelCollectorConfig(path, []byte("a: 1\n"), func(string) error {
		t.Fatal("validator called for an unchanged config")
		return nil
	})
	require.NoError(t, err)
	assert.False(t, written)
}

func TestWriteOtelCollectorConfig_Changed(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("a: 1\n"), 0o600))

	var validated string
	written, err := writeOtelCollectorConfig(path, []byte("a: 2\n"), func(p string) error {
		b, err := os.ReadFile(p)
		require.NoError(t, err)
		validated = string(b)
		return nil
	})
	require.NoError(t, err)
	assert.True(t, written)
	assert.Equal(t, "a: 2\n", validated, "the validator sees the new config")
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a: 2\n", string(b))
	assert.NoFileExists(t, path+".new")
}

func TestWriteOtelCollectorConfig_Missing(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")

	written, err := writeOtelCollectorConfig(path, []byte("a: 1\n"), func(string) error { return nil })
	require.NoError(t, err)
	assert.True(t, written)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a: 1\n", string(b))
}

func TestWriteOtelCollectorConfig_InvalidKeepsOld(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("a: 1\n"), 0o600))

	invalid := errors.New("invalid config")
	written, err := writeOtelCollectorConfig(path, []byte("a: 2\n"), func(string) error { return invalid })
	require.ErrorIs(t, err, invalid)
	assert.False(t, written)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "a: 1\n", string(b))
	assert.NoFileExists(t, path+".new")
}

func TestQuoteYAMLString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "'simple'", quoteYAMLString("simple"))
	require.Equal(t, "''''", quoteYAMLString("'"))
	require.Equal(t, "'a''b'", quoteYAMLString("a'b"))
}

func TestBuildOtelCollectorConfigYAML_ValidYAMLStructure(t *testing.T) {
	t.Parallel()

	yaml := buildOtelCollectorConfigYAML("127.0.0.1:9000", "default", "clickhouse", 7, 7, 90, otel.ServerLogs{})

	require.Contains(t, yaml, "receivers:")
	require.Contains(t, yaml, "processors:")
	require.Contains(t, yaml, "exporters:")
	require.Contains(t, yaml, "service:")
}

// fakeSupervisorctl returns a supervisorctl stand-in that records its arguments and exits with the code in the
// returned file ("0" by default).
func fakeSupervisorctl(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	code := filepath.Join(dir, "code")
	require.NoError(t, os.WriteFile(code, []byte("0"), 0o600))
	script := filepath.Join(dir, "supervisorctl")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho \"$@\" >> "+calls+"\nexit $(cat "+code+")\n"), 0o700))
	return script, calls, code
}

func TestRestartOtelCollectorIfPending(t *testing.T) {
	t.Parallel()

	script, calls, code := fakeSupervisorctl(t)
	s := &Service{supervisorctlPath: script, l: logrus.WithField("test", t.Name())}

	s.restartOtelCollectorIfPending(true)
	assert.NoFileExists(t, calls, "nothing pending, no restart")

	s.otelRestartPending = true
	s.restartOtelCollectorIfPending(false)
	assert.NoFileExists(t, calls, "a disabled collector is not started")
	assert.True(t, s.otelRestartPending)

	require.NoError(t, os.WriteFile(code, []byte("1"), 0o600))
	s.restartOtelCollectorIfPending(true)
	assert.True(t, s.otelRestartPending, "a failed restart is retried on the next update")

	require.NoError(t, os.WriteFile(code, []byte("0"), 0o600))
	s.restartOtelCollectorIfPending(true)
	assert.False(t, s.otelRestartPending)

	b, err := os.ReadFile(calls)
	require.NoError(t, err)
	assert.Equal(t, "restart otel-collector\nrestart otel-collector\n", string(b))
}
