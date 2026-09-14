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
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

const (
	defaultLimit = 10
	maxLimit     = 100
	groupByQuery = "queryid"
)

// orderMetrics maps the contract's order_by values to QAN metric names.
var orderMetrics = map[string]string{
	"load":             "load",
	"total_query_time": "query_time",
	"avg_query_time":   "query_time",
	"count":            "num_queries",
}

type topQueriesInput struct {
	ServiceName string `json:"service_name,omitempty" jsonschema:"Filter by service name (from pmm_inventory)"`
	ServiceID   string `json:"service_id,omitempty" jsonschema:"Filter by service id (from pmm_inventory)"`
	PeriodFrom  string `json:"period_from,omitempty" jsonschema:"Window start: RFC3339 or relative such as now-1h (default now-1h)"`
	PeriodTo    string `json:"period_to,omitempty" jsonschema:"Window end: RFC3339 or relative (default now)"`
	OrderBy     string `json:"order_by,omitempty" jsonschema:"Ranking metric: load (default), total_query_time, avg_query_time or count"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Number of queries, 1-100 (default 10)"`
}

type queryDetailInput struct {
	QueryID    string `json:"queryid" jsonschema:"Query id from pmm_top_queries"`
	ServiceID  string `json:"service_id,omitempty" jsonschema:"Restrict to one service id"`
	PeriodFrom string `json:"period_from,omitempty" jsonschema:"Window start: RFC3339 or relative such as now-1h (default now-1h)"`
	PeriodTo   string `json:"period_to,omitempty" jsonschema:"Window end: RFC3339 or relative (default now)"`
}

func (s *Service) registerQANTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_top_queries",
		Title: "Top queries by load",
		Description: "Rank the worst queries for a service over a time window, from PMM Query Analytics (QAN). " +
			"Returns query fingerprints with load and timing metrics. Pass a returned queryid to " +
			"pmm_query_detail or pmm_get_explain.",
		Annotations: readOnly("Top queries by load"),
	}, handle(s, "pmm_top_queries", s.topQueries))

	mcp.AddTool(server, &mcp.Tool{
		Name:  "pmm_query_detail",
		Title: "Query detail",
		Description: "Fetch the fingerprint, schema, engine and key diagnostic metrics (rows examined/sent, " +
			"no_index_used, full_scan, filesort) plus an example statement for a single query, by queryid.",
		Annotations: readOnly("Query detail"),
	}, handle(s, "pmm_query_detail", s.queryDetail))
}

func (s *Service) topQueries(ctx context.Context, req *mcp.CallToolRequest, in topQueriesInput) (*mcp.CallToolResult, error) {
	orderBy := in.OrderBy
	if orderBy == "" {
		orderBy = "load"
	}
	metric, ok := orderMetrics[orderBy]
	if !ok {
		return nil, newToolError(codeInvalidInput, "order_by must be one of load, total_query_time, avg_query_time, count; got '%s'", orderBy)
	}
	limit := in.Limit
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, newToolError(codeInvalidInput, "limit must be between 1 and %d; got %d", maxLimit, limit)
	}
	from, to, err := parseWindow(in.PeriodFrom, in.PeriodTo, s.now())
	if err != nil {
		return nil, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	body := qan_service.GetReportBody{
		PeriodStartFrom: strfmt.DateTime(from),
		PeriodStartTo:   strfmt.DateTime(to),
		GroupBy:         groupByQuery,
		OrderBy:         "-" + metric,
		Offset:          0,
		Limit:           int64(limit),
		Columns:         []string{"load", "num_queries", "query_time"},
		MainMetric:      metric,
		Labels:          []*qan_service.GetReportParamsBodyLabelsItems0{},
	}
	switch {
	case in.ServiceName != "":
		body.Labels = append(body.Labels, &qan_service.GetReportParamsBodyLabelsItems0{Key: "service_name", Value: []string{in.ServiceName}})
	case in.ServiceID != "":
		body.Labels = append(body.Labels, &qan_service.GetReportParamsBodyLabelsItems0{Key: "service_id", Value: []string{in.ServiceID}})
	}

	report, err := s.api.GetReport(ctx, auth, body)
	if err != nil {
		return nil, err
	}

	// Confirmed against PMM 3.8.1: queryid is Row.dimension; rows[0] is the
	// TOTAL aggregate (empty dimension) and is skipped; per-metric stats are
	// nested under <name>.stats; fingerprint/num_queries/qps/load are also
	// present top-level. Row.database arrives empty for MySQL rows, so schema
	// comes from pmm_query_detail.
	var lines []string
	rank := 0
	for _, row := range report.Rows {
		if row.Dimension == "" {
			continue
		}
		rank++
		numQueries := finite(row.NumQueries)
		load := finite(row.Load)
		var total, avg float64
		if qt, ok := row.Metrics["query_time"]; ok && qt.Stats != nil {
			total = finite(qt.Stats.Sum)
			avg = finite(qt.Stats.Avg)
			if avg == 0 && numQueries > 0 {
				avg = total / numQueries
			}
		}
		if nq, ok := row.Metrics["num_queries"]; ok && nq.Stats != nil && numQueries == 0 {
			numQueries = finite(nq.Stats.Sum)
		}
		if ld, ok := row.Metrics["load"]; ok && ld.Stats != nil && load == 0 {
			load = finite(ld.Stats.SumPerSec)
		}
		fingerprint := row.Fingerprint
		if fingerprint == "" {
			fingerprint = row.Dimension
		}
		lines = append(lines, fmt.Sprintf("%d. [%s] load=%s calls=%s total=%s avg=%s\n   %s",
			rank, row.Dimension, fmtNum3(load), fmtNum(numQueries), fmtSeconds(total), fmtSeconds(avg), fingerprint))
	}
	if len(lines) == 0 {
		return textResult("No queries found for that service / time window."), nil
	}

	text := fmt.Sprintf("Top %d queries (order_by=%s):\n\n%s", len(lines), orderBy, strings.Join(lines, "\n"))
	text += "\n\n" + link("Open this workload in PMM QAN", qanOverviewURL(s.publicBaseURL(ctx, req.Extra.Header), in.ServiceName, from, to))
	return textResult(text), nil
}

func (s *Service) queryDetail(ctx context.Context, req *mcp.CallToolRequest, in queryDetailInput) (*mcp.CallToolResult, error) {
	if in.QueryID == "" {
		return nil, newToolError(codeInvalidInput, "queryid is required")
	}
	from, to, err := parseWindow(in.PeriodFrom, in.PeriodTo, s.now())
	if err != nil {
		return nil, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	d, err := s.fetchQueryDetail(ctx, auth, in, from, to)
	if err != nil {
		return nil, err
	}
	if d.metrics == nil && d.example == nil || d.fingerprint == "" && d.example == nil && len(d.metrics.Metrics) == 0 {
		return nil, newToolError(codeNotFound,
			"no data for queryid '%s' in the selected window; widen period_from or check the id with pmm_top_queries", in.QueryID)
	}
	if d.engine != "" {
		d.version = s.serviceVersion(ctx, auth, d.engine, d.serviceName)
	}

	base := s.publicBaseURL(ctx, req.Extra.Header)
	return textResult(d.render(s.rawSQL(), base, from, to)), nil
}

// queryDetail is the merged view of qan:getMetrics and qan/query:getExample
// for one queryid.
type queryDetail struct {
	queryID     string
	engine      string
	version     string
	serviceName string
	database    string
	schema      string
	fingerprint string
	tables      []string
	metrics     *queryMetrics
	example     *qan_service.GetQueryExampleOKBodyQueryExamplesItems0
}

// fetchQueryDetail runs the two QAN calls, each tolerated independently so
// partial data still returns; it fails only when both fail.
func (s *Service) fetchQueryDetail(ctx context.Context, auth callerAuth, in queryDetailInput, from, to time.Time) (*queryDetail, error) {
	l := s.l.WithField("tool", "pmm_query_detail")

	labels := []*qan_service.GetMetricsParamsBodyLabelsItems0{{Key: groupByQuery, Value: []string{in.QueryID}}}
	exampleLabels := []*qan_service.GetQueryExampleParamsBodyLabelsItems0{{Key: groupByQuery, Value: []string{in.QueryID}}}
	if in.ServiceID != "" {
		labels = append(labels, &qan_service.GetMetricsParamsBodyLabelsItems0{Key: "service_id", Value: []string{in.ServiceID}})
		exampleLabels = append(exampleLabels, &qan_service.GetQueryExampleParamsBodyLabelsItems0{Key: "service_id", Value: []string{in.ServiceID}})
	}

	metrics, metricsErr := s.api.GetMetrics(ctx, auth, qan_service.GetMetricsBody{
		PeriodStartFrom: strfmt.DateTime(from),
		PeriodStartTo:   strfmt.DateTime(to),
		FilterBy:        in.QueryID,
		GroupBy:         groupByQuery,
		Labels:          labels,
		Totals:          false,
	})
	if metricsErr != nil {
		l.Debugf("qan:getMetrics failed: %s.", metricsErr)
	}
	examples, exampleErr := s.api.GetQueryExample(ctx, auth, qan_service.GetQueryExampleBody{
		PeriodStartFrom: strfmt.DateTime(from),
		PeriodStartTo:   strfmt.DateTime(to),
		FilterBy:        in.QueryID,
		GroupBy:         groupByQuery,
		Labels:          exampleLabels,
		Limit:           1,
	})
	if exampleErr != nil {
		l.Debugf("qan/query:getExample failed: %s.", exampleErr)
	}
	if metricsErr != nil && exampleErr != nil {
		return nil, metricsErr
	}

	d := &queryDetail{queryID: in.QueryID, metrics: metrics}
	if metrics != nil {
		d.fingerprint = metrics.Fingerprint
		if metrics.Metadata != nil {
			d.database = metrics.Metadata.Database
			d.schema = metrics.Metadata.Schema
			d.serviceName = metrics.Metadata.ServiceName
			d.engine = metrics.Metadata.ServiceType
		}
	}
	if examples != nil && len(examples.QueryExamples) > 0 {
		d.example = examples.QueryExamples[0]
		if d.schema == "" {
			d.schema = d.example.Schema
		}
		if d.engine == "" {
			d.engine = d.example.ServiceType
		}
		d.tables = d.example.Tables
	}
	return d, nil
}

// render formats the detail for the model.
func (d *queryDetail) render(rawSQL bool, base string, from, to time.Time) string {
	parts := []string{"queryid: " + d.queryID}
	if d.engine != "" {
		engine := d.engine
		if d.version != "" {
			engine += " " + d.version
		}
		parts = append(parts, "engine: "+engine)
	}
	if d.serviceName != "" {
		parts = append(parts, "service: "+d.serviceName)
	}
	// QAN fills database for PostgreSQL and MongoDB, schema for MySQL.
	if d.database != "" {
		parts = append(parts, "database: "+d.database)
	}
	if d.schema != "" {
		parts = append(parts, "schema: "+d.schema)
	}
	if len(d.tables) > 0 {
		parts = append(parts, "tables: "+strings.Join(d.tables, ", "))
	}
	if d.fingerprint != "" {
		parts = append(parts, "fingerprint:\n"+d.fingerprint)
	}
	if d.metrics != nil {
		if km := keyMetrics(d.metrics.Metrics); km != "" {
			parts = append(parts, "metrics: "+km)
		}
	}

	// With raw SQL disabled, never emit the stored example (it carries literal
	// values); fall back to PMM's normalized explain_fingerprint. Source-verified
	// (percona/pmm v3): the agent builds explain_fingerprint from the
	// performance-schema DIGEST_TEXT with numbered placeholders, so literals
	// never enter it; MySQL agents populate it even with examples disabled,
	// pg_stat_monitor does not. Do NOT switch to POST /v1/qan:explainFingerprint:
	// that endpoint deliberately returns the RAW example when one is stored.
	if d.example != nil {
		switch {
		case rawSQL && d.example.Example != "":
			parts = append(parts, "example:\n```sql\n"+d.example.Example+"\n```")
		case !rawSQL && d.example.ExplainFingerprint != "":
			parts = append(parts, "example (normalized, literals stripped; raw SQL disabled):\n```sql\n"+d.example.ExplainFingerprint+"\n```")
		}
	}

	parts = append(parts, "", link("View this query in PMM", qanQueryURL(base, d.queryID, d.serviceName, from, to, "")))
	return strings.Join(parts, "\n")
}

// keyMetrics summarizes the diagnostic metrics the triage cares about.
func keyMetrics(m map[string]metricStats) string {
	var out []string
	add := func(name, label string, pick func(metricStats) float64, skipZero bool) {
		v, ok := m[name]
		if !ok {
			return
		}
		f := pick(v)
		if skipZero && f == 0 {
			return
		}
		out = append(out, label+"="+fmtNum3(f))
	}
	sum := func(v metricStats) float64 { return finite(v.Sum) }
	avg := func(v metricStats) float64 { return finite(v.Avg) }
	add("num_queries", "calls", sum, false)
	add("query_time", "query_time_sum", sum, false)
	add("query_time", "query_time_avg", avg, false)
	add("rows_examined", "rows_examined_avg", avg, false)
	add("rows_sent", "rows_sent_avg", avg, false)
	add("lock_time", "lock_time_sum", sum, true)
	add("no_index_used", "no_index_used", sum, true)
	add("full_scan", "full_scan", sum, true)
	add("filesort", "filesort", sum, true)
	add("tmp_table_on_disk", "tmp_table_on_disk", sum, true)
	add("shared_blks_read", "shared_blks_read_sum", sum, true)
	add("blk_read_time", "blk_read_time_sum", sum, true)
	return strings.Join(out, ", ")
}

// serviceVersion reads one service's version from the exporter metric for its
// engine; any failure returns "".
func (s *Service) serviceVersion(ctx context.Context, auth callerAuth, engine, serviceName string) string {
	vm, ok := versionMetrics[engine]
	if !ok || serviceName == "" {
		return ""
	}
	samples, err := s.queryMetrics(ctx, auth, vm.metric+`{service_name="`+escapeLabel(serviceName)+`"}`, time.Time{})
	if err != nil || len(samples) == 0 {
		return ""
	}
	return samples[0].Labels[vm.label]
}

// escapeLabel escapes a value used inside a PromQL label matcher.
func escapeLabel(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v)
}

// engineOf looks a service up in the inventory and returns its engine.
func (s *Service) engineOf(ctx context.Context, auth callerAuth, serviceID string) (serviceInfo, error) {
	services, err := s.listServices(ctx, auth)
	if err != nil {
		return serviceInfo{}, err
	}
	i := slices.IndexFunc(services, func(svc serviceInfo) bool { return svc.ServiceID == serviceID })
	if i < 0 {
		return serviceInfo{}, newToolError(codeNotFound, "service '%s' not found; use pmm_inventory to list service ids", serviceID)
	}
	return services[i], nil
}
