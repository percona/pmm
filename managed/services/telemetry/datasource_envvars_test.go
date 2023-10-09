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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvVarsDatasource(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{
		"TEST_ENV_VAR1": "1",
		"TEST_ENV_VAR2": "test",
		"TEST_ENV_VAR3": "true",
		"TEST_ENV_VAR4": "1.1",
		"TEST_ENV_VAR5": "false",
	}

	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	for key, val := range envVars {
		os.Setenv(key, val)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
		cancel()
	})

	config := &Config{
		ID:      "test",
		Source:  "ENV_VARS",
		Query:   "TEST_ENV_VAR1,TEST_ENV_VAR2,TEST_ENV_VAR3,TEST_ENV_VAR4,TEST_ENV_VAR5",
		Summary: "EnvVar test query",
		Transform: &ConfigTransform{
			Type:   "JSON",
			Metric: "test_env_vars",
		},
	}

	t.Run("get metrics from environment", func(t *testing.T) {
		t.Parallel()

		conf := &DSConfigEnvVars{
			Enabled: true,
		}
		dsEnvVars := NewDataSourceEnvVars(*conf, logEntry)

		err := dsEnvVars.Init(ctx)
		require.NoError(t, err)

		metrics, err := dsEnvVars.FetchMetrics(ctx, *config)
		require.NoError(t, err)
		assert.Equal(t, len(metrics), len(envVars))

		metric1 := metrics[0]
		assert.Equal(t, metric1.Key, "TEST_ENV_VAR1")
		assert.Equal(t, metric1.Value, "1")

		metric3 := metrics[2]
		assert.Equal(t, metric3.Key, "TEST_ENV_VAR3")
		assert.Equal(t, metric3.Value, "true")

		err = dsEnvVars.Dispose(ctx)
		require.NoError(t, err)
	})
}
