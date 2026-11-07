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

// instantStep is the step an instant query is checked with: VictoriaMetrics' default.
const instantStep = 5 * time.Minute

// scopeLabels are the labels whose matcher narrows a regex-named selector to some services or nodes.
var scopeLabels = []string{"service_id", "service_name", "node_id", "node_name"}

// checkExpr refuses, before any request, an expression that would read more
// than 7 days or select too broadly; an instant query has span 0 and step instantStep.
func checkExpr(expr string, span, step time.Duration) error {
	if len(expr) > maxExprBytes {
		return newToolError(codeInvalidInput, "the expression is %d bytes; the cap is 4 KiB (%d bytes): query fewer metrics per call", len(expr), maxExprBytes)
	}
	if hasWithTemplate(expr) {
		return newToolError(codeInvalidInput, "MetricsQL WITH templates are not accepted; write the expression out")
	}
	e, err := metricsql.Parse(expr)
	if err != nil {
		return newToolError(codeInvalidInput, "cannot parse the expression: %s", err)
	}

	var selectors int
	var lookbehind time.Duration
	err = walkSelectors(e, step, func(s selector) error {
		selectors++
		lookbehind = max(lookbehind, s.lookbehind)
		return checkSelector(s)
	})
	if err != nil {
		return err
	}
	if selectors > maxSelectors {
		return newToolError(codeInvalidInput,
			"the expression has %d selectors, each OR group inside braces counting as one; the cap is %d selectors: query fewer metrics per call",
			selectors, maxSelectors)
	}

	read := addCapped(span, lookbehind)
	if read > maxSpan {
		return newToolError(codeInvalidInput,
			"the query reads %s (range %s + lookbehind %s); the cap is %s: shorten the range, or the rollup windows, subquery ranges and offsets",
			formatSpan(read), formatSpan(span), formatSpan(lookbehind), formatSpan(maxSpan))
	}
	return checkSubquerySteps(e, step, read)
}

// selector is one OR group of a MetricExpr's label filters, which VictoriaMetrics
// matches on its own, so each is checked and counted on its own.
type selector struct {
	filters []metricsql.LabelFilter
	// lookbehind is how far back from each evaluated point the selector's path reads.
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

// walkSelectors calls visit for each selector of e with its path's lookbehind, the sum of the
// rollup windows, subquery ranges and offsets on it plus one step when none is a window, and refuses `@`.
func walkSelectors(e metricsql.Expr, step time.Duration, visit func(selector) error) error {
	return walkPath(e, step, 0, false, visit)
}

func walkPath(e metricsql.Expr, step, lookbehind time.Duration, windowed bool, visit func(selector) error) error {
	switch e := e.(type) {
	case *metricsql.MetricExpr:
		if !windowed {
			lookbehind = addCapped(lookbehind, step)
		}
		groups := e.LabelFilterss
		if len(groups) == 0 {
			// `{}` parses to no group, yet selects every series.
			groups = [][]metricsql.LabelFilter{nil}
		}
		for _, filters := range groups {
			err := visit(selector{filters: filters, lookbehind: lookbehind})
			if err != nil {
				return err
			}
		}
		return nil

	case *metricsql.RollupExpr:
		if e.At != nil {
			return newToolError(codeInvalidInput,
				"the @ modifier is not accepted: it reads outside the query's time range; set the query's time or period instead")
		}
		window := durationOf(e.Window, step)
		lookbehind = addCapped(lookbehind, addCapped(window, durationOf(e.Offset, step)))
		return walkPath(e.Expr, step, lookbehind, windowed || window > 0, visit)

	case *metricsql.FuncExpr:
		return walkArgs(e.Args, step, lookbehind, windowed, visit)

	case *metricsql.AggrFuncExpr:
		return walkArgs(e.Args, step, lookbehind, windowed, visit)

	case *metricsql.BinaryOpExpr:
		err := walkPath(e.Left, step, lookbehind, windowed, visit)
		if err != nil {
			return err
		}
		return walkPath(e.Right, step, lookbehind, windowed, visit)

	case *metricsql.NumberExpr, *metricsql.StringExpr, *metricsql.DurationExpr:
		return nil

	default:
		return newToolError(codeInvalidInput,
			"cannot check '%s'; write the expression with selectors, functions, aggregations and operators", e.AppendString(nil))
	}
}

func walkArgs(args []metricsql.Expr, step, lookbehind time.Duration, windowed bool, visit func(selector) error) error {
	for _, arg := range args {
		err := walkPath(arg, step, lookbehind, windowed, visit)
		if err != nil {
			return err
		}
	}
	return nil
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
			s, minNamePrefix)
	case len(prefix) < minNamePrefix:
		return newToolError(codeInvalidInput,
			"selector %s matches names by a regex whose literal prefix '%s' is %d bytes; it needs at least %d, as in __name__=~\"mysql_global_status_.*\"",
			s, prefix, len(prefix), minNamePrefix)
	case !hasScope(s.filters):
		return newToolError(codeInvalidInput,
			"selector %s matches its name by regex, so it also needs a service_id, service_name, node_id or node_name matcher, "+
				"with = or with =~ and literal alternatives such as \"db1|db2\"",
			s)
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

// literalAlternatives reports whether every |-separated alternative of re is a single literal.
func literalAlternatives(re string) bool {
	for alt := range strings.SplitSeq(re, "|") {
		parsed, err := syntax.Parse(alt, syntax.Perl)
		if err != nil || parsed.Op != syntax.OpLiteral {
			return false
		}
	}
	return true
}

// checkSubquerySteps refuses a subquery whose step is below the span read ÷ maxSubquerySteps.
func checkSubquerySteps(e metricsql.Expr, step, read time.Duration) error {
	minStep := read / maxSubquerySteps
	var err error
	metricsql.VisitAll(e, func(node metricsql.Expr) {
		re, ok := node.(*metricsql.RollupExpr)
		if !ok || !re.ForSubquery() || err != nil {
			return
		}
		s := step
		if re.Step != nil {
			s = durationOf(re.Step, step)
		}
		if s < minStep {
			err = newToolError(codeInvalidInput,
				"the subquery step %s is too small: the query reads %s, so a subquery step must be at least %s (the span read ÷ %d)",
				formatSpan(s), formatSpan(read), formatSpan((minStep + time.Second - 1).Truncate(time.Second)), maxSubquerySteps)
		}
	})
	return err
}

// hasWithTemplate reports whether expr holds the keyword WITH followed by "(" outside strings and
// comments, skipped as the metricsql lexer does, since metricsql.Parse expands a template before any check.
func hasWithTemplate(expr string) bool {
	const keyword = "with"

	for i := 0; i < len(expr); i++ {
		switch expr[i] {
		case '"', '\'', '`':
			i = closingQuote(expr, i)
		case '#':
			i = endOfComment(expr, i)
		case '\\':
			// An escaped character belongs to an identifier: it opens no string or comment.
			i++
		default:
			end := i + len(keyword)
			if end <= len(expr) && strings.EqualFold(expr[i:end], keyword) && nextByte(expr, end) == '(' {
				return true
			}
		}
	}
	return false
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

// durationOf returns the absolute value of d, the i unit being step, capped at the largest time.Duration.
func durationOf(d *metricsql.DurationExpr, step time.Duration) time.Duration {
	ms := d.Duration(step.Milliseconds())
	maxMs := int64(math.MaxInt64 / time.Millisecond)
	if ms > maxMs || ms < -maxMs {
		return math.MaxInt64
	}
	return time.Duration(max(ms, -ms)) * time.Millisecond
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
