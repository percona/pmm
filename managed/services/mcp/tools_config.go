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
	"cmp"
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type configInput struct {
	ServiceID   string `json:"service_id,omitempty" jsonschema:"Service id from pmm_inventory (preferred; same identifier every other tool takes)"`
	ServiceName string `json:"service_name,omitempty" jsonschema:"Service name from pmm_inventory. Only needed when service_id is not supplied"`
	Engine      string `json:"engine,omitempty" jsonschema:"mysql or postgresql. Defaults to the service's own engine"`
	Filter      string `json:"filter,omitempty" jsonschema:"Only variables whose name contains this substring (e.g. innodb)"`
	At          string `json:"at,omitempty" jsonschema:"Point in time: RFC3339 or relative such as now-1h (default now)"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum variables to return; default 200, maximum 1000"`
	Offset      int    `json:"offset,omitempty" jsonschema:"Number of variables to skip, for the next page"`
}

const (
	configDefaultLimit = 200
	configMaxLimit     = 1000
)

// configResult is the result of pmm_get_config.
type configResult struct {
	ServiceName string     `json:"service_name" jsonschema:"Untrusted. Service name."`
	Engine      string     `json:"engine" jsonschema:"Engine: mysql or postgresql."`
	Version     string     `json:"version,omitempty" jsonschema:"Untrusted. Server version, left out when it cannot be read."`
	At          time.Time  `json:"at" jsonschema:"Point in time the variables were read at."`
	Variables   []variable `json:"variables" jsonschema:"Variables on this page, sorted by name."`
	Page        page       `json:"page" jsonschema:"Paging of the variables that match filter."`
	PMMURL      string     `json:"pmm_url" jsonschema:"Link to the service in PMM Query Analytics, over the hour before at."`
}

// variable is one server variable of a pmm_get_config result.
type variable struct {
	Name  string `json:"name" jsonschema:"Variable name, without the exporter's metric prefix."`
	Value string `json:"value" jsonschema:"Untrusted. Value as PMM's metrics hold it."`
}

func (r configResult) render() string {
	view := link("View this service in PMM QAN", r.PMMURL)
	at := "At: " + r.At.UTC().Format(time.RFC3339)
	if r.Page.Total == 0 {
		return "No configuration variables found for that service (check the service and that its metrics are being collected).\n" + at + "\n\n" + view
	}
	if len(r.Variables) == 0 {
		return r.Page.pastEnd("variable list") + "\n" + at + "\n\n" + view
	}

	header := fmt.Sprintf("-- %d variables (numeric knobs from PMM metrics)", r.Page.Total)
	if r.Version != "" {
		header = fmt.Sprintf("-- %s %s - %d variables (numeric knobs from PMM metrics)", r.Engine, r.Version, r.Page.Total)
	}
	lines := []string{header, "-- " + at}
	for _, v := range r.Variables {
		lines = append(lines, v.Name+"\t"+v.Value)
	}
	text := strings.Join(lines, "\n") + "\n\n" + view
	if m := r.Page.marker(); m != "" {
		text += "\n" + m
	}
	return text
}

// configMetrics maps an engine to the metric prefix the exporter uses for
// numeric server variables.
var configMetrics = map[string]string{
	engineMySQL:      "mysql_global_variables_",
	enginePostgreSQL: "pg_settings_",
}

func (s *Service) registerConfigTools(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_get_config",
		Title: "Database configuration",
		Description: "Retrieve a monitored database's server configuration (tuning variables) from PMM's collected " +
			"metrics, as SHOW GLOBAL VARIABLES-style name<TAB>value lines plus the server version. Limitation: " +
			"exporters publish numeric variables only; string-valued knobs such as sql_mode are not in metrics.",
		Annotations: readOnly("Database configuration"),
	}, s.config)
}

func (s *Service) config(ctx context.Context, req *mcp.CallToolRequest, in configInput) (configResult, error) {
	if in.ServiceID == "" && in.ServiceName == "" {
		return configResult{}, newToolError(codeInvalidInput, "service_id is required (service_name is also accepted)")
	}
	// Validate an explicit engine before any API round trip.
	if in.Engine != "" {
		_, ok := configMetrics[in.Engine]
		if !ok {
			return configResult{}, newToolError(codeInvalidInput, "engine must be mysql or postgresql; got '%s'", in.Engine)
		}
	}
	err := checkPage(in.Offset, in.Limit, configMaxLimit)
	if err != nil {
		return configResult{}, err
	}
	at, err := parseTime(in.At, s.now())
	if err != nil {
		return configResult{}, err
	}
	// UTC in whole seconds, so that the text and the structured at read the same.
	at = at.UTC().Truncate(time.Second)
	auth := callerAuthFromHeader(req.Extra.Header)

	w := window{From: at.Add(-hourWindow), To: at}
	services, err := s.listServices(ctx, auth, w)
	if err != nil {
		return configResult{}, err
	}
	var svc serviceInfo
	var found bool
	if in.ServiceID != "" {
		i := slices.IndexFunc(services, func(o serviceInfo) bool { return o.ServiceID == in.ServiceID })
		found = i >= 0
		if found {
			svc = services[i]
		}
	} else {
		svc, found = serviceNamed(services, in.ServiceName)
	}
	if !found {
		return configResult{}, errNotInList(w, "; use pmm_inventory to list service ids")
	}
	serviceName := svc.ServiceName
	// An explicit engine wins over the service list's.
	engine := cmp.Or(in.Engine, svc.Engine)
	if engine == "" {
		engine = engineMySQL
	}

	prefix, ok := configMetrics[engine]
	if !ok {
		// An explicit engine was validated above, so this engine came from the service list.
		return configResult{}, newToolError(codeInvalidInput,
			"pmm_get_config covers MySQL and PostgreSQL services; service '%s' is %s", serviceName, engine)
	}

	samples, err := s.queryMetrics(ctx, auth,
		`{__name__=~"`+prefix+`.+", service_name="`+escapeLabel(serviceName)+`"}`, at)
	if err != nil {
		return configResult{}, err
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
	paged, pg, err := paginate(names, in.Offset, in.Limit, configDefaultLimit, configMaxLimit)
	if err != nil {
		return configResult{}, err
	}

	r := configResult{
		ServiceName: serviceName,
		Engine:      engine,
		At:          at,
		Variables:   make([]variable, 0, len(paged)),
		Page:        pg,
		PMMURL:      qanOverviewURL(s.publicBaseURL(ctx, req.Extra.Header), serviceName, at.Add(-hourWindow), at),
	}
	for _, name := range paged {
		r.Variables = append(r.Variables, variable{Name: name, Value: variables[name]})
	}
	if len(names) > 0 {
		r.Version = s.serviceVersion(ctx, auth, engine, serviceName)
	}
	return r, nil
}

// formatMetricValue renders integers without an exponent and keeps other
// numbers as-is; Prometheus returns values as strings, sometimes exponential.
func formatMetricValue(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
