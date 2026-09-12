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
)

// TestVMAgentConfigGolden pins the complete process configuration PMM Server hands to a vmagent,
// as literal strings, for every deployment shape and injection variant. The other tests describe
// relationships between inputs and outputs and may refer to the production constants; this one
// must not, so that a typo in a placeholder or URL cannot be masked by a test comparing against
// the same constant. The placeholders are rendered on the client from its pmm-agent.yaml.
func TestVMAgentConfigGolden(t *testing.T) {
	clearVMAgentEnv(t)

	const (
		internalVM = "http://127.0.0.1:9090/prometheus/"
		externalVM = "http://victoriametrics:8428"
		// Credentials in the fixtures are fixtures, not secrets.
		externalVMWithCreds = "http://vmuser:vmpass@victoriametrics:8428"
		haVMAuth            = "http://victoriametrics_pmm:vm-password@pmm-ha-vmauth.pmm.svc.cluster.local:8427/"
		haVMAuthNoCreds     = "http://pmm-ha-vmauth.pmm.svc.cluster.local:8427/"
	)

	wantArgs := []string{
		"-envflag.enable=true",
		"-envflag.prefix=VMAGENT_",
		"-httpListenAddr=127.0.0.1:{{.listen_port}}",
		"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
		"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
	}

	// The pre-PMM_HA_VM_* chart put the vmauth credential under the VMAGENT_ passthrough names.
	legacyChartCreds := map[string]string{
		"VMAGENT_remoteWrite_basicAuth_username": "victoriametrics_pmm",
		"VMAGENT_remoteWrite_basicAuth_password": "vm-password",
	}

	testCases := []struct {
		name       string
		vmURL      string
		deployment vmAgentDeployment
		injected   map[string]string
		wantEnv    []string
	}{
		{
			name:  "standalone, internal VM, client",
			vmURL: internalVM,
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password={{.server_password}}",
				"VMAGENT_remoteWrite_basicAuth_username={{.server_username}}",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write",
			},
		},
		{
			name:  "standalone, external VM without credentials, client or built-in agent",
			vmURL: externalVM,
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write",
			},
		},
		{
			name:  "standalone, external VM with credentials in PMM_VM_URL",
			vmURL: externalVMWithCreds,
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=vmpass",
				"VMAGENT_remoteWrite_basicAuth_username=vmuser",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write",
			},
		},
		{
			name:  "standalone, external VM, credentials injected (the documented shape)",
			vmURL: externalVM,
			injected: map[string]string{
				"VMAGENT_remoteWrite_basicAuth_username": "vmuser",
				"VMAGENT_remoteWrite_basicAuth_password": "vmpass",
			},
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=vmpass",
				"VMAGENT_remoteWrite_basicAuth_username=vmuser",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write",
			},
		},
		{
			name:       "HA, client, legacy chart secret keys alongside PMM_VM_URL",
			vmURL:      haVMAuth,
			deployment: vmAgentDeployment{haEnabled: true},
			injected:   legacyChartCreds,
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=vm-password",
				"VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write",
			},
		},
		{
			name:       "HA, client, credential from PMM_VM_URL only",
			vmURL:      haVMAuth,
			deployment: vmAgentDeployment{haEnabled: true},
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=vm-password",
				"VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write",
			},
		},
		{
			name:       "HA, built-in agent, legacy chart secret keys alongside PMM_VM_URL",
			vmURL:      haVMAuth,
			deployment: vmAgentDeployment{haEnabled: true, isServerAgent: true},
			injected:   legacyChartCreds,
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=vm-password",
				"VMAGENT_remoteWrite_basicAuth_username=victoriametrics_pmm",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=http://pmm-ha-vmauth.pmm.svc.cluster.local:8427/api/v1/write",
			},
		},
		{
			name:  "any, remote-write URL injected alone",
			vmURL: internalVM,
			injected: map[string]string{
				"VMAGENT_remoteWrite_url": "https://collector.example.com/api/v1/write",
			},
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			},
		},
		{
			name:  "any, remote-write URL and credentials injected",
			vmURL: internalVM,
			injected: map[string]string{
				"VMAGENT_remoteWrite_url":                "https://collector.example.com/api/v1/write",
				"VMAGENT_remoteWrite_basicAuth_username": "collector-user",
				"VMAGENT_remoteWrite_basicAuth_password": "collector-pass",
			},
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=collector-pass",
				"VMAGENT_remoteWrite_basicAuth_username=collector-user",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			},
		},
		{
			name:  "any, remote-write URL and only a username injected",
			vmURL: internalVM,
			injected: map[string]string{
				"VMAGENT_remoteWrite_url":                "https://collector.example.com/api/v1/write",
				"VMAGENT_remoteWrite_basicAuth_username": "collector-user",
			},
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_username=collector-user",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=https://collector.example.com/api/v1/write",
			},
		},
		{
			name:  "standalone, external VM with a password-only credential in PMM_VM_URL",
			vmURL: "http://:vmpass@victoriametrics:8428",
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_basicAuth_password=vmpass",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url=http://victoriametrics:8428/api/v1/write",
			},
		},
		{
			name:       "HA, client, PMM_VM_URL without credentials (HARemoteWriteWarning reports it)",
			vmURL:      haVMAuthNoCreds,
			deployment: vmAgentDeployment{haEnabled: true},
			wantEnv: []string{
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.injected {
				t.Setenv(k, v)
			}
			actual := mustVMAgentConfig(t, "scrape_configs: []", newVMParams(t, tc.vmURL), tc.deployment)
			assert.Equal(t, wantArgs, actual.Args)
			assert.Equal(t, tc.wantEnv, actual.Env)
			assert.Equal(t, map[string]string{"vmagentscrapecfg": "scrape_configs: []"}, actual.TextFiles)
		})
	}
}
