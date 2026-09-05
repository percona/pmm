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
	"sort"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// valkeyExporterConfig returns the desired configuration of the valkey_exporter process.
func valkeyExporterConfig(node *models.Node, service *models.Service, exporter *models.Agent, redactMode redactMode,
	pmmAgentVersion *version.Parsed,
) *agentv1.SetStateRequest_AgentProcess {
	listenAddress := getExporterListenAddress(node, exporter)
	tdp := exporter.TemplateDelimiters(service)
	args := []string{
		"--web.listen-address=" + listenAddress + ":" + tdp.Left + " .listen_port " + tdp.Right,
		"--include-config-metrics",
		"--include-system-metrics",
	}

	if exporter.ExporterOptions.MetricsPath != "" {
		args = append(args, "--web.telemetry-path="+exporter.ExporterOptions.MetricsPath)
	}

	textFiles := exporter.Files()
	if exporter.TLS {
		if exporter.TLSSkipVerify {
			args = append(args, "--skip-tls-verification")
		}

		// The flag names come from oliver006/redis_exporter, shipped as valkey_exporter;
		// all four have been stable since v1.72.1, the build the first Valkey release used.
		tlsFileFlags := []struct{ file, flag string }{
			{models.TLSCaFileName, "--tls-ca-cert-file"},
			{models.TLSCertFileName, "--tls-client-cert-file"},
			{models.TLSKeyFileName, "--tls-client-key-file"},
		}
		for _, f := range tlsFileFlags {
			if _, ok := textFiles[f.file]; ok {
				args = append(args, f.flag+"="+tdp.Left+" .TextFiles."+f.file+" "+tdp.Right)
			}
		}
	}

	dsnParams := models.DSNParams{}
	connectionTimeout := exporter.EffectiveDialTimeout()

	args = append(args, "--redis.addr="+exporter.DSN(service, dsnParams, tdp, pmmAgentVersion))
	args = append(args, "--connection-timeout="+connectionTimeout.String())
	// valkey_exporter parses flags with the stdlib flag package, which rejects --log.level
	// and has no fatal level.
	args = withLogLevelFlag(args, "--log-level", exporter.LogLevel, pmmAgentVersion, false)
	sort.Strings(args)

	res := &agentv1.SetStateRequest_AgentProcess{
		Type:               inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER,
		TemplateLeftDelim:  tdp.Left,
		TemplateRightDelim: tdp.Right,
		Args:               args,
		TextFiles:          textFiles,
	}
	if redactMode != exposeSecrets {
		res.RedactWords = redactWords(exporter)
	}
	return res
}
