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
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestConfig(t *testing.T) {
	t.Parallel()

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
	s := New(configDir, &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}})
	settings := &models.Settings{
		DataRetention:    30 * 24 * time.Hour,
		PMMPublicAddress: "192.168.0.42:8443",
	}
	settings.VictoriaMetrics.CacheEnabled = pointer.ToBool(false)

	err = s.UpdateConfiguration(settings, nil)
	require.NoError(t, err)

	for _, tmpl := range templates.Templates() {
		n := tmpl.Name()
		if n == "" {
			continue
		}
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
	s := New(configDir, &models.Params{VMParams: vmParams, PGParams: pgParams, HAParams: &models.HAParams{}})
	settings := &models.Settings{
		DataRetention:    30 * 24 * time.Hour,
		PMMPublicAddress: "192.168.0.42:8443",
	}
	settings.VictoriaMetrics.CacheEnabled = pointer.ToBool(false)

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

		t.Run(tmpl.Name(), func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(configDir, tmpl.Name()+"_envvars.ini")) //nolint:gosec
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings, nil)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	for str, expected := range map[string]*bool{
		`pmm-agent                        STOPPED   Sep 20 08:55 AM`:         pointer.ToBool(false),
		`pmm-managed                      RUNNING   pid 826, uptime 0:19:36`: pointer.ToBool(true),
		`pmm-init                         EXITED    Sep 20 07:42 AM`:         nil,
		`pmm-init                         STARTING`:                          pointer.ToBool(true), // no last column in that case
	} {
		assert.Equal(t, expected, parseStatus(str), "%q", str)
	}
}
