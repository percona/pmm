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
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/prototext"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/stringset"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const (
	// Constants for delayed batch updates.
	updateBatchDelay                = time.Second
	stateChangeTimeout              = 5 * time.Second
	loggerComponentNameStateUpdater = "state-updater"
)

// StateUpdater handles updating status of agents.
type StateUpdater struct {
	db       *reform.DB
	r        *Registry
	vmdb     prometheusService
	vmParams victoriaMetricsParams
	nomad    nomad
}

// NewStateUpdater creates new agent state updater.
func NewStateUpdater(db *reform.DB, r *Registry, vmdb prometheusService, vmParams victoriaMetricsParams, nomad nomad) *StateUpdater {
	return &StateUpdater{
		db:       db,
		r:        r,
		vmdb:     vmdb,
		vmParams: vmParams,
		nomad:    nomad,
	}
}

// RequestStateUpdate requests state update on pmm-agent with given ID. It sets
// the status to done if the agent is not connected.
func (u *StateUpdater) RequestStateUpdate(ctx context.Context, pmmAgentID string) {
	l := logger.Get(ctx).WithField("component", loggerComponentNameStateUpdater)

	agent, err := u.r.get(pmmAgentID)
	if err != nil {
		l.Infof("RequestStateUpdate: %s.", err)
		return
	}

	select {
	case agent.stateChangeChan <- struct{}{}:
	default:
	}
}

// UpdateAgentsState sends SetStateRequest to all pmm-agents with push metrics agents.
func (u *StateUpdater) UpdateAgentsState(ctx context.Context) error {
	pmmAgents, err := models.FindAllPMMAgentsIDs(u.db.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("cannot find pmmAgentsIDs for AgentsState update: %w", err)
	}
	var wg sync.WaitGroup
	limiter := make(chan struct{}, 10) //nolint:mnd
	for _, pmmAgentID := range pmmAgents {
		wg.Add(1)
		limiter <- struct{}{}
		go func(pmmAgentID string) {
			defer wg.Done()
			u.RequestStateUpdate(ctx, pmmAgentID)
			<-limiter
		}(pmmAgentID)
	}
	wg.Wait()
	return nil
}

// runStateChangeHandler runs pmm-agent state update loop for given pmm-agent until ctx is canceled or agent is kicked.
func (u *StateUpdater) runStateChangeHandler(ctx context.Context, agent *pmmAgentInfo) {
	l := logger.Get(ctx).
		WithField("component", loggerComponentNameStateUpdater).
		WithField("agent_id", agent.id)

	l.Info("Starting runStateChangeHandler ...")
	defer l.Info("Done runStateChangeHandler.")

	// stateChangeChan, state update loop, and RequestStateUpdate method ensure that state
	// is reloaded when requested, but several requests are batched together to avoid too often reloads.
	// That allows the caller to just call RequestStateUpdate when it seems fit.
	if cap(agent.stateChangeChan) != 1 {
		panic("stateChangeChan should have capacity 1")
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-agent.kickChan:
			return

		case <-agent.stateChangeChan:
			// batch several update requests together by delaying the first one
			sleepCtx, sleepCancel := context.WithTimeout(ctx, updateBatchDelay)
			<-sleepCtx.Done()
			sleepCancel()

			if ctx.Err() != nil {
				return
			}

			nCtx, cancel := context.WithTimeout(ctx, stateChangeTimeout)
			err := u.sendSetStateRequest(nCtx, agent)
			if err != nil {
				// Now that the channel honours nCtx, a request loop busy for longer than
				// stateChangeTimeout lands here, so say what timed out.
				l.Errorf("Failed to send SetState request: %s", err)
				u.RequestStateUpdate(ctx, agent.id)
			}
			cancel()
		}
	}
}

// sendSetStateRequest sends SetStateRequest to given pmm-agent.
func (u *StateUpdater) sendSetStateRequest(ctx context.Context, agent *pmmAgentInfo) error { //nolint:gocognit,cyclop,maintidx
	l := logger.Get(ctx).WithField("component", loggerComponentNameStateUpdater)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			l.Warnf("sendSetStateRequest took %s.", dur)
		}
	}()
	// Bind the queries to the request context, so that waiting for a free connection
	// in the pool is bounded by stateChangeTimeout instead of blocking indefinitely.
	q := u.db.WithContext(ctx)

	pmmAgent, err := models.FindAgentByID(q, agent.id)
	if err != nil {
		return fmt.Errorf("failed to get PMM Agent: %w", err)
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return fmt.Errorf("failed to parse PMM agent version %q: %w", *pmmAgent.Version, err)
	}

	settings, err := models.GetSettings(q)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	filters := models.AgentFilters{
		PMMAgentID:  agent.id,
		IgnoreNomad: !settings.IsNomadEnabled(),
		// fetch enabled only
		Disabled: new(false),
	}
	agents, err := models.FindAgents(q, filters)
	if err != nil {
		return fmt.Errorf("failed to collect agents: %w", err)
	}

	// Resolve the Services and Nodes the rows reference in two queries. The loop below read
	// them one row at a time, and re-read the Node the pmm-agent runs on for every row - on a
	// host with many services that is hundreds of sequential round trips, which now have to
	// fit into stateChangeTimeout.
	serviceIDs := make(map[string]struct{}, len(agents))
	nodeIDs := make(map[string]struct{}, len(agents))
	addID := func(ids map[string]struct{}, id string) {
		if id != "" {
			ids[id] = struct{}{}
		}
	}

	addID(nodeIDs, pointer.GetString(pmmAgent.RunsOnNodeID))
	for _, row := range agents {
		addID(serviceIDs, pointer.GetString(row.ServiceID))
		addID(nodeIDs, pointer.GetString(row.NodeID))
	}

	services, err := models.FindServicesByIDs(q, stringset.ToSlice(serviceIDs))
	if err != nil {
		return fmt.Errorf("failed to collect services: %w", err)
	}
	nodeRows, err := models.FindNodesByIDs(q, stringset.ToSlice(nodeIDs))
	if err != nil {
		return fmt.Errorf("failed to collect nodes: %w", err)
	}
	nodes := make(map[string]*models.Node, len(nodeRows))
	for _, node := range nodeRows {
		nodes[node.NodeID] = node
	}

	// Rows that appear after the bulk queries are read individually, so a missing row
	// still fails the way it did before.
	findService := func(id string) (*models.Service, error) {
		if service, ok := services[id]; ok {
			return service, nil
		}

		return models.FindServiceByID(q, id)
	}
	findNode := func(id string) (*models.Node, error) {
		if node, ok := nodes[id]; ok {
			return node, nil
		}

		return models.FindNodeByID(q, id)
	}

	pmmAgentNode, err := findNode(pointer.GetString(pmmAgent.RunsOnNodeID))
	if err != nil {
		return fmt.Errorf("failed to get the Node the pmm-agent runs on: %w", err)
	}

	redactMode := redactSecrets
	if l.Logger.GetLevel() >= logrus.DebugLevel {
		redactMode = exposeSecrets
	}

	rdsExporters := make(map[*models.Node]*models.Agent)
	agentProcesses := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	builtinAgents := make(map[string]*agentv1.SetStateRequest_BuiltinAgent)
	for _, row := range agents {
		switch row.AgentType {
		case models.PMMAgentType:
			continue
		case models.VMAgentType:
			scrapeCfg, err := u.vmdb.BuildScrapeConfigForVMAgent(q, agent.id)
			if err != nil {
				return fmt.Errorf("cannot get agent scrape config for agent %s: %w", agent.id, err)
			}
			agentProcesses[row.AgentID] = vmAgentConfig(string(scrapeCfg), u.vmParams)
		case models.NomadAgentType:
			node, err := findNode(pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			params, err := nomadClientConfig(u.nomad, node, row)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.NodeExporterType:
			node, err := findNode(pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}

			params, err := nodeExporterConfig(node, row, pmmAgentVersion)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.RDSExporterType:
			node, err := findNode(pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			// rdsExporters is keyed by *Node, so two exporters sharing a Node would collapse
			// into one entry - and one exporter would be dropped - now that the lookup
			// returns the same pointer for both.
			rdsNode := *node
			rdsExporters[&rdsNode] = row

		case models.ExternalExporterType:
			// ignore

		case models.AzureDatabaseExporterType:
			service, err := findService(pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}
			config, err := azureDatabaseExporterConfig(row, service, redactMode, pmmAgentVersion)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = config

		// Agents with exactly one Service
		case models.MySQLdExporterType, models.MongoDBExporterType, models.PostgresExporterType, models.ProxySQLExporterType,
			models.ValkeyExporterType, models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType,
			models.QANMongoDBProfilerAgentType, models.QANMongoDBMongologAgentType,
			models.QANPostgreSQLPgStatementsAgentType, models.QANPostgreSQLPgStatMonitorAgentType,
			models.RTAMongoDBAgentType:
			service, err := findService(pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}
			switch row.AgentType { //nolint:exhaustive
			case models.MySQLdExporterType:
				cfg, err := mysqldExporterConfig(pmmAgentNode, service, row, redactMode, pmmAgentVersion)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.MongoDBExporterType:
				cfg, err := mongodbExporterConfig(pmmAgentNode, service, row, redactMode, pmmAgentVersion)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.PostgresExporterType:
				cfg, err := postgresExporterConfig(pmmAgentNode, service, row, redactMode, pmmAgentVersion)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.ProxySQLExporterType:
				agentProcesses[row.AgentID] = proxysqlExporterConfig(pmmAgentNode, service, row, redactMode, pmmAgentVersion)
			case models.ValkeyExporterType:
				agentProcesses[row.AgentID] = valkeyExporterConfig(pmmAgentNode, service, row, redactMode, pmmAgentVersion)
			case models.QANMySQLPerfSchemaAgentType:
				builtinAgents[row.AgentID] = qanMySQLPerfSchemaAgentConfig(service, row, pmmAgentVersion)
			case models.QANMySQLSlowlogAgentType:
				builtinAgents[row.AgentID] = qanMySQLSlowlogAgentConfig(service, row, pmmAgentVersion)
			case models.QANMongoDBProfilerAgentType:
				builtinAgents[row.AgentID] = qanMongoDBProfilerAgentConfig(service, row, pmmAgentVersion)
			case models.QANMongoDBMongologAgentType:
				builtinAgents[row.AgentID] = qanMongoDBMongologAgentConfig(service, row, pmmAgentVersion)
			case models.QANPostgreSQLPgStatementsAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatementsAgentConfig(service, row, pmmAgentVersion)
			case models.QANPostgreSQLPgStatMonitorAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatMonitorAgentConfig(service, row, pmmAgentVersion)
			case models.RTAMongoDBAgentType:
				builtinAgents[row.AgentID] = rtaMongoDBAgentConfig(service, row, pmmAgentVersion)
			}

		default:
			return fmt.Errorf("cannot send request for unknown agent type %s", row.AgentType)
		}
	}

	// we do start rds exporter per AWS account.
	if len(rdsExporters) != 0 {
		// Create a new map to hold the groups of RDS exporters
		groupedRdsExporters := make(map[string]map[*models.Node]*models.Agent)

		// Iterate over the rdsExporters map
		for node, exporter := range rdsExporters {
			awsAccessKey := exporter.AWSOptions.AWSAccessKey

			if _, ok := groupedRdsExporters[awsAccessKey]; !ok {
				groupedRdsExporters[awsAccessKey] = make(map[*models.Node]*models.Agent)
			}

			groupedRdsExporters[awsAccessKey][node] = exporter
		}

		for awsAccessKey, exporters := range groupedRdsExporters {
			// TODO: split by 50 exporters per group
			groupID := u.r.roster.add(agent.id, rdsPrefix+awsAccessKey, exporters)
			c, err := rdsExporterConfig(exporters, redactMode, pmmAgentVersion)
			if err != nil {
				return err
			}
			agentProcesses[groupID] = c
		}
	}

	state := &agentv1.SetStateRequest{
		AgentProcesses: agentProcesses,
		BuiltinAgents:  builtinAgents,
	}

	// Check log level before calling formatting function.
	// Do not waste resources in case debug level is not enabled.
	if l.Logger.IsLevelEnabled(logrus.DebugLevel) {
		l.Debugf("sendSetStateRequest:\n%s\n", prototext.Format(logger.RedactMessage(state)))
	}

	resp, err := agent.channel.SendAndWaitResponse(ctx, state)
	if err != nil {
		return err
	}
	l.Infof("SetState response: %+v.", resp)
	return nil
}
