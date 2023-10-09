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

package supervisord

import (
	"os"
	"path/filepath"
	"testing"
	"text/template"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestConfig(t *testing.T) {
	t.Parallel()
	gRPCMessageMaxSize := uint32(100 * 1024 * 1024)

	pmmUpdateCheck := NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker_logs"))
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	s := New(configDir, pmmUpdateCheck, vmParams, models.PGParams{}, gRPCMessageMaxSize)
	settings := &models.Settings{
		DataRetention:   30 * 24 * time.Hour,
		AlertManagerURL: "https://external-user:passw!,ord@external-alertmanager:6443/alerts",
	}
	settings.VictoriaMetrics.CacheEnabled = false

	for _, tmpl := range templates.Templates() {
		n := tmpl.Name()
		if n == "" || n == "dbaas-controller" {
			continue
		}

		tmpl := tmpl
		t.Run(tmpl.Name(), func(t *testing.T) {
			t.Parallel()
			expected, err := os.ReadFile(filepath.Join(configDir, tmpl.Name()+".ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings, nil)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}

func TestDBaaSController(t *testing.T) {
	t.Parallel()
	gRPCMessageMaxSize := uint32(100 * 1024 * 1024)

	pmmUpdateCheck := NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker_logs"))
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	s := New(configDir, pmmUpdateCheck, vmParams, models.PGParams{}, gRPCMessageMaxSize)

	var tp *template.Template
	for _, tmpl := range templates.Templates() {
		if tmpl.Name() == "dbaas-controller" {
			tp = tmpl
			break
		}
	}

	tests := []struct {
		Enabled bool
		File    string
	}{
		{
			Enabled: true,
			File:    "dbaas-controller_enabled",
		},
		{
			Enabled: false,
			File:    "dbaas-controller_disabled",
		},
	}
	for _, test := range tests {
		st := models.Settings{
			DBaaS: struct {
				Enabled bool `json:"enabled"`
			}{
				Enabled: test.Enabled,
			},
		}

		expected, err := os.ReadFile(filepath.Join(configDir, test.File+".ini")) //nolint:gosec
		require.NoError(t, err)
		actual, err := s.marshalConfig(tp, &st, nil)
		require.NoError(t, err)
		assert.Equal(t, string(expected), string(actual))
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	for str, expected := range map[string]*bool{
		`pmm-agent                        STOPPED   Sep 20 08:55 AM`:         pointer.ToBool(false),
		`pmm-managed                      RUNNING   pid 826, uptime 0:19:36`: pointer.ToBool(true),
		`pmm-update-perform               EXITED    Sep 20 07:42 AM`:         nil,
		`pmm-update-perform               STARTING`:                          pointer.ToBool(true), // no last column in that case
	} {
		assert.Equal(t, expected, parseStatus(str), "%q", str)
	}
}

func TestAddAlertManagerParam(t *testing.T) {
	t.Parallel()

	t.Run("empty alertmanager url", func(t *testing.T) {
		t.Parallel()
		params := make(map[string]interface{})
		err := addAlertManagerParams("", params)
		require.NoError(t, err)
		require.Equal(t, "http://127.0.0.1:9093/alertmanager", params["AlertmanagerURL"])
	})

	t.Run("simple alertmanager url", func(t *testing.T) {
		t.Parallel()
		params := make(map[string]interface{})
		err := addAlertManagerParams("https://some-alertmanager", params)
		require.NoError(t, err)
		require.Equal(t, "http://127.0.0.1:9093/alertmanager,https://some-alertmanager", params["AlertmanagerURL"])
	})

	t.Run("extract username and password from alertmanager url", func(t *testing.T) {
		t.Parallel()
		params := make(map[string]interface{})
		err := addAlertManagerParams("https://username1:PAsds!234@some-alertmanager", params)
		require.NoError(t, err)
		require.Equal(t, "http://127.0.0.1:9093/alertmanager,https://some-alertmanager", params["AlertmanagerURL"])
		require.Equal(t, ",username1", params["AlertManagerUser"])
		require.Equal(t, `,"PAsds!234"`, params["AlertManagerPassword"])
	})

	t.Run("incorrect alertmanager url", func(t *testing.T) {
		t.Parallel()
		params := make(map[string]interface{})
		err := addAlertManagerParams("*:9095", params)
		require.EqualError(t, err, `cannot parse AlertManagerURL: parse "*:9095": first path segment in URL cannot contain colon`)
		require.Equal(t, "http://127.0.0.1:9093/alertmanager", params["AlertmanagerURL"])
	})
}

func TestSavePMMConfig(t *testing.T) {
	t.Parallel()
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	tests := []struct {
		description string
		params      map[string]any
		file        string
	}{
		{
			description: "disable internal postgresql db",
			params:      map[string]any{"DisableInternalDB": true, "DisableSupervisor": false},
			file:        "pmm-db_disabled",
		},
		{
			description: "enable internal postgresql db",
			params:      map[string]any{"DisableInternalDB": false, "DisableSupervisor": false},
			file:        "pmm-db_enabled",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			expected, err := os.ReadFile(filepath.Join(configDir, test.file+".ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := marshalConfig(test.params)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}
