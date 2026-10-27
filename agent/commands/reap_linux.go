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
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// leftoverTimeouts bound how long stopping a leftover vmagent may delay pmm-agent's start.
type leftoverTimeouts struct {
	stop time.Duration // after SIGTERM, before SIGKILL
	kill time.Duration // after SIGKILL, before giving up
}

const (
	// The directory, under pmm-agent's temporary directory, that vmagent keeps its persistent
	// queue in. It must match -remoteWrite.tmpDataPath in managed/services/agents/vmagent.go.
	vmagentQueueDirName = "vmagent-temp-dir"

	leftoverPollInterval = 100 * time.Millisecond
)

// stopLeftoverVMAgents stops vmagent processes still holding this pmm-agent's persistent queue (see
// isLeftoverVMAgent) before pmm-agent starts a vmagent of its own. See PMM-15773.
func stopLeftoverVMAgents(vmagentPath, tempDir string, l *logrus.Entry) {
	stopLeftovers(vmagentPath, tempDir, leftoverTimeouts{stop: 5 * time.Second, kill: 5 * time.Second}, unix.PidfdOpen, l)
}

func stopLeftovers(vmagentPath, tempDir string, timeouts leftoverTimeouts, pidfdOpen func(int, int) (int, error), l *logrus.Entry) {
	if vmagentPath == "" || tempDir == "" {
		return
	}
	// /proc/<pid>/exe has its symlinks resolved, so the configured path must have them resolved too.
	if resolved, err := filepath.EvalSymlinks(vmagentPath); err == nil {
		vmagentPath = resolved
	}
	queueDir := filepath.Join(tempDir, vmagentQueueDirName)

	procs, err := procfs.AllProcs()
	if err != nil {
		l.Warnf("Failed to look for vmagent processes holding %s: %s.", queueDir, err)
		return
	}

	self := os.Getpid()
	var stopping []*leftover
	for _, p := range procs {
		if p.PID == self {
			continue
		}
		// Any of these fails for a process that exited while we were listing, or that belongs
		// to another user - neither is ours to stop.
		exe, err := p.Executable()
		if err != nil {
			continue
		}
		cmdline, err := p.CmdLine()
		if err != nil {
			continue
		}
		if !isLeftoverVMAgent(exe, cmdline, vmagentPath, queueDir) {
			continue
		}
		stat, err := p.Stat()
		if err != nil {
			continue
		}

		lo, err := openLeftover(p.PID, stat.Starttime, pidfdOpen)
		if err != nil {
			continue // gone already
		}
		l.Warnf("Stopping vmagent (PID %d, parent PID %d) left running from before this pmm-agent started: "+
			"it holds %s, so the vmagent this pmm-agent starts would exit on the queue lock.", p.PID, stat.PPID, queueDir)
		lo.signal(unix.SIGTERM)
		stopping = append(stopping, lo)
	}

	// Signaled together, so they share the budget rather than spend it one after another.
	deadline := time.Now().Add(timeouts.stop)
	for _, lo := range stopping {
		if !lo.waitExit(time.Until(deadline)) {
			l.Warnf("vmagent PID %d did not exit within %s, killing it.", lo.pid, timeouts.stop)
			lo.signal(unix.SIGKILL)
			// The lock is released only once the process has exited, so killing it is not enough.
			if !lo.waitExit(timeouts.kill) {
				l.Warnf("vmagent PID %d is still running after SIGKILL; "+
					"the vmagent this pmm-agent starts may fail on the queue lock.", lo.pid)
			}
		}
		lo.close()
	}
}

// leftover is a process being stopped, held by a pidfd so that a PID reused after it exits can
// never be signaled in its place. Without pidfd support (Linux before 5.3) it falls back to the
// process start time, re-checked before every signal.
type leftover struct {
	pid       int
	starttime uint64
	fd        int // -1 without pidfd support
}

var errLeftoverGone = errors.New("process is gone")

func openLeftover(pid int, starttime uint64, pidfdOpen func(int, int) (int, error)) (*leftover, error) {
	fd, err := pidfdOpen(pid, 0)
	switch {
	case errors.Is(err, unix.ENOSYS):
		fd = -1
	case err != nil:
		return nil, err
	}

	lo := &leftover{pid: pid, starttime: starttime, fd: fd}
	// The PID may have been reused between listing it and opening it: the pidfd refers to
	// whichever process holds the PID now, so confirm that is still the one inspected.
	if !lo.running() {
		lo.close()
		return nil, errLeftoverGone
	}

	return lo, nil
}

// running reports whether the inspected process is still alive. A zombie is not: it has released
// its locks, and only its exit status is waiting for whoever reaps it.
func (lo *leftover) running() bool {
	p, err := procfs.NewProc(lo.pid)
	if err != nil {
		return false
	}
	stat, err := p.Stat()
	if err != nil {
		return false
	}

	return stat.Starttime == lo.starttime && stat.State != "Z" && stat.State != "X"
}

func (lo *leftover) signal(sig unix.Signal) {
	if lo.fd >= 0 {
		_ = unix.PidfdSendSignal(lo.fd, sig, nil, 0) // ESRCH: it has exited already
		return
	}
	if lo.running() {
		_ = unix.Kill(lo.pid, sig)
	}
}

// waitExit waits up to timeout for the process to exit, and reports whether it has.
func (lo *leftover) waitExit(timeout time.Duration) bool {
	if lo.fd >= 0 {
		// A pidfd becomes readable when its process exits.
		fds := []unix.PollFd{{Fd: int32(lo.fd), Events: unix.POLLIN}} //nolint:gosec // fd fits int32
		deadline := time.Now().Add(timeout)
		for {
			n, err := unix.Poll(fds, int(max(time.Until(deadline), 0).Milliseconds()))
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return err == nil && n > 0
		}
	}

	deadline := time.Now().Add(timeout)
	for lo.running() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(leftoverPollInterval)
	}

	return true
}

func (lo *leftover) close() {
	if lo.fd >= 0 {
		_ = unix.Close(lo.fd)
		lo.fd = -1
	}
}
