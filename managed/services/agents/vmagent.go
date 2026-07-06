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
	"net/url"
	"os"
	"sort"
	"strings"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/utils/envvars"
)

var (
	maxScrapeSizeEnv     = "PMM_PROMSCRAPE_MAX_SCRAPE_SIZE"
	maxScrapeSizeDefault = "64MiB"
)

// extractCredentialsFromURL extracts username and password from a URL string.
// Returns empty strings if no credentials are found or if there's an error parsing the URL.
func extractCredentialsFromURL(urlStr string) (string, string) {
	if urlStr == "" {
		return "", ""
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil || parsedURL.User == nil {
		return "", ""
	}

	username := parsedURL.User.Username()
	password := ""
	if pwd, ok := parsedURL.User.Password(); ok {
		password = pwd
	}

	return username, password
}

// vmAgentConfig returns desired configuration of vmagent process.
func vmAgentConfig(scrapeCfg string, params victoriaMetricsParams, haEnabled bool) *agentv1.SetStateRequest_AgentProcess {
	if params.ExternalVM() && !haEnabled {
		// Scenario: standalone deployment with externally configured VM.
		return vmAgentConfigExternalVM(scrapeCfg, params)
	}
	// Scenarios:
	// a) standalone deployment with internal VM
	// b) HA deployment (external VM by default)
	return vmAgentConfigServerProxy(scrapeCfg, haEnabled)
}

// vmAgentConfigServerProxy configures the vmagent so that writes go through the PMM server's VM proxy,
// authenticated by default with each client's own PMM server credentials.
func vmAgentConfigServerProxy(scrapeCfg string, dropInjectedAuth bool) *agentv1.SetStateRequest_AgentProcess {
	return buildVMAgentProcess(scrapeCfg, vmAgentSettings{
		remoteWriteURL:   "{{.server_url}}/victoriametrics/api/v1/write",
		dropInjectedAuth: dropInjectedAuth,
		auth:             &basicAuth{username: "{{.server_username}}", password: "{{.server_password}}"},
	})
}

// vmAgentConfigExternalVM prepares the vmagent configuration so that clients push directly to the external VictoriaMetrics,
// authenticated with the credentials embedded in its URL. Deployment-injected VMAGENT_ overrides are honored.
func vmAgentConfigExternalVM(scrapeCfg string, params victoriaMetricsParams) *agentv1.SetStateRequest_AgentProcess {
	vmURL := params.URL()

	var auth *basicAuth
	if vmUsername, vmPassword := extractCredentialsFromURL(vmURL); vmUsername != "" {
		auth = &basicAuth{username: vmUsername, password: vmPassword}
	}

	return buildVMAgentProcess(scrapeCfg, vmAgentSettings{
		remoteWriteURL: vmURL + "api/v1/write",
		auth:           auth,
	})
}

// basicAuth holds default remote-write basic-auth credentials.
type basicAuth struct {
	username string
	password string
}

// vmAgentSettings captures settings that differ between deployment scenarios.
type vmAgentSettings struct {
	remoteWriteURL   string     // default for VMAGENT_remoteWrite_url, can be overridden by an injected value
	dropInjectedAuth bool       // dropInjectedAuth discards injected VMAGENT_remoteWrite_basicAuth_*.
	auth             *basicAuth // basic-auth default, applied only if not injected
}

// buildVMAgentProcess assembles the vmagent process configuration common to all scenarios.
func buildVMAgentProcess(scrapeCfg string, settings vmAgentSettings) *agentv1.SetStateRequest_AgentProcess {
	interfaceToBind := envvars.GetInterfaceToBind()
	// Only keep the specified exceptions as command line arguments
	args := []string{
		"-envflag.enable=true",
		"-envflag.prefix=VMAGENT_",
		"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
		"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
		"-httpListenAddr=" + interfaceToBind + ":{{.listen_port}}",
	}

	maxScrapeSize := maxScrapeSizeDefault
	if space := os.Getenv(maxScrapeSizeEnv); space != "" {
		maxScrapeSize = space
	}

	// Move all other parameters to environment variables
	var envs []string

	// First, collect all VMAGENT_ environment variables from the system
	systemEnvs := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, envvars.EnvVMAgentPrefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				systemEnvs[parts[0]] = parts[1]
			}
		}
	}

	// An injected remote-write URL means the target is no longer the server proxy, so injected
	// credentials belong with it and must not be dropped.
	if settings.dropInjectedAuth {
		if _, urlInjected := systemEnvs["VMAGENT_remoteWrite_url"]; !urlInjected {
			delete(systemEnvs, "VMAGENT_remoteWrite_basicAuth_username")
			delete(systemEnvs, "VMAGENT_remoteWrite_basicAuth_password")
		}
	}

	// Helper function to add env var only if not already set by system
	addEnvIfNotSet := func(key, value string) {
		if _, exists := systemEnvs[key]; !exists {
			envs = append(envs, key+"="+value)
		}
	}

	// Add the parameters that were previously command line arguments (only if not overridden)
	addEnvIfNotSet("VMAGENT_remoteWrite_url", settings.remoteWriteURL)
	addEnvIfNotSet("VMAGENT_remoteWrite_tlsInsecureSkipVerify", "{{.server_insecure}}")
	addEnvIfNotSet("VMAGENT_promscrape_maxScrapeSize", maxScrapeSize)
	addEnvIfNotSet("VMAGENT_remoteWrite_maxDiskUsagePerURL", "1073741824") // 1GB disk queue size
	addEnvIfNotSet("VMAGENT_loggerLevel", "INFO")

	if settings.auth != nil {
		addEnvIfNotSet("VMAGENT_remoteWrite_basicAuth_username", settings.auth.username)
		if settings.auth.password != "" {
			addEnvIfNotSet("VMAGENT_remoteWrite_basicAuth_password", settings.auth.password)
		}
	}

	// Add all system VMAGENT_ environment variables
	for key, value := range systemEnvs {
		envs = append(envs, key+"="+value)
	}

	sort.Strings(envs)
	sort.Strings(args)
	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_VM_AGENT,
		TemplateLeftDelim:  "{{",
		TemplateRightDelim: "}}",
		Args:               args,
		Env:                envs,
		TextFiles: map[string]string{
			"vmagentscrapecfg": scrapeCfg,
		},
	}

	return res
}
