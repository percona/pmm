// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package alertmanager

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/percona-platform/saas/pkg/alert"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/percona/promconfig"
	"github.com/percona/promconfig/alertmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestIsReady(t *testing.T) {
	New(nil).GenerateBaseConfigs() // this method should not use database

	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	svc := New(db)

	assert.NoError(t, svc.updateConfiguration(ctx))
	assert.NoError(t, svc.IsReady(ctx))
}

// marshalAndValidate populates, marshals and validates config.
func marshalAndValidate(t *testing.T, svc *Service, base *alertmanager.Config) string {
	b, err := svc.marshalConfig(base)
	require.NoError(t, err)

	t.Logf("config:\n%s", b)

	err = svc.validateConfig(context.Background(), b)
	require.NoError(t, err)
	return string(b)
}

func TestPopulateConfig(t *testing.T) {
	New(nil).GenerateBaseConfigs() // this method should not use database

	t.Run("without receivers and routes", func(t *testing.T) {
		tests.SetTestIDReader(t)
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		svc := New(db)

		cfg := svc.loadBaseConfig()
		cfg.Global = &alertmanager.GlobalConfig{
			SlackAPIURL: "https://hooks.slack.com/services/abc/123/xyz",
		}

		actual := marshalAndValidate(t, svc, cfg)
		expected := strings.TrimSpace(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
    resolve_timeout: 0s
    smtp_require_tls: false
    slack_api_url: https://hooks.slack.com/services/abc/123/xyz
route:
    receiver: empty
    continue: false
receivers:
    - name: empty
    - name: disabled
templates: []
		`) + "\n"
		assert.Equal(t, expected, actual, "actual:\n%s", actual)
	})

	t.Run("with receivers and routes", func(t *testing.T) {
		tests.SetTestIDReader(t)
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		svc := New(db)

		channel1, err := models.CreateChannel(db.Querier, &models.CreateChannelParams{
			Summary: "channel1",
			EmailConfig: &models.EmailConfig{
				To: []string{"test@test.test", "test2@test.test"},
			},
			Disabled: false,
		})
		require.NoError(t, err)

		channel2, err := models.CreateChannel(db.Querier, &models.CreateChannelParams{
			Summary: "channel2",
			PagerDutyConfig: &models.PagerDutyConfig{
				RoutingKey: "ms-pagerduty-dev",
			},
			Disabled: false,
		})
		require.NoError(t, err)

		channel3, err := models.CreateChannel(db.Querier, &models.CreateChannelParams{
			Summary: "channel3",
			PagerDutyConfig: &models.PagerDutyConfig{
				RoutingKey: "ms-pagerduty-dev",
			},
			Disabled: true,
		})
		require.NoError(t, err)

		_, err = models.CreateTemplate(db.Querier, &models.CreateTemplateParams{
			Template: &alert.Template{
				Name:    "test_template",
				Version: 1,
				Summary: "summary",
				Tiers:   []common.Tier{common.Anonymous},
				Expr:    "expr",
				Params: []alert.Parameter{{
					Name:    "param",
					Summary: "param summary",
					Unit:    "%",
					Type:    alert.Float,
					Range:   []interface{}{float64(10), float64(100)},
					Value:   float64(50),
				}},
				For:         promconfig.Duration(3 * time.Second),
				Severity:    common.Warning,
				Labels:      map[string]string{"foo": "bar"},
				Annotations: nil,
			},
			Source: "USER_FILE",
		})
		require.NoError(t, err)

		rule1, err := models.CreateRule(db.Querier, &models.CreateRuleParams{
			TemplateName: "test_template",
			Disabled:     true,
			RuleParams: []models.RuleParam{{
				Name:       "test",
				Type:       models.Float,
				FloatValue: 3.14,
			}},
			For:      5 * time.Second,
			Severity: models.Severity(common.Warning),
			CustomLabels: map[string]string{
				"foo": "bar",
			},
			Filters: []models.Filter{{
				Type: models.Equal,
				Key:  "service_name",
				Val:  "mysql1",
			}},
			ChannelIDs: []string{channel1.ID, channel2.ID},
		})
		require.NoError(t, err)

		// create another rule with same channelIDs to check for redundant receivers.
		rule2, err := models.CreateRule(db.Querier, &models.CreateRuleParams{
			TemplateName: "test_template",
			Disabled:     true,
			RuleParams: []models.RuleParam{{
				Name:       "test",
				Type:       models.Float,
				FloatValue: 3.14,
			}},
			For:      5 * time.Second,
			Severity: models.Severity(common.Warning),
			CustomLabels: map[string]string{
				"foo": "baz",
			},
			Filters: []models.Filter{{
				Type: models.Equal,
				Key:  "service_name",
				Val:  "mysql2",
			}},
			ChannelIDs: []string{channel1.ID, channel2.ID, channel3.ID},
		})
		require.NoError(t, err)

		// create another rule without channelID and check if it is absent in the config.
		_, err = models.CreateRule(db.Querier, &models.CreateRuleParams{
			TemplateName: "test_template",
			Disabled:     true,
			RuleParams: []models.RuleParam{{
				Name:       "test",
				Type:       models.Float,
				FloatValue: 3.14,
			}},
			For:      5 * time.Second,
			Severity: models.Severity(common.Warning),
			CustomLabels: map[string]string{
				"foo": "baz",
			},
		})
		require.NoError(t, err)

		// CreateRule with disabled channel
		rule4, err := models.CreateRule(db.Querier, &models.CreateRuleParams{
			TemplateName: "test_template",
			Disabled:     true,
			RuleParams: []models.RuleParam{{
				Name:       "test",
				Type:       models.Float,
				FloatValue: 3.14,
			}},
			Filters: []models.Filter{{
				Type: models.Equal,
				Key:  "service_name",
				Val:  "mysql3",
			}},
			For:      5 * time.Second,
			Severity: models.Severity(common.Warning),
			CustomLabels: map[string]string{
				"foo": "baz",
			},
			ChannelIDs: []string{channel3.ID},
		})
		require.NoError(t, err)

		_, err = models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{
			EmailAlertingSettings: &models.EmailAlertingSettings{
				From:      "from@test.com",
				Smarthost: "1.2.3.4:80",
				Hello:     "host",
				Username:  "user",
				Password:  "password",
				Identity:  "id",
				Secret:    "secret",
			},
			SlackAlertingSettings: &models.SlackAlertingSettings{
				URL: "https://hooks.slack.com/services/abc/456/xyz",
			},
		})
		require.NoError(t, err)

		actual := marshalAndValidate(t, svc, svc.loadBaseConfig())
		expected := strings.TrimSpace(fmt.Sprintf(`
# Managed by pmm-managed. DO NOT EDIT.
---
global:
    resolve_timeout: 0s
    smtp_from: from@test.com
    smtp_hello: host
    smtp_smarthost: 1.2.3.4:80
    smtp_auth_username: user
    smtp_auth_password: password
    smtp_auth_secret: secret
    smtp_auth_identity: id
    smtp_require_tls: false
    slack_api_url: https://hooks.slack.com/services/abc/456/xyz
route:
    receiver: empty
    continue: false
    routes:
        - receiver: %[1]s + %[2]s
          match:
            rule_id: %[3]s
            service_name: mysql1
          continue: false
        - receiver: %[1]s + %[2]s
          match:
            rule_id: %[4]s
            service_name: mysql2
          continue: false
        - receiver: disabled
          match:
            rule_id: %[5]s
            service_name: mysql3
          continue: false
receivers:
    - name: empty
    - name: disabled
    - name: %[1]s + %[2]s
      email_configs:
        - send_resolved: false
          to: test@test.test
        - send_resolved: false
          to: test2@test.test
      pagerduty_configs:
        - send_resolved: false
          routing_key: ms-pagerduty-dev
          description: '[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]{{ range .Alerts -}}{{ if .Labels.severity }}[{{ .Labels.severity | toUpper }}]{{ end }} {{ .Annotations.summary }}{{ end }}'
          details:
            firing: |-
                {{ range .Alerts -}}
                Alert: {{ if .Labels.severity }}[{{ .Labels.severity | toUpper }}]{{ end }} {{ .Annotations.summary }}
                Description: {{ .Annotations.description }}
                Details:
                {{ if .Labels.node_name }}  - node_name: {{ .Labels.node_name }}
                {{ end }}{{ if .Labels.node_id }}  - node_id: {{ .Labels.node_id }}
                {{ end }}{{ if .Labels.service_name }}  - service_name: {{ .Labels.service_name }}
                {{ end }}{{ if .Labels.service_id }}  - service_id: {{ .Labels.service_id }}
                {{ end }}{{ if .Labels.service_type }}  - service_type: {{ .Labels.service_type }}
                {{ end }}{{ if .Labels.rule_id }}  - rule_id: {{ .Labels.rule_id }}
                {{ end }}{{ if .Labels.alertgroup }}  - alertgroup: {{ .Labels.alertgroup }}
                {{ end }}{{ if .Labels.template_name }}  - template_name: {{ .Labels.template_name }}
                {{ end }}{{ if .Labels.severity }}  - severity: {{ .Labels.severity }}
                {{ end }}{{ if .Labels.agent_id }}  - agent_id: {{ .Labels.agent_id }}
                {{ end }}{{ if .Labels.agent_type }}  - agent_type: {{ .Labels.agent_type }}
                {{ end }}{{ if .Labels.job }}  - job: {{ .Labels.job }}
                {{ end }}

                {{ end }}
templates: []
`, channel1.ID, channel2.ID, rule1.ID, rule2.ID, rule4.ID)) + "\n"
		assert.Equal(t, expected, actual, "actual:\n%s", actual)
	})
}

func TestGenerateReceivers(t *testing.T) {
	t.Parallel()

	chanMap := map[string]*models.Channel{
		"1": {
			ID:   "1",
			Type: models.Slack,
			SlackConfig: &models.SlackConfig{
				Channel: "channel1",
			},
		},
		"2": {
			ID:   "2",
			Type: models.Slack,
			SlackConfig: &models.SlackConfig{
				Channel: "channel2",
			},
		},
		"3": {
			ID:   "3",
			Type: models.Slack,
			SlackConfig: &models.SlackConfig{
				Channel: "channel3",
			},
			Disabled: true,
		},
	}
	recvSet := map[string]models.ChannelIDs{
		"1":   {"1"},
		"2":   {"2"},
		"1+2": {"1", "2"},
	}
	s := New(nil)
	actualR, err := s.generateReceivers(chanMap, recvSet)
	require.NoError(t, err)
	actual, err := yaml.Marshal(actualR)
	require.NoError(t, err)

	expected := strings.TrimSpace(`
- name: "1"
  slack_configs:
    - send_resolved: false
      channel: channel1
      title: '[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]'
      text: |-
        {{ range .Alerts -}}
        *Alert:* {{ if .Labels.severity }}`+"`{{ .Labels.severity | toUpper }}`"+`{{ end }} {{ .Annotations.summary }}
        *Description:* {{ .Annotations.description }}
        *Details:*
        {{ if .Labels.node_name }}     • *node_name:* `+"`{{ .Labels.node_name }}`"+`
        {{ end }}{{ if .Labels.node_id }}     • *node_id:* `+"`{{ .Labels.node_id }}`"+`
        {{ end }}{{ if .Labels.service_name }}     • *service_name:* `+"`{{ .Labels.service_name }}`"+`
        {{ end }}{{ if .Labels.service_id }}     • *service_id:* `+"`{{ .Labels.service_id }}`"+`
        {{ end }}{{ if .Labels.service_type }}     • *service_type:* `+"`{{ .Labels.service_type }}`"+`
        {{ end }}{{ if .Labels.rule_id }}     • *rule_id:* `+"`{{ .Labels.rule_id }}`"+`
        {{ end }}{{ if .Labels.alertgroup }}     • *alertgroup:* `+"`{{ .Labels.alertgroup }}`"+`
        {{ end }}{{ if .Labels.template_name }}     • *template_name:* `+"`{{ .Labels.template_name }}`"+`
        {{ end }}{{ if .Labels.severity }}     • *severity:* `+"`{{ .Labels.severity }}`"+`
        {{ end }}{{ if .Labels.agent_id }}     • *agent_id:* `+"`{{ .Labels.agent_id }}`"+`
        {{ end }}{{ if .Labels.agent_type }}     • *agent_type:* `+"`{{ .Labels.agent_type }}`"+`
        {{ end }}{{ if .Labels.job }}     • *job:* `+"`{{ .Labels.job }}`"+`
        {{ end }}

        {{ end }}
      short_fields: false
      link_names: false
- name: 1+2
  slack_configs:
    - send_resolved: false
      channel: channel1
      title: '[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]'
      text: |-
        {{ range .Alerts -}}
        *Alert:* {{ if .Labels.severity }}`+"`{{ .Labels.severity | toUpper }}`"+`{{ end }} {{ .Annotations.summary }}
        *Description:* {{ .Annotations.description }}
        *Details:*
        {{ if .Labels.node_name }}     • *node_name:* `+"`{{ .Labels.node_name }}`"+`
        {{ end }}{{ if .Labels.node_id }}     • *node_id:* `+"`{{ .Labels.node_id }}`"+`
        {{ end }}{{ if .Labels.service_name }}     • *service_name:* `+"`{{ .Labels.service_name }}`"+`
        {{ end }}{{ if .Labels.service_id }}     • *service_id:* `+"`{{ .Labels.service_id }}`"+`
        {{ end }}{{ if .Labels.service_type }}     • *service_type:* `+"`{{ .Labels.service_type }}`"+`
        {{ end }}{{ if .Labels.rule_id }}     • *rule_id:* `+"`{{ .Labels.rule_id }}`"+`
        {{ end }}{{ if .Labels.alertgroup }}     • *alertgroup:* `+"`{{ .Labels.alertgroup }}`"+`
        {{ end }}{{ if .Labels.template_name }}     • *template_name:* `+"`{{ .Labels.template_name }}`"+`
        {{ end }}{{ if .Labels.severity }}     • *severity:* `+"`{{ .Labels.severity }}`"+`
        {{ end }}{{ if .Labels.agent_id }}     • *agent_id:* `+"`{{ .Labels.agent_id }}`"+`
        {{ end }}{{ if .Labels.agent_type }}     • *agent_type:* `+"`{{ .Labels.agent_type }}`"+`
        {{ end }}{{ if .Labels.job }}     • *job:* `+"`{{ .Labels.job }}`"+`
        {{ end }}

        {{ end }}
      short_fields: false
      link_names: false
    - send_resolved: false
      channel: channel2
      title: '[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]'
      text: |-
        {{ range .Alerts -}}
        *Alert:* {{ if .Labels.severity }}`+"`{{ .Labels.severity | toUpper }}`"+`{{ end }} {{ .Annotations.summary }}
        *Description:* {{ .Annotations.description }}
        *Details:*
        {{ if .Labels.node_name }}     • *node_name:* `+"`{{ .Labels.node_name }}`"+`
        {{ end }}{{ if .Labels.node_id }}     • *node_id:* `+"`{{ .Labels.node_id }}`"+`
        {{ end }}{{ if .Labels.service_name }}     • *service_name:* `+"`{{ .Labels.service_name }}`"+`
        {{ end }}{{ if .Labels.service_id }}     • *service_id:* `+"`{{ .Labels.service_id }}`"+`
        {{ end }}{{ if .Labels.service_type }}     • *service_type:* `+"`{{ .Labels.service_type }}`"+`
        {{ end }}{{ if .Labels.rule_id }}     • *rule_id:* `+"`{{ .Labels.rule_id }}`"+`
        {{ end }}{{ if .Labels.alertgroup }}     • *alertgroup:* `+"`{{ .Labels.alertgroup }}`"+`
        {{ end }}{{ if .Labels.template_name }}     • *template_name:* `+"`{{ .Labels.template_name }}`"+`
        {{ end }}{{ if .Labels.severity }}     • *severity:* `+"`{{ .Labels.severity }}`"+`
        {{ end }}{{ if .Labels.agent_id }}     • *agent_id:* `+"`{{ .Labels.agent_id }}`"+`
        {{ end }}{{ if .Labels.agent_type }}     • *agent_type:* `+"`{{ .Labels.agent_type }}`"+`
        {{ end }}{{ if .Labels.job }}     • *job:* `+"`{{ .Labels.job }}`"+`
        {{ end }}

        {{ end }}
      short_fields: false
      link_names: false
- name: "2"
  slack_configs:
    - send_resolved: false
      channel: channel2
      title: '[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}]'
      text: |-
        {{ range .Alerts -}}
        *Alert:* {{ if .Labels.severity }}`+"`{{ .Labels.severity | toUpper }}`"+`{{ end }} {{ .Annotations.summary }}
        *Description:* {{ .Annotations.description }}
        *Details:*
        {{ if .Labels.node_name }}     • *node_name:* `+"`{{ .Labels.node_name }}`"+`
        {{ end }}{{ if .Labels.node_id }}     • *node_id:* `+"`{{ .Labels.node_id }}`"+`
        {{ end }}{{ if .Labels.service_name }}     • *service_name:* `+"`{{ .Labels.service_name }}`"+`
        {{ end }}{{ if .Labels.service_id }}     • *service_id:* `+"`{{ .Labels.service_id }}`"+`
        {{ end }}{{ if .Labels.service_type }}     • *service_type:* `+"`{{ .Labels.service_type }}`"+`
        {{ end }}{{ if .Labels.rule_id }}     • *rule_id:* `+"`{{ .Labels.rule_id }}`"+`
        {{ end }}{{ if .Labels.alertgroup }}     • *alertgroup:* `+"`{{ .Labels.alertgroup }}`"+`
        {{ end }}{{ if .Labels.template_name }}     • *template_name:* `+"`{{ .Labels.template_name }}`"+`
        {{ end }}{{ if .Labels.severity }}     • *severity:* `+"`{{ .Labels.severity }}`"+`
        {{ end }}{{ if .Labels.agent_id }}     • *agent_id:* `+"`{{ .Labels.agent_id }}`"+`
        {{ end }}{{ if .Labels.agent_type }}     • *agent_type:* `+"`{{ .Labels.agent_type }}`"+`
        {{ end }}{{ if .Labels.job }}     • *job:* `+"`{{ .Labels.job }}`"+`
        {{ end }}

        {{ end }}
      short_fields: false
      link_names: false
`) + "\n"
	assert.Equal(t, expected, string(actual), "actual:\n%s", actual)
}
