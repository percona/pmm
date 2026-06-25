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

	"github.com/percona/pmm/managed/pi/common"
)

func TestParseMultiExpressionTemplate(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile(filepath.Join("..", "..", "data", "alerting-templates", "node_high_cpu_load.yml"))
	require.NoError(t, err)

	templates, err := Parse(strings.NewReader(string(b)), &ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	})
	require.NoError(t, err)
	require.Len(t, templates, 1)

	template := templates[0]
	assert.True(t, template.UsesMultipleExpressions())
	assert.Equal(t, "C", template.Condition)
	require.Len(t, template.Queries, 2)
	require.Len(t, template.Expressions, 1)
	assert.Equal(t, "$A > $B", template.Expressions[0].Expression)
}

func TestValidateMultiExpressionTemplateRequiresCondition(t *testing.T) {
	t.Parallel()

	template := Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Queries: []TemplateQuery{{
			RefID: "A",
			Expr:  "up",
		}},
	}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "condition")
}

func TestValidateMultiExpressionTemplateRejectsLegacyExpr(t *testing.T) {
	t.Parallel()

	template := Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Expr:     "up",
		Queries: []TemplateQuery{
			{RefID: "A", Expr: "up"},
		},
		Condition: "A",
	}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expr should be empty")
}

func TestValidateMultiExpressionTemplateRejectsDuplicateRefID(t *testing.T) {
	t.Parallel()

	template := Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Queries: []TemplateQuery{
			{RefID: "A", Expr: "up"},
		},
		Expressions: []TemplateExpression{
			{RefID: "A", Type: "math", Expression: "$A > 1"},
		},
		Condition: "A",
	}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestValidateSingleExpressionRejectsConditionWithoutQueries(t *testing.T) {
	t.Parallel()

	template := Template{
		Name:      "test_template",
		Version:   1,
		Summary:   "summary",
		For:       300,
		Severity:  common.Warning,
		Expr:      "up == 0",
		Condition: "A",
	}

	err := template.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queries are required")
}
