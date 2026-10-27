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
	"strconv"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// The directory, under pmm-agent's temporary directory, that vmagent keeps its persistent
	// queue in. It must match -remoteWrite.tmpDataPath in managed/services/agents/vmagent.go.
	vmagentQueueDirName = "vmagent-temp-dir"

	// How long stopping a leftover vmagent may delay pmm-agent's start: after SIGTERM, and then after
	// SIGKILL, since its lock is released only once it has exited.
	leftoverStopTimeout = 5 * time.Second
	leftoverKillTimeout = 5 * time.Second

	leftoverPollInterval = 100 * time.Millisecond
)

type leftoverTimeouts struct {
	stop time.Duration // after SIGTERM, before SIGKILL
	kill time.Duration // after SIGKILL, before giving up
}

// stopLeftoverVMAgents stops the vmagent processes holding this pmm-agent's persistent queue, before
// pmm-agent starts a vmagent of its own that would exit on their lock. See PMM-15773.
//
// The parent is deliberately not what decides. Every exporter gets SIGKILL from pmm-agent as its
// parent-death signal, so a pmm-agent that is killed outright takes them with it; a vmagent still
// holding the queue when pmm-agent starts was left by something else, and only one vmagent can use
// the queue.
func stopLeftoverVMAgents(vmagentPath, tempDir string, l *logrus.Entry) {
	timeouts := leftoverTimeouts{stop: leftoverStopTimeout, kill: leftoverKillTimeout}
	stopLeftovers(vmagentPath, tempDir, timeouts, unix.PidfdOpen, l)
}

func stopLeftovers(vmagentPath, tempDir string, timeouts leftoverTimeouts, pidfdOpen func(int, int) (int, error), l *logrus.Entry) {
	if vmagentPath == "" || tempDir == "" {
		return
	}
	// /proc/<pid>/exe has its symlinks resolved, so the configured path must have them resolved too.
	resolved, err := filepath.EvalSymlinks(vmagentPath)
	if err == nil {
		vmagentPath = resolved
	}

	queueDir := filepath.Join(tempDir, vmagentQueueDirName)
	queue, err := os.Stat(queueDir)
	if err != nil {
		return // no queue yet, so nothing can be holding it
	}

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
		if err != nil || !isVMAgentExecutable(exe, vmagentPath) {
			continue
		}
		// Read only now: reading the command line of a process stuck in the kernel can block, and
		// every other process on the host would be one more chance of that.
		cmdline, err := p.CmdLine()
		if err != nil {
			continue
		}
		path, ok := tmpDataPath(cmdline)
		if !ok || !holdsQueue(p.PID, path, queue) {
			continue
		}
		stat, err := p.Stat()
		if err != nil {
			continue
		}

		lo, err := openLeftover(p.PID, stat.Starttime, pidfdOpen, l)
		if err != nil {
			continue // gone already
		}
		l.Warnf("Stopping vmagent (PID %d, parent PID %d) left running from before this pmm-agent started: "+
			"it holds %s, so the vmagent this pmm-agent starts would exit on the queue lock.", p.PID, stat.PPID, queueDir)
		warnLiveParent(stat.PPID, queueDir, l)
		lo.signal(unix.SIGTERM)
		stopping = append(stopping, lo)
	}

	// Signaled together, so they share the budget rather than spend it one after another.
	deadline := time.Now().Add(timeouts.stop)
	for _, lo := range stopping {
		if !lo.waitExit(time.Until(deadline)) {
			l.Warnf("vmagent PID %d did not exit within %s, killing it.", lo.pid, timeouts.stop)
			lo.signal(unix.SIGKILL)
			if !lo.waitExit(timeouts.kill) {
				l.Warnf("vmagent PID %d is still running after SIGKILL; "+
					"the vmagent this pmm-agent starts may fail on the queue lock.", lo.pid)
			}
		}
		lo.close()
	}
}

// holdsQueue reports whether path, a process's -remoteWrite.tmpDataPath, is the queue directory.
// They are compared as files rather than strings: the same directory may be spelled differently,
// and in another mount namespace the same spelling may be a different directory.
func holdsQueue(pid int, path string, queue os.FileInfo) bool {
	base := "root"
	if !filepath.IsAbs(path) {
		base = "cwd"
	}
	fi, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid), base, path))
	if err != nil {
		return false
	}

	return os.SameFile(fi, queue)
}

// warnLiveParent says what started a leftover vmagent whose parent is still alive: that parent may
// well start it again, and then only the two sharing a queue is the problem to fix.
func warnLiveParent(ppid int, queueDir string, l *logrus.Entry) {
	if ppid <= 1 {
		return
	}
	exe := "unknown"
	parent, err := procfs.NewProc(ppid)
	if err == nil {
		parentExe, exeErr := parent.Executable()
		if exeErr == nil && parentExe != "" {
			exe = parentExe
		}
	}
	l.Errorf("The leftover vmagent was started by PID %d (%s), which is still running and may start it again. "+
		"Only one pmm-agent can use %s.", ppid, exe, queueDir)
}

// leftover is a process being stopped, held by a pidfd so that a PID reused after it exits can
// never be signaled in its place. Without pidfd support it falls back to the process start time,
// re-checked before every signal.
type leftover struct {
	pid       int
	starttime uint64
	fd        int // -1 without pidfd support
}

var errLeftoverGone = errors.New("process is gone")

func openLeftover(pid int, starttime uint64, pidfdOpen func(int, int) (int, error), l *logrus.Entry) (*leftover, error) {
	fd, err := pidfdOpen(pid, 0)
	switch {
	case err == nil:
	case errors.Is(err, unix.ESRCH):
		return nil, errLeftoverGone
	default:
		// ENOSYS before Linux 5.3; EPERM under a seccomp profile older than pidfd; EMFILE.
		if !errors.Is(err, unix.ENOSYS) {
			l.Debugf("pidfd_open(%d) failed, falling back to the start time: %s.", pid, err)
		}
		fd = -1
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
	deadline := time.Now().Add(timeout)
	if lo.fd >= 0 {
		// A pidfd becomes readable when its process exits.
		fds := []unix.PollFd{{Fd: int32(lo.fd), Events: unix.POLLIN}} //nolint:gosec // fd fits int32
		for {
			n, err := unix.Poll(fds, int(max(time.Until(deadline), 0).Milliseconds()))
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return err == nil && n > 0
		}
	}

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
