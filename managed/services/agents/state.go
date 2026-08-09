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

	"github.com/percona/pmm/agent/utils/backoff"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/utils/logger"
	"github.com/percona/pmm/utils/rateLimiter"
)

const (
	// Constants for delayed batch updates.
	updateBatchDelay = time.Second
	// State update parameters.
	stateChangeTimeout = 5 * time.Second
	// Backoff delays to distribute re-try attempts to update agent's state in case of failure (e.g. DB timeout).
	backoffMinDelay                 = 1 * time.Second
	backoffMaxDelay                 = 10 * time.Second
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
	//
	stateUpdateRateLimiter *rateLimiter.ConcurrencyLimiter
}

// NewStateUpdater creates new agent state updater.
func NewStateUpdater(db *reform.DB,
	r *Registry,
	vmdb prometheusService,
	vmParams victoriaMetricsParams,
	nomad nomad,
	maxConcurrentUpdates int32) *StateUpdater {
	return &StateUpdater{
		db:       db,
		r:        r,
		vmdb:     vmdb,
		vmParams: vmParams,
		nomad:    nomad,
		stateUpdateRateLimiter: rateLimiter.NewConcurrencyLimiter(maxConcurrentUpdates),
	}
}

// RequestStateUpdate requests state update on pmm-agent with given ID. It sets
// the status to done if the agent is not connected.
func (u *StateUpdater) RequestStateUpdate(ctx context.Context, pmmAgentID string) {
	l := logger.Get(ctx).WithFields(logrus.Fields{
		"component":    loggerComponentNameStateUpdater,
		"pmm_agent_id": pmmAgentID,
	})

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

var errStateUpdateLimitExceeded = errors.New("state update limit exceeded")

// runStateChangeHandler runs pmm-agent state update loop for given pmm-agent until ctx is canceled or agent is kicked.
func (u *StateUpdater) runStateChangeHandler(ctx context.Context, agent pmmAgentInfo) {
	// NOTE: ctx here tied to gRPC stream from /agent.v1.AgentService/Connect handler
	// and is alive while connection to pmm-agent is up.
	l := logger.Get(ctx).WithField("component", loggerComponentNameStateUpdater)

	l.Info("Starting runStateChangeHandler ...")
	defer l.Info("Done runStateChangeHandler.")

	// stateChangeChan, state update loop, and RequestStateUpdate method ensure that state
	// is reloaded when requested, but several requests are batched together to avoid too often reloads.
	// That allows the caller to just call RequestStateUpdate when it seems fit.
	if cap(agent.stateChangeChan) != 1 {
		panic("stateChangeChan should have capacity 1")
	}

	// stateUpdateBackoff is used to avoid thundering herd problem when many
	// pmm-agents are trying to update their state at the same time
	// and fail due to (e.g. DB or context timeout). It is used to avoid system degradation
	// (exhausted DB connections in particular).
	stateUpdateBackoff := backoff.New(backoffMinDelay, backoffMaxDelay)
	timer := time.NewTimer(updateBatchDelay)
	defer timer.Stop()

	stateUpdateErrorHandler := func(ctx context.Context, hErr error) {
		l.Error(hErr)
		if errors.Is(hErr, context.DeadlineExceeded) ||
			errors.Is(hErr, errStateUpdateLimitExceeded) {
			// state update failed due to context timeout
			// (most likely - waiting for free SQL connection in pool).
			// Sleep for a while to avoid thundering herd problem when many
			// pmm-agents are trying to update their state at the same time.
			timer.Reset(stateUpdateBackoff.Delay())
			select {
			case <-timer.C:
			case <-ctx.Done():
				return
			}
		}
		u.RequestStateUpdate(ctx, agent.id)
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-agent.kickChan:
			return

		case <-agent.stateChangeChan:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(updateBatchDelay)

			select {
			// batch several update requests together by delaying the first one
			case <-timer.C:
			case <-ctx.Done():
				return
			}

			if !u.stateUpdateRateLimiter.TryAcquire() {
				stateUpdateErrorHandler(ctx, errStateUpdateLimitExceeded)
				continue
			}
			nCtx, cancel := context.WithTimeout(ctx, stateChangeTimeout)
			err := u.sendSetStateRequest(nCtx, agent)
			cancel()
			u.stateUpdateRateLimiter.Release()
			if err != nil {
				stateUpdateErrorHandler(ctx, err)
				continue
			}
			// seems DB came back to normal - reset backoff.
			stateUpdateBackoff.Reset()
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

	// Use singleflight to avoid fetching settings for each pmm-agent separately.
	fetchedSettings, err, _ := u.dbGroup.Do("settings", func() (any, error) { //nolint:contextcheck
		// NOTE 1: leader's context is not used here directly in order to allow to finish
		// the request to DB, so that even if leader's request is already terminated -
		// the rest of waiters in singleflight group will receive the response from DB.
		settingsCtx, cancel := context.WithTimeout(context.Background(), stateChangeTimeout)
		defer cancel()

		sett, fetchErr := models.GetSettings(u.db.WithContext(settingsCtx))
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
		l.WithError(err).Error("failed to get settings")
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
	// It is completely OK to re-use the same Querier for multiple queries, as it is safe for concurrent use
	// and creates less preasure on GC.
	q := u.db.WithContext(ctx)
	agents, err := models.FindAgents(q, filters)
	if err != nil {
		l.WithError(err).Errorf("failed to collect agents")
		return fmt.Errorf("failed to collect agents: %w", err)
	}

	// pre-fetch node info since it's common for all subagents of particluar pmm-agent.
	// Use singleflight to avoid fetching settings for each pmm-agent separately in cases
	// when several pmm-agents are running on the same node.
	nodeKey := "node/" + pmmAgentInfo.runsOnNodeID
	fetchedNode, err, _ := u.dbGroup.Do(nodeKey, func() (any, error) { //nolint:contextcheck
		// NOTE 1: leader's context is not used here directly in order to allow to finish
		// the request to DB, so that even if leader's request is already terminated -
		// the rest of waiters in singleflight group will receive the response from DB.
		nodeCtx, cancel := context.WithTimeout(context.Background(), stateChangeTimeout)
		defer cancel()
		node, fetchErr := models.FindNodeByID(u.db.WithContext(nodeCtx), pmmAgentInfo.runsOnNodeID)
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
		l.WithError(err).
			WithField("node_id", pmmAgentInfo.runsOnNodeID).
			Error("failed to fetch node info")
		return fmt.Errorf("failed to fetch node info: %w", err)
	}
	node, ok := fetchedNode.(*models.Node)
	if !ok {
		l.WithField("node_id", pmmAgentInfo.runsOnNodeID).
			Errorf("failed to cast Node: %T", fetchedNode)
		return fmt.Errorf("failed to fetch node %s info", pmmAgentInfo.runsOnNodeID)
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
			params, err := nodeExporterConfig(node, row, pmmAgentInfo.version)
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
			config, err := azureDatabaseExporterConfig(row, service, redactMode, pmmAgentInfo.version)
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
				cfg, err := mysqldExporterConfig(node, service, row, redactMode, pmmAgentInfo.version)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.MongoDBExporterType:
				cfg, err := mongodbExporterConfig(node, service, row, redactMode, pmmAgentInfo.version)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.PostgresExporterType:
				cfg, err := postgresExporterConfig(node, service, row, redactMode, pmmAgentInfo.version)
				if err != nil {
					return err
				}
				agentProcesses[row.AgentID] = cfg
			case models.ProxySQLExporterType:
				agentProcesses[row.AgentID] = proxysqlExporterConfig(node, service, row, redactMode, pmmAgentInfo.version)
			case models.ValkeyExporterType:
				agentProcesses[row.AgentID] = valkeyExporterConfig(node, service, row, redactMode, pmmAgentInfo.version)
			case models.QANMySQLPerfSchemaAgentType:
				builtinAgents[row.AgentID] = qanMySQLPerfSchemaAgentConfig(service, row, pmmAgentInfo.version)
			case models.QANMySQLSlowlogAgentType:
				builtinAgents[row.AgentID] = qanMySQLSlowlogAgentConfig(service, row, pmmAgentInfo.version)
			case models.QANMongoDBProfilerAgentType:
				builtinAgents[row.AgentID] = qanMongoDBProfilerAgentConfig(service, row, pmmAgentInfo.version)
			case models.QANMongoDBMongologAgentType:
				builtinAgents[row.AgentID] = qanMongoDBMongologAgentConfig(service, row, pmmAgentInfo.version)
			case models.QANPostgreSQLPgStatementsAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatementsAgentConfig(service, row, pmmAgentInfo.version)
			case models.QANPostgreSQLPgStatMonitorAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatMonitorAgentConfig(service, row, pmmAgentInfo.version)
			case models.RTAMongoDBAgentType:
				builtinAgents[row.AgentID] = rtaMongoDBAgentConfig(service, row, pmmAgentInfo.version)
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
			c, err := rdsExporterConfig(exporters, redactMode, pmmAgentInfo.version)
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
