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
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	alerting "github.com/percona/pmm/api/alerting/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/alert"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
)

const overridableWiringYAML = `templates:
  - name: test_overridable_wiring
    version: 1
    summary: Overridable threshold wiring
    queries:
      - ref_id: A
        expr: |-
          (1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
    expressions:
      - ref_id: C
        type: math
        expression: "$A > [[ .threshold ]]"
    condition: C
    params:
      - name: threshold
        summary: A percentage from configured maximum
        unit: "%"
        type: float
        range: [0, 100]
        value: 80
        overridable: true
    for: 5m
    severity: warning
    annotations:
      summary: Node high CPU load ({{ $labels.node_name }})
`

func templateFromYAML(t *testing.T, yaml string) *models.Template {
	t.Helper()

	parsed, err := alert.Parse(strings.NewReader(yaml), &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	})
	require.NoError(t, err)
	require.Len(t, parsed, 1)

	tm, err := models.ConvertTemplate(&parsed[0], models.UserAPISource)
	require.NoError(t, err)

	return tm
}

// TestCreateRuleRegistersOverridableRule covers the registry lifecycle: a rule with an
// overridable parameter gets a PMM-minted ID, that ID reaches Grafana as a label and the
// database as a row, and the snapshot it stores is what later resolves the threshold.
func TestCreateRuleRegistersOverridableRule(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	tm := templateFromYAML(t, overridableWiringYAML)
	plain := templateFromYAML(t, multiExpressionWiringYAML)

	setup := func(t *testing.T) (*Service, *mockGrafanaClient) {
		t.Helper()

		m := newMockGrafanaClient(t)
		svc, err := NewService(db, m)
		require.NoError(t, err)
		svc.templates = map[string]models.Template{
			tm.Name:    *tm,
			plain.Name: *plain,
		}

		return svc, m
	}

	thresholdParam := []*alerting.ParamValue{{
		Name:  "threshold",
		Type:  alerting.ParamType_PARAM_TYPE_FLOAT,
		Value: &alerting.ParamValue_Float{Float: 80},
	}}

	createRule := func(t *testing.T, svc *Service, templateName string) error {
		t.Helper()

		_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			TemplateName: templateName,
			Name:         "test-rule",
			FolderUid:    "folder-uid",
			Group:        "test-group",
			Severity:     managementv1.Severity_SEVERITY_WARNING,
			Params:       thresholdParam,
		})

		return err
	}

	t.Run("stamps the identity label and injects the threshold step", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)

		var captured *services.Rule
		m.On("CreateAlertRule", mock.Anything, "folder-uid", "test-group", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(4).(*services.Rule) }). //nolint:forcetypeassert
			Return(nil)

		res, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			TemplateName: tm.Name,
			Name:         "test-rule",
			FolderUid:    "folder-uid",
			Group:        "test-group",
			Severity:     managementv1.Severity_SEVERITY_WARNING,
			Params:       thresholdParam,
		})
		require.NoError(t, err)
		require.NotNil(t, captured)

		ruleID := captured.Labels["pmm_rule_id"]
		require.NotEmpty(t, ruleID, "an overridable rule must carry its PMM identity")
		assert.Equal(t, ruleID, res.RuleId, "the response must return the same identity the rule carries")

		byRef := dataByRefID(t, captured.GrafanaAlert.Data)
		require.Contains(t, byRef, "T_threshold")
		assert.Contains(t, exprOf(t, byRef["T_threshold"]), `rule_id="`+ruleID+`"`,
			"the injected query must select the same rule ID the label carries")

		// The registry row exists and holds the snapshot the resolver will need.
		rule, err := models.FindAlertRuleByID(db.Querier, ruleID)
		require.NoError(t, err)

		param, ok := rule.Params["threshold"]
		require.True(t, ok)
		assert.InDelta(t, 80.0, param.Default, 0.0001)
		assert.Equal(t, "node_name", param.JoinLabel)
		assert.Equal(t, []string{"node"}, param.Scopes)
		assert.Equal(t, "%", param.Unit)
		require.NotNil(t, param.Min)
		require.NotNil(t, param.Max)
		assert.InDelta(t, 0.0, *param.Min, 0.0001)
		assert.InDelta(t, 100.0, *param.Max, 0.0001)
	})

	t.Run("the stored default is the value supplied, not the template's", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)

		var captured *services.Rule
		m.On("CreateAlertRule", mock.Anything, "folder-uid", "test-group", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(4).(*services.Rule) }). //nolint:forcetypeassert
			Return(nil)

		_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			TemplateName: tm.Name,
			Name:         "test-rule",
			FolderUid:    "folder-uid",
			Group:        "test-group",
			Severity:     managementv1.Severity_SEVERITY_WARNING,
			Params: []*alerting.ParamValue{{
				Name:  "threshold",
				Type:  alerting.ParamType_PARAM_TYPE_FLOAT,
				Value: &alerting.ParamValue_Float{Float: 42},
			}},
		})
		require.NoError(t, err)

		rule, err := models.FindAlertRuleByID(db.Querier, captured.Labels["pmm_rule_id"])
		require.NoError(t, err)
		assert.InDelta(t, 42.0, rule.Params["threshold"].Default, 0.0001)
	})

	t.Run("a rule with no overridable parameters is not registered", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)

		var captured *services.Rule
		m.On("CreateAlertRule", mock.Anything, "folder-uid", "test-group", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(4).(*services.Rule) }). //nolint:forcetypeassert
			Return(nil)

		before, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)

		require.NoError(t, createRule(t, svc, plain.Name))
		require.NotNil(t, captured)

		assert.Empty(t, captured.Labels["pmm_rule_id"])
		assert.NotContains(t, dataByRefID(t, captured.GrafanaAlert.Data), "T_threshold")

		after, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)
		assert.Len(t, after, len(before), "no registry row should have been written")
	})

	// TestCreateRule writes the registry row before creating the rule in Grafana, so a
	// Grafana failure must not leave the row behind. The other ordering would be worse -
	// a rule whose thresholds can never be overridden - but this one still has to clean up.
	t.Run("a Grafana failure rolls the registry row back", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)
		m.On("CreateAlertRule", mock.Anything, "folder-uid", "test-group", mock.Anything, mock.Anything).
			Return(errors.New("grafana rejected the rule"))

		before, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)

		err = createRule(t, svc, tm.Name)
		require.Error(t, err)

		after, err := models.FindAlertRules(db.Querier)
		require.NoError(t, err)
		assert.Len(t, after, len(before), "the registry row must not outlive the failed creation")
	})
}

func TestCollectOverridableParams(t *testing.T) {
	t.Parallel()

	template := overridableRuleTemplate()

	t.Run("snapshots the supplied value and derived join label", func(t *testing.T) {
		t.Parallel()

		params, err := collectOverridableParams(template, AlertExprParamsValues{{
			Name:       "threshold",
			Type:       models.Float,
			FloatValue: 55,
		}})
		require.NoError(t, err)

		require.Contains(t, params, "threshold")
		assert.InDelta(t, 55.0, params["threshold"].Default, 0.0001)
		assert.Equal(t, "node_name", params["threshold"].JoinLabel)
		assert.Equal(t, []string{alert.OverrideScopeNode}, params["threshold"].Scopes)
	})

	t.Run("service and cluster scopes join on service_name", func(t *testing.T) {
		t.Parallel()

		scoped := overridableRuleTemplate()
		scoped.Params[0].OverrideScopes = []string{alert.OverrideScopeService, alert.OverrideScopeCluster}

		params, err := collectOverridableParams(scoped, AlertExprParamsValues{{
			Name:       "threshold",
			Type:       models.Float,
			FloatValue: 55,
		}})
		require.NoError(t, err)
		assert.Equal(t, "service_name", params["threshold"].JoinLabel)
	})

	t.Run("a missing value is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := collectOverridableParams(template, AlertExprParamsValues{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no value supplied")
	})

	t.Run("a template with no overridable parameters snapshots nothing", func(t *testing.T) {
		t.Parallel()

		plain := overridableRuleTemplate()
		plain.Params[0].Overridable = false

		params, err := collectOverridableParams(plain, AlertExprParamsValues{})
		require.NoError(t, err)
		assert.Nil(t, params)
	})
}

// TestBuiltInOverridableTemplates pins exactly which shipped templates expose an
// overridable threshold. Marking one is not a free change: the injected threshold query
// joins on a single label, so a template whose observed query does not carry that label
// would generate a rule that silently never matches.
func TestBuiltInOverridableTemplates(t *testing.T) {
	t.Parallel()

	// Only node scope resolves so far, so a template qualifies when it is
	// multi-expression and aggregates by node_name.
	want := []string{"pmm_node_high_cpu_load"}

	files, err := filepath.Glob(filepath.Join("..", "..", "data", "alerting-templates", "*.yml"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	var got []string

	for _, file := range files {
		b, err := os.ReadFile(file) //nolint:gosec
		require.NoError(t, err)

		templates, err := alert.Parse(strings.NewReader(string(b)), &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		})
		require.NoError(t, err, "built-in template %s must parse", filepath.Base(file))

		for _, template := range templates {
			if len(template.OverridableParams()) != 0 {
				got = append(got, template.Name)
			}
		}
	}

	sort.Strings(got)
	assert.Equal(t, want, got,
		"adding a template here needs a join label the threshold query can match on; "+
			"templates that aggregate by (cluster) and drop node_name must not be marked overridable")
}
