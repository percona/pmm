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
	"github.com/percona/pmm/managed/utils/stringset"
	"github.com/percona/pmm/version"
)

// scrapeConfigLookup resolves the Services, Nodes and pmm-agents referenced by a set of Agents.
// They are fetched in three bulk queries up front: a node running a dozen services used to cost
// one Service, one Node and one pmm-agent lookup per exporter, re-reading the very same pmm-agent
// row and its Node for every exporter of that node.
type scrapeConfigLookup struct {
	l         *logrus.Entry
	q         *reform.Querier
	services  map[string]*models.Service
	nodes     map[string]*models.Node
	pmmAgents map[string]*models.Agent
	versions  map[string]*version.Parsed
}

func newScrapeConfigLookup(l *logrus.Entry, q *reform.Querier, agents []*models.Agent) (*scrapeConfigLookup, error) {
	serviceIDs := make(map[string]struct{}, len(agents))
	pmmAgentIDs := make(map[string]struct{}, len(agents))
	nodeIDs := make(map[string]struct{}, len(agents))
	addID := func(ids map[string]struct{}, id string) {
		if id != "" {
			ids[id] = struct{}{}
		}
	}

	for _, agent := range agents {
		addID(serviceIDs, pointer.GetString(agent.ServiceID))
		addID(pmmAgentIDs, pointer.GetString(agent.PMMAgentID))
		addID(nodeIDs, pointer.GetString(agent.NodeID))
		addID(nodeIDs, pointer.GetString(agent.RunsOnNodeID))
	}

	services, err := models.FindServicesByIDs(q, stringset.ToSlice(serviceIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to find services for scrape config: %w", err)
	}
	for _, service := range services {
		addID(nodeIDs, service.NodeID)
	}

	pmmAgents, err := models.FindAgentsByIDs(q, stringset.ToSlice(pmmAgentIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to find pmm-agents for scrape config: %w", err)
	}
	lookup := &scrapeConfigLookup{
		l:         l,
		q:         q,
		services:  services,
		nodes:     make(map[string]*models.Node, len(nodeIDs)),
		pmmAgents: make(map[string]*models.Agent, len(pmmAgents)),
		versions:  make(map[string]*version.Parsed, len(pmmAgents)),
	}
	for _, pmmAgent := range pmmAgents {
		lookup.pmmAgents[pmmAgent.AgentID] = pmmAgent
		// The Node a pmm-agent runs on is only known once the pmm-agent row is in hand.
		addID(nodeIDs, pointer.GetString(pmmAgent.RunsOnNodeID))
	}

	nodes, err := models.FindNodesByIDs(q, stringset.ToSlice(nodeIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to find nodes for scrape config: %w", err)
	}
	for _, node := range nodes {
		lookup.nodes[node.NodeID] = node
	}

	return lookup, nil
}

// service returns the Service with the given ID. Rows that were not prefetched - because they
// appeared after the bulk query - are read individually, so a missing row fails exactly as before.
func (s *scrapeConfigLookup) service(id string) (*models.Service, error) {
	if service, ok := s.services[id]; ok {
		return service, nil
	}

	service, err := models.FindServiceByID(s.q, id)
	if err != nil {
		return nil, err
	}
	s.services[id] = service

	return service, nil
}

// node returns the Node with the given ID.
func (s *scrapeConfigLookup) node(id string) (*models.Node, error) {
	if node, ok := s.nodes[id]; ok {
		return node, nil
	}

	node, err := models.FindNodeByID(s.q, id)
	if err != nil {
		return nil, err
	}
	s.nodes[id] = node

	return node, nil
}

// pmmAgent returns the pmm-agent with the given ID.
func (s *scrapeConfigLookup) pmmAgent(id string) (*models.Agent, error) {
	if agent, ok := s.pmmAgents[id]; ok {
		return agent, nil
	}

	agent, err := models.FindAgentByID(s.q, id)
	if err != nil {
		return nil, err
	}
	s.pmmAgents[id] = agent

	return agent, nil
}

// pmmAgentVersion returns the parsed version of the given pmm-agent, or nil if it cannot be parsed.
func (s *scrapeConfigLookup) pmmAgentVersion(agent *models.Agent) *version.Parsed {
	if parsed, ok := s.versions[agent.AgentID]; ok {
		return parsed
	}

	parsed, err := version.Parse(pointer.GetString(agent.Version))
	if err != nil {
		s.l.Warnf("couldn't parse pmm-agent version for pmm-agent %s: %s", agent.AgentID, err)
	}
	s.versions[agent.AgentID] = parsed

	return parsed
}

// AddScrapeConfigs - adds agents scrape configuration to given scrape config,
// pmm_agent_id and push_metrics used for filtering.
func AddScrapeConfigs(l *logrus.Entry, cfg *config.Config, q *reform.Querier, //nolint:gocognit,cyclop,maintidx
	globalResolutions *models.MetricsResolutions, pmmAgentID *string, pushMetrics bool, skipExternalAgents bool,
) error {
	agents, err := models.FindAgentsForScrapeConfig(q, pmmAgentID, pushMetrics)
	if err != nil {
		return fmt.Errorf("failed to find agent for scrape config: %w", err)
	}

	lookup, err := newScrapeConfigLookup(l, q, agents)
	if err != nil {
		return err
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
			paramsService, err = lookup.service(pointer.GetString(agent.ServiceID))
			if err != nil {
				return err
			}
		}

		// find Node for this Agent or Service
		var paramsNode *models.Node
		switch {
		case agent.NodeID != nil:
			paramsNode, err = lookup.node(pointer.GetString(agent.NodeID))
		case paramsService != nil:
			paramsNode, err = lookup.node(paramsService.NodeID)
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
			pmmAgent, err = lookup.pmmAgent(*agent.PMMAgentID)
			if err != nil {
				return fmt.Errorf("failed to find pmm-agent for scrape config: %w", err)
			}
			paramPMMAgentVersion = lookup.pmmAgentVersion(pmmAgent)
		}
		switch {
		case pushMetrics:
			paramsHost = models.LocalhostAddr
		case agent.PMMAgentID != nil:
			pmmAgentNode, err = lookup.node(pointer.GetString(pmmAgent.RunsOnNodeID))
			if err != nil {
				return fmt.Errorf("failed to find Node by pmm-agent for scrape config: %w", err)
			}
			paramsHost = pmmAgentNode.Address
		case agent.RunsOnNodeID != nil:
			externalExporterNode, err := lookup.node(pointer.GetString(agent.RunsOnNodeID))
			if err != nil {
				return fmt.Errorf("failed to find Node for scrape config: %w", err)
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
		case models.RTAMongoDBAgentType:
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
