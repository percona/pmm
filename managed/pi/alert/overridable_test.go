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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/pi/common"
)

const (
	mysqlTooManyConnectionsExpr = `max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
mysql_global_variables_max_connections
* 100
> bool [[ .threshold ]]`

	mongodbPbmBackupStaleExpr = `(
  time() - max by (cluster) (
    max_over_time(
      mongodb_pbm_backup_last_transition_ts{status="done"}[30d]
    )
  )
) > [[ .threshold ]]`
)

func overridableTemplate(param Parameter, expression string) Template {
	return Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Queries: []TemplateQuery{
			{RefID: "A", Expr: "cpu"},
		},
		Expressions: []TemplateExpression{
			{RefID: "C", Type: "math", Expression: expression},
		},
		Condition: "C",
		Params:    []Parameter{param},
	}
}

func singleExprOverridableTemplate(expr string, params ...Parameter) Template {
	return Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Expr:     expr,
		Params:   params,
	}
}

func TestValidateOverridableParamValid(t *testing.T) {
	t.Parallel()

	template := overridableTemplate(
		Parameter{Name: "threshold", Summary: "s", Type: Float, Value: 80, Overridable: true},
		"$A > [[ .threshold ]]",
	)

	require.NoError(t, template.Validate())
}

func TestValidateOverridableParamRejectsNonFloat(t *testing.T) {
	t.Parallel()

	template := overridableTemplate(
		Parameter{Name: "threshold", Summary: "s", Type: String, Value: "x", Overridable: true},
		"$A > [[ .threshold ]]",
	)

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a float")
}

func TestValidateOverridableParamSingleExpressionValid(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		expr string
	}{
		{name: "gt bool", expr: "a > bool [[ .threshold ]]"},
		{name: "lt bool", expr: "a < bool [[ .threshold ]]"},
		{name: "gte bool", expr: "a >= bool [[ .threshold ]]"},
		{name: "lte bool", expr: "a <= bool [[ .threshold ]]"},
		{name: "eq bool", expr: "a == bool [[ .threshold ]]"},
		{name: "ne bool", expr: "a != bool [[ .threshold ]]"},
		{name: "gt no bool", expr: "a > [[ .threshold ]]"},
		{name: "multiline", expr: "a\n> bool [[ .threshold ]]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			template := singleExprOverridableTemplate(tc.expr,
				Parameter{Name: "threshold", Summary: "s", Type: Float, Value: 80, Overridable: true},
			)
			require.NoError(t, template.Validate())
		})
	}
}

func TestValidateOverridableParamSingleExpressionRejects(t *testing.T) {
	t.Parallel()

	param := Parameter{Name: "threshold", Summary: "s", Type: Float, Value: 80, Overridable: true}

	for _, tc := range []struct {
		name    string
		tmpl    Template
		wantErr string
	}{
		{
			name: "token not present",
			tmpl: singleExprOverridableTemplate("a > 80", param),
			wantErr: "overridable parameter 'threshold' is not referenced as a [[ .threshold ]] token in the expression",
		},
		{
			name: "token present twice",
			tmpl: singleExprOverridableTemplate("a > [[ .threshold ]] or b > [[ .threshold ]]", param),
			wantErr: "overridable parameter 'threshold' is referenced 2 times in the expression; it must be referenced exactly once",
		},
		{
			name: "token not at end",
			tmpl: singleExprOverridableTemplate("a > [[ .threshold ]] * 1024", param),
			wantErr: "overridable parameter 'threshold' must be the right-hand side of the expression's final comparison",
		},
		{
			name: "token followed by and",
			tmpl: singleExprOverridableTemplate("a > [[ .threshold ]] and up == 1", param),
			wantErr: "overridable parameter 'threshold' must be the right-hand side of the expression's final comparison",
		},
		{
			name: "token on left",
			tmpl: singleExprOverridableTemplate("[[ .threshold ]] > x", param),
			wantErr: "overridable parameter 'threshold' must be the right-hand side of the expression's final comparison",
		},
		{
			name: "multiple overridable params",
			tmpl: singleExprOverridableTemplate("a > [[ .threshold ]]",
				Parameter{Name: "threshold", Summary: "s", Type: Float, Value: 80, Overridable: true},
				Parameter{Name: "other", Summary: "s", Type: Float, Value: 90, Overridable: true},
			),
			wantErr: "single-expression templates support at most one overridable parameter, got 2",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.tmpl.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestValidateOverridableParamRejectsUnreferenced(t *testing.T) {
	t.Parallel()

	template := overridableTemplate(
		Parameter{Name: "threshold", Summary: "s", Type: Float, Value: 80, Overridable: true},
		"$A > 0",
	)

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not referenced")
}

func TestFindSingleExprThresholdRef(t *testing.T) {
	t.Parallel()

	t.Run("mysql_too_many_connections", func(t *testing.T) {
		t.Parallel()

		ref, err := FindSingleExprThresholdRef(mysqlTooManyConnectionsExpr, "threshold")
		require.NoError(t, err)
		assert.Equal(t, `max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
mysql_global_variables_max_connections
* 100
`, ref.LHS)
		assert.Equal(t, ">", ref.Operator)
		assert.True(t, ref.Bool)
	})

	t.Run("mongodb_pbm_backup_stale", func(t *testing.T) {
		t.Parallel()

		ref, err := FindSingleExprThresholdRef(mongodbPbmBackupStaleExpr, "threshold")
		require.NoError(t, err)
		assert.Equal(t, `(
  time() - max by (cluster) (
    max_over_time(
      mongodb_pbm_backup_last_transition_ts{status="done"}[30d]
    )
  )
) `, ref.LHS)
		assert.Equal(t, ">", ref.Operator)
		assert.False(t, ref.Bool)
	})
}
