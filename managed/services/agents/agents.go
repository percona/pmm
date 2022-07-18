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
	"fmt"

	"github.com/AlekSi/pointer"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

type redactMode int

const (
	redactSecrets redactMode = iota
	exposeSecrets
)

var pmmAgentPathsBaseSupport = version.MustParse("2.22.99")

// redactWords returns words that should be redacted from given Agent logs/output.
func redactWords(agent *models.Agent) []string {
	var words []string
	if s := pointer.GetString(agent.Password); s != "" {
		words = append(words, s)
	}
	if s := pointer.GetString(agent.AgentPassword); s != "" {
		words = append(words, s)
	}
	if s := pointer.GetString(agent.AWSSecretKey); s != "" {
		words = append(words, s)
	}
	if agent.AzureOptions != nil {
		if s := agent.AzureOptions.ClientSecret; s != "" {
			words = append(words, s)
		}
	}
	return words
}

// pathsBase returns paths base and in case of unsupported PMM client old hardcoded value.
func pathsBase(agentVersion *version.Parsed, tdpLeft, tdpRight string) string {
	if agentVersion == nil || agentVersion.Less(pmmAgentPathsBaseSupport) {
		return "/usr/local/percona/pmm2"
	}

	return tdpLeft + " .paths_base " + tdpRight
}

// ensureAuthParams updates agent start parameters to contain prometheus webconfig.
func ensureAuthParams(exporter *models.Agent, params *agentpb.SetStateRequest_AgentProcess, agentVersion *version.Parsed, minAuthVersion *version.Parsed) error {
	if agentVersion.Less(minAuthVersion) {
		params.Env = append(params.Env, fmt.Sprintf("HTTP_AUTH=pmm:%s", exporter.GetAgentPassword()))
	} else {
		if params.TextFiles == nil {
			params.TextFiles = make(map[string]string)
		}

		wcf, err := exporter.BuildWebConfigFile()
		if err != nil {
			return err
		}
		params.TextFiles["webConfigPlaceholder"] = wcf
		// see https://github.com/prometheus/exporter-toolkit/tree/v0.1.0/https
		params.Args = append(params.Args, "--web.config="+params.TemplateLeftDelim+" .TextFiles.webConfigPlaceholder "+params.TemplateRightDelim)
	}

	return nil
}
