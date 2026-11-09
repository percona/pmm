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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// Must match -remoteWrite.tmpDataPath in managed/services/agents/vmagent.go.
	vmagentQueueDirName = "vmagent-temp-dir"

	tmpDataPathFlag = "remoteWrite.tmpDataPath"

	// What /proc/<pid>/exe ends with once an upgrade has replaced the running binary.
	deletedExeSuffix = " (deleted)"

	// TASK_COMM_LEN less its terminating NUL: the kernel cuts a process's command name to it.
	commLen = 15

	// Reading /proc can block on a process stuck in the kernel, so pmm-agent starts after leftoverBudget anyway.
	leftoverBudget      = 15 * time.Second
	leftoverStopTimeout = 5 * time.Second
	leftoverKillTimeout = 5 * time.Second

	leftoverPollInterval = 100 * time.Millisecond

	queueLockCheckInterval = time.Minute
)

// leftoverTimeouts bounds the whole sweep, and the waits after SIGTERM and after SIGKILL that all leftovers share.
type leftoverTimeouts struct {
	budget time.Duration
	stop   time.Duration
	kill   time.Duration
}

// stopLeftoverVMAgents stops the vmagents started before this pmm-agent that hold its queue, whatever
// their parent; see PMM-15773.
func stopLeftoverVMAgents(ctx context.Context, vmagentPath, tempDir string, l *logrus.Entry) {
	self, err := procfs.Self()
	var stat procfs.ProcStat
	if err == nil {
		stat, err = self.Stat()
	}
	if err != nil {
		l.Warnf("Not looking for leftover vmagents, as this pmm-agent's start time cannot be read: %s.", err)
		return
	}

	timeouts := leftoverTimeouts{budget: leftoverBudget, stop: leftoverStopTimeout, kill: leftoverKillTimeout}
	stopLeftoversWithin(ctx, vmagentPath, tempDir, stat.Starttime, timeouts, unix.PidfdOpen, l)
}

// stopLeftoversWithin waits for stopLeftovers until it returns, the budget runs out or ctx is done, and
// returns a channel closed once stopLeftovers returns.
func stopLeftoversWithin(
	ctx context.Context, vmagentPath, tempDir string, startedBefore uint64, timeouts leftoverTimeouts,
	pidfdOpen func(int, int) (int, error), l *logrus.Entry,
) <-chan struct{} {
	done := make(chan struct{})
	// Left running past the budget, it cannot hit this pmm-agent's own vmagent: it stops only older ones.
	go func() {
		defer close(done)
		stopLeftovers(ctx, vmagentPath, tempDir, startedBefore, timeouts, pidfdOpen, l)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	case <-time.After(timeouts.budget):
		l.Warnf("Stopping leftover vmagents is taking longer than %s, starting without waiting for it; "+
			"the vmagent this pmm-agent starts may fail on the queue lock until it is done.", timeouts.budget)
	}

	return done
}

// stopLeftovers stops the vmagents holding the queue under tempDir that started before startedBefore, in
// clock ticks since boot, and signals none once ctx is done.
func stopLeftovers(
	ctx context.Context, vmagentPath, tempDir string, startedBefore uint64, timeouts leftoverTimeouts,
	pidfdOpen func(int, int) (int, error), l *logrus.Entry,
) {
	if vmagentPath == "" || tempDir == "" {
		return
	}
	comms := []string{comm(vmagentPath)}
	// /proc/<pid>/exe has its symlinks resolved, so the configured path must have them resolved too.
	resolved, err := filepath.EvalSymlinks(vmagentPath)
	if err == nil {
		vmagentPath = resolved
		comms = append(comms, comm(resolved))
	}

	queueDir := filepath.Join(tempDir, vmagentQueueDirName)
	queue, err := os.Stat(queueDir)
	if err != nil {
		return
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
		// openLeftover reads the start time again, so everything read in between is about this one process.
		stat, err := p.Stat()
		if err != nil || stat.Starttime >= startedBefore || !slices.Contains(comms, stat.Comm) {
			continue
		}
		// Read only for processes named like vmagent, as reading /proc of one stuck in the kernel can block.
		exe, err := p.Executable()
		if err != nil || !isVMAgentExecutable(exe, vmagentPath) {
			continue
		}
		cmdline, err := p.CmdLine()
		if err != nil {
			continue
		}
		path, ok := tmpDataPath(cmdline)
		if !ok || !holdsQueue(p.PID, path, queue) {
			continue
		}

		lo, err := openLeftover(p.PID, stat.Starttime, pidfdOpen, l)
		if err != nil {
			continue
		}
		if ctx.Err() != nil {
			lo.close()
			break
		}
		l.Warnf("Stopping vmagent (PID %d, parent PID %d) left running from before this pmm-agent started: "+
			"it holds %s, so the vmagent this pmm-agent starts would exit on the queue lock.", p.PID, stat.PPID, queueDir)
		warnCurrentParent(stat.PPID, queueDir, l)
		lo.signal(unix.SIGTERM, l)
		stopping = append(stopping, lo)
	}

	waitLeftovers(ctx, stopping, timeouts, l)
	for _, lo := range stopping {
		lo.close()
	}
}

// waitLeftovers waits for the signaled leftovers to exit, and kills those still running after timeouts.stop.
func waitLeftovers(ctx context.Context, stopping []*leftover, timeouts leftoverTimeouts, l *logrus.Entry) {
	// Signaled together, so they share one wait after SIGTERM, and one after SIGKILL.
	deadline := time.Now().Add(timeouts.stop)
	var killed []*leftover
	for _, lo := range stopping {
		if lo.waitExit(time.Until(deadline)) || ctx.Err() != nil {
			continue
		}
		l.Warnf("vmagent PID %d did not exit within %s, killing it.", lo.pid, timeouts.stop)
		lo.signal(unix.SIGKILL, l)
		killed = append(killed, lo)
	}
	deadline = time.Now().Add(timeouts.kill)
	for _, lo := range killed {
		if !lo.waitExit(time.Until(deadline)) {
			l.Warnf("vmagent PID %d is still running after SIGKILL; "+
				"the vmagent this pmm-agent starts may fail on the queue lock.", lo.pid)
		}
	}
}

// comm returns the command name /proc/<pid>/stat shows for a process started from path.
func comm(path string) string {
	name := filepath.Base(path)
	return name[:min(len(name), commLen)]
}

// isVMAgentExecutable reports whether exe, a process's /proc/<pid>/exe, is the vmagent binary, also once an
// upgrade has replaced it.
func isVMAgentExecutable(exe, vmagentPath string) bool {
	if exe == "" || vmagentPath == "" {
		return false
	}

	return filepath.Clean(strings.TrimSuffix(exe, deletedExeSuffix)) == filepath.Clean(vmagentPath)
}

// tmpDataPath returns the -remoteWrite.tmpDataPath of a vmagent command line, the queue directory it locks,
// taking the last one in any form Go's flag package accepts.
func tmpDataPath(cmdline []string) (string, bool) {
	var path string
	for i := 1; i < len(cmdline); i++ {
		flag, ok := strings.CutPrefix(cmdline[i], "-")
		if !ok {
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(flag, "-"), "=")
		if name != tmpDataPathFlag {
			continue
		}
		if !hasValue {
			if i+1 == len(cmdline) {
				break
			}
			i++
			value = cmdline[i]
		}
		path = value
	}

	return path, path != ""
}

// holdsQueue reports whether path, a process's -remoteWrite.tmpDataPath, is the queue directory, compared as
// files since one directory has many spellings, and one spelling is another directory in another mount namespace.
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

// warnCurrentParent names the current parent of a leftover vmagent, which may have started it, or may be a
// subreaper it was re-parented to.
func warnCurrentParent(ppid int, queueDir string, l *logrus.Entry) {
	if ppid <= 1 {
		return
	}
	l.Warnf("The leftover vmagent's current parent is PID %d (%s): if that started it, it may start it again, "+
		"and only one pmm-agent can use %s.", ppid, executable(ppid), queueDir)
}

// executable returns the /proc/<pid>/exe of a process, or "unknown".
func executable(pid int) string {
	p, err := procfs.NewProc(pid)
	if err != nil {
		return "unknown"
	}
	exe, err := p.Executable()
	if err != nil || exe == "" {
		return "unknown"
	}

	return exe
}

// watchQueueLock warns once about each process, other than this pmm-agent's children, that holds a lock in its
// vmagent queue and so keeps its vmagent from starting; the sweep at start cannot stop one started later.
func watchQueueLock(ctx context.Context, tempDir string, l *logrus.Entry) {
	queueDir := filepath.Join(tempDir, vmagentQueueDirName)
	reported := make(map[int]bool)
	t := time.NewTicker(queueLockCheckInterval)
	defer t.Stop()
	for {
		warnQueueLockHolders(queueDir, os.Getpid(), reported, l)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// warnQueueLockHolders warns about the holders of a lock in queueDir that are neither children of self nor in
// reported, and adds them to reported.
func warnQueueLockHolders(queueDir string, self int, reported map[int]bool, l *logrus.Entry) {
	for _, pid := range queueLockHolders(queueDir) {
		if reported[pid] {
			continue
		}
		p, err := procfs.NewProc(pid)
		if err != nil {
			continue
		}
		stat, err := p.Stat()
		if err != nil || stat.PPID == self {
			continue
		}
		reported[pid] = true
		started := "unknown"
		startTime, err := stat.StartTime()
		if err == nil {
			started = time.Unix(int64(startTime), 0).UTC().Format(time.RFC3339)
		}
		l.Warnf("PID %d (%s, started %s, parent PID %d %s) holds the lock on %s, so this pmm-agent's vmagent cannot start.",
			pid, executable(pid), started, stat.PPID, executable(stat.PPID), queueDir)
	}
}

// queueLockHolders returns the PIDs that /proc/locks shows holding a lock on a file directly in queueDir.
func queueLockHolders(queueDir string) []int {
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		return nil
	}
	files := make(map[string]bool, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if ok {
			files[fmt.Sprintf("%02x:%02x:%d", unix.Major(st.Dev), unix.Minor(st.Dev), st.Ino)] = true
		}
	}

	locks, err := os.ReadFile("/proc/locks")
	if err != nil {
		return nil
	}
	var pids []int
	for line := range strings.Lines(string(locks)) {
		// "1: FLOCK  ADVISORY  WRITE <pid> <major>:<minor>:<inode> 0 EOF"; a waiter has "->" after the ID.
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[1] == "->" || !files[fields[5]] {
			continue
		}
		pid, err := strconv.Atoi(fields[4])
		if err == nil && pid > 0 && !slices.Contains(pids, pid) {
			pids = append(pids, pid)
		}
	}

	return pids
}

// leftover is a process being stopped, held by a pidfd so that a reused PID is never signaled in its place,
// or without pidfd support by its start time, checked again before every signal.
type leftover struct {
	pid       int
	starttime uint64
	// fd is -1 without pidfd support.
	fd       int
	readStat func(pid int) (procfs.ProcStat, error)
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

	lo := &leftover{pid: pid, starttime: starttime, fd: fd, readStat: readProcStat}
	// The PID may have been reused since it was listed, so leave alone a process not confirmed as the one inspected.
	if alive, _ := lo.state(); !alive {
		lo.close()
		return nil, errLeftoverGone
	}

	return lo, nil
}

func readProcStat(pid int) (procfs.ProcStat, error) {
	p, err := procfs.NewProc(pid)
	if err != nil {
		return procfs.ProcStat{}, err
	}

	return p.Stat()
}

// state reports whether the process is alive, and whether it is gone; it is neither when /proc cannot be
// read, such as on EMFILE.
func (lo *leftover) state() (bool, bool) {
	stat, err := lo.readStat(lo.pid)
	switch {
	case errors.Is(err, os.ErrNotExist), errors.Is(err, unix.ESRCH):
		return false, true
	case err != nil:
		return false, false
	// Another process has the PID now, or it is a zombie, which has released its locks.
	case stat.Starttime != lo.starttime, stat.State == "Z", stat.State == "X":
		return false, true
	default:
		return true, false
	}
}

func (lo *leftover) signal(sig unix.Signal, l *logrus.Entry) {
	var err error
	if lo.fd >= 0 {
		err = unix.PidfdSendSignal(lo.fd, sig, nil, 0)
	} else if alive, _ := lo.state(); alive {
		err = unix.Kill(lo.pid, sig)
	}
	// ESRCH: it has exited already.
	if err != nil && !errors.Is(err, unix.ESRCH) {
		l.Warnf("Failed to send %s to vmagent PID %d: %s.", unix.SignalName(sig), lo.pid, err)
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

	// Only a process known to be gone has exited: one that cannot be read may still hold the lock.
	for {
		if _, gone := lo.state(); gone {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(leftoverPollInterval)
	}
}

func (lo *leftover) close() {
	if lo.fd >= 0 {
		_ = unix.Close(lo.fd)
		lo.fd = -1
	}
}
