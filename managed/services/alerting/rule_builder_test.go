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

package alerting

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	alertingv1 "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/pi/alert"
)

func TestBuildGrafanaRuleDataSingleExpression(t *testing.T) {
	t.Parallel()

	data, condition, err := buildGrafanaRuleData(&alert.Template{
		Expr: "up == 1",
	}, "metrics-uid", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "A", condition)
	require.Len(t, data, 1)
	assert.Equal(t, "A", data[0].RefID)
	assert.Equal(t, "metrics-uid", data[0].DatasourceUID)
}

func TestBuildGrafanaRuleDataMultiExpression(t *testing.T) {
	t.Parallel()

	data, condition, err := buildGrafanaRuleData(&alert.Template{
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: "cpu > 0"},
			{RefID: "B", Expr: "vector(80)"},
		},
		Expressions: []alert.TemplateExpression{{
			RefID:      "C",
			Type:       "math",
			Expression: "$A > $B",
		}},
		Condition: "C",
	}, "metrics-uid", map[string]string{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "C", condition)
	require.Len(t, data, 3)

	assert.Equal(t, "A", data[0].RefID)
	assert.Equal(t, "metrics-uid", data[0].DatasourceUID)

	assert.Equal(t, "C", data[2].RefID)
	assert.Equal(t, grafanaExprDatasourceUID, data[2].DatasourceUID)

	var mathModel mathExpressionModel
	err = json.Unmarshal(data[2].Model, &mathModel)
	require.NoError(t, err)
	assert.Equal(t, "math", mathModel.Type)
	assert.Equal(t, "$A > $B", mathModel.Expression)
}

func TestBuildGrafanaRuleDataMultiExpressionWithParamsAndFilters(t *testing.T) {
	t.Parallel()

	data, condition, err := buildGrafanaRuleData(&alert.Template{
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: `rate(node_cpu_seconds_total{mode="idle"}[[ .window ]])`},
			{RefID: "B", Expr: "vector([[ .threshold ]])"},
		},
		Expressions: []alert.TemplateExpression{{
			RefID:      "C",
			Type:       "math",
			Expression: "$A < $B",
		}},
		Condition: "C",
	}, "metrics-uid", map[string]string{
		"window":    "[5m]",
		"threshold": "80",
	}, []*alertingv1.Filter{{
		Type:   alertingv1.FilterType_FILTER_TYPE_MATCH,
		Label:  "node_name",
		Regexp: "db.*",
	}})
	require.NoError(t, err)
	assert.Equal(t, "C", condition)
	require.Len(t, data, 3)

	var queryModelA promQueryModel
	err = json.Unmarshal(data[0].Model, &queryModelA)
	require.NoError(t, err)
	assert.Equal(t, `label_match(rate(node_cpu_seconds_total{mode="idle"}[5m]), "node_name", "(db.*)|")`, queryModelA.Expr)

	var queryModelB promQueryModel
	err = json.Unmarshal(data[1].Model, &queryModelB)
	require.NoError(t, err)
	// Query B is a bare constant (vector(80)) with no node_name label. The trailing empty
	// alternative "(db.*)|" ensures the filter preserves it instead of emptying it.
	assert.Equal(t, `label_match(vector(80), "node_name", "(db.*)|")`, queryModelB.Expr)
}

func TestBuildGrafanaRuleDataModelContract(t *testing.T) {
	t.Parallel()

	data, _, err := buildGrafanaRuleData(&alert.Template{
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: "cpu"},
			{RefID: "B", Expr: "vector(80)"},
		},
		Expressions: []alert.TemplateExpression{{RefID: "C", Type: "math", Expression: "$A > $B"}},
		Condition:   "C",
	}, "metrics-uid", map[string]string{}, nil)
	require.NoError(t, err)
	require.Len(t, data, 3)

	// Query node (leg A): instant query over the last queryRelativeFromSeconds seconds.
	assert.Equal(t, queryRelativeFromSeconds, data[0].RelativeTimeRange.From)
	assert.Equal(t, 0, data[0].RelativeTimeRange.To)
	var queryModel promQueryModel
	require.NoError(t, json.Unmarshal(data[0].Model, &queryModel))
	assert.True(t, queryModel.Instant)
	assert.False(t, queryModel.Hide)
	assert.Equal(t, queryIntervalMs, queryModel.IntervalMs)
	assert.Equal(t, maxDataPoints, queryModel.MaxDataPoints)

	// Math node (leg C): server-side expression, no time range, __expr__ datasource.
	assert.Equal(t, 0, data[2].RelativeTimeRange.From)
	assert.Equal(t, 0, data[2].RelativeTimeRange.To)
	var mathModel mathExpressionModel
	require.NoError(t, json.Unmarshal(data[2].Model, &mathModel))
	assert.False(t, mathModel.Hide)
	assert.Equal(t, queryIntervalMs, mathModel.IntervalMs)
	assert.Equal(t, maxDataPoints, mathModel.MaxDataPoints)
	assert.Equal(t, map[string]string{"type": grafanaExprDatasourceUID, "uid": grafanaExprDatasourceUID}, mathModel.Datasource)
}

func TestBuildGrafanaRuleDataMismatchFilter(t *testing.T) {
	t.Parallel()

	data, _, err := buildGrafanaRuleData(&alert.Template{
		Queries:     []alert.TemplateQuery{{RefID: "A", Expr: "up"}},
		Expressions: []alert.TemplateExpression{{RefID: "C", Type: "math", Expression: "$A > 0"}},
		Condition:   "C",
	}, "metrics-uid", map[string]string{}, []*alertingv1.Filter{{
		Type:   alertingv1.FilterType_FILTER_TYPE_MISMATCH,
		Label:  "node_name",
		Regexp: "staging.*",
	}})
	require.NoError(t, err)

	var queryModel promQueryModel
	require.NoError(t, json.Unmarshal(data[0].Model, &queryModel))
	assert.Equal(t, `label_mismatch(up, "node_name", "staging.*")`, queryModel.Expr)
}

func TestBuildGrafanaRuleDataMultiExpressionErrors(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		tmpl    *alert.Template
		filters []*alertingv1.Filter
		wantErr string
	}{
		{
			name: "unfillable query param",
			tmpl: &alert.Template{
				Queries:     []alert.TemplateQuery{{RefID: "A", Expr: "up > [[ .missing ]]"}},
				Expressions: []alert.TemplateExpression{{RefID: "C", Type: "math", Expression: "$A > 0"}},
				Condition:   "C",
			},
			wantErr: "failed to fill query A",
		},
		{
			name: "unfillable expression param",
			tmpl: &alert.Template{
				Queries:     []alert.TemplateQuery{{RefID: "A", Expr: "up"}},
				Expressions: []alert.TemplateExpression{{RefID: "C", Type: "math", Expression: "$A > [[ .missing ]]"}},
				Condition:   "C",
			},
			wantErr: "failed to fill expression C",
		},
		{
			name: "unknown filter type",
			tmpl: &alert.Template{
				Queries:     []alert.TemplateQuery{{RefID: "A", Expr: "up"}},
				Expressions: []alert.TemplateExpression{{RefID: "C", Type: "math", Expression: "$A > 0"}},
				Condition:   "C",
			},
			filters: []*alertingv1.Filter{{Type: alertingv1.FilterType(999), Label: "l", Regexp: "r"}},
			wantErr: "unknown filter type",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := buildGrafanaRuleData(tc.tmpl, "metrics-uid", map[string]string{}, tc.filters)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
