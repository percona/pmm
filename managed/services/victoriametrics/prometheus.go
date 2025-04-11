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
	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
	config "github.com/percona/promconfig"
)

// AddScrapeConfigs - adds agents scrape configuration to given scrape config,
// pmm_agent_id and push_metrics used for filtering.
func AddScrapeConfigs(l *logrus.Entry, cfg *config.Config, q *reform.Querier, //nolint:cyclop,maintidx
	globalResolutions *models.MetricsResolutions, pmmAgentID *string, pushMetrics bool,
) error {
	agents, err := models.FindAgentsForScrapeConfig(q, pmmAgentID, pushMetrics)
	if err != nil {
		return errors.WithStack(err)
	}

	var rdsParams []*scrapeConfigParams
	for _, agent := range agents {
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
		var paramPMMAgentVersion *version.Parsed
		var pmmAgent *models.Agent
		if agent.PMMAgentID != nil {
			// extract node address through pmm-agent
			pmmAgent, err = models.FindAgentByID(q, *agent.PMMAgentID)
			if err != nil {
				return errors.WithStack(err)
			}
			paramPMMAgentVersion, err = version.Parse(pointer.GetString(pmmAgent.Version))
			if err != nil {
				l.Warnf("couldn't parse pmm-agent version for pmm-agent %s: %q", pmmAgent.AgentID, err)
			}
		}
		switch {
		// special case for push metrics mode,
		// vmagent scrapes it from localhost.
		case pushMetrics:
			paramsHost = "127.0.0.1"
		case agent.PMMAgentID != nil:
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

		mr := *globalResolutions // copy global resolutions
		if agent.ExporterOptions.MetricsResolutions != nil {
			if agent.ExporterOptions.MetricsResolutions.MR != 0 {
				mr.MR = agent.ExporterOptions.MetricsResolutions.MR
			}
			if agent.ExporterOptions.MetricsResolutions.HR != 0 {
				mr.HR = agent.ExporterOptions.MetricsResolutions.HR
			}
			if agent.ExporterOptions.MetricsResolutions.LR != 0 {
				mr.LR = agent.ExporterOptions.MetricsResolutions.LR
			}
		}

		var scfgs []*config.ScrapeConfig
		switch agent.AgentType {
		case models.NodeExporterType:
			scfgs, err = scrapeConfigsForNodeExporter(&scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           nil,
				agent:             agent,
				metricsResolution: &mr,
			})

		case models.MySQLdExporterType:
			scfgs, err = scrapeConfigsForMySQLdExporter(&scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
			})

		case models.MongoDBExporterType:
			scfgs, err = scrapeConfigsForMongoDBExporter(&scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				pmmAgentVersion:   paramPMMAgentVersion,
				metricsResolution: &mr,
			})

		case models.PostgresExporterType:
			scfgs, err = scrapeConfigsForPostgresExporter(&scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				streamParse:       true,
				metricsResolution: &mr,
			})

		case models.ProxySQLExporterType:
			scfgs, err = scrapeConfigsForProxySQLExporter(&scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
			})

		case models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType:
			continue
		case models.QANMongoDBProfilerAgentType, models.QANMongoDBMongologAgentType:
			continue
		case models.QANPostgreSQLPgStatementsAgentType, models.QANPostgreSQLPgStatMonitorAgentType:
			continue

		case models.RDSExporterType:
			rdsParams = append(rdsParams, &scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
			})
			continue

		case models.ExternalExporterType:
			scfgs, err = scrapeConfigsForExternalExporter(&mr, &scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
			})

		case models.VMAgentType:
			scfgs, err = scrapeConfigsForVMAgent(&mr, &scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           nil,
				agent:             agent,
				metricsResolution: &mr,
			})

		case models.AzureDatabaseExporterType:
			scfgs, err = scrapeConfigsForAzureDatabase(&mr, &scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
			})
		case models.NomadClientType:
			scfgs, err = scrapeConfigsForNomadAgent(&mr, &scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
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

	scfgs := scrapeConfigsForRDSExporter(rdsParams)
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scfgs...)

	return nil
}

// AddInternalServicesToScrape adds internal services metrics to scrape targets.
func AddInternalServicesToScrape(cfg *config.Config, s models.MetricsResolutions) {
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs,
		scrapeConfigForGrafana(s.MR),
		scrapeConfigForPMMManaged(s.MR),
		scrapeConfigForQANAPI2(s.MR),
		scrapeConfigForClickhouse(s.MR))
}
