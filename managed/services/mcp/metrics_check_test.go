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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/VictoriaMetrics/metricsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckExpr(t *testing.T) {
	t.Parallel()

	joined := func(n int, sel string) string {
		return strings.TrimSuffix(strings.Repeat(sel+" + ", n), " + ")
	}
	long := `mysql_up{service_name="` + strings.Repeat("a", 5000-len(`mysql_up{service_name=""}`)) + `"}`
	require.Len(t, long, 5000)

	const (
		scope  = "service_id, service_name, node_id or node_name"
		capped = "the cap is 7d"
	)
	for _, tc := range []struct {
		name string
		expr string
		span time.Duration
		// step is what the caller sends: zero stands for 5m on an instant query
		// and for the smallest step of at most 2,000 points on a range query.
		step time.Duration
		// want lists substrings of the invalid_input hint; nil means the expression passes.
		want []string
	}{
		{expr: `rate(mysql_global_status_questions[5m])`, span: 6 * day},
		{expr: `sum by (service_name) (rate(mysql_global_status_questions{service_name="db1"}[5m]))`, span: time.Hour},
		{expr: `{__name__=~"mysql_global_status_.*", service_name=~"db1|db2"}`, span: time.Hour},
		{expr: `{__name__=~"mysql_global_status_.*", service_name=~"db\\.prod"}`, span: time.Hour},
		{expr: `{__name__=~"mysql_global_status_.*", service_name=~"db.prod"}`, span: time.Hour, want: []string{scope}},
		{expr: `{__name__=~"mysql_global_status_.*", service_name=~".+"}`, span: time.Hour, want: []string{scope}},
		{expr: `rate(mysql_global_status_questions[7d])`},
		{
			expr: `rate(mysql_global_status_questions[5m])`, span: 7 * day,
			want: []string{"reads 7d5m", capped, "the query reads 7d5m (range 7d + lookbehind 5m); the cap is 7d"},
		},
		{expr: `{__name__=~".+"}`, span: time.Hour, want: []string{"metric name"}},
		{expr: `{job="mysql"}`, span: time.Hour, want: []string{"metric name"}},
		{expr: `{__name__!="up"}`, span: time.Hour, want: []string{"metric name"}},
		{name: "an = name is copied into both OR groups", expr: `{__name__="up" or service_name="db1"}`, span: time.Hour},
		{expr: `{__name__=~"mysql_global_status_.*", service_name="db1" or service_name="db2"}`, span: time.Hour, want: []string{"metric name"}},
		{expr: `{job="x" or job="y"}`, span: time.Hour, want: []string{"metric name"}},
		{expr: `mysql_up{service_name="db1" or service_name="db2"}`, span: time.Hour},
		{
			expr: `{__name__=~"mysql_global_status_.*", service_name="db1" or __name__=~"mysql_global_status_.*"}`, span: time.Hour,
			want: []string{scope},
		},
		{expr: `{__name__=~"mysql_.*"}`, span: time.Hour, want: []string{scope}},
		{expr: `{__name__=~"my.*", service_name="db1"}`, span: time.Hour, want: []string{"prefix"}},
		{expr: `rate(mysql_global_status_questions[365d])`, want: []string{capped}},
		{name: "rate without a window over 6d at step 2d", expr: `rate(mysql_global_status_questions)`, span: 6 * day, step: 2 * day, want: []string{capped}},
		{name: "a bare selector over 6d at step 2d", expr: `mysql_global_status_questions`, span: 6 * day, step: 2 * day, want: []string{capped}},
		{expr: `max_over_time(rate(up[5m])[30d:5m])`, want: []string{capped}},
		{expr: `max_over_time(rate(mysql_global_status_questions[4d])[4d:1h])`, want: []string{"reads 8d", "reads 8d1h5m (range 0s + lookbehind 8d1h5m)"}},
		{expr: `rate(mysql_global_status_questions[4d]) + rate(mysql_global_status_queries[4d])`},
		{expr: `rate(mysql_global_status_questions[4d] offset 4d)`, want: []string{"reads 8d", "reads 8d (range 0s + lookbehind 8d)"}},
		{expr: `up offset 8d`, want: []string{capped}},
		{expr: `up @ 1700000000`, want: []string{"@"}},
		{expr: `up @ start()`, want: []string{"@"}},
		{expr: `WITH (f(x) = rate(x[5m])) f(mysql_global_status_questions)`, span: time.Hour, want: []string{"WITH", "MetricsQL WITH templates are not accepted; write the expression out"}},
		{expr: `mysql_global_status_questions{service_name="with (db1)"}`, span: time.Hour},
		{expr: `max_over_time(rate(mysql_global_status_questions[5m])[6d:1s])`, want: []string{"subquery step", "4m20s"}},
		{expr: `max_over_time(rate(mysql_global_status_questions[5m])[6d:5m])`},
		{name: "11 selectors joined by +", expr: joined(11, `mysql_up{service_name="db1"}`), span: time.Hour, want: []string{"10 selectors"}},
		{
			name: "6 selectors of two OR groups each, joined by +",
			expr: joined(6, `mysql_up{service_name="db1" or service_name="db2"}`), span: time.Hour,
			want: []string{"10 selectors"},
		},
		{name: "5,000-byte expression", expr: long, span: time.Hour, want: []string{"4 KiB"}},
		{expr: `rate(`, span: time.Hour, want: []string{"parse"}},

		{name: "10 selectors joined by +", expr: joined(10, `mysql_up{service_name="db1"}`), span: time.Hour},
		{name: "5 selectors of two OR groups each", expr: joined(5, `mysql_up{service_name="db1" or service_name="db2"}`), span: time.Hour},
		{name: "{} parses to no OR group and is still a selector", expr: `{}`, span: time.Hour, want: []string{"metric name"}},
		{expr: `{__name__=""}`, span: time.Hour, want: []string{"metric name"}},
		{expr: `{__name__=~"mysql_global_status_.*", service_name=""}`, span: time.Hour, want: []string{scope}},
		{expr: `{__name__=~"mysql_global_status_.*", service_name=~"db1|"}`, span: time.Hour, want: []string{scope}},
		{expr: `{__name__=~"mysql_global_status_.*", service_name!="db1"}`, span: time.Hour, want: []string{scope}},
		{name: "a zero window reads one step back", expr: `rate(mysql_global_status_questions[0s])`, span: 7 * day, step: time.Hour, want: []string{"reads 7d1h"}},
		{name: "i units count the step", expr: `rate(mysql_global_status_questions[3000i])`, want: []string{"reads 10d10h"}},
		{name: "a negative offset counts its absolute value", expr: `up offset -8d`, want: []string{capped}},
		{name: "a subquery without a step takes the query's", expr: `max_over_time(rate(mysql_global_status_questions[5m])[6d:])`, span: time.Hour, want: []string{"subquery step"}},
		{expr: `max_over_time(rate(mysql_global_status_questions[5m])[6d:])`},
		{name: "a lowercase template", expr: `with (q = mysql_up) q`, span: time.Hour, want: []string{"WITH templates"}},
		{
			name: "a quote in a comment does not hide a template",
			expr: "# it's a template\nWITH # args\n(f(x) = rate(x[5m])) f(mysql_global_status_questions)", span: time.Hour,
			want: []string{"WITH templates"},
		},
		{
			name: "an escaped quote in a name opens no string",
			expr: `mysql\"up{service_name="db1"} + with (q = mysql_up) q # "`, span: time.Hour,
			want: []string{"backslash at byte 5"},
		},
		{expr: `sum without (job) (mysql_up{service_name="db1"})`, span: time.Hour},

		// A subquery reads at its own step, from its range + step + 5m back.
		{name: "100i inside a subquery is 100 subquery steps", expr: `max_over_time(rate(mysql_global_status_questions[100i])[1h:1d])`, want: []string{capped, "reads 101d1h5m"}},
		{name: "a bare selector inside a subquery reads one subquery step", expr: `max_over_time(mysql_global_status_questions[1h:30d])`, want: []string{capped, "reads 60d1h5m"}},
		{expr: `max_over_time(rate(mysql_global_status_questions[1i])[1h:365d])`, want: []string{capped, "reads 730d1h5m"}},
		{name: "a 7d subquery also reads its step and lead", expr: `max_over_time(mysql_global_status_questions[7d:5m])`, want: []string{capped, "reads 7d15m"}},
		{
			name: "a rollup of an aggregation is an implicit subquery",
			expr: `rate(sum(mysql_up{service_name="db1"}))`, span: 6 * day, step: 12 * time.Hour,
			want: []string{"reads 7d5m (range 6d + lookbehind 1d5m)"},
		},
		{name: "a window over an aggregation is an implicit subquery", expr: `max_over_time(sum(mysql_up{service_name="db1"})[7d])`, want: []string{capped, "reads 7d15m"}},

		// Nothing that metricsql.Parse would expand reaches it.
		{name: "a dotted capital I spells WITH", expr: `WİTH (q = mysql_up{service_name="db1"}) q`, span: time.Hour, want: []string{"non-ASCII character at byte 1"}},
		{expr: `ttf(ttf(ttf(mysql_up{service_name="db1"})))`, span: time.Hour, want: []string{"template ttf()"}},
		{
			name: "14 nested ttf calls",
			expr: strings.Repeat("ttf(", 14) + `mysql_up{service_name="db1"}` + strings.Repeat(")", 14), span: time.Hour,
			want: []string{"template ttf()"},
		},
		{expr: `ru(mysql_up{service_name="db1"}, mysql_up{service_name="db2"})`, span: time.Hour, want: []string{"template ru()"}},
		{expr: `range_median(mysql_up{service_name="db1"})`, span: time.Hour, want: []string{"template range_median()"}},
		{expr: `alias(mysql_up{service_name="db1"}, "up")`, span: time.Hour, want: []string{"template alias()"}},
		{name: "an escaped name spells ttf", expr: `t\x74f(mysql_up{service_name="db1"})`, span: time.Hour, want: []string{"backslash at byte 1"}},
		{expr: `mysql\_up{service_name="db1"}`, span: time.Hour, want: []string{"backslash at byte 5"}},
		{expr: `mysql_up{service_name="db1"} # café`, span: time.Hour, want: []string{"non-ASCII character at byte 34"}},
		{name: "non-ASCII inside a label value passes", expr: `mysql_up{service_name="dbé"}`, span: time.Hour},
		{name: "a backslash inside a label value passes", expr: `mysql_up{service_name="db\\1"}`, span: time.Hour},

		{expr: `{__name__=~"mysql_global_status_.*", service_name=~"(?i)db1"}`, span: time.Hour, want: []string{scope}},
		{name: "a span past the largest duration", expr: `rate(mysql_global_status_questions[99999999999y])`, want: []string{"reads more than 7d; the cap is 7d"}},
		{
			name: "the selector count comes before the selector rules",
			expr: `{job="x"} + ` + joined(10, `mysql_up{service_name="db1"}`), span: time.Hour,
			want: []string{"10 selectors"},
		},
	} {
		name := tc.name
		if name == "" {
			name = tc.expr
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			step := tc.step
			if step == 0 {
				step = instantStep
				if tc.span > 0 {
					step = max(time.Second, (tc.span/maxSubquerySteps + time.Second - 1).Truncate(time.Second))
				}
			}
			err := checkExpr(tc.expr, tc.span, step)
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			te, ok := errors.AsType[*toolError](err)
			require.True(t, ok, "want a *toolError, got %v", err)
			assert.Equal(t, codeInvalidInput, te.code)
			for _, w := range tc.want {
				assert.Contains(t, te.message, w)
			}
		})
	}
}

func TestWalkSelectors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		expr  string
		step  time.Duration
		want  []string
		steps []time.Duration
	}{
		{
			expr: `max_over_time(rate(a[1h] offset 30m)[1d:5m]) / on() b{x="1" or x="2"} + c[10m] offset -1h - rate(sum(d))`,
			step: 2 * time.Minute,
			want: []string{
				`{__name__="a"} 1d1h40m`,
				`{__name__="b", x="1"} 2m`,
				`{__name__="b", x="2"} 2m`,
				`{__name__="c"} 1h10m`,
				`{__name__="d"} 9m`,
			},
			steps: []time.Duration{5 * time.Minute, 2 * time.Minute},
		},
		{
			// [1h:] takes the enclosing subquery's step, not the query's.
			expr: `max_over_time(max_over_time(e[1h:])[1d:1h])`, step: 2 * time.Minute,
			want: []string{`{__name__="e"} 1d4h10m`}, steps: []time.Duration{time.Hour, time.Hour},
		},
		{
			expr: `max_over_time(rate(f[100i])[1h:1d])`, step: 5 * time.Minute,
			want: []string{`{__name__="f"} 101d1h5m`}, steps: []time.Duration{day},
		},
		{
			expr: `max_over_time(sum(g)[7d])`, step: 5 * time.Minute,
			want: []string{`{__name__="g"} 7d15m`}, steps: []time.Duration{5 * time.Minute},
		},
		{
			expr: `max_over_time(h[1h:1m] offset 1d)`, step: 5 * time.Minute,
			want: []string{`{__name__="h"} 1d1h7m`}, steps: []time.Duration{time.Minute},
		},
		{
			// Only quantile_over_time's second argument is read over the window.
			expr: `quantile_over_time(scalar(i), j[5m])`, step: time.Minute,
			want: []string{`{__name__="i"} 1m`, `{__name__="j"} 5m`},
		},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			t.Parallel()

			e, err := metricsql.Parse(tc.expr)
			require.NoError(t, err)
			w, err := walkSelectors(e, tc.step)
			require.NoError(t, err)

			got := make([]string, 0, len(w.selectors))
			for _, s := range w.selectors {
				got = append(got, s.String()+" "+formatSpan(s.lookbehind))
			}
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.steps, w.subquerySteps)
		})
	}
}

// opaqueExpr stands for a node type a later metricsql might add, which the walk cannot measure.
type opaqueExpr struct{}

func (opaqueExpr) AppendString(dst []byte) []byte {
	return append(dst, "opaque(x)"...)
}

func TestWalkSelectorsRefusesUnknownNodes(t *testing.T) {
	t.Parallel()

	for _, e := range []metricsql.Expr{
		opaqueExpr{},
		&metricsql.FuncExpr{Name: "abs", Args: []metricsql.Expr{opaqueExpr{}}},
		&metricsql.RollupExpr{Expr: opaqueExpr{}},
	} {
		_, err := walkSelectors(e, instantStep)
		te, ok := errors.AsType[*toolError](err)
		require.True(t, ok, "want a *toolError, got %v", err)
		assert.Equal(t, codeInvalidInput, te.code)
		assert.Contains(t, te.message, "cannot check 'opaque(x)'")
	}
}

func TestCheckExprHintSize(t *testing.T) {
	t.Parallel()

	fill := func(prefix, suffix string) string {
		return prefix + strings.Repeat("\x01", maxExprBytes-len(prefix)-len(suffix)) + suffix
	}
	for _, expr := range []string{
		fill("", ""),
		fill("rate(", ""),
		fill(`{job="`, `"}`),
		fill(`{__name__=~"m.`, `"}`),
		fill(`{__name__=~"mysql_`, `"}`),
	} {
		require.Len(t, expr, maxExprBytes)
		err := checkExpr(expr, time.Hour, 2*time.Second)
		te, ok := errors.AsType[*toolError](err)
		require.True(t, ok, "want a *toolError, got %v", err)
		assert.LessOrEqual(t, len(te.message), 512, te.message[:min(len(te.message), 100)])
	}
}
