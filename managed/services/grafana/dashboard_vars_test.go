// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

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
}

func TestBuildGrafanaRenderQueryValues_ScenesGating(t *testing.T) {
	qLegacy := buildGrafanaRenderQueryValues("12", "now-1h", "now", 1, 1000, 500, 1, "browser", "", 35, map[string]string{})
	assert.Empty(t, qLegacy.Get("__feature.dashboardSceneSolo"))
	assert.Empty(t, qLegacy.Get("viewPanel"))

	qScenes := buildGrafanaRenderQueryValues("12", "now-1h", "now", 1, 1000, 500, 1, "browser", "", 39, map[string]string{})
	assert.Equal(t, "true", qScenes.Get("__feature.dashboardSceneSolo"))
	assert.Equal(t, "panel-12", qScenes.Get("viewPanel"))
}
