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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

const (
	// Where the built-in templates live in the repository. At runtime the
	// same files are read from /usr/local/percona/alerting-templates inside the image.
	shippedTemplatesDir = "../../data/alerting-templates"

	// The expected provisioning files. Regenerate them with
	// UPDATE_GOLDEN=1 go test ./managed/services/alerting/... -run TestRenderProvisioningFile
	// and read the diff before committing: these files are the contract users' silences and
	// notification policies are pinned to.
	goldenDir = "../../testdata/alerting-provisioning"

	// The UID Grafana derives for the provisioned "Metrics" datasource.
	testDatasourceUID = "PA58DA793C7250F1B"
)

// shippedTemplates reads the built-in templates from the repository, applying the same strict
// validation the server applies at startup.
func shippedTemplates(t *testing.T) map[string]models.Template {
	t.Helper()

	loaded, err := loadBuiltinTemplatesFromDir(shippedTemplatesDir)
	require.NoError(t, err)
	require.NotEmpty(t, loaded)

	templates := make(map[string]models.Template, len(loaded))
	for _, template := range loaded {
		templates[template.Name] = *template
	}
	return templates
}

// alertingEnabled builds the only setting the renderer still reads. The two per-bundle toggles are
// environment variables now, and reach the renderer as bundleGates.
func alertingEnabled(enabled bool) *models.Settings {
	settings := &models.Settings{}
	settings.Alerting.Enabled = new(enabled)
	return settings
}

// allGates is an HA deployment with both bundles switched on.
func allGates() bundleGates {
	return bundleGates{haEnabled: true, haAlertsEnabled: true, componentAlertsEnabled: true}
}

// assertGolden compares rendered content against a committed fixture byte for byte, so that a
// change to indentation, field order or any frozen value has to be made deliberately.
func assertGolden(t *testing.T, name string, actual []byte) {
	t.Helper()

	path := filepath.Join(goldenDir, name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.MkdirAll(goldenDir, 0o750))
		require.NoError(t, os.WriteFile(path, append(actual, '\n'), 0o644)) //nolint:gosec
		t.Logf("updated %s", path)
		return
	}

	expected, err := os.ReadFile(path) //nolint:gosec
	require.NoError(t, err, "golden file missing; regenerate with UPDATE_GOLDEN=1")
	assert.Equal(t, string(expected), string(actual)+"\n")
}

func TestRenderProvisioningFile(t *testing.T) {
	t.Parallel()

	templates := shippedTemplates(t)

	for _, tc := range []struct {
		name     string
		settings *models.Settings
		gates    bundleGates
		golden   string
	}{
		{
			name:     "HA cluster runs both bundles",
			settings: alertingEnabled(true),
			gates:    allGates(),
			golden:   "ha-and-components.json",
		},
		{
			name:     "standalone runs the component bundle only",
			settings: alertingEnabled(true),
			gates:    bundleGates{haEnabled: false, haAlertsEnabled: true, componentAlertsEnabled: true},
			golden:   "components-only.json",
		},
		{
			name:     "both toggles off deletes everything",
			settings: alertingEnabled(true),
			gates:    bundleGates{haEnabled: true},
			golden:   "disabled.json",
		},
		{
			name:     "disabling Percona Alerting takes the rules with it",
			settings: alertingEnabled(false),
			gates:    allGates(),
			golden:   "disabled.json",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := renderProvisioningFile(templates, testDatasourceUID, tc.settings, tc.gates, nil)
			require.NoError(t, err)
			require.NoError(t, validateProvisioningFile(actual))
			assertGolden(t, tc.golden, actual)
		})
	}
}

// TestRenderProvisioningFileIsDeterministic guards the change detection: the provisioner compares a
// hash of the rendered file against what Grafana last ingested, so unstable output would restart
// Grafana on every reconcile tick.
func TestRenderProvisioningFileIsDeterministic(t *testing.T) {
	t.Parallel()

	templates := shippedTemplates(t)
	settings := alertingEnabled(true)

	first, err := renderProvisioningFile(templates, testDatasourceUID, settings, allGates(), nil)
	require.NoError(t, err)

	for range 10 {
		again, err := renderProvisioningFile(templates, testDatasourceUID, settings, allGates(), nil)
		require.NoError(t, err)
		require.Equal(t, string(first), string(again))
	}
}

// TestProvisionedRuleContract pins the values that users' notification policies and silences depend
// on. They are derived from the templates rather than duplicated in the catalog, so this test is
// what stops a template edit silently changing a shipped rule's identity or timing.
func TestProvisionedRuleContract(t *testing.T) {
	t.Parallel()

	templates := shippedTemplates(t)

	expected := map[string]struct {
		bundle   string
		folder   string
		title    string
		severity string
		forValue time.Duration
	}{
		"pmm-ha-no-leader":         {haBundleID, haFolderTitle, "PMM HA cluster has no active leader", "critical", 3 * time.Minute},
		"pmm-ha-split-brain":       {haBundleID, haFolderTitle, "PMM HA split-brain detected", "critical", 3 * time.Minute},
		"pmm-ha-quorum-at-risk":    {haBundleID, haFolderTitle, "PMM HA quorum at risk", "critical", 3 * time.Minute},
		"pmm-ha-leader-flapping":   {haBundleID, haFolderTitle, "PMM HA leader is flapping", "warning", 5 * time.Minute},
		"pmm-ha-node-unreachable":  {haBundleID, haFolderTitle, "PMM HA node unreachable", "warning", 5 * time.Minute},
		"pmm-victoriametrics-down": {componentsBundleID, componentsFolderTitle, "PMM VictoriaMetrics is down", "critical", 5 * time.Minute},
		"pmm-clickhouse-down":      {componentsBundleID, componentsFolderTitle, "PMM ClickHouse is down", "critical", 5 * time.Minute},
		"pmm-grafana-down":         {componentsBundleID, componentsFolderTitle, "PMM Grafana is down", "critical", 5 * time.Minute},
		"pmm-qan-api2-down":        {componentsBundleID, componentsFolderTitle, "PMM Query Analytics API is down", "warning", 5 * time.Minute},
	}

	var count int
	for _, bundle := range builtinBundles {
		for _, rule := range bundle.rules {
			want, ok := expected[rule.uid]
			require.Truef(t, ok, "rule %q is not in the frozen contract", rule.uid)
			count++

			template, ok := templates[rule.templateName]
			require.Truef(t, ok, "rule %q references missing template %q", rule.uid, rule.templateName)

			assert.Equal(t, want.bundle, bundle.id, "rule %q changed bundle", rule.uid)
			assert.Equal(t, want.folder, bundle.folder, "rule %q changed folder", rule.uid)
			assert.Equal(t, want.title, template.Summary, "rule %q changed title", rule.uid)
			assert.Equal(t, want.forValue, template.For, "rule %q changed for duration", rule.uid)

			labels := renderedLabels(t, templates, bundle, rule)
			assert.Equal(t, want.severity, labels["severity"], "rule %q changed severity", rule.uid)
			assert.Equal(t, "1", labels["percona_alerting"])
			assert.Equal(t, "1", labels[provisionedLabel])
			assert.Equal(t, rule.templateName, labels["template_name"])
		}
	}
	assert.Len(t, expected, count, "the catalog and the frozen contract disagree on the rule set")
}

func renderedLabels(t *testing.T, templates map[string]models.Template, bundle provisioningBundle, rule provisionedRule) map[string]string {
	t.Helper()

	rendered, err := renderProvisionedRule(rule, templates[rule.templateName], testDatasourceUID)
	require.NoError(t, err)
	assert.Equal(t, provisionedNoDataState, rendered.NoDataState)
	assert.Equal(t, provisionedExecErrState, rendered.ExecErrState)
	assert.NotEmpty(t, bundle.folder)
	return rendered.Labels
}

// TestRenderProvisioningFileRejectsBadTemplate proves the failure mode the whole design depends on:
// a template PMM cannot render produces an error and no file, rather than a partial file that would
// stop Grafana from starting and take the PMM user interface down with it.
func TestRenderProvisioningFileRejectsBadTemplate(t *testing.T) {
	t.Parallel()

	templates := shippedTemplates(t)
	broken := templates["pmm_ha_no_leader"]
	broken.Yaml = "not: a valid template"
	templates["pmm_ha_no_leader"] = broken

	actual, err := renderProvisioningFile(templates, testDatasourceUID, alertingEnabled(true), allGates(), nil)
	require.Error(t, err)
	assert.Nil(t, actual)
}

func TestRenderProvisioningFileRequiresDatasource(t *testing.T) {
	t.Parallel()

	_, err := renderProvisioningFile(shippedTemplates(t), "", alertingEnabled(true), allGates(), nil)
	require.Error(t, err)
}

func TestRenderProvisioningFileRejectsMissingTemplate(t *testing.T) {
	t.Parallel()

	templates := shippedTemplates(t)
	delete(templates, "pmm_clickhouse_down")

	_, err := renderProvisioningFile(templates, testDatasourceUID, alertingEnabled(true), allGates(), nil)
	require.ErrorContains(t, err, "pmm_clickhouse_down")
}

func TestValidateProvisioningFile(t *testing.T) {
	t.Parallel()

	valid := func() provisioningFile {
		return provisioningFile{
			APIVersion: 1,
			Groups: []provisioningGroup{{
				OrgID:    provisionedOrgID,
				Name:     provisionedGroupName,
				Folder:   componentsFolderTitle,
				Interval: provisionedInterval.String(),
				Rules: []provisioningRule{{
					UID:          "pmm-grafana-down",
					Title:        "PMM Grafana is down",
					Condition:    "A",
					For:          "5m",
					NoDataState:  provisionedNoDataState,
					ExecErrState: provisionedExecErrState,
					Data:         []services.Data{{RefID: "A", DatasourceUID: testDatasourceUID}},
				}},
			}},
			DeleteRules: []provisioningRuleDelete{{OrgID: provisionedOrgID, UID: "pmm-ha-no-leader"}},
		}
	}

	marshal := func(t *testing.T, file provisioningFile) []byte {
		t.Helper()
		b, err := json.Marshal(file)
		require.NoError(t, err)
		return b
	}

	t.Run("accepts a well formed file", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateProvisioningFile(marshal(t, valid())))
	})

	t.Run("rejects a deletion without an organisation", func(t *testing.T) {
		t.Parallel()
		// Grafana gives a deletion entry no default organisation, so this would silently delete
		// nothing at all.
		file := valid()
		file.DeleteRules[0].OrgID = 0
		require.ErrorContains(t, validateProvisioningFile(marshal(t, file)), "orgId")
	})

	t.Run("rejects a rule that is both provisioned and deleted", func(t *testing.T) {
		t.Parallel()
		file := valid()
		file.DeleteRules[0].UID = file.Groups[0].Rules[0].UID
		require.ErrorContains(t, validateProvisioningFile(marshal(t, file)), "both provisioned and deleted")
	})

	t.Run("rejects a condition that names no query", func(t *testing.T) {
		t.Parallel()
		file := valid()
		file.Groups[0].Rules[0].Condition = "C"
		require.ErrorContains(t, validateProvisioningFile(marshal(t, file)), "condition")
	})

	t.Run("rejects a for shorter than the evaluation interval", func(t *testing.T) {
		t.Parallel()
		file := valid()
		file.Groups[0].Rules[0].For = "10s"
		require.ErrorContains(t, validateProvisioningFile(marshal(t, file)), "shorter than the evaluation interval")
	})

	t.Run("rejects a query without a datasource", func(t *testing.T) {
		t.Parallel()
		file := valid()
		file.Groups[0].Rules[0].Data[0].DatasourceUID = ""
		require.ErrorContains(t, validateProvisioningFile(marshal(t, file)), "datasource")
	})
}
