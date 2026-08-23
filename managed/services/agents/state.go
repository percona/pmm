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
	"math/rand/v2"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/prototext"
	"gopkg.in/reform.v1"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/version"
)

const (
	// Constants for delayed batch updates.
	updateBatchDelay                = time.Second
	stateChangeTimeout              = 5 * time.Second
	loggerComponentNameStateUpdater = "state-updater"

	// Constants for the failure retry backoff.
	retryBaseDelay  = time.Second
	retryMaxDelay   = 30 * time.Second
	retryMaxShift   = 5
	retryJitterFrac = 2
)

// retryDelay returns an exponentially increasing, jittered delay for the given number of
// consecutive sendSetStateRequest failures.
//
// The jitter matters as much as the backoff: a fleet that failed together would otherwise
// retry together, rebuilding the same thundering herd on every interval (PMM-15228).
func retryDelay(failures int) time.Duration {
	if failures < 1 {
		failures = 1
	}
	d := min(retryBaseDelay<<min(failures-1, retryMaxShift), retryMaxDelay)
	// Spread over [d/2, d) so agents that failed together pick different moments.
	half := int64(d) / retryJitterFrac
	return time.Duration(half + rand.Int64N(half)) //nolint:gosec // retry spacing, not security-sensitive
}

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
	pmmAgents, err := models.FindAllPMMAgentsIDs(u.db.Querier)
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

	var failures int

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
			cancel()
			if err == nil {
				failures = 0
				continue
			}

			// Now that the channel honours nCtx, a request loop busy for longer than
			// stateChangeTimeout lands here, so say what timed out.
			failures++
			l.Errorf("Failed to send SetState request (attempt %d): %s", failures, err)

			// Wait before re-requesting. Without this every failure re-queues immediately,
			// so a fleet reconnecting at once keeps the server saturated with retries and
			// never lets it recover (PMM-15228).
			delay := retryDelay(failures)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-agent.kickChan:
				timer.Stop()
				return
			case <-timer.C:
			}
			u.RequestStateUpdate(ctx, agent.id)
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
	pmmAgent, err := models.FindAgentByID(u.db.Querier, agent.id)
	if err != nil {
		return fmt.Errorf("failed to get PMM Agent: %w", err)
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return fmt.Errorf("failed to parse PMM agent version %q: %w", *pmmAgent.Version, err)
	}

	settings, err := models.GetSettings(u.db.Querier)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	filters := models.AgentFilters{
		PMMAgentID:  agent.id,
		IgnoreNomad: !settings.IsNomadEnabled(),
		// fetch enabled only
		Disabled: new(false),
	}
	agents, err := models.FindAgents(u.db.Querier, filters)
	if err != nil {
		return fmt.Errorf("failed to collect agents: %w", err)
	}

	redactMode := redactSecrets
	if l.Logger.GetLevel() >= logrus.DebugLevel {
		redactMode = exposeSecrets
	}

	// Agent rows belonging to one pmm-agent share the same node, and several of them
	// share the same service, so the per-row lookups below repeated the same queries.
	// Under a fleet-wide reconnect that repetition dominates the cost (PMM-15228).
	nodeCache := make(map[string]*models.Node, len(agents))
	serviceCache := make(map[string]*models.Service, len(agents))

	getNode := func(id string) (*models.Node, error) {
		if n, ok := nodeCache[id]; ok {
			return n, nil
		}
		n, err := models.FindNodeByID(u.db.Querier, id)
		if err != nil {
			return nil, err
		}
		nodeCache[id] = n
		return n, nil
	}
	getService := func(id string) (*models.Service, error) {
		if s, ok := serviceCache[id]; ok {
			return s, nil
		}
		s, err := models.FindServiceByID(u.db.Querier, id)
		if err != nil {
			return nil, err
		}
		serviceCache[id] = s
		return s, nil
	}

	rdsExporters := make(map[*models.Node]*models.Agent)
	agentProcesses := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	builtinAgents := make(map[string]*agentv1.SetStateRequest_BuiltinAgent)
	for _, row := range agents {
		switch row.AgentType {
		case models.PMMAgentType:
			continue
		case models.VMAgentType:
			scrapeCfg, err := u.vmdb.BuildScrapeConfigForVMAgent(ctx, agent.id)
			if err != nil {
				return fmt.Errorf("cannot get agent scrape config for agent %s: %w", agent.id, err)
			}
			agentProcesses[row.AgentID] = vmAgentConfig(string(scrapeCfg), u.vmParams)
		case models.NomadAgentType:
			node, err := getNode(pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			params, err := nomadClientConfig(u.nomad, node, row)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.NodeExporterType:
			node, err := getNode(pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}

			params, err := nodeExporterConfig(node, row, pmmAgentVersion)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.RDSExporterType:
			node, err := getNode(pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			rdsExporters[node] = row

		case models.ExternalExporterType:
			// ignore

		case models.AzureDatabaseExporterType:
			service, err := getService(pointer.GetString(row.ServiceID))
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
			service, err := getService(pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}
			node, _ := getNode(pointer.GetString(pmmAgent.RunsOnNodeID))
			switch row.AgentType { //nolint:exhaustive
			case models.MySQLdExporterType:
				cfg, err := mysqldExporterConfig(node, service, row, redactMode, pmmAgentVersion)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.MongoDBExporterType:
				cfg, err := mongodbExporterConfig(node, service, row, redactMode, pmmAgentVersion)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.PostgresExporterType:
				cfg, err := postgresExporterConfig(node, service, row, redactMode, pmmAgentVersion)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.ProxySQLExporterType:
				agentProcesses[row.AgentID] = proxysqlExporterConfig(node, service, row, redactMode, pmmAgentVersion)
			case models.ValkeyExporterType:
				agentProcesses[row.AgentID] = valkeyExporterConfig(node, service, row, redactMode, pmmAgentVersion)
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
