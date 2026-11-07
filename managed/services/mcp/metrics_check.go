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
	"math"
	"regexp"
	"regexp/syntax"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/VictoriaMetrics/metricsql"
)

// The execution budget of a model-written MetricsQL expression.
const (
	maxExprBytes  = 4096
	maxSelectors  = 10
	maxSpan       = 7 * day
	minNamePrefix = 3
)

// maxSubquerySteps bounds the steps a subquery takes over the span the query reads.
const maxSubquerySteps = 2000

// maxImplicitPoints bounds the points per series of a subquery VictoriaMetrics runs, at the step
// in effect, for a rollup over anything but a selector.
const maxImplicitPoints = 4000

// instantStep is the step an instant query is checked with: VictoriaMetrics' default.
const instantStep = 5 * time.Minute

// subqueryLeadSteps is how many of its own steps a subquery starts early: one, and one more as its start aligns down to the step.
const subqueryLeadSteps = 2

// subqueryLead is how much further back than its range and step VictoriaMetrics starts a subquery: its default maxSilenceInterval.
const subqueryLead = 5 * time.Minute

// maxEchoBytes bounds the part of the caller's input that a hint quotes back.
const maxEchoBytes = 200

// scopeLabels are the labels whose matcher narrows a regex-named selector to some services or nodes.
var scopeLabels = []string{"service_id", "service_name", "node_id", "node_name"}

// templateKeywords are the words before "(" that make metricsql.Parse expand a template: WITH, and the
// built-in templates, which repeat their arguments.
var templateKeywords = []string{"with", "ttf", "ru", "range_median", "alias"}

// checkExpr refuses, before any request, an expression that would read more than 7 days
// or select too broadly; an instant query has span 0, and a step of 0 or less is instantStep.
func checkExpr(expr string, span, step time.Duration) error {
	if step <= 0 {
		step = instantStep
	}
	if len(expr) > maxExprBytes {
		return newToolError(codeInvalidInput, "the expression is %d bytes; the cap is 4 KiB (%d bytes): query fewer metrics per call", len(expr), maxExprBytes)
	}
	err := checkBeforeParse(expr)
	if err != nil {
		return err
	}
	e, err := metricsql.Parse(expr)
	if err != nil {
		return newToolError(codeInvalidInput, "cannot parse the expression: %s", echo(err.Error()))
	}

	w, err := walkSelectors(e, step)
	if err != nil {
		return err
	}
	if len(w.selectors) > maxSelectors {
		return newToolError(codeInvalidInput,
			"the expression has %d selectors, each OR group inside braces counting as one; the cap is %d selectors: query fewer metrics per call",
			len(w.selectors), maxSelectors)
	}
	var lookbehind time.Duration
	for _, s := range w.selectors {
		err = checkSelector(s)
		if err != nil {
			return err
		}
		lookbehind = max(lookbehind, s.lookbehind)
	}

	read := addCapped(span, lookbehind)
	if read == math.MaxInt64 {
		return newToolError(codeInvalidInput,
			"the query reads more than %[1]s; the cap is %[1]s: shorten the range, or the rollup windows, subquery ranges, subquery steps and offsets",
			formatSpan(maxSpan))
	}
	if read > maxSpan {
		return newToolError(codeInvalidInput,
			"the query reads %s (range %s + lookbehind %s); the cap is %s: shorten the range, or the rollup windows, subquery ranges, subquery steps and offsets",
			formatSpan(read), formatSpan(span), formatSpan(lookbehind), formatSpan(maxSpan))
	}

	minStep := read / maxSubquerySteps
	for _, s := range w.subquerySteps {
		if s < minStep {
			return newToolError(codeInvalidInput,
				"the subquery step %s is too small: the query reads %s, so a subquery step must be at least %s (the span read ÷ %d)",
				formatSpan(s), formatSpan(read), formatSpan(ceilSecond(minStep)), maxSubquerySteps)
		}
	}

	for _, sq := range w.implicitSubqueries {
		evaluated := addCapped(span, sq.reach)
		points := float64(evaluated) / float64(sq.step)
		if points > maxImplicitPoints {
			// The lead steps in reach grow with the step, so they come out before dividing.
			minStep := (evaluated - subqueryLeadSteps*sq.step) / (maxImplicitPoints - subqueryLeadSteps)
			return newToolError(codeInvalidInput,
				"at the query's step of %s, a rollup over an expression that is not a selector runs as a subquery of %d points per series; "+
					"the cap is %d: raise step to at least %s, or shorten that rollup's window",
				formatSpan(sq.step), int64(math.Ceil(points)), maxImplicitPoints, formatSpan(ceilSecond(minStep)))
		}
	}
	return nil
}

// selector is one OR group of a MetricExpr's label filters, which VictoriaMetrics
// matches on its own, so each is checked and counted on its own.
type selector struct {
	filters []metricsql.LabelFilter
	// lookbehind is how far back from each point of the query the selector's path reads.
	lookbehind time.Duration
}

// String returns the selector in MetricsQL form.
func (s selector) String() string {
	b := []byte{'{'}
	for i, f := range s.filters {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = f.AppendString(b)
	}
	return string(append(b, '}'))
}

// exprWalk is what a parsed expression reads, path by path, as VictoriaMetrics evaluates it.
type exprWalk struct {
	selectors []selector
	// subquerySteps holds the step of each subquery written as [range:step] or [range:], in the order met.
	subquerySteps []time.Duration
	// implicitSubqueries holds each subquery VictoriaMetrics runs for a rollup over anything but a selector.
	implicitSubqueries []implicitSubquery
}

// implicitSubquery is a subquery no step was written for, run at the step in effect where it stands.
type implicitSubquery struct {
	step time.Duration
	// reach is how much longer than the query's range its inner expression is evaluated over.
	reach time.Duration
}

// path is what the walk carries down one branch of the expression.
type path struct {
	// step is the step the branch is evaluated at.
	step time.Duration
	// lookbehind is how far back from each point of the query the branch has read so far.
	lookbehind time.Duration
	// extent is how much longer than the query's range the branch is evaluated over; offsets shift it, not lengthen it.
	extent time.Duration
}

// walkSelectors walks e evaluated at step, and refuses `@` and any node it cannot measure.
func walkSelectors(e metricsql.Expr, step time.Duration) (exprWalk, error) {
	var w exprWalk
	err := w.expr(e, path{step: step})
	return w, err
}

// expr walks e along p.
func (w *exprWalk) expr(e metricsql.Expr, p path) error {
	switch e := e.(type) {
	case *metricsql.MetricExpr:
		// A selector without a window reads one step back.
		w.metric(e, addCapped(p.lookbehind, p.step))
		return nil

	case *metricsql.RollupExpr:
		return w.rollup(e, p)

	case *metricsql.FuncExpr:
		rollupArg := metricsql.GetRollupArgIdx(e)
		for i, arg := range e.Args {
			walk := w.expr
			if i == rollupArg {
				walk = w.rollupArg
			}
			err := walk(arg, p)
			if err != nil {
				return err
			}
		}
		return nil

	case *metricsql.AggrFuncExpr:
		for _, arg := range e.Args {
			err := w.expr(arg, p)
			if err != nil {
				return err
			}
		}
		return nil

	case *metricsql.BinaryOpExpr:
		err := w.expr(e.Left, p)
		if err != nil {
			return err
		}
		return w.expr(e.Right, p)

	case *metricsql.NumberExpr, *metricsql.StringExpr, *metricsql.DurationExpr:
		return nil

	default:
		return newToolError(codeInvalidInput,
			"cannot check '%s'; write the expression with selectors, functions, aggregations and operators", echo(string(e.AppendString(nil))))
	}
}

// rollupArg walks what a rollup function reads over its window, which VictoriaMetrics
// evaluates as a subquery at p.step unless it is a selector or a rollup.
func (w *exprWalk) rollupArg(e metricsql.Expr, p path) error {
	switch e.(type) {
	case *metricsql.MetricExpr, *metricsql.RollupExpr:
		return w.expr(e, p)
	default:
		return w.subquery(e, 0, p.step, false, p)
	}
}

// rollup walks re: a selector read over a window and offset, or else a subquery, explicit or not.
func (w *exprWalk) rollup(re *metricsql.RollupExpr, p path) error {
	if re.At != nil {
		return newToolError(codeInvalidInput,
			"the @ modifier is not accepted: it reads outside the query's time range; set the query's time or period instead")
	}

	me, ok := re.Expr.(*metricsql.MetricExpr)
	if ok && !re.ForSubquery() {
		lookbehind := addCapped(p.lookbehind, durationOf(re.Offset, p.step))
		// VictoriaMetrics reads at least one step back, whatever the window.
		w.metric(me, addCapped(lookbehind, max(durationOf(re.Window, p.step), p.step)))
		return nil
	}

	inner := durationOf(re.Step, p.step)
	if inner == 0 {
		inner = p.step
	}
	// A nested subquery resolves i in its window and offset at the enclosing step, while an
	// instant query's root subquery runs as a range query at its own step; the larger covers both.
	units := max(p.step, inner)
	p.lookbehind = addCapped(p.lookbehind, durationOf(re.Offset, units))
	return w.subquery(re.Expr, durationOf(re.Window, units), inner, re.ForSubquery(), p)
}

// subquery walks e evaluated at its own step from window + 2 steps + subqueryLead before each outer
// point: VictoriaMetrics starts it one step and subqueryLead early, then aligns that start down to the step.
func (w *exprWalk) subquery(e metricsql.Expr, window, step time.Duration, explicit bool, p path) error {
	lead := addCapped(window, addCapped(addCapped(step, step), subqueryLead))
	inner := path{step: step, lookbehind: addCapped(p.lookbehind, lead), extent: addCapped(p.extent, lead)}
	if explicit {
		w.subquerySteps = append(w.subquerySteps, step)
	} else {
		w.implicitSubqueries = append(w.implicitSubqueries, implicitSubquery{step: step, reach: inner.extent})
	}
	return w.expr(e, inner)
}

func (w *exprWalk) metric(me *metricsql.MetricExpr, lookbehind time.Duration) {
	groups := me.LabelFilterss
	if len(groups) == 0 {
		// `{}` parses to no group, yet selects every series.
		groups = [][]metricsql.LabelFilter{nil}
	}
	for _, filters := range groups {
		w.selectors = append(w.selectors, selector{filters: filters, lookbehind: lookbehind})
	}
}

// checkSelector refuses a selector that names no metric, or matches its name
// by regex without narrowing it to some services or nodes.
func checkSelector(s selector) error {
	var exact bool
	var prefix string
	for _, f := range s.filters {
		if f.Label != "__name__" || f.IsNegative || f.Value == "" {
			continue
		}
		if !f.IsRegexp {
			exact = true
			continue
		}
		re, err := regexp.Compile(f.Value)
		if err != nil {
			continue
		}
		p, _ := re.LiteralPrefix()
		if len(p) > len(prefix) {
			prefix = p
		}
	}

	switch {
	case exact:
		return nil
	case prefix == "":
		return newToolError(codeInvalidInput,
			"selector %s has no metric name: match __name__ with = (or write the name before the braces, as in mysql_up{service_name=\"db1\"}), "+
				"or with =~ and a regex that starts with at least %d literal bytes, as in __name__=~\"mysql_global_status_.*\"; pmm_list_metrics lists the names",
			echo(s.String()), minNamePrefix)
	case len(prefix) < minNamePrefix:
		return newToolError(codeInvalidInput,
			"selector %s matches names by a regex whose literal prefix '%s' is %d bytes; it needs at least %d, as in __name__=~\"mysql_global_status_.*\"",
			echo(s.String()), prefix, len(prefix), minNamePrefix)
	case !hasScope(s.filters):
		return newToolError(codeInvalidInput,
			"selector %s matches its name by regex, so it also needs a service_id, service_name, node_id or node_name matcher, "+
				"with = or with =~ and literal alternatives such as \"db1|db2\"",
			echo(s.String()))
	default:
		return nil
	}
}

// hasScope reports whether a filter narrows the selector to a known set of services or nodes.
func hasScope(filters []metricsql.LabelFilter) bool {
	for _, f := range filters {
		if f.IsNegative || f.Value == "" {
			continue
		}
		if !slices.Contains(scopeLabels, f.Label) {
			continue
		}
		if !f.IsRegexp || literalAlternatives(f.Value) {
			return true
		}
	}
	return false
}

// literalAlternatives reports whether every |-separated alternative of re is a single case-sensitive literal.
func literalAlternatives(re string) bool {
	for alt := range strings.SplitSeq(re, "|") {
		parsed, err := syntax.Parse(alt, syntax.Perl)
		if err != nil || parsed.Op != syntax.OpLiteral || parsed.Flags&syntax.FoldCase != 0 {
			return false
		}
	}
	return true
}

// checkBeforeParse refuses what metricsql.Parse would expand before any check sees it: a template,
// and a non-ASCII byte or a backslash outside quoted strings, which can spell one.
func checkBeforeParse(expr string) error {
	comment := false
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		switch {
		case c >= utf8.RuneSelf:
			return newToolError(codeInvalidInput,
				"the expression has a non-ASCII character at byte %d, outside a quoted string; only label values and other strings may hold one", i)
		case c == '\\':
			return newToolError(codeInvalidInput,
				"the expression has a backslash at byte %d, outside a quoted string; write names without escapes", i)
		case comment:
			comment = c != '\n'
		case c == '#':
			comment = true
		case c == '"' || c == '\'' || c == '`':
			i = closingQuote(expr, i)
		default:
			err := checkTemplateAt(expr, i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// checkTemplateAt refuses a template keyword at i followed by "(", whatever precedes it.
func checkTemplateAt(expr string, i int) error {
	for _, keyword := range templateKeywords {
		end := i + len(keyword)
		if end > len(expr) || !strings.EqualFold(expr[i:end], keyword) || nextByte(expr, end) != '(' {
			continue
		}
		if keyword == "with" {
			return newToolError(codeInvalidInput, "MetricsQL WITH templates are not accepted; write the expression out")
		}
		return newToolError(codeInvalidInput,
			"the MetricsQL template %s() is not accepted, since it repeats its arguments; write out the expression it stands for", expr[i:end])
	}
	return nil
}

// closingQuote returns the index of the quote that closes the string opening
// at start: the next one not escaped by an odd number of backslashes.
func closingQuote(s string, start int) int {
	quote := s[start]
	for i := start + 1; i < len(s); i++ {
		if s[i] != quote {
			continue
		}
		backslashes := 0
		for s[i-backslashes-1] == '\\' {
			backslashes++
		}
		if backslashes%2 == 0 {
			return i
		}
	}
	return len(s)
}

func endOfComment(s string, start int) int {
	n := strings.IndexByte(s[start:], '\n')
	if n < 0 {
		return len(s)
	}
	return start + n
}

// nextByte returns the first byte from i on that is neither space nor part of a comment, 0 at the end.
func nextByte(s string, i int) byte {
	for i < len(s) {
		switch s[i] {
		case ' ', '\t', '\n', '\v', '\f', '\r':
			i++
		case '#':
			i = endOfComment(s, i)
		default:
			return s[i]
		}
	}
	return 0
}

// echo returns s cut to maxEchoBytes on a rune boundary, with an ellipsis when cut.
func echo(s string) string {
	if len(s) <= maxEchoBytes {
		return s
	}
	n := maxEchoBytes
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + "…"
}

// durationOf returns the absolute value of d, the i unit being step, capped at the largest time.Duration.
func durationOf(d *metricsql.DurationExpr, step time.Duration) time.Duration {
	ms := d.Duration(step.Milliseconds())
	maxMs := int64(math.MaxInt64 / time.Millisecond)
	if ms > maxMs || ms < -maxMs {
		return math.MaxInt64
	}
	return time.Duration(max(ms, -ms)) * time.Millisecond
}

// ceilSecond rounds d up to a whole second.
func ceilSecond(d time.Duration) time.Duration {
	return (d + time.Second - 1).Truncate(time.Second)
}

// addCapped adds two non-negative durations, capped at the largest time.Duration.
func addCapped(a, b time.Duration) time.Duration {
	if a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

// formatSpan writes d as days plus the rest, as MetricsQL durations read: 7d5m, 8d, 1h30m.
func formatSpan(d time.Duration) string {
	days, rest := d/day, d%day
	var s string
	if days > 0 {
		s = strconv.FormatInt(int64(days), 10) + "d"
		if rest == 0 {
			return s
		}
	}
	r := rest.String()
	if strings.HasSuffix(r, "m0s") {
		r = strings.TrimSuffix(r, "0s")
	}
	if strings.HasSuffix(r, "h0m") {
		r = strings.TrimSuffix(r, "0m")
	}
	return s + r
}
