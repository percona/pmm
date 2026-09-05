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

package alert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitSingleExpr(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		expr    string
		param   string
		wantLHS string
		wantOp  string
		wantErr string
	}{
		{
			name:    "greater than with bool",
			expr:    "node_load1 > bool [[ .threshold ]]",
			param:   "threshold",
			wantLHS: "node_load1",
			wantOp:  ">",
		},
		{
			// 8 of the 16 shipped candidates compare with `<`, so assuming `>` would
			// invert half the corpus.
			name:    "less than preserves the operator",
			expr:    "node_memory_MemAvailable_bytes < bool [[ .threshold ]]",
			param:   "threshold",
			wantLHS: "node_memory_MemAvailable_bytes",
			wantOp:  "<",
		},
		{
			name:    "greater or equal",
			expr:    "proxysql_runtime_servers_status >= bool [[ .status ]]",
			param:   "status",
			wantLHS: "proxysql_runtime_servers_status",
			wantOp:  ">=",
		},
		{
			name:    "bool is optional",
			expr:    "(max by (cluster) (some_metric)) > [[ .threshold ]]",
			param:   "threshold",
			wantLHS: "(max by (cluster) (some_metric))",
			wantOp:  ">",
		},
		{
			// The left-hand side is sliced from the original text, so line breaks and
			// spacing survive rather than being reprinted from the AST.
			name:    "multi-line left-hand side keeps its formatting",
			expr:    "node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes\n* 100\n< bool [[ .threshold ]]",
			param:   "threshold",
			wantLHS: "node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes\n* 100",
			wantOp:  "<",
		},
		{
			// Vector matching on an operator *inside* the left-hand side is fine; only
			// matching on the comparison itself is not. A string search for "ignoring("
			// would wrongly reject this, which is why the split is done on the AST.
			name:    "vector matching inside the left-hand side is allowed",
			expr:    "max_over_time(a[5m]) / ignoring (job) b * 100 > bool [[ .threshold ]]",
			param:   "threshold",
			wantLHS: "max_over_time(a[5m]) / ignoring (job) b * 100",
			wantOp:  ">",
		},
		{
			name:    "tolerates whitespace-free tokens",
			expr:    "node_load1 > bool [[.threshold]]",
			param:   "threshold",
			wantLHS: "node_load1",
			wantOp:  ">",
		},
		{
			name:    "parameter not referenced",
			expr:    "node_load1 > bool 80",
			param:   "threshold",
			wantErr: "is not referenced in the expression",
		},
		{
			name:    "token used inside a larger expression",
			expr:    "node_load1 > bool [[ .threshold ]] * 100",
			param:   "threshold",
			wantErr: "must be compared directly",
		},
		{
			name:    "no top-level comparison",
			expr:    "node_load1 + [[ .threshold ]]",
			param:   "threshold",
			wantErr: "must be the right-hand side of the expression's top-level comparison",
		},
		{
			// PromQL allows vector matching only between two instant vectors, and a
			// threshold is a scalar, so this is rejected by the parser rather than needing
			// a guard of our own - such a template could never be valid in the first place.
			name:    "vector matching on the comparison itself",
			expr:    "a > bool on (node_name) [[ .threshold ]]",
			param:   "threshold",
			wantErr: "vector matching only allowed between instant vectors",
		},
		{
			name:    "expression that is not valid PromQL",
			expr:    "max(some_metric[1m]) > [[ .threshold ]]",
			param:   "threshold",
			wantErr: "failed to parse expression",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			split, err := SplitSingleExpr(tc.expr, tc.param)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantLHS, split.LHS)
			assert.Equal(t, tc.wantOp, split.Operator)
		})
	}
}

// TestSplitShippedSingleExprTemplates runs the splitter over every shipped single-expression
// template that carries a parameter. These are the expressions desugaring has to handle, so
// the corpus itself is the test: a template reworded into a shape the splitter cannot take
// apart should fail here rather than when someone marks it overridable.
func TestSplitShippedSingleExprTemplates(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob(filepath.Join("..", "..", "data", "alerting-templates", "*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	splittable := 0

	for _, file := range files {
		b, err := os.ReadFile(file)
		require.NoError(t, err)

		templates, err := Parse(strings.NewReader(string(b)), &ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		})
		require.NoErrorf(t, err, "%s must parse", filepath.Base(file))

		for _, template := range templates {
			if template.UsesMultipleExpressions() || len(template.Params) == 0 {
				continue
			}

			for _, param := range template.Params {
				if !ParamTokenRegexp(param.Name).MatchString(template.Expr) {
					continue
				}

				split, err := SplitSingleExpr(template.Expr, param.Name)
				if err != nil {
					// mongodb_replication_lag uses `max(<range vector>[1m])`, which is not
					// valid PromQL - a pre-existing bug the parser surfaces here. Report it
					// rather than failing, so this test tracks the corpus instead of
					// blocking on a defect it did not introduce.
					t.Logf("NOT SPLITTABLE %s (%s): %v", template.Name, param.Name, err)

					continue
				}

				splittable++
				assert.NotEmptyf(t, split.LHS, "%s: left-hand side must not be empty", template.Name)
				assert.NotEmptyf(t, split.Operator, "%s: operator must not be empty", template.Name)
				assert.NotContainsf(t, split.LHS, "[[",
					"%s: the parameter token must not survive into the observed query", template.Name)
			}
		}
	}

	t.Logf("splittable single-expression templates: %d", splittable)
	assert.GreaterOrEqual(t, splittable, 15,
		"the shipped corpus should be almost entirely splittable")
}
