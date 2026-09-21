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

	t.Run("SEP env variables", func(t *testing.T) {
		t.Parallel()

		envs := []string{"PMM_ENABLE_SEP=1", "PMM_SEP_POSTGRES_PASSWORD=s3cr3t"}
		expectedEnvVars := &models.ChangeSettingsParams{}

		gotEnvVars, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Equal(t, expectedEnvVars, gotEnvVars)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
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

	t.Run("Skipped clickhouse env vars", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PMM_CLICKHOUSE_DATABASE=pmm",
			"PMM_CLICKHOUSE_ADDR=127.0.0.1:9000",
			"PMM_CLICKHOUSE_USER=default",
			"PMM_CLICKHOUSE_PASSWORD=secret",
			"PMM_CLICKHOUSE_DATASOURCE_USER=grafana",
			"PMM_CLICKHOUSE_DATASOURCE_PASSWORD=secret",
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

	t.Run("Skipped internal node name prefixes env var", func(t *testing.T) {
		t.Parallel()

		envs := []string{"PMM_INTERNAL_NODE_NAME_PREFIXES=pmm-pmm-ha-pg-db-"}
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

	t.Run("VMAGENT_remoteWrite_url without credentials warns", func(t *testing.T) {
		t.Parallel()

		envs := []string{"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write"}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "VMAGENT_remoteWrite_url redirects the metric writes")
	})

	t.Run("VMAGENT_remoteWrite_url with credentials does not warn", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			"VMAGENT_remoteWrite_basicAuth_username=collector",
			"VMAGENT_remoteWrite_basicAuth_password=secret",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("VMAGENT credentials without a remote-write URL do not warn", func(t *testing.T) {
		t.Parallel()

		// Credentials without a URL override (what HA charts injected before the PMM_HA_VM_* keys).
		envs := []string{
			"VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm",
			"VMAGENT_remoteWrite_basicAuth_password=vm-password",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("VMAGENT_remoteWrite_url set but empty is ignored with a warning", func(t *testing.T) {
		t.Parallel()

		// vmagent exits on an empty URL; ignoring the variable keeps PMM's default write path.
		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_remoteWrite_url="})
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "VMAGENT_remoteWrite_url is set but empty and is ignored")
	})

	t.Run("an empty basic-auth pair is ignored, not counted as a credential", func(t *testing.T) {
		t.Parallel()

		// A Helm value left blank: the URL override is then unauthenticated and must be reported as such.
		envs := []string{
			"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			"VMAGENT_remoteWrite_basicAuth_username=",
			"VMAGENT_remoteWrite_basicAuth_password=",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 3)
		assert.Contains(t, gotWarns[0], "VMAGENT_remoteWrite_basicAuth_username is set but empty")
		assert.Contains(t, gotWarns[1], "VMAGENT_remoteWrite_basicAuth_password is set but empty")
		assert.Contains(t, gotWarns[2], "redirects the metric writes")
	})

	t.Run("an empty tuning variable is ignored with a warning", func(t *testing.T) {
		t.Parallel()

		// vmagent panics on an empty -loggerLevel.
		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_loggerLevel="})
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "VMAGENT_loggerLevel is set but empty and is ignored")
	})

	t.Run("VMAGENT_remoteWrite_url with half a basic-auth pair warns about the pair", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			"VMAGENT_remoteWrite_basicAuth_username=collector",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "only one half")
	})

	t.Run("VMAGENT_remoteWrite_url with credentials in the URL does not warn", func(t *testing.T) {
		t.Parallel()

		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_remoteWrite_url=https://collector:secret@collector.example.com/api/v1/write"})
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("VMAGENT_remoteWrite_url with a bearer token does not warn", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			"VMAGENT_remoteWrite_bearerToken=abc",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("an upper-cased VMAGENT_REMOTEWRITE_URL is inert and does not warn", func(t *testing.T) {
		t.Parallel()

		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_REMOTEWRITE_URL=https://collector.example.com/api/v1/write"})
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("a URL override among other variables produces exactly one warning", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"PMM_ENABLE_UPDATES=true",
			"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			"VMAGENT_loggerLevel=INFO",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Len(t, gotWarns, 1)
	})

	t.Run("half a basic-auth pair warns even without a remote-write URL", func(t *testing.T) {
		t.Parallel()

		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_remoteWrite_basicAuth_password=vm-password"})
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "only one half")
	})

	t.Run("half a basic-auth pair warns alongside an additive method", func(t *testing.T) {
		t.Parallel()

		// An additive method composes with a pair rather than replacing one, so it must not make
		// a lone half look complete: PMM still withholds its own credential for that half, and
		// vmagent would send the username with no password.
		for _, additive := range []string{
			"VMAGENT_remoteWrite_headers=AccountID: 1",
			"VMAGENT_remoteWrite_tlsCertFile=/run/secrets/client.pem",
		} {
			envs := []string{"VMAGENT_remoteWrite_basicAuth_username=collector", additive}
			_, gotErrs, gotWarns := ParseEnvVars(envs)
			assert.Nil(t, gotErrs, additive)
			require.Len(t, gotWarns, 1, additive)
			assert.Contains(t, gotWarns[0], "only one half", additive)
		}
	})

	t.Run("an additive method with a whole basic-auth pair does not warn", func(t *testing.T) {
		t.Parallel()

		envs := []string{
			"VMAGENT_remoteWrite_basicAuth_username=collector",
			"VMAGENT_remoteWrite_basicAuth_password=secret",
			"VMAGENT_remoteWrite_headers=AccountID: 1",
		}

		_, gotErrs, gotWarns := ParseEnvVars(envs)
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("VMAGENT_remoteWrite_url that does not parse is an error that never echoes it", func(t *testing.T) {
		t.Parallel()

		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_remoteWrite_url=https://collector:secret@[::1"})
		require.Len(t, gotErrs, 1)
		assert.Contains(t, gotErrs[0].Error(), "not a valid URL")
		assert.NotContains(t, gotErrs[0].Error(), "secret")
		assert.Nil(t, gotWarns)
	})

	t.Run("VMAGENT_remoteWrite_url with an @ outside the userinfo still warns about credentials", func(t *testing.T) {
		t.Parallel()

		_, gotErrs, gotWarns := ParseEnvVars([]string{"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write?tenant=a@b"})
		assert.Nil(t, gotErrs)
		require.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "redirects the metric writes")
	})

	t.Run("VMAGENT_remoteWrite_url may be a client-side template or a list", func(t *testing.T) {
		t.Parallel()

		for _, value := range []string{
			"{{.server_url}}/victoriametrics/api/v1/write",
			"https://a.example.com/api/v1/write,https://b.example.com/api/v1/write",
		} {
			_, gotErrs, _ := ParseEnvVars([]string{"VMAGENT_remoteWrite_url=" + value})
			assert.Nil(t, gotErrs, value)
		}
	})

	t.Run("PMM_VM_URL must be an http or https URL with a host", func(t *testing.T) {
		t.Parallel()

		for _, bad := range []string{"vm:8428", "//vm:8428/", "vm.example.com", "ftp://vm:8428"} {
			_, gotErrs, _ := ParseEnvVars([]string{"PMM_VM_URL=" + bad})
			assert.Len(t, gotErrs, 1, bad)
		}
		_, gotErrs, gotWarns := ParseEnvVars([]string{"PMM_VM_URL=http://user:pass@vm:8428"})
		assert.Nil(t, gotErrs)
		assert.Nil(t, gotWarns)
	})

	t.Run("an empty PMM_VM_URL is ignored, not rejected", func(t *testing.T) {
		t.Parallel()

		// A template that renders an unset value produces this, and it started PMM Server before
		// PMM_VM_URL was validated. kingpin falls back to its default for it either way.
		_, gotErrs, gotWarns := ParseEnvVars([]string{"PMM_VM_URL="})
		assert.Nil(t, gotErrs)
		assert.Len(t, gotWarns, 1)
		assert.Contains(t, gotWarns[0], "PMM_VM_URL is set but empty")
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

func TestRedactSecretEnvVar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{key: "PMM_CLICKHOUSE_DATASOURCE_PASSWORD", value: "s3cret", expected: "<redacted>"},
		{key: "PMM_CLICKHOUSE_PASSWORD", value: "s3cret", expected: "<redacted>"},
		{key: "PMM_POSTGRES_DBPASSWORD", value: "s3cret", expected: "<redacted>"},
		{key: "GF_SECURITY_ADMIN_PASSWORD", value: "s3cret", expected: "<redacted>"},
		{key: "AWS_SECRET_KEY", value: "s3cret", expected: "<redacted>"},
		{key: "PMM_CLICKHOUSE_DATASOURCE_USER", value: "grafana", expected: "grafana"},
		{key: "PMM_DATA_RETENTION", value: "72h", expected: "72h"},
		{key: "PMM_VM_URL", value: "http://victoriametrics_pmm:vm-password@vmauth:8427/", expected: "http://<redacted>@vmauth:8427/"},
		{key: "PMM_VM_URL", value: "http://vmauth:8427/", expected: "http://vmauth:8427/"},
		{key: "VMAGENT_remoteWrite_url", value: "https://user:p%40ss@collector.example.com/api/v1/write", expected: "https://<redacted>@collector.example.com/api/v1/write"},
		{key: "VMAGENT_remoteWrite_url", value: "https://collector.example.com/api/v1/write?tenant=a@b", expected: "https://collector.example.com/api/v1/write?tenant=a@b"},
		{key: "PMM_VM_URL", value: "https://cdn.example.com/logo@2x.png", expected: "https://cdn.example.com/logo@2x.png"},
		{key: "PMM_VM_URL", value: "http://user:secret@[::1", expected: "<redacted>"},
		{key: "PMM_VM_URL", value: "user:secret@victoriametrics:8428", expected: "<redacted>@victoriametrics:8428"},
		{key: "PMM_VM_URL", value: "user:secret@victoriametrics:8428/path", expected: "<redacted>@victoriametrics:8428/path"},
		{key: "PMM_VM_URL", value: "victoriametrics:8428", expected: "victoriametrics:8428"},
		{key: "PMM_PUBLIC_ADDRESS", value: "pmm.example.com", expected: "pmm.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, redactSecretEnvVar(tt.key, tt.value))
		})
	}
}

func TestVMAgentRemoteWriteReplacesBasicAuth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{name: "nothing", env: map[string]string{"VMAGENT_loggerLevel": "INFO"}},
		{name: "username only", env: map[string]string{EnvVMAgentRemoteWriteUsername: "u"}, want: true},
		{name: "password file only", env: map[string]string{"VMAGENT_remoteWrite_basicAuth_passwordFile": "/run/secrets/p"}, want: true},
		{name: "basic-auth pair", env: map[string]string{EnvVMAgentRemoteWriteUsername: "u", EnvVMAgentRemoteWritePassword: "p"}, want: true},
		{name: "bearer token", env: map[string]string{"VMAGENT_remoteWrite_bearerToken": "t"}, want: true},
		{name: "OAuth2 client", env: map[string]string{"VMAGENT_remoteWrite_oauth2_clientID": "c"}, want: true},
		{name: "custom headers compose", env: map[string]string{"VMAGENT_remoteWrite_headers": "X-Scope-OrgID:1"}},
		{name: "client certificate composes", env: map[string]string{"VMAGENT_remoteWrite_tlsCertFile": "/run/secrets/c"}},
		{name: "upper-cased names are inert", env: map[string]string{"VMAGENT_REMOTEWRITE_BEARERTOKEN": "t"}},
		// The pair with VMAgentRemoteWriteAuthFromEnv: an additive method alongside half a pair
		// still replaces PMM's credential, so that half must still be reported as incomplete.
		{
			name: "custom headers alongside half a pair still replace",
			env:  map[string]string{EnvVMAgentRemoteWriteUsername: "u", "VMAGENT_remoteWrite_headers": "X-Scope-OrgID:1"},
			want: true,
		},
		{
			name: "a client certificate alongside half a pair still replaces",
			env:  map[string]string{EnvVMAgentRemoteWritePassword: "p", "VMAGENT_remoteWrite_tlsCertFile": "/run/secrets/c"},
			want: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, VMAgentRemoteWriteReplacesBasicAuth(tc.env))
		})
	}
}

func TestVMAgentRemoteWriteAuthFromEnv(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		env  map[string]string
		want VMAgentRemoteWriteAuth
	}{
		{name: "nothing", env: map[string]string{"VMAGENT_loggerLevel": "INFO"}, want: VMAgentRemoteWriteAuthNone},
		{name: "username only", env: map[string]string{EnvVMAgentRemoteWriteUsername: "u"}, want: VMAgentRemoteWriteAuthPartial},
		{name: "password only", env: map[string]string{EnvVMAgentRemoteWritePassword: "p"}, want: VMAgentRemoteWriteAuthPartial},
		{name: "basic-auth pair", env: map[string]string{EnvVMAgentRemoteWriteUsername: "u", EnvVMAgentRemoteWritePassword: "p"}, want: VMAgentRemoteWriteAuthComplete},
		{name: "username with a password file", env: map[string]string{EnvVMAgentRemoteWriteUsername: "u", "VMAGENT_remoteWrite_basicAuth_passwordFile": "/run/secrets/p"}, want: VMAgentRemoteWriteAuthComplete},
		{name: "password file only", env: map[string]string{"VMAGENT_remoteWrite_basicAuth_passwordFile": "/run/secrets/p"}, want: VMAgentRemoteWriteAuthPartial},
		{name: "username file only", env: map[string]string{"VMAGENT_remoteWrite_basicAuth_usernameFile": "/run/secrets/u"}, want: VMAgentRemoteWriteAuthPartial},
		{name: "username file with a password", env: map[string]string{"VMAGENT_remoteWrite_basicAuth_usernameFile": "/run/secrets/u", EnvVMAgentRemoteWritePassword: "p"}, want: VMAgentRemoteWriteAuthComplete},
		{name: "username and password files", env: map[string]string{"VMAGENT_remoteWrite_basicAuth_usernameFile": "/run/secrets/u", "VMAGENT_remoteWrite_basicAuth_passwordFile": "/run/secrets/p"}, want: VMAgentRemoteWriteAuthComplete},
		{name: "bearer token", env: map[string]string{"VMAGENT_remoteWrite_bearerToken": "t"}, want: VMAgentRemoteWriteAuthComplete},
		{name: "custom headers", env: map[string]string{"VMAGENT_remoteWrite_headers": "X-Auth: t"}, want: VMAgentRemoteWriteAuthComplete},
		{name: "client certificate", env: map[string]string{"VMAGENT_remoteWrite_tlsCertFile": "/run/secrets/c"}, want: VMAgentRemoteWriteAuthComplete},
		// An additive method composes with a pair instead of replacing one, so it must not report
		// a lone half as complete: PMM withholds its own credential for that half either way.
		{
			name: "custom headers do not complete half a pair",
			env:  map[string]string{EnvVMAgentRemoteWriteUsername: "u", "VMAGENT_remoteWrite_headers": "X-Auth: t"},
			want: VMAgentRemoteWriteAuthPartial,
		},
		{
			name: "a client certificate does not complete half a pair",
			env:  map[string]string{EnvVMAgentRemoteWritePassword: "p", "VMAGENT_remoteWrite_tlsCertFile": "/run/secrets/c"},
			want: VMAgentRemoteWriteAuthPartial,
		},
		{
			name: "custom headers compose with a whole pair",
			env:  map[string]string{EnvVMAgentRemoteWriteUsername: "u", EnvVMAgentRemoteWritePassword: "p", "VMAGENT_remoteWrite_headers": "X-Auth: t"},
			want: VMAgentRemoteWriteAuthComplete,
		},
		// An exclusive method stays above the half-a-pair branch: it takes the place of a pair.
		{
			name: "a bearer token outranks half a pair",
			env:  map[string]string{EnvVMAgentRemoteWriteUsername: "u", "VMAGENT_remoteWrite_bearerToken": "t"},
			want: VMAgentRemoteWriteAuthComplete,
		},
		{name: "upper-cased names are inert", env: map[string]string{"VMAGENT_REMOTEWRITE_BASICAUTH_USERNAME": "u", "VMAGENT_REMOTEWRITE_BASICAUTH_PASSWORD": "p"}, want: VMAgentRemoteWriteAuthNone},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, VMAgentRemoteWriteAuthFromEnv(tc.env))
		})
	}
}
