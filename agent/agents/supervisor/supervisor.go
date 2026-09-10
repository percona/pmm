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
	"sync/atomic"
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

	// Budget for waiting on stopped Agents' status forwarders in one call.
	//
	// Those waits reach the server - a stopping Agent's last statuses go out through Changes()
	// and the client waits for each to be acknowledged - so left unbounded they took the whole
	// supervisor down with a wedged connection, s.rw included, and with it every SetState and
	// the local status API. Giving up risks only reporting a stopped Agent's last statuses out
	// of order: an abandoned Agent keeps what it owns until it does stop (see
	// releaseAgentResources), and its replacement starts on a port of its own. It also leaves a
	// forwarder that stopAll must not close the channels under, hence the count of them below.
	// The budget covers the whole call rather than each Agent so that a batch of them cannot
	// hold s.rw for N times as long; they are all canceled before any of them is waited on, so
	// they spend it concurrently rather than one after another. It is sized well above
	// process.killT, so a normal SIGTERM/SIGKILL stop never eats into it. See PMM-15431.
	agentsStopTimeout = 60 * time.Second
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
	// instances names the current instance of each Agent ID. An Agent abandoned by
	// waitAgentStopped keeps running under an ID its replacement now owns, and its late
	// statuses would otherwise speak for that replacement - a DONE among them deletes the
	// replacement's status outright, leaving a running Agent reported as stopped with nothing
	// to correct it. See PMM-15431.
	instances    map[string]uint64
	nextInstance uint64

	// forwarders counts the running Agent status forwarders, so that stopAll never closes the
	// channels they send to while one of them is still able to send. See PMM-15431.
	forwarders atomic.Int64

	// for unit tests only
	agentsStopTimeout time.Duration
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
// Changes of Agent statuses are reported via Changes() channel, QAN data via QANRequests(), and RTA
// data via RTARequests(). All three must be read, under the same context that stops the Supervisor:
// they are closed once it has stopped, but an Agent whose status forwarder outlives the stop budget
// leaves them open rather than risk a send on a closed channel (see stopAll), so waiting only for
// the close can wait forever.
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
		instances:      make(map[string]uint64),

		agentsStopTimeout: agentsStopTimeout,
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

	deadline := time.Now().Add(s.agentsStopTimeout)
	s.setAgentProcesses(state.AgentProcesses, deadline)
	s.setBuiltinAgents(state.BuiltinAgents, deadline)
}

// RestartAgents restarts all existing agents.
func (s *Supervisor) RestartAgents() {
	s.rw.Lock()
	defer s.rw.Unlock()

	deadline := time.Now().Add(s.agentsStopTimeout)

	// Iterate a snapshot of the keys: the give-up branch below deletes the entry and
	// tryStartProcess re-inserts the same key, and an entry created during a range may be
	// produced again by that range.
	ids := make([]string, 0, len(s.agentProcesses))
	for id := range s.agentProcesses {
		ids = append(ids, id)
	}

	// Cancel them all first - see setAgentProcesses.
	for _, id := range ids {
		s.agentProcesses[id].cancel()
	}

	for _, id := range ids {
		agent := s.agentProcesses[id]
		port := agent.listenPort
		if !s.waitAgentStopped(id, agent.done, deadline) {
			// See the same branch in setAgentProcesses.
			delete(s.agentProcesses, id)
			s.releaseAgentResources(id, agent.done, port, "")
			port = 0
		}

		err := s.tryStartProcess(id, agent.requestedState, port)
		if err != nil {
			s.l.Errorf("Failed to restart Agent: %s.", err)
		}
	}

	for _, agent := range s.builtinAgents {
		agent.cancel()
	}

	for id, agent := range s.builtinAgents {
		s.waitAgentStopped(id, agent.done, deadline)

		err := s.startBuiltin(id, agent.requestedState)
		if err != nil {
			s.l.Errorf("Failed to restart Agent: %s.", err)
		}
	}
}

// waitAgentStopped waits for a canceled Agent's status forwarder to finish, until deadline. It
// reports whether the Agent stopped; an abandoned one is still running, so everything that assumes
// it is gone has to wait for it instead - see releaseAgentResources.
func (s *Supervisor) waitAgentStopped(agentID string, done <-chan struct{}, deadline time.Time) bool {
	// Take the answer if it is already there. Once the budget is spent both cases below are
	// ready, and select would pick between them at random - reporting a timeout for every
	// other Agent that had in fact stopped cleanly.
	select {
	case <-done:
		return true
	default:
	}

	t := time.NewTimer(time.Until(deadline))
	defer t.Stop()

	select {
	case <-done:
		return true
	case <-t.C:
		s.l.Errorf("Agent %s did not report itself stopped, proceeding without it.", agentID)
		return false
	}
}

// releaseAgentResources gives back what an Agent owned: its port reservation, and its temporary
// directory unless agentTmp is empty. A built-in Agent has no port, hence port 0.
//
// Both need the Agent actually gone, which one that outlived the stop budget is not, so it is
// waited for in a goroutine of its own instead - off s.rw, and off the call that gave up on it.
// Releasing a port a live exporter still listens on fails and keeps the reservation for good,
// since the Agent that held it is already forgotten and nothing is left to retry it, and clearing
// the directory takes files out from under a running Agent. Only the port is given back that way:
// by then the ID may belong to a replacement, and the directory is the replacement's. See
// PMM-15431.
func (s *Supervisor) releaseAgentResources(agentID string, done <-chan struct{}, port uint16, agentTmp string) {
	select {
	case <-done:
		s.releasePortAndTempDir(agentID, port, agentTmp)
	default:
		s.l.Warnf("Agent %s is still running, freeing what it owns once it stops.", agentID)
		go func() {
			select {
			case <-done:
				// Deliberately not agentTmp: an Agent with this ID may have been
				// re-created while this one was stopping, and removing it now
				// would take the replacement's TLS certificates and text files
				// with it. A directory left behind is cleaned on the next start
				// (see cleanupTmp), and reused as-is if the ID comes back before
				// that.
				s.releasePortAndTempDir(agentID, port, "")
			case <-s.ctx.Done():
				// pmm-agent is on its way out: the OS takes the port back, and
				// the temporary directory is cleaned on the next start.
			}
		}()
	}
}

func (s *Supervisor) releasePortAndTempDir(agentID string, port uint16, agentTmp string) {
	if port != 0 {
		err := s.portsRegistry.Release(port)
		if err != nil {
			s.l.Errorf("Failed to release port %d of Agent %s: %s.", port, agentID, err)
		}
	}

	if agentTmp == "" {
		return
	}

	err := os.RemoveAll(agentTmp)
	if err != nil {
		s.l.Warnf("Failed to cleanup directory '%s': %s", agentTmp, err.Error())
	}
}

// forward sends v to ch, and gives up only if the Supervisor is stopping and ch has no room.
// Nothing drains these channels once the client is gone, so parking on a full one leaks the
// forwarder and keeps stopAll from closing them; preferring the send keeps the final statuses of a
// normal shutdown, which the client is still draining. See PMM-15431.
func forward[T any](ctx context.Context, ch chan<- T, v T) bool {
	select {
	case ch <- v:
		return true
	default:
	}

	select {
	case ch <- v:
		return true
	case <-ctx.Done():
		return false
	}
}

// startInstance makes a new instance of agentID the current one and returns its token.
// Must be called with s.rw held for writing.
func (s *Supervisor) startInstance(agentID string) uint64 {
	s.arw.Lock()
	defer s.arw.Unlock()

	s.nextInstance++
	s.instances[agentID] = s.nextInstance

	return s.nextInstance
}

// storeLastStatus records status against agentID and reports whether it did. It does not, and the
// caller must report nothing either, once instance is no longer the current one for that ID: the
// checking and the storing share a lock precisely so that a replacement cannot slip in between.
func (s *Supervisor) storeLastStatus(agentID string, instance uint64, status inventoryv1.AgentStatus) bool {
	s.arw.Lock()
	defer s.arw.Unlock()

	if s.instances[agentID] != instance {
		return false
	}

	if status == inventoryv1.AgentStatus_AGENT_STATUS_DONE {
		delete(s.lastStatuses, agentID)
		delete(s.instances, agentID)

		return true
	}

	s.lastStatuses[agentID] = status

	return true
}

// setAgentProcesses starts/restarts/stops Agent processes.
// Must be called with s.rw held for writing.
func (s *Supervisor) setAgentProcesses(agentProcesses map[string]*agentv1.SetStateRequest_AgentProcess, deadline time.Time) {
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

	// Cancel them all first, so that they stop concurrently and the budget spent on the first
	// one to hang is time the rest are already using. Canceling one at a time left every Agent
	// after a hung one with no budget at all, so Agents that stop normally were abandoned too -
	// and an abandoned one being restarted gives up its port for a new one and overlaps its
	// replacement. See PMM-15431.
	for _, agentID := range toStop {
		s.agentProcesses[agentID].cancel()
	}
	for _, agentID := range toRestart {
		s.agentProcesses[agentID].cancel()
	}

	// stop first to avoid extra load
	for _, agentID := range toStop {
		agent := s.agentProcesses[agentID]
		s.waitAgentStopped(agentID, agent.done, deadline)

		delete(s.agentProcesses, agentID)

		agentTmp := filepath.Join(s.cfg.Get().Paths.TempDir, trimPrefix(agent.requestedState.Type.String()), agentID)
		s.releaseAgentResources(agentID, agent.done, agent.listenPort, agentTmp)
	}

	// restart while preserving port
	for _, agentID := range toRestart {
		agent := s.agentProcesses[agentID]
		port := agent.listenPort
		if !s.waitAgentStopped(agentID, agent.done, deadline) {
			// It may still be listening, so let the replacement have a port of its
			// own and hand this one back once it is free. Forget the Agent either
			// way: it has been canceled, so if starting the replacement fails, a
			// later SetState has to treat it as gone rather than as still running.
			delete(s.agentProcesses, agentID)
			s.releaseAgentResources(agentID, agent.done, port, "")
			port = 0
		}

		err := s.tryStartProcess(agentID, agentProcesses[agentID], port)
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
func (s *Supervisor) setBuiltinAgents(builtinAgents map[string]*agentv1.SetStateRequest_BuiltinAgent, deadline time.Time) {
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

	// Cancel them all first - see setAgentProcesses.
	for _, agentID := range toStop {
		s.builtinAgents[agentID].cancel()
	}
	for _, agentID := range toRestart {
		s.builtinAgents[agentID].cancel()
	}

	// stop first to avoid extra load
	for _, agentID := range toStop {
		agent := s.builtinAgents[agentID]
		s.waitAgentStopped(agentID, agent.done, deadline)

		delete(s.builtinAgents, agentID)

		agentTmp := filepath.Join(s.cfg.Get().Paths.TempDir, trimPrefix(agent.requestedState.Type.String()), agentID)
		s.releaseAgentResources(agentID, agent.done, 0, agentTmp)
	}

	// restart
	for _, agentID := range toRestart {
		agent := s.builtinAgents[agentID]
		s.waitAgentStopped(agentID, agent.done, deadline)

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

const (
	typeTestSleep       inventoryv1.AgentType = 998 // process
	typeTestNoop        inventoryv1.AgentType = 999 // built-in
	processRetryCount   int                   = 3
	startProcessWaiting                       = 2 * time.Second
)

func (s *Supervisor) tryStartProcess(agentID string, agentProcess *agentv1.SetStateRequest_AgentProcess, port uint16) error {
	var err error
	for range processRetryCount {
		if port == 0 {
			var _port uint16
			_port, err = s.portsRegistry.Reserve()
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

	processWrapper := process.New(processParams, agentProcess.RedactWords, l)
	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), processWrapper.Run)

	version, err := s.version(agentProcess.Type, processParams.Path)
	if err != nil {
		l.Warnf("Cannot parse version for type %s", agentType)
	}

	done := make(chan struct{})
	instance := s.startInstance(agentID)
	s.forwarders.Add(1)
	go func() {
		defer func() {
			// Before done, so that an observed done implies this forwarder is out of
			// the count. Deferred so that giving up below still accounts for it.
			s.forwarders.Add(-1)
			close(done)
		}()

		for status := range processWrapper.Changes() {
			if !s.storeLastStatus(agentID, instance, status) {
				// Abandoned: a replacement owns this ID now.
				continue
			}
			l.Infof("Sending status: %s (port %d).", status, port)
			if !forward(s.ctx, s.changes, &agentv1.StateChangedRequest{
				AgentId:         agentID,
				Status:          status,
				ListenPort:      uint32(port),
				ProcessExecPath: processParams.Path,
				Version:         version,
			}) {
				return
			}
		}
	}()

	processInfo := &agentProcessInfo{ //nolint:forcetypeassert
		cancel:          cancel,
		done:            done,
		requestedState:  proto.Clone(agentProcess).(*agentv1.SetStateRequest_AgentProcess),
		listenPort:      port,
		processExecPath: processParams.Path,
		logStore:        logStore,
	}

	t := time.NewTimer(startProcessWaiting)
	defer t.Stop()
	select {
	case isInitialized := <-processWrapper.IsInitialized():
		if !isInitialized {
			// TODO: handle initialization error for nomad agent
			if agentProcess.Type == inventoryv1.AgentType_AGENT_TYPE_NOMAD_AGENT {
				s.handleNomadAgent(agentID, instance, processInfo, l)
				return nil
			}
			defer cancel()
			return processWrapper.GetError()
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
	instance uint64,
	processInfo *agentProcessInfo,
	l *logrus.Entry,
) {
	done := make(chan struct{})
	s.agentProcesses[agentID] = processInfo

	status := inventoryv1.AgentStatus_AGENT_STATUS_DONE
	s.storeLastStatus(agentID, instance, status)
	l.Warn("Cannot start Nomad Agent: cgroups are not writable.")
	l.Infof("Sending status: %s (port %d).", status, processInfo.listenPort)
	// Bounded: this runs with s.rw held for writing, so parking on a full channel would
	// block every later SetState, AgentsList and stopAll for good. See PMM-15431.
	select {
	case s.changes <- &agentv1.StateChangedRequest{
		AgentId:         agentID,
		Status:          status,
		ListenPort:      uint32(processInfo.listenPort),
		ProcessExecPath: processInfo.processExecPath,
	}:
	case <-s.ctx.Done():
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

	case typeTestNoop:
		agent = noop.New()

	default:
		err = fmt.Errorf("unhandled agent type %[1]s (%[1]d)", builtinAgent.Type)
	}

	if err != nil {
		cancel()
		return err
	}

	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), agent.Run)

	instance := s.startInstance(agentID)
	s.forwarders.Add(1)
	go func() {
		defer func() {
			// Before done, so that an observed done implies this forwarder is out of
			// the count. Deferred so that giving up below still accounts for it.
			s.forwarders.Add(-1)
			close(done)
		}()

		rtaBucketLastCollectTime := timestamppb.New(time.Now()).AsTime()

		for change := range agent.Changes() {
			if change.Status != inventoryv1.AgentStatus_AGENT_STATUS_UNSPECIFIED {
				if !s.storeLastStatus(agentID, instance, change.Status) {
					// Abandoned: a replacement owns this ID now.
					continue
				}
				l.Infof("Sending status: %s.", change.Status)
				if !forward(s.ctx, s.changes, &agentv1.StateChangedRequest{
					AgentId: agentID,
					Status:  change.Status,
				}) {
					return
				}
			}
			if change.MetricsBucket != nil {
				l.Infof("Sending %d metrics buckets.", len(change.MetricsBucket))
				if !forward(s.ctx, s.qanRequests, &agentv1.QANCollectRequest{
					MetricsBucket: change.MetricsBucket,
				}) {
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

				if !forward(s.ctx, s.rtaRequests, &rtav1.CollectRequest{
					Queries: change.RTAQueriesBucket,
				}) {
					return
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
	case typeTestSleep:
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

	deadline := time.Now().Add(s.agentsStopTimeout)
	s.setAgentProcesses(nil, deadline)
	s.setBuiltinAgents(nil, deadline)

	s.l.Infof("Done.")

	// Closing these while a forwarder can still send panics, and the waits above are bounded,
	// so a forwarder can outlive them. Leaving the channels open is the safe failure: their
	// only consumer selects on its own context rather than reading them until they close, and
	// the process is on its way out anyway. See PMM-15431.
	if n := s.forwarders.Load(); n != 0 {
		s.l.Errorf("%d Agent status forwarders are still running, leaving their channels open.", n)
		return
	}

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
