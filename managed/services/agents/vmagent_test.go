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

package agents

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestMaxScrapeSize(t *testing.T) {
	t.Run("by default 64MiB", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize="+maxScrapeSizeDefault)
	})
	t.Run("overridden with ENV", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		newValue := "16MiB"
		t.Setenv(maxScrapeSizeEnv, newValue)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize="+newValue)
	})
	t.Run("VMAGENT_ ENV variables", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		t.Setenv("VM_remoteWrite_basicAuth_password", "password")
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VM_remoteWrite_basicAuth_password=password")
	})
	t.Run("External Victoria Metrics ENV variables", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics:8428")
		require.NoError(t, err)
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	})
	t.Run("External Victoria Metrics with credentials in URL", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
		require.NoError(t, err)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://user:pass@victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=pass")
		// Should not contain server credentials
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})
	t.Run("External Victoria Metrics with username only in URL", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user@victoriametrics:8428")
		require.NoError(t, err)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://user@victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		// Should not contain password or server credentials
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})
	t.Run("External Victoria Metrics with special characters in credentials", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user%40domain:p%40ss%21@victoriametrics:8428")
		require.NoError(t, err)
		actual := vmAgentConfig("", params)
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://user%40domain:p%40ss%21@victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user@domain")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=p@ss!")
	})
	t.Run("System VMAGENT_ variables override defaults", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		// Set system environment variables that should override defaults
		t.Setenv("VMAGENT_loggerLevel", "DEBUG")
		t.Setenv("VMAGENT_remoteWrite_maxDiskUsagePerURL", "2147483648") // 2GB instead of 1GB
		actual := vmAgentConfig("", params)

		// Verify that system variables override defaults
		assert.Contains(t, actual.Env, "VMAGENT_loggerLevel=DEBUG")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_maxDiskUsagePerURL=2147483648")

		// Verify that non-overridden defaults are still present
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")

		// Verify that the default values are NOT present when overridden
		assert.NotContains(t, actual.Env, "VMAGENT_loggerLevel=INFO")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824")
	})
	t.Run("httpListenAddr is in Args not Env", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		actual := vmAgentConfig("", params)

		// Verify that httpListenAddr is in Args (not overrideable)
		found := false
		for _, arg := range actual.Args {
			if strings.Contains(arg, "-httpListenAddr=") {
				found = true
				break
			}
		}
		assert.True(t, found, "httpListenAddr should be in Args")

		// Verify that httpListenAddr is NOT in Env
		for _, env := range actual.Env {
			assert.NotContains(t, env, "VMAGENT_httpListenAddr=", "httpListenAddr should not be in Env")
		}
	})
}

func TestVMAgentExternalVM(t *testing.T) {
	testCases := []struct {
		name                  string
		vmURL                 string
		expectedUsername      string
		expectedPassword      string
		shouldHaveCredentials bool
	}{
		{
			name:                  "No credentials in URL",
			vmURL:                 "http://victoriametrics:8428",
			expectedUsername:      "",
			expectedPassword:      "",
			shouldHaveCredentials: false,
		},
		{
			name:                  "Username and password in URL",
			vmURL:                 "http://user:pass@victoriametrics:8428",
			expectedUsername:      "user",
			expectedPassword:      "pass",
			shouldHaveCredentials: true,
		},
		{
			name:                  "Username only in URL",
			vmURL:                 "http://user@victoriametrics:8428",
			expectedUsername:      "user",
			expectedPassword:      "",
			shouldHaveCredentials: true,
		},
		{
			name:                  "URL encoded credentials",
			vmURL:                 "http://user%40domain:p%40ss%21@victoriametrics:8428",
			expectedUsername:      "user@domain",
			expectedPassword:      "p@ss!",
			shouldHaveCredentials: true,
		},
		{
			name:                  "Complex password with special chars",
			vmURL:                 "http://admin:my%2Bpassword%3D123@victoriametrics:8428",
			expectedUsername:      "admin",
			expectedPassword:      "my+password=123",
			shouldHaveCredentials: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, tc.vmURL)
			require.NoError(t, err)

			actual := vmAgentConfig("", params)

			// External VM uses actual URL
			expectedURL := params.URL() + "api/v1/write"
			assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url="+expectedURL)

			if tc.shouldHaveCredentials {
				// Should have extracted credentials
				assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username="+tc.expectedUsername)
				if tc.expectedPassword != "" {
					assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password="+tc.expectedPassword)
				} else {
					assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=")
				}
			} else {
				// Should not have any credentials for external VM without auth
				assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=")
				assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=")
			}
			// Should not have server credentials
			assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
			assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		})
	}
}

func TestVMAgentInternalVM(t *testing.T) {
	t.Run("Internal VM uses server credentials", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params)

		// Internal VM should use templated URL
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")

		// Internal VM should use server credentials
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")

		// Should not have any extracted credentials
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=admin")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=secret")
	})
}

func TestExtractCredentialsFromURL(t *testing.T) {
	testCases := []struct {
		name             string
		url              string
		expectedUsername string
		expectedPassword string
	}{
		{
			name:             "Empty URL",
			url:              "",
			expectedUsername: "",
			expectedPassword: "",
		},
		{
			name:             "URL without credentials",
			url:              "http://example.com:8428",
			expectedUsername: "",
			expectedPassword: "",
		},
		{
			name:             "URL with username and password",
			url:              "http://user:pass@example.com:8428",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "URL with username only",
			url:              "http://user@example.com:8428",
			expectedUsername: "user",
			expectedPassword: "",
		},
		{
			name:             "URL with empty username and password",
			url:              "http://:pass@example.com:8428",
			expectedUsername: "",
			expectedPassword: "pass",
		},
		{
			name:             "URL with URL-encoded credentials",
			url:              "http://user%40domain:p%40ss%21@example.com:8428",
			expectedUsername: "user@domain",
			expectedPassword: "p@ss!",
		},
		{
			name:             "Invalid URL",
			url:              "://invalid-url",
			expectedUsername: "",
			expectedPassword: "",
		},
		{
			name:             "URL with complex password",
			url:              "http://admin:my%2Bpassword%3D123@example.com:8428",
			expectedUsername: "admin",
			expectedPassword: "my+password=123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			username, password := extractCredentialsFromURL(tc.url)
			assert.Equal(t, tc.expectedUsername, username, "Username mismatch")
			assert.Equal(t, tc.expectedPassword, password, "Password mismatch")
		})
	}
}
