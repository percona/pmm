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
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-openapi/strfmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

const (
	defaultLimit = 10
	maxLimit     = 100
	groupByQuery = "queryid"

	// Extra columns per pmm_top_queries call, at most.
	maxColumns = 10
	// Longest search text, in characters.
	maxSearchLen = 200
)

// ordering is the QAN sort key and main metric for one order_by value; qan-api2
// sorts query_time by its average and load by its sum.
type ordering struct {
	orderKey   string
	mainMetric string
}

// orderMetrics maps order_by values to a QAN ordering; load and total_query_time
// share a sort key because load is sum(query_time) over the window.
var orderMetrics = map[string]ordering{
	"load":             {orderKey: "load", mainMetric: "load"},
	"total_query_time": {orderKey: "load", mainMetric: "query_time"},
	"avg_query_time":   {orderKey: "query_time", mainMetric: "query_time"},
	"count":            {orderKey: "num_queries", mainMetric: "num_queries"},
}

// groupByDimensions are the qan-api2 dimensions a report can be grouped by.
var groupByDimensions = []string{groupByQuery, "service_name", "database", "schema", "username", "client_host", "application_name", "cmd_type"}

// labelKey matches a label or dimension name; any other name is rejected (PMM-15715).
var labelKey = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// baseColumns are on every report row already.
var baseColumns = []string{"load", "num_queries", "query_time"}

type topQueriesInput struct {
	ServiceName string              `json:"service_name,omitempty" jsonschema:"Filter by service name (from pmm_inventory)"`
	ServiceID   string              `json:"service_id,omitempty" jsonschema:"Filter by service id (from pmm_inventory)"`
	PeriodFrom  string              `json:"period_from,omitempty" jsonschema:"Window start: RFC3339 or relative such as now-1h (default now-1h)"`
	PeriodTo    string              `json:"period_to,omitempty" jsonschema:"Window end: RFC3339 or relative (default now)"`
	GroupBy     string              `json:"group_by,omitempty" jsonschema:"Row dimension, default queryid; the tool description lists the others"`
	Labels      map[string][]string `json:"labels,omitempty" jsonschema:"Filters: a label or dimension name mapped to the values to keep, such as username: [app]"`
	Columns     []string            `json:"columns,omitempty" jsonschema:"Up to 10 more QAN metrics per row; the tool description lists the names"`
	OrderBy     string              `json:"order_by,omitempty" jsonschema:"load (default), total_query_time, avg_query_time, count or [-]column; see tool description"`
	Offset      int                 `json:"offset,omitempty" jsonschema:"Rows to skip, for paging (default 0)"`
	Limit       int                 `json:"limit,omitempty" jsonschema:"Number of rows, 1-100 (default 10)"`
	Search      string              `json:"search,omitempty" jsonschema:"Keep rows whose queryid, fingerprint or group_by value contains this text; 200 characters max"`
}

type queryDetailInput struct {
	QueryID    string `json:"queryid" jsonschema:"Query id from pmm_top_queries"`
	ServiceID  string `json:"service_id,omitempty" jsonschema:"Restrict to one service id"`
	PeriodFrom string `json:"period_from,omitempty" jsonschema:"Window start: RFC3339 or relative such as now-1h (default now-1h)"`
	PeriodTo   string `json:"period_to,omitempty" jsonschema:"Window end: RFC3339 or relative (default now)"`
}

// topQueriesResult is the result of pmm_top_queries.
type topQueriesResult struct {
	Rows    []queryRow `json:"rows" jsonschema:"Report rows on this page, in rank order."`
	Page    page       `json:"page" jsonschema:"Paging of the report, which QAN pages."`
	GroupBy string     `json:"group_by" jsonschema:"Dimension the rows are grouped by."`
	OrderBy string     `json:"order_by" jsonschema:"Ranking, as the request gave it."`
	Window  window     `json:"window" jsonschema:"Time window of the report."`
	PMMURL  string     `json:"pmm_url" jsonschema:"Link to this workload in PMM Query Analytics."`
}

// queryRow is one row of a pmm_top_queries report.
type queryRow struct {
	Rank           int                `json:"rank" jsonschema:"Position in the whole report, 1-based, offset included."`
	Dimension      string             `json:"dimension" jsonschema:"Untrusted. The row's group_by value, such as a queryid or a username; empty when QAN has none."`
	Fingerprint    string             `json:"fingerprint,omitempty" jsonschema:"Untrusted. A queryid row's normalized statement, masked or withheld with raw SQL off."`
	Load           float64            `json:"load" jsonschema:"Query load: total query time divided by the window's length."`
	Calls          float64            `json:"calls" jsonschema:"Number of calls."`
	QueryTimeTotal float64            `json:"query_time_total" jsonschema:"Total query time, in seconds."`
	QueryTimeAvg   float64            `json:"query_time_avg" jsonschema:"Average query time per call, in seconds."`
	Columns        map[string]float64 `json:"columns,omitempty" jsonschema:"Requested columns by name: the per-call average for an avg column, else the sum."`
	QueryRef       *queryRef          `json:"query_ref,omitempty" jsonschema:"The query and window, for pmm_query_detail and pmm_get_explain; on queryid rows."`
	PMMURL         string             `json:"pmm_url,omitempty" jsonschema:"Link to this row in PMM."`

	// stats are the requested columns the row has, in request order, for the text.
	stats []columnStat
}

// columnStat is both statistics of one requested column of a report row.
type columnStat struct {
	name     string
	avg, sum float64
}

func (r topQueriesResult) render() string {
	workload := link("Open this workload in PMM QAN", r.PMMURL)
	if len(r.Rows) == 0 {
		header := "No queries found for that service / time window."
		if r.Page.Offset > 0 {
			header = fmt.Sprintf("No rows at offset %d; the report has %d.", r.Page.Offset, r.Page.Total)
		}
		return header + "\n" + windowLine(r.Window) + "\n\n" + workload
	}
	lines := make([]string, 0, len(r.Rows))
	for _, row := range r.Rows {
		lines = append(lines, r.renderRow(row))
	}
	text := fmt.Sprintf("Rows %d-%d of %d (group_by=%s, order_by=%s):\n%s\n\n%s",
		r.Page.Offset+1, r.Page.Offset+len(r.Rows), r.Page.Total, r.GroupBy, r.OrderBy, windowLine(r.Window), strings.Join(lines, "\n"))
	text += "\n\n" + workload
	if m := r.Page.marker(); m != "" {
		text += "\n" + m
	}
	return text
}

// renderRow formats one report row: its dimension, load, calls, total and
// average time, the requested columns, and for queryid rows the fingerprint.
func (r topQueriesResult) renderRow(row queryRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d. [%s] load=%s calls=%s total=%s avg=%s", row.Rank, cmp.Or(row.Dimension, "(empty)"),
		fmtNum3(row.Load), fmtNum(row.Calls), fmtSeconds(row.QueryTimeTotal), fmtSeconds(row.QueryTimeAvg))
	for _, st := range row.stats {
		c, _ := qanColumnByName(st.name)
		ranked := st.name == strings.TrimPrefix(r.OrderBy, "-")
		if c.avg || ranked {
			fmt.Fprintf(&b, " %s_avg=%s", st.name, fmtNum3(st.avg))
		}
		if !c.avg || ranked {
			fmt.Fprintf(&b, " %s=%s", st.name, fmtNum3(st.sum))
		}
	}
	if r.GroupBy == groupByQuery {
		b.WriteString("\n   " + cmp.Or(row.Fingerprint, row.Dimension))
	}
	return b.String()
}

func (s *Service) registerQANTools(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_top_queries",
		Title: "Top queries by load",
		Description: "Rank queries, or another QAN dimension (group_by: service_name, database, schema, username, client_host, " +
			"application_name, cmd_type), over a time window, from PMM Query Analytics (QAN). order_by load and total_query_time " +
			"rank by total time, avg_query_time by the slowest calls, count by frequency, and a column by its value (- for " +
			"descending): ranked by the total, or by the per-call average for time metrics, as QAN does, and printed as both. " +
			"Rows carry load, calls and timing plus the requested columns; queryid rows (the default) also carry " +
			"the fingerprint, and the queryid goes to pmm_query_detail or pmm_get_explain. Page with offset. Columns: " +
			qanColumnsText() + ".",
		Annotations: readOnly("Top queries by load"),
	}, s.topQueries)

	addTool(s, server, &mcp.Tool{
		Name:  "pmm_query_detail",
		Title: "Query detail",
		Description: "Fetch the fingerprint, schema, engine and key diagnostic metrics (rows examined/sent, " +
			"no_index_used, full_scan, filesort) plus an example statement for a single query, by queryid.",
		Annotations: readOnly("Query detail"),
	}, s.queryDetail)
}

func (s *Service) topQueries(ctx context.Context, req *mcp.CallToolRequest, in topQueriesInput) (topQueriesResult, error) {
	from, to, err := parseWindow(in.PeriodFrom, in.PeriodTo, s.now())
	if err != nil {
		return topQueriesResult{}, err
	}
	q, err := newReportQuery(in, from, to)
	if err != nil {
		return topQueriesResult{}, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	report, err := s.api.GetReport(ctx, auth, q.body)
	if err != nil {
		return topQueriesResult{}, err
	}
	raw := s.rawSQL()
	rowEngine := func(string) string { return "" }
	if !raw && q.groupBy == groupByQuery && slices.ContainsFunc(report.Rows, func(r qanReportRow) bool { return quotingMatters(r.Fingerprint) }) {
		rowEngine = s.reportEngines(ctx, auth, q, report.Rows, from, to)
	}

	// Confirmed against PMM 3.8.1: rows[0] is the TOTAL aggregate (empty
	// dimension); per-metric stats are nested under <name>.stats. Row.database
	// arrives empty for MySQL rows, so schema comes from pmm_query_detail.
	rows := make([]queryRow, 0, len(report.Rows))
	for i, row := range report.Rows {
		if i == 0 && row.Dimension == "" {
			continue
		}
		rows = append(rows, q.row(row, q.offset+len(rows)+1, rowEngine(row.Dimension), raw))
	}
	if q.groupBy == groupByQuery && len(rows) > 0 {
		serviceID := s.pinnedServiceID(ctx, auth, q.labels)
		for i := range rows {
			if rows[i].Dimension != "" {
				rows[i].QueryRef = &queryRef{QueryID: rows[i].Dimension, ServiceID: serviceID, PeriodFrom: from, PeriodTo: to}
			}
		}
	}

	total := int(report.TotalRows)
	return topQueriesResult{
		Rows:    rows,
		Page:    page{Total: total, Offset: q.offset, Limit: int(q.body.Limit), Truncated: q.offset+len(rows) < total},
		GroupBy: q.groupBy,
		OrderBy: q.orderBy,
		Window:  window{From: from, To: to},
		PMMURL:  qanOverviewURL(s.publicBaseURL(ctx, req.Extra.Header), in.ServiceName, from, to),
	}, nil
}

// pinnedServiceID returns the id of the one service a report's filters pin, by one service_id or else one
// service_name found in the caller's service list, and "" when they pin none or that lookup fails.
func (s *Service) pinnedServiceID(ctx context.Context, auth callerAuth, labels map[string][]string) string {
	if ids := labels["service_id"]; len(ids) == 1 {
		return ids[0]
	}
	names := labels["service_name"]
	if len(names) != 1 {
		return ""
	}
	services, err := s.listServices(ctx, auth, false)
	if err != nil {
		s.l.WithField("tool", "pmm_top_queries").Debugf("Cannot list services, query_ref will have no service_id: %s.", err)
		return ""
	}
	i := slices.IndexFunc(services, func(svc serviceInfo) bool { return svc.ServiceName == names[0] })
	if i < 0 {
		return ""
	}
	return services[i].ServiceID
}

// reportQuery is a validated pmm_top_queries request.
type reportQuery struct {
	groupBy string
	// orderBy is as the caller wrote it.
	orderBy string
	// columns are the extra columns, in request order.
	columns []string
	offset  int
	labels  map[string][]string
	body    qan_service.GetReportBody
}

// newReportQuery validates a pmm_top_queries input and builds its getReport body.
func newReportQuery(in topQueriesInput, from, to time.Time) (*reportQuery, error) {
	q := &reportQuery{groupBy: cmp.Or(in.GroupBy, groupByQuery), orderBy: cmp.Or(in.OrderBy, "load"), offset: in.Offset}
	if !slices.Contains(groupByDimensions, q.groupBy) {
		return nil, newToolError(codeInvalidInput, "group_by must be one of %s; got '%s'", strings.Join(groupByDimensions, ", "), q.groupBy)
	}
	limit := cmp.Or(in.Limit, defaultLimit)
	if limit < 1 || limit > maxLimit {
		return nil, newToolError(codeInvalidInput, "limit must be between 1 and %d; got %d", maxLimit, limit)
	}
	if in.Offset < 0 {
		return nil, newToolError(codeInvalidInput, "offset must be 0 or more; got %d", in.Offset)
	}
	if utf8.RuneCountInString(in.Search) > maxSearchLen {
		return nil, newToolError(codeInvalidInput, "search must be at most %d characters", maxSearchLen)
	}
	columns, err := checkColumns(in.Columns)
	if err != nil {
		return nil, err
	}
	qanOrder, mainMetric, orderColumn, err := parseOrderBy(q.orderBy)
	if err != nil {
		return nil, err
	}
	if orderColumn != "" && !slices.Contains(columns, orderColumn) {
		columns = append(columns, orderColumn)
	}
	q.columns = columns
	q.labels, err = reportLabels(in)
	if err != nil {
		return nil, err
	}

	labels := make([]*qan_service.GetReportParamsBodyLabelsItems0, 0, len(q.labels))
	for _, key := range slices.Sorted(maps.Keys(q.labels)) {
		labels = append(labels, &qan_service.GetReportParamsBodyLabelsItems0{Key: key, Value: q.labels[key]})
	}
	q.body = qan_service.GetReportBody{
		PeriodStartFrom: strfmt.DateTime(from),
		PeriodStartTo:   strfmt.DateTime(to),
		GroupBy:         q.groupBy,
		OrderBy:         qanOrder,
		Offset:          int64(in.Offset),
		Limit:           int64(limit),
		Columns:         append(slices.Clone(baseColumns), columns...),
		MainMetric:      mainMetric,
		Labels:          labels,
		Search:          in.Search,
	}
	return q, nil
}

// checkColumns validates the requested columns against the catalogue and drops repeats.
func checkColumns(columns []string) ([]string, error) {
	if len(columns) > maxColumns {
		return nil, newToolError(codeInvalidInput, "columns takes at most %d names; got %d", maxColumns, len(columns))
	}
	out := make([]string, 0, len(columns))
	for _, c := range columns {
		if _, ok := qanColumnByName(c); !ok {
			return nil, newToolError(codeInvalidInput, "unknown column '%s'; the tool description lists the valid names", c)
		}
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out, nil
}

// parseOrderBy maps order_by to qan-api2's order_by and main metric: one of the
// four named rankings (descending), or [-]<column>, whose column is returned too.
func parseOrderBy(orderBy string) (string, string, string, error) {
	if o, ok := orderMetrics[orderBy]; ok {
		return "-" + o.orderKey, o.mainMetric, "", nil
	}
	name := strings.TrimPrefix(orderBy, "-")
	if _, ok := qanColumnByName(name); !ok {
		return "", "", "", newToolError(codeInvalidInput,
			"order_by must be load, total_query_time, avg_query_time, count, or a column from the tool description with an optional - prefix; got '%s'", orderBy)
	}
	return orderBy, name, name, nil
}

// reportLabels merges the labels input with the service_name and service_id
// shortcuts, and checks every name and value (PMM-15715).
func reportLabels(in topQueriesInput) (map[string][]string, error) {
	labels := maps.Clone(in.Labels)
	if labels == nil {
		labels = make(map[string][]string)
	}
	for _, sc := range []struct{ key, value string }{{"service_name", in.ServiceName}, {"service_id", in.ServiceID}} {
		if sc.value == "" {
			continue
		}
		if vals, ok := labels[sc.key]; ok && !slices.Equal(vals, []string{sc.value}) {
			return nil, newToolError(codeInvalidInput, "%s and labels.%s disagree; pass one of them", sc.key, sc.key)
		}
		labels[sc.key] = []string{sc.value}
	}
	for key, values := range labels {
		if !labelKey.MatchString(key) {
			return nil, newToolError(codeInvalidInput, "label name '%s' must match %s", key, labelKey)
		}
		if len(values) == 0 {
			return nil, newToolError(codeInvalidInput, "labels.%s needs at least one value", key)
		}
		for _, v := range values {
			err := checkQANValue("labels."+key, v)
			if err != nil {
				return nil, err
			}
		}
	}
	return labels, nil
}

// row builds one report row: its load, calls, total and average time, the
// requested columns, and for queryid rows the fingerprint, masked unless raw.
func (q *reportQuery) row(row qanReportRow, rank int, engine string, raw bool) queryRow {
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

	r := queryRow{
		Rank:           rank,
		Dimension:      row.Dimension,
		Load:           load,
		Calls:          numQueries,
		QueryTimeTotal: total,
		QueryTimeAvg:   avg,
		Columns:        make(map[string]float64, len(q.columns)),
	}
	for _, name := range q.columns {
		m, ok := row.Metrics[name]
		if slices.Contains(baseColumns, name) || !ok || m.Stats == nil {
			continue
		}
		c, _ := qanColumnByName(name)
		st := columnStat{name: name, avg: finite(m.Stats.Avg), sum: finite(m.Stats.Sum)}
		r.stats = append(r.stats, st)
		r.Columns[name] = st.sum
		if c.avg {
			r.Columns[name] = st.avg
		}
	}
	if q.groupBy == groupByQuery {
		r.Fingerprint = fingerprintText(row.Fingerprint, engine, raw)
	}
	return r
}

// queryDetailResult is the result of pmm_query_detail.
type queryDetailResult struct {
	QueryID      string             `json:"queryid" jsonschema:"Query id."`
	Engine       string             `json:"engine,omitempty" jsonschema:"Engine: mysql, postgresql or mongodb."`
	Version      string             `json:"version,omitempty" jsonschema:"Untrusted. Server version, left out when it cannot be read."`
	ServiceName  string             `json:"service_name,omitempty" jsonschema:"Untrusted. Service name."`
	Database     string             `json:"database,omitempty" jsonschema:"Untrusted. Database name; QAN fills it for PostgreSQL and MongoDB."`
	Schema       string             `json:"schema,omitempty" jsonschema:"Untrusted. Schema name; QAN fills it for MySQL."`
	Fingerprint  string             `json:"fingerprint,omitempty" jsonschema:"Untrusted. Normalized statement, masked or withheld when raw SQL is off."`
	Tables       []string           `json:"tables,omitempty" jsonschema:"Untrusted. Tables the stored example reads."`
	Metrics      map[string]float64 `json:"metrics,omitempty" jsonschema:"Key diagnostic metrics by the names the text prints; zero counters are left out."`
	Example      string             `json:"example,omitempty" jsonschema:"Untrusted. Example statement, raw with raw SQL on, else normalized; left out when withheld."`
	QueryRef     queryRef           `json:"query_ref" jsonschema:"The query, its service and the window, for pmm_get_explain and pmm_get_schema."`
	Window       window             `json:"window" jsonschema:"Time window of the data."`
	PMMURL       string             `json:"pmm_url" jsonschema:"Link to this query in PMM Query Analytics."`
	Partial      bool               `json:"partial" jsonschema:"Set when qan:getMetrics or qan/query:getExample failed; its error is in metrics_error or example_error."`
	MetricsError *partError         `json:"metrics_error,omitempty" jsonschema:"Failure of qan:getMetrics, which returns the fingerprint, metrics and service."`
	ExampleError *partError         `json:"example_error,omitempty" jsonschema:"Failure of qan/query:getExample, which returns the example and tables."`

	// keyMetrics are Metrics in the order the text prints them.
	keyMetrics []metricValue
	// rawSQL is set when Example is the stored statement rather than its normalized form.
	rawSQL bool
}

func (r queryDetailResult) render() string {
	parts := []string{"queryid: " + r.QueryID, windowLine(r.Window)}
	if r.Engine != "" {
		engine := r.Engine
		if r.Version != "" {
			engine += " " + r.Version
		}
		parts = append(parts, "engine: "+engine)
	}
	if r.ServiceName != "" {
		parts = append(parts, "service: "+r.ServiceName)
	}
	// QAN fills database for PostgreSQL and MongoDB, schema for MySQL.
	if r.Database != "" {
		parts = append(parts, "database: "+r.Database)
	}
	if r.Schema != "" {
		parts = append(parts, "schema: "+r.Schema)
	}
	if len(r.Tables) > 0 {
		parts = append(parts, "tables: "+strings.Join(r.Tables, ", "))
	}
	if r.Fingerprint != "" {
		parts = append(parts, "fingerprint:\n"+r.Fingerprint)
	}
	if len(r.keyMetrics) > 0 {
		parts = append(parts, "metrics: "+metricsText(r.keyMetrics))
	}
	if r.Example != "" {
		label := "example (normalized, literals stripped; raw SQL disabled):"
		if r.rawSQL {
			label = "example:"
		}
		parts = append(parts, label+"\n```sql\n"+r.Example+"\n```")
	}
	for _, p := range []struct {
		call string
		err  *partError
	}{{"qan:getMetrics", r.MetricsError}, {"qan/query:getExample", r.ExampleError}} {
		if p.err != nil {
			parts = append(parts, fmt.Sprintf("partial: %s failed (%s)", p.call, p.err.Code))
		}
	}

	parts = append(parts, "", link("View this query in PMM", r.PMMURL))
	return strings.Join(parts, "\n")
}

func (s *Service) queryDetail(ctx context.Context, req *mcp.CallToolRequest, in queryDetailInput) (queryDetailResult, error) {
	if in.QueryID == "" {
		return queryDetailResult{}, newToolError(codeInvalidInput, "queryid is required")
	}
	err := cmp.Or(checkQANValue("queryid", in.QueryID), checkQANValue("service_id", in.ServiceID))
	if err != nil {
		return queryDetailResult{}, err
	}
	from, to, err := parseWindow(in.PeriodFrom, in.PeriodTo, s.now())
	if err != nil {
		return queryDetailResult{}, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)

	d, err := s.fetchQueryDetail(ctx, auth, in, from, to)
	if err != nil {
		return queryDetailResult{}, err
	}
	if d.metrics == nil && d.example == nil || d.fingerprint == "" && d.example == nil && len(d.metrics.Metrics) == 0 {
		return queryDetailResult{}, newToolError(codeNotFound,
			"no data for queryid '%s' in the selected window; widen period_from or check the id with pmm_top_queries", in.QueryID)
	}
	if d.engine != "" {
		d.version = s.serviceVersion(ctx, auth, d.engine, d.serviceName)
	}

	return d.result(s.rawSQL(), s.publicBaseURL(ctx, req.Extra.Header), from, to), nil
}

// queryDetail is the merged view of qan:getMetrics and qan/query:getExample
// for one queryid.
type queryDetail struct {
	queryID     string
	engine      string
	version     string
	serviceID   string
	serviceName string
	database    string
	schema      string
	fingerprint string
	tables      []string
	metrics     *queryMetrics
	example     *qan_service.GetQueryExampleOKBodyQueryExamplesItems0
	// metricsErr and exampleErr are the failures of the two QAN calls; at most one is set.
	metricsErr error
	exampleErr error
}

// fetchQueryDetail runs the two QAN calls, each tolerated independently so
// partial data still returns with the other's error; it fails only when both fail.
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

	d := &queryDetail{queryID: in.QueryID, metrics: metrics, metricsErr: metricsErr, exampleErr: exampleErr}
	if metrics != nil {
		d.fingerprint = metrics.Fingerprint
		if metrics.Metadata != nil {
			d.database = metrics.Metadata.Database
			d.schema = metrics.Metadata.Schema
			d.serviceID = metrics.Metadata.ServiceID
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
		d.serviceID = cmp.Or(d.example.ServiceID, d.serviceID)
		d.tables = d.example.Tables
	}
	return d, nil
}

// result builds the detail's result, with the fingerprint and example masked
// unless rawSQL is set, so that both renderings show the same text.
func (d *queryDetail) result(rawSQL bool, base string, from, to time.Time) queryDetailResult {
	r := queryDetailResult{
		QueryID:     d.queryID,
		Engine:      d.engine,
		Version:     d.version,
		ServiceName: d.serviceName,
		Database:    d.database,
		Schema:      d.schema,
		Tables:      d.tables,
		QueryRef:    queryRef{QueryID: d.queryID, ServiceID: d.serviceID, PeriodFrom: from, PeriodTo: to},
		Window:      window{From: from, To: to},
		PMMURL:      qanQueryURL(base, d.queryID, d.serviceName, from, to, ""),
		rawSQL:      rawSQL,
	}
	if d.fingerprint != "" {
		r.Fingerprint = fingerprintText(d.fingerprint, d.engine, rawSQL)
	}
	if d.metrics != nil {
		r.keyMetrics = keyMetrics(d.metrics.Metrics)
		r.Metrics = make(map[string]float64, len(r.keyMetrics))
		for _, m := range r.keyMetrics {
			r.Metrics[m.name] = m.value
		}
	}

	// With raw SQL off, show the agent's explain_fingerprint (never qan:explainFingerprint,
	// which returns the raw example) unmasked if unquoted, keeping its :1 placeholders.
	if d.example != nil {
		switch {
		case rawSQL:
			r.Example = d.example.Example
		case fingerprintText(d.example.ExplainFingerprint, d.engine, false) != withheldFingerprint:
			r.Example = d.example.ExplainFingerprint
		}
	}

	if d.metricsErr != nil {
		r.MetricsError = partErrorOf(d.metricsErr)
	}
	if d.exampleErr != nil {
		r.ExampleError = partErrorOf(d.exampleErr)
	}
	r.Partial = r.MetricsError != nil || r.ExampleError != nil
	return r
}

// windowLine renders a result's window for its text, in RFC 3339 UTC like the structured window.
func windowLine(w window) string {
	return "Window: " + w.From.UTC().Format(time.RFC3339) + " – " + w.To.UTC().Format(time.RFC3339)
}

// checkQANValue rejects a QAN filter value with a quote or a backslash (PMM-15715).
func checkQANValue(input, value string) error {
	if strings.ContainsAny(value, `'\`) {
		return newToolError(codeInvalidInput, "%s must not contain a quote (') or a backslash (\\)", input)
	}
	return nil
}

// withheldFingerprint replaces a SQL fingerprint that still holds a literal.
const withheldFingerprint = "(fingerprint withheld: PMM stored this statement unnormalized, and PMM_MCP_RAW_SQL is off)"

// fingerprintText masks a fingerprint unless raw SQL is on: a SQL one with a quoted
// string under its engine's quoting (either, if unknown) is a raw statement and is withheld.
func fingerprintText(fingerprint, engine string, raw bool) string {
	if raw {
		return fingerprint
	}
	if strings.HasPrefix(fingerprint, "db.") {
		return maskMongoFingerprint(fingerprint)
	}
	masked, quoted := maskSQL(fingerprint, engine == engineMySQL)
	if engine != engineMySQL && engine != enginePostgreSQL {
		_, mysqlQuoted := maskSQL(fingerprint, true)
		quoted = quoted || mysqlQuoted
	}
	if quoted {
		return withheldFingerprint
	}
	return masked
}

// quotingMatters reports whether a SQL fingerprint reads differently under MySQL
// and PostgreSQL quoting, so that showing it needs the report's engine.
func quotingMatters(fingerprint string) bool {
	if fingerprint == "" || strings.HasPrefix(fingerprint, "db.") {
		return false
	}
	return fingerprintText(fingerprint, engineMySQL, false) != fingerprintText(fingerprint, enginePostgreSQL, false)
}

// reportEngines returns the engine whose quoting reads each queryid's
// fingerprint: the report's one SQL engine when its filters or QAN's data name
// one, else each row's own, from rowEngines (PMM-15529).
func (s *Service) reportEngines(ctx context.Context, auth callerAuth, q *reportQuery, rows []qanReportRow, from, to time.Time) func(string) string {
	engine := s.reportEngine(ctx, auth, q.labels, from, to)
	if _, filtered := q.labels["service_type"]; engine != "" || filtered {
		// A service_type filter is the caller's; rowEngines would replace it.
		return func(string) string { return engine }
	}
	engines := s.rowEngines(ctx, auth, q.body, rows)
	return func(queryID string) string { return engines[queryID] }
}

// reportEngine returns the engine of a whole report: its one SQL service_type,
// else the engine of the one service it is filtered to, else the only SQL
// engine QAN holds data for in the window, else "".
func (s *Service) reportEngine(ctx context.Context, auth callerAuth, labels map[string][]string, from, to time.Time) string {
	if types, ok := labels["service_type"]; ok {
		if len(types) == 1 && (types[0] == engineMySQL || types[0] == enginePostgreSQL) {
			return types[0]
		}
		return ""
	}
	var name, id string
	if names := labels["service_name"]; len(names) == 1 {
		name = names[0]
	}
	if ids := labels["service_id"]; len(ids) == 1 {
		id = ids[0]
	}
	if name != "" || id != "" {
		return s.serviceEngine(ctx, auth, name, id)
	}
	types, err := s.api.QANServiceTypes(ctx, auth, from, to)
	if err != nil {
		s.l.WithField("tool", "pmm_top_queries").Debugf("qan metrics:getFilters failed: %s.", err)
		return ""
	}
	sqlTypes := slices.DeleteFunc(types, func(t string) bool { return t == engineMongoDB })
	if len(sqlTypes) != 1 {
		return ""
	}
	return sqlTypes[0]
}

// rowEngines finds each ambiguous queryid's engine with one more report, the
// same one filtered to MySQL: a queryid with all of its calls there is MySQL,
// one absent is PostgreSQL, and one in both stays unknown, so it is withheld.
func (s *Service) rowEngines(ctx context.Context, auth callerAuth, body qan_service.GetReportBody, rows []qanReportRow) map[string]string {
	calls := make(map[string]float64)
	var ids []string
	for _, r := range rows {
		if r.Dimension != "" && quotingMatters(r.Fingerprint) {
			ids = append(ids, r.Dimension)
			calls[r.Dimension] = finite(r.NumQueries)
		}
	}
	body.Labels = append(slices.Clone(body.Labels),
		&qan_service.GetReportParamsBodyLabelsItems0{Key: groupByQuery, Value: ids},
		&qan_service.GetReportParamsBodyLabelsItems0{Key: "service_type", Value: []string{engineMySQL}})
	body.Columns, body.OrderBy, body.MainMetric = []string{"num_queries"}, "-num_queries", "num_queries"
	body.Offset, body.Limit = 0, int64(len(ids))
	mysql, err := s.api.GetReport(ctx, auth, body)
	if err != nil {
		s.l.WithField("tool", "pmm_top_queries").Debugf("qan metrics:getReport for MySQL rows failed: %s.", err)
		return nil
	}

	engines := make(map[string]string, len(ids))
	for _, id := range ids {
		engines[id] = enginePostgreSQL
	}
	for _, r := range mysql.Rows {
		if want, ok := calls[r.Dimension]; ok {
			engines[r.Dimension] = ""
			if finite(r.NumQueries) == want {
				engines[r.Dimension] = engineMySQL
			}
		}
	}
	return engines
}

// serviceEngine returns the engine of the service a report is filtered to, by
// name or else by id, or "" when it is not found.
func (s *Service) serviceEngine(ctx context.Context, auth callerAuth, name, id string) string {
	if name == "" && id == "" {
		return ""
	}
	services, err := s.listServices(ctx, auth, false)
	if err != nil {
		return ""
	}
	for _, svc := range services {
		if (name != "" && svc.ServiceName == name) || (name == "" && svc.ServiceID == id) {
			return svc.Engine
		}
	}
	return ""
}

// metricValue is one named diagnostic metric of pmm_query_detail.
type metricValue struct {
	name  string
	value float64
}

// metricsText renders metrics as name=value pairs, comma-separated.
func metricsText(metrics []metricValue) string {
	pairs := make([]string, 0, len(metrics))
	for _, m := range metrics {
		pairs = append(pairs, m.name+"="+fmtNum3(m.value))
	}
	return strings.Join(pairs, ", ")
}

// keyMetrics returns the diagnostic metrics the triage cares about, in the order the text prints them.
func keyMetrics(m map[string]metricStats) []metricValue {
	var out []metricValue
	add := func(name, label string, pick func(metricStats) float64, skipZero bool) {
		v, ok := m[name]
		if !ok {
			return
		}
		f := pick(v)
		if skipZero && f == 0 {
			return
		}
		out = append(out, metricValue{name: label, value: f})
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
	add("docs_examined", "docs_examined_avg", avg, true)
	add("keys_examined", "keys_examined_avg", avg, true)
	add("docs_returned", "docs_returned_avg", avg, true)
	add("response_length", "response_length_avg", avg, true)
	add("storage_bytes_read", "storage_bytes_read_avg", avg, true)
	add("locks_global_acquire_count_read_shared", "locks_global_acquire_count_read_shared", sum, true)
	return out
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
	services, err := s.listServices(ctx, auth, false)
	if err != nil {
		return serviceInfo{}, err
	}
	i := slices.IndexFunc(services, func(svc serviceInfo) bool { return svc.ServiceID == serviceID })
	if i < 0 {
		return serviceInfo{}, newToolError(codeNotFound, "service '%s' not found; use pmm_inventory to list service ids", serviceID)
	}
	return services[i], nil
}
