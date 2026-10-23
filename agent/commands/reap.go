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

package commands

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// How long an orphaned exporter is given to exit after SIGTERM before it is killed outright.
	orphanStopTimeout  = 5 * time.Second
	orphanPollInterval = 100 * time.Millisecond

	// A process reparented after its own parent died. Exporters are started without a process
	// group of their own, so this is what being left behind looks like.
	initPID = 1
)

// reapOrphanedExporters stops exporters left running by a previous pmm-agent.
//
// Stopping an Agent means cancelling it, which signals its process (see agent/agents/process).
// That only happens when pmm-agent is allowed to shut down, and the Client upgrade does not allow
// it: pmm-agent is replaced with SIGKILL, which cannot be caught, so every exporter keeps running
// and is reparented to init. The pmm-agent that takes over knows nothing
// about them, and cleanupTmp is no help either - it runs on the way out, not on the way in.
//
// One leftover exporter holding an exclusive resource then blocks its own replacement for good.
// A leftover vmagent keeps an exclusive lock on its persistent queue, and that queue is one directory per
// host rather than one per Agent, so the vmagent this pmm-agent starts panics within a second of
// starting - every time, until somebody kills the old process by hand. See PMM-15773.
func reapOrphanedExporters(exportersBase string, l *logrus.Entry) {
	procs, err := procfs.AllProcs()
	if err != nil {
		l.Warnf("Failed to look for exporters left behind by a previous pmm-agent: %s.", err)

		return
	}

	signaled := make([]int, 0, len(procs))
	for _, p := range procs {
		// Either of these fails for a process that exited while we were listing, or that
		// belongs to another user - neither is ours to reap.
		exe, err := p.Executable()
		if err != nil {
			continue
		}
		stat, err := p.Stat()
		if err != nil {
			continue
		}

		if !isOrphanedExporter(exe, exportersBase, stat.PPID) {
			continue
		}

		l.Warnf("Stopping %s (PID %d), left running by a previous pmm-agent.", exe, p.PID)
		err = unix.Kill(p.PID, unix.SIGTERM)
		if err != nil {
			l.Warnf("Failed to signal PID %d: %s.", p.PID, err)

			continue
		}
		signaled = append(signaled, p.PID)
	}

	// They are init's children rather than ours, so there is nothing to wait on: asking
	// whether the PID still exists is all that is available.
	deadline := time.Now().Add(orphanStopTimeout)
	for _, pid := range signaled {
		for unix.Kill(pid, 0) == nil && time.Now().Before(deadline) {
			time.Sleep(orphanPollInterval)
		}

		if unix.Kill(pid, 0) == nil {
			l.Warnf("PID %d did not exit within %s, killing it.", pid, orphanStopTimeout)
			err = unix.Kill(pid, unix.SIGKILL)
			if err != nil {
				l.Warnf("Failed to kill PID %d: %s.", pid, err)
			}
		}
	}
}

// isOrphanedExporter reports whether a process is an exporter left behind by a previous pmm-agent:
// its executable lives in this pmm-agent's exporters directory, and it has been reparented, which
// means the pmm-agent that started it is gone.
//
// A process whose parent is still alive is deliberately left alone. Those are somebody's running
// Agents - on a host with a second pmm-agent, they are its - and killing them would be this same
// bug pointed the other way.
func isOrphanedExporter(exe, exportersBase string, ppid int) bool {
	if exe == "" || exportersBase == "" || ppid != initPID {
		return false
	}

	// Relative rather than a string prefix, so that the exporters directory cannot match a
	// sibling whose name merely starts with it: /opt/exporters-old against /opt/exporters.
	rel, err := filepath.Rel(filepath.Clean(exportersBase), filepath.Clean(exe))
	if err != nil {
		return false
	}

	return rel != "." && !strings.HasPrefix(rel, "..")
}
