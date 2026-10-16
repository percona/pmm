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

package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
)

const (
	nginxStatusFilter   = `attributes.status != nil and int(attributes.status) < 400`
	vmproxyAccessFilter = `attributes.level == "info" and attributes.msg != nil and attributes.msg matches "^[A-Z]+: /"`
)

func parseServerReceivers(t *testing.T, logs ServerLogs) map[string]map[string]any {
	t.Helper()
	var parsed struct {
		Receivers map[string]map[string]any `yaml:"receivers"`
	}
	require.NoError(t, yaml.Unmarshal([]byte("receivers:\n"+logs.ReceiversYAML), &parsed), logs.ReceiversYAML)
	return parsed.Receivers
}

func TestBuildServerLogs(t *testing.T) {
	t.Parallel()

	presets := []*models.LogParserPreset{
		{Name: "pmm_managed", OperatorYAML: "- type: key_value_parser\n  parse_from: body\n"},
		{Name: "postgres", LineStartPattern: `^\d{4}-\d{2}-\d{2} \S+ \[\d+\] (LOG|ERROR):`, OperatorYAML: "- type: regex_parser\n  regex: '^(?P<message>.*)$'\n"},
		{Name: "nginx_access", OperatorYAML: "- type: key_value_parser\n  parse_from: body\n"},
		{Name: "pmm_agent", OperatorYAML: "- type: key_value_parser\n  parse_from: body\n"},
	}
	logs := BuildServerLogs(presets, ServerNode{NodeID: "pmm-server", NodeName: "pmm-server"})

	require.Len(t, logs.ReceiverIDs, 14)
	assert.Equal(t, "filelog/srv_other", logs.ReceiverIDs[13])

	receivers := parseServerReceivers(t, logs)
	require.Len(t, receivers, 14)
	for _, id := range logs.ReceiverIDs {
		require.Contains(t, receivers, id)
	}

	managed := receivers["filelog/srv_pmm_managed"]
	assert.Equal(t, []any{"/srv/logs/pmm-managed.log"}, managed["include"])
	assert.Equal(t, "end", managed["start_at"])
	assert.Equal(t, ServerLogStorage, managed["storage"])
	assert.Equal(t, map[string]any{"node_id": "pmm-server", "node_name": "pmm-server", "service.name": "pmm-managed"}, managed["resource"])
	assert.Equal(t, []any{map[string]any{"type": "key_value_parser", "parse_from": "body", "on_error": "send_quiet"}}, managed["operators"])

	pg := receivers["filelog/srv_postgresql14"]
	assert.Equal(t, map[string]any{"line_start_pattern": presets[1].LineStartPattern}, pg["multiline"])

	nginx := receivers["filelog/srv_nginx"]
	ops, ok := nginx["operators"].([]any)
	require.True(t, ok)
	require.Len(t, ops, 2)
	assert.Equal(t, map[string]any{"type": "filter", "expr": nginxStatusFilter, "on_error": "send_quiet"}, ops[1])

	// vmproxy logs every request nginx already logged; keep only its other lines.
	ops, ok = receivers["filelog/srv_vmproxy"]["operators"].([]any)
	require.True(t, ok)
	require.Len(t, ops, 2)
	assert.Equal(t, map[string]any{"type": "filter", "expr": vmproxyAccessFilter, "on_error": "send_quiet"}, ops[1])
	ops, ok = receivers["filelog/srv_qan_api2"]["operators"].([]any)
	require.True(t, ok)
	assert.Len(t, ops, 1, "same preset, no filter")

	assert.Equal(t, "clickhouse-server", receivers["filelog/srv_clickhouse_server"]["resource"].(map[string]any)["service.name"])
	assert.NotContains(t, receivers["filelog/srv_pmm_init"], "operators")

	other := receivers["filelog/srv_other"]
	assert.Equal(t, []any{"/srv/logs/*.log"}, other["include"])
	assert.Equal(t, "pmm-server", other["resource"].(map[string]any)["service.name"])
	assert.NotContains(t, other, "operators")
	exclude, ok := other["exclude"].([]any)
	require.True(t, ok)
	assert.ElementsMatch(t, []any{
		"/srv/logs/pmm-managed.log", "/srv/logs/pmm-agent.log", "/srv/logs/qan-api2.log", "/srv/logs/vmproxy.log",
		"/srv/logs/nginx.log", "/srv/logs/grafana.log", "/srv/logs/clickhouse-server.log", "/srv/logs/postgresql14.log",
		"/srv/logs/supervisord.log", "/srv/logs/otel-collector.log", "/srv/logs/victoriametrics.log", "/srv/logs/vmalert.log",
		"/srv/logs/pmm-init.log",
	}, exclude)
}

func TestBuildServerLogs_NoPresets(t *testing.T) {
	t.Parallel()

	logs := BuildServerLogs(nil, ServerNode{NodeID: "pmm-server", NodeName: "pmm-server"})

	receivers := parseServerReceivers(t, logs)
	require.Len(t, receivers, 14)
	for _, id := range logs.ReceiverIDs {
		require.Contains(t, receivers, id)
		assert.NotContains(t, receivers[id], "operators", id)
	}
}

func TestBuildServerLogs_HANode(t *testing.T) {
	t.Parallel()

	logs := BuildServerLogs(nil, ServerNode{NodeID: "pmm-server", NodeName: "pmm-server", HANodeID: "pmm-ha-1"})

	for id, r := range parseServerReceivers(t, logs) {
		res, ok := r["resource"].(map[string]any)
		require.True(t, ok, id)
		assert.Equal(t, "pmm-ha-1", res["ha_node_id"], id)
	}
}

// The server collector reads its own log, so an operator that logs its errors would feed them back to itself.
func TestBuildServerLogs_QuietOnError(t *testing.T) {
	t.Parallel()

	presets := []*models.LogParserPreset{{Name: "otel_collector", OperatorYAML: `- type: regex_parser
  regex: '^(?P<message>.*)$'
- type: time_parser
  parse_from: attributes.timestamp
  on_error: drop
- type: move
  from: attributes.message
  to: body
  on_error: send
- type: remove
  field: attributes.level
  on_error: send_quiet
- type: router
  routes: []
`}}
	logs := BuildServerLogs(presets, ServerNode{NodeID: "pmm-server", NodeName: "pmm-server"})

	ops, ok := parseServerReceivers(t, logs)["filelog/srv_otel_collector"]["operators"].([]any)
	require.True(t, ok)
	onError := make([]any, 0, len(ops))
	for _, op := range ops {
		onError = append(onError, op.(map[string]any)["on_error"])
	}
	assert.Equal(t, []any{"send_quiet", "drop_quiet", "send_quiet", "send_quiet", nil}, onError)
	assert.Equal(t, `^(?P<message>.*)$`, ops[0].(map[string]any)["regex"])
}

func TestBuildServerLogs_InvalidPresetYAML(t *testing.T) {
	t.Parallel()

	presets := []*models.LogParserPreset{{Name: "grafana", OperatorYAML: "- type: [unclosed"}}
	logs := BuildServerLogs(presets, ServerNode{NodeID: "pmm-server", NodeName: "pmm-server"})

	assert.NotContains(t, parseServerReceivers(t, logs)["filelog/srv_grafana"], "operators")
}
