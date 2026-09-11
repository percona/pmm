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

func TestParamReferencedInExpressions(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()

	assert.True(t, template.ParamReferencedInExpressions("threshold"))
	assert.False(t, template.ParamReferencedInExpressions("missing"))
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

// singleExprTemplate returns a valid single-expression template whose one param is
// overridable, so the desugaring constraints can be varied one at a time.
func singleExprTemplate() Template {
	template := overridableTemplate()
	template.Queries = nil
	template.Expressions = nil
	template.Condition = ""
	template.Expr = "up > bool [[ .threshold ]]"

	return template
}

func TestValidateOverridableAcceptsSplittableSingleExpression(t *testing.T) {
	t.Parallel()

	template := singleExprTemplate()
	require.NoError(t, template.Validate())
}

// An expression that cannot be split must fail when the template is parsed. Accepting it
// would produce a rule whose threshold silently never applies.
func TestValidateOverridableRejectsUnsplittableSingleExpression(t *testing.T) {
	t.Parallel()

	template := singleExprTemplate()
	template.Expr = "up > bool [[ .threshold ]] * 100"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be compared directly")
}

func TestValidateOverridableRejectsTwoParamsOnSingleExpression(t *testing.T) {
	t.Parallel()

	template := singleExprTemplate()
	template.Expr = "up > bool [[ .threshold ]]"
	template.Params = append(template.Params, Parameter{
		Name:        "second",
		Summary:     "second",
		Type:        Float,
		Value:       1,
		Overridable: true,
	})

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most one overridable parameter")
}

func TestValidateOverridableRejectsUnreferencedParam(t *testing.T) {
	t.Parallel()

	template := overridableTemplate()
	template.Expressions[0].Expression = "$A > 80"

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be referenced by an expression step")
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
