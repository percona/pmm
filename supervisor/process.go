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

package supervisor

import (
	"context"
	"os/exec"
	"time"

	"github.com/percona/pmm/api/inventory"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	runningT = time.Second     // STARTING -> RUNNING delay
	killT    = 5 * time.Second // SIGTERM -> SIGKILL delay

	keepLogLines = 100
)

// process represents sub-agent process.
//
// Process object should be created with newProcess. It then handles process starting, restarting with backoff,
// reading its output. Process is gracefully stopped when context passed to newProcess is canceled.
// Changes of process status are reported via Changes channel which must be read until it is closed.
//
// Process status is changed by finite state machine (see agent_status.dot).
// Each state logic is encapsulated in toXXX methods. Each method sends a new status to the changes channel,
// implements its own logic, and then switches to then next state via "go toXXX()". "go" statement is used
// only to avoid stack overflow; there are no extra goroutines for states.
type process struct {
	ctx     context.Context
	params  *processParams
	l       *logrus.Entry
	pl      *processLogger
	changes chan inventory.AgentStatus
	backoff *backoff
	ctxDone chan struct{}

	// recreated on each restart
	cmd     *exec.Cmd
	cmdDone chan struct{}
}

type processParams struct {
	path string
	args []string
	env  []string
}

func newProcess(ctx context.Context, params *processParams, l *logrus.Entry) *process {
	b := new(backoff)
	b.Reset()

	p := &process{
		ctx:     ctx,
		params:  params,
		l:       l,
		pl:      newProcessLogger(l, keepLogLines),
		changes: make(chan inventory.AgentStatus, 1),
		backoff: b,
		ctxDone: make(chan struct{}),
	}

	go func() {
		<-ctx.Done()
		p.l.Infof("Process: context canceled.")
		close(p.ctxDone)
	}()

	go p.toStarting()

	return p
}

// STARTING -> RUNNING
// STARTING -> WAITING
func (p *process) toStarting() {
	p.l.Infof("Process: starting.")
	p.changes <- inventory.AgentStatus_STARTING

	p.cmd = exec.Command(p.params.path, p.params.args...) //nolint:gosec
	p.cmd.Env = p.params.env
	p.cmd.Stdout = p.pl
	p.cmd.Stderr = p.pl
	setSysProcAttr(p.cmd)

	p.cmdDone = make(chan struct{})

	if err := p.cmd.Start(); err != nil {
		p.l.Warnf("Process: failed to start: %s.", err)
		go p.toWaiting()
		return
	}

	go func() {
		// p.cmd.ProcessState is checked once cmdDone is closed, so error there can be ignored
		_ = p.cmd.Wait()
		close(p.cmdDone)
	}()

	t := time.NewTimer(runningT)
	defer t.Stop()
	select {
	case <-t.C:
		go p.toRunning()
	case <-p.cmdDone:
		p.l.Warnf("Process: exited early: %s.", p.cmd.ProcessState)
		go p.toWaiting()
	}
}

// RUNNING -> STOPPING
// RUNNING -> WAITING
func (p *process) toRunning() {
	p.l.Infof("Process: running.")
	p.changes <- inventory.AgentStatus_RUNNING

	p.backoff.Reset()

	select {
	case <-p.ctxDone:
		go p.toStopping()
	case <-p.cmdDone:
		p.l.Warnf("Process: exited: %s.", p.cmd.ProcessState)
		go p.toWaiting()
	}
}

// WAITING -> STARTING
// WAITING -> DONE
func (p *process) toWaiting() {
	delay := p.backoff.Delay()

	p.l.Infof("Process: waiting %s.", delay)
	p.changes <- inventory.AgentStatus_WAITING

	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-t.C:
		go p.toStarting()
	case <-p.ctxDone:
		go p.toDone()
	}
}

// STOPPING -> DONE
func (p *process) toStopping() {
	p.l.Infof("Process: stopping (sending SIGTERM)...")
	p.changes <- inventory.AgentStatus_STOPPING

	if err := p.cmd.Process.Signal(unix.SIGTERM); err != nil {
		p.l.Errorf("Process: failed to send SIGTERM: %s.", err)
	}

	t := time.NewTimer(killT)
	defer t.Stop()
	select {
	case <-p.cmdDone:
		// nothing
	case <-t.C:
		p.l.Warnf("Process: still alive after %s, sending SIGKILL...", killT)
		if err := p.cmd.Process.Signal(unix.SIGKILL); err != nil {
			p.l.Errorf("Process: failed to send SIGKILL: %s.", err)
		}
		<-p.cmdDone
	}

	p.l.Infof("Process: exited: %s.", p.cmd.ProcessState)
	go p.toDone()
}

func (p *process) toDone() {
	p.l.Info("Process: done.")
	p.changes <- inventory.AgentStatus_DONE

	close(p.changes)
}

// Changes returns channel that should be read until it is closed.
func (p *process) Changes() <-chan inventory.AgentStatus {
	return p.changes
}

// Logs returns latest process logs.
func (p *process) Logs() []string {
	return p.pl.Latest()
}
