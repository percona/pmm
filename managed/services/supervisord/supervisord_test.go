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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
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
	settings.VictoriaMetrics.CacheEnabled = new(false)
	settings.Nomad.Enabled = new(true)

	for _, tmpl := range templates.Templates() {
		n := tmpl.Name()
		if n == "" {
			continue
		}
		t.Run(tmpl.Name(), func(t *testing.T) {
			t.Parallel()
			expected, err := os.ReadFile(filepath.Join(configDir, tmpl.Name()+".ini"))
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings)
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
	settings.VictoriaMetrics.CacheEnabled = new(false)

	// Test environment variables being passed to VictoriaMetrics.
	t.Setenv("VM_search_maxQueryLen", "2MB")
	t.Setenv("VM_search_latencyOffset", "10s")
	t.Setenv("VM_search_maxUniqueTimeseries", "500000000")
	t.Setenv("VM_search_maxSamplesPerQuery", "1600000000")
	t.Setenv("VM_search_maxQueueDuration", "100s")
	t.Setenv("VM_search_logSlowQueryDuration", "300s")
	t.Setenv("VM_search_maxQueryDuration", "9s")
	t.Setenv("VM_promscrape_streamParse", "false")
	t.Setenv("VM_maxIngestionRate", "5000000")

	for _, tmpl := range templates.Templates() {
		n := tmpl.Name()
		if n != "victoriametrics" { // just test the VM template
			continue
		}

		t.Run(tmpl.Name(), func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(configDir, tmpl.Name()+"_envvars.ini"))
			require.NoError(t, err)
			actual, err := s.marshalConfig(tmpl, settings)
			require.NoError(t, err)
			assert.Equal(t, string(expected), string(actual))
		})
	}
}

func TestDevContainer(t *testing.T) {
	t.Run("UpdateConfiguration", func(t *testing.T) {
		// logrus.SetLevel(logrus.DebugLevel)
		vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		s := New("/etc/supervisord.d", &models.Params{VMParams: vmParams, PGParams: &models.PGParams{}, HAParams: &models.HAParams{}})

		// This test rewrites /etc/supervisord.d and reloads supervisord, so it only runs
		// where PMM's supervisord is present: the dev container or the PMM Server image.
		_, statErr := os.Stat("/etc/supervisord.d")
		if s.supervisorctlPath == "" || statErr != nil {
			t.Skip("supervisord is not installed, skipping")
		}

		// restore original files after test, and remove any the test creates
		originals := make(map[string][]byte)
		matches, err := filepath.Glob("/etc/supervisord.d/*.ini")
		require.NoError(t, err)
		for _, m := range matches {
			b, err := os.ReadFile(m)
			require.NoError(t, err)
			originals[m] = b
		}
		defer func() {
			current, globErr := filepath.Glob("/etc/supervisord.d/*.ini")
			require.NoError(t, globErr)
			for _, name := range current {
				if _, ok := originals[name]; !ok {
					require.NoError(t, os.Remove(name))
				}
			}
			for name, b := range originals {
				err = os.WriteFile(name, b, 0)
				require.NoError(t, err)
			}
			// force update supervisor config
			_, err = s.supervisorctl(t.Context(), "update")
			require.NoError(t, err)
		}()

		settings := &models.Settings{
			DataRetention: 3600 * time.Hour,
		}

		b, err := s.marshalConfig(templates.Lookup("victoriametrics"), settings)
		require.NoError(t, err)
		changed, err := s.saveConfigAndReload("victoriametrics", b)
		require.NoError(t, err)
		assert.True(t, changed)
		changed, err = s.saveConfigAndReload("victoriametrics", b)
		require.NoError(t, err)
		assert.False(t, changed)

		err = s.UpdateConfiguration(settings)
		require.NoError(t, err)
	})
}

func TestProgramRunning(t *testing.T) {
	vmParams, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)

	s := New("/etc/supervisord.d", &models.Params{VMParams: vmParams, PGParams: &models.PGParams{}, HAParams: &models.HAParams{}})
	if s.supervisorctlPath == "" {
		t.Skip("supervisorctl not found")
	}

	assert.True(t, s.ProgramRunning(t.Context(), "nginx"))
	assert.False(t, s.ProgramRunning(t.Context(), "no-such-program"))
}

func TestSupervisorctlCancellation(t *testing.T) {
	t.Parallel()

	sleep, err := exec.LookPath("sleep")
	require.NoError(t, err)

	s := &Service{
		supervisorctlPath: sleep,
		l:                 logrus.WithField("component", "supervisord-test"),
	}

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = s.supervisorctl(ctx, "60")
	require.Error(t, err)
	assert.Less(t, time.Since(start), 10*time.Second, "cancellation should kill the blocked process")
}

// fakeSupervisorctl returns the path to a script that prints the given `supervisorctl status`
// output and exits with the given code.
func fakeSupervisorctl(t *testing.T, output string, exitCode int) string {
	t.Helper()

	script := "#!/bin/sh\n"
	if output != "" {
		script += fmt.Sprintf("printf '%%s\\n' '%s'\n", output)
	}
	script += fmt.Sprintf("exit %d\n", exitCode)

	path := filepath.Join(t.TempDir(), "supervisorctl")
	require.NoError(t, os.WriteFile(path, []byte(script), 0o700))
	return path
}

func TestProgramRunningStatuses(t *testing.T) {
	t.Parallel()

	// Output and exit codes below were captured from supervisorctl in a PMM Server container.
	// UNKNOWN is the one state that cannot be induced on demand, so its line is synthetic.
	for _, tc := range []struct {
		name     string
		output   string
		exitCode int
		running  bool
	}{
		{"running", "nginx                            RUNNING   pid 87, uptime 0:02:44", 0, true},
		{"starting", "t-starting                       STARTING  ", 0, true},
		{"backoff", "t-backoff                        BACKOFF   Exited too quickly (process log may have details)", 0, true},
		{"stopping", "t-stopping                       STOPPING  ", 0, true},
		{"stopped", "t-starting                       STOPPED   Aug 26 06:51 AM", 3, false},
		{"exited", "pmm-init                         EXITED    Aug 26 06:49 AM", 3, false},
		{"fatal", "t-fatal                          FATAL     Exited too quickly (process log may have details)", 3, false},
		{"unknown", "t-unknown                        UNKNOWN   ", 3, false},
		{"no such process", "no-such-program: ERROR (no such process)", 4, false},
		{"no output", "", 4, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &Service{
				supervisorctlPath: fakeSupervisorctl(t, tc.output, tc.exitCode),
				l:                 logrus.WithField("component", "supervisord-test"),
			}
			assert.Equal(t, tc.running, s.ProgramRunning(t.Context(), "program"))
		})
	}
}
