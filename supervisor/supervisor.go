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

// Package supervisor provides process supervisor for running Agents.
package supervisor

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/percona/pmm/api/agent"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-agent/config"
)

// Supervisor manages all agent's processes.
type Supervisor struct {
	ctx           context.Context
	paths         *config.Paths
	portsRegistry *portsRegistry
	changes       chan agent.StateChangedRequest
	l             *logrus.Entry

	m      sync.Mutex
	agents map[string]*agentInfo
}

type agentInfo struct {
	process        *process
	cancel         func()
	done           chan struct{}
	requestedState *agent.SetStateRequest_AgentProcess
	listenPort     uint16
}

// NewSupervisor creates new Supervisor object.
func NewSupervisor(ctx context.Context, paths *config.Paths, ports *config.Ports) *Supervisor {
	supervisor := &Supervisor{
		ctx:           ctx,
		paths:         paths,
		portsRegistry: newPortsRegistry(ports.Min, ports.Max, nil),
		changes:       make(chan agent.StateChangedRequest, 10),
		l:             logrus.WithField("component", "supervisor"),

		agents: make(map[string]*agentInfo),
	}

	go func() {
		<-ctx.Done()
		supervisor.stopAll()
	}()

	return supervisor
}

// SetState starts or updates all agents placed in args and stops all agents not placed in args, but already run.
func (s *Supervisor) SetState(agentProcesses map[string]*agent.SetStateRequest_AgentProcess) {
	s.m.Lock()
	defer s.m.Unlock()

	if err := s.ctx.Err(); err != nil {
		s.l.Errorf("Ignoring SetState: %s.", err)
		return
	}

	toStart, toRestart, toStop := s.filter(agentProcesses)

	// We have to wait for processes to terminate before starting a new ones to reuse ports.
	// If that place is slow, we can cancel them all in parallel, but then we still have to wait.

	// stop first to avoid extra load
	for _, agentID := range toStop {
		agent := s.agents[agentID]
		agent.cancel()
		<-agent.done // wait before releasing port

		if err := s.portsRegistry.Release(agent.listenPort); err != nil {
			s.l.Errorf("Failed to release port: %s.", err)
		}
		delete(s.agents, agentID)
	}

	// restart while preserving port
	for _, agentID := range toRestart {
		agent := s.agents[agentID]
		agent.cancel()
		<-agent.done // wait before reusing port

		agent, err := s.start(agentID, agentProcesses[agentID], agent.listenPort)
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
			continue
		}
		s.agents[agentID] = agent
	}

	// start new agents
	for _, agentID := range toStart {
		port, err := s.portsRegistry.Reserve()
		if err != nil {
			s.l.Errorf("Failed to reserve port: %s.", err)
			// TODO report that error to server
			continue
		}

		agent, err := s.start(agentID, agentProcesses[agentID], port)
		if err != nil {
			s.l.Errorf("Failed to start Agent: %s.", err)
			// TODO report that error to server
			continue
		}
		s.agents[agentID] = agent
	}
}

// Changes returns channel with agent's state changes.
func (s *Supervisor) Changes() <-chan agent.StateChangedRequest {
	return s.changes
}

// filter extracts IDs of the Agents that should be started, restarted with new parameters, or stopped,
// and filters out IDs of the Agents that should not be changed.
func (s *Supervisor) filter(agentProcesses map[string]*agent.SetStateRequest_AgentProcess) (toStart, toRestart, toStop []string) {
	// existing agents not present in the new requested state should be stopped
	for agentID := range s.agents {
		if agentProcesses[agentID] == nil {
			toStop = append(toStop, agentID)
		}
	}

	// detect new and changed agents
	for agentID, agentProcess := range agentProcesses {
		agent := s.agents[agentID]
		if agent == nil {
			toStart = append(toStart, agentID)
			continue
		}

		// compare parameters before templating
		if proto.Equal(agent.requestedState, agentProcess) {
			continue
		}

		toRestart = append(toRestart, agentID)
	}

	sort.Strings(toStop)
	sort.Strings(toRestart)
	sort.Strings(toStart)
	return
}

// start starts agent and returns its info.
func (s *Supervisor) start(agentID string, agentProcess *agent.SetStateRequest_AgentProcess, port uint16) (*agentInfo, error) {
	processParams, err := s.processParams(agentID, agentProcess, port)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(s.ctx)
	l := logrus.WithFields(logrus.Fields{
		"component": "agent",
		"agentID":   agentID,
		"type":      strings.ToLower(agentProcess.Type.String()),
	})
	process := newProcess(ctx, processParams, l)

	done := make(chan struct{})
	go func() {
		for status := range process.Changes() {
			s.changes <- agent.StateChangedRequest{
				AgentId:    agentID,
				Status:     status,
				ListenPort: uint32(port),
			}
		}
		close(done)
	}()

	return &agentInfo{
		process:        process,
		cancel:         cancel,
		done:           done,
		requestedState: proto.Clone(agentProcess).(*agent.SetStateRequest_AgentProcess),
		listenPort:     port,
	}, nil
}

// _ as a first rune is kept for possible extensions
var textFileRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// processParams makes processParams from SetStateRequest parameters and other data.
func (s *Supervisor) processParams(agentID string, agentProcess *agent.SetStateRequest_AgentProcess, port uint16) (*processParams, error) {
	var processParams processParams
	switch agentProcess.Type {
	case agent.Type_NODE_EXPORTER:
		processParams.path = s.paths.NodeExporter
	case agent.Type_MYSQLD_EXPORTER:
		processParams.path = s.paths.MySQLdExporter
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
		"ListenPort": port,
	}

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

	processParams.args = make([]string, len(agentProcess.Args))
	for i, e := range agentProcess.Args {
		b, err := renderTemplate("args", e, templateParams)
		if err != nil {
			return nil, err
		}
		processParams.args[i] = string(b)
	}

	processParams.env = make([]string, len(agentProcess.Env))
	for i, e := range agentProcess.Env {
		b, err := renderTemplate("env", e, templateParams)
		if err != nil {
			return nil, err
		}
		processParams.env[i] = string(b)
	}

	return &processParams, nil
}

func (s *Supervisor) stopAll() {
	s.m.Lock()
	defer s.m.Unlock()

	wait := make([]chan struct{}, 0, len(s.agents))
	for _, agent := range s.agents {
		agent.cancel()
		wait = append(wait, agent.done)
	}
	s.agents = nil

	for _, ch := range wait {
		<-ch
	}
	close(s.changes)
}
