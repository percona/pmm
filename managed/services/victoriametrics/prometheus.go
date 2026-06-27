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

	"github.com/AlekSi/pointer"
	config "github.com/percona/promconfig"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/version"
)

// AddScrapeConfigs - adds agents scrape configuration to given scrape config,
// pmm_agent_id and push_metrics used for filtering.
func AddScrapeConfigs(l *logrus.Entry, cfg *config.Config, q *reform.Querier, //nolint:gocognit,cyclop,maintidx
	globalResolutions *models.MetricsResolutions, pmmAgentID *string, pushMetrics bool, skipExternalAgents bool,
) error {
	agents, err := models.FindAgentsForScrapeConfig(q, pmmAgentID, pushMetrics)
	if err != nil {
		return fmt.Errorf("failed to find agent for scrape config: %w", err)
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
		var pmmAgentNode *models.Node
		if agent.PMMAgentID != nil {
			// find a related pmm-agent to get the node address (runs_on_node_id)
			pmmAgent, err = models.FindAgentByID(q, *agent.PMMAgentID)
			if err != nil {
				return fmt.Errorf("failed to find pmm-agent for scrape config: %w", err)
			}
			paramPMMAgentVersion, err = version.Parse(pointer.GetString(pmmAgent.Version))
			if err != nil {
				l.Warnf("couldn't parse pmm-agent version for pmm-agent %s: %q", pmmAgent.AgentID, err)
			}
		}
		switch {
		case pushMetrics:
			paramsHost = models.LocalhostAddr
		case agent.PMMAgentID != nil:
			pmmAgentNode = &models.Node{NodeID: pointer.GetString(pmmAgent.RunsOnNodeID)}
			err = q.Reload(pmmAgentNode)
			if err != nil {
				return fmt.Errorf("failed to reload Node by pmm-agent for scrape config: %w", err)
			}
			paramsHost = pmmAgentNode.Address
		case agent.RunsOnNodeID != nil:
			externalExporterNode := &models.Node{NodeID: pointer.GetString(agent.RunsOnNodeID)}
			err = q.Reload(externalExporterNode)
			if err != nil {
				return fmt.Errorf("failed to reload Node for scrape config: %w", err)
			}
			paramsHost = externalExporterNode.Address
		default:
			l.Warnf("It's not possible to get host, skipping scrape config for %s.", agent)

			continue
		}

		// In HA mode, skip generating scrape config for agents that run on other PMM Server nodes.
		// These agents listen on 127.0.0.1 and are unreachable from this PMM instance.
		// We check the node where the pmm-agent runs (not the service node).
		if !pushMetrics && pmmAgentNode != nil && pmmAgentNode.NodeID != models.PMMServerNodeID && pmmAgentNode.IsPMMServerNode {
			l.Debugf("Skip the scrape config for %s agent %s running on remote PMM Server node %s in HA mode",
				agent.AgentType, agent.AgentID, pmmAgentNode.NodeName)
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

		case models.ValkeyExporterType:
			scfgs, err = scrapeConfigForValkeyExporter(&scrapeConfigParams{
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
		case models.RTAMongoDBAgentType, models.RTAMySQLAgentType:
			continue
		case models.RDSExporterType:
			if skipExternalAgents && pointer.GetString(agent.RunsOnNodeID) == models.PMMServerNodeID {
				l.Debugf("Skip the scrape config for RDSExporter %s running on PMM Server in HA non-leader mode", agent.AgentID)
				continue
			}
			rdsParams = append(rdsParams, &scrapeConfigParams{
				host:              paramsHost,
				node:              paramsNode,
				service:           paramsService,
				agent:             agent,
				metricsResolution: &mr,
			})
			continue

		case models.ExternalExporterType:
			if skipExternalAgents && pointer.GetString(agent.RunsOnNodeID) == models.PMMServerNodeID {
				l.Debugf("Skip the scrape config for ExternalExporter %s running on PMM Server in HA non-leader mode", agent.AgentID)
				continue
			}
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
		case models.NomadAgentType:
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
func addInternalServicesToScrape(s models.MetricsResolutions, svc *Service, pmmServerNodeName string) []*config.ScrapeConfig {
	cfg := make([]*config.ScrapeConfig, 0, 4) //nolint:mnd
	cfg = append(
		cfg,
		scrapeConfigForGrafana(s.MR, pmmServerNodeName),
		scrapeConfigForPMMManaged(s.MR, pmmServerNodeName),
		scrapeConfigForQANAPI2(s.MR, pmmServerNodeName),
	)

	if svc.chParams.ExternalClickHouse() {
		svc.l.Warnf("Skip internal ClickHouse scrape config, ClickHouse is configured to run externally.")
		return cfg
	}

	return append(cfg, scrapeConfigForClickhouse(s.MR, pmmServerNodeName))
}
