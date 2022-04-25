// pmm-agent
// Copyright 2019 Percona LLC
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
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"

	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/percona/pmm-agent/agents"
	"github.com/percona/pmm-agent/agents/mongodb"
	"github.com/percona/pmm-agent/agents/mysql/perfschema"
	"github.com/percona/pmm-agent/agents/mysql/slowlog"
	"github.com/percona/pmm-agent/agents/noop"
	"github.com/percona/pmm-agent/agents/postgres/pgstatmonitor"
	"github.com/percona/pmm-agent/agents/postgres/pgstatstatements"
	"github.com/percona/pmm-agent/agents/process"
	"github.com/percona/pmm-agent/config"
	"github.com/percona/pmm-agent/utils/templates"
)

// Supervisor manages all Agents, both processes and built-in.
type Supervisor struct {
	ctx           context.Context
	paths         *config.Paths
	serverCfg     *config.Server
	portsRegistry *portsRegistry
	changes       chan *agentpb.StateChangedRequest
	qanRequests   chan *agentpb.QANCollectRequest
	l             *logrus.Entry

	rw             sync.RWMutex
	agentProcesses map[string]*agentProcessInfo
	builtinAgents  map[string]*builtinAgentInfo

	arw          sync.RWMutex
	lastStatuses map[string]inventorypb.AgentStatus
}

// agentProcessInfo describes Agent process.
type agentProcessInfo struct {
	cancel          func()          // to cancel Process.Run(ctx)
	done            <-chan struct{} // closes when Process.Changes() channel closes
	requestedState  *agentpb.SetStateRequest_AgentProcess
	listenPort      uint16
	processExecPath string
}

// builtinAgentInfo describes built-in Agent.
type builtinAgentInfo struct {
	cancel         func()          // to cancel AgentType.Run(ctx)
	done           <-chan struct{} // closes when AgentType.Changes() channel closes
	requestedState *agentpb.SetStateRequest_BuiltinAgent
	describe       func(chan<- *prometheus.Desc)  // agent's func to describe Prometheus metrics
	collect        func(chan<- prometheus.Metric) // agent's func to provide Prometheus metrics
}

// NewSupervisor creates new Supervisor object.
//
// Supervisor is gracefully stopped when context passed to NewSupervisor is canceled.
// Changes of Agent statuses are reported via Changes() channel which must be read until it is closed.
// QAN data is sent to QANRequests() channel which must be read until it is closed.
func NewSupervisor(ctx context.Context, paths *config.Paths, ports *config.Ports, server *config.Server) *Supervisor {
	supervisor := &Supervisor{
		ctx:           ctx,
		paths:         paths,
		serverCfg:     server,
		portsRegistry: newPortsRegistry(ports.Min, ports.Max, nil),
		changes:       make(chan *agentpb.StateChangedRequest, 10),
		qanRequests:   make(chan *agentpb.QANCollectRequest, 10),
		l:             logrus.WithField("component", "supervisor"),

		agentProcesses: make(map[string]*agentProcessInfo),
		builtinAgents:  make(map[string]*builtinAgentInfo),
		lastStatuses:   make(map[string]inventorypb.AgentStatus),
	}

	go func() {
		<-ctx.Done()
		supervisor.stopAll()
	}()

	return supervisor
}

// AgentsList returns info for all Agents managed by this supervisor.
func (s *Supervisor) AgentsList() []*agentlocalpb.AgentInfo {
	s.rw.RLock()
	defer s.rw.RUnlock()
	s.arw.RLock()
	defer s.arw.RUnlock()

	res := make([]*agentlocalpb.AgentInfo, 0, len(s.agentProcesses)+len(s.builtinAgents))

	for id, agent := range s.agentProcesses {
		info := &agentlocalpb.AgentInfo{
			AgentId:         id,
			AgentType:       agent.requestedState.Type,
			Status:          s.lastStatuses[id],
			ListenPort:      uint32(agent.listenPort),
			ProcessExecPath: agent.processExecPath,
		}
		res = append(res, info)
	}

	for id, agent := range s.builtinAgents {
		info := &agentlocalpb.AgentInfo{
			AgentId:   id,
			AgentType: agent.requestedState.Type,
			Status:    s.lastStatuses[id],
		}
		res = append(res, info)
	}

	sort.Slice(res, func(i, j int) bool { return res[i].AgentId < res[j].AgentId })
	return res
}

// Changes returns channel with Agent's state changes.
func (s *Supervisor) Changes() <-chan *agentpb.StateChangedRequest {
	return s.changes
}

// QANRequests returns channel with Agent's QAN Collect requests.
func (s *Supervisor) QANRequests() <-chan *agentpb.QANCollectRequest {
	return s.qanRequests
}

// SetState starts or updates all agents placed in args and stops all agents not placed in args, but already run.
func (s *Supervisor) SetState(state *agentpb.SetStateRequest) {
	// do not process SetState requests concurrently for internal state consistency and implementation simplicity
	s.rw.Lock()
	defer s.rw.Unlock()

	// check if we waited for lock too long
	if err := s.ctx.Err(); err != nil {
		s.l.Errorf("Ignoring SetState: %s.", err)
		return
	}

	s.setAgentProcesses(state.AgentProcesses)
	s.setBuiltinAgents(state.BuiltinAgents)
}

func (s *Supervisor) storeLastStatus(agentID string, status inventorypb.AgentStatus) {
	s.arw.Lock()
	defer s.arw.Unlock()

	switch status {
	case inventorypb.AgentStatus_DONE:
		delete(s.lastStatuses, agentID)
	default:
		s.lastStatuses[agentID] = status
	}
}

// setAgentProcesses starts/restarts/stops Agent processes.
// Must be called with s.rw held for writing.
func (s *Supervisor) setAgentProcesses(agentProcesses map[string]*agentpb.SetStateRequest_AgentProcess) {
	existingParams := make(map[string]agentpb.AgentParams)
	for id, p := range s.agentProcesses {
		existingParams[id] = p.requestedState
	}
	newParams := make(map[string]agentpb.AgentParams)
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

		if err := s.portsRegistry.Release(agent.listenPort); err != nil {
			s.l.Errorf("Failed to release port: %s.", err)
		}

		delete(s.agentProcesses, agentID)
	}

	// restart while preserving port
	for _, agentID := range toRestart {
		agent := s.agentProcesses[agentID]
		agent.cancel()
		<-agent.done

		if err := s.startProcess(agentID, agentProcesses[agentID], agent.listenPort); err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}

	// start new agents
	for _, agentID := range toStart {
		port, err := s.portsRegistry.Reserve()
		if err != nil {
			s.l.Errorf("Failed to reserve port: %s.", err)
			// TODO report that error to server
			continue
		}

		if err := s.startProcess(agentID, agentProcesses[agentID], port); err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}
}

// setBuiltinAgents starts/restarts/stops built-in Agents.
// Must be called with s.rw held for writing.
func (s *Supervisor) setBuiltinAgents(builtinAgents map[string]*agentpb.SetStateRequest_BuiltinAgent) {
	existingParams := make(map[string]agentpb.AgentParams)
	for id, agent := range s.builtinAgents {
		existingParams[id] = agent.requestedState
	}
	newParams := make(map[string]agentpb.AgentParams)
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
	}

	// restart
	for _, agentID := range toRestart {
		agent := s.builtinAgents[agentID]
		agent.cancel()
		<-agent.done

		if err := s.startBuiltin(agentID, builtinAgents[agentID]); err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}

	// start new agents
	for _, agentID := range toStart {
		if err := s.startBuiltin(agentID, builtinAgents[agentID]); err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
		}
	}
}

// filter extracts IDs of the Agents that should be started, restarted with new parameters, or stopped,
// and filters out IDs of the Agents that should not be changed.
func filter(existing, new map[string]agentpb.AgentParams) (toStart, toRestart, toStop []string) {
	// existing agents not present in the new requested state should be stopped
	for existingID := range existing {
		if new[existingID] == nil {
			toStop = append(toStop, existingID)
		}
	}

	// detect new and changed agents
	for newID, newParams := range new {
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
	return
}

//nolint:golint
const (
	type_TEST_SLEEP inventorypb.AgentType = 998 // process
	type_TEST_NOOP  inventorypb.AgentType = 999 // built-in
)

// startProcess starts Agent's process.
// Must be called with s.rw held for writing.
func (s *Supervisor) startProcess(agentID string, agentProcess *agentpb.SetStateRequest_AgentProcess, port uint16) error {
	processParams, err := s.processParams(agentID, agentProcess, port)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(s.ctx)
	agentType := strings.ToLower(agentProcess.Type.String())
	l := logrus.WithFields(logrus.Fields{
		"component": "agent-process",
		"agentID":   agentID,
		"type":      agentType,
	})
	l.Debugf("Starting: %s.", processParams)

	process := process.New(processParams, agentProcess.RedactWords, l)
	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), process.Run)

	done := make(chan struct{})
	go func() {
		for status := range process.Changes() {
			s.storeLastStatus(agentID, status)
			l.Infof("Sending status: %s (port %d).", status, port)
			s.changes <- &agentpb.StateChangedRequest{
				AgentId:         agentID,
				Status:          status,
				ListenPort:      uint32(port),
				ProcessExecPath: processParams.Path,
			}
		}
		close(done)
	}()

	s.agentProcesses[agentID] = &agentProcessInfo{
		cancel:          cancel,
		done:            done,
		requestedState:  proto.Clone(agentProcess).(*agentpb.SetStateRequest_AgentProcess),
		listenPort:      port,
		processExecPath: processParams.Path,
	}
	return nil
}

// startBuiltin starts built-in Agent.
// Must be called with s.rw held for writing.
func (s *Supervisor) startBuiltin(agentID string, builtinAgent *agentpb.SetStateRequest_BuiltinAgent) error {
	ctx, cancel := context.WithCancel(s.ctx)
	agentType := strings.ToLower(builtinAgent.Type.String())
	l := logrus.WithFields(logrus.Fields{
		"component": "agent-builtin",
		"agentID":   agentID,
		"type":      agentType,
	})

	done := make(chan struct{})
	var agent agents.BuiltinAgent
	var err error

	var dsn string
	if builtinAgent.TextFiles != nil {
		tempDir := filepath.Join(s.paths.TempDir, strings.ToLower(builtinAgent.Type.String()), agentID)
		dsn, err = templates.RenderDSN(builtinAgent.Dsn, builtinAgent.TextFiles, tempDir)
		if err != nil {
			cancel()
			return err
		}
	} else {
		dsn = builtinAgent.Dsn
	}

	switch builtinAgent.Type {
	case inventorypb.AgentType_QAN_MYSQL_PERFSCHEMA_AGENT:
		params := &perfschema.Params{
			DSN:                  dsn,
			AgentID:              agentID,
			DisableQueryExamples: builtinAgent.DisableQueryExamples,
			TextFiles:            builtinAgent.GetTextFiles(),
			TLSSkipVerify:        builtinAgent.TlsSkipVerify,
		}
		agent, err = perfschema.New(params, l)

	case inventorypb.AgentType_QAN_MONGODB_PROFILER_AGENT:
		params := &mongodb.Params{
			DSN:     dsn,
			AgentID: agentID,
		}
		agent, err = mongodb.New(params, l)

	case inventorypb.AgentType_QAN_MYSQL_SLOWLOG_AGENT:
		params := &slowlog.Params{
			DSN:                  dsn,
			AgentID:              agentID,
			SlowLogFilePrefix:    s.paths.SlowLogFilePrefix,
			DisableQueryExamples: builtinAgent.DisableQueryExamples,
			MaxSlowlogFileSize:   builtinAgent.MaxQueryLogSize,
			TextFiles:            builtinAgent.GetTextFiles(),
			TLSSkipVerify:        builtinAgent.TlsSkipVerify,
		}
		agent, err = slowlog.New(params, l)

	case inventorypb.AgentType_QAN_POSTGRESQL_PGSTATEMENTS_AGENT:
		params := &pgstatstatements.Params{
			DSN:       dsn,
			AgentID:   agentID,
			TextFiles: builtinAgent.GetTextFiles(),
		}
		agent, err = pgstatstatements.New(params, l)

	case inventorypb.AgentType_QAN_POSTGRESQL_PGSTATMONITOR_AGENT:
		params := &pgstatmonitor.Params{
			DSN:                  dsn,
			AgentID:              agentID,
			TextFiles:            builtinAgent.GetTextFiles(),
			DisableQueryExamples: builtinAgent.DisableQueryExamples,
		}
		agent, err = pgstatmonitor.New(params, l)

	case type_TEST_NOOP:
		agent = noop.New()

	default:
		err = errors.Errorf("unhandled agent type %[1]s (%[1]d).", builtinAgent.Type)
	}

	if err != nil {
		cancel()
		return err
	}

	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), agent.Run)

	go func() {
		for change := range agent.Changes() {
			if change.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
				s.storeLastStatus(agentID, change.Status)
				l.Infof("Sending status: %s.", change.Status)
				s.changes <- &agentpb.StateChangedRequest{
					AgentId: agentID,
					Status:  change.Status,
				}
			}
			if change.MetricsBucket != nil {
				l.Infof("Sending %d buckets.", len(change.MetricsBucket))
				s.qanRequests <- &agentpb.QANCollectRequest{
					MetricsBucket: change.MetricsBucket,
				}
			}
		}
		close(done)
	}()

	s.builtinAgents[agentID] = &builtinAgentInfo{
		cancel:         cancel,
		done:           done,
		requestedState: proto.Clone(builtinAgent).(*agentpb.SetStateRequest_BuiltinAgent),
		describe:       agent.Describe,
		collect:        agent.Collect,
	}
	return nil
}

// "_" at the begginging is reserved for possible extensions
var textFileRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`) //nolint:gochecknoglobals

// processParams makes *process.Params from SetStateRequest parameters and other data.
func (s *Supervisor) processParams(agentID string, agentProcess *agentpb.SetStateRequest_AgentProcess, port uint16) (*process.Params, error) {
	var processParams process.Params
	processParams.Type = agentProcess.Type

	templateParams := map[string]interface{}{
		"listen_port": port,
	}
	switch agentProcess.Type {
	case inventorypb.AgentType_NODE_EXPORTER:
		templateParams["paths_base"] = s.paths.PathsBase
		processParams.Path = s.paths.NodeExporter
	case inventorypb.AgentType_MYSQLD_EXPORTER:
		templateParams["paths_base"] = s.paths.PathsBase
		processParams.Path = s.paths.MySQLdExporter
	case inventorypb.AgentType_MONGODB_EXPORTER:
		processParams.Path = s.paths.MongoDBExporter
	case inventorypb.AgentType_POSTGRES_EXPORTER:
		templateParams["paths_base"] = s.paths.PathsBase
		processParams.Path = s.paths.PostgresExporter
	case inventorypb.AgentType_PROXYSQL_EXPORTER:
		processParams.Path = s.paths.ProxySQLExporter
	case inventorypb.AgentType_RDS_EXPORTER:
		processParams.Path = s.paths.RDSExporter
	case inventorypb.AgentType_AZURE_DATABASE_EXPORTER:
		processParams.Path = s.paths.AzureExporter
	case type_TEST_SLEEP:
		processParams.Path = "sleep"
	case inventorypb.AgentType_VM_AGENT:
		// add template params for vmagent.
		templateParams["server_insecure"] = s.serverCfg.InsecureTLS
		templateParams["server_url"] = fmt.Sprintf("https://%s", s.serverCfg.Address)
		if s.serverCfg.WithoutTLS {
			templateParams["server_url"] = fmt.Sprintf("http://%s", s.serverCfg.Address)
		}
		templateParams["server_password"] = s.serverCfg.Password
		templateParams["server_username"] = s.serverCfg.Username
		templateParams["tmp_dir"] = s.paths.TempDir
		processParams.Path = s.paths.VMAgent
	default:
		return nil, errors.Errorf("unhandled agent type %[1]s (%[1]d).", agentProcess.Type)
	}

	if processParams.Path == "" {
		return nil, errors.Errorf("no path for agent type %[1]s (%[1]d).", agentProcess.Type)
	}

	tr := &templates.TemplateRenderer{
		TextFiles:          agentProcess.TextFiles,
		TemplateLeftDelim:  agentProcess.TemplateLeftDelim,
		TemplateRightDelim: agentProcess.TemplateRightDelim,
		TempDir:            filepath.Join(s.paths.TempDir, strings.ToLower(agentProcess.Type.String()), agentID),
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

	processParams.Env = make([]string, len(agentProcess.Env))
	for i, e := range agentProcess.Env {
		b, err := tr.RenderTemplate("env", e, templateParams)
		if err != nil {
			return nil, err
		}
		processParams.Env[i] = string(b)
	}

	return &processParams, nil
}

// stopAll stops all agents.
func (s *Supervisor) stopAll() {
	s.rw.Lock()
	defer s.rw.Unlock()

	s.setAgentProcesses(nil)
	s.setBuiltinAgents(nil)

	s.l.Infof("Done.")
	close(s.qanRequests)
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

// check interfaces
var (
	_ prometheus.Collector = (*Supervisor)(nil)
)
