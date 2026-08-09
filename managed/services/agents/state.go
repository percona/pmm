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
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
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
)

// StateUpdater handles updating status of agents.
type StateUpdater struct {
	db       *reform.DB
	r        *Registry
	vmdb     prometheusService
	vmParams victoriaMetricsParams
	nomad    nomad
	// dbGroup deduplicates concurrent requests to DB (for example models.GetSettings).
	// In case of massive pmm-agents connection attempts  no need to query DB for each of them, just one is enough.
	dbGroup singleflight.Group
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
	var wg errgroup.Group
	wg.SetLimit(runtime.GOMAXPROCS(0))

	for i := range pmmAgents {
		wg.Go(func() error {
			u.RequestStateUpdate(ctx, pmmAgents[i])
			return nil
		})
	}
	return wg.Wait()
}

// runStateChangeHandler runs pmm-agent state update loop for given pmm-agent until ctx is canceled or agent is kicked.
func (u *StateUpdater) runStateChangeHandler(ctx context.Context, agent pmmAgentInfo) {
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
				l.Error(err)
				u.RequestStateUpdate(ctx, agent.id)
			}
			cancel()
		}
	}
}

// sendSetStateRequest sends SetStateRequest to given pmm-agent.
func (u *StateUpdater) sendSetStateRequest(ctx context.Context, pmmAgentInfo pmmAgentInfo) error { //nolint:gocognit,cyclop,maintidx
	l := logger.Get(ctx).WithField("component", loggerComponentNameStateUpdater)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			l.Warnf("sendSetStateRequest took %s.", dur)
		}
	}()
	// It is completely OK to re-use the same Querier for multiple queries, as it is safe for concurrent use
	// and creates less preasure on GC.
	q := u.db.WithContext(ctx)
	pmmAgent, err := models.FindAgentByID(q, pmmAgentInfo.id)
	if err != nil {
		return fmt.Errorf("failed to get PMM Agent: %w", err)
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return fmt.Errorf("failed to parse PMM agent version %q: %w", *pmmAgent.Version, err)
	}

	// Use singleflight to avoid fetching settings for each pmm-agent separately.
	fetchedSettings, err, _ := u.dbGroup.Do("settings", func() (any, error) {
		// NOTE 1: The first request for a singlefligh.Do() becomes the leader and runs the closure with its ctx.
		// If this ctx is canceled - it will have no effect on the closure, so that all waiting
		// requests will be able to get the result.

		// NOTE 2: leader's context is not used here directly in order to allow to finish
		// the request to DB, so that even if leader's request is already terminated -
		// the rest of waiters in singleflight group will receive the response from DB.
		deadLine, ok := ctx.Deadline()
		if !ok {
			deadLine = time.Now().Add(stateChangeTimeout)
		}
		settingsCtx, cancel := context.WithDeadline(context.Background(), deadLine)
		defer cancel()

		sett, fetchErr := models.GetSettings(u.db.WithContext(settingsCtx)) //nolint:contextcheck
		if fetchErr != nil {
			// IMPORTANT: On error, we call Forget(hash) IMMEDIATELY.
			// This prevents the error from getting stuck in the internal singleflight map
			// and allows the next request to retry immediately (e.g. if the DB is being restored).
			u.dbGroup.Forget("settings")
			return nil, fetchErr
		}
		return sett, nil
	})
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	settings, ok := fetchedSettings.(*models.Settings)
	if !ok {
		l.Errorf("failed to cast settings: %T", fetchedSettings)
		return errors.New("failed to get settings")
	}
	filters := models.AgentFilters{
		PMMAgentID:  pmmAgentInfo.id,
		IgnoreNomad: !settings.IsNomadEnabled(),
		// fetch enabled only
		Disabled: new(false),
	}
	agents, err := models.FindAgents(q, filters)
	if err != nil {
		return fmt.Errorf("failed to collect agents: %w", err)
	}

	// pre-fetch node info since it's common for all subagents of particluar pmm-agent.
	// Use singleflight to avoid fetching settings for each pmm-agent separately in cases
	// when several pmm-agents are running on the same node.
	nodeKey := "node/" + pointer.GetString(pmmAgent.RunsOnNodeID)
	fetchedNode, err, _ := u.dbGroup.Do(nodeKey, func() (any, error) {
		// NOTE 1: The first request for a singlefligh.Do() becomes the leader and runs the closure with its ctx.
		// If this ctx is canceled - it will have no effect on the closure, so that all waiting
		// requests will be able to get the result.

		// NOTE 2: leader's context is not used here directly in order to allow to finish
		// the request to DB, so that even if leader's request is already terminated -
		// the rest of waiters in singleflight group will receive the response from DB.
		deadLine, ok := ctx.Deadline()
		if !ok {
			deadLine = time.Now().Add(stateChangeTimeout)
		}
		nodeCtx, cancel := context.WithDeadline(context.Background(), deadLine)
		defer cancel()
		node, fetchErr := models.FindNodeByID(u.db.WithContext(nodeCtx), pointer.GetString(pmmAgent.RunsOnNodeID))
		if fetchErr != nil {
			// IMPORTANT: On error, we call Forget(nodeKey) IMMEDIATELY.
			// This prevents the error from getting stuck in the internal singleflight map
			// and allows the next request to retry immediately (e.g. if the DB is being restored).
			u.dbGroup.Forget(nodeKey)
			return nil, fetchErr
		}
		return node, nil
	})
	if err != nil {
		return fmt.Errorf("failed to fetch node info: %w", err)
	}

	node, ok := fetchedNode.(*models.Node)
	if !ok {
		l.WithField("node_id", pointer.GetString(pmmAgent.RunsOnNodeID)).
			Errorf("failed to cast Node: %T", fetchedNode)
		return fmt.Errorf("failed to fetch node %s info", pointer.GetString(pmmAgent.RunsOnNodeID))
	}

	redactMode := redactSecrets
	if l.Logger.IsLevelEnabled(logrus.DebugLevel) {
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
			scrapeCfg, err := u.vmdb.BuildScrapeConfigForVMAgent(q, pmmAgentInfo.id)
			if err != nil {
				return fmt.Errorf("cannot get agent scrape config for agent %s: %w", pmmAgentInfo.id, err)
			}
			agentProcesses[row.AgentID] = vmAgentConfig(string(scrapeCfg), u.vmParams)
		case models.NomadAgentType:
			params, err := nomadClientConfig(u.nomad, node, row)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.NodeExporterType:
			params, err := nodeExporterConfig(node, row, pmmAgentVersion)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.RDSExporterType:
			rdsNode, err := models.FindNodeByID(q, pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			rdsExporters[rdsNode] = row

		case models.ExternalExporterType:
			// ignore

		case models.AzureDatabaseExporterType:
			service, err := models.FindServiceByID(q, pointer.GetString(row.ServiceID))
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
			service, err := models.FindServiceByID(q, pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}
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
			groupID := u.r.roster.add(pmmAgentInfo.id, rdsPrefix+awsAccessKey, exporters)
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

	resp, err := pmmAgentInfo.channel.SendAndWaitResponse(state)
	if err != nil {
		return err
	}
	l.Infof("SetState response: %+v.", resp)
	return nil
}
