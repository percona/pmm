// Copyright (C) 2017 Percona LLC
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
	"sort"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

// vmAgentConfig returns desired configuration of vmagent process.
func vmAgentConfig(scrapeCfg string) *agentpb.SetStateRequest_AgentProcess {
	args := []string{
		"-remoteWrite.url={{.server_url}}/victoriametrics/api/v1/write",
		"-remoteWrite.tlsInsecureSkipVerify={{.server_insecure}}",
		"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
		"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
		// 1GB disk queue size.
		"-remoteWrite.maxDiskUsagePerURL=1073741824",
		"-loggerLevel=INFO",
		"-httpListenAddr=127.0.0.1:{{.listen_port}}",
		// needed for login/password at client side.
		"-envflag.enable=true",
		"-promscrape.noStaleMarkers=true",
		"-minStalenessInterval=0s",
	}

	sort.Strings(args)

	envs := []string{
		"remoteWrite_basicAuth_username={{.server_username}}",
		"remoteWrite_basicAuth_password={{.server_password}}",
	}
	sort.Strings(envs)

	res := &agentpb.SetStateRequest_AgentProcess{
		Type:               inventorypb.AgentType_VM_AGENT,
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
