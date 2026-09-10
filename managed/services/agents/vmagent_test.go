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

// assertNoEnvWithPrefix asserts that no entry in env starts with prefix. Unlike
// assert.NotContains on a slice (which matches exact elements only), this catches
// an entry with the given key regardless of its value.
func assertNoEnvWithPrefix(t *testing.T, env []string, prefix string) {
	t.Helper()
	for _, e := range env {
		assert.False(t, strings.HasPrefix(e, prefix), "unexpected env entry %q (prefix %q)", e, prefix)
	}
}

func TestMaxScrapeSize(t *testing.T) {
	t.Run("by default 64MiB", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize="+maxScrapeSizeDefault)
	})
	t.Run("overridden with ENV", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		newValue := "16MiB"
		t.Setenv(maxScrapeSizeEnv, newValue)
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize="+newValue)
	})
	t.Run("VMAGENT_ ENV variables", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		t.Setenv("VM_remoteWrite_basicAuth_password", "password")
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VM_remoteWrite_basicAuth_password=password")
	})
	t.Run("External Victoria Metrics ENV variables", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics:8428")
		require.NoError(t, err)
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	})
	t.Run("External Victoria Metrics with credentials in URL", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
		require.NoError(t, err)
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		// Credentials are stripped from the URL and passed via env only.
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=pass")
		// Should not contain server credentials
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		// Credentials provided via env only, args would silently override deployment-injected overrides
		for _, arg := range actual.Args {
			assert.NotContains(t, arg, "-remoteWrite.basicAuth.")
		}
	})
	t.Run("External Victoria Metrics with username only in URL", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user@victoriametrics:8428")
		require.NoError(t, err)
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		// Should not contain any password, nor the server username
		assertNoEnvWithPrefix(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	})
	t.Run("External Victoria Metrics with special characters in credentials", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user%40domain:p%40ss%21@victoriametrics:8428")
		require.NoError(t, err)
		actual := vmAgentConfig("", params, vmAgentDeployment{})
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user@domain")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=p@ss!")
		for _, arg := range actual.Args {
			assert.NotContains(t, arg, "-remoteWrite.basicAuth.")
		}
	})
	t.Run("System VMAGENT_ variables override defaults", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)
		// Set system environment variables that should override defaults
		t.Setenv("VMAGENT_loggerLevel", "DEBUG")
		t.Setenv("VMAGENT_remoteWrite_maxDiskUsagePerURL", "2147483648") // 2GB instead of 1GB
		actual := vmAgentConfig("", params, vmAgentDeployment{})

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
		actual := vmAgentConfig("", params, vmAgentDeployment{})

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
		name             string
		vmURL            string
		expectedUsername string
		expectedPassword string
	}{
		{
			name:  "No credentials in URL",
			vmURL: "http://victoriametrics:8428",
		},
		{
			name:             "Username and password in URL",
			vmURL:            "http://user:pass@victoriametrics:8428",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "Username only in URL",
			vmURL:            "http://user@victoriametrics:8428",
			expectedUsername: "user",
		},
		{
			name:             "Password only in URL",
			vmURL:            "http://:pass@victoriametrics:8428",
			expectedPassword: "pass",
		},
		{
			name:             "URL encoded credentials",
			vmURL:            "http://user%40domain:p%40ss%21@victoriametrics:8428",
			expectedUsername: "user@domain",
			expectedPassword: "p@ss!",
		},
		{
			name:             "Complex password with special chars",
			vmURL:            "http://admin:my%2Bpassword%3D123@victoriametrics:8428",
			expectedUsername: "admin",
			expectedPassword: "my+password=123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, tc.vmURL)
			require.NoError(t, err)

			actual := vmAgentConfig("", params, vmAgentDeployment{})

			// External VM pushes directly to the external URL, always without userinfo:
			// credentials travel via the basic-auth env vars only.
			assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")

			if tc.expectedUsername != "" {
				assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username="+tc.expectedUsername)
			} else {
				assertNoEnvWithPrefix(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=")
			}
			if tc.expectedPassword != "" {
				assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password="+tc.expectedPassword)
			} else {
				assertNoEnvWithPrefix(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=")
			}
			// Should not have server credentials
			assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
			assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		})
	}

	t.Run("External VM honors an injected remote-write URL", func(t *testing.T) {
		// The URL escape hatch applies to the direct-push path too: an operator-injected
		// VMAGENT_remoteWrite_url wins over the PMM_VM_URL-derived default, while the
		// URL-extracted credentials are still emitted as defaults.
		t.Setenv("VMAGENT_remoteWrite_url", "https://other.example.com/api/v1/write")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=https://other.example.com/api/v1/write")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=pass")
	})
}

func TestVMAgentInternalVM(t *testing.T) {
	t.Run("Internal VM uses server credentials", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{})

		// Internal VM should use templated URL
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")

		// Internal VM should use server credentials
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")

		// Should not have any extracted credentials
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=admin")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=secret")
	})

	t.Run("Internal VM sets expected defaults", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{})

		// Positive assertions for defaults that are otherwise only checked indirectly.
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824")
		assert.Contains(t, actual.Env, "VMAGENT_loggerLevel=INFO")
	})

	t.Run("Internal VM honors operator-injected basic-auth", func(t *testing.T) {
		// No standalone deployment mode sets VMAGENT_remoteWrite_basicAuth_* by default, so an
		// injected value is a deliberate operator choice (e.g. a shared PMM service account for
		// all client writes) and wins over the per-client server credentials. This matches the
		// long-standing standalone behavior; only HA drops these (see TestVMAgentHA).
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "injected_user")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "injected_pass")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=injected_user")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=injected_pass")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})

	t.Run("Internal VM keeps injected basic-auth when a remote-write URL is also injected", func(t *testing.T) {
		// Injected credentials are honored in standalone regardless of the URL; with an injected
		// redirect they travel to the custom endpoint instead of the server proxy.
		t.Setenv("VMAGENT_remoteWrite_url", "https://collector.example.com/api/v1/write")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "injected_user")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "injected_pass")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=injected_user")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=injected_pass")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})
}

// TestVMAgentStandaloneServerAgent pins that isServerAgent is inert outside HA.
// PMM Server's own built-in agent runs with isServerAgent=true in every
// deployment; only in HA does that flag bypass the write-proxy. In standalone
// (haEnabled=false) routing must be identical to a normal client, whatever the
// VM URL. Guards against isServerAgent leaking into the standalone dispatch.
func TestVMAgentStandaloneServerAgent(t *testing.T) {
	deployment := vmAgentDeployment{isServerAgent: true} // haEnabled stays false

	t.Run("standalone server agent with internal VM routes through the server proxy", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params, deployment)

		// Same as a plain client: templated proxy URL + server credentials.
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})

	t.Run("standalone server agent with external VM pushes directly", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, deployment)

		// Same as a plain client: direct push, URL creds stripped to env, no proxy template.
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=pass")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	})
}

func TestVMAgentHA(t *testing.T) {
	// In HA mode the configured VictoriaMetrics URL (PMM_VM_URL) points at an in-cluster-only
	// endpoint that external pmm-clients cannot reach, so agents must route metric writes through
	// the PMM server regardless of the external-looking URL (PMM-14678).
	t.Run("HA routes external VM through the server with server credentials", func(t *testing.T) {
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		// Must use the server-proxied path + server credentials...
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")

		// ...and must NOT leak the external VM URL or its embedded credentials (env or args).
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url=http://user:pass@victoriametrics:8428/api/v1/write")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=pass")
		for _, arg := range actual.Args {
			assert.NotContains(t, arg, "-remoteWrite.basicAuth.username=user")
			assert.NotContains(t, arg, "-remoteWrite.basicAuth.password=pass")
		}
	})

	t.Run("HA server agent pushes directly to the in-cluster VM, not through the proxy", func(t *testing.T) {
		// PMM Server's own built-in agent is in-cluster and reaches vmauth directly, so it must NOT
		// be routed through the server write-proxy: it carries no Grafana credential for the proxy's
		// auth and would otherwise fail to push its self-monitoring metrics (all pmm-ha nodes then
		// show Unknown). It keeps the pre-proxy behavior - push straight to the injected VM endpoint
		// with the deployment's VM credentials (PMM-14678 follow-up).
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "victoriametrics_pmm")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "vm-password")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics_pmm:vm-password@vmauth:8427")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true, isServerAgent: true})

		// Direct push to vmauth with the VM credentials...
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://vmauth:8427/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=vm-password")
		// ...and NOT the server-proxy path or its template credentials.
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	})

	t.Run("HA ignores deployment VM basic-auth override and uses server credentials", func(t *testing.T) {
		// The HA chart injects the VictoriaMetrics credentials as VMAGENT_remoteWrite_basicAuth_*
		// (for a direct push to vmauth). When routing through the server those must be ignored,
		// otherwise the server's /victoriametrics/ proxy rejects them with 401 (PMM-14678).
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "victoriametrics_pmm")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "vm-password")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics_pmm:vm-password@vmauth:8427")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=vm-password")
	})

	t.Run("HA honors an operator-injected remote-write URL", func(t *testing.T) {
		// No deployment mode ever sets VMAGENT_remoteWrite_url, so its presence is an explicit
		// operator override (e.g. an all-in-cluster HA setup pushing directly to vmauth to skip
		// the proxy hop) and wins over the server-proxy default, same as in non-HA.
		t.Setenv("VMAGENT_remoteWrite_url", "http://vmauth:8427/api/v1/write")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics_pmm:vm-password@vmauth:8427")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://vmauth:8427/api/v1/write")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		// With no credentials injected alongside the URL, the server-credential defaults
		// still apply - an operator redirecting writes is expected to inject matching
		// credentials too (see the paired-injection subtest below).
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})

	t.Run("HA honors injected remote-write URL together with injected basic-auth", func(t *testing.T) {
		// When an operator injects a remote-write URL, the target is no longer the server proxy,
		// so injected credentials travel with it (the chart-injected vmauth creds match the vmauth
		// endpoint in this direct-push setup). Same escape-hatch semantics as non-HA.
		t.Setenv("VMAGENT_remoteWrite_url", "http://vmauth:8427/api/v1/write")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "victoriametrics_pmm")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "vm-password")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics_pmm:vm-password@vmauth:8427")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url=http://vmauth:8427/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=vm-password")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	})

	t.Run("HA routes external VM without credentials through the server", func(t *testing.T) {
		// Even when PMM_VM_URL carries no basic-auth, HA must still route through the server proxy
		// with server credentials rather than pushing directly to the in-cluster VM endpoint.
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics:8428")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write")
	})

	t.Run("HA emits the canonical Args set with no external VM basic-auth args", func(t *testing.T) {
		// Credentials are passed via env only; no scenario may append basic-auth args.
		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.ElementsMatch(t, []string{
			"-envflag.enable=true",
			"-envflag.prefix=VMAGENT_",
			"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
			"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
			"-httpListenAddr=127.0.0.1:{{.listen_port}}",
		}, actual.Args)
	})

	t.Run("HA preserves injected tuning vars while dropping basic-auth", func(t *testing.T) {
		// The HA credential drop must be scoped to exactly the two basic-auth keys: other
		// injected VMAGENT_ variables are the documented central-tuning mechanism and must
		// keep passing through.
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "16MiB")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "victoriametrics_pmm")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "vm-password")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://victoriametrics_pmm:vm-password@vmauth:8427")
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.Contains(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize=16MiB")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=vm-password")
	})

	t.Run("HA with internal VM URL routes through the server proxy", func(t *testing.T) {
		// The dispatch must not depend on ExternalVM(): HA routes through the server proxy and
		// drops injected credentials even when PMM_VM_URL is the default internal address.
		t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "victoriametrics_pmm")
		t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "vm-password")

		params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
		require.NoError(t, err)

		actual := vmAgentConfig("", params, vmAgentDeployment{haEnabled: true})

		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
		assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm")
		assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=vm-password")
	})
}

func TestVMAgentExternalVMBasicAuthOverridePreserved(t *testing.T) {
	// Guards that the HA-only credential drop never applies to a genuine external VM (non-HA):
	// a deployment-supplied VMAGENT_remoteWrite_basicAuth_* override must be preserved.
	t.Setenv("VMAGENT_remoteWrite_basicAuth_username", "override_user")
	t.Setenv("VMAGENT_remoteWrite_basicAuth_password", "override_pass")

	params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, "http://user:pass@victoriametrics:8428")
	require.NoError(t, err)

	actual := vmAgentConfig("", params, vmAgentDeployment{})

	// The injected override survives and wins over the URL-extracted credentials...
	assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=override_user")
	assert.Contains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=override_pass")
	assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username=user")
	assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password=pass")
	// ...and server credentials are not used for an external VM.
	assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_username={{.server_username}}")
	assert.NotContains(t, actual.Env, "VMAGENT_remoteWrite_basicAuth_password={{.server_password}}")
	// No basic-auth args either: args would outrank the injected env vars and defeat the override.
	for _, arg := range actual.Args {
		assert.NotContains(t, arg, "-remoteWrite.basicAuth.")
	}
}

func TestVMAgentScrapeConfigPassthrough(t *testing.T) {
	params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, models.VMBaseURL)
	require.NoError(t, err)

	scrapeCfg := "global:\n  scrape_interval: 15s\n"
	actual := vmAgentConfig(scrapeCfg, params, vmAgentDeployment{})

	assert.Equal(t, scrapeCfg, actual.TextFiles["vmagentscrapecfg"])
}

func TestSplitURLCredentials(t *testing.T) {
	testCases := []struct {
		name             string
		url              string
		expectedURL      string
		expectedUsername string
		expectedPassword string
	}{
		{
			name:        "Empty URL",
			url:         "",
			expectedURL: "",
		},
		{
			name:        "URL without credentials",
			url:         "http://example.com:8428",
			expectedURL: "http://example.com:8428",
		},
		{
			name:             "URL with username and password",
			url:              "http://user:pass@example.com:8428",
			expectedURL:      "http://example.com:8428",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
		{
			name:             "URL with username only",
			url:              "http://user@example.com:8428",
			expectedURL:      "http://example.com:8428",
			expectedUsername: "user",
		},
		{
			name:             "URL with empty username and password",
			url:              "http://:pass@example.com:8428",
			expectedURL:      "http://example.com:8428",
			expectedPassword: "pass",
		},
		{
			name:             "URL with URL-encoded credentials",
			url:              "http://user%40domain:p%40ss%21@example.com:8428",
			expectedURL:      "http://example.com:8428",
			expectedUsername: "user@domain",
			expectedPassword: "p@ss!",
		},
		{
			name:        "Invalid URL",
			url:         "://invalid-url",
			expectedURL: "://invalid-url",
		},
		{
			name:             "URL with complex password",
			url:              "http://admin:my%2Bpassword%3D123@example.com:8428",
			expectedURL:      "http://example.com:8428",
			expectedUsername: "admin",
			expectedPassword: "my+password=123",
		},
		{
			name:             "Trailing slash, path and query preserved",
			url:              "http://user:pass@example.com:8428/prometheus/?timeout=5s",
			expectedURL:      "http://example.com:8428/prometheus/?timeout=5s",
			expectedUsername: "user",
			expectedPassword: "pass",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanURL, username, password := splitURLCredentials(tc.url)
			assert.Equal(t, tc.expectedURL, cleanURL, "URL mismatch")
			assert.Equal(t, tc.expectedUsername, username, "Username mismatch")
			assert.Equal(t, tc.expectedPassword, password, "Password mismatch")
		})
	}
}
