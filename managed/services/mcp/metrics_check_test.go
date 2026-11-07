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
		{expr: `max_over_time(rate(mysql_global_status_questions[4d])[4d:1h])`, want: []string{"reads 8d", "reads 8d (range 0s + lookbehind 8d)"}},
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
			want: []string{"WITH templates"},
		},
		{expr: `sum without (job) (mysql_up{service_name="db1"})`, span: time.Hour},
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

	e, err := metricsql.Parse(`max_over_time(rate(a[1h] offset 30m)[1d:5m]) / on() b{x="1" or x="2"} + c[10m] offset -1h`)
	require.NoError(t, err)

	var got []string
	err = walkSelectors(e, 2*time.Minute, func(s selector) error {
		got = append(got, s.String()+" "+formatSpan(s.lookbehind))
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		`{__name__="a"} 1d1h30m`,
		`{__name__="b", x="1"} 2m`,
		`{__name__="b", x="2"} 2m`,
		`{__name__="c"} 1h10m`,
	}, got)
}
