// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package grafana

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeDashboardVarsGolden(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "dashboard_merge.json"))
	require.NoError(t, err)
	var env dashboardAPIEnvelope
	require.NoError(t, json.Unmarshal(b, &env))

	t.Run("defaults", func(t *testing.T) {
		m, err := MergeDashboardVars(env.Dashboard, nil)
		require.NoError(t, err)
		assert.Equal(t, "svc-default", m["var-service_name"])
		assert.Equal(t, "$__all", m["var-region"])
	})

	t.Run("override_unknown_rejected", func(t *testing.T) {
		_, err := MergeDashboardVars(env.Dashboard, map[string]string{"nope": "x"})
		require.Error(t, err)
	})

	t.Run("override_case_insensitive_var_prefix", func(t *testing.T) {
		m, err := MergeDashboardVars(env.Dashboard, map[string]string{"VAR-service_name": "other"})
		require.NoError(t, err)
		assert.Equal(t, "other", m["var-service_name"])
		assert.Equal(t, "$__all", m["var-region"])
	})

	t.Run("reject_all_for_non_all_var", func(t *testing.T) {
		_, err := MergeDashboardVars(env.Dashboard, map[string]string{"service_name": "$__all"})
		require.Error(t, err)
	})

	t.Run("explicit_empty_override_clears_dashboard_default", func(t *testing.T) {
		m, err := MergeDashboardVars(env.Dashboard, map[string]string{"service_name": ""})
		require.NoError(t, err)
		v, has := m["var-service_name"]
		assert.True(t, has, "empty override should produce explicit var-service_name= in render query")
		assert.Empty(t, v)
		assert.Equal(t, "$__all", m["var-region"])
	})
}

func TestBuildGrafanaRenderQueryValues_MatchesGrafanaUIShape(t *testing.T) {
	q := buildGrafanaRenderQueryValues("12", "now-1h", "now", 1, 1000, 500, 1, "browser", "", map[string]string{"var-x": "y"})
	assert.Equal(t, "panel-12", q.Get("panelId"))
	assert.Equal(t, "true", q.Get("__feature.dashboardScene"))
	assert.Equal(t, "true", q.Get("hideLogo"))
	assert.Empty(t, q.Get("__feature.dashboardSceneSolo"))
	assert.Empty(t, q.Get("viewPanel"))
	assert.Equal(t, "1m", q.Get("refresh"))
	assert.Equal(t, "browser", q.Get("timezone"))
	assert.Equal(t, "browser", q.Get("tz"))
	assert.Equal(t, "y", q.Get("var-x"))
}

func TestBuildGrafanaRenderQueryValues_SplitsIANAAndBrowserTimezone(t *testing.T) {
	q := buildGrafanaRenderQueryValues("8", "now-1h", "now", 1, 1000, 500, 1, "Europe/Amsterdam", "", nil)
	assert.Equal(t, "browser", q.Get("timezone"))
	assert.Equal(t, "Europe/Amsterdam", q.Get("tz"))
}

func TestBuildGrafanaRenderQueryValues_SceneUsesNormalizedPanelID(t *testing.T) {
	q := buildGrafanaRenderQueryValues("panel-7", "now-1h", "now", 1, 1000, 500, 1, "browser", "", map[string]string{})
	assert.Equal(t, "panel-7", q.Get("panelId"))
}

func TestBuildGrafanaRenderQueryValues_EmitsExplicitEmptyVars(t *testing.T) {
	q := buildGrafanaRenderQueryValues("8", "now-1h", "now", 1, 1000, 500, 1, "browser", "", map[string]string{
		"var-cluster": "",
		"var-x":       "y",
	})
	_, hasCluster := q["var-cluster"]
	assert.True(t, hasCluster, "empty variable must still be emitted as var-cluster= to match Grafana share URLs")
	assert.Empty(t, q.Get("var-cluster"))
	assert.Equal(t, "y", q.Get("var-x"))
}

func TestSanitizeTemplateValue(t *testing.T) {
	assert.Equal(t, "$__auto", sanitizeTemplateValue("interval", "$__auto_interval_interval"))
	assert.Equal(t, "$__auto", sanitizeTemplateValue("interval", "$__auto_interval"))
	assert.Empty(t, sanitizeTemplateValue("agent_id", "/agent_id/1093bb03-007c-41e9-80f2-930f37fa1733"))
	assert.Equal(t, "e30e782e-8384-47e6-80d6-6187391f2f35", sanitizeTemplateValue("agent_id", "e30e782e-8384-47e6-80d6-6187391f2f35"))
	assert.Equal(t, "b658dda3-8571-47be-a40e-46daa78f9c54", sanitizeTemplateValue("node_id", "/node_id/b658dda3-8571-47be-a40e-46daa78f9c54"))
	assert.Equal(t, "a9cecc72-2add-4c24-a47c-2dd2dec8f02c", sanitizeTemplateValue("service_id", "/service_id/a9cecc72-2add-4c24-a47c-2dd2dec8f02c"))
	assert.Equal(t, "plain-uuid-0000-0000-000000000001", sanitizeTemplateValue("service_id", "plain-uuid-0000-0000-000000000001"))
}
