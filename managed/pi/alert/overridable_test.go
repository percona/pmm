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

// overridableTemplate returns a valid multi-expression template whose single param is
// overridable, so each test can vary exactly the one field it cares about.
func overridableTemplate() Template {
	return Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Queries: []TemplateQuery{{
			RefID: "A",
			Expr:  "up",
		}},
		Expressions: []TemplateExpression{{
			RefID:      "C",
			Type:       "math",
			Expression: "$A > [[ .threshold ]]",
		}},
		Condition: "C",
		Params: []Parameter{{
			Name:        "threshold",
			Summary:     "threshold",
			Type:        Float,
			Value:       80,
			Overridable: true,
		}},
	}
}

func TestParamTokenRegexp(t *testing.T) {
	t.Parallel()

	re := ParamTokenRegexp("threshold")

	assert.True(t, re.MatchString("$A > [[ .threshold ]]"))
	assert.True(t, re.MatchString("$A > [[.threshold]]"))
	assert.True(t, re.MatchString("$A > [[   .threshold   ]]"))

	assert.False(t, re.MatchString("$A > [[ .other ]]"))
	assert.False(t, re.MatchString("$A > 80"))

	// A prefix must not match a longer parameter name.
	assert.False(t, ParamTokenRegexp("thresh").MatchString("$A > [[ .threshold ]]"))
}

func TestParamTokenRegexpQuotesName(t *testing.T) {
	t.Parallel()

	// A name containing regexp metacharacters must be matched literally, not as a pattern.
	re := ParamTokenRegexp("a.b")
	assert.True(t, re.MatchString("[[ .a.b ]]"))
	assert.False(t, re.MatchString("[[ .axb ]]"))
}

func TestGetOverrideScopesDefaultsToNode(t *testing.T) {
	t.Parallel()

	param := Parameter{Name: "threshold", Type: Float, Overridable: true}
	assert.Equal(t, []string{OverrideScopeNode}, param.GetOverrideScopes())

	param.OverrideScopes = []string{OverrideScopeService, OverrideScopeCluster}
	assert.Equal(t, []string{OverrideScopeService, OverrideScopeCluster}, param.GetOverrideScopes())
}

func TestOverridableParams(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params = append(template.Params, Parameter{
		Name:    "other",
		Summary: "other",
		Type:    Float,
		Value:   1,
	})

	params := template.OverridableParams()
	require.Len(t, params, 1)
	assert.Equal(t, "threshold", params[0].Name)
}

func TestValidateOverridableTemplate(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	require.NoError(t, template.Validate())
}

func TestValidateOverridableRejectsSingleExpression(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Queries, template.Expressions, template.Condition = nil, nil, ""
	template.Expr = "up > bool [[ .threshold ]]"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires the queries and expressions template form")
}

func TestValidateOverridableRejectsUnreferencedParam(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Expressions[0].Expression = "$A > 80"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not referenced by any expression")
}

func TestValidateOverridableRejectsParamComparedAgainstNoStep(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Expressions[0].Expression = "1 < [[ .threshold ]] && $A > 0"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overridable parameter 'threshold' is not compared against any query in expression C")
}

// A math expression has no query for the default threshold to fan out over.
func TestValidateOverridableRejectsParamComparedAgainstAnExpression(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Queries = append(template.Queries, TemplateQuery{RefID: "M", Expr: "up"})
	template.Expressions = []TemplateExpression{
		{RefID: "B", Type: "math", Expression: "$A * 2"},
		// $B is the nearest step, so the param must not be paired with the earlier query $M.
		{RefID: "C", Type: "math", Expression: "$M > 0 && $B > [[ .threshold ]]"},
	}
	template.Condition = "C"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compared against expression $B, but can only be compared against a MetricsQL query")
}

func TestValidateOverridableRejectsTokenInQuery(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Queries[0].Expr = "up > [[ .threshold ]]"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "overridable parameter 'threshold' is used in query A, but can only be used in expressions")
}

func TestValidateOverridableRequiresBareComparand(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		expression string
		valid      bool
	}{
		{expression: "$A >= [[.threshold]]", valid: true},
		{expression: "($A < [[ .threshold ]]) || $A == 0", valid: true},
		{expression: "$A > [[ .threshold ]] && $A < 1000", valid: true},
		{expression: "$A > [[ .threshold ]] * 100"},
		{expression: "$A > [[ .threshold ]] + 5"},
		{expression: "$A > -[[ .threshold ]]"},
		{expression: "$A > ([[ .threshold ]])"},
		{expression: "[[ .threshold ]] < $A"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			t.Parallel()

			template := overridableTemplate()
			template.Expressions[0].Expression = tc.expression

			err := template.Validate()
			if tc.valid {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), "whole right-hand side of a comparison")
		})
	}
}

func TestValidateOverridableRejectsParamComparedAgainstTwoQueries(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Queries = append(template.Queries, TemplateQuery{RefID: "B", Expr: "up"})
	template.Expressions = []TemplateExpression{
		{RefID: "C", Type: "math", Expression: "$A > [[ .threshold ]]"},
		{RefID: "D", Type: "math", Expression: "$B > [[ .threshold ]] && $C"},
	}
	template.Condition = "D"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compared against both $A and $B")
}

func TestValidateOverridableAcceptsParamReusedAgainstOneQuery(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Expressions = []TemplateExpression{
		{RefID: "C", Type: "math", Expression: "$A > [[ .threshold ]]"},
		{RefID: "D", Type: "math", Expression: "$A > [[ .threshold ]] && $C"},
	}
	template.Condition = "D"

	require.NoError(t, template.Validate())
}

func TestValidateOverridableRejectsNonFloat(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params[0].Type = String
	template.Params[0].Value = "80"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be of type float")
}

func TestValidateOverridableRejectsUnknownScope(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params[0].OverrideScopes = []string{"rack"}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown override scope")
}

func TestValidateRejectsScopesWithoutOverridable(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params[0].Overridable = false
	template.Params[0].OverrideScopes = []string{OverrideScopeNode}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not overridable")
}

func TestValidateAcceptsNonOverridableTemplates(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params[0].Overridable = false
	template.Expressions[0].Expression = "$A > [[ .threshold ]]"

	require.NoError(t, template.Validate())
}

func TestValidateOverridableRejectsMixedScopeFamilies(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params[0].OverrideScopes = []string{OverrideScopeNode, OverrideScopeCluster}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "join on different labels")
}

func TestValidateOverridableAcceptsServiceAndCluster(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Params[0].OverrideScopes = []string{OverrideScopeService, OverrideScopeCluster}

	require.NoError(t, template.Validate())
}

func TestObservedQueryResolvesReduceOfClickHouse(t *testing.T) {
	t.Parallel()

	template := with(clickHouseTemplate(), func(t *Template) {
		t.Params[0].Overridable = true
		t.Queries = append(t.Queries, TemplateQuery{RefID: "M", Expr: "up"})
		t.Expressions[1].Expression = "$M > 0 && $B > [[ .threshold ]]"
	})
	require.NoError(t, template.Validate())

	observed, err := template.ObservedQueryForParam("threshold")
	require.NoError(t, err)
	assert.Equal(t, "A", observed.RefID, "the param is compared against $B, which reduces A")
	assert.Equal(t, ClickHouse, observed.Datasource)
}

func TestValidateOverridableRejectsReduceOfMetricsQL(t *testing.T) {
	t.Parallel()

	template := with(clickHouseTemplate(), func(t *Template) {
		t.Params[0].Overridable = true
		t.Queries[0].Datasource = Metrics
	})

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compared against expression $B")
}
