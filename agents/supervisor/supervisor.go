// pmm-agent
// Copyright (C) 2018 Percona LLC
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

// Package supervisor provides supervisor for running Agents.
package supervisor

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/percona/pmm/api/agentpb"
	inventorypb "github.com/percona/pmm/api/inventory"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/agents/builtin/mysql"
	"github.com/percona/pmm-agent/agents/builtin/noop"
	"github.com/percona/pmm-agent/agents/process"
	"github.com/percona/pmm-agent/config"
)

// Supervisor manages all Agents, both processes and built-in.
type Supervisor struct {
	ctx           context.Context
	paths         *config.Paths
	portsRegistry *portsRegistry
	changes       chan agentpb.StateChangedRequest
	qanRequests   chan agentpb.QANCollectRequest
	l             *logrus.Entry

	m              sync.Mutex
	agentProcesses map[string]*agentProcessInfo
	builtinAgents  map[string]*builtinAgentInfo
}

// agentProcessInfo describes Agent process.
type agentProcessInfo struct {
	cancel         func()        // to cancel Run(ctx)
	done           chan struct{} // closed when Run(ctx) exits
	requestedState *agentpb.SetStateRequest_AgentProcess
	listenPort     uint16
}

// builtinAgentInfo describes built-in Agent.
type builtinAgentInfo struct {
	cancel         func()        // to cancel Run(ctx)
	done           chan struct{} // closed when Run(ctx) exits
	requestedState *agentpb.SetStateRequest_BuiltinAgent
}

// NewSupervisor creates new Supervisor object.
func NewSupervisor(ctx context.Context, paths *config.Paths, ports *config.Ports) *Supervisor {
	supervisor := &Supervisor{
		ctx:           ctx,
		paths:         paths,
		portsRegistry: newPortsRegistry(ports.Min, ports.Max, nil),
		changes:       make(chan agentpb.StateChangedRequest, 10),
		qanRequests:   make(chan agentpb.QANCollectRequest, 10),
		l:             logrus.WithField("component", "supervisor"),

		agentProcesses: make(map[string]*agentProcessInfo),
		builtinAgents:  make(map[string]*builtinAgentInfo),
	}

	go func() {
		<-ctx.Done()
		supervisor.stopAll()
	}()

	return supervisor
}

// SetState starts or updates all agents placed in args and stops all agents not placed in args, but already run.
func (s *Supervisor) SetState(state *agentpb.SetStateRequest) {
	s.m.Lock()
	defer s.m.Unlock()

	if err := s.ctx.Err(); err != nil {
		s.l.Errorf("Ignoring SetState: %s.", err)
		return
	}

	s.setAgentProcesses(state.AgentProcesses)
	s.setBuiltinAgents(state.BuiltinAgents)
}

// setAgentProcesses starts/restarts/stops Agent processes.
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

	// We have to wait for processes to terminate before starting a new ones to reuse ports,
	// and to send all state updates.
	// If that place is slow, we can cancel them all in parallel, but then we still have to wait.

	// stop first to avoid extra load
	for _, agentID := range toStop {
		agent := s.agentProcesses[agentID]
		agent.cancel()
		<-agent.done // wait before releasing port

		if err := s.portsRegistry.Release(agent.listenPort); err != nil {
			s.l.Errorf("Failed to release port: %s.", err)
		}
		delete(s.agentProcesses, agentID)
	}

	// restart while preserving port
	for _, agentID := range toRestart {
		agent := s.agentProcesses[agentID]
		agent.cancel()
		<-agent.done // wait before reusing port

		agent, err := s.startProcess(agentID, agentProcesses[agentID], agent.listenPort)
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
			continue
		}
		s.agentProcesses[agentID] = agent
	}

	// start new agents
	for _, agentID := range toStart {
		port, err := s.portsRegistry.Reserve()
		if err != nil {
			s.l.Errorf("Failed to reserve port: %s.", err)
			// TODO report that error to server
			continue
		}

		agent, err := s.startProcess(agentID, agentProcesses[agentID], port)
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
			continue
		}
		s.agentProcesses[agentID] = agent
	}
}

// setBuiltinAgents starts/restarts/stops built-in Agents.
func (s *Supervisor) setBuiltinAgents(builtinAgents map[string]*agentpb.SetStateRequest_BuiltinAgent) {
	existingParams := make(map[string]agentpb.AgentParams)
	for id, p := range s.builtinAgents {
		existingParams[id] = p.requestedState
	}
	newParams := make(map[string]agentpb.AgentParams)
	for id, p := range builtinAgents {
		newParams[id] = p
	}
	toStart, toRestart, toStop := filter(existingParams, newParams)

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

		agent, err := s.startBuiltin(agentID, builtinAgents[agentID])
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
			continue
		}
		s.builtinAgents[agentID] = agent
	}

	// start new agents
	for _, agentID := range toStart {
		agent, err := s.startBuiltin(agentID, builtinAgents[agentID])
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
			continue
		}
		s.builtinAgents[agentID] = agent
	}
}

// Changes returns channel with agent's state changes.
func (s *Supervisor) Changes() <-chan agentpb.StateChangedRequest {
	return s.changes
}

// QANRequests returns channel with agent's QAN Collect requests.
func (s *Supervisor) QANRequests() <-chan agentpb.QANCollectRequest {
	return s.qanRequests
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
	type_TEST_SLEEP agentpb.Type = 998 // process
	type_TEST_NOOP  agentpb.Type = 999 // built-in
)

// startProcess starts Agent's process and returns its info.
func (s *Supervisor) startProcess(agentID string, agentProcess *agentpb.SetStateRequest_AgentProcess, port uint16) (*agentProcessInfo, error) {
	processParams, err := s.processParams(agentID, agentProcess, port)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(s.ctx)
	agentType := strings.ToLower(agentProcess.Type.String())
	l := logrus.WithFields(logrus.Fields{
		"component": "agent-process",
		"agentID":   agentID,
		"type":      agentType,
	})
	process := process.New(processParams, l)
	go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), process.Run)

	done := make(chan struct{})
	go func() {
		for status := range process.Changes() {
			s.changes <- agentpb.StateChangedRequest{
				AgentId:    agentID,
				Status:     status,
				ListenPort: uint32(port),
			}
		}
		close(done)
	}()

	return &agentProcessInfo{
		cancel:         cancel,
		done:           done,
		requestedState: proto.Clone(agentProcess).(*agentpb.SetStateRequest_AgentProcess),
		listenPort:     port,
	}, nil
}

// startBuiltin starts builtin agent and returns its info.
func (s *Supervisor) startBuiltin(agentID string, builtinAgent *agentpb.SetStateRequest_BuiltinAgent) (*builtinAgentInfo, error) {
	ctx, cancel := context.WithCancel(s.ctx)
	agentType := strings.ToLower(builtinAgent.Type.String())
	l := logrus.WithFields(logrus.Fields{
		"component": "agent-builtin",
		"agentID":   agentID,
		"type":      agentType,
	})

	done := make(chan struct{})
	switch builtinAgent.Type {
	case agentpb.Type_QAN_MYSQL_PERFSCHEMA_AGENT:
		params := &mysql.Params{
			DSN: builtinAgent.Dsn,
		}
		m, err := mysql.New(params, l)
		if err != nil {
			cancel()
			return nil, err
		}
		go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), m.Run)

		go func() {
			for change := range m.Changes() {
				if change.Status != inventorypb.AgentStatus_AGENT_STATUS_INVALID {
					s.changes <- agentpb.StateChangedRequest{
						AgentId: agentID,
						Status:  change.Status,
					}
				} else {
					s.qanRequests <- agentpb.QANCollectRequest{
						Message: change.Request,
					}
				}
			}
			close(done)
		}()

	case type_TEST_NOOP:
		n := noop.New()
		go pprof.Do(ctx, pprof.Labels("agentID", agentID, "type", agentType), n.Run)

		go func() {
			for status := range n.Changes() {
				s.changes <- agentpb.StateChangedRequest{
					AgentId: agentID,
					Status:  status,
				}
			}
			close(done)
		}()

	default:
		cancel()
		return nil, errors.Errorf("unhandled agent type %[1]s (%[1]d).", builtinAgent.Type)
	}

	return &builtinAgentInfo{
		cancel:         cancel,
		done:           done,
		requestedState: proto.Clone(builtinAgent).(*agentpb.SetStateRequest_BuiltinAgent),
	}, nil
}

// "_" at the begginging is reserved for possible extensions
var textFileRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`) //nolint:gochecknoglobals

// processParams makes *process.Params from SetStateRequest parameters and other data.
func (s *Supervisor) processParams(agentID string, agentProcess *agentpb.SetStateRequest_AgentProcess, port uint16) (*process.Params, error) {
	var processParams process.Params
	switch agentProcess.Type {
	case agentpb.Type_NODE_EXPORTER:
		processParams.Path = s.paths.NodeExporter
	case agentpb.Type_MYSQLD_EXPORTER:
		processParams.Path = s.paths.MySQLdExporter
	case agentpb.Type_MONGODB_EXPORTER:
		processParams.Path = s.paths.MongoDBExporter
	case agentpb.Type_POSTGRES_EXPORTER:
		processParams.Path = s.paths.PostgresExporter
	case type_TEST_SLEEP:
		processParams.Path = "sleep"
	default:
		return nil, errors.Errorf("unhandled agent type %[1]s (%[1]d).", agentProcess.Type)
	}

	renderTemplate := func(name, text string, params map[string]interface{}) ([]byte, error) {
		t := template.New(name)
		t.Delims(agentProcess.TemplateLeftDelim, agentProcess.TemplateRightDelim)
		t.Option("missingkey=error")

		var buf bytes.Buffer
		if _, err := t.Parse(text); err != nil {
			return nil, errors.WithStack(err)
		}
		if err := t.Execute(&buf, params); err != nil {
			return nil, errors.WithStack(err)
		}
		return buf.Bytes(), nil
	}

	templateParams := map[string]interface{}{
		"listen_port": port,
	}

	// render files only if they are present to avoid creating temporary directory for every agent
	if len(agentProcess.TextFiles) > 0 {
		dir := filepath.Join(s.paths.TempDir, fmt.Sprintf("%s-%s", strings.ToLower(agentProcess.Type.String()), agentID))
		if err := os.RemoveAll(dir); err != nil {
			return nil, errors.WithStack(err)
		}
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, errors.WithStack(err)
		}

		textFiles := make(map[string]string, len(agentProcess.TextFiles)) // template name => full file path
		for name, text := range agentProcess.TextFiles {
			// avoid /, .., ., \, and other special symbols
			if !textFileRE.MatchString(name) {
				return nil, errors.Errorf("invalid text file name %q", name)
			}

			b, err := renderTemplate(name, text, templateParams)
			if err != nil {
				return nil, err
			}

			path := filepath.Join(dir, name)
			if err = ioutil.WriteFile(path, b, 0640); err != nil {
				return nil, errors.WithStack(err)
			}
			textFiles[name] = path
		}
		templateParams["TextFiles"] = textFiles
	}

	processParams.Args = make([]string, len(agentProcess.Args))
	for i, e := range agentProcess.Args {
		b, err := renderTemplate("args", e, templateParams)
		if err != nil {
			return nil, err
		}
		processParams.Args[i] = string(b)
	}

	processParams.Env = make([]string, len(agentProcess.Env))
	for i, e := range agentProcess.Env {
		b, err := renderTemplate("env", e, templateParams)
		if err != nil {
			return nil, err
		}
		processParams.Env[i] = string(b)
	}

	return &processParams, nil
}

func (s *Supervisor) stopAll() {
	s.m.Lock()
	defer s.m.Unlock()

	wait := make([]chan struct{}, 0, len(s.agentProcesses)+len(s.builtinAgents))

	for _, agent := range s.agentProcesses {
		agent.cancel()
		wait = append(wait, agent.done)
	}
	s.agentProcesses = nil

	for _, agent := range s.builtinAgents {
		agent.cancel()
		wait = append(wait, agent.done)
	}
	s.builtinAgents = nil

	for _, ch := range wait {
		<-ch
	}
	close(s.qanRequests)
	close(s.changes)
}
