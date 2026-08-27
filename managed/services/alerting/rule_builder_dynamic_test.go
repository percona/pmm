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
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	alertingv1 "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/pi/alert"
	"github.com/percona/pmm/managed/services"
)

const testObservedExpr = `avg by(node_name) (rate(node_cpu_seconds_total[5m]))`

func overridableRuleTemplate() *alert.Template {
	return &alert.Template{
		Name:    "test_template",
		Version: 1,
		Summary: "summary",
		Queries: []alert.TemplateQuery{{
			RefID: "A",
			Expr:  testObservedExpr,
		}},
		Expressions: []alert.TemplateExpression{{
			RefID:      "C",
			Type:       "math",
			Expression: "$A > [[ .threshold ]]",
		}},
		Condition: "C",
		Params: []alert.Parameter{{
			Name:        "threshold",
			Summary:     "threshold",
			Type:        alert.Float,
			Value:       80,
			Overridable: true,
		}},
	}
}

// dataByRefID indexes generated steps so assertions can address one by name.
func dataByRefID(t *testing.T, data []services.Data) map[string]services.Data {
	t.Helper()

	byRef := make(map[string]services.Data, len(data))
	for _, item := range data {
		byRef[item.RefID] = item
	}

	return byRef
}

// exprOf pulls the PromQL back out of a generated prom query step.
func exprOf(t *testing.T, item services.Data) string {
	t.Helper()

	var model promQueryModel
	require.NoError(t, json.Unmarshal(item.Model, &model))

	return model.Expr
}

// expressionOf pulls the body back out of a generated math expression step.
func expressionOf(t *testing.T, item services.Data) string {
	t.Helper()

	var model mathExpressionModel
	require.NoError(t, json.Unmarshal(item.Model, &model))

	return model.Expression
}

func TestBuildRuleDataInjectsThresholdQuery(t *testing.T) {
	t.Parallel()

	data, condition, err := buildGrafanaRuleData(
		overridableRuleTemplate(), "metrics-uid", "rule-1",
		map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "C", condition)

	byRef := dataByRefID(t, data)
	require.Len(t, data, 3, "observed query, injected threshold, math expression")
	require.Contains(t, byRef, "T_threshold")

	expr := exprOf(t, byRef["T_threshold"])

	// The override clause, mapping the collector's generic target label onto this
	// rule's join label.
	assert.Contains(t, expr, `pmm_alert_threshold_override{rule_id="rule-1", param="threshold"}`)
	assert.Contains(t, expr, `label_replace(`)
	assert.Contains(t, expr, `"node_name", "$1", "target", "(.*)"`)

	// The default clause, fanned out over the rule's own observed query.
	assert.Contains(t, expr, `or (max by (node_name) (`+testObservedExpr+`) * 0 + 80)`)

	// The threshold query is a metrics query, not an expression.
	assert.Equal(t, "metrics-uid", byRef["T_threshold"].DatasourceUID)
}

func TestBuildRuleDataSwapsTokenForThresholdRef(t *testing.T) {
	t.Parallel()

	data, _, err := buildGrafanaRuleData(
		overridableRuleTemplate(), "metrics-uid", "rule-1",
		map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)

	body := expressionOf(t, dataByRefID(t, data)["C"])

	assert.Equal(t, "$A > $T_threshold", body)
	assert.NotContains(t, body, "80", "the default must never be baked into the expression")
}

// TestBuildRuleDataWithoutRuleIDIsUnchanged pins the compatibility path: a rule with no
// PMM-minted ID generates exactly what it did before this feature existed.
func TestBuildRuleDataWithoutRuleIDIsUnchanged(t *testing.T) {
	t.Parallel()

	data, _, err := buildGrafanaRuleData(
		overridableRuleTemplate(), "metrics-uid", "",
		map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)

	require.Len(t, data, 2)
	assert.NotContains(t, dataByRefID(t, data), "T_threshold")
	assert.Equal(t, "$A > 80", expressionOf(t, dataByRefID(t, data)["C"]))
}

func TestBuildRuleDataWithoutOverridableParams(t *testing.T) {
	t.Parallel()

	template := overridableRuleTemplate()
	template.Params[0].Overridable = false

	data, _, err := buildGrafanaRuleData(
		template, "metrics-uid", "rule-1",
		map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)

	require.Len(t, data, 2)
	assert.Equal(t, "$A > 80", expressionOf(t, dataByRefID(t, data)["C"]))
}

// TestThresholdQueryMatchesCollectorDescriptor is the contract test between the generated
// PromQL and the emitted series. The proof-of-concept this replaces shipped a builder
// querying pmm_alert_threshold while the collector emitted pmm_alert_threshold_override,
// so every rule matched nothing and silently never fired. Nothing about that failure is
// visible at runtime, which is why it is asserted here.
func TestThresholdQueryMatchesCollectorDescriptor(t *testing.T) {
	t.Parallel()

	desc := NewAlertThresholdMetricsCollector(nil).desc.String()

	fqName := regexp.MustCompile(`fqName: "([^"]+)"`).FindStringSubmatch(desc)
	require.Len(t, fqName, 2, "could not read fqName from %s", desc)

	labels := regexp.MustCompile(`variableLabels: \{([^}]*)\}`).FindStringSubmatch(desc)
	require.Len(t, labels, 2, "could not read variableLabels from %s", desc)

	data, _, err := buildGrafanaRuleData(
		overridableRuleTemplate(), "metrics-uid", "rule-1",
		map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)

	expr := exprOf(t, dataByRefID(t, data)["T_threshold"])

	assert.Contains(t, expr, fqName[1]+"{", "the query must select the metric the collector registers")

	for _, label := range strings.Split(labels[1], ",") {
		label = strings.TrimSpace(label)
		require.NotEmpty(t, label)
		assert.Contains(t, expr, label, "the query must reference every label the collector emits")
	}
}

func TestThresholdRefIDAvoidsTemplateCollision(t *testing.T) {
	t.Parallel()

	template := overridableRuleTemplate()
	// The template already uses the ref ID the parameter would otherwise claim.
	template.Queries = append(template.Queries, alert.TemplateQuery{
		RefID: "T_threshold",
		Expr:  "up",
	})

	data, _, err := buildGrafanaRuleData(
		template, "metrics-uid", "rule-1",
		map[string]string{"threshold": "80"}, nil)
	require.NoError(t, err)

	byRef := dataByRefID(t, data)
	assert.Contains(t, byRef, "T_threshold_1")
	assert.Equal(t, "$A > $T_threshold_1", expressionOf(t, byRef["C"]))
}

// TestThresholdPairsEachParamWithItsOwnQuery covers a template comparing two queries in
// one expression: each parameter's default must fan out over the query it is actually
// compared against, not over whichever query happens to come first.
func TestThresholdPairsEachParamWithItsOwnQuery(t *testing.T) {
	t.Parallel()

	template := &alert.Template{
		Name:    "dual",
		Version: 1,
		Summary: "summary",
		Queries: []alert.TemplateQuery{
			{RefID: "A", Expr: "query_a"},
			{RefID: "B", Expr: "query_b"},
		},
		Expressions: []alert.TemplateExpression{{
			RefID:      "C",
			Type:       "math",
			Expression: "$A > [[ .first ]] && $B > [[ .second ]]",
		}},
		Condition: "C",
		Params: []alert.Parameter{
			{Name: "first", Summary: "first", Type: alert.Float, Value: 1, Overridable: true},
			{Name: "second", Summary: "second", Type: alert.Float, Value: 2, Overridable: true},
		},
	}

	data, _, err := buildGrafanaRuleData(
		template, "metrics-uid", "rule-1",
		map[string]string{"first": "1", "second": "2"}, nil)
	require.NoError(t, err)

	byRef := dataByRefID(t, data)

	assert.Contains(t, exprOf(t, byRef["T_first"]), `(query_a) * 0 + 1)`)
	assert.Contains(t, exprOf(t, byRef["T_second"]), `(query_b) * 0 + 2)`)
	assert.Equal(t, "$A > $T_first && $B > $T_second", expressionOf(t, byRef["C"]))
}

// TestThresholdQueryIsNotFiltered pins that alert filters narrow the observed query but
// not the threshold. A filtered threshold would leave the targets the filter excludes
// with no threshold at all.
func TestThresholdQueryIsNotFiltered(t *testing.T) {
	t.Parallel()

	data, _, err := buildGrafanaRuleData(
		overridableRuleTemplate(), "metrics-uid", "rule-1",
		map[string]string{"threshold": "80"},
		[]*alertingv1.Filter{{
			Type:   alertingv1.FilterType_FILTER_TYPE_MATCH,
			Label:  "node_name",
			Regexp: "prod-.*",
		}})
	require.NoError(t, err)

	byRef := dataByRefID(t, data)

	assert.Contains(t, exprOf(t, byRef["A"]), "label_match(", "the observed query is filtered")
	assert.NotContains(t, exprOf(t, byRef["T_threshold"]), "label_match(", "the threshold query is not")
}

func TestJoinLabelForScopes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		scopes  []string
		want    string
		wantErr string
	}{
		{name: "node", scopes: []string{alert.OverrideScopeNode}, want: nodeJoinLabel},
		{name: "service", scopes: []string{alert.OverrideScopeService}, want: serviceJoinLabel},
		{name: "cluster", scopes: []string{alert.OverrideScopeCluster}, want: serviceJoinLabel},
		{
			name:   "service and cluster share a join label",
			scopes: []string{alert.OverrideScopeService, alert.OverrideScopeCluster},
			want:   serviceJoinLabel,
		},
		{
			name:    "node cannot be mixed with cluster",
			scopes:  []string{alert.OverrideScopeNode, alert.OverrideScopeCluster},
			wantErr: "join on different labels",
		},
		{
			name:    "unknown scope",
			scopes:  []string{"rack"},
			wantErr: "unknown override scope",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := joinLabelForScopes(tc.scopes)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestObservedQueryForParamErrors(t *testing.T) {
	t.Parallel()

	t.Run("parameter compared against no query", func(t *testing.T) {
		t.Parallel()

		template := overridableRuleTemplate()
		template.Expressions[0].Expression = "[[ .threshold ]] > 1"

		_, err := observedQueryForParam(template, "threshold")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not compared against any query")
	})

	t.Run("parameter referenced by no expression", func(t *testing.T) {
		t.Parallel()

		_, err := observedQueryForParam(overridableRuleTemplate(), "missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not referenced by any expression")
	})
}
