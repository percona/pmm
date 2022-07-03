// pmm-managed
// Copyright (C) 2017 Percona LLC
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

// Package validators contains environment variables validator.
package envvars

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestEnvVarValidator(t *testing.T) {
	t.Parallel()

	t.Run("Valid env variables", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"DISABLE_UPDATES=True",
			"DISABLE_TELEMETRY=True",
			"METRICS_RESOLUTION=5m",
			"METRICS_RESOLUTION_MR=5s",
			"METRICS_RESOLUTION_LR=1h",
			"DATA_RETENTION=72h",
		}
		expectedEnvVars := &models.ChangeSettingsParams{
			DataRetention:    72 * time.Hour,
			DisableTelemetry: true,
			DisableSTT:       false,
			DisableUpdates:   true,
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Minute,
				MR: 5 * time.Second,
				LR: time.Hour,
			},
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, gotEnvVars, expectedEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("Unknown env variables", func(t *testing.T) {
		t.Parallel()

		envs := []string{"UNKNOWN_VAR=VAL", "ANOTHER_UNKNOWN_VAR=VAL"}
		expectedEnvVars := &models.ChangeSettingsParams{}
		expectedWarns := []string{
			`unknown environment variable "UNKNOWN_VAR=VAL"`,
			`unknown environment variable "ANOTHER_UNKNOWN_VAR=VAL"`,
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, gotEnvVars, expectedEnvVars)
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
		}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, gotEnvVars, expectedEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("Invalid env variables values", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"DISABLE_UPDATES",
			"DISABLE_TELEMETRY",
			"DISABLE_UPDATES=5",
			"DISABLE_TELEMETRY=X",
			"METRICS_RESOLUTION=5f",
			"METRICS_RESOLUTION_MR=s5",
			"METRICS_RESOLUTION_LR=1hour",
			"DATA_RETENTION=keep one week",
		}
		expectedEnvVars := &models.ChangeSettingsParams{}
		expectedErrs := []error{
			fmt.Errorf(`failed to parse environment variable "DISABLE_UPDATES"`),
			fmt.Errorf(`failed to parse environment variable "DISABLE_TELEMETRY"`),
			fmt.Errorf(`invalid value "5" for environment variable "DISABLE_UPDATES"`),
			fmt.Errorf(`invalid value "x" for environment variable "DISABLE_TELEMETRY"`),
			fmt.Errorf(`environment variable "METRICS_RESOLUTION=5f" has invalid duration 5f`),
			fmt.Errorf(`environment variable "METRICS_RESOLUTION_MR=s5" has invalid duration s5`),
			fmt.Errorf(`environment variable "METRICS_RESOLUTION_LR=1hour" has invalid duration 1hour`),
			fmt.Errorf(`environment variable "DATA_RETENTION=keep one week" has invalid duration keep one week`),
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, gotEnvVars, expectedEnvVars)
		assert.Equal(t, gotErrs, expectedErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("SAAS env vars with warnings", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PERCONA_TEST_SAAS_HOST=host:333",
		}
		expectedEnvVars := &models.ChangeSettingsParams{}
		expectedWarns := []string{
			`environment variable "PERCONA_TEST_SAAS_HOST" IS NOT SUPPORTED and WILL BE REMOVED IN THE FUTURE`,
		}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Equal(t, expectedWarns, gotWarns)
	})

	t.Run("SAAS env vars with errors", func(t *testing.T) {
		t.Parallel()

		for _, k := range []string{
			"PERCONA_TEST_AUTH_HOST",
			"PERCONA_TEST_CHECKS_HOST",
			"PERCONA_TEST_TELEMETRY_HOST",
		} {
			expected := fmt.Errorf(`environment variable %q is removed and replaced by "PERCONA_TEST_SAAS_HOST"`, k)
			envs := []string{k + "=host:333"}
			_, gotErrs, gotWarns := ParseEnvVars(envs)
			assert.Equal(t, []error{expected}, gotErrs)
			assert.Nil(t, gotWarns)
		}
	})

	t.Run("Parse SAAS host", func(t *testing.T) {
		t.Parallel()

		userCase := []struct {
			value   string
			err     string
			respVal string
		}{
			{value: "host", err: "", respVal: "host"},
			{value: ":111", err: `environment variable "PERCONA_TEST_SAAS_HOST" has invalid format ":111". Expected host[:port]`, respVal: ""},
			{value: "host:555", err: "", respVal: "host"},
			{value: "[2001:cafe:8221:9a0f:4dc7:4bb:8581:d186]:333", err: "", respVal: "2001:cafe:8221:9a0f:4dc7:4bb:8581:d186"},
			{value: "ho:st:444", err: "address ho:st:444: too many colons in address", respVal: ""},
		}
		for _, c := range userCase {
			value, err := parseSAASHost(c.value)
			assert.Equal(t, c.respVal, value)
			if c.err == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, c.err, err.Error())
			}
		}
	})

	t.Run("Parse Platform API Timeout", func(t *testing.T) {
		t.Parallel()

		userCase := []struct {
			value   string
			respVal time.Duration
			msg     string
		}{
			{value: "", respVal: time.Second * 30, msg: "Environment variable \"PERCONA_PLATFORM_API_TIMEOUT\" is not set, using \"30s\" as a default timeout for platform API."},
			{value: "10s", respVal: time.Second * 10, msg: "Using \"10s\" as a timeout for platform API."},
			{value: "xxx", respVal: time.Second * 30, msg: "Using \"30s\" as a default: failed to parse platform API timeout \"xxx\": invalid duration error."},
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
		assert.Equal(t, gotEnvVars, expectedEnvVars)
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
		assert.Equal(t, gotEnvVars, expectedEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})
}
