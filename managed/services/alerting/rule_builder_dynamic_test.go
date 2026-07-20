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

	"github.com/percona/pmm/managed/pi/alert"
)

func modelExpr(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var m promQueryModel
	require.NoError(t, json.Unmarshal(raw, &m))
	return m.Expr
}

func mathExpr(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var m mathExpressionModel
	require.NoError(t, json.Unmarshal(raw, &m))
	return m.Expression
}

func TestBuildGrafanaRuleDataOverridableThreshold(t *testing.T) {
	t.Parallel()

	tmpl := &alert.Template{
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: `(1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100`},
		},
		Expressions: []alert.TemplateExpression{
			{RefID: "C", Type: "math", Expression: "$A > [[ .threshold ]]"},
		},
		Condition: "C",
		Params: []alert.Parameter{
			{Name: "threshold", Summary: "s", Type: alert.Float, Overridable: true, Value: 80},
		},
	}

	data, condition, err := buildGrafanaRuleData(tmpl, "metrics-uid", "rid-1", map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "C", condition)
	require.Len(t, data, 3) // A + injected T_threshold + C

	// Injected threshold query, reduced to the join label so it matches leg A.
	assert.Equal(t, "T_threshold", data[1].RefID)
	assert.Equal(t, "metrics-uid", data[1].DatasourceUID)
	assert.Equal(t, `max by (node_name) (pmm_alert_threshold{rule_id="rid-1", param="threshold"})`, modelExpr(t, data[1].Model))

	// Expression references the injected ref, not the baked-in default.
	assert.Equal(t, "$A > $T_threshold", mathExpr(t, data[2].Model))
	assert.NotContains(t, mathExpr(t, data[2].Model), "80")
}

func TestBuildGrafanaRuleDataMultipleOverridableThresholds(t *testing.T) {
	t.Parallel()

	tmpl := &alert.Template{
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: "cpu"},
			{RefID: "B", Expr: "mem"},
		},
		Expressions: []alert.TemplateExpression{
			{RefID: "C", Type: "math", Expression: "$A > [[ .cpu_threshold ]] && $B > [[ .memory_threshold ]]"},
		},
		Condition: "C",
		Params: []alert.Parameter{
			{Name: "cpu_threshold", Summary: "s", Type: alert.Float, Overridable: true, Value: 80},
			{Name: "memory_threshold", Summary: "s", Type: alert.Float, Overridable: true, Value: 90},
		},
	}

	data, _, err := buildGrafanaRuleData(tmpl, "metrics-uid", "rid-2", map[string]string{
		"cpu_threshold":    "80",
		"memory_threshold": "90",
	}, nil)
	require.NoError(t, err)
	require.Len(t, data, 5) // A, B, T_cpu_threshold, T_memory_threshold, C

	// Injected queries are appended after the template queries, sorted by param name.
	byRef := map[string]string{}
	for _, d := range data {
		if d.DatasourceUID == "metrics-uid" {
			byRef[d.RefID] = modelExpr(t, d.Model)
		}
	}
	assert.Equal(t, `max by (node_name) (pmm_alert_threshold{rule_id="rid-2", param="cpu_threshold"})`, byRef["T_cpu_threshold"])
	assert.Equal(t, `max by (node_name) (pmm_alert_threshold{rule_id="rid-2", param="memory_threshold"})`, byRef["T_memory_threshold"])

	// Both tokens swapped; the combined condition is preserved.
	assert.Equal(t, "$A > $T_cpu_threshold && $B > $T_memory_threshold", mathExpr(t, data[4].Model))
}

func TestBuildGrafanaRuleDataOverridableRefIDCollision(t *testing.T) {
	t.Parallel()

	// A template query already uses the ref ID the injector would pick.
	tmpl := &alert.Template{
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: "cpu"},
			{RefID: "T_threshold", Expr: "something"},
		},
		Expressions: []alert.TemplateExpression{
			{RefID: "C", Type: "math", Expression: "$A > [[ .threshold ]]"},
		},
		Condition: "C",
		Params: []alert.Parameter{
			{Name: "threshold", Summary: "s", Type: alert.Float, Overridable: true, Value: 80},
		},
	}

	data, _, err := buildGrafanaRuleData(tmpl, "metrics-uid", "rid-3", map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)
	require.Len(t, data, 4) // A, T_threshold (template), T_threshold_1 (injected), C

	assert.Equal(t, "$A > $T_threshold_1", mathExpr(t, data[3].Model))
}

func TestBuildGrafanaRuleDataNonOverridableUnchanged(t *testing.T) {
	t.Parallel()

	// Without overridable params the default is baked in as before.
	tmpl := &alert.Template{
		Queries: []alert.TemplateQuery{{RefID: "A", Expr: "cpu"}},
		Expressions: []alert.TemplateExpression{
			{RefID: "C", Type: "math", Expression: "$A > [[ .threshold ]]"},
		},
		Condition: "C",
		Params: []alert.Parameter{
			{Name: "threshold", Summary: "s", Type: alert.Float, Value: 80},
		},
	}

	data, _, err := buildGrafanaRuleData(tmpl, "metrics-uid", "", map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)
	require.Len(t, data, 2) // A + C, no injected query
	assert.Equal(t, "$A > 80", mathExpr(t, data[1].Model))
}
