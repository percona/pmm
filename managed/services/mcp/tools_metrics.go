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
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// Series one selector may match over the span a query reads.
	maxSeriesPerSelector = 2000

	// Times the rules raise a step the caller did not choose before the query is refused with the last hint, and times
	// the selectors are counted again over the longer span a raised step reads before a query that would read further is refused.
	maxStepRaises = 4
)

type queryMetricsInput struct {
	Query      string `json:"query" jsonschema:"MetricsQL expression, for example sum by (service_name) (rate(mysql_global_status_questions[5m]))"`
	Time       string `json:"time,omitempty" jsonschema:"Instant query: evaluation time, RFC3339 or relative such as now-1h; default now. Not with a period"`
	PeriodFrom string `json:"period_from,omitempty" jsonschema:"Range query: start, RFC3339 or relative such as now-6h; default 1 h before period_to"`
	PeriodTo   string `json:"period_to,omitempty" jsonschema:"Range query: end, RFC3339 or relative; default now"`
	Step       string `json:"step,omitempty" jsonschema:"Range query only: step in whole seconds, such as 30s or 5m; default keeps the range within 2000 points"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Maximum series to return, the largest by maximum first: 1 to 100, default 20"`
}

func (s *Service) registerMetricsTools(server *mcp.Server) {
	addTool(s, server, &mcp.Tool{
		Name:  "pmm_query_metrics",
		Title: "Query metrics",
		Description: "Run a MetricsQL query (PromQL-compatible) on PMM's metrics, instant at time (default now) or over a range with " +
			"period_from and period_to (default the last hour). Aggregate first: ask for sum by (service_name) (rate(mysql_global_status_questions[5m])) " +
			"or topk(10, ...) rather than raw series. Every selector needs a metric name, and a regex name also a service_* or node_* matcher; " +
			"an expression reads at most 7 days and a selector may match at most 2000 series, or the query is refused with the reason. " +
			"A result holds the top limit series (default 20, at most 100) by maximum, in at most 2000 points and 16 KiB: raw points when they fit, " +
			"summaries (min, avg, max, last) when they don't. NaN and infinite values are null.",
		Annotations: readOnly("Query metrics"),
	}, s.metricsQuery)

	addTool(s, server, &mcp.Tool{
		Name:  "pmm_list_metrics",
		Title: "List metrics",
		Description: "Discover what PMM's metrics hold before writing a query. kind names (default) lists metric names matching search, " +
			"optionally of one service (service_id or service_name); over limit names it returns groups of names by prefix with counts instead. " +
			"kind labels lists label names, kind values the values of one label, and kind series the label sets of the series, each narrowed by match " +
			"selectors (required for series). Each looks back at most 7 days (period_from, default 1 hour), returns at most 1000 entries from " +
			"VictoriaMetrics, and pages with offset and limit (default 100, at most 1000); a page is also held to 16 KiB.",
		Annotations: readOnly("List metrics"),
	}, s.listMetrics)

	s.registerHealthTool(server)
}

// metricsPlan is a validated call of pmm_query_metrics.
type metricsPlan struct {
	instant bool
	// start and end bound a range query; both are the evaluation time of an instant query.
	start, end time.Time
	// step is the step of a range query; stepGiven tells whether the caller chose it.
	step      time.Duration
	stepGiven bool
	limit     int
}

// span is the range a query covers; an instant query covers none.
func (p metricsPlan) span() time.Duration {
	return p.end.Sub(p.start)
}

// check checks e at the plan's step with checkParsed, and the root of an instant query with checkInstantRoot.
func (p metricsPlan) check(e metricsql.Expr) (time.Duration, []selector, error) {
	read, selectors, err := checkParsed(e, p.span(), p.step)
	if err == nil && p.instant {
		err = checkInstantRoot(e)
	}
	return read, selectors, err
}

func (s *Service) planMetrics(in queryMetricsInput) (metricsPlan, error) {
	limit := in.Limit
	if limit == 0 {
		limit = defaultSeries
	}
	if limit < 1 || limit > maxSeriesShown {
		return metricsPlan{}, newToolError(codeInvalidInput, "limit must be between 1 and %d; got %d", maxSeriesShown, in.Limit)
	}
	plan := metricsPlan{limit: limit}
	now := s.now()

	if in.PeriodFrom == "" && in.PeriodTo == "" {
		if in.Step != "" {
			return metricsPlan{}, newToolError(codeInvalidInput, "step applies to range queries only; pass period_from or period_to, or leave step out")
		}
		at, err := parseTime(in.Time, now)
		if err != nil {
			return metricsPlan{}, err
		}
		plan.instant = true
		plan.end = at.UTC().Truncate(time.Second)
		plan.start = plan.end
		return plan, nil
	}

	if in.Time != "" {
		return metricsPlan{}, newToolError(codeInvalidInput, "time is for an instant query; use either time, or period_from and period_to")
	}
	start, end, err := parseWindow(in.PeriodFrom, in.PeriodTo, now)
	if err != nil {
		return metricsPlan{}, err
	}
	plan.start, plan.end = start, end

	var requested time.Duration
	if in.Step != "" {
		requested, err = parseStep(in.Step)
		if err != nil {
			return metricsPlan{}, err
		}
		plan.stepGiven = true
	}
	plan.step = chooseStep(plan.span(), requested)
	if plan.stepGiven && plan.step != requested {
		return metricsPlan{}, newToolError(codeInvalidInput,
			"at step %s, the window of %s is %d points per series; the cap is %d: raise step to at least %s, or shorten the window",
			formatStep(requested), formatSpan(plan.span()), int64(plan.span()/requested)+1, maxPoints, formatStep(plan.step))
	}
	return plan, nil
}

// parseStep reads a step written as a Go duration in whole seconds.
func parseStep(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, newToolError(codeInvalidInput,
			"step '%s' is not a duration: write it in seconds, minutes or hours, such as 30s, 5m or 1h", echo(s))
	}
	if d <= 0 || d%time.Second != 0 {
		return 0, newToolError(codeInvalidInput, "step must be a positive whole number of seconds; got '%s'", echo(s))
	}
	return d, nil
}

// formatStep writes a step the way parseStep reads it: 30s, 5m, 1h, 1h30m.
func formatStep(d time.Duration) string {
	r := d.String()
	if strings.HasSuffix(r, "m0s") {
		r = strings.TrimSuffix(r, "0s")
	}
	if strings.HasSuffix(r, "h0m") {
		r = strings.TrimSuffix(r, "0m")
	}
	return r
}

func (s *Service) metricsQuery(ctx context.Context, req *mcp.CallToolRequest, in queryMetricsInput) (metricsResult, error) {
	plan, err := s.planMetrics(in)
	if err != nil {
		return metricsResult{}, err
	}
	e, err := parseExpr(in.Query)
	if err != nil {
		return metricsResult{}, err
	}
	auth := callerAuthFromHeader(req.Extra.Header)
	narrow := "narrow the query with a service_name or node_name matcher, and aggregate, as in sum by (service_name) (" + hintExpr(in.Query) + ")"

	// Each pass checks the expression at the step, and a refusal can raise a step the caller did not choose. Once it
	// passes, the selectors are counted over the span it reads, again whenever a raise makes it read further back, and
	// the count can raise the step so that the answer fits under the read cap.
	raise := stepRaise{expr: in.Query, chosen: plan.step, ruleStep: plan.step, per: seriesPerMatched(e)}
	var uid string
	// counted is the span the selectors were last counted over, and counts how many times they were.
	counted, counts := time.Duration(-1), 0
	for {
		read, selectors, err := plan.check(e)
		if err != nil {
			err = raise.afterRefusal(&plan, err)
			if err != nil {
				return metricsResult{}, err
			}
			continue
		}
		if read <= counted {
			break
		}
		if counts > maxStepRaises {
			return metricsResult{}, newToolError(codeInvalidInput,
				"the step that fits the answer in the 4 MiB that MCP reads keeps rising, since each rise makes the query read further back, "+
					"where its selectors match more series: aggregate, as in sum by (service_name) (%s), or narrow the selectors with a service_name "+
					"or node_name matcher", hintExpr(in.Query))
		}

		if uid == "" {
			var release func()
			uid, release, err = s.startMetricsRequests(ctx, auth)
			if err != nil {
				return metricsResult{}, err
			}
			defer release()
		}
		matched, err := s.checkSeriesCount(ctx, auth, uid, selectors, plan.start.Add(plan.span()-read), plan.end)
		if err != nil {
			return metricsResult{}, narrowIfTooLarge(err, narrow)
		}
		counted, counts = read, counts+1
		step, err := raise.fitAnswer(plan, matched)
		if err != nil {
			return metricsResult{}, err
		}
		if step == plan.step {
			break
		}
		plan.step = step
	}

	if plan.instant {
		samples, err := s.api.QueryInstant(ctx, auth, uid, promRequest{Query: in.Query, Time: plan.end})
		if err != nil {
			return metricsResult{}, narrowIfTooLarge(err, narrow)
		}
		series := make([]metricSeries, len(samples))
		for i, sample := range samples {
			series[i] = metricSeries{Labels: sample.Labels, Points: []point{{T: plan.end, V: sample.Value}}}
		}
		return shapeSeries(in.Query, series, plan.limit, nil, "", ""), nil
	}
	series, err := s.api.QueryRange(ctx, auth, uid, promRequest{Query: in.Query, Start: plan.start, End: plan.end, Step: plan.step})
	if err != nil {
		return metricsResult{}, narrowIfTooLarge(err, narrow)
	}
	return shapeSeries(in.Query, series, plan.limit, &window{From: plan.start, To: plan.end}, formatStep(plan.step), raise.note(plan.step)), nil
}

// stepRaise is how the step of a query rises from the one planMetrics chose: to ruleStep, which the rules that reasons
// name need, then so that the answer fits under the read cap. A step the caller gave never rises.
type stepRaise struct {
	expr             string
	chosen, ruleStep time.Duration
	reasons          []string
	ruleRaises       int
	// per is how many series the answer has for each series its selectors match, 0 when it needs no fitting.
	per int
	// matched is the largest count of the selectors' series, and answered the series of the answer last fitted.
	matched  matchedSeries
	answered int
}

// afterRefusal raises plan's step to what the rule that refused it needs, at most maxStepRaises times, and returns
// nil; otherwise it returns the refusal, with why the step rose to where it was refused.
func (r *stepRaise) afterRefusal(plan *metricsPlan, err error) error {
	if plan.step != r.ruleStep {
		return newToolError(codeInvalidInput,
			"the answer for up to %d series fits in the 4 MiB that MCP reads only at a step of %s or more, and at that step the query reads past its budget: "+
				"aggregate, as in sum by (service_name) (%s), or narrow the selectors", r.answered, formatStep(plan.step), hintExpr(r.expr))
	}
	tooSmall, ok := errors.AsType[*stepTooSmallError](err)
	if ok && !plan.instant && !plan.stepGiven && tooSmall.minStep > plan.step && r.ruleRaises < maxStepRaises {
		plan.step, r.ruleStep = tooSmall.minStep, tooSmall.minStep
		r.reasons = appendOnce(r.reasons, tooSmall.reason)
		r.ruleRaises++
		return nil
	}
	te, ok := errors.AsType[*toolError](err)
	if ok && plan.step != r.chosen {
		return newToolError(te.code, "the step rises to %s so that %s; at that step: %s", formatStep(plan.step), strings.Join(r.reasons, ", and "), te.message)
	}
	return err
}

// fitAnswer returns the smallest step, from the one the rules need, at which the answer of the largest count so far
// of a range query without an aggregation fits under the read cap. It refuses any step when not even one point per
// series fits, and a step the caller gave that is too small.
func (r *stepRaise) fitAnswer(plan metricsPlan, matched matchedSeries) (time.Duration, error) {
	if matched.count > r.matched.count {
		r.matched = matched
	}
	if plan.instant || r.per == 0 {
		return plan.step, nil
	}
	answer := matchedSeries{count: r.matched.count * r.per, labelBytes: r.matched.labelBytes * r.per}
	step, ok := answerStep(answer, plan.span(), r.ruleStep)
	if !ok {
		return 0, newToolError(codeInvalidInput,
			"the answer for up to %d series is over the 4 MiB that MCP reads at any step: narrow the selectors with a service_name or node_name matcher, "+
				"or aggregate, as in sum by (service_name) (%s)", answer.count, hintExpr(r.expr))
	}
	if step != plan.step && plan.stepGiven {
		return 0, newToolError(codeInvalidInput,
			"at step %s, the answer for up to %d series is over the 4 MiB that MCP reads: raise step to at least %s, or aggregate, as in sum by (service_name) (%s)",
			formatStep(plan.step), answer.count, formatStep(step), hintExpr(r.expr))
	}
	r.answered = answer.count
	return step, nil
}

// note says why the step rose to step, empty when it did not.
func (r *stepRaise) note(step time.Duration) string {
	var notes []string
	if r.ruleStep != r.chosen {
		notes = append(notes, fmt.Sprintf("The step was raised from %s to %s so that %s.",
			formatStep(r.chosen), formatStep(r.ruleStep), strings.Join(r.reasons, ", and ")))
	}
	if step != r.ruleStep {
		notes = append(notes, fmt.Sprintf("The step was raised from %s to %s so that the answer for up to %d series fits in the 4 MiB that MCP reads; "+
			"aggregate for a finer step.", formatStep(r.ruleStep), formatStep(step), r.answered))
	}
	return strings.Join(notes, " ")
}

// startMetricsRequests takes a metric-call slot for the caller and finds the Metrics datasource, returning its UID and
// the slot's release.
func (s *Service) startMetricsRequests(ctx context.Context, auth callerAuth) (string, func(), error) {
	release, err := s.startMetricCall(auth)
	if err != nil {
		return "", nil, err
	}
	uid, err := s.metricsDatasourceUID(ctx, auth)
	if err != nil {
		release()
		return "", nil, err
	}
	return uid, release, nil
}

// seriesPerMatched returns how many series e can return for each series its selectors match, or 0 when it has an
// aggregation, which can return fewer.
func seriesPerMatched(e metricsql.Expr) int {
	aggregated := false
	metricsql.VisitAll(e, func(e metricsql.Expr) {
		_, ok := e.(*metricsql.AggrFuncExpr)
		aggregated = aggregated || ok
	})
	if aggregated {
		return 0
	}
	return resultsPerSeries(e)
}

// minMaxAvg are the results of rollup and the other rollup_* functions but rollup_candlestick.
var minMaxAvg = []string{"min", "max", "avg"}

// rollupResults are the rollup functions that return a series for each of their results, for each series they read,
// unless a non-empty second argument picks one result.
var rollupResults = map[string][]string{
	"rollup":                 minMaxAvg,
	"rollup_candlestick":     {"open", "high", "low", "close"},
	"rollup_delta":           minMaxAvg,
	"rollup_deriv":           minMaxAvg,
	"rollup_increase":        minMaxAvg,
	"rollup_rate":            minMaxAvg,
	"rollup_scrape_interval": minMaxAvg,
}

// resultsPerSeries returns how many series e returns at most for each series it reads, up to maxRollupNames+1: the
// functions an aggr_over_time call names, or the results of a rollupResults function, multiplied when nested.
func resultsPerSeries(e metricsql.Expr) int {
	switch e := e.(type) {
	case *metricsql.RollupExpr:
		return resultsPerSeries(e.Expr)
	case *metricsql.BinaryOpExpr:
		return max(resultsPerSeries(e.Left), resultsPerSeries(e.Right))
	case *metricsql.AggrFuncExpr:
		// An aggregation returns at most as many series as it reads.
		return mostResults(e.Args)
	case *metricsql.FuncExpr:
		most := mostResults(e.Args)
		name := strings.ToLower(e.Name)
		if name == "aggr_over_time" {
			var names []string
			for _, arg := range e.Args {
				names = appendRollupNames(names, arg)
			}
			return min(most*max(len(names), 1), maxRollupNames+1)
		}
		if results, ok := rollupResults[name]; ok && !picksOneResult(e.Args) {
			return min(most*len(results), maxRollupNames+1)
		}
		return most
	default:
		return 1
	}
}

// mostResults returns the largest resultsPerSeries of args, and 1 for none.
func mostResults(args []metricsql.Expr) int {
	most := 1
	for _, arg := range args {
		most = max(most, resultsPerSeries(arg))
	}
	return most
}

// picksOneResult tells whether a rollupResults function's arguments name one result: VictoriaMetrics returns all of
// them for a missing or empty second argument.
func picksOneResult(args []metricsql.Expr) bool {
	if len(args) <= 1 {
		return false
	}
	tag, ok := args[len(args)-1].(*metricsql.StringExpr)
	return ok && tag.S != ""
}

// narrowIfTooLarge turns a response over the 4 MiB bound into advice to narrow the request: the expression or
// selectors are the model's own.
func narrowIfTooLarge(err error, advice string) error {
	if !errors.Is(err, errResponseTooLarge) {
		return err
	}
	return newToolError(codeInvalidInput, "the answer is larger than 4 MiB: %s", advice)
}

// hintExpr is the caller's expression as a hint quotes it, or <expr> when it is long.
func hintExpr(expr string) string {
	if len(expr) > maxHintExprBytes {
		return "<expr>"
	}
	return expr
}

// checkInstantRoot refuses an instant query whose root is a range selector or a subquery, which
// VictoriaMetrics answers with a range of samples per series rather than one value.
func checkInstantRoot(e metricsql.Expr) error {
	for {
		re, ok := e.(*metricsql.RollupExpr)
		if !ok {
			return nil
		}
		if re.Window != nil || re.ForSubquery() {
			break
		}
		e = re.Expr
	}
	return newToolError(codeInvalidInput,
		"an instant query returns one value per series, so it can't be a range selector or a subquery such as x[5m] or x[1h:1m]: "+
			"wrap it in a rollup such as last_over_time() or rate(), or send a range query with period_from and period_to")
}

// matchedSeries is what the selectors of a query match: how many series, and the bytes their labels take in a JSON answer.
type matchedSeries struct {
	count      int
	labelBytes int
}

// checkSeriesCount refuses a query one of whose selectors matches more than maxSeriesPerSelector series over [start, end],
// and returns what they all match.
func (s *Service) checkSeriesCount(ctx context.Context, auth callerAuth, uid string, selectors []selector, start, end time.Time) (matchedSeries, error) {
	var m matchedSeries
	for _, selector := range selectors {
		sel := selector.String()
		series, err := s.api.Series(ctx, auth, uid, []string{sel}, start, end, maxSeriesPerSelector+1)
		if err != nil {
			return matchedSeries{}, err
		}
		if len(series) > maxSeriesPerSelector {
			return matchedSeries{}, newToolError(codeInvalidInput,
				"this selector matches more than %d series over the span; narrow it with a service_name or node_name matcher, and aggregate (selector: %s)",
				maxSeriesPerSelector, echo(sel))
		}
		m.count += len(series)
		for _, labels := range series {
			for k, v := range labels {
				m.labelBytes += len(k) + len(v) + answerLabelBytes
			}
		}
	}
	return m, nil
}

const (
	listKindNames  = "names"
	listKindLabels = "labels"
	listKindValues = "values"
	listKindSeries = "series"
)

const (
	// Most entries MCP lists from one VictoriaMetrics answer; the request asks for one more, to tell that there are more.
	maxListed = 1000
	// Page size of pmm_list_metrics.
	defaultListLimit = 100
	// Longest look-back of pmm_list_metrics.
	maxListLookback = maxSpan
	// Number of '_'-separated parts of a name that a group of names shares, at first.
	groupPrefixParts = 3
)

// listInputs are the inputs each kind of pmm_list_metrics takes, besides kind, period_from, limit and offset.
var listInputs = map[string][]string{
	listKindNames:  {"search", "service_id", "service_name"},
	listKindLabels: {"match"},
	listKindValues: {"label", "match"},
	listKindSeries: {"match"},
}

type listMetricsInput struct {
	Kind        string   `json:"kind,omitempty" jsonschema:"What to list: names (default), labels, values or series"`
	Search      string   `json:"search,omitempty" jsonschema:"kind names: text a metric name contains, such as mongodb; at most 200 characters; empty lists every name"`
	ServiceID   string   `json:"service_id,omitempty" jsonschema:"kind names: only metrics of this service ID"`
	ServiceName string   `json:"service_name,omitempty" jsonschema:"kind names: only metrics of this service name"`
	Label       string   `json:"label,omitempty" jsonschema:"kind values: the label whose values to list, such as service_name"`
	Match       []string `json:"match,omitempty" jsonschema:"kinds labels, values, series: plain selectors such as mysql_up; required for series; each names a metric"`
	PeriodFrom  string   `json:"period_from,omitempty" jsonschema:"Start of the look-back, RFC3339 or relative such as now-6h; default now-1h, at most 7 days back"`
	Limit       int      `json:"limit,omitempty" jsonschema:"Entries per page: 1 to 1000, default 100"`
	Offset      int      `json:"offset,omitempty" jsonschema:"Entries to skip, to read the next page"`
}

// metricName is one metric name, with its meaning and type when the catalogue has them.
type metricName struct {
	Name    string `json:"name" jsonschema:"Untrusted. The metric name."`
	Meaning string `json:"meaning,omitempty" jsonschema:"What the metric measures, from the catalogue; left out for a metric it does not list."`
	Type    string `json:"type,omitempty" jsonschema:"counter or gauge, from the catalogue; left out for a metric it does not list."`
}

// nameGroup is a prefix shared by a number of metric names.
type nameGroup struct {
	Prefix string `json:"prefix" jsonschema:"Untrusted. The shared prefix; ends in _* for a group of several names."`
	Count  int    `json:"count" jsonschema:"Metric names with the prefix."`
}

// listMetricsResult is the result of pmm_list_metrics; one of Names, Groups, Labels, Values and Series is set.
type listMetricsResult struct {
	Kind   string              `json:"kind" jsonschema:"What was listed: names, labels, values or series."`
	Names  []metricName        `json:"names,omitempty" jsonschema:"kind names: the metric names on this page."`
	Groups []nameGroup         `json:"groups,omitempty" jsonschema:"kind names: more names than a page; the names by prefix, with counts."`
	Labels []string            `json:"labels,omitempty" jsonschema:"Untrusted. kind labels: the label names on this page."`
	Values []string            `json:"values,omitempty" jsonschema:"Untrusted. kind values: the values on this page."`
	Series []map[string]string `json:"series,omitempty" jsonschema:"Untrusted. kind series: the label sets on this page, __name__ included."`
	Page   page                `json:"page" jsonschema:"Paging: total entries VictoriaMetrics returned, this page's offset and size, and whether more follow."`
	Hint   string              `json:"hint,omitempty" jsonschema:"How to narrow the search, when VictoriaMetrics returned its maximum or series were left out for size."`
}

var _ renderer = listMetricsResult{}

func (r listMetricsResult) render() string {
	var b strings.Builder
	b.WriteString("Kind: " + r.Kind + "\n")
	fmt.Fprintf(&b, "Total: %d\n", r.Page.Total)
	// Names, prefixes, labels and values are quoted as renderLabels quotes label values, so that none can span lines.
	for _, n := range r.Names {
		b.WriteString(strconv.Quote(n.Name))
		if n.Type != "" {
			b.WriteString(" [" + n.Type + "]")
		}
		if n.Meaning != "" {
			b.WriteString(": " + n.Meaning)
		}
		b.WriteString("\n")
	}
	for _, g := range r.Groups {
		fmt.Fprintf(&b, "%s (%d)\n", strconv.Quote(g.Prefix), g.Count)
	}
	for _, l := range r.Labels {
		b.WriteString(strconv.Quote(l) + "\n")
	}
	for _, v := range r.Values {
		b.WriteString(strconv.Quote(v) + "\n")
	}
	for _, s := range r.Series {
		b.WriteString(renderLabels(s) + "\n")
	}
	// Truncated also stands for VictoriaMetrics' cap, so the marker shows only while entries follow this page.
	if m := r.Page.marker(); m != "" && r.Page.Limit > 0 && r.Page.Offset+r.Page.Limit < r.Page.Total {
		b.WriteString("\n" + m + "\n")
	}
	if r.Hint != "" {
		b.WriteString("\n" + r.Hint + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// listPlan is a validated call of pmm_list_metrics.
type listPlan struct {
	kind string
	// label is the label whose values kind names or values lists.
	label      string
	match      []string
	start, end time.Time
	limit      int
}

func (s *Service) planList(in listMetricsInput) (listPlan, error) {
	kind := in.Kind
	if kind == "" {
		kind = listKindNames
	}
	allowed, ok := listInputs[kind]
	if !ok {
		return listPlan{}, newToolError(codeInvalidInput, "kind '%s' is not one of names, labels, values or series", echo(kind))
	}
	given := map[string]bool{
		"search":       in.Search != "",
		"service_id":   in.ServiceID != "",
		"service_name": in.ServiceName != "",
		"label":        in.Label != "",
		"match":        len(in.Match) > 0,
	}
	for _, name := range slices.Sorted(maps.Keys(given)) {
		if given[name] && !slices.Contains(allowed, name) {
			return listPlan{}, newToolError(codeInvalidInput, "kind %s does not take %s; it takes %s", kind, name, strings.Join(allowed, ", "))
		}
	}
	if in.ServiceID != "" && in.ServiceName != "" {
		return listPlan{}, newToolError(codeInvalidInput, "pass service_id or service_name, not both")
	}
	if utf8.RuneCountInString(in.Search) > maxSearchLen {
		return listPlan{}, newToolError(codeInvalidInput, "search must be at most %d characters", maxSearchLen)
	}
	if kind == listKindValues && in.Label == "" {
		return listPlan{}, newToolError(codeInvalidInput, "kind values needs a label, such as service_name")
	}
	if in.Label != "" && !labelNamePattern.MatchString(in.Label) {
		return listPlan{}, errBadLabel(in.Label)
	}
	if kind == listKindSeries && len(in.Match) == 0 {
		return listPlan{}, newToolError(codeInvalidInput, "kind series needs match, a selector such as up{service_name=\"db1\"}")
	}
	if len(in.Match) > maxSelectors {
		return listPlan{}, newToolError(codeInvalidInput, "match has %d selectors; the cap is %d", len(in.Match), maxSelectors)
	}
	err := checkPage(in.Offset, in.Limit, maxListed)
	if err != nil {
		return listPlan{}, err
	}

	start, end, err := parseWindow(in.PeriodFrom, "", s.now())
	if err != nil {
		return listPlan{}, err
	}
	lookback := end.Sub(start)
	if lookback > maxListLookback {
		return listPlan{}, newToolError(codeInvalidInput,
			"period_from looks back %s; the cap is %s: use a later start", formatSpan(lookback), formatSpan(maxListLookback))
	}
	for _, sel := range in.Match {
		e, err := parseExpr(sel)
		if err != nil {
			return listPlan{}, err
		}
		// The list endpoints read the index over the window, not samples, so a selector has no lookbehind: span 0.
		_, _, err = checkParsed(e, 0, instantStep)
		if err != nil {
			return listPlan{}, err
		}
		// A function, an operator, a range or an offset is refused: a match is a series selector alone.
		_, ok := e.(*metricsql.MetricExpr)
		if !ok {
			return listPlan{}, newToolError(codeInvalidInput,
				"match '%s' is not a series selector: write a metric name and label matchers, such as mysql_up{service_name=\"db1\"}, "+
					"without functions, operators, ranges or offset", echo(sel))
		}
	}

	plan := listPlan{kind: kind, label: in.Label, match: in.Match, start: start, end: end, limit: cmp.Or(in.Limit, defaultListLimit)}
	if kind == listKindNames {
		plan.label = "__name__"
		sel := nameSelector(in.Search, in.ServiceID, in.ServiceName)
		if sel != "" {
			plan.match = []string{sel}
		}
	}
	return plan, nil
}

// nameSelector is the selector of a name search (names containing search, of the service when one is given), empty
// when it has no matcher, and the one selector that skips checkExpr, since it reads names only (ADR-15).
func nameSelector(search, serviceID, serviceName string) string {
	var matchers []string
	if search != "" {
		matchers = append(matchers, `__name__=~".*`+escapeLabel(regexp.QuoteMeta(search))+`.*"`)
	}
	if serviceID != "" {
		matchers = append(matchers, `service_id="`+escapeLabel(serviceID)+`"`)
	}
	if serviceName != "" {
		matchers = append(matchers, `service_name="`+escapeLabel(serviceName)+`"`)
	}
	if len(matchers) == 0 {
		return ""
	}
	return "{" + strings.Join(matchers, ",") + "}"
}

// groupNames returns names as they are when there are at most limit of them, and otherwise groups them by
// their first three '_'-separated parts, or by the first two when that is still over limit.
func groupNames(names []string, limit int) ([]string, []nameGroup) {
	if len(names) <= limit {
		return names, nil
	}
	groups := groupByPrefix(names, groupPrefixParts)
	if len(groups) > limit {
		groups = groupByPrefix(names, groupPrefixParts-1)
	}
	return nil, groups
}

func groupByPrefix(names []string, parts int) []nameGroup {
	counts := make(map[string]int)
	for _, n := range names {
		p := strings.Split(n, "_")
		prefix := n
		if len(p) > parts {
			prefix = strings.Join(p[:parts], "_") + "_*"
		}
		counts[prefix]++
	}
	groups := make([]nameGroup, 0, len(counts))
	for _, prefix := range slices.Sorted(maps.Keys(counts)) {
		groups = append(groups, nameGroup{Prefix: prefix, Count: counts[prefix]})
	}
	return groups
}

func (s *Service) listMetrics(ctx context.Context, req *mcp.CallToolRequest, in listMetricsInput) (listMetricsResult, error) {
	plan, err := s.planList(in)
	if err != nil {
		return listMetricsResult{}, err
	}

	auth := callerAuthFromHeader(req.Extra.Header)
	uid, release, err := s.startMetricsRequests(ctx, auth)
	if err != nil {
		return listMetricsResult{}, err
	}
	defer release()

	var values []string
	var series []map[string]string
	switch plan.kind {
	case listKindLabels:
		values, err = s.api.Labels(ctx, auth, uid, plan.match, plan.start, plan.end, maxListed+1)
	case listKindSeries:
		series, err = s.api.Series(ctx, auth, uid, plan.match, plan.start, plan.end, maxListed+1)
	default:
		values, err = s.api.LabelValues(ctx, auth, uid, plan.label, plan.match, plan.start, plan.end, maxListed+1)
	}
	if err != nil {
		return listMetricsResult{}, narrowIfTooLarge(err,
			"narrow the request with a longer search, a service_name, or a match selector with a service_name or node_name matcher")
	}

	res := listMetricsResult{Kind: plan.kind}
	total := len(values) + len(series)
	if total > maxListed {
		res.Hint = "VictoriaMetrics returned more than 1000 entries, so some are not listed: " + narrowAdvice(plan.kind)
	}
	slices.Sort(values)
	values = slices.Compact(values)
	switch plan.kind {
	case listKindNames:
		res = pageNames(res, values, in.Offset, plan.limit)
	case listKindLabels:
		res.Labels, res.Page = paginate(values, in.Offset, plan.limit, defaultListLimit)
	case listKindValues:
		res.Values, res.Page = paginate(values, in.Offset, plan.limit, defaultListLimit)
	case listKindSeries:
		// In the order of their labels, each label set a JSON object even when VictoriaMetrics sent null.
		type keyedSeries struct {
			key    string
			labels map[string]string
		}
		keyed := make([]keyedSeries, len(series))
		for i, labels := range series {
			if labels == nil {
				labels = map[string]string{}
			}
			keyed[i] = keyedSeries{key: renderLabels(labels), labels: labels}
		}
		slices.SortStableFunc(keyed, func(a, b keyedSeries) int { return strings.Compare(a.key, b.key) })
		for i, k := range keyed {
			series[i] = k.labels
		}
		res.Series, res.Page = paginate(series, in.Offset, plan.limit, defaultListLimit)
	}

	res = fitList(res)
	if total > maxListed {
		res.Page.Truncated = true
	}
	return res, nil
}

func narrowAdvice(kind string) string {
	switch kind {
	case listKindNames:
		return "narrow the search with a longer search text or a service_name"
	case listKindSeries:
		return "narrow match with a service_name or node_name matcher"
	default:
		return "narrow it with match"
	}
}

// pageNames sets the names of one page, or their groups when there are more than limit.
func pageNames(res listMetricsResult, names []string, offset, limit int) listMetricsResult {
	names, groups := groupNames(names, limit)
	if groups != nil {
		res.Groups, res.Page = paginate(groups, offset, limit, defaultListLimit)
		return res
	}
	names, res.Page = paginate(names, offset, limit, defaultListLimit)
	res.Names = make([]metricName, len(names))
	for i, n := range names {
		entry := metricsCatalogIndex()[n]
		res.Names[i] = metricName{Name: n, Meaning: entry.Meaning, Type: entry.Type}
	}
	return res
}

// fitList keeps the most entries of a page, of any kind, whose text and structured content fit maxMetricsBytes.
func fitList(res listMetricsResult) listMetricsResult {
	if resultSize(res) <= maxMetricsBytes {
		return res
	}

	hint := "The " + res.Kind + " do not fit in 16 KiB, so some are left out: " + narrowAdvice(res.Kind) + ", or read the next page with offset."
	if res.Hint != "" {
		hint = res.Hint + ". " + hint
	}
	cut := func(n int) listMetricsResult {
		r := res
		r.Names, r.Groups, r.Labels = res.Names[:min(n, len(res.Names))], res.Groups[:min(n, len(res.Groups))], res.Labels[:min(n, len(res.Labels))]
		r.Values, r.Series = res.Values[:min(n, len(res.Values))], res.Series[:min(n, len(res.Series))]
		r.Page.Limit = n
		r.Page.Truncated = true
		r.Hint = hint
		return r
	}
	// One kind of entries is set, so the page holds this many.
	shown := len(res.Names) + len(res.Groups) + len(res.Labels) + len(res.Values) + len(res.Series)
	kept := sort.Search(shown, func(n int) bool { return resultSize(cut(n+1)) > maxMetricsBytes })
	res = cut(kept)
	if kept == 0 {
		res.Hint += fmt.Sprintf(" The entry at offset %d alone does not fit in 16 KiB: pass offset=%d to skip it.", res.Page.Offset, res.Page.Offset+1)
	}
	return res
}
