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
	"errors"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// Series one selector may match over the span a query reads.
	maxSeriesPerSelector = 2000

	// Metric calls one caller may have in flight.
	maxMetricCallsPerCaller = 4
	// Metric calls one pmm-managed replica may have in flight.
	maxMetricCallsPerReplica = 8
)

type queryMetricsInput struct {
	Query      string `json:"query" jsonschema:"MetricsQL expression, for example sum by (service_name) (rate(mysql_global_status_questions[5m]))"`
	Time       string `json:"time,omitempty" jsonschema:"Instant query: evaluation time, RFC3339 or relative such as now-1h; default now. Not with a period"`
	PeriodFrom string `json:"period_from,omitempty" jsonschema:"Range query: start, RFC3339 or relative such as now-6h; default now-1h when only period_to is given"`
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
}

// metricsPlan is a validated call of pmm_query_metrics.
type metricsPlan struct {
	instant bool
	// at is the evaluation time of an instant query.
	at time.Time
	// start and end bound a range query.
	start, end time.Time
	// step is the step of a range query; stepGiven tells whether the caller chose it.
	step      time.Duration
	stepGiven bool
	limit     int
}

// span is the range a query covers; an instant query covers none.
func (p metricsPlan) span() time.Duration {
	if p.instant {
		return 0
	}
	return p.end.Sub(p.start)
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
		plan.at = at.UTC().Truncate(time.Second)
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

	checkStep := instantStep
	if !plan.instant {
		checkStep = plan.step
	}
	read, err := checkExpr(in.Query, plan.span(), checkStep)
	// A step the caller did not choose rises to what an implicit subquery needs; one the caller chose never changes.
	tooSmall, ok := errors.AsType[*stepTooSmallError](err)
	if ok && !plan.instant && !plan.stepGiven {
		plan.step = max(plan.step, tooSmall.minStep)
		checkStep = plan.step
		read, err = checkExpr(in.Query, plan.span(), checkStep)
	}
	if err != nil {
		return metricsResult{}, err
	}

	auth := callerAuthFromHeader(req.Extra.Header)
	release, err := s.startMetrics(auth)
	if err != nil {
		return metricsResult{}, err
	}
	defer release()

	uid, err := s.metricsDatasourceUID(ctx, auth)
	if err != nil {
		return metricsResult{}, err
	}

	selectors, err := selectorsOf(in.Query, checkStep)
	if err != nil {
		return metricsResult{}, err
	}
	from, to := plan.at, plan.at
	if !plan.instant {
		from, to = plan.start, plan.end
	}
	err = s.checkSeriesCount(ctx, auth, uid, selectors, from.Add(-(read - plan.span())), to)
	if err != nil {
		return metricsResult{}, narrowIfTooLarge(err, in.Query)
	}

	if plan.instant {
		samples, err := s.api.QueryInstant(ctx, auth, uid, promRequest{Query: in.Query, Time: plan.at, Timeout: metricsTimeout})
		if err != nil {
			return metricsResult{}, narrowIfTooLarge(err, in.Query)
		}
		series := make([]metricSeries, len(samples))
		for i, sample := range samples {
			series[i] = metricSeries{Labels: sample.Labels, Points: []point{{T: plan.at, V: sample.Value}}}
		}
		return shapeSeries(in.Query, series, plan.limit, nil, ""), nil
	}

	series, err := s.api.QueryRange(ctx, auth, uid, promRequest{
		Query: in.Query, Start: plan.start, End: plan.end, Step: plan.step, Timeout: metricsTimeout,
	})
	if err != nil {
		return metricsResult{}, narrowIfTooLarge(err, in.Query)
	}
	return shapeSeries(in.Query, series, plan.limit, &window{From: plan.start, To: plan.end}, formatStep(plan.step)), nil
}

// narrowIfTooLarge turns a response over the 4 MiB bound into advice to narrow the query: the expression is the model's own.
func narrowIfTooLarge(err error, expr string) error {
	if !errors.Is(err, errResponseTooLarge) {
		return err
	}
	if len(expr) > maxHintExprBytes {
		expr = "<expr>"
	}
	return newToolError(codeInvalidInput,
		"the answer is larger than 4 MiB: narrow the query with a service_name or node_name matcher, and aggregate, as in sum by (service_name) (%s)", expr)
}

// startMetrics takes a metric-call slot for the caller, or answers rate_limited.
func (s *Service) startMetrics(a callerAuth) (func(), error) {
	release, hit := s.metrics.acquire(callerKey(a))
	switch hit {
	case limitPerCaller:
		return nil, newToolError(codeRateLimited,
			"at most %d metric calls can be in flight for these credentials; retry when one finishes", maxMetricCallsPerCaller)
	case limitPerReplica:
		return nil, newToolError(codeRateLimited,
			"this replica already has %d metric calls in flight; retry when one finishes", maxMetricCallsPerReplica)
	case limitNone:
	}
	return release, nil
}

// selectorsOf returns each selector of expr, one per OR group, in MetricsQL form, as walked at step.
func selectorsOf(expr string, step time.Duration) ([]string, error) {
	e, err := metricsql.Parse(expr)
	if err != nil {
		return nil, newToolError(codeInvalidInput, "cannot parse the expression: %s", echo(err.Error()))
	}
	w, err := walkSelectors(e, step)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(w.selectors))
	for i, sel := range w.selectors {
		out[i] = sel.String()
	}
	return out, nil
}

// checkSeriesCount refuses a query one of whose selectors matches more than maxSeriesPerSelector series over [start, end].
func (s *Service) checkSeriesCount(ctx context.Context, auth callerAuth, uid string, selectors []string, start, end time.Time) error {
	for _, sel := range selectors {
		series, err := s.api.Series(ctx, auth, uid, []string{sel}, start, end, maxSeriesPerSelector+1)
		if err != nil {
			return err
		}
		if len(series) > maxSeriesPerSelector {
			return newToolError(codeInvalidInput,
				"this selector matches more than %d series over the span; narrow it with a service_name or node_name matcher, and aggregate (selector: %s)",
				maxSeriesPerSelector, echo(sel))
		}
	}
	return nil
}
