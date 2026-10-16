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
	"strings"

	"github.com/percona/pmm/managed/models"
)

const (
	// ServerLogDir is where PMM Server's components write their logs.
	ServerLogDir = "/srv/logs"
	// ServerLogStorage is the file_storage extension that keeps the server log read positions.
	ServerLogStorage = "file_storage/srv_logs"
)

// ServerNode identifies the PMM Server node whose logs are collected.
type ServerNode struct {
	NodeID   string
	NodeName string
	// HANodeID is set in HA mode so each server's lines stay apart.
	HANodeID string
}

// ServerLogs holds the filelog receivers for PMM Server's own logs, ready to embed in the collector config.
type ServerLogs struct {
	// ReceiversYAML is the receiver entries for the "receivers:" block, indented two spaces.
	ReceiversYAML string
	ReceiverIDs   []string
}

type serverLogFile struct {
	name   string
	preset string
	// dropExpr, when set, adds a filter operator that drops the lines it matches.
	dropExpr string
}

// serverLogFiles maps each known file in ServerLogDir to its parser preset; "raw" means no parsing.
var serverLogFiles = []serverLogFile{
	{name: "pmm-managed.log", preset: "pmm_managed"},
	{name: "pmm-agent.log", preset: "pmm_agent"},
	{name: "qan-api2.log", preset: "pmm_agent"},
	{name: "vmproxy.log", preset: "pmm_agent"},
	// nginx logs every API call; keep 4xx, 5xx and error-log lines (no status).
	{name: "nginx.log", preset: "nginx_access", dropExpr: `attributes.status != nil and int(attributes.status) < 400`},
	{name: "grafana.log", preset: "grafana"},
	{name: "clickhouse-server.log", preset: "clickhouse_server"},
	{name: "postgresql14.log", preset: "postgres"},
	{name: "supervisord.log", preset: "supervisord"},
	{name: "otel-collector.log", preset: "otel_collector"},
	{name: "victoriametrics.log", preset: "victoriametrics"},
	{name: "vmalert.log", preset: "victoriametrics"},
	{name: "pmm-init.log", preset: "raw"},
}

// BuildServerLogs returns a filelog receiver per known PMM Server log file and one for every other *.log file.
func BuildServerLogs(presets []*models.LogParserPreset, node ServerNode) ServerLogs {
	byName := make(map[string]*models.LogParserPreset, len(presets))
	for _, p := range presets {
		byName[p.Name] = p
	}

	var sb strings.Builder
	ids := make([]string, 0, len(serverLogFiles)+1)
	known := make([]string, 0, len(serverLogFiles))
	for _, f := range serverLogFiles {
		path := ServerLogDir + "/" + f.name
		stem := strings.TrimSuffix(f.name, ".log")
		id := "filelog/srv_" + sanitizePresetName(stem)
		ids = append(ids, id)
		known = append(known, path)

		var lineStart, operators string
		if p := byName[f.preset]; p != nil {
			lineStart = p.LineStartPattern
			operators = p.OperatorYAML
		}
		if operators != "" && f.dropExpr != "" {
			operators = strings.TrimSuffix(operators, "\n") + "\n- type: filter\n  expr: " + quoteYAMLString(f.dropExpr) + "\n"
		}
		writeServerReceiver(&sb, id, []string{path}, nil, stem, node, lineStart, operators)
	}

	ids = append(ids, "filelog/srv_other")
	writeServerReceiver(&sb, "filelog/srv_other", []string{ServerLogDir + "/*.log"}, known, "pmm-server", node, "", "")

	return ServerLogs{ReceiversYAML: sb.String(), ReceiverIDs: ids}
}

func writeServerReceiver(sb *strings.Builder, id string, include, exclude []string, serviceName string, node ServerNode, lineStart, operators string) {
	sb.WriteString("  " + id + ":\n")
	sb.WriteString("    include: " + yamlFlowList(include) + "\n")
	if len(exclude) != 0 {
		sb.WriteString("    exclude: " + yamlFlowList(exclude) + "\n")
	}
	sb.WriteString("    start_at: end\n")
	sb.WriteString("    storage: " + ServerLogStorage + "\n")
	sb.WriteString("    resource:\n")
	sb.WriteString("      node_id: " + quoteYAMLString(node.NodeID) + "\n")
	sb.WriteString("      node_name: " + quoteYAMLString(node.NodeName) + "\n")
	sb.WriteString("      service.name: " + quoteYAMLString(serviceName) + "\n")
	if node.HANodeID != "" {
		sb.WriteString("      ha_node_id: " + quoteYAMLString(node.HANodeID) + "\n")
	}
	sb.WriteString(FilelogMultilineYAML(lineStart, "    "))
	if operators != "" {
		sb.WriteString("    operators:\n" + IndentYAML(operators, "      "))
	}
}

func yamlFlowList(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = quoteYAMLString(s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
