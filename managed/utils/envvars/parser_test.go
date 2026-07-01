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

package envvars

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestEnvVarValidator(t *testing.T) {
	t.Parallel()

	t.Run("Valid env variables", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PMM_ENABLE_UPDATES=false",
			"PMM_ENABLE_TELEMETRY=True",
			"PMM_METRICS_RESOLUTION=5m",
			"PMM_METRICS_RESOLUTION_MR=5s",
			"PMM_METRICS_RESOLUTION_LR=1h",
			"PMM_DATA_RETENTION=72h",
			"PMM_UPDATE_SNOOZE_DURATION=1h",
		}
		expectedEnvVars := &models.ChangeSettingsParams{
			DataRetention:   72 * time.Hour,
			EnableTelemetry: new(true),
			EnableUpdates:   new(false),
			EnableAdvisors:  nil,
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Minute,
				MR: 5 * time.Second,
				LR: time.Hour,
			},
			UpdateSnoozeDuration: time.Hour,
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("Unknown env variables", func(t *testing.T) {
		t.Parallel()

		envs := []string{"UNKNOWN_VAR=VAL", "ANOTHER_UNKNOWN_VAR=VAL"}
		expectedEnvVars := &models.ChangeSettingsParams{}
		expectedWarns := []string{
			"unknown environment variable UNKNOWN_VAR=VAL",
			"unknown environment variable ANOTHER_UNKNOWN_VAR=VAL",
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Equal(t, expectedWarns, gotWarns)
	})

	t.Run("Default env vars", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
			"HOSTNAME=host",
			"TERM=xterm-256color",
			"HOME=/home/user/",
			"LC_ALL=en_US.utf8",
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("Optional env vars", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"container=podman",
			"no_proxy=localhost",
			"http_proxy=http://localhost",
			"https_proxy=http://localhost",
			"NO_PROXY=localhost",
			"HTTP_PROXY=http://localhost",
			"HTTPS_PROXY=http://localhost",
			"PMM_INSTALL_METHOD=Helm",
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("PMM_ADRE_URL valid", func(t *testing.T) {
		t.Parallel()

		envs := []string{"PMM_ADRE_URL=http://holmesgpt:8080"}
		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
		assert.NotNil(t, gotEnvVars.AdreURL)
		assert.Equal(t, "http://holmesgpt:8080", *gotEnvVars.AdreURL)
		assert.NotNil(t, gotEnvVars.EnableAdre)
		assert.True(t, *gotEnvVars.EnableAdre)
	})

	t.Run("PMM_ADRE_TLS_SKIP_VERIFY valid", func(t *testing.T) {
		t.Parallel()

		envs := []string{"PMM_ADRE_TLS_SKIP_VERIFY=true"}
		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
		require.NotNil(t, gotEnvVars.AdreTLSSkipVerify)
		assert.True(t, *gotEnvVars.AdreTLSSkipVerify)
	})

	t.Run("PMM_ADRE_URL invalid", func(t *testing.T) {
		t.Parallel()

		envs := []string{"PMM_ADRE_URL=not-a-url"}
		gotEnvVars, gotErrs, _ := ParseEnvVars(envs)
		assert.Len(t, gotErrs, 1)
		assert.Contains(t, gotErrs[0].Error(), "PMM_ADRE_URL")
		assert.Nil(t, gotEnvVars.AdreURL)
	})

	t.Run("PMM_ADRE_URL plaintext public rejected", func(t *testing.T) {
		t.Parallel()

		envs := []string{"PMM_ADRE_URL=http://holmes.example.com"}
		gotEnvVars, gotErrs, _ := ParseEnvVars(envs)
		assert.Len(t, gotErrs, 1)
		assert.Contains(t, gotErrs[0].Error(), "PMM_ADRE_URL")
		assert.Nil(t, gotEnvVars.AdreURL)
	})

	t.Run("PMM_ADRE_URL plaintext public allowed with dev flag", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PMM_ADRE_URL=http://holmes.example.com",
			"PMM_DEV_ADRE_ALLOW_INSECURE_URL=true",
		}
		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
		require.NotNil(t, gotEnvVars.AdreURL)
		assert.Equal(t, "http://holmes.example.com", *gotEnvVars.AdreURL)
	})

	t.Run("Skipped clickhouse env vars", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PMM_CLICKHOUSE_DATABASE=pmm",
			"PMM_CLICKHOUSE_ADDR=127.0.0.1:9000",
			"PMM_CLICKHOUSE_USER=default",
			"PMM_CLICKHOUSE_PASSWORD=secret",
			"PMM_CLICKHOUSE_HOST=127.0.0.1",
			"PMM_CLICKHOUSE_PORT=9000",
			"PMM_CLICKHOUSE_IS_CLUSTER=true",
			"PMM_CLICKHOUSE_CLUSTER_NAME=cluster1",
			"PMM_CLICKHOUSE_NODES=n1,n2",
			"PMM_CLICKHOUSE_CONFIG=low-memory",
			"PMM_DISABLE_BUILTIN_CLICKHOUSE=1",
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("Invalid env variables values", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PMM_ENABLE_UPDATES",
			"PMM_ENABLE_TELEMETRY",
			"PMM_ENABLE_UPDATES=5",
			"PMM_ENABLE_TELEMETRY=X",
			"PMM_METRICS_RESOLUTION=5f",
			"PMM_METRICS_RESOLUTION_MR=s5",
			"PMM_METRICS_RESOLUTION_LR=1hour",
			"PMM_DATA_RETENTION=keep one week",
			"PMM_UPDATE_SNOOZE_DURATION=one week",
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		expectedErrs := []error{
			errors.New(`failed to parse environment variable "PMM_ENABLE_UPDATES"`),
			errors.New(`failed to parse environment variable "PMM_ENABLE_TELEMETRY"`),
			errors.New(`invalid value "5" for environment variable "PMM_ENABLE_UPDATES"`),
			errors.New(`invalid value "x" for environment variable "PMM_ENABLE_TELEMETRY"`),
			errors.New(`environment variable "PMM_METRICS_RESOLUTION=5f" has invalid duration 5f`),
			errors.New(`environment variable "PMM_METRICS_RESOLUTION_MR=s5" has invalid duration s5`),
			errors.New(`environment variable "PMM_METRICS_RESOLUTION_LR=1hour" has invalid duration 1hour`),
			errors.New(`environment variable "PMM_DATA_RETENTION=keep one week" has invalid duration keep one week`),
			errors.New(`environment variable "PMM_UPDATE_SNOOZE_DURATION=one week" has invalid duration one week`),
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Equal(t, expectedErrs, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("env vars with 'PERCONA_*' prefix show warnings", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PERCONA_TEST_PLATFORM_ADDRESS=https://host:333",
			"PERCONA_TEST_CHECKS_PUBLIC_KEY=some key",
			"PERCONA_TEST_AUTH_HOST=host:333",
			"PERCONA_TEST_CHECKS_HOST=host:333",
			"PERCONA_TEST_TELEMETRY_HOST=host:333",
			"PERCONA_TEST_SAAS_HOST=host:333",
			"PERCONA_TELEMETRY_DISABLE=1", // this one shouldn't trigger the warning
		}
		expectedEnvVars := &models.ChangeSettingsParams{}
		expectedWarns := []string{
			`PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation`,
			`PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation`,
			`PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation`,
			`PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation`,
			`PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation`,
			`PERCONA_* env variables are NOT SUPPORTED, please use PMM_* env variables, for details please check our documentation`,
		}

		gotEnvVars, _, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Equal(t, expectedWarns, gotWarns)
	})

	t.Run("Parse Platform API Timeout", func(t *testing.T) {
		t.Parallel()

		userCase := []struct {
			value   string
			respVal time.Duration
			msg     string
		}{
			{
				value: "", respVal: time.Second * 30,
				msg: "Setting the default timeout for Platform API to 30s.",
			},
			{
				value: "10s", respVal: time.Second * 10,
				msg: "Set the timeout for Platform API to 10s.",
			},
			{
				value: "xxx", respVal: time.Second * 30,
				msg: "Set the default Platform API to 30s: failed to parse timeout xxx: invalid duration error.",
			},
		}
		for _, c := range userCase {
			value, msg := parsePlatformAPITimeout(c.value)
			assert.Equal(t, c.respVal, value)
			assert.Equal(t, c.msg, msg)
		}
	})

	t.Run("Grafana env vars", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			`GF_AUTH_GENERIC_OAUTH_ALLOWED_DOMAINS='example.com'`,
			`GF_AUTH_GENERIC_OAUTH_ENABLED='true'`,
			`GF_PATHS_CONFIG="/etc/grafana/grafana.ini"`,
			`GF_PATHS_DATA="/var/lib/grafana"`,
			`GF_PATHS_HOME="/usr/share/grafana"`,
			`GF_PATHS_LOGS="/var/log/grafana"`,
			`GF_PATHS_PLUGINS="/var/lib/grafana/plugins"`,
			`GF_PATHS_PROVISIONING="/etc/grafana/provisioning"`,
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("k8s env vars", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			`MONITORING_SERVICE_PORT=tcp://10.96.84.150:443`,
			`MONITORING_SERVICE_PORT_443_TCP_PORT=443`,
			`MONITORING_SERVICE_SERVICE_HOST=10.96.84.150`,
			`KUBERNETES_PORT=tcp://10.96.0.1:443`,
			`KUBERNETES_PORT_443_TCP_PORT=443`,
			`KUBERNETES_SERVICE_HOST=10.96.0.1`,
			`MONITORING_SERVICE_PORT_443_TCP=tcp://10.96.84.150:443`,
			`KUBERNETES_PORT_443_TCP_PROTO=tcp`,
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})
}
