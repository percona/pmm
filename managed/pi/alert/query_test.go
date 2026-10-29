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
	require.Len(t, template.Queries, 1)
	require.Len(t, template.Expressions, 1)
	assert.Equal(t, "$A > [[ .threshold ]]", template.Expressions[0].Expression)
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

// TestValidateMultiExpressionTemplateRejectsQueryExpressionRefIDCollision covers the case
// where a query and an expression share a ref_id.
func TestValidateMultiExpressionTemplateRejectsQueryExpressionRefIDCollision(t *testing.T) {
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
	assert.Contains(t, err.Error(), "duplicate template expression ref_id")
}

// TestValidateMultiExpressionStepsErrors covers the individual validation branches for
// multi-expression templates that are not otherwise exercised.
func TestValidateMultiExpressionStepsErrors(t *testing.T) {
	t.Parallel()

	base := func(queries []TemplateQuery, expressions []TemplateExpression, condition string) Template {
		return Template{
			Name:        "test_template",
			Version:     1,
			Summary:     "summary",
			For:         300,
			Severity:    common.Warning,
			Queries:     queries,
			Expressions: expressions,
			Condition:   condition,
		}
	}

	for _, tc := range []struct {
		name    string
		tmpl    Template
		wantErr string
	}{
		{
			name:    "condition does not resolve to a ref_id",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}}, nil, "Z"),
			wantErr: "does not match any query or expression ref_id",
		},
		{
			name:    "duplicate ref_id within queries",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}, {RefID: "A", Expr: "down"}}, nil, "A"),
			wantErr: "duplicate template query ref_id",
		},
		{
			name:    "empty query ref_id",
			tmpl:    base([]TemplateQuery{{RefID: "", Expr: "up"}}, nil, "A"),
			wantErr: "query ref_id is empty",
		},
		{
			name:    "empty query expression",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: ""}}, nil, "A"),
			wantErr: "query A expression is empty",
		},
		{
			name:    "unsupported expression type",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}}, []TemplateExpression{{RefID: "C", Type: "reduce", Expression: "$A"}}, "C"),
			wantErr: "unsupported type",
		},
		{
			name:    "empty expression ref_id",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}}, []TemplateExpression{{RefID: "", Type: "math", Expression: "$A > 1"}}, "A"),
			wantErr: "expression ref_id is empty",
		},
		{
			name:    "empty expression body",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}}, []TemplateExpression{{RefID: "C", Type: "math", Expression: ""}}, "C"),
			wantErr: "expression C is empty",
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
