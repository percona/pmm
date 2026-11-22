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
	"sort"
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

// The estimate of a range answer's size: bytes per point, such as [1700000000,"12345"], per series besides its labels,
// and per label besides its name and value, as in "name":"value",.
const (
	answerPointBytes  = 32
	answerSeriesBytes = 32
	answerLabelBytes  = 6
)

// maxRangeAnswerBytes is the estimated answer a range query's step is raised to fit: 3.5 MiB, under the 4 MiB read cap.
const maxRangeAnswerBytes = maxVMResponseBytes / 8 * 7

// internalLabels are PMM's own labels, which tell a caller nothing about the workload.
var internalLabels = []string{"agent_id", "agent_type", "node_id", "machine_id", "instance", "job"}

// chooseStep returns the query step, 0 for an instant query (no span), else the larger of the requested one and the
// smallest whole number of seconds that keeps the span within maxPoints points, and never under a second.
func chooseStep(span, requested time.Duration) time.Duration {
	if span <= 0 {
		return 0
	}
	// span/step + 1 points fit in maxPoints when the step is over span/maxPoints.
	return max(ceilSecond(requested), (span/maxPoints).Truncate(time.Second)+time.Second)
}

// answerBytes estimates the size of a range answer of the matched series over span at step.
func answerBytes(m matchedSeries, span, step time.Duration) int {
	points := int(span/step) + 1
	return m.labelBytes + m.count*(answerSeriesBytes+points*answerPointBytes)
}

// answerStep returns the smallest step, not under step, whose answer of the matched series is within maxRangeAnswerBytes;
// step itself when the answer fits, or when even one point per series does not.
func answerStep(m matchedSeries, span, step time.Duration) time.Duration {
	if m.count == 0 || answerBytes(m, span, step) <= maxRangeAnswerBytes {
		return step
	}
	perSeries := (maxRangeAnswerBytes - m.labelBytes - m.count*answerSeriesBytes) / (m.count * answerPointBytes)
	if perSeries < 1 {
		return step
	}
	// span/step + 1 points fit in perSeries when the step is over span/perSeries.
	return max(step, (span/time.Duration(perSeries)).Truncate(time.Second)+time.Second)
}

// outPoint is one raw point of a series; V is null for NaN and ±Inf.
type outPoint struct {
	T time.Time `json:"t" jsonschema:"Sample time."`
	V *float64  `json:"v" jsonschema:"Sample value; null when it is NaN or infinite."`
}

// seriesSummary is one series of a metrics result.
type seriesSummary struct {
	Labels map[string]string `json:"labels" jsonschema:"Untrusted. Labels other than shared_labels and PMM's internal ones that tell no series apart; empty when none."`
	Min    *float64          `json:"min" jsonschema:"Smallest finite value; null when there is none."`
	Avg    *float64          `json:"avg" jsonschema:"Average of the finite values; null when there is none."`
	Max    *float64          `json:"max" jsonschema:"Largest finite value; null when there is none."`
	Last   *float64          `json:"last" jsonschema:"Last value; null when it is NaN or infinite."`
	LastAt time.Time         `json:"last_at" jsonschema:"Time of the last value."`
	Points []outPoint        `json:"points,omitempty" jsonschema:"The raw points; left out when the result is summarized."`
}

// metricsResult is the result of pmm_query_metrics.
type metricsResult struct {
	Query        string            `json:"query" jsonschema:"The expression that ran; only its first 200 bytes when the whole does not fit in the response."`
	Window       *window           `json:"window,omitempty" jsonschema:"The time range a range query covers."`
	Step         string            `json:"step,omitempty" jsonschema:"The step of a range query."`
	SharedLabels map[string]string `json:"shared_labels,omitempty" jsonschema:"Untrusted. Labels with the same value on every series shown."`
	Series       []seriesSummary   `json:"series" jsonschema:"The series, the largest maximum first."`
	TotalSeries  int               `json:"total_series" jsonschema:"Series the query returned, shown or not."`
	Truncated    bool              `json:"truncated" jsonschema:"True when some series are left out."`
	Summarized   bool              `json:"summarized" jsonschema:"True when the series carry no raw points."`
	Hint         string            `json:"hint,omitempty" jsonschema:"How to narrow the query, when series are left out."`
	Note         string            `json:"note,omitempty" jsonschema:"Values returned as null, and a step the server raised so that the answer fits."`
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

// shapeSeries fits series, with the query, window, step and note (nil and empty for an instant query), to the
// response budget: the top limit series by maximum, raw points when they fit and summaries when they don't,
// and fewer series while the text and the JSON together exceed 16 KiB.
func shapeSeries(expr string, series []metricSeries, limit int, w *window, step, note string) metricsResult {
	head := metricsResult{Query: expr, Window: w, Step: step, Note: note}
	// The JSON escapes <, > and & to six bytes each, so a query echoed twice can be over the budget alone.
	if metricsSize(buildMetricsResult(head, nil, len(series), false)) > maxMetricsBytes {
		head.Query = echo(expr)
	}
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
	if n := labelsFit(kept); n < len(kept) {
		raw = false
		kept = kept[:n]
	}
	res := buildMetricsResult(head, kept, len(series), raw)
	size := metricsSize(res)
	if raw && size > maxMetricsBytes {
		raw = false
		res = buildMetricsResult(head, kept, len(series), raw)
		size = metricsSize(res)
	}
	if size <= maxMetricsBytes {
		return res
	}
	// Each series kept adds a row, so the size grows with the count, and a binary search builds the result log2(limit) times.
	fit := sort.Search(len(kept), func(n int) bool {
		return metricsSize(buildMetricsResult(head, kept[:n+1], len(series), raw)) > maxMetricsBytes
	})
	return buildMetricsResult(head, kept[:fit], len(series), raw)
}

// labelsFit returns how many of the series, from the first, can fit in maxMetricsBytes: a result shows each distinct
// label other than the internal ones at least once in its text and once in its JSON, which bounds its size from below.
func labelsFit(series []rankedSeries) int {
	seen := make(map[[2]string]bool)
	size := 0
	for i, s := range series {
		for k, v := range s.summary.Labels {
			if slices.Contains(internalLabels, k) || seen[[2]string{k, v}] {
				continue
			}
			seen[[2]string{k, v}] = true
			// Once in the text and once in the JSON.
			size += len(k) + len(v) + len(k) + len(v)
		}
		if size > maxMetricsBytes {
			return i
		}
	}
	return len(series)
}

// metricsSize is what a response costs: its text plus its structured content.
func metricsSize(r metricsResult) int {
	// A result holds no NaN or ±Inf and no unsupported type, so it always marshals.
	b, _ := json.Marshal(r) //nolint:errchkjson
	return len(r.render()) + len(b)
}

// rankSeries computes a series' statistics over the finite points.
func rankSeries(s metricSeries) rankedSeries {
	r := rankedSeries{summary: seriesSummary{Labels: s.Labels}, max: math.Inf(-1), points: make([]outPoint, len(s.Points))}
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
		if math.IsInf(avg, 0) {
			// The sum of finite values overflowed; dividing each value first keeps the average finite.
			avg = 0
			for _, p := range r.points {
				if p.V != nil {
					avg += *p.V / float64(finite)
				}
			}
		}
		// The mean lies between the extremes, which are finite; rounding of the sums can carry it past them.
		avg = min(max(avg, lo), hi)
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
func buildMetricsResult(head metricsResult, kept []rankedSeries, total int, raw bool) metricsResult {
	res := head
	res.Series = make([]seriesSummary, len(kept))
	res.TotalSeries = total
	res.Truncated = len(kept) < total
	res.Summarized = !raw

	labels := visibleLabels(kept)
	var shared map[string]string
	if len(kept) > 1 {
		shared = sharedLabels(labels)
	}
	if len(shared) > 0 {
		res.SharedLabels = shared
	}

	nonFinite := 0
	for i, s := range kept {
		row := s.summary
		row.Labels = make(map[string]string, len(labels[i]))
		for k, v := range labels[i] {
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
		res.Hint = leftOutHint(head.Query, total-len(kept), len(kept))
	}
	if nonFinite > 0 {
		nulls := fmt.Sprintf("%d non-finite values (NaN or infinite) are left out of min, avg and max; a last that is not finite is null.", nonFinite)
		if raw {
			nulls = fmt.Sprintf("%d values were NaN or infinite: they are null in points and left out of min, avg and max.", nonFinite)
		}
		res.Note = strings.TrimSpace(head.Note + " " + nulls)
	}
	return res
}

// visibleLabels returns the labels of each series without PMM's internal ones, except the internal ones whose
// values differ between series that the other labels do not tell apart.
func visibleLabels(series []rankedSeries) []map[string]string {
	out := make([]map[string]string, len(series))
	groups := make(map[string][]int)
	for i, s := range series {
		out[i] = make(map[string]string, len(s.summary.Labels))
		for k, v := range s.summary.Labels {
			if !slices.Contains(internalLabels, k) {
				out[i][k] = v
			}
		}
		key := renderLabels(out[i])
		groups[key] = append(groups[key], i)
	}

	var keep []string
	for _, group := range groups {
		first := series[group[0]].summary.Labels
		for _, k := range internalLabels {
			v, ok := first[k]
			differs := slices.ContainsFunc(group[1:], func(i int) bool {
				w, has := series[i].summary.Labels[k]
				return has != ok || w != v
			})
			if differs && !slices.Contains(keep, k) {
				keep = append(keep, k)
			}
		}
	}
	for i, s := range series {
		for _, k := range keep {
			v, ok := s.summary.Labels[k]
			if ok {
				out[i][k] = v
			}
		}
	}
	return out
}

// sharedLabels returns the labels that every label set has with the same value.
func sharedLabels(labels []map[string]string) map[string]string {
	shared := maps.Clone(labels[0])
	for _, l := range labels[1:] {
		for k, v := range shared {
			if got, ok := l[k]; !ok || got != v {
				delete(shared, k)
			}
		}
	}
	return shared
}

func leftOutHint(expr string, left, shown int) string {
	if shown == 0 {
		return fmt.Sprintf("%d series are left out: their labels are too large to show. "+
			"Narrow the query, or aggregate away the large labels with sum by (service_name) (<expr>).", left)
	}
	if len(expr) > maxHintExprBytes {
		expr = "<expr>"
	}
	return fmt.Sprintf("%d series are left out; %d are shown, the largest by maximum. "+
		"Narrow the selectors with a service_* or node_* matcher, keep the largest with topk(10, %[3]s), "+
		"or aggregate with sum by (service_name) (%[3]s).", left, shown, expr)
}
