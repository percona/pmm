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
	"context"
	"os"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	alerting "github.com/percona/pmm/api/alerting/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/platform"
	"github.com/percona/pmm/managed/utils/testdb"
)

const (
	devPlatformAddress   = "https://check-dev.percona.com"
	devPlatformPublicKey = "RWTg+ZmCCjt7O8eWeAmTLAqW+1ozUbpRSKSwNTmO+exlS5KEIPYWuYdX"
	testBadTemplates     = "../../../testdata/alerting/bad"
	testTemplates        = "../../../testdata/alerting/user2"
	testTemplates2       = "../../../testdata/alerting/user"
	issuerURL            = "https://id-dev.percona.com/oauth2/aus15pi5rjdtfrcH51d7/v1"
)

func TestCollect(t *testing.T) {
	t.Parallel()
	clientID, clientSecret := os.Getenv("PMM_DEV_OAUTH_CLIENT_ID"), os.Getenv("PMM_DEV_OAUTH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Skip("Environment variables PMM_DEV_OAUTH_CLIENT_ID / PMM_DEV_OAUTH_CLIENT_SECRET are not defined, skipping test")
	}

	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	platformClient, err := platform.NewClient(db, devPlatformAddress)
	require.NoError(t, err)

	insertSSODetails := &models.PerconaSSODetailsInsert{
		IssuerURL:              issuerURL,
		PMMManagedClientID:     clientID,
		PMMManagedClientSecret: clientSecret,
		Scope:                  "percona",
	}
	err = models.InsertPerconaSSODetails(db.Querier, insertSSODetails)
	require.NoError(t, err)

	t.Run("builtin are valid", func(t *testing.T) {
		t.Parallel()

		svc, err := NewService(db, platformClient, nil)
		require.NoError(t, err)
		_, err = svc.loadTemplatesFromAssets(ctx)
		require.NoError(t, err)
	})

	t.Run("bad template paths", func(t *testing.T) {
		t.Parallel()

		svc, err := NewService(db, platformClient, nil)
		require.NoError(t, err)
		svc.userTemplatesPath = testBadTemplates
		templates, err := svc.loadTemplatesFromUserFiles(ctx)
		assert.Empty(t, templates)
		require.NoError(t, err)
	})

	t.Run("valid template paths", func(t *testing.T) {
		t.Parallel()

		svc, err := NewService(db, platformClient, nil)
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

func TestDownloadTemplates(t *testing.T) {
	clientID, clientSecret := os.Getenv("PMM_DEV_OAUTH_CLIENT_ID"), os.Getenv("PMM_DEV_OAUTH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		t.Skip("Environment variables PMM_DEV_OAUTH_CLIENT_ID / PMM_DEV_OAUTH_CLIENT_SECRET are not defined, skipping test")
	}

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	platformClient, err := platform.NewClient(db, devPlatformAddress)
	require.NoError(t, err)

	svc, err := NewService(db, platformClient, nil)
	svc.platformPublicKeys = []string{devPlatformPublicKey}
	require.NoError(t, err)

	insertSSODetails := &models.PerconaSSODetailsInsert{
		IssuerURL:              issuerURL,
		PMMManagedClientID:     clientID,
		PMMManagedClientSecret: clientSecret,
		Scope:                  "percona",
	}
	err = models.InsertPerconaSSODetails(db.Querier, insertSSODetails)
	require.NoError(t, err)

	t.Run("normal", func(t *testing.T) {
		assert.Empty(t, svc.GetTemplates())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		templates, err := svc.downloadTemplates(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, templates)
		assert.NotEmpty(t, svc.GetTemplates())
	})

	t.Run("with disabled telemetry", func(t *testing.T) {
		_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{
			EnableTelemetry: pointer.ToBool(false),
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		templates, err := svc.downloadTemplates(ctx)
		require.NoError(t, err)
		assert.Empty(t, templates)
		assert.Empty(t, svc.GetTemplates())
	})
}

func TestTemplateValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	platformClient, err := platform.NewClient(db, devPlatformAddress)
	require.NoError(t, err)

	t.Run("create a template with missing param", func(t *testing.T) {
		t.Parallel()

		const templateWithMissingParam = `
---
templates: 
  - name: template_with_missing_param_1
    version: 1
    summary: Template with missing param 1
    tiers: [anonymous, registered]
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

		svc, err := NewService(db, platformClient, nil)
		require.NoError(t, err)
		resp, err := svc.CreateTemplate(ctx, &alerting.CreateTemplateRequest{
			Yaml: templateWithMissingParam,
		})
		assert.Nil(t, resp)
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = failed to fill expression "+
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
    tiers: [anonymous, registered]
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
    tiers: [anonymous, registered]
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

		svc, err := NewService(db, platformClient, nil)
		require.NoError(t, err)
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
		assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = failed to fill expression "+
			"placeholders: template: :4:5: executing \"\" at <.threshold>: map has no entry for key \"threshold\".")
	})
}
