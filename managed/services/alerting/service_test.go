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
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	alerting "github.com/percona/pmm/api/alerting/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	testBadTemplates = "../../testdata/alerting-templates/bad"
	testTemplates    = "../../testdata/alerting-templates/user2"
	testTemplates2   = "../../testdata/alerting-templates/user"
)

func enableAlerting(t *testing.T, db *reform.DB) {
	t.Helper()

	// Ensure alerting is enabled for this sub-test
	_, err := db.Exec(`INSERT INTO settings (settings) VALUES ('{"alerting": {"enabled": true}}') ON CONFLICT DO NOTHING`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE settings SET settings = '{"alerting": {"enabled": true}}'`)
	require.NoError(t, err)
}

func disableAlerting(t *testing.T, db *reform.DB) {
	t.Helper()

	// Ensure alerting is enabled for this sub-test
	_, err := db.Exec(`INSERT INTO settings (settings) VALUES ('{"alerting": {"enabled": false}}') ON CONFLICT DO NOTHING`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE settings SET settings = '{"alerting": {"enabled": false}}'`)
	require.NoError(t, err)
}

func TestCollect(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	enableAlerting(t, db)

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

func TestListTemplates(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	t.Run("Disabled", func(t *testing.T) {
		disableAlerting(t, db)
		t.Cleanup(func() {
			enableAlerting(t, db)
		})

		resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
		require.Nil(t, resp)
		require.ErrorIs(t, err, status.Error(codes.FailedPrecondition, services.ErrAlertingDisabled.Error()))
	})

	t.Run("Overflow", func(t *testing.T) {
		t.Run("SeverityOverflow", func(t *testing.T) {
			svc.rw.Lock()
			svc.templates = map[string]models.Template{
				"overflow": {
					Name:     "overflow",
					Severity: math.MaxInt32 + 1,
				},
			}
			svc.rw.Unlock()

			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
			assert.Nil(t, resp)
			require.ErrorContains(t, err, "alerting template (name=overflow) severity 2147483648 is out of range for int32")
		})

		t.Run("SeverityUnderflow", func(t *testing.T) {
			svc.rw.Lock()
			svc.templates = map[string]models.Template{
				"underflow": {
					Name:     "underflow",
					Severity: math.MinInt32 - 1,
				},
			}
			svc.rw.Unlock()

			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
			assert.Nil(t, resp)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "alerting template (name=underflow) severity -2147483649 is out of range for int32")
		})
	})

	t.Run("Pagination", func(t *testing.T) {
		svc.rw.Lock()
		svc.templates = map[string]models.Template{
			"t1": {Name: "t1"},
			"t2": {Name: "t2"},
			"t3": {Name: "t3"},
		}
		svc.rw.Unlock()

		t.Run("NegativePagination", func(t *testing.T) {
			_, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{
				PageIndex: new(int32(-1)),
			})
			require.ErrorIs(t, err, status.Errorf(codes.InvalidArgument, "Page index must be non-negative"))

			_, err = svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{
				PageSize: new(int32(-1)),
			})
			require.ErrorIs(t, err, status.Errorf(codes.InvalidArgument, "Page size must be non-negative"))
		})

		t.Run("PageSizeZero", func(t *testing.T) {
			// PageSize 0 is handled as "return all" in the service logic
			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{
				PageSize: new(int32(0)),
			})
			require.NoError(t, err)
			assert.Equal(t, int32(3), resp.TotalItems)
			assert.Len(t, resp.Templates, 3)
		})

		t.Run("FullPage", func(t *testing.T) {
			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{
				PageSize: new(int32(2)),
			})
			require.NoError(t, err)
			assert.Equal(t, int32(2), resp.TotalPages)
			assert.Equal(t, int32(3), resp.TotalItems)
			assert.Len(t, resp.Templates, 2)
		})

		t.Run("LastPage", func(t *testing.T) {
			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{
				PageSize:  new(int32(2)),
				PageIndex: new(int32(1)),
			})
			require.NoError(t, err)
			assert.Equal(t, int32(2), resp.TotalPages)
			assert.Len(t, resp.Templates, 1)
			assert.Equal(t, "t3", resp.Templates[0].Name)
		})

		t.Run("PageIndexOutOfRange", func(t *testing.T) {
			resp, err := svc.ListTemplates(ctx, &alerting.ListTemplatesRequest{
				PageIndex: new(int32(10)),
				PageSize:  new(int32(2)),
			})
			require.NoError(t, err)
			assert.Empty(t, resp.Templates)
			assert.Equal(t, int32(3), resp.TotalItems)
		})
	})

	t.Run("Reload", func(t *testing.T) {
		enableAlerting(t, db)

		// Create a new service instance to avoid interference with the previous test's templates.
		// It changes the templates in memory that affects the other tests.
		// Need to test the reload functionality in isolation.
		svcR, errR := NewService(db, nil)
		require.NoError(t, errR)

		svcR.rw.Lock()
		svcR.templates = map[string]models.Template{
			"t1": {Name: "t1"},
			"t2": {Name: "t2"},
			"t3": {Name: "t3"},
		}
		svcR.rw.Unlock()

		// Inject a manual template into memory
		svcR.rw.Lock()
		svcR.templates = map[string]models.Template{"manual_entry": {Name: "manual_entry"}}
		svcR.rw.Unlock()

		// Point to a directory with known templates
		svcR.userTemplatesPath = testTemplates2

		// Verify it's there without reload
		resp, err := svcR.ListTemplates(ctx, &alerting.ListTemplatesRequest{})
		require.NoError(t, err)
		assert.Len(t, resp.Templates, 1)
		assert.Equal(t, "manual_entry", resp.Templates[0].Name)

		// Trigger reload
		resp, err = svcR.ListTemplates(ctx, &alerting.ListTemplatesRequest{Reload: true})
		require.NoError(t, err)

		// The manual entry should be gone, replaced by templates from testTemplates2 and built-ins
		assert.Greater(t, resp.TotalItems, int32(1))
		foundManual := false
		for _, tmpl := range resp.Templates {
			if tmpl.Name == "manual_entry" {
				foundManual = true
			}
		}
		assert.False(t, foundManual)
	})
}

func TestCreateTemplate(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	enableAlerting(t, db)

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	t.Run("Disabled", func(t *testing.T) {
		disableAlerting(t, db)

		t.Cleanup(func() {
			enableAlerting(t, db)
		})

		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: "templates: []"})
		require.ErrorIs(t, err, status.Error(codes.FailedPrecondition, services.ErrAlertingDisabled.Error()))
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: "invalid: ["})
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "yaml: line 1: did not find expected node content"))
	})

	t.Run("DuplicateNamesInRequest", func(t *testing.T) {
		const yaml = `
templates:
  - name: dupl_req_template
    expr: 'avg(cpu_usage) > [[ .threshold ]]'
    params:
      - name: threshold
        summary: CPU Usage Threshold
        type: float
        range:
          - 0
          - 100
        value: 80
    for: 1m
    version: 1
    summary: High CPU Usage
    severity: critical
    labels:
      template_label: static
      dynamic_label: '[[ .threshold ]]'
    annotations:
      summary: 'High CPU: [[ .threshold ]]'
      description: 'Value is {{ $value }}'
  - name: dupl_req_template
    expr: 'avg(cpu_usage) > [[ .threshold ]]'
    params:
      - name: threshold
        summary: CPU Usage Threshold
        type: float
        range:
          - 0
          - 100
        value: 80
    for: 1m
    version: 1
    summary: High CPU Usage
    severity: critical
    labels:
      template_label: static
      dynamic_label: '[[ .threshold ]]'
    annotations:
      summary: 'High CPU: [[ .threshold ]]'
      description: 'Value is {{ $value }}'
`
		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "Template with name 'dupl_req_template' declared more than once."))
	})

	t.Run("DuplicateTemplate", func(t *testing.T) {
		const yaml = `
templates:
  - name: dupl_test_template
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`
		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.NoError(t, err)
		assert.Contains(t, svc.GetTemplates(), "dupl_test_template")

		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.ErrorIs(t, err, status.Error(codes.AlreadyExists, "Template with name \"dupl_test_template\" already exists."))
	})

	t.Run("ForbiddenPrefix", func(t *testing.T) {
		const yaml = `
templates:
  - name: pmm_system_rule
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`
		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "pmm_system_rule: template name should not start with 'pmm_' or 'saas_' prefix"))
	})

	t.Run("MissingParam", func(t *testing.T) {
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
			"placeholders: template: :4:5: executing \"\" at <.threshold>: map has no entry for key \"threshold\"")
	})

	t.Run("Success", func(t *testing.T) {
		const yaml = `
templates:
  - name: create_test_template
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`

		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.NoError(t, err)
		assert.Contains(t, svc.GetTemplates(), "create_test_template")
	})
}

func TestUpdateTemplate(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	enableAlerting(t, db)

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	t.Run("Disabled", func(t *testing.T) {
		disableAlerting(t, db)

		t.Cleanup(func() {
			enableAlerting(t, db)
		})

		_, err = svc.UpdateTemplate(ctx, &alerting.UpdateTemplateRequest{Name: "crud_test_template", Yaml: "templates: []"})
		require.ErrorIs(t, err, status.Error(codes.FailedPrecondition, services.ErrAlertingDisabled.Error()))
	})

	t.Run("EmptyTemplate", func(t *testing.T) {
		_, err = svc.UpdateTemplate(ctx, &alerting.UpdateTemplateRequest{Name: "empty_internal_name", Yaml: "templates: []"})
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "Request should contain exactly one rule template"))
	})

	t.Run("NotFound", func(t *testing.T) {
		const yaml = `
templates:
  - name: not_found_internal_name
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`
		_, err = svc.UpdateTemplate(ctx, &alerting.UpdateTemplateRequest{Name: "not_found_internal_name", Yaml: yaml})
		require.ErrorIs(t, err, status.Error(codes.NotFound, "Template with name \"not_found_internal_name\" not found."))
	})

	t.Run("MultipleTemplatesError", func(t *testing.T) {
		const yaml = `
templates:
  - name: t1
    expr: metric > 1
  - name: t2
    expr: metric > 2`
		_, err = svc.UpdateTemplate(ctx, &alerting.UpdateTemplateRequest{Name: "crud_test_template", Yaml: yaml})
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "Failed to parse rule template"))
	})

	t.Run("MissingParam", func(t *testing.T) {
		t.Parallel()

		const validTemplate = `
---
templates: 
  - name: valid_template_11
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

	t.Run("Success", func(t *testing.T) {
		const yaml = `
templates:
  - name: update_test_template
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`

		const newYaml = `
templates:
  - name: update_test_template
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 5m
    version: 1
    summary: "Updated Summary Text"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`

		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.NoError(t, err)
		assert.Contains(t, svc.GetTemplates(), "update_test_template")

		_, err = svc.UpdateTemplate(ctx, &alerting.UpdateTemplateRequest{
			Name: "update_test_template",
			Yaml: newYaml,
		})
		require.NoError(t, err)

		templates := svc.GetTemplates()
		assert.Contains(t, templates, "update_test_template")
		assert.Equal(t, "Updated Summary Text", templates["update_test_template"].Summary)
	})
}

func TestDeleteTemplate(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	enableAlerting(t, db)

	svc, err := NewService(db, nil)
	require.NoError(t, err)

	t.Run("Disabled", func(t *testing.T) {
		disableAlerting(t, db)

		t.Cleanup(func() {
			enableAlerting(t, db)
		})

		_, err = svc.DeleteTemplate(ctx, &alerting.DeleteTemplateRequest{Name: "updated_internal_name"})
		require.ErrorIs(t, err, status.Error(codes.FailedPrecondition, services.ErrAlertingDisabled.Error()))
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err = svc.DeleteTemplate(ctx, &alerting.DeleteTemplateRequest{Name: "not_found_internal_name"})
		require.ErrorIs(t, err, status.Error(codes.NotFound, "Template with name \"not_found_internal_name\" not found."))
	})

	t.Run("Success", func(t *testing.T) {
		const yaml = `
templates:
  - name: delete_test_template
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`

		_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
		require.NoError(t, err)
		assert.Contains(t, svc.GetTemplates(), "delete_test_template")

		_, err = svc.DeleteTemplate(ctx, &alerting.DeleteTemplateRequest{Name: "delete_test_template"})
		require.NoError(t, err)
		assert.NotContains(t, svc.GetTemplates(), "delete_test_template")
	})
}

func TestCreateRule(t *testing.T) {
	ctx := t.Context()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)
	enableAlerting(t, db)

	mockGrafana := &mockGrafanaClient{}
	svc, err := NewService(db, mockGrafana)
	require.NoError(t, err)

	// Setup a base template for rule testing
	const yaml = `
templates:
  - name: rule_logic_template
    expr: avg(cpu_usage) > [[ .threshold ]]
    params:
      - name: threshold
        summary: "CPU Usage Threshold"
        type: float
        range: [0, 100]
        value: 80
    for: 1m
    version: 1
    summary: "High CPU Usage"
    severity: critical
    labels:
      template_label: "static"
      dynamic_label: "[[ .threshold ]]"
    annotations:
      summary: "High CPU: [[ .threshold ]]"
      description: "Value is {{ $value }}"`

	_, err = svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{Yaml: yaml})
	require.NoError(t, err)

	t.Run("InputValidation", func(t *testing.T) {
		type testCase struct {
			name string
			req  *alerting.CreateRuleRequest
			err  string
		}
		cases := []testCase{
			{"NoTemplate", &alerting.CreateRuleRequest{}, "Template name should be specified"},
			{"NoFolder", &alerting.CreateRuleRequest{TemplateName: "t"}, "Folder UID should be specified"},
			{"NoGroup", &alerting.CreateRuleRequest{TemplateName: "t", FolderUid: "f"}, "Rule group name should be specified"},
			{"CountMismatch", &alerting.CreateRuleRequest{
				TemplateName: "rule_logic_template",
				FolderUid:    "f",
				Group:        "g",
				Params:       []*alerting.ParamValue{},
			},
				"Expression requires 1 parameters, but got 0.",
			},
			{"ParamsMismatch", &alerting.CreateRuleRequest{
				TemplateName: "rule_logic_template",
				FolderUid:    "f",
				Group:        "g",
				Params: []*alerting.ParamValue{
					{Name: "threshold",
						Value: &alerting.ParamValue_Float{
							Float: 101,
						},
						Type: alerting.ParamType_PARAM_TYPE_FLOAT,
					}},
			},
				"Parameter threshold value is greater than required maximum."},
		}
		mockGrafana.On("GetDatasourceUIDByName", mock.Anything, "Metrics").Return("ds-123", nil).Maybe()
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// Setup expectation as optional to prevent panics on unexpected validation paths

				_, err := svc.CreateRule(ctx, tc.req)
				require.ErrorIs(t, err, status.Error(codes.InvalidArgument, tc.err))
			})
		}
		mockGrafana.AssertExpectations(t)

		cases = []testCase{
			{"UnknownTemplate", &alerting.CreateRuleRequest{TemplateName: "ghost", FolderUid: "f", Group: "g"}, "Template with name \"ghost\" not found."},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.CreateRule(ctx, tc.req)
				require.ErrorIs(t, err, status.Error(codes.NotFound, tc.err))
			})
		}
	})

	t.Run("GrafanaIntegrationErrors", func(t *testing.T) {
		t.Run("DatasourceLookupFail", func(t *testing.T) {
			mockGrafana.On("GetDatasourceUIDByName", mock.Anything, "Metrics").
				Return("", errors.New("grafana connection refused"))
			_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
				TemplateName: "rule_logic_template",
				FolderUid:    "f",
				Group:        "g",
			})
			require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "Expression requires 1 parameters, but got 0."))
			mockGrafana.AssertExpectations(t)
		})

		t.Run("CreateRuleFail", func(t *testing.T) {
			mockGrafana.On("GetDatasourceUIDByName", mock.Anything, "Metrics").
				Return("ds-123", nil)

			_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
				TemplateName: "rule_logic_template",
				FolderUid:    "f",
				Group:        "g",
			})
			require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "Expression requires 1 parameters, but got 0."))
			mockGrafana.AssertExpectations(t)
		})
	})

	t.Run("Success", func(t *testing.T) {
		var capturedRule *services.Rule
		mockGrafana.On("GetDatasourceUIDByName", mock.Anything, "Metrics").
			Return("ds-123", nil)
		mockGrafana.On("CreateAlertRule", mock.Anything, "my-folder", "my-group", "30s", mock.MatchedBy(func(r *services.Rule) bool {
			capturedRule = r
			return true
		})).
			Return(nil)

		_, err := svc.CreateRule(ctx, &alerting.CreateRuleRequest{
			Name:         "Complex Rule",
			TemplateName: "rule_logic_template",
			FolderUid:    "my-folder",
			Group:        "my-group",
			Interval:     durationpb.New(30 * time.Second),
			For:          durationpb.New(2 * time.Minute),
			Params: []*alerting.ParamValue{
				{Name: "threshold", Value: &alerting.ParamValue_Float{Float: 92.5}, Type: alerting.ParamType_PARAM_TYPE_FLOAT},
			},
			Filters: []*alerting.Filter{
				{Type: alerting.FilterType_FILTER_TYPE_MATCH, Label: "service", Regexp: "web"},
				{Type: alerting.FilterType_FILTER_TYPE_MISMATCH, Label: "env", Regexp: "test"},
			},
			CustomLabels: map[string]string{"user_label": "val"},
			Severity:     managementv1.Severity(common.Critical),
		})

		require.NoError(t, err)
		require.NotNil(t, capturedRule)
		assert.Equal(t, "Complex Rule", capturedRule.GrafanaAlert.Title)
		assert.Equal(t, "label_mismatch(label_match(avg(cpu_usage) > 92.5, \"service\", \"web\"), \"env\", \"test\")", capturedRule.GrafanaAlert.Data[0].Model.Expr)
		assert.Equal(t, "2m0s", capturedRule.For)
		assert.Equal(t, "val", capturedRule.Labels["user_label"])
		assert.Equal(t, "92.5", capturedRule.Labels["dynamic_label"])
		assert.Equal(t, "High CPU: 92.5", capturedRule.Annotations["summary"])
		mockGrafana.AssertExpectations(t)
	})
}
