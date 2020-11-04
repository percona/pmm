// pmm-managed
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

// Package prometheus contains business logic of working with Prometheus.
package prometheus

import (
	"github.com/AlekSi/pointer"
	config "github.com/percona/promconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

func AddScrapeConfigs(l *logrus.Entry, cfg *config.Config, q *reform.Querier, s *models.MetricsResolutions) error {
	agents, err := q.SelectAllFrom(models.AgentTable, "WHERE NOT disabled AND listen_port IS NOT NULL ORDER BY agent_type, agent_id")
	if err != nil {
		return errors.WithStack(err)
	}

	var rdsParams []*scrapeConfigParams
	for _, str := range agents {
		agent := str.(*models.Agent)

		if agent.AgentType == models.PMMAgentType {
			// TODO https://jira.percona.com/browse/PMM-4087
			continue
		}

		// sanity check
		if (agent.NodeID != nil) && (agent.ServiceID != nil) {
			l.Panicf("Both agent.NodeID and agent.ServiceID are present: %s", agent)
		}

		// find Service for this Agent
		var paramsService *models.Service
		if agent.ServiceID != nil {
			paramsService, err = models.FindServiceByID(q, pointer.GetString(agent.ServiceID))
			if err != nil {
				return err
			}
		}

		// find Node for this Agent or Service
		var paramsNode *models.Node
		switch {
		case agent.NodeID != nil:
			paramsNode, err = models.FindNodeByID(q, pointer.GetString(agent.NodeID))
		case paramsService != nil:
			paramsNode, err = models.FindNodeByID(q, paramsService.NodeID)
		}
		if err != nil {
			return err
		}

		// find Node address where the agent runs
		var paramsHost string
		switch {
		case agent.PMMAgentID != nil:
			// extract node address through pmm-agent
			pmmAgent, err := models.FindAgentByID(q, *agent.PMMAgentID)
			if err != nil {
				return errors.WithStack(err)
			}
			pmmAgentNode := &models.Node{NodeID: pointer.GetString(pmmAgent.RunsOnNodeID)}
			if err = q.Reload(pmmAgentNode); err != nil {
				return errors.WithStack(err)
			}
			paramsHost = pmmAgentNode.Address
		case agent.RunsOnNodeID != nil:
			externalExporterNode := &models.Node{NodeID: pointer.GetString(agent.RunsOnNodeID)}
			if err = q.Reload(externalExporterNode); err != nil {
				return errors.WithStack(err)
			}
			paramsHost = externalExporterNode.Address
		default:
			l.Warnf("It's not possible to get host, skipping scrape config for %s.", agent)

			continue
		}

		var scfgs []*config.ScrapeConfig
		switch agent.AgentType {
		case models.NodeExporterType:
			scfgs, err = scrapeConfigsForNodeExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: nil,
				agent:   agent,
			})

		case models.MySQLdExporterType:
			scfgs, err = scrapeConfigsForMySQLdExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.MongoDBExporterType:
			scfgs, err = scrapeConfigsForMongoDBExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.PostgresExporterType:
			scfgs, err = scrapeConfigsForPostgresExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.ProxySQLExporterType:
			scfgs, err = scrapeConfigsForProxySQLExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		case models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType:
			continue
		case models.QANMongoDBProfilerAgentType:
			continue
		case models.QANPostgreSQLPgStatementsAgentType, models.QANPostgreSQLPgStatMonitorAgentType:
			continue

		case models.RDSExporterType:
			rdsParams = append(rdsParams, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})
			continue

		case models.ExternalExporterType:
			scfgs, err = scrapeConfigsForExternalExporter(s, &scrapeConfigParams{
				host:    paramsHost,
				node:    paramsNode,
				service: paramsService,
				agent:   agent,
			})

		default:
			l.Warnf("Skipping scrape config for %s.", agent)
			continue
		}

		if err != nil {
			l.Warnf("Failed to add %s %q, skipping: %s.", agent.AgentType, agent.AgentID, err)
		}
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)
	}

	scfgs := scrapeConfigsForRDSExporter(s, rdsParams)
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)

	return nil
}

// AddInternalServicesToScrape adds internal services metrics to scrape targets.
func AddInternalServicesToScrape(cfg *config.Config, s models.MetricsResolutions, dbaas bool) {
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs,
		scrapeConfigForAlertmanager(s.MR),
		scrapeConfigForGrafana(s.MR),
		scrapeConfigForPMMManaged(s.MR),
		scrapeConfigForQANAPI2(s.MR),
	)
	// TODO Refactor to remove boolean positional parameter when Prometheus or PERCONA_TEST_DBAAS is removed
	if dbaas {
		cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrapeConfigForDBaaSController(s.MR))
	}
}
