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

package ia

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestConvertTemplate(t *testing.T) {
	ctx := context.Background()
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

	// Enable IA
	settings, err := models.GetSettings(db)
	require.NoError(t, err)
	settings.IntegratedAlerting.Enabled = true
	err = models.SaveSettings(db, settings)
	require.NoError(t, err)

	alertManager := new(mockAlertManager)
	alertManager.On("RequestConfigurationUpdate").Return()

	// Create channel
	channels := NewChannelsService(db, alertManager)
	respC, err := channels.AddChannel(context.Background(), &iav1beta1.AddChannelRequest{
		Summary: "test channel",
		EmailConfig: &iav1beta1.EmailConfig{
			SendResolved: false,
			To:           []string{"test@test.test"},
		},
		Disabled: false,
	})
	require.NoError(t, err)
	channelID := respC.ChannelId

	// Load test templates
	templates := NewTemplatesService(db)
	templates.userTemplatesPath = testTemplates2
	templates.Collect(ctx)

	t.Run("normal", func(t *testing.T) {
		testDir, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.RemoveAll(testDir)
			require.NoError(t, err)
		})

		vmAlert := new(mockVmAlert)
		vmAlert.On("RequestConfigurationUpdate").Return()

		// Create test rule
		rules := NewRulesService(db, templates, vmAlert, alertManager)
		rules.rulesPath = testDir
		resp, err := rules.CreateAlertRule(context.Background(), &iav1beta1.CreateAlertRuleRequest{
			TemplateName: "test_template",
			Disabled:     false,
			Summary:      "some testing rule",
			Params: []*iav1beta1.RuleParam{{
				Name: "threshold",
				Type: iav1beta1.ParamType_FLOAT,
				Value: &iav1beta1.RuleParam_Float{
					Float: 1.22,
				},
			}},
			For:      durationpb.New(2 * time.Second),
			Severity: managementpb.Severity_SEVERITY_INFO,
			CustomLabels: map[string]string{
				"baz": "faz",
			},
			Filters: []*iav1beta1.Filter{{
				Type:  iav1beta1.FilterType_EQUAL,
				Key:   "some_key",
				Value: "'60'",
			}},
			ChannelIds: []string{channelID},
		})
		require.NoError(t, err)
		ruleID := resp.RuleId

		// Write vmAlert rules files
		rules.writeVMAlertRulesFiles()

		file, err := ioutil.ReadFile(ruleFileName(testDir, ruleID))
		require.NoError(t, err)

		expected := fmt.Sprintf(`---
groups:
    - name: PMM Integrated Alerting
      rules:
        - alert: %s
          expr: |-
            max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
            mysql_global_variables_max_connections
            * 100
            > 1.22
          for: 2s
          labels:
            baz: faz
            foo: bar
            ia: "1"
            rule_id: %s
            severity: SEVERITY_INFO
            template_name: test_template
          annotations:
            description: |-
                More than 1.22%% of MySQL connections are in use on {{ $labels.instance }}
                VALUE = {{ $value }}
                LABELS: {{ $labels }}
            rule: some testing rule
            summary: MySQL too many connections (instance {{ $labels.instance }})
`, ruleID, ruleID)

		assert.Equal(t, expected, string(file))
	})

	t.Run("disabled", func(t *testing.T) {
		testDir, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.RemoveAll(testDir)
			require.NoError(t, err)
		})

		vmAlert := new(mockVmAlert)
		vmAlert.On("RequestConfigurationUpdate").Return()

		// Create test rule
		rules := NewRulesService(db, templates, vmAlert, alertManager)
		rules.rulesPath = testDir
		resp, err := rules.CreateAlertRule(context.Background(), &iav1beta1.CreateAlertRuleRequest{
			TemplateName: "test_template",
			Disabled:     true,
			Summary:      "some testing rule",
			Params: []*iav1beta1.RuleParam{{
				Name: "threshold",
				Type: iav1beta1.ParamType_FLOAT,
				Value: &iav1beta1.RuleParam_Float{
					Float: 1.22,
				},
			}},
			For:      durationpb.New(2 * time.Second),
			Severity: managementpb.Severity_SEVERITY_INFO,
			CustomLabels: map[string]string{
				"baz": "faz",
			},
			Filters: []*iav1beta1.Filter{{
				Type:  iav1beta1.FilterType_EQUAL,
				Key:   "some_key",
				Value: "60",
			}},
			ChannelIds: []string{channelID},
		})
		require.NoError(t, err)
		ruleID := resp.RuleId

		// Write vmAlert rules files
		rules.writeVMAlertRulesFiles()

		filename := ruleFileName(testDir, ruleID)
		_, err = os.Stat(filename)
		assert.EqualError(t, err, fmt.Sprintf("stat %s: no such file or directory", filename))
	})
}

func ruleFileName(testDir, ruleID string) string {
	return testDir + "/" + strings.TrimPrefix(ruleID, "/rule_id/") + ".yml"
}
