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
	"strings"

	"github.com/sirupsen/logrus"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// OfflineRemoteWriteURL is where an offline pmm-agent's vmagent writes. The .invalid name never resolves,
// so every sample stays in vmagent's on-disk queue until `pmm-agent offline-push` sends it to PMM Server.
const OfflineRemoteWriteURL = "http://pmm-offline.invalid/victoriametrics/api/v1/write"

// ExporterProcess returns the exporter process PMM Server would send to pmm-agent; service is nil for node_exporter.
func ExporterProcess(node *models.Node, service *models.Service, exporter *models.Agent, v *version.Parsed) (*agentv1.SetStateRequest_AgentProcess, error) {
	switch exporter.AgentType {
	case models.NodeExporterType:
		return nodeExporterConfig(node, exporter, v)
	case models.MySQLdExporterType:
		return mysqldExporterConfig(node, node, service, exporter, redactSecrets, v)
	case models.MongoDBExporterType:
		return mongodbExporterConfig(node, service, exporter, redactSecrets, v)
	case models.PostgresExporterType:
		return postgresExporterConfig(node, node, service, exporter, redactSecrets, v)
	default:
		return nil, fmt.Errorf("unsupported agent type %q", exporter.AgentType)
	}
}

// OfflineVMAgentProcess returns a vmagent that scrapes scrapeCfg and queues every sample on disk in queueDir.
func OfflineVMAgentProcess(scrapeCfg, queueDir, maxDiskUsage string) *agentv1.SetStateRequest_AgentProcess {
	p := buildVMAgentProcess(logrus.WithField("component", "offline"), scrapeCfg, remoteWrite{url: OfflineRemoteWriteURL})
	for i, arg := range p.Args {
		if strings.HasPrefix(arg, "-remoteWrite.tmpDataPath=") {
			p.Args[i] = "-remoteWrite.tmpDataPath=" + queueDir // outside tmp_dir, which pmm-agent wipes on every stop
		}
	}
	if maxDiskUsage != "" {
		p.Args = append(p.Args, "-remoteWrite.maxDiskUsagePerURL="+maxDiskUsage) // a flag beats PMM's 1 GiB env default
	}
	return p
}
