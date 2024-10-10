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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	"context"
	"os"
	"testing"

	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvVarsDatasource(t *testing.T) {
	// NOTE: t.Parallel() is not possible when using a different set of envvars for each test.
	t.Parallel()

	type testEnvVars map[string]string

	ctx, cancel := context.WithCancel(context.Background())
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	setup := func(t *testing.T, envVars testEnvVars) (DataSource, func()) {
		t.Helper()
		for key, val := range envVars {
			os.Setenv(key, val) //nolint:errcheck
		}

		evConf := &DSConfigEnvVars{
			Enabled: true,
		}
		dsEnvVars := NewDataSourceEnvVars(*evConf, logEntry)

		return dsEnvVars, func() {
			for key := range envVars {
				os.Unsetenv(key) //nolint:errcheck
			}
			err := dsEnvVars.Dispose(ctx)
			require.NoError(t, err)
		}
	}

	t.Cleanup(func() {
		cancel()
	})

	t.Run("Basic", func(t *testing.T) {
		t.Parallel()

		envVars := testEnvVars{
			"TEST_ENV_VAR1": "1",
			"TEST_ENV_VAR2": "test",
			"TEST_ENV_VAR3": "true",
			"TEST_ENV_VAR4": "1.1",
			"TEST_ENV_VAR5": "",
		}
		config := &Config{
			ID:      "test",
			Source:  "ENV_VARS",
			Summary: "EnvVar test query",
			Data: []ConfigData{
				{MetricName: "test_env_var1", Column: "TEST_ENV_VAR1"},
				{MetricName: "test_env_var2", Column: "TEST_ENV_VAR2"},
				{MetricName: "test_env_var3", Column: "TEST_ENV_VAR3"},
				{MetricName: "test_env_var4", Column: "TEST_ENV_VAR4"},
				{MetricName: "test_env_var5", Column: "TEST_ENV_VAR5"},
			},
		}

		dsEnvVars, dispose := setup(t, envVars)
		t.Cleanup(func() { dispose() })

		err := dsEnvVars.Init(ctx)
		require.NoError(t, err)

		metrics, err := dsEnvVars.FetchMetrics(ctx, *config)
		require.NoError(t, err)

		expected := []*pmmv1.ServerMetric_Metric{
			{Key: "test_env_var1", Value: "1"},
			{Key: "test_env_var2", Value: "test"},
			{Key: "test_env_var3", Value: "true"},
			{Key: "test_env_var4", Value: "1.1"},
		}
		assert.Equal(t, expected, metrics)
	})

	t.Run("StripValues", func(t *testing.T) {
		t.Parallel()

		envVars := testEnvVars{
			"TEST_ENV_VAR6":  "1",
			"TEST_ENV_VAR7":  "test",
			"TEST_ENV_VAR8":  "true",
			"TEST_ENV_VAR9":  "1.1",
			"TEST_ENV_VAR10": "",
		}
		config := &Config{
			ID:      "test",
			Source:  "ENV_VARS",
			Summary: "EnvVar test query",
			Transform: &ConfigTransform{
				Type: "StripValues",
			},
			Data: []ConfigData{
				{MetricName: "test_env_var6", Column: "TEST_ENV_VAR6"},
				{MetricName: "test_env_var7", Column: "TEST_ENV_VAR7"},
				{MetricName: "test_env_var8", Column: "TEST_ENV_VAR8"},
				{MetricName: "test_env_var9", Column: "TEST_ENV_VAR9"},
				{MetricName: "test_env_var10", Column: "TEST_ENV_VAR10"},
			},
		}

		dsEnvVars, dispose := setup(t, envVars)
		t.Cleanup(func() { dispose() })

		err := dsEnvVars.Init(ctx)
		require.NoError(t, err)

		metrics, err := dsEnvVars.FetchMetrics(ctx, *config)
		require.NoError(t, err)

		expected := []*pmmv1.ServerMetric_Metric{
			{Key: "test_env_var6", Value: "1"},
			{Key: "test_env_var7", Value: "1"},
			{Key: "test_env_var8", Value: "1"},
			{Key: "test_env_var9", Value: "1"},
		}
		metrics, err = transformExportValues(config, metrics)
		require.NoError(t, err)
		assert.Equal(t, expected, metrics)
	})
}
