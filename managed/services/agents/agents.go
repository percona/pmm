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
	"context"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// AgentService handles generic communication with the Agent.
type AgentService struct {
	r *Registry
}

// NewAgentService returns new agent service.
func NewAgentService(r *Registry) *AgentService {
	return &AgentService{
		r: r,
	}
}

// Logs by Agent ID.
func (a *AgentService) Logs(_ context.Context, pmmAgentID, agentID string, limit uint32) ([]string, uint32, error) {
	agent, err := a.r.get(pmmAgentID)
	if err != nil {
		return nil, 0, err
	}

	resp, err := agent.channel.SendAndWaitResponse(&agentv1.AgentLogsRequest{
		AgentId: agentID,
		Limit:   limit,
	})
	if err != nil {
		return nil, 0, err
	}

	agentLogsResponse, ok := resp.(*agentv1.AgentLogsResponse)
	if !ok {
		return nil, 0, errors.New("wrong response from agent (not AgentLogsResponse model)")
	}

	return agentLogsResponse.GetLogs(), agentLogsResponse.GetAgentConfigLogLinesCount(), nil
}

// PBMSwitchPITR switches Point-in-Time Recovery feature for pbm on the pmm-agent.
func (a *AgentService) PBMSwitchPITR(pmmAgentID, dsn string, files map[string]string, tdp *models.DelimiterPair, enabled bool) error {
	agent, err := a.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	req := &agentv1.PBMSwitchPITRRequest{
		Dsn: dsn,
		TextFiles: &agentv1.TextFiles{
			Files:              files,
			TemplateLeftDelim:  tdp.Left,
			TemplateRightDelim: tdp.Right,
		},
		Enabled: enabled,
	}

	_, err = agent.channel.SendAndWaitResponse(req)
	return err
}

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
	if agent.AWSOptions != nil {
		if s := agent.AWSOptions.AWSSecretKey; s != "" {
			words = append(words, s)
		}
	}
	if agent.AzureOptions != nil {
		if s := agent.AzureOptions.ClientSecret; s != "" {
			words = append(words, s)
		}
	}
	if agent.MongoDBOptions != nil {
		if s := agent.MongoDBOptions.TLSCertificateKey; s != "" {
			words = append(words, s)
		}
		if s := agent.MongoDBOptions.TLSCertificateKeyFilePassword; s != "" {
			words = append(words, s)
		}
	}
	if agent.MySQLOptions != nil {
		if s := agent.MySQLOptions.TLSKey; s != "" {
			words = append(words, s)
		}
	}
	if agent.PostgreSQLOptions != nil {
		if s := agent.PostgreSQLOptions.SSLKey; s != "" {
			words = append(words, s)
		}
	}

	return words
}

// pathsBase returns paths base and in case of unsupported PMM client old hardcoded value.
func pathsBase(agentVersion *version.Parsed, tdpLeft, tdpRight string) string {
	if agentVersion == nil || agentVersion.Less(pmmAgentPathsBaseSupport) {
		return "/usr/local/percona/pmm"
	}

	return tdpLeft + " .paths_base " + tdpRight
}

// ensureAuthParams updates agent start parameters to contain prometheus webconfig.
func ensureAuthParams(exporter *models.Agent, params *agentv1.SetStateRequest_AgentProcess,
	agentVersion *version.Parsed, minAuthVersion *version.Parsed, useNewTLSConfig bool,
) error {
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
		if useNewTLSConfig {
			params.Args = append(params.Args, "--web.config.file="+params.TemplateLeftDelim+" .TextFiles.webConfigPlaceholder "+params.TemplateRightDelim)
		} else {
			params.Args = append(params.Args, "--web.config="+params.TemplateLeftDelim+" .TextFiles.webConfigPlaceholder "+params.TemplateRightDelim)
		}
	}

	return nil
}

// getExporterListenAddress returns the appropriate listen address to use for a given exporter.
func getExporterListenAddress(_ *models.Node, exporter *models.Agent) string {
	switch {
	case exporter.ExporterOptions.ExposeExporter:
		return "0.0.0.0"
	case exporter.ExporterOptions.PushMetrics:
		return "127.0.0.1"

	}

	return "0.0.0.0"
}
