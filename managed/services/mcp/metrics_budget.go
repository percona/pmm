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
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The response budget of pmm_query_metrics.
const (
	defaultSeries       = 20
	maxSeriesShown      = 100
	maxMetricsBytes     = 16 << 10
	defaultMetricsRange = time.Hour
)

// maxPoints bounds the points of a response, and the steps of a query or a subquery over the span it reads.
const maxPoints = 2000

// maxHintExprBytes bounds the expression a hint quotes back, so that a long one cannot use up the byte budget.
const maxHintExprBytes = 200

// internalLabels are PMM's own labels, which tell a caller nothing about the workload.
var internalLabels = []string{"agent_id", "agent_type", "node_id", "machine_id", "instance", "job"}

// chooseStep returns the query step: the requested one, or the smallest whole number of seconds that keeps
// the span within maxPoints points, and never under a second. An instant query (no span) has no step.
func chooseStep(span, requested time.Duration) time.Duration {
	if span <= 0 {
		return 0
	}
	return max(ceilSecond(requested), ceilSecond(span/maxPoints), time.Second)
}

// outPoint is one raw point of a series; V is null for NaN and ±Inf.
type outPoint struct {
	T time.Time `json:"t" jsonschema:"Sample time."`
	V *float64  `json:"v" jsonschema:"Sample value; null when it is NaN or infinite."`
}

// seriesSummary is one series of a metrics result.
type seriesSummary struct {
	Labels map[string]string `json:"labels" jsonschema:"Untrusted. The series' labels other than shared_labels and PMM's internal ones; empty when there are none."`
	Min    *float64          `json:"min" jsonschema:"Smallest finite value; null when there is none."`
	Avg    *float64          `json:"avg" jsonschema:"Average of the finite values; null when there is none."`
	Max    *float64          `json:"max" jsonschema:"Largest finite value; null when there is none."`
	Last   *float64          `json:"last" jsonschema:"Last value; null when it is NaN or infinite."`
	LastAt time.Time         `json:"last_at" jsonschema:"Time of the last value."`
	Points []outPoint        `json:"points,omitempty" jsonschema:"The raw points; left out when the result is summarized."`
}

// metricsResult is the result of pmm_query_metrics.
type metricsResult struct {
	Query        string            `json:"query" jsonschema:"The expression that ran."`
	Window       *window           `json:"window,omitempty" jsonschema:"The time range a range query covers."`
	Step         string            `json:"step,omitempty" jsonschema:"The step of a range query."`
	SharedLabels map[string]string `json:"shared_labels,omitempty" jsonschema:"Untrusted. Labels with the same value on every series shown."`
	Series       []seriesSummary   `json:"series" jsonschema:"The series, the largest maximum first."`
	TotalSeries  int               `json:"total_series" jsonschema:"Series the query returned, shown or not."`
	Truncated    bool              `json:"truncated" jsonschema:"True when some series are left out."`
	Summarized   bool              `json:"summarized" jsonschema:"True when the series carry no raw points."`
	Hint         string            `json:"hint,omitempty" jsonschema:"How to narrow the query, when series are left out."`
	Note         string            `json:"note,omitempty" jsonschema:"Values returned as null."`
}

var _ renderer = metricsResult{}

func (r metricsResult) render() string {
	var b strings.Builder
	b.WriteString("Query: " + r.Query + "\n")
	if r.Window != nil {
		b.WriteString(windowLine(*r.Window) + "\n")
	}
	if r.Step != "" {
		b.WriteString("Step: " + r.Step + "\n")
	}
	if len(r.SharedLabels) > 0 {
		b.WriteString("Shared labels: " + renderLabels(r.SharedLabels) + "\n")
	}
	fmt.Fprintf(&b, "Series: %d of %d", len(r.Series), r.TotalSeries)
	if r.Summarized {
		b.WriteString(" (summaries; raw points left out)")
	}
	b.WriteString("\n")
	for _, s := range r.Series {
		fmt.Fprintf(&b, "%s min=%s avg=%s max=%s last=%s\n",
			renderLabels(s.Labels), fmtOptional(s.Min), fmtOptional(s.Avg), fmtOptional(s.Max), fmtOptional(s.Last))
	}
	if r.Hint != "" {
		b.WriteString("\n" + r.Hint + "\n")
	}
	if r.Note != "" {
		b.WriteString("\n" + r.Note + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderLabels writes labels as {name="value", ...} in name order, values quoted so that none can span lines.
func renderLabels(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for _, k := range slices.Sorted(maps.Keys(labels)) {
		parts = append(parts, k+"="+strconv.Quote(labels[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func fmtOptional(v *float64) string {
	if v == nil {
		return "null"
	}
	return fmtNum3(*v)
}

// rankedSeries is a series with its statistics, before it is placed in a result.
type rankedSeries struct {
	summary   seriesSummary
	max       float64
	nonFinite int
	points    []outPoint
}

// shapeSeries fits series to the response budget: the top limit series by maximum, raw points when they
// fit and summaries when they don't, and fewer series while the text and the JSON together exceed 16 KiB.
func shapeSeries(expr string, series []metricSeries, limit int) metricsResult {
	if limit <= 0 {
		limit = defaultSeries
	}
	limit = min(limit, maxSeriesShown)

	ranked := make([]rankedSeries, len(series))
	for i, s := range series {
		ranked[i] = rankSeries(s)
	}
	slices.SortStableFunc(ranked, func(a, b rankedSeries) int { return cmp.Compare(b.max, a.max) })
	kept := ranked[:min(limit, len(ranked))]

	points := 0
	for _, s := range kept {
		points += len(s.points)
	}

	raw := points <= maxPoints
	res := buildMetricsResult(expr, kept, len(series), raw)
	if raw && metricsSize(res) > maxMetricsBytes {
		raw = false
		res = buildMetricsResult(expr, kept, len(series), raw)
	}
	for metricsSize(res) > maxMetricsBytes && len(kept) > 0 {
		kept = kept[:len(kept)-1]
		res = buildMetricsResult(expr, kept, len(series), raw)
	}
	return res
}

// metricsSize is what a response costs: its text plus its structured content.
func metricsSize(r metricsResult) int {
	// A result holds no NaN or ±Inf and no unsupported type, so it always marshals.
	b, _ := json.Marshal(r) //nolint:errchkjson
	return len(r.render()) + len(b)
}

// rankSeries drops a series' internal labels and computes its statistics over the finite points.
func rankSeries(s metricSeries) rankedSeries {
	labels := make(map[string]string, len(s.Labels))
	for k, v := range s.Labels {
		if !slices.Contains(internalLabels, k) {
			labels[k] = v
		}
	}

	r := rankedSeries{summary: seriesSummary{Labels: labels}, max: math.Inf(-1), points: make([]outPoint, len(s.Points))}
	var sum float64
	finite := 0
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, p := range s.Points {
		r.points[i] = outPoint{T: p.T, V: nullable(p.V)}
		if r.points[i].V == nil {
			r.nonFinite++
			continue
		}
		finite++
		sum += p.V
		lo = min(lo, p.V)
		hi = max(hi, p.V)
	}
	if finite > 0 {
		avg := sum / float64(finite)
		r.summary.Min, r.summary.Avg, r.summary.Max = &lo, &avg, &hi
		r.max = hi
	}
	if n := len(s.Points); n > 0 {
		r.summary.Last = nullable(s.Points[n-1].V)
		r.summary.LastAt = s.Points[n-1].T
	}
	return r
}

// buildMetricsResult assembles a result from the kept series, moving the labels they all share to SharedLabels.
func buildMetricsResult(expr string, kept []rankedSeries, total int, raw bool) metricsResult {
	res := metricsResult{
		Query:       expr,
		Series:      make([]seriesSummary, len(kept)),
		TotalSeries: total,
		Truncated:   len(kept) < total,
		Summarized:  !raw,
	}

	var shared map[string]string
	if len(kept) > 1 {
		shared = sharedLabels(kept)
	}
	if len(shared) > 0 {
		res.SharedLabels = shared
	}

	nonFinite := 0
	for i, s := range kept {
		row := s.summary
		row.Labels = make(map[string]string, len(s.summary.Labels))
		for k, v := range s.summary.Labels {
			if _, ok := shared[k]; !ok {
				row.Labels[k] = v
			}
		}
		if raw {
			row.Points = s.points
		}
		res.Series[i] = row
		nonFinite += s.nonFinite
	}

	if res.Truncated {
		res.Hint = leftOutHint(expr, total-len(kept), len(kept))
	}
	if nonFinite > 0 {
		res.Note = fmt.Sprintf("%d values were NaN or infinite: they are null in points and left out of min, avg and max.", nonFinite)
	}
	return res
}

// sharedLabels returns the labels that every series has with the same value.
func sharedLabels(series []rankedSeries) map[string]string {
	shared := maps.Clone(series[0].summary.Labels)
	for _, s := range series[1:] {
		for k, v := range shared {
			if got, ok := s.summary.Labels[k]; !ok || got != v {
				delete(shared, k)
			}
		}
	}
	return shared
}

func leftOutHint(expr string, left, shown int) string {
	if len(expr) > maxHintExprBytes {
		expr = "<expr>"
	}
	return fmt.Sprintf("%d series are left out; %d are shown, the largest by maximum. "+
		"Narrow the selectors with a service_* or node_* matcher, keep the largest with topk(10, %[3]s), "+
		"or aggregate with sum by (service_name) (%[3]s).", left, shown, expr)
}
