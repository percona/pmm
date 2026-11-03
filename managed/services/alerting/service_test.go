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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	alerting "github.com/percona/pmm/api/alerting/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/alert"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	testBadTemplates = "../../testdata/alerting-templates/bad"
	testTemplates    = "../../testdata/alerting-templates/user2"
	testTemplates2   = "../../testdata/alerting-templates/user"
)

func TestCollect(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	t.Run("builtin are valid", func(t *testing.T) {
		t.Parallel()

		svc, err := NewService(db, nil)
		require.NoError(t, err)
		_, err = svc.loadBuiltinTemplates()
		require.NoError(t, err)
	})

	t.Run("bad template paths", func(t *testing.T) {
		t.Parallel()

		svc, err := NewService(db, nil)
		require.NoError(t, err)
		svc.userTemplatesPath = testBadTemplates
		templates, err := svc.loadTemplatesFromUserFiles(ctx)
		assert.Empty(t, templates)
		require.NoError(t, err)
	})

	t.Run("valid template paths", func(t *testing.T) {
		t.Parallel()

		svc, err := NewService(db, nil)
		require.NoError(t, err)
		svc.userTemplatesPath = testTemplates2
		svc.CollectTemplates(ctx)

		templates := svc.GetTemplates()
		require.NotEmpty(t, templates)
		assert.Contains(t, templates, "test_template")
		assert.Contains(t, templates, "pmm_mysql_down")
		assert.Contains(t, templates, "pmm_mysql_restarted")
		assert.Contains(t, templates, "pmm_mysql_too_many_connections")

		// check whether map was cleared and updated on a subsequent call
		svc.userTemplatesPath = testTemplates
		svc.CollectTemplates(ctx)

		templates = svc.GetTemplates()
		require.NotEmpty(t, templates)
		assert.NotContains(t, templates, "test_template")
		assert.Contains(t, templates, "test_template_2")
	})
}

func TestTemplateValidation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	t.Run("create a template with missing param", func(t *testing.T) {
		t.Parallel()

		const templateWithMissingParam = `
---
templates: 
  - name: template_with_missing_param_1
    version: 1
    summary: Template with missing param 1
    expr: |-
      max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
      mysql_global_variables_max_connections
      * 100
      > [[ .threshold ]]
    params: 
      - name: from
        summary: A percentage from configured maximum
        unit: '%'
        type: float
        range: [0, 100]
        value: 80
      - name: to
        summary: A percentage from configured maximum
        unit: '%'
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    severity: warning
    labels: 
      foo: bar
    annotations: 
      description: |-
        More than [[ .threshold ]]% of MySQL connections are in use on {{ $labels.instance }}
        VALUE = {{ $value }}
        LABELS: {{ $labels }}
      summary: MySQL too many connections (instance {{ $labels.instance }})
`
		resp, err := svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{
			Yaml: templateWithMissingParam,
		})
		assert.Nil(t, resp)
		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = failed to fill expression "+
			"placeholders: template: :4:5: executing \"\" at <.threshold>: map has no entry for key \"threshold\".")
	})

	t.Run("update valid template with a template with missing param", func(t *testing.T) {
		t.Parallel()

		const validTemplate = `
---
templates: 
  - name: valid_template_1
    version: 1
    summary: Valid template 1
    expr: |-
      max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
      mysql_global_variables_max_connections
      * 100
      > [[ .threshold ]]
    params:
      - name: threshold
        summary: A threshold
        unit: '%'
        type: float
        range: [0, 100]
        value: 95
      - name: from
        summary: A percentage from configured maximum
        unit: '%'
        type: float
        range: [0, 100]
        value: 80
      - name: to
        summary: A percentage from configured maximum
        unit: '%'
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    severity: warning
    labels: 
      foo: bar
    annotations: 
      description: |-
        More than [[ .threshold ]]% of MySQL connections are in use on {{ $labels.instance }}
        VALUE = {{ $value }}
        LABELS: {{ $labels }}
      summary: MySQL too many connections (instance {{ $labels.instance }})
`

		const templateWithMissingParam = `
---
templates: 
  - name: valid_template_1
    version: 1
    summary: Actually this template isn't valid because of missing threshold param :) 
    expr: |-
      max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
      mysql_global_variables_max_connections
      * 100
      > [[ .threshold ]]
    params:
      - name: from
        summary: A percentage from configured maximum
        unit: '%'
        type: float
        range: [0, 100]
        value: 80
      - name: to
        summary: A percentage from configured maximum
        unit: '%'
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    severity: warning
    labels: 
      foo: bar
    annotations: 
      description: |-
        More than [[ .threshold ]]% of MySQL connections are in use on {{ $labels.instance }}
        VALUE = {{ $value }}
        LABELS: {{ $labels }}
      summary: MySQL too many connections (instance {{ $labels.instance }})
`
		createResp, err := svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{
			Yaml: validTemplate,
		})
		require.NoError(t, err)
		assert.NotNil(t, createResp)

		resp, err := svc.UpdateTemplate(ctx, &alerting.UpdateTemplateRequest{
			Name: "valid_template_1",
			Yaml: templateWithMissingParam,
		})
		assert.Nil(t, resp)
		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = failed to fill expression "+
			"placeholders: template: :4:5: executing \"\" at <.threshold>: map has no entry for key \"threshold\".")
	})
}

func TestEnsureRuleLabel(t *testing.T) {
	t.Parallel()

	t.Run("adds missing label", func(t *testing.T) {
		t.Parallel()

		labels := map[string]string{
			"template_name": "pmm_mysql_down",
		}

		ensureRuleLabel(labels, "node_name", buildRuleLabelTemplate("node_name", "A"))
		assert.Equal(
			t,
			"{{ if $labels.node_name }}{{ $labels.node_name }}{{ else }}{{ $values.A.Labels.node_name }}{{ end }}",
			labels["node_name"],
		)
	})

	t.Run("does not override existing label", func(t *testing.T) {
		t.Parallel()

		labels := map[string]string{
			"service_name": "custom-service",
		}

		ensureRuleLabel(labels, "service_name", "{{ $labels.service_name }}")
		assert.Equal(t, "custom-service", labels["service_name"])
	})
}

func TestQueryRefForRuleLabels(t *testing.T) {
	t.Parallel()

	t.Run("defaults to A for single-expression templates", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "A", queryRefForRuleLabels(&alert.Template{}))
	})

	t.Run("uses first query ref for multi-expression templates", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Q1", queryRefForRuleLabels(&alert.Template{
			Queries: []alert.TemplateQuery{{RefID: "Q1"}, {RefID: "Q2"}},
		}))
	})
}

func TestConvertTemplate(t *testing.T) {
	t.Parallel()

	parse := func(t *testing.T, y string) models.Template {
		t.Helper()
		templates, err := alert.Parse(strings.NewReader(y), &alert.ParseParams{
			DisallowUnknownFields:    true,
			DisallowInvalidTemplates: true,
		})
		require.NoError(t, err)
		require.Len(t, templates, 1)
		tm, err := models.ConvertTemplate(&templates[0], models.UserAPISource)
		require.NoError(t, err)
		return *tm
	}

	t.Run("exposes multi-query steps", func(t *testing.T) {
		t.Parallel()

		const multiYAML = `templates:
  - name: pmm_node_high_cpu_load_test
    version: 1
    summary: Node high CPU load
    queries:
      - ref_id: A
        expr: |-
          (1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
      - ref_id: B
        expr: |-
          label_replace(vector(10), "node_name", "pmm-server", "", "") or (group by(node_name) (node_cpu_seconds_total) * 0 + [[ .threshold ]])
    expressions:
      - ref_id: C
        type: math
        expression: "$A > $B"
    condition: C
    params:
      - name: threshold
        summary: A percentage from configured maximum
        unit: "%"
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    severity: warning
    annotations:
      summary: Node high CPU load ({{ $labels.node_name }})
      description: '{{ $labels.node_name }} CPU load is more than [[ .threshold ]]%.'
`

		tm := parse(t, multiYAML)

		at, err := parseAlertTemplate(tm.Yaml)
		require.NoError(t, err)

		dto, err := convertTemplate(logrus.WithField("test", t.Name()), tm, at)
		require.NoError(t, err)

		// expr keeps the first query (ref A) for backward compatibility.
		assert.Equal(t, tm.Expr, dto.Expr)

		require.Len(t, dto.Queries, 2)
		assert.Equal(t, "A", dto.Queries[0].RefId)
		assert.Equal(t, "B", dto.Queries[1].RefId)
		assert.Contains(t, dto.Queries[1].Expr, "[[ .threshold ]]") // placeholders kept intact

		require.Len(t, dto.Expressions, 1)
		assert.Equal(t, "C", dto.Expressions[0].RefId)
		assert.Equal(t, "math", dto.Expressions[0].Type)
		assert.Equal(t, "$A > $B", dto.Expressions[0].Expression)

		assert.Equal(t, "C", dto.Condition)
	})

	t.Run("leaves multi-query fields empty for single-expression templates", func(t *testing.T) {
		t.Parallel()

		const singleYAML = `templates:
  - name: pmm_single_expr_test
    version: 1
    summary: Single expr
    expr: node_cpu_seconds_total > [[ .threshold ]]
    params:
      - name: threshold
        summary: threshold
        unit: "%"
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    severity: warning
    annotations:
      summary: s
      description: d
`

		tm := parse(t, singleYAML)

		at, err := parseAlertTemplate(tm.Yaml)
		require.NoError(t, err)

		// A single-expression template is cached too, but applyMultiExpressionFields
		// leaves the multi-query fields empty for it.
		dto, err := convertTemplate(logrus.WithField("test", t.Name()), tm, at)
		require.NoError(t, err)

		assert.NotEmpty(t, dto.Expr)
		assert.Empty(t, dto.Queries)
		assert.Empty(t, dto.Expressions)
		assert.Empty(t, dto.Condition)
	})

	t.Run("rejects unparseable YAML at parse time", func(t *testing.T) {
		t.Parallel()

		// Parsing moved to collect time: CollectTemplates calls parseAlertTemplate
		// once per template and logs-and-skips on failure (so one bad template no
		// longer fails the whole list). This asserts that parse gate rejects
		// malformed YAML; the single/multi cases above cover the convertTemplate seam.
		_, err := parseAlertTemplate("templates: [unclosed")
		require.Error(t, err)
	})
}

// multiExpressionWiringYAML is a multi-expression template with a data query (A), a constant
// threshold query (B), a math condition (C), and a threshold param.
const multiExpressionWiringYAML = `templates:
  - name: test_multi_expression_wiring
    version: 1
    summary: Multi-expression wiring
    queries:
      - ref_id: A
        expr: |-
          (1 - avg by(node_name) (rate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
      - ref_id: B
        expr: vector([[ .threshold ]])
    expressions:
      - ref_id: C
        type: math
        expression: "$A > $B"
    condition: C
    params:
      - name: threshold
        summary: A percentage from configured maximum
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    severity: warning
    annotations:
      summary: Node high CPU load ({{ $labels.node_name }})
`

// Not t.Parallel: testdb.Open DROP/CREATEs a single shared database, so running this
// concurrently with other DB-backed tests would deadlock on the open connection.
func TestCreateRuleMultiExpression(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	parsed, err := alert.Parse(strings.NewReader(multiExpressionWiringYAML), &alert.ParseParams{
		DisallowUnknownFields:    true,
		DisallowInvalidTemplates: true,
	})
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	tm, err := models.ConvertTemplate(&parsed[0], models.UserAPISource)
	require.NoError(t, err)

	// setup builds a Service backed by testdb with the multi-expression template registered
	// and a fresh mock grafana client whose expectations are auto-asserted on cleanup.
	setup := func(t *testing.T) (*Service, *mockGrafanaClient) {
		t.Helper()
		m := newMockGrafanaClient(t)
		svc, err := NewService(db, m)
		require.NoError(t, err)
		svc.templates = map[string]models.Template{tm.Name: *tm}
		return svc, m
	}

	thresholdParam := []*alerting.ParamValue{{
		Name:  "threshold",
		Type:  alerting.ParamType_PARAM_TYPE_FLOAT,
		Value: &alerting.ParamValue_Float{Float: 80},
	}}

	t.Run("wires condition, data nodes and grouping labels", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)
		var captured *services.Rule
		m.On("CreateAlertRule", mock.Anything, "folder-uid", "test-group", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(4).(*services.Rule) }).
			Return(nil)

		_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			TemplateName: tm.Name,
			Name:         "test-rule",
			FolderUid:    "folder-uid",
			Group:        "test-group",
			Severity:     managementv1.Severity_SEVERITY_WARNING,
			Params:       thresholdParam,
		})
		require.NoError(t, err)
		require.NotNil(t, captured)

		// Condition and the three data nodes (A/B queries + C math) with the right datasources.
		assert.Equal(t, "C", captured.GrafanaAlert.Condition)
		require.Len(t, captured.GrafanaAlert.Data, 3)
		assert.Equal(t, "A", captured.GrafanaAlert.Data[0].RefID)
		assert.Equal(t, "metrics-uid", captured.GrafanaAlert.Data[0].DatasourceUID)
		assert.Equal(t, "B", captured.GrafanaAlert.Data[1].RefID)
		assert.Equal(t, "metrics-uid", captured.GrafanaAlert.Data[1].DatasourceUID)
		assert.Equal(t, "C", captured.GrafanaAlert.Data[2].RefID)
		assert.Equal(t, grafanaExprDatasourceUID, captured.GrafanaAlert.Data[2].DatasourceUID)

		// Injected labels, including the grouping labels keyed on the first query ref (A).
		assert.Equal(t, "1", captured.Labels["percona_alerting"])
		assert.Equal(t, tm.Name, captured.Labels["template_name"])
		assert.NotEmpty(t, captured.Labels["severity"])
		assert.Equal(t, buildRuleLabelTemplate("node_name", "A"), captured.Labels["node_name"])
		assert.Equal(t, buildRuleLabelTemplate("service_name", "A"), captured.Labels["service_name"])
	})

	t.Run("filter preserves the constant query", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)
		var captured *services.Rule
		m.On("CreateAlertRule", mock.Anything, "folder-uid", "test-group", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(4).(*services.Rule) }).
			Return(nil)

		_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			TemplateName: tm.Name,
			Name:         "test-rule",
			FolderUid:    "folder-uid",
			Group:        "test-group",
			Severity:     managementv1.Severity_SEVERITY_WARNING,
			Params:       thresholdParam,
			Filters: []*alerting.Filter{{
				Type:   alerting.FilterType_FILTER_TYPE_MATCH,
				Label:  "node_name",
				Regexp: "db.*",
			}},
		})
		require.NoError(t, err)
		require.NotNil(t, captured)
		require.Len(t, captured.GrafanaAlert.Data, 3)

		// The constant query B is wrapped with the empty-alternative so it is not emptied.
		var queryB promQueryModel
		require.NoError(t, json.Unmarshal(captured.GrafanaAlert.Data[1].Model, &queryB))
		assert.Contains(t, queryB.Expr, `label_match(vector(80), "node_name", "(db.*)|")`)
	})

	t.Run("unknown template returns NotFound", func(t *testing.T) {
		svc, m := setup(t)
		m.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("metrics-uid", nil)

		_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			TemplateName: "does_not_exist",
			FolderUid:    "folder-uid",
			Group:        "test-group",
		})
		assert.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("missing required fields return InvalidArgument", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			req  *alerting.CreateRuleRequest
		}{
			{"no template name", &alerting.CreateRuleRequest{FolderUid: "folder-uid", Group: "test-group"}},
			{"no folder", &alerting.CreateRuleRequest{TemplateName: tm.Name, Group: "test-group"}},
			{"no group", &alerting.CreateRuleRequest{TemplateName: tm.Name, FolderUid: "folder-uid"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				svc, _ := setup(t)
				_, err := svc.CreateRule(ctx, tc.req)
				assert.Equal(t, codes.InvalidArgument, status.Code(err))
			})
		}
	})
}

// TestMultiExpressionTemplateParamValidation checks that the CreateTemplate param dry-run
// rejects a multi-expression template that references an undeclared parameter in either a
// query or an expression.
// Not t.Parallel: see TestCreateRuleMultiExpression. The subtests reuse the single db opened
// here, so they stay parallel without contending on testdb's shared database.
//
//nolint:tparallel // parent must stay serial.
func TestMultiExpressionTemplateParamValidation(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	t.Run("undeclared param in a query", func(t *testing.T) {
		t.Parallel()
		yaml := `templates:
  - name: test_multi_missing_query_param
    version: 1
    summary: s
    queries:
      - ref_id: A
        expr: up > [[ .missing ]]
    expressions:
      - ref_id: C
        type: math
        expression: "$A > 0"
    condition: C
    for: 1m
    severity: warning
    annotations:
      summary: s
`
		resp, err := svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), `map has no entry for key "missing"`)
	})

	t.Run("undeclared param in an expression", func(t *testing.T) {
		t.Parallel()
		yaml := `templates:
  - name: test_multi_missing_expression_param
    version: 1
    summary: s
    queries:
      - ref_id: A
        expr: up
    expressions:
      - ref_id: C
        type: math
        expression: "$A > [[ .missing ]]"
    condition: C
    for: 1m
    severity: warning
    annotations:
      summary: s
`
		resp, err := svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Contains(t, err.Error(), `map has no entry for key "missing"`)
	})
}

// TestListTemplatesUsesCache exercises the full path:
// CollectTemplates parses and converts each template once and caches the finished
// proto, and ListTemplates serves it straight from that cache (no per-request
// parsing or conversion). It covers both a multi-expression template
// (query/expression/condition populated) and a single-expression one (those fields
// empty) through the real service flow.
func TestListTemplatesUsesCache(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	enabled := true
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAlerting: &enabled})
	require.NoError(t, err)

	const singleExpressionYAML = `templates:
  - name: test_single_expression_list
    version: 1
    summary: Single expression
    expr: up == 0
    for: 5m
    severity: warning
    annotations:
      summary: s
      description: d
`

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "multi.yml"), []byte(multiExpressionWiringYAML), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "single.yml"), []byte(singleExpressionYAML), 0o600))

	svc, err := NewService(db, nil)
	require.NoError(t, err)
	svc.userTemplatesPath = dir
	svc.CollectTemplates(ctx)

	resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
	require.NoError(t, err)

	got := make(map[string]*alerting.Template, len(resp.Templates))
	for _, tmpl := range resp.Templates {
		got[tmpl.Name] = tmpl
	}

	// Multi-expression template: fields populated from the collect-time cache.
	multi := got["test_multi_expression_wiring"]
	require.NotNil(t, multi)
	require.Len(t, multi.Queries, 2)
	assert.Equal(t, "A", multi.Queries[0].RefId)
	assert.Equal(t, "B", multi.Queries[1].RefId)
	require.Len(t, multi.Expressions, 1)
	assert.Equal(t, "C", multi.Expressions[0].RefId)
	assert.Equal(t, "C", multi.Condition)

	// Single-expression template: no multi-expression fields.
	single := got["test_single_expression_list"]
	require.NotNil(t, single)
	assert.NotEmpty(t, single.Expr)
	assert.Empty(t, single.Queries)
	assert.Empty(t, single.Expressions)
	assert.Empty(t, single.Condition)
}

// TestListTemplatesSkipsUnparseableTemplate covers the log-and-skip branch of
// CollectTemplates. Collected YAML is always machine-generated by alert.ToYAML,
// so a re-parse failure is effectively unreachable in practice; we simulate the
// broken round-trip by corrupting the stored yaml column of a valid template.
// The template must still be listed (from its other columns) but without its
// multi-expression fields, its cached converted proto must likewise carry no
// multi-expression fields, and the rest of the collect (builtin templates) must survive.
func TestListTemplatesSkipsUnparseableTemplate(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	enabled := true
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAlerting: &enabled})
	require.NoError(t, err)

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	// Create a valid multi-expression template, then corrupt only its stored YAML
	// so it survives loadTemplatesFromDB but fails parseAlertTemplate at collect time.
	_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: multiExpressionWiringYAML})
	require.NoError(t, err)

	_, err = sqlDB.Exec("UPDATE alert_rule_templates SET yaml = $1 WHERE name = $2",
		"templates: [unclosed", "test_multi_expression_wiring")
	require.NoError(t, err)

	svc.CollectTemplates(ctx)

	resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
	require.NoError(t, err)

	var broken *alerting.Template
	for _, tmpl := range resp.Templates {
		if tmpl.Name == "test_multi_expression_wiring" {
			broken = tmpl
		}
	}

	// Still listed (from its non-YAML columns), but with no multi-expression fields.
	require.NotNil(t, broken, "template with unparseable YAML should still be listed")
	assert.Empty(t, broken.Queries)
	assert.Empty(t, broken.Expressions)
	assert.Empty(t, broken.Condition)

	// The broken template is cached in its converted form (that is how it stays
	// listed), but with no multi-expression fields because its YAML failed to parse.
	cached, ok := svc.convertedTemplates["test_multi_expression_wiring"]
	require.True(t, ok, "broken template must still be cached in converted form")
	assert.Empty(t, cached.Queries)
	assert.Empty(t, cached.Expressions)
	assert.Empty(t, cached.Condition)

	// The collect continued rather than aborting: builtin templates are still present.
	assert.Greater(t, len(resp.Templates), 1, "one bad template must not empty the whole list")
}

// TestListTemplatesConcurrentSharedProto guards the core assumption of the converted-proto
// cache: ListTemplates hands the SAME *alerting.Template pointers to every caller, so many
// concurrent requests marshal one shared, unmodified message. The protobuf runtime makes
// concurrent Marshal of an unmodified message safe; run this under -race to catch any
// accidental mutation of a cached proto that would break that guarantee.
func TestListTemplatesConcurrentSharedProto(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	enabled := true
	_, err := models.UpdateSettings(db, &models.ChangeSettingsParams{EnableAlerting: &enabled})
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "multi.yml"), []byte(multiExpressionWiringYAML), 0o600))

	svc, err := NewService(db, nil)
	require.NoError(t, err)
	svc.userTemplatesPath = dir
	svc.CollectTemplates(ctx)

	// Sanity check that the same pointer is served across calls (the shared-cache contract).
	first, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
	require.NoError(t, err)
	second, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, first.Templates)
	require.Same(t, first.Templates[0], second.Templates[0], "ListTemplates should serve shared cached protos")

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
			assert.NoError(t, err)
			for _, tmpl := range resp.Templates {
				_, err := proto.Marshal(tmpl)
				assert.NoError(t, err)
			}
		}()
	}
	wg.Wait()
}
