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

const gRPCMessageMaxSize = uint32(100 * 1024 * 1024)

func TestConfig(t *testing.T) {
	t.Parallel()

	pmmUpdateCheck := NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker_logs"))
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	pgParams := &models.PGParams{
		Addr:        "127.0.0.1:5432",
		DBName:      "postgres",
		DBUsername:  "db_username",
		DBPassword:  "db_password",
		SSLMode:     "verify",
		SSLCAPath:   "path-to-CA-cert",
		SSLKeyPath:  "path-to-key",
		SSLCertPath: "path-to-cert",
	}
	s := New(configDir, pmmUpdateCheck, &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}}, gRPCMessageMaxSize)
	settings := &models.Settings{
		DataRetention:    30 * 24 * time.Hour,
		AlertManagerURL:  "https://external-user:passw!,ord@external-alertmanager:6443/alerts",
		PMMPublicAddress: "192.168.0.42:8443",
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

func TestConfigVictoriaMetricsEnvvars(t *testing.T) {
	pmmUpdateCheck := NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker_logs"))
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	pgParams := &models.PGParams{
		Addr:        "127.0.0.1:5432",
		DBName:      "postgres",
		DBUsername:  "db_username",
		DBPassword:  "db_password",
		SSLMode:     "verify",
		SSLCAPath:   "path-to-CA-cert",
		SSLKeyPath:  "path-to-key",
		SSLCertPath: "path-to-cert",
	}
	s := New(configDir, pmmUpdateCheck, &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}}, gRPCMessageMaxSize)
	settings := &models.Settings{
		DataRetention:   30 * 24 * time.Hour,
		AlertManagerURL: "https://external-user:passw!,ord@external-alertmanager:6443/alerts",
	}
	settings.VictoriaMetrics.CacheEnabled = false

	// Test environment variables being passed to VictoriaMetrics.
	t.Setenv("VM_search_maxQueryLen", "2MB")
	t.Setenv("VM_search_latencyOffset", "10s")
	t.Setenv("VM_search_maxUniqueTimeseries", "500000000")
	t.Setenv("VM_search_maxSamplesPerQuery", "1600000000")
	t.Setenv("VM_search_maxQueueDuration", "100s")
	t.Setenv("VM_search_logSlowQueryDuration", "300s")
	t.Setenv("VM_search_maxQueryDuration", "9s")
	t.Setenv("VM_promscrape_streamParse", "false")

	for _, tmpl := range templates.Templates() {
		n := tmpl.Name()
		if n != "victoriametrics" { // just test the VM template
			continue
		}

		tmpl := tmpl
		t.Run(tmpl.Name(), func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(configDir, tmpl.Name()+"_envvars.ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings, nil)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}

func TestDBaaSController(t *testing.T) {
	t.Parallel()

	pmmUpdateCheck := NewPMMUpdateChecker(logrus.WithField("component", "supervisord/pmm-update-checker_logs"))
	configDir := filepath.Join("..", "..", "testdata", "supervisord.d")
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)
	s := New(configDir, pmmUpdateCheck, &models.Params{VMParams: vmParams, PGParams: &models.PGParams{}, HAParams: &models.HAParams{}}, gRPCMessageMaxSize)

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
