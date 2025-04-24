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
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
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
	updateBatchDelay   = time.Second
	stateChangeTimeout = 5 * time.Second
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
	l := logger.Get(ctx)

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
		return errors.Wrap(err, "cannot find pmmAgentsIDs for AgentsState update")
	}
	var wg sync.WaitGroup
	limiter := make(chan struct{}, 10)
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
	l := logger.Get(ctx).WithField("agent_id", agent.id)

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
func (u *StateUpdater) sendSetStateRequest(ctx context.Context, agent *pmmAgentInfo) error { //nolint:cyclop
	l := logger.Get(ctx)
	start := time.Now()
	defer func() {
		if dur := time.Since(start); dur > time.Second {
			l.Warnf("sendSetStateRequest took %s.", dur)
		}
	}()
	pmmAgent, err := models.FindAgentByID(u.db.Querier, agent.id)
	if err != nil {
		return errors.Wrap(err, "failed to get PMM Agent")
	}
	pmmAgentVersion, err := version.Parse(*pmmAgent.Version)
	if err != nil {
		return errors.Wrapf(err, "failed to parse PMM agent version %q", *pmmAgent.Version)
	}

	settings, err := models.GetSettings(u.db.Querier)
	if err != nil {
		return errors.Wrap(err, "failed to get settings")
	}

	filters := models.AgentFilters{
		PMMAgentID:  agent.id,
		IgnoreNomad: !settings.IsNomadEnabled(),
	}
	agents, err := models.FindAgents(u.db.Querier, filters)
	if err != nil {
		return errors.Wrap(err, "failed to collect agents")
	}

	redactMode := redactSecrets
	if l.Logger.GetLevel() >= logrus.DebugLevel {
		redactMode = exposeSecrets
	}

	rdsExporters := make(map[*models.Node]*models.Agent)
	agentProcesses := make(map[string]*agentv1.SetStateRequest_AgentProcess)
	builtinAgents := make(map[string]*agentv1.SetStateRequest_BuiltinAgent)
	for _, row := range agents {
		if row.Disabled {
			continue
		}

		// Ordered the same as AgentType consts
		switch row.AgentType {
		case models.PMMAgentType:
			continue
		case models.VMAgentType:
			scrapeCfg, err := u.vmdb.BuildScrapeConfigForVMAgent(agent.id)
			if err != nil {
				return errors.Wrapf(err, "cannot get agent scrape config for agent: %s", agent.id)
			}
			agentProcesses[row.AgentID] = vmAgentConfig(string(scrapeCfg), u.vmParams)
		case models.NomadAgentType:
			node, err := models.FindNodeByID(u.db.Querier, pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			params, err := nomadClientConfig(u.nomad, node, row)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.NodeExporterType:
			node, err := models.FindNodeByID(u.db.Querier, pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}

			params, err := nodeExporterConfig(node, row, pmmAgentVersion)
			if err != nil {
				return err
			}
			agentProcesses[row.AgentID] = params

		case models.RDSExporterType:
			node, err := models.FindNodeByID(u.db.Querier, pointer.GetString(row.NodeID))
			if err != nil {
				return err
			}
			rdsExporters[node] = row

		case models.ExternalExporterType:
			// ignore

		case models.AzureDatabaseExporterType:
			service, err := models.FindServiceByID(u.db.Querier, pointer.GetString(row.ServiceID))
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
			models.QANMySQLPerfSchemaAgentType, models.QANMySQLSlowlogAgentType, models.QANMongoDBProfilerAgentType, models.QANPostgreSQLPgStatementsAgentType,
			models.QANPostgreSQLPgStatMonitorAgentType:

			service, err := models.FindServiceByID(u.db.Querier, pointer.GetString(row.ServiceID))
			if err != nil {
				return err
			}
			node, _ := models.FindNodeByID(u.db.Querier, pointer.GetString(pmmAgent.RunsOnNodeID))
			switch row.AgentType { //nolint:exhaustive
			case models.MySQLdExporterType:
				agentProcesses[row.AgentID] = mysqldExporterConfig(node, service, row, redactMode, pmmAgentVersion)
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
			case models.QANMySQLPerfSchemaAgentType:
				builtinAgents[row.AgentID] = qanMySQLPerfSchemaAgentConfig(service, row, pmmAgentVersion)
			case models.QANMySQLSlowlogAgentType:
				builtinAgents[row.AgentID] = qanMySQLSlowlogAgentConfig(service, row, pmmAgentVersion)
			case models.QANMongoDBProfilerAgentType:
				builtinAgents[row.AgentID] = qanMongoDBProfilerAgentConfig(service, row, pmmAgentVersion)
			case models.QANPostgreSQLPgStatementsAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatementsAgentConfig(service, row, pmmAgentVersion)
			case models.QANPostgreSQLPgStatMonitorAgentType:
				builtinAgents[row.AgentID] = qanPostgreSQLPgStatMonitorAgentConfig(service, row, pmmAgentVersion)
			}

		default:
			return errors.Errorf("unhandled Agent type %s", row.AgentType)
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
	l.Debugf("sendSetStateRequest:\n%s", prototext.Format(state))

	resp, err := agent.channel.SendAndWaitResponse(state)
	if err != nil {
		return err
	}
	l.Infof("SetState response: %+v.", resp)
	return nil
}
