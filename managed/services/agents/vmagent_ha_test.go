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

	"github.com/percona/pmm/managed/models"
)

// chartShape injects what the pmm-ha chart puts into every PMM Server pod besides PMM_VM_URL:
// the vmauth credentials under the VMAGENT_ passthrough names.
func chartShape(t *testing.T) {
	t.Helper()
	t.Setenv(envRemoteWriteUsername, "victoriametrics_pmm")
	t.Setenv(envRemoteWritePassword, "vm-password")
}

func TestHARemoteWrite(t *testing.T) {
	vmCreds := remoteWrite{username: "victoriametrics_pmm", password: "vm-password", source: credentialVMURL}

	t.Run("clients write to their PMM Server address with the VM credential", func(t *testing.T) {
		want := vmCreds
		want.url = serverProxyWriteURL
		assert.Equal(t, want, haRemoteWrite(newVMParams(t, testVMAuth), false))
	})

	t.Run("the server agent writes to vmauth directly", func(t *testing.T) {
		want := vmCreds
		want.url = testVMAuthWrite
		assert.Equal(t, want, haRemoteWrite(newVMParams(t, testVMAuth), true))
	})

	t.Run("an internal VM URL falls back to the standalone pair", func(t *testing.T) {
		params := newVMParams(t, models.VMBaseURL)
		assert.Equal(t, serverProxyRemoteWrite(), haRemoteWrite(params, false))
		assert.Equal(t, serverProxyRemoteWrite(), haRemoteWrite(params, true))
	})
}

func TestVMAgentHA(t *testing.T) {
	client := vmAgentDeployment{haEnabled: true}
	server := vmAgentDeployment{haEnabled: true, isServerAgent: true}
	build := func(t *testing.T, vmURL string, d vmAgentDeployment) ([]string, []string) {
		t.Helper()
		actual := vmAgentConfig(testLogger(), "", newVMParams(t, vmURL), d)
		return actual.Env, actual.Args
	}

	t.Run("chart shape: clients write via their PMM Server address with the VM credential", func(t *testing.T) {
		chartShape(t)
		env, args := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
		assertNoServerCredentialTemplates(t, env)
		assertNotAnywhere(t, nil, args, "basicAuth.")
	})

	t.Run("the VM credential comes from PMM_VM_URL even without the chart's VMAGENT_ variables", func(t *testing.T) {
		// The chart may stop exposing the credential under VMAGENT_ names; PMM_VM_URL is the source.
		env, _ := build(t, testVMAuth, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("injected URL with the chart credential: all-in-cluster fleets write to vmauth directly", func(t *testing.T) {
		chartShape(t)
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
		chartShape(t)
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
		chartShape(t)
		env, args := build(t, testVMAuth, server)
		assertEnv(t, env, envRemoteWriteURL, testVMAuthWrite)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
		assertNoServerCredentialTemplates(t, env)
		assertNotAnywhere(t, nil, args, "basicAuth.")
		url, _ := envValue(env, envRemoteWriteURL)
		assert.NotContains(t, url, "@")
	})

	t.Run("the server agent follows an injected URL too", func(t *testing.T) {
		chartShape(t)
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		env, _ := build(t, testVMAuth, server)
		assertEnv(t, env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("internal VM URL: clients get the standalone pair", func(t *testing.T) {
		env, _ := build(t, models.VMBaseURL, client)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, serverUsernameTmpl, serverPasswordTmpl)
	})

	t.Run("Kubernetes service-link noise passes through without affecting routing", func(t *testing.T) {
		chartShape(t)
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_PORT", "8429")
		env, _ := build(t, testVMAuth, client)
		assertEnv(t, env, "VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "victoriametrics_pmm", "vm-password")
	})

	t.Run("no HA agent ever receives a PMM Server credential template", func(t *testing.T) {
		for _, d := range []vmAgentDeployment{client, server} {
			env, _ := build(t, testVMAuth, d)
			assertNoServerCredentialTemplates(t, env)
		}
		chartShape(t)
		for _, d := range []vmAgentDeployment{client, server} {
			env, _ := build(t, testVMAuth, d)
			assertNoServerCredentialTemplates(t, env)
		}
	})
}
