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
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}}, []TemplateExpression{{RefID: "C", Type: "threshold", Expression: "$A"}}, "C"),
			wantErr: "unsupported type",
		},
		{
			name:    "unknown query datasource",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up", Datasource: "loki"}}, nil, "A"),
			wantErr: "template query A: unhandled query datasource 'loki', want 'metrics' or 'clickhouse'",
		},
		{
			name:    "query ref_id with punctuation",
			tmpl:    base([]TemplateQuery{{RefID: "cpu-load", Expr: "up"}}, nil, "cpu-load"),
			wantErr: "template query ref_id 'cpu-load' must start with a letter",
		},
		{
			name:    "query ref_id starting with a digit",
			tmpl:    base([]TemplateQuery{{RefID: "1", Expr: "up"}}, nil, "1"),
			wantErr: "template query ref_id '1' must start with a letter",
		},
		{
			name:    "expression ref_id with punctuation",
			tmpl:    base([]TemplateQuery{{RefID: "A", Expr: "up"}}, []TemplateExpression{{RefID: "C.1", Type: "math", Expression: "$A > 1"}}, "A"),
			wantErr: "template expression ref_id 'C.1' must start with a letter",
		},
		{
			name:    "reduce with unknown reducer",
			tmpl:    with(clickHouseTemplate(), func(t *Template) { t.Expressions[0].Reducer = "avg" }),
			wantErr: "expression B has unsupported reducer 'avg'",
		},
		{
			name:    "reduce of more than one step",
			tmpl:    with(clickHouseTemplate(), func(t *Template) { t.Expressions[0].Expression = "$A + 1" }),
			wantErr: "expression B must reduce a single step",
		},
		{
			name:    "reduce of an unknown step",
			tmpl:    with(clickHouseTemplate(), func(t *Template) { t.Expressions[0].Expression = "$Z" }),
			wantErr: "expression B reduces $Z, which does not match any query or expression ref_id",
		},
		{
			name:    "ClickHouse query as condition",
			tmpl:    with(clickHouseTemplate(), func(t *Template) { t.Condition = "A" }),
			wantErr: "condition A is a ClickHouse query",
		},
		{
			name:    "ClickHouse query read by math",
			tmpl:    with(clickHouseTemplate(), func(t *Template) { t.Expressions[1].Expression = "${A} > 1" }),
			wantErr: "expression C uses ClickHouse query $A directly",
		},
		{
			name: "string param in ClickHouse query",
			tmpl: with(clickHouseTemplate(), func(t *Template) {
				t.Queries[0].Expr = "SELECT 1 WHERE service_name = '[[ .name ]]'"
				t.Params = []Parameter{{Name: "name", Summary: "name", Type: String, Value: "db"}}
			}),
			wantErr: "query A uses string parameter 'name'",
		},
		{
			name:    "$values of a ClickHouse query",
			tmpl:    with(clickHouseTemplate(), func(t *Template) { t.Annotations = map[string]string{"summary": "{{ $values.A.Value }}"} }),
			wantErr: "'summary' reads $values.A",
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

// clickHouseTemplate returns a valid template that reduces a ClickHouse query before comparing it.
func clickHouseTemplate() Template {
	return Template{
		Name:     "test_template",
		Version:  1,
		Summary:  "summary",
		For:      300,
		Severity: common.Warning,
		Queries: []TemplateQuery{{
			RefID:      "A",
			Expr:       "SELECT $__toTime AS time, service_name, max(m_query_time_max) AS v FROM pmm.metrics GROUP BY service_name",
			Datasource: ClickHouse,
		}},
		Expressions: []TemplateExpression{
			{RefID: "B", Type: "reduce", Reducer: "max", Expression: "$A"},
			{RefID: "C", Type: "math", Expression: "$B > [[ .threshold ]]"},
		},
		Condition:   "C",
		Params:      []Parameter{{Name: "threshold", Summary: "threshold", Type: Float, Value: 10}},
		Annotations: map[string]string{"summary": "{{ $values.B.Value }}"},
	}
}

// with returns t after applying change, so table cases can vary one field of a fixture.
func with(t Template, change func(*Template)) Template {
	change(&t)
	return t
}

func TestValidateClickHouseTemplate(t *testing.T) {
	t.Parallel()

	template := clickHouseTemplate()
	require.NoError(t, template.Validate())

	template.Queries[0].Datasource = Metrics
	require.NoError(t, template.Validate(), "an explicit metrics datasource is valid too")
}

func TestParseClickHouseTemplate(t *testing.T) {
	t.Parallel()

	templates, err := Parse(strings.NewReader(`---
templates:
  - name: test_template
    version: 1
    summary: summary
    queries:
      - ref_id: A
        datasource: clickhouse
        expr: SELECT 1
    expressions:
      - ref_id: B
        type: reduce
        reducer: last
        expression: $A
      - ref_id: C
        type: math
        expression: $B > 0
    condition: C
    for: 1m
    severity: warning
`), &ParseParams{DisallowUnknownFields: true, DisallowInvalidTemplates: true})
	require.NoError(t, err)
	require.Len(t, templates, 1)
	assert.Equal(t, ClickHouse, templates[0].Queries[0].Datasource)
	assert.Equal(t, "last", templates[0].Expressions[0].Reducer)
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
