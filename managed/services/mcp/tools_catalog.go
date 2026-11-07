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
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	catalogKindQANColumns = "qan_columns"
	catalogKindMetrics    = "metrics"
)

// catalogKinds are the catalogues pmm_catalog serves.
var catalogKinds = []string{catalogKindQANColumns, catalogKindMetrics}

type catalogInput struct {
	Kind   string `json:"kind" jsonschema:"Catalogue: qan_columns, the names pmm_top_queries takes in columns and order_by; or metrics, the PMM metrics"`
	Engine string `json:"engine,omitempty" jsonschema:"Keep one engine's entries: mysql, postgresql, mongodb, plus all-engine ones (qan_columns); node too for metrics"`
}

// catalogEntry is one entry of a catalogue.
type catalogEntry struct {
	Name        string   `json:"name" jsonschema:"Entry name."`
	Engine      string   `json:"engine,omitempty" jsonschema:"Engine the entry belongs to; left out for an entry every engine has."`
	Aggregation string   `json:"aggregation,omitempty" jsonschema:"How QAN reports the metric: sum, or avg for a per-call average (qan_columns)."`
	Meaning     string   `json:"meaning,omitempty" jsonschema:"What the metric measures (metrics)."`
	Unit        string   `json:"unit,omitempty" jsonschema:"Unit of the value query returns, as Grafana names it: short, ops, bytes, Bps, s or none (metrics)."`
	Type        string   `json:"type,omitempty" jsonschema:"counter, which only grows (query it with rate), or gauge (metrics)."`
	Labels      []string `json:"labels,omitempty" jsonschema:"Label names on the metric's samples as the exporter exposes them; PMM adds its own (metrics)."`
	Query       string   `json:"query,omitempty" jsonschema:"A ready MetricsQL expression for pmm_query_metrics; replace <service> (or <node>) with a name (metrics)."`
}

// catalogResult is the result of pmm_catalog.
type catalogResult struct {
	Kind    string         `json:"kind" jsonschema:"Catalogue that was read."`
	Engine  string         `json:"engine,omitempty" jsonschema:"Engine filter the request gave."`
	Entries []catalogEntry `json:"entries" jsonschema:"Catalogue entries, in catalogue order."`
}

func (r catalogResult) render() string {
	lines := make([]string, 0, len(r.Entries)+1)
	if r.Kind == catalogKindMetrics {
		lines = append(lines, fmt.Sprintf("%s (%d), as name [engine] type unit: meaning; labels: names; query:", r.Kind, len(r.Entries)))
	} else {
		lines = append(lines, fmt.Sprintf("%s (%d), as name [engine] aggregation:", r.Kind, len(r.Entries)))
	}
	for _, e := range r.Entries {
		line := "- " + e.Name
		if e.Engine != "" {
			line += " [" + e.Engine + "]"
		}
		if e.Aggregation != "" {
			line += " " + e.Aggregation
		}
		if e.Query != "" {
			line += " " + e.Type + " " + e.Unit + ": " + e.Meaning
			if len(e.Labels) > 0 {
				line += "; labels: " + strings.Join(e.Labels, ", ")
			}
			line += "; query: " + e.Query
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (s *Service) registerCatalogTools(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_catalog",
		Title: "Catalogues",
		Description: "List the names a PMM tool accepts. kind qan_columns lists the QAN metrics that pmm_top_queries takes in " +
			"columns and order_by, with the engine each belongs to (none for a metric every engine has) and whether QAN " +
			"reports it as a sum or a per-call average; engine keeps one engine's entries plus the all-engine ones. " +
			"kind metrics lists the PMM metrics for mysql, postgresql, mongodb and node, each with its meaning, unit, " +
			"type (counter or gauge), labels and a ready MetricsQL query to run with pmm_query_metrics after replacing " +
			"<service> (or <node>); engine keeps one engine's metrics.",
		Annotations: readOnly("Catalogues"),
	}, s.catalog)
}

func (s *Service) catalog(_ context.Context, _ *mcp.CallToolRequest, in catalogInput) (catalogResult, error) {
	if !slices.Contains(catalogKinds, in.Kind) {
		return catalogResult{}, newToolError(codeInvalidInput, "kind must be one of %s; got '%s'", strings.Join(catalogKinds, ", "), in.Kind)
	}
	engines := []string{engineMySQL, enginePostgreSQL, engineMongoDB}
	if in.Kind == catalogKindMetrics {
		engines = metricsEngines
	}
	if in.Engine != "" && !slices.Contains(engines, in.Engine) {
		return catalogResult{}, newToolError(codeInvalidInput, "engine must be one of %s; got '%s'", strings.Join(engines, ", "), in.Engine)
	}

	if in.Kind == catalogKindMetrics {
		return metricsCatalogResult(in.Engine), nil
	}

	entries := make([]catalogEntry, 0, len(qanColumns))
	for _, c := range qanColumns {
		if in.Engine != "" && c.engine != "" && c.engine != in.Engine {
			continue
		}
		aggregation := "sum"
		if c.avg {
			aggregation = "avg"
		}
		entries = append(entries, catalogEntry{Name: c.name, Engine: c.engine, Aggregation: aggregation})
	}
	return catalogResult{Kind: in.Kind, Engine: in.Engine, Entries: entries}, nil
}

func metricsCatalogResult(engine string) catalogResult {
	entries := make([]catalogEntry, 0, len(metricsCatalog))
	for _, m := range metricsCatalog {
		if engine != "" && m.Engine != engine {
			continue
		}
		entries = append(entries, catalogEntry{
			Name: m.Name, Engine: m.Engine, Meaning: m.Meaning, Unit: m.Unit, Type: m.Type, Labels: m.Labels, Query: m.Query,
		})
	}
	return catalogResult{Kind: catalogKindMetrics, Engine: engine, Entries: entries}
}
