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
	assert.Equal(t, `label_match(rate(node_cpu_seconds_total{mode="idle"}[5m]), "node_name", "db.*")`, queryModelA.Expr)

	var queryModelB promQueryModel
	err = json.Unmarshal(data[1].Model, &queryModelB)
	require.NoError(t, err)
	assert.Equal(t, `label_match(vector(80), "node_name", "db.*")`, queryModelB.Expr)
}
