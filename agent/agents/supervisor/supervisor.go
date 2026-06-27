// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package supervisor provides supervisor for running Agents.
package supervisor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/agents"
	"github.com/percona/pmm/agent/agents/mongodb/mongolog"
	mongoprofiler "github.com/percona/pmm/agent/agents/mongodb/profiler"
	mongorta "github.com/percona/pmm/agent/agents/mongodb/realtimeanalytics"
	"github.com/percona/pmm/agent/agents/mysql/perfschema"
	"github.com/percona/pmm/agent/agents/mysql/slowlog"
	"github.com/percona/pmm/agent/agents/noop"
	"github.com/percona/pmm/agent/agents/postgres/pgstatmonitor"
	"github.com/percona/pmm/agent/agents/postgres/pgstatstatements"
	"github.com/percona/pmm/agent/agents/process"
	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/tailog"
	"github.com/percona/pmm/agent/utils/templates"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	agentlocal "github.com/percona/pmm/api/agentlocal/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	rtav1 "github.com/percona/pmm/api/realtimeanalytics/v1"
)

const (
	changesBufferSize     = 100
	qanRequestsBufferSize = 100
	rtaRequestsBufferSize = 100
)

// configGetter allows for getting a config.
type configGetter interface {
	Get() *config.Config
}

// Supervisor manages all Agents, both processes and built-in.
type Supervisor struct {
	// TODO: refactor to move context outside of struct
	ctx            context.Context //nolint:containedctx
	agentVersioner agentVersioner
	cfg            configGetter
	portsRegistry  *portsRegistry
	changes        chan *agentv1.StateChangedRequest
	qanRequests    chan *agentv1.QANCollectRequest
	rtaRequests    chan *rtav1.CollectRequest
	l              *logrus.Entry

	rw             sync.RWMutex
	agentProcesses map[string]*agentProcessInfo
	builtinAgents  map[string]*builtinAgentInfo

	arw          sync.RWMutex
	lastStatuses map[string]inventoryv1.AgentStatus
}

// agentProcessInfo describes Agent process.
type agentProcessInfo struct {
	cancel          func()          // to cancel Process.Run(ctx)
	done            <-chan struct{} // closes when Process.Changes() channel closes
	requestedState  *agentv1.SetStateRequest_AgentProcess
	listenPort      uint16
	processExecPath string
	logStore        *tailog.Store // store logs
}

// builtinAgentInfo describes built-in Agent.
type builtinAgentInfo struct {
	cancel         func()          // to cancel AgentType.Run(ctx)
	done           <-chan struct{} // closes when AgentType.Changes() channel closes
	requestedState *agentv1.SetStateRequest_BuiltinAgent
	describe       func(chan<- *prometheus.Desc)  // agent's func to describe Prometheus metrics
	collect        func(chan<- prometheus.Metric) // agent's func to provide Prometheus metrics
	logStore       *tailog.Store                  // store logs
}

// NewSupervisor creates new Supervisor object.
//
// Supervisor is gracefully stopped when context passed to NewSupervisor is canceled.
// Changes of Agent statuses are reported via Changes() channel which must be read until it is closed.
// QAN data is sent to QANRequests() channel which must be read until it is closed.
// RTA data is sent to RTARequests() channel which must be read until it is closed.
func NewSupervisor(ctx context.Context, av agentVersioner, cfg configGetter) *Supervisor {
	return &Supervisor{
		ctx:            ctx,
		agentVersioner: av,
		cfg:            cfg,
		portsRegistry:  newPortsRegistry(cfg.Get().Ports.Min, cfg.Get().Ports.Max, nil),
		changes:        make(chan *agentv1.StateChangedRequest, changesBufferSize),
		qanRequests:    make(chan *agentv1.QANCollectRequest, qanRequestsBufferSize),
		rtaRequests:    make(chan *rtav1.CollectRequest, rtaRequestsBufferSize),
		l:              logrus.WithField("component", "supervisor"),

		agentProcesses: make(map[string]*agentProcessInfo),
		builtinAgents:  make(map[string]*builtinAgentInfo),
		lastStatuses:   make(map[string]inventoryv1.AgentStatus),
	}
}

// Run waits for context and stop all agents when it's done.
func (s *Supervisor) Run(ctx context.Context) {
	<-ctx.Done()
	s.stopAll() //nolint:contextcheck
}

// AgentsList returns info for all Agents managed by this supervisor.
func (s *Supervisor) AgentsList() []*agentlocal.AgentInfo {
	s.rw.RLock()
	defer s.rw.RUnlock()
	s.arw.RLock()
	defer s.arw.RUnlock()

	res := make([]*agentlocal.AgentInfo, 0, len(s.agentProcesses)+len(s.builtinAgents))

	for id, agent := range s.agentProcesses {
		info := &agentlocal.AgentInfo{
			AgentId:         id,
			AgentType:       agent.requestedState.Type,
			Status:          s.lastStatuses[id],
			ListenPort:      uint32(agent.listenPort),
			ProcessExecPath: agent.processExecPath,
		}
		res = append(res, info)
	}

	for id, agent := range s.builtinAgents {
		info := &agentlocal.AgentInfo{
			AgentId:   id,
			AgentType: agent.requestedState.Type,
			Status:    s.lastStatuses[id],
		}
		res = append(res, info)
	}

	sort.Slice(res, func(i, j int) bool { return res[i].AgentId < res[j].AgentId })
	return res
}

// AgentsLogs returns logs for all Agents managed by this supervisor.
func (s *Supervisor) AgentsLogs() map[string][]string {
	s.rw.RLock()
	defer s.rw.RUnlock()

	res := make(map[string][]string, len(s.agentProcesses)+len(s.builtinAgents))

	for id, agent := range s.agentProcesses {
		res[fmt.Sprintf("%s %s", agent.requestedState.Type.String(), id)], _ = agent.logStore.GetLogs()
	}

	for id, agent := range s.builtinAgents {
		res[fmt.Sprintf("%s %s", agent.requestedState.Type.String(), id)], _ = agent.logStore.GetLogs()
	}
	return res
}

// AgentLogByID returns logs by Agent ID.
func (s *Supervisor) AgentLogByID(id string) ([]string, uint) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	agentProcess, ok := s.agentProcesses[id]
	if ok {
		return agentProcess.logStore.GetLogs()
	}

	builtinAgent, ok := s.builtinAgents[id]
	if ok {
		return builtinAgent.logStore.GetLogs()
	}

	return nil, 0
}

// ClearChangesChannel drains state change channel.
func (s *Supervisor) ClearChangesChannel() {
	for {
		select {
		case _, ok := <-s.changes:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

// Changes returns channel with Agent's state changes.
func (s *Supervisor) Changes() <-chan *agentv1.StateChangedRequest {
	return s.changes
}

// QANRequests returns channel with Agent's QAN Collect requests.
func (s *Supervisor) QANRequests() <-chan *agentv1.QANCollectRequest {
	return s.qanRequests
}

// RTARequests returns channel with Agent's RTA Collect requests.
func (s *Supervisor) RTARequests() <-chan *rtav1.CollectRequest {
	return s.rtaRequests
}

// SetState starts or updates all agents placed in args and stops all agents not placed in args, but already run.
func (s *Supervisor) SetState(state *agentv1.SetStateRequest) {
	// do not process SetState requests concurrently for internal state consistency and implementation simplicity
	s.rw.Lock()
	defer s.rw.Unlock()

	// check if we waited for lock too long
	err := s.ctx.Err()
	if err != nil {
		s.l.Errorf("Ignoring SetState: %s.", err)
		return
	}

	s.setAgentProcesses(state.AgentProcesses)
	s.setBuiltinAgents(state.BuiltinAgents)
}

// RestartAgents restarts all existing agents.
func (s *Supervisor) RestartAgents() {
	s.rw.Lock()
	defer s.rw.Unlock()

	for id, agent := range s.agentProcesses {
		agent.cancel()
		<-agent.done

		err := s.tryStartProcess(id, agent.requestedState, agent.listenPort)
		if err != nil {
			s.l.Errorf("Failed to restart Agent: %s.", err)
		}
	}

	for id, agent := range s.builtinAgents {
		agent.cancel()
		<-agent.done

		err := s.startBuiltin(id, agent.requestedState)
		if err != nil {
			s.l.Errorf("Failed to restart Agent: %s.", err)
		}
	}
}

func (s *Supervisor) storeLastStatus(agentID string, status inventoryv1.AgentStatus) {
	s.arw.Lock()
	defer s.arw.Unlock()

	if status == inventoryv1.AgentStatus_AGENT_STATUS_DONE {
		delete(s.lastStatuses, agentID)
		return
	}

	s.lastStatuses[agentID] = status
}

// setAgentProcesses starts/restarts/stops Agent processes.
// Must be called with s.rw held for writing.
func (s *Supervisor) setAgentProcesses(agentProcesses map[string]*agentv1.SetStateRequest_AgentProcess) {
	existingParams := make(map[string]agentv1.AgentParams)
	for id, p := range s.agentProcesses {
		existingParams[id] = p.requestedState
	}
	newParams := make(map[string]agentv1.AgentParams)
	for id, p := range agentProcesses {
		newParams[id] = p
	}
	toStart, toRestart, toStop := filter(existingParams, newParams)
	if len(toStart)+len(toRestart)+len(toStop) == 0 {
		return
	}
	s.l.Infof("Starting %d, restarting %d, and stopping %d agent processes.", len(toStart), len(toRestart), len(toStop))

	// We have to wait for Agents to terminate before starting a new ones to send all state updates,
	// and to reuse ports.
	// If that place is slow, we can cancel them all in parallel, but then we still have to wait.

	// stop first to avoid extra load
	for _, agentID := range toStop {
		agent := s.agentProcesses[agentID]
		agent.cancel()
		<-agent.done

		err := s.portsRegistry.Release(agent.listenPort)
		if err != nil {
			s.l.Errorf("Failed to release port: %s.", err)
		}

		delete(s.agentProcesses, agentID)

		agentTmp := filepath.Join(s.cfg.Get().Paths.TempDir, trimPrefix(agent.requestedState.Type.String()), agentID)
		err = os.RemoveAll(agentTmp)
		if err != nil {
			s.l.Warnf("Failed to cleanup directory '%s': %s", agentTmp, err.Error())
		}
	}

	// restart while preserving port
	for _, agentID := range toRestart {
		agent := s.agentProcesses[agentID]
		agent.cancel()
		<-agent.done

		err := s.tryStartProcess(agentID, agentProcesses[agentID], agent.listenPort)
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}

	// start new agents
	for _, agentID := range toStart {
		err := s.tryStartProcess(agentID, agentProcesses[agentID], 0)
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}
}

// setBuiltinAgents starts/restarts/stops built-in Agents.
// Must be called with s.rw held for writing.
func (s *Supervisor) setBuiltinAgents(builtinAgents map[string]*agentv1.SetStateRequest_BuiltinAgent) {
	existingParams := make(map[string]agentv1.AgentParams)
	for id, agent := range s.builtinAgents {
		existingParams[id] = agent.requestedState
	}
	newParams := make(map[string]agentv1.AgentParams)
	for id, agent := range builtinAgents {
		newParams[id] = agent
	}
	toStart, toRestart, toStop := filter(existingParams, newParams)
	if len(toStart)+len(toRestart)+len(toStop) == 0 {
		return
	}
	s.l.Infof("Starting %d, restarting %d, and stopping %d built-in agents.", len(toStart), len(toRestart), len(toStop))

	// We have to wait for Agents to terminate before starting a new ones to send all state updates.
	// If that place is slow, we can cancel them all in parallel, but then we still have to wait.

	// stop first to avoid extra load
	for _, agentID := range toStop {
		agent := s.builtinAgents[agentID]
		agent.cancel()
		<-agent.done

		delete(s.builtinAgents, agentID)

		agentTmp := filepath.Join(s.cfg.Get().Paths.TempDir, trimPrefix(agent.requestedState.Type.String()), agentID)
		err := os.RemoveAll(agentTmp)
		if err != nil {
			s.l.Warnf("Failed to cleanup directory '%s': %s", agentTmp, err.Error())
		}
	}

	// restart
	for _, agentID := range toRestart {
		agent := s.builtinAgents[agentID]
		agent.cancel()
		<-agent.done

		err := s.startBuiltin(agentID, builtinAgents[agentID])
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}

	// start new agents
	for _, agentID := range toStart {
		err := s.startBuiltin(agentID, builtinAgents[agentID])
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}
}

// filter extracts IDs of the Agents that should be started, restarted with new parameters, or stopped,
// and filters out IDs of the Agents that should not be changed.
func filter(existing, ap map[string]agentv1.AgentParams) ([]string, []string, []string) {
	toStart := make([]string, 0, len(ap))
	toRestart := make([]string, 0, len(ap))
	toStop := make([]string, 0, len(existing))

	// existing agents not present in the new requested state should be stopped
	for existingID := range existing {
		if ap[existingID] == nil {
			toStop = append(toStop, existingID)
		}
	}

	// detect new and changed agents
	for newID, newParams := range ap {
		existingParams := existing[newID]
		if existingParams == nil {
			toStart = append(toStart, newID)
			continue
		}

		// compare parameters before templating
		if proto.Equal(existingParams, newParams) {
			continue
		}

		toRestart = append(toRestart, newID)
	}

	sort.Strings(toStop)
	sort.Strings(toRestart)
	sort.Strings(toStart)

	return toStart, toRestart, toStop
}

//nolint:revive
const (
	type_TEST_SLEEP       inventoryv1.AgentType = 998 // process
	type_TEST_NOOP        inventoryv1.AgentType = 999 // built-in
	process_Retry_Time    int                   = 3
	start_Process_Waiting                       = 2 * time.Second
)

func (s *Supervisor) tryStartProcess(agentID string, agentProcess *agentv1.SetStateRequest_AgentProcess, port uint16) error {
	var err error
	for range process_Retry_Time {
		if port == 0 {
			_port, err := s.portsRegistry.Reserve()
			if err != nil {
				s.l.Errorf("Failed to reserve port: %s.", err)
				continue
			}
			port = _port
		}

		err = s.startProcess(agentID, agentProcess, port)
		if err == nil {
			return nil
		}

		port = 0
	}
	return err
}

// startProcess starts Agent's process.
// Must be called with s.rw held for writing.
func (s *Supervisor) startProcess(agentID string, agentProcess *agentv1.SetStateRequest_AgentProcess, port uint16) error {
	processParams, err := s.processParams(agentID, agentProcess, port)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(s.ctx)
	agentType := trimPrefix(agentProcess.Type.String())
	logStore := tailog.NewStore(s.cfg.Get().LogLinesCount)
	l := s.agentLogger(logStore).WithFields(logrus.Fields{
		"component": "agent-process",
		"agentID":   agentID,
		"type":      agentType,
	})
	l.Debugf("Starting: %s.", processParams)

	process := process.New(processParams, agentProcess.RedactWords, l)
	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), process.Run)

	version, err := s.version(agentProcess.Type, processParams.Path)
	if err != nil {
		l.Warnf("Cannot parse version for type %s", agentType)
	}

	done := make(chan struct{})
	go func() {
		for status := range process.Changes() {
			s.storeLastStatus(agentID, status)
			l.Infof("Sending status: %s (port %d).", status, port)
			s.changes <- &agentv1.StateChangedRequest{
				AgentId:         agentID,
				Status:          status,
				ListenPort:      uint32(port),
				ProcessExecPath: processParams.Path,
				Version:         version,
			}
		}
		close(done)
	}()

	processInfo := &agentProcessInfo{ //nolint:forcetypeassert
		cancel:          cancel,
		done:            done,
		requestedState:  proto.Clone(agentProcess).(*agentv1.SetStateRequest_AgentProcess),
		listenPort:      port,
		processExecPath: processParams.Path,
		logStore:        logStore,
	}

	t := time.NewTimer(start_Process_Waiting)
	defer t.Stop()
	select {
	case isInitialized := <-process.IsInitialized():
		if !isInitialized {
			// TODO: handle initialization error for nomad agent
			if agentProcess.Type == inventoryv1.AgentType_AGENT_TYPE_NOMAD_AGENT {
				s.handleNomadAgent(agentID, processInfo, l)
				return nil
			}
			defer cancel()
			return process.GetError()
		}
	case <-t.C:
	}

	//nolint:forcetypeassert
	s.agentProcesses[agentID] = processInfo
	return nil
}

//nolint:funcorder
func (s *Supervisor) handleNomadAgent(
	agentID string,
	processInfo *agentProcessInfo,
	l *logrus.Entry,
) {
	done := make(chan struct{})
	s.agentProcesses[agentID] = processInfo

	status := inventoryv1.AgentStatus_AGENT_STATUS_DONE
	s.storeLastStatus(agentID, status)
	l.Warn("Cannot start Nomad Agent: cgroups are not writable.")
	l.Infof("Sending status: %s (port %d).", status, processInfo.listenPort)
	s.changes <- &agentv1.StateChangedRequest{
		AgentId:         agentID,
		Status:          status,
		ListenPort:      uint32(processInfo.listenPort),
		ProcessExecPath: processInfo.processExecPath,
	}

	close(done)
}

// startBuiltin starts built-in Agent.
// Must be called with s.rw held for writing.
func (s *Supervisor) startBuiltin(agentID string, builtinAgent *agentv1.SetStateRequest_BuiltinAgent) error {
	cfg := s.cfg.Get()

	ctx, cancel := context.WithCancel(s.ctx)
	agentType := trimPrefix(builtinAgent.Type.String())
	logStore := tailog.NewStore(cfg.LogLinesCount)
	l := s.agentLogger(logStore).WithFields(logrus.Fields{
		"component": "agent-builtin",
		"agentID":   agentID,
		"type":      agentType,
	})

	done := make(chan struct{})
	var agent agents.BuiltinAgent
	var err error

	var dsn string
	if builtinAgent.TextFiles != nil {
		tempDir := filepath.Join(cfg.Paths.TempDir, trimPrefix(builtinAgent.Type.String()), agentID)
		dsn, err = templates.RenderDSN(builtinAgent.Dsn, builtinAgent.TextFiles, tempDir)
		if err != nil {
			cancel()
			return err
		}
	} else {
		dsn = builtinAgent.Dsn
	}

	switch builtinAgent.Type {
	case inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_PERFSCHEMA_AGENT:
		params := &perfschema.Params{
			DSN:                    dsn,
			AgentID:                agentID,
			MaxQueryLength:         builtinAgent.MaxQueryLength,
			DisableCommentsParsing: builtinAgent.DisableCommentsParsing,
			DisableQueryExamples:   builtinAgent.DisableQueryExamples,
			TextFiles:              builtinAgent.GetTextFiles(),
			TLSSkipVerify:          builtinAgent.TlsSkipVerify,
			PerfschemaRefreshRate:  cfg.PerfschemaRefreshRate,
		}
		agent, err = perfschema.New(params, l)

	case inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_PROFILER_AGENT:
		params := &mongoprofiler.Params{
			DSN:            dsn,
			AgentID:        agentID,
			MaxQueryLength: builtinAgent.MaxQueryLength,
		}
		agent, err = mongoprofiler.New(params, l)

	case inventoryv1.AgentType_AGENT_TYPE_QAN_MONGODB_MONGOLOG_AGENT:
		params := &mongolog.Params{
			DSN:            dsn,
			AgentID:        agentID,
			MaxQueryLength: builtinAgent.MaxQueryLength,
		}
		agent, err = mongolog.New(params, l)

	case inventoryv1.AgentType_AGENT_TYPE_QAN_MYSQL_SLOWLOG_AGENT:
		params := &slowlog.Params{
			DSN:                    dsn,
			AgentID:                agentID,
			SlowLogFilePrefix:      cfg.Paths.SlowLogFilePrefix,
			MaxQueryLength:         builtinAgent.MaxQueryLength,
			DisableCommentsParsing: builtinAgent.DisableCommentsParsing,
			DisableQueryExamples:   builtinAgent.DisableQueryExamples,
			MaxSlowlogFileSize:     builtinAgent.MaxQueryLogSize,
			TextFiles:              builtinAgent.GetTextFiles(),
			TLSSkipVerify:          builtinAgent.TlsSkipVerify,
			TLS:                    false,
		}
		agent, err = slowlog.New(params, l)

	case inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATEMENTS_AGENT:
		params := &pgstatstatements.Params{
			DSN:                    dsn,
			AgentID:                agentID,
			MaxQueryLength:         builtinAgent.MaxQueryLength,
			DisableCommentsParsing: builtinAgent.DisableCommentsParsing,
			TextFiles:              builtinAgent.GetTextFiles(),
		}
		agent, err = pgstatstatements.New(params, l)

	case inventoryv1.AgentType_AGENT_TYPE_QAN_POSTGRESQL_PGSTATMONITOR_AGENT:
		params := &pgstatmonitor.Params{
			DSN:                    dsn,
			AgentID:                agentID,
			MaxQueryLength:         builtinAgent.MaxQueryLength,
			TextFiles:              builtinAgent.GetTextFiles(),
			DisableCommentsParsing: builtinAgent.DisableCommentsParsing,
			DisableQueryExamples:   builtinAgent.DisableQueryExamples,
		}
		agent, err = pgstatmonitor.New(params, l)

	case inventoryv1.AgentType_AGENT_TYPE_RTA_MONGODB_AGENT:
		params := &mongorta.Params{
			DSN:             dsn,
			AgentID:         agentID,
			ServiceID:       builtinAgent.ServiceId,
			ServiceName:     builtinAgent.ServiceName,
			CollectInterval: builtinAgent.RtaOptions.GetCollectInterval().AsDuration(),
		}
		agent, err = mongorta.New(params, l)

	case type_TEST_NOOP:
		agent = noop.New()

	default:
		err = fmt.Errorf("unhandled agent type %[1]s (%[1]d)", builtinAgent.Type)
	}

	if err != nil {
		cancel()
		return err
	}

	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), agent.Run)

	go func() {
		qanDeliveryRequests := make(chan *agentv1.QANCollectRequest, qanRequestsBufferSize)
		qanDeliveryDone := make(chan struct{})
		go func() {
			defer close(qanDeliveryDone)
			s.deliverQANRequests(ctx, l, agentID, qanDeliveryRequests)
		}()
		defer close(done)
		defer func() {
			close(qanDeliveryRequests)
			<-qanDeliveryDone
		}()

		rtaBucketLastCollectTime := timestamppb.New(time.Now()).AsTime()

		for change := range agent.Changes() {
			if change.Status != inventoryv1.AgentStatus_AGENT_STATUS_UNSPECIFIED {
				s.storeLastStatus(agentID, change.Status)
				l.Infof("Sending status: %s.", change.Status)
				s.changes <- &agentv1.StateChangedRequest{
					AgentId: agentID,
					Status:  change.Status,
				}
			}
			if change.MetricsBucket != nil {
				request := &agentv1.QANCollectRequest{
					MetricsBucket: change.MetricsBucket,
				}
				select {
				case qanDeliveryRequests <- request:
				case <-ctx.Done():
					return
				}
			}

			if len(change.RTAQueriesBucket) != 0 {
				// It may appear that buckets in channel are not in order of their collection.
				// This may happen because of some bucket is huge and takes a lot of time to process,
				// so the next one is already collected and sent to channel.
				// We check that collect time of the next bucket is not earlier than the prev one.
				// See MongoDBRTA.collectCurrentOps() for details.
				currentBucketCollectTime := change.RTAQueriesBucket[0].QueryCollectTime.AsTime()
				if rtaBucketLastCollectTime.After(currentBucketCollectTime) {
					continue
				}

				l.Infof("Sending %d RTA queries buckets.", len(change.RTAQueriesBucket))

				rtaBucketLastCollectTime = currentBucketCollectTime

				s.rtaRequests <- &rtav1.CollectRequest{
					Queries: change.RTAQueriesBucket,
				}
			}
		}
	}()

	//nolint:forcetypeassert
	s.builtinAgents[agentID] = &builtinAgentInfo{
		cancel:         cancel,
		done:           done,
		requestedState: proto.Clone(builtinAgent).(*agentv1.SetStateRequest_BuiltinAgent),
		describe:       agent.Describe,
		collect:        agent.Collect,
		logStore:       logStore,
	}
	return nil
}

func (s *Supervisor) deliverQANRequests(ctx context.Context, l *logrus.Entry, agentID string, requests <-chan *agentv1.QANCollectRequest) {
	var qanDeliveryOffset time.Duration
	qanDeliveryOffsetAssigned := false
	var releaseQANDeliveryOffset func()

	for request := range requests {
		if !qanDeliveryOffsetAssigned {
			var offset time.Duration
			offset, releaseQANDeliveryOffset = agents.RandomMinuteOffset(agentID)
			qanDeliveryOffset = offset
			qanDeliveryOffsetAssigned = true
		}

		delay := agents.DelayUntilOffset(time.Now(), time.Minute, qanDeliveryOffset)
		s.sendQANRequest(ctx, l, request, delay)
	}

	if releaseQANDeliveryOffset != nil {
		releaseQANDeliveryOffset()
	}
}

func (s *Supervisor) sendQANRequest(ctx context.Context, l *logrus.Entry, request *agentv1.QANCollectRequest, delay time.Duration) bool {
	if delay > 0 {
		l.Debugf("Scheduling QAN delivery in %s.", delay)
		t := time.NewTimer(delay)
		select {
		case <-t.C:
		case <-ctx.Done():
			if !t.Stop() {
				select {
				case <-t.C:
				default:
				}
			}
			return false
		}
	}

	l.Infof("Sending %d metrics buckets.", len(request.MetricsBucket))
	select {
	case s.qanRequests <- request:
		return true
	case <-ctx.Done():
		return false
	}
}

// agentLogger write logs to Store so can get last N.
func (s *Supervisor) agentLogger(logStore *tailog.Store) *logrus.Logger {
	return &logrus.Logger{
		Out:          io.MultiWriter(os.Stderr, logStore),
		Hooks:        logrus.StandardLogger().Hooks,
		Formatter:    logrus.StandardLogger().Formatter,
		ReportCaller: logrus.StandardLogger().ReportCaller,
		Level:        logrus.StandardLogger().GetLevel(),
		ExitFunc:     logrus.StandardLogger().ExitFunc,
	}
}

// processParams makes *process.Params from SetStateRequest parameters and other data.
func (s *Supervisor) processParams(agentID string, agentProcess *agentv1.SetStateRequest_AgentProcess, port uint16) (*process.Params, error) {
	var processParams process.Params
	processParams.Type = agentProcess.Type

	cfg := s.cfg.Get()
	templateParams := map[string]any{
		"listen_port": port,
	}
	switch agentProcess.Type {
	case inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER:
		templateParams["paths_base"] = cfg.Paths.PathsBase
		processParams.Path = cfg.Paths.NodeExporter
	case inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER:
		templateParams["paths_base"] = cfg.Paths.PathsBase
		processParams.Path = cfg.Paths.MySQLdExporter
	case inventoryv1.AgentType_AGENT_TYPE_MONGODB_EXPORTER:
		processParams.Path = cfg.Paths.MongoDBExporter
	case inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER:
		templateParams["paths_base"] = cfg.Paths.PathsBase
		processParams.Path = cfg.Paths.PostgresExporter
	case inventoryv1.AgentType_AGENT_TYPE_PROXYSQL_EXPORTER:
		processParams.Path = cfg.Paths.ProxySQLExporter
	case inventoryv1.AgentType_AGENT_TYPE_RDS_EXPORTER:
		processParams.Path = cfg.Paths.RDSExporter
	case inventoryv1.AgentType_AGENT_TYPE_AZURE_DATABASE_EXPORTER:
		processParams.Path = cfg.Paths.AzureExporter
	case inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER:
		templateParams["paths_base"] = cfg.Paths.PathsBase
		processParams.Path = cfg.Paths.ValkeyExporter
	case type_TEST_SLEEP:
		processParams.Path = "sleep"
	case inventoryv1.AgentType_AGENT_TYPE_VM_AGENT:
		templateParams["server_insecure"] = cfg.Server.InsecureTLS
		templateParams["server_url"] = "https://" + cfg.Server.Address
		if cfg.Server.WithoutTLS {
			templateParams["server_url"] = "http://" + cfg.Server.Address
		}
		templateParams["server_password"] = cfg.Server.Password
		templateParams["server_username"] = cfg.Server.Username
		templateParams["tmp_dir"] = cfg.Paths.TempDir
		processParams.Path = cfg.Paths.VMAgent
	case inventoryv1.AgentType_AGENT_TYPE_NOMAD_AGENT:
		templateParams["server_host"] = cfg.Server.URL().Hostname()
		templateParams["nomad_data_dir"] = cfg.Paths.NomadDataDir
		processParams.Path = cfg.Paths.Nomad
		processParams.Env = append(processParams.Env, os.Environ()...)
	default:
		return nil, fmt.Errorf("unhandled agent type %[1]s (%[1]d)", agentProcess.Type)
	}

	if processParams.Path == "" {
		return nil, fmt.Errorf("no path for agent type %[1]s (%[1]d)", agentProcess.Type)
	}

	tr := &templates.TemplateRenderer{
		TextFiles:          agentProcess.TextFiles,
		TemplateLeftDelim:  agentProcess.TemplateLeftDelim,
		TemplateRightDelim: agentProcess.TemplateRightDelim,
		TempDir:            filepath.Join(cfg.Paths.TempDir, trimPrefix(agentProcess.Type.String()), agentID),
	}

	processParams.TemplateRenderer = tr
	processParams.TemplateParams = templateParams

	templateParams, err := tr.RenderFiles(templateParams)
	if err != nil {
		return nil, err
	}

	processParams.Args = make([]string, len(agentProcess.Args))
	for i, e := range agentProcess.Args {
		b, err := tr.RenderTemplate("args", e, templateParams)
		if err != nil {
			return nil, err
		}
		processParams.Args[i] = string(b)
	}

	if agentProcess.Type == inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER && cfg.ProcMountsPath != "" {
		processParams.Args = append(processParams.Args, "--collector.filesystem.proc-mounts-path="+cfg.ProcMountsPath)
	}

	env := make([]string, len(agentProcess.Env))
	for i, e := range agentProcess.Env {
		b, err := tr.RenderTemplate("env", e, templateParams)
		if err != nil {
			return nil, err
		}
		env[i] = string(b)
	}
	processParams.Env = append(processParams.Env, env...)

	for _, varName := range agentProcess.EnvVariableNames {
		value, exists := os.LookupEnv(varName)
		if !exists {
			s.l.Warnf("Environment variable %s not found in pmm-agent environment for agent %s", varName, agentID)
			continue
		}

		processParams.Env = append(processParams.Env, fmt.Sprintf("%s=%s", varName, value))
		s.l.Debugf("Resolved environment variable %s for agent %s", varName, agentID)
	}

	return &processParams, nil
}

func (s *Supervisor) version(agentType inventoryv1.AgentType, path string) (string, error) {
	switch agentType {
	case inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, nodeExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, mysqldExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_MONGODB_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, mongodbExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_POSTGRES_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, postgresExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_PROXYSQL_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, proxysqlExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_RDS_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, rdsExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_AZURE_DATABASE_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, azureMetricsExporterRegexp, "--version")
	case inventoryv1.AgentType_AGENT_TYPE_VALKEY_EXPORTER:
		return s.agentVersioner.BinaryVersion(path, 0, valkeyExporterRegexp, "--version")
	default:
		return "", nil
	}
}

// stopAll stops all agents.
func (s *Supervisor) stopAll() {
	s.rw.Lock()
	defer s.rw.Unlock()

	s.setAgentProcesses(nil)
	s.setBuiltinAgents(nil)

	s.l.Infof("Done.")
	close(s.qanRequests)
	close(s.rtaRequests)
	close(s.changes)
}

// Describe implements prometheus.Collector.
func (s *Supervisor) Describe(ch chan<- *prometheus.Desc) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	for _, agent := range s.builtinAgents {
		agent.describe(ch)
	}
}

// Collect implement prometheus.Collector.
func (s *Supervisor) Collect(ch chan<- prometheus.Metric) {
	s.rw.RLock()
	defer s.rw.RUnlock()

	for _, agent := range s.builtinAgents {
		agent.collect(ch)
	}
}

// trimPrefix converts AgentType to lowercase and removes "agent_type_" prefix from it.
func trimPrefix(s string) string {
	return strings.TrimPrefix(strings.ToLower(s), "agent_type_")
}

// check interfaces.
var (
	_ prometheus.Collector = (*Supervisor)(nil)
)
