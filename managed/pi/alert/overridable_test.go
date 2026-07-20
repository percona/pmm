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

func TestValidateOverridableParamRejectsSingleExpression(t *testing.T) {
	t.Parallel()

	template := Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Expr:     "cpu > [[ .threshold ]]",
		Params: []Parameter{
			{Name: "threshold", Summary: "s", Type: Float, Value: 80, Overridable: true},
		},
	}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not multi-expression")
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
