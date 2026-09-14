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

package mcp

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type configInput struct {
	ServiceName string `json:"service_name" jsonschema:"Service name from pmm_inventory (metrics are labelled by it)"`
	Engine      string `json:"engine,omitempty" jsonschema:"mysql (default) or postgresql"`
	Filter      string `json:"filter,omitempty" jsonschema:"Only variables whose name contains this substring (e.g. innodb)"`
	At          string `json:"at,omitempty" jsonschema:"Point in time: RFC3339 or relative such as now-1h (default now)"`
}

// configMetrics maps an engine to the metric prefix the exporter uses for
// numeric server variables.
var configMetrics = map[string]string{
	engineMySQL:      "mysql_global_variables_",
	enginePostgreSQL: "pg_settings_",
}

func (s *Service) registerConfigTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_get_config",
		Title: "Database configuration",
		Description: "Retrieve a monitored database's server configuration (tuning variables) from PMM's collected " +
			"metrics, as SHOW GLOBAL VARIABLES-style name<TAB>value lines plus the server version. Limitation: " +
			"exporters publish numeric variables only; string-valued knobs such as sql_mode are not in metrics.",
		Annotations: readOnly("Database configuration"),
	}, s.config)
}

func (s *Service) config(ctx context.Context, req *mcp.CallToolRequest, in configInput) (*mcp.CallToolResult, any, error) {
	if in.ServiceName == "" {
		return nil, nil, newToolError(codeInvalidInput, "service_name is required")
	}
	engine := in.Engine
	if engine == "" {
		engine = engineMySQL
	}
	prefix, ok := configMetrics[engine]
	if !ok {
		return nil, nil, newToolError(codeInvalidInput, "engine must be mysql or postgresql; got '%s'", in.Engine)
	}
	at, err := parseTime(in.At, s.now())
	if err != nil {
		return nil, nil, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	samples, err := s.queryMetrics(ctx, auth,
		`{__name__=~"`+prefix+`.+", service_name="`+escapeLabel(in.ServiceName)+`"}`, at)
	if err != nil {
		return nil, nil, s.fail("pmm_get_config", err)
	}

	variables := map[string]string{}
	for _, sample := range samples {
		name := sample.Labels["__name__"]
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		variables[strings.TrimPrefix(name, prefix)] = formatMetricValue(sample.Value)
	}

	names := make([]string, 0, len(variables))
	needle := strings.ToLower(in.Filter)
	for name := range variables {
		if needle != "" && !strings.Contains(strings.ToLower(name), needle) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return textResult("No configuration variables found for that service (check service_name and that metrics are being collected)."), nil, nil
	}

	version := s.serviceVersion(ctx, auth, engine, in.ServiceName)
	header := fmt.Sprintf("-- %d variables (numeric knobs from PMM metrics)", len(names))
	if version != "" {
		header = fmt.Sprintf("-- %s %s - %d variables (numeric knobs from PMM metrics)", engine, version, len(names))
	}
	lines := make([]string, 0, len(names)+1)
	lines = append(lines, header)
	for _, name := range names {
		lines = append(lines, name+"\t"+variables[name])
	}
	text := strings.Join(lines, "\n")
	text += "\n\n" + link("View this service in PMM", qanOverviewURL(s.publicBaseURL(ctx, req.Extra.Header), in.ServiceName, at.Add(-hourWindow), at))
	return textResult(text), nil, nil
}

// formatMetricValue renders integers without an exponent and keeps other
// numbers as-is; Prometheus returns values as strings, sometimes exponential.
func formatMetricValue(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
