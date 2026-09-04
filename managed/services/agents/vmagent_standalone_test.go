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

func TestStandaloneRemoteWrite(t *testing.T) {
	t.Run("internal VM: clients write through the server with their own credentials", func(t *testing.T) {
		for _, vmURL := range []string{models.VMBaseURL, "http://localhost:9090/prometheus/"} {
			assert.Equal(t, serverProxyRemoteWrite(), standaloneRemoteWrite(newVMParams(t, vmURL)), vmURL)
		}
	})

	t.Run("external VM: write directly, no credentials", func(t *testing.T) {
		assert.Equal(t, remoteWrite{url: testExternalVMWrite, source: credentialNone},
			standaloneRemoteWrite(newVMParams(t, testExternalVM)))
	})

	t.Run("external VM: credentials come from PMM_VM_URL", func(t *testing.T) {
		assert.Equal(t, remoteWrite{url: testExternalVMWrite, username: "vmuser", password: "vmpass", source: credentialVMURL},
			standaloneRemoteWrite(newVMParams(t, testExternalVMAuth)))
	})
}

// TestVMAgentStandaloneInternalVM covers the default deployment: VictoriaMetrics inside PMM Server.
func TestVMAgentStandaloneInternalVM(t *testing.T) {
	build := func(t *testing.T) []string {
		t.Helper()
		return vmAgentConfig(testLogger(), "", newVMParams(t, models.VMBaseURL), vmAgentDeployment{}).Env
	}

	t.Run("clients write through the server with their own credentials", func(t *testing.T) {
		env := build(t)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, serverUsernameTmpl, serverPasswordTmpl)
	})

	t.Run("injected credentials win", func(t *testing.T) {
		// Only useful as a shared PMM credential for all client writes, but it is the operator's call.
		t.Setenv(envRemoteWriteUsername, "shared-user")
		t.Setenv(envRemoteWritePassword, "shared-pass")
		env := build(t)
		assertEnv(t, env, envRemoteWriteURL, serverProxyWriteURL)
		assertCredentials(t, env, "shared-user", "shared-pass")
		assertNoServerCredentialTemplates(t, env)
	})

	t.Run("injected URL alone withholds the server credentials", func(t *testing.T) {
		// {{.server_username}}/{{.server_password}} render to each client's own PMM credentials;
		// they are never sent to an endpoint the operator chose.
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		env := build(t)
		assertEnv(t, env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, env, "", "")
		assertNoServerCredentialTemplates(t, env)
	})

	t.Run("injected URL with credentials", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		t.Setenv(envRemoteWriteUsername, "collector-user")
		t.Setenv(envRemoteWritePassword, "collector-pass")
		env := build(t)
		assertEnv(t, env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, env, "collector-user", "collector-pass")
	})

	t.Run("injected URL with credential templates restores per-client credentials", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, "https://pmm.example.com/victoriametrics/api/v1/write")
		t.Setenv(envRemoteWriteUsername, serverUsernameTmpl)
		t.Setenv(envRemoteWritePassword, serverPasswordTmpl)
		env := build(t)
		assertEnv(t, env, envRemoteWriteURL, "https://pmm.example.com/victoriametrics/api/v1/write")
		assertCredentials(t, env, serverUsernameTmpl, serverPasswordTmpl)
	})
}

// TestVMAgentStandaloneExternalVM covers PMM_VM_URL pointing at an operator-run VictoriaMetrics.
func TestVMAgentStandaloneExternalVM(t *testing.T) {
	build := func(t *testing.T, vmURL string, d vmAgentDeployment) ([]string, []string) {
		t.Helper()
		actual := vmAgentConfig(testLogger(), "", newVMParams(t, vmURL), d)
		return actual.Env, actual.Args
	}

	t.Run("no credentials: write directly, emit none", func(t *testing.T) {
		env, args := build(t, testExternalVM, vmAgentDeployment{})
		assertEnv(t, env, envRemoteWriteURL, testExternalVMWrite)
		assertCredentials(t, env, "", "")
		assertNoServerCredentialTemplates(t, env)
		assertNotAnywhere(t, env, args, "basicAuth.")
	})

	t.Run("credentials in PMM_VM_URL move into the environment", func(t *testing.T) {
		env, args := build(t, testExternalVMAuth, vmAgentDeployment{})
		assertEnv(t, env, envRemoteWriteURL, testExternalVMWrite)
		assertCredentials(t, env, "vmuser", "vmpass")
		assertNotAnywhere(t, nil, args, "vmpass")
		url, _ := envValue(env, envRemoteWriteURL)
		assert.NotContains(t, url, "@")
	})

	t.Run("documented shape: bare PMM_VM_URL plus injected credentials", func(t *testing.T) {
		t.Setenv(envRemoteWriteUsername, "vmuser")
		t.Setenv(envRemoteWritePassword, "vmpass")
		env, _ := build(t, testExternalVM, vmAgentDeployment{})
		assertEnv(t, env, envRemoteWriteURL, testExternalVMWrite)
		assertCredentials(t, env, "vmuser", "vmpass")
	})

	t.Run("injected credentials win over the URL's", func(t *testing.T) {
		t.Setenv(envRemoteWriteUsername, "other-user")
		t.Setenv(envRemoteWritePassword, "other-pass")
		env, args := build(t, testExternalVMAuth, vmAgentDeployment{})
		assertCredentials(t, env, "other-user", "other-pass")
		assertNotAnywhere(t, env, args, "vmpass")
	})

	t.Run("injected URL alone withholds the URL's credentials", func(t *testing.T) {
		// The credentials in PMM_VM_URL describe the endpoint PMM was pointed at, not the injected one.
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		env, args := build(t, testExternalVMAuth, vmAgentDeployment{})
		assertEnv(t, env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, env, "", "")
		assertNotAnywhere(t, env, args, "vmpass")
	})

	t.Run("injected URL with credentials: the split read/write endpoints case", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, "http://vminsert:8480/insert/0/prometheus/api/v1/write")
		t.Setenv(envRemoteWriteUsername, "vmuser")
		t.Setenv(envRemoteWritePassword, "vmpass")
		env, _ := build(t, testExternalVMAuth, vmAgentDeployment{})
		assertEnv(t, env, envRemoteWriteURL, "http://vminsert:8480/insert/0/prometheus/api/v1/write")
		assertCredentials(t, env, "vmuser", "vmpass")
	})

	t.Run("percent-encoded credentials are decoded", func(t *testing.T) {
		env, _ := build(t, "http://us%40er:p%40ss@victoriametrics:8428", vmAgentDeployment{})
		assertCredentials(t, env, "us@er", "p@ss")
	})

	t.Run("the server's own agent gets the same configuration as a client", func(t *testing.T) {
		clientEnv, _ := build(t, testExternalVMAuth, vmAgentDeployment{})
		serverEnv, _ := build(t, testExternalVMAuth, vmAgentDeployment{isServerAgent: true})
		assert.Equal(t, clientEnv, serverEnv)
	})
}
