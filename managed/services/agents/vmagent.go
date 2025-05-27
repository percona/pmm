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
	"fmt"
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
func extractCredentialsFromURL(urlStr string) (username, password string) {
	if urlStr == "" {
		return "", ""
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil || parsedURL.User == nil {
		return "", ""
	}

	username = parsedURL.User.Username()
	if pwd, ok := parsedURL.User.Password(); ok {
		password = pwd
	}

	return username, password
}

// vmAgentConfig returns desired configuration of vmagent process.
func vmAgentConfig(scrapeCfg string, params victoriaMetricsParams) *agentv1.SetStateRequest_AgentProcess {
	serverURL := "{{.server_url}}/victoriametrics/"
	var vmUsername, vmPassword string

	if params.ExternalVM() {
		serverURL = params.URL()

		// Extract username and password from external VM URL if present
		vmUsername, vmPassword = extractCredentialsFromURL(serverURL)
	}

	maxScrapeSize := maxScrapeSizeDefault
	if space := os.Getenv(maxScrapeSizeEnv); space != "" {
		maxScrapeSize = space
	}

	interfaceToBind := envvars.GetInterfaceToBind()

	// Only keep the specified exceptions as command line arguments
	args := []string{
		"-envflag.enable=true",
		"-envflag.prefix=VMAGENT_",
		"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
		"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
		"-httpListenAddr=" + interfaceToBind + ":{{.listen_port}}",
	}

	sort.Strings(args)

	// Move all other parameters to environment variables
	var envs []string

	// First, collect all VMAGENT_ environment variables from the system
	systemEnvs := make(map[string]string)
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, envvars.ENVvmAgentPrefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				systemEnvs[parts[0]] = parts[1]
			}
		}
	}

	// Helper function to add env var only if not already set by system
	addEnvIfNotSet := func(key, value string) {
		if _, exists := systemEnvs[key]; !exists {
			envs = append(envs, key+"="+value)
		}
	}

	// Add the parameters that were previously command line arguments (only if not overridden)
	addEnvIfNotSet("VMAGENT_remoteWrite_url", fmt.Sprintf("%sapi/v1/write", serverURL))
	addEnvIfNotSet("VMAGENT_remoteWrite_tlsInsecureSkipVerify", "{{.server_insecure}}")
	addEnvIfNotSet("VMAGENT_promscrape_maxScrapeSize", maxScrapeSize)
	addEnvIfNotSet("VMAGENT_remoteWrite_maxDiskUsagePerURL", "1073741824") // 1GB disk queue size
	addEnvIfNotSet("VMAGENT_loggerLevel", "INFO")

	// Set authentication based on VM type
	if params.ExternalVM() && vmUsername != "" {
		// Use credentials from external VM URL
		addEnvIfNotSet("VMAGENT_remoteWrite_basicAuth_username", vmUsername)
		if vmPassword != "" {
			addEnvIfNotSet("VMAGENT_remoteWrite_basicAuth_password", vmPassword)
		}
	} else if !params.ExternalVM() {
		// Use PMM server credentials for internal VM
		addEnvIfNotSet("VMAGENT_remoteWrite_basicAuth_username", "{{.server_username}}")
		addEnvIfNotSet("VMAGENT_remoteWrite_basicAuth_password", "{{.server_password}}")
	}

	// Add all system VMAGENT_ environment variables
	for key, value := range systemEnvs {
		envs = append(envs, key+"="+value)
	}

	sort.Strings(envs)

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
