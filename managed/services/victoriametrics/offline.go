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

package victoriametrics

import (
	"fmt"
	"time"

	config "github.com/percona/promconfig"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// OfflineTarget is an exporter of an offline pmm-agent and the Service it monitors (nil for node_exporter).
type OfflineTarget struct {
	Service *models.Service
	Agent   *models.Agent
}

// OfflineScrapeConfig returns the vmagent scrape config PMM Server would build for targets on node,
// at PMM's default metrics resolutions.
func OfflineScrapeConfig(node *models.Node, targets []OfflineTarget, v *version.Parsed) ([]byte, error) {
	resolutions := models.MetricsResolutions{HR: 5 * time.Second, MR: 10 * time.Second, LR: time.Minute}
	var cfg config.Config
	for _, t := range targets {
		params := &scrapeConfigParams{host: models.LocalhostAddr, node: node, service: t.Service, agent: t.Agent, pmmAgentVersion: v, metricsResolution: &resolutions}
		var scfgs []*config.ScrapeConfig
		var err error
		switch t.Agent.AgentType {
		case models.NodeExporterType:
			scfgs, err = scrapeConfigsForNodeExporter(params)
		case models.MySQLdExporterType:
			scfgs, err = scrapeConfigsForMySQLdExporter(params)
		case models.MongoDBExporterType:
			scfgs, err = scrapeConfigsForMongoDBExporter(params)
		case models.PostgresExporterType:
			params.streamParse = true // as AddScrapeConfigs does
			scfgs, err = scrapeConfigsForPostgresExporter(params)
		default:
			err = fmt.Errorf("unsupported agent type %q", t.Agent.AgentType)
		}
		if err != nil {
			return nil, err
		}
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
	}
	return yaml.Marshal(cfg)
}
