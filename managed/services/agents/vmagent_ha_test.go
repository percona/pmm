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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// legacyChartCreds injects the vmauth credential under the VMAGENT_ passthrough names, as pmm-ha
// chart releases before the PMM_HA_VM_* keys did. Kept because that shape exists during a chart
// and server upgrade window, and because an operator can still inject the pair deliberately.
func legacyChartCreds(t *testing.T) {
	t.Helper()
	t.Setenv(envRemoteWriteUsername, "victoriametrics_pmm")
	t.Setenv(envRemoteWritePassword, "vm-password")
}

func TestHARemoteWrite(t *testing.T) {
	vmCreds := remoteWrite{username: "victoriametrics_pmm", password: "vm-password", source: credentialVMURL}

	t.Run("clients write to their PMM Server address with the VM credential", func(t *testing.T) {
		want := vmCreds
		want.url = serverProxyWriteURL
		rw, err := haRemoteWrite(newVMParams(t, testVMAuth), false)
		require.NoError(t, err)
		assert.Equal(t, want, rw)
	})

	t.Run("the server agent writes to vmauth directly", func(t *testing.T) {
		want := vmCreds
		want.url = testVMAuthWrite
		rw, err := haRemoteWrite(newVMParams(t, testVMAuth), true)
		require.NoError(t, err)
		assert.Equal(t, want, rw)
	})

	t.Run("a VM URL without credentials yields pairs without any credential", func(t *testing.T) {
		client, err := haRemoteWrite(newVMParams(t, testVMAuthNoCreds), false)
		require.NoError(t, err)
		assert.Equal(t, remoteWrite{url: serverProxyWriteURL, source: credentialNone}, client)
		server, err := haRemoteWrite(newVMParams(t, testVMAuthNoCreds), true)
		require.NoError(t, err)
		assert.Equal(t, remoteWrite{url: testVMAuthWrite, source: credentialNone}, server)
	})

	t.Run("an internal VM URL falls back to the standalone pair", func(t *testing.T) {
		params := newVMParams(t, models.VMBaseURL)
		client, err := haRemoteWrite(params, false)
		require.NoError(t, err)
		assert.Equal(t, serverProxyRemoteWrite(), client)
		server, err := haRemoteWrite(params, true)
		require.NoError(t, err)
		assert.Equal(t, serverProxyRemoteWrite(), server)
	})
}

func TestHARemoteWriteWarning(t *testing.T) {
	clearVMAgentEnv(t)

	t.Run("internal VM URL is unsupported in HA", func(t *testing.T) {
		assert.Contains(t, HARemoteWriteWarning(newVMParams(t, models.VMBaseURL)), "not supported")
	})

	t.Run("external VM URL with credentials is fine", func(t *testing.T) {
		assert.Empty(t, HARemoteWriteWarning(newVMParams(t, testVMAuth)))
	})

	t.Run("external VM URL without credentials and nothing injected warns", func(t *testing.T) {
		assert.Contains(t, HARemoteWriteWarning(newVMParams(t, testVMAuthNoCreds)), "carries no credentials")
	})

	t.Run("injected credentials satisfy a credential-less URL", func(t *testing.T) {
		legacyChartCreds(t)
		assert.Empty(t, HARemoteWriteWarning(newVMParams(t, testVMAuthNoCreds)))
	})

	t.Run("an unparsable URL is left to startup validation", func(t *testing.T) {
		assert.Empty(t, HARemoteWriteWarning(fakeVMParams{externalVM: true, url: "http://[::1"}))
	})
}

func TestVMAgentHA(t *testing.T) {
	clearVMAgentEnv(t)
	client := vmAgentDeployment{haEnabled: true}
	server := vmAgentDeployment{haEnabled: true, isServerAgent: true}
	build := func(t *testing.T, vmURL string, d vmAgentDeployment) ([]string, []string) {
		t.Helper()
		actual := mustVMAgentConfig(t, "", newVMParams(t, vmURL), d)
		return actual.Env, actual.Args
	}

	t.Run("clients write via their PMM Server address with the VM credential", func(t *testing.T) {
		env, args := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
		assertNoServerCredentialTemplates(t, env)
		assertNotAnywhere(t, nil, args, "basicAuth.")
	})

	t.Run("legacy chart credentials under VMAGENT_ names are the same credential", func(t *testing.T) {
		legacyChartCreds(t)
		env, _ := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
		assertNoServerCredentialTemplates(t, env)
	})

	t.Run("injected URL with the VM credential: all-in-cluster fleets write to vmauth directly", func(t *testing.T) {
		legacyChartCreds(t)
		t.Setenv(envRemoteWriteURL, testVMAuthWrite)
		env, _ := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, testVMAuthWrite)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("injected URL alone emits no credential", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		env, args := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, env, "", "")
		assertNotAnywhere(t, env, args, "vm-password")
	})

	t.Run("tuning variables pass through unchanged", func(t *testing.T) {
		t.Setenv("VMAGENT_loggerLevel", "WARN")
		t.Setenv("VMAGENT_remoteWrite_maxDiskUsagePerURL", "52428800")
		env, _ := build(t, testVMAuth, client)
		assertEnv(t, env, "VMAGENT_loggerLevel", "WARN")
		assertEnv(t, env, "VMAGENT_remoteWrite_maxDiskUsagePerURL", "52428800")
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("an operator credential different from the URL's wins", func(t *testing.T) {
		t.Setenv(envRemoteWriteUsername, "other-user")
		t.Setenv(envRemoteWritePassword, "other-pass")
		env, args := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "other-user", "other-pass")
		assertNotAnywhere(t, env, args, "vm-password")
	})

	t.Run("the server agent writes to vmauth directly with the VM credential", func(t *testing.T) {
		env, args := build(t, testVMAuth, server)
		assertEnv(t, env, envRemoteWriteURL, testVMAuthWrite)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
		assertNoServerCredentialTemplates(t, env)
		assertNotAnywhere(t, nil, args, "basicAuth.")
		url, _ := envValue(env, envRemoteWriteURL)
		assert.NotContains(t, url, "@")
	})

	t.Run("the server agent follows an injected URL too", func(t *testing.T) {
		legacyChartCreds(t)
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		env, _ := build(t, testVMAuth, server)
		assertEnv(t, env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("a VM URL without credentials emits none and no PMM templates", func(t *testing.T) {
		// HARemoteWriteWarning reports this shape at startup; the output must still be well-formed.
		for _, d := range []vmAgentDeployment{client, server} {
			env, _ := build(t, testVMAuthNoCreds, d)
			assertCredentials(t, env, "", "")
			assertNoServerCredentialTemplates(t, env)
		}
	})

	t.Run("internal VM URL: clients get the standalone pair", func(t *testing.T) {
		env, _ := build(t, models.VMBaseURL, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, serverUsernameTmpl, serverPasswordTmpl)
	})

	t.Run("Kubernetes service-link noise passes through without affecting routing", func(t *testing.T) {
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_PORT", "8429")
		env, _ := build(t, testVMAuth, client)
		assertEnv(t, env, "VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("no HA agent receives a PMM Server credential template with an external VM URL", func(t *testing.T) {
		for _, vmURL := range []string{testVMAuth, testVMAuthNoCreds} {
			for _, d := range []vmAgentDeployment{client, server} {
				env, _ := build(t, vmURL, d)
				assertNoServerCredentialTemplates(t, env)
			}
		}
		legacyChartCreds(t)
		for _, d := range []vmAgentDeployment{client, server} {
			env, _ := build(t, testVMAuth, d)
			assertNoServerCredentialTemplates(t, env)
		}
	})
}
