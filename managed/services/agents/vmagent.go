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

// vmAgentConfig returns desired configuration of vmagent process.
func vmAgentConfig(scrapeCfg string, params victoriaMetricsParams) *agentv1.SetStateRequest_AgentProcess {
	serverURL := "{{.server_url}}/victoriametrics/"
	if params.ExternalVM() {
		serverURL = params.URL()
	}
	maxScrapeSize := maxScrapeSizeDefault
	if space := os.Getenv(maxScrapeSizeEnv); space != "" {
		maxScrapeSize = space
	}

	interfaceToBind := envvars.GetInterfaceToBind()

	args := []string{
		fmt.Sprintf("-remoteWrite.url=%sapi/v1/write", serverURL),
		"-remoteWrite.tlsInsecureSkipVerify={{.server_insecure}}",
		"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
		"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
		"-promscrape.maxScrapeSize=" + maxScrapeSize,
		// 1GB disk queue size.
		"-remoteWrite.maxDiskUsagePerURL=1073741824",
		"-loggerLevel=INFO",
		"-httpListenAddr=" + interfaceToBind + ":{{.listen_port}}",
		// needed for login/password at client side.
		"-envflag.enable=true",
		"-envflag.prefix=VMAGENT_",
	}
	args = append(args, params.VMAgentArgs()...)

	sort.Strings(args)

	var envs []string
	if !params.ExternalVM() {
		envs = []string{
			"VMAGENT_remoteWrite_basicAuth_username={{.server_username}}",
			"VMAGENT_remoteWrite_basicAuth_password={{.server_password}}",
		}
	}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, envvars.ENVvmAgentPrefix) {
			envs = append(envs, env)
		}
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
