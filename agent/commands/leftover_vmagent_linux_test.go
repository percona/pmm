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
	"bufio"
	"context"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

const leftoverHelperEnv = "PMM_TEST_LEFTOVER_VMAGENT"

// TestLeftoverVMAgentHelper is not a test but the stand-in vmagent: the test binary plays the vmagent binary.
func TestLeftoverVMAgentHelper(t *testing.T) {
	mode := os.Getenv(leftoverHelperEnv)
	if mode == "" {
		t.Skip("helper process for the leftover vmagent tests")
	}
	if mode == "ignore-sigterm" {
		signal.Ignore(syscall.SIGTERM)
	}
	if mode == "lock" {
		queueDir, _ := tmpDataPath(os.Args)
		f, err := os.Create(filepath.Join(queueDir, "flock.lock"))
		require.NoError(t, err)
		require.NoError(t, unix.Flock(int(f.Fd()), unix.LOCK_EX))
		// The finalizer of an unreachable f would close the fd, and with it the lock.
		defer runtime.KeepAlive(f)
	}
	_, _ = os.Stdout.WriteString("ready\n")
	time.Sleep(time.Hour)
}

// startLeftover starts a stand-in vmagent using queueDir, waits until it is up, and returns it.
func startLeftover(t *testing.T, queueDir, mode string) *exec.Cmd {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLeftoverVMAgentHelper$", "--",
		"-remoteWrite.tmpDataPath="+queueDir)
	cmd.Env = append(os.Environ(), leftoverHelperEnv+"="+mode)
	// Not left sleeping if the test binary dies.
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)

	return cmd
}

// newQueue creates a pmm-agent temporary directory with a vmagent queue in it, as vmagent does.
func newQueue(t *testing.T) (string, string) {
	t.Helper()

	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, vmagentQueueDirName)
	require.NoError(t, os.MkdirAll(queueDir, 0o700))

	return tempDir, queueDir
}

// state returns the process state of a child of the test, "Z" once it has exited, as only its Cleanup reaps it.
func state(t *testing.T, pid int) string {
	t.Helper()

	p, err := procfs.NewProc(pid)
	require.NoError(t, err)
	stat, err := p.Stat()
	require.NoError(t, err)

	return stat.State
}

// exitedBy waits for cmd and returns the signal it was killed by, or 0 if it was not.
func exitedBy(t *testing.T, cmd *exec.Cmd) syscall.Signal {
	t.Helper()

	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		// Waited for here, as a second Wait in startLeftover's Cleanup would never return while this one waits.
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("process is still running")
	}

	status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0
	}

	return status.Signal()
}

func testExecutable(t *testing.T) string {
	t.Helper()

	exe, err := os.Executable()
	require.NoError(t, err)

	return exe
}

func TestStopLeftoverVMAgents(t *testing.T) {
	t.Parallel()

	l := logrus.WithField("test", t.Name())
	short := leftoverTimeouts{budget: 15 * time.Second, stop: 300 * time.Millisecond, kill: 5 * time.Second}
	// The stand-ins start after the test binary, which plays pmm-agent, so any start time counts as earlier.
	const anyStart = uint64(math.MaxUint64)

	// Subtests check that the process has exited by the time stopLeftovers returns, as only that releases the lock.
	t.Run("stops a vmagent holding this queue with SIGTERM", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "run")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftovers returned")
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, cmd))
	})

	t.Run("kills one that ignores SIGTERM", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "ignore-sigterm")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftovers returned")
		assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
	})

	// The wait after SIGKILL is shared too, which no test process can outlast to show.
	t.Run("kills several that ignore SIGTERM after one shared wait", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		first := startLeftover(t, queueDir, "ignore-sigterm")
		second := startLeftover(t, queueDir, "ignore-sigterm")
		timeouts := leftoverTimeouts{budget: 15 * time.Second, stop: time.Second, kill: 5 * time.Second}

		start := time.Now()
		stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, timeouts, unix.PidfdOpen, l)
		assert.Less(t, time.Since(start), 2*timeouts.stop-50*time.Millisecond, "each had a wait after SIGTERM of its own")
		for _, cmd := range []*exec.Cmd{first, second} {
			assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftovers returned")
			assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
		}
	})

	t.Run("waits for a search that finishes", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "ignore-sigterm")

		stopLeftoversWithin(t.Context(), testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftoversWithin returned")
		assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
	})

	// A blocking pidfd_open stands in for a /proc read stuck in the kernel.
	t.Run("does not wait for a search that is stuck, which still stops the leftover", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "ignore-sigterm")

		stuck := make(chan struct{})
		stuckOpen := func(pid, flags int) (int, error) {
			<-stuck
			return unix.PidfdOpen(pid, flags)
		}
		timeouts := leftoverTimeouts{budget: 300 * time.Millisecond, stop: 100 * time.Millisecond, kill: 100 * time.Millisecond}

		start := time.Now()
		done := stopLeftoversWithin(t.Context(), testExecutable(t), tempDir, anyStart, timeouts, stuckOpen, l)
		assert.Less(t, time.Since(start), 5*time.Second)
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid))

		close(stuck)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("the search did not finish")
		}
		assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
	})

	t.Run("signals nothing once pmm-agent is shutting down", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "run")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		stopLeftovers(ctx, testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		// Long enough for a SIGTERM to take effect.
		time.Sleep(short.stop)
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid), "signaled after shutdown")
	})

	t.Run("leaves a vmagent started after this pmm-agent alone", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "run")
		p, err := procfs.NewProc(cmd.Process.Pid)
		require.NoError(t, err)
		stat, err := p.Stat()
		require.NoError(t, err)

		stopLeftovers(t.Context(), testExecutable(t), tempDir, stat.Starttime, short, unix.PidfdOpen, l)
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid), "pmm-agent's own vmagent must keep running")
	})

	// Without a pidfd only /proc tells whether the process is gone, and a read failing on EMFILE does not.
	t.Run("does not take a process it cannot read for gone", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "run")
		p, err := procfs.NewProc(cmd.Process.Pid)
		require.NoError(t, err)
		stat, err := p.Stat()
		require.NoError(t, err)

		unreadable := &leftover{pid: cmd.Process.Pid, starttime: stat.Starttime, fd: -1, readStat: func(int) (procfs.ProcStat, error) {
			return procfs.ProcStat{}, unix.EMFILE
		}}
		unreadable.signal(unix.SIGTERM, l)
		assert.False(t, unreadable.waitExit(300*time.Millisecond), "taken for gone")
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid), "signaled without being confirmed")

		gone := &leftover{pid: cmd.Process.Pid, starttime: stat.Starttime, fd: -1, readStat: func(int) (procfs.ProcStat, error) {
			return procfs.ProcStat{}, &os.PathError{Op: "open", Path: "/proc/1/stat", Err: unix.ENOENT}
		}}
		assert.True(t, gone.waitExit(time.Minute))
	})

	// Linux before 5.3 has no pidfd_open; a seccomp profile older than pidfd refuses it.
	t.Run("without pidfd", func(t *testing.T) {
		t.Parallel()

		for _, errno := range []unix.Errno{unix.ENOSYS, unix.EPERM} {
			noPidfd := func(int, int) (int, error) { return -1, errno }

			tempDir, queueDir := newQueue(t)
			term := startLeftover(t, queueDir, "run")
			stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, noPidfd, l)
			assert.Equal(t, "Z", state(t, term.Process.Pid), "%s: still running when stopLeftovers returned", errno)
			assert.Equal(t, syscall.SIGTERM, exitedBy(t, term))

			tempDir, queueDir = newQueue(t)
			kill := startLeftover(t, queueDir, "ignore-sigterm")
			stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, noPidfd, l)
			assert.Equal(t, "Z", state(t, kill.Process.Pid), "%s: still running when stopLeftovers returned", errno)
			assert.Equal(t, syscall.SIGKILL, exitedBy(t, kill))
		}
	})

	// Compared as files, so a queue reached through a symlink is the same queue.
	t.Run("stops one holding this queue under another spelling", func(t *testing.T) {
		t.Parallel()

		tempDir, _ := newQueue(t)
		link := filepath.Join(t.TempDir(), "link")
		require.NoError(t, os.Symlink(tempDir, link))
		cmd := startLeftover(t, filepath.Join(link, vmagentQueueDirName), "run")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, cmd))
	})

	t.Run("stops one running the binary a configured symlink points to", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "run")
		vmagent := filepath.Join(t.TempDir(), "vmagent")
		require.NoError(t, os.Symlink(testExecutable(t), vmagent))

		stopLeftovers(t.Context(), vmagent, tempDir, anyStart, short, unix.PidfdOpen, l)
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, cmd))
	})

	t.Run("leaves a vmagent with another queue alone", func(t *testing.T) {
		t.Parallel()

		_, otherQueue := newQueue(t)
		cmd := startLeftover(t, otherQueue, "run")
		tempDir, _ := newQueue(t)

		stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		// Not Signal(0), which also succeeds on a zombie nobody has reaped yet.
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid), "the other queue's vmagent must keep running")
	})

	t.Run("does nothing before the queue exists", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cmd := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "run")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, anyStart, short, unix.PidfdOpen, l)
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid))
	})

	// The PID may be reused between listing a process and opening it.
	t.Run("does not take another process for the one listed", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "run")
		p, err := procfs.NewProc(cmd.Process.Pid)
		require.NoError(t, err)
		stat, err := p.Stat()
		require.NoError(t, err)

		lo, err := openLeftover(cmd.Process.Pid, stat.Starttime+1, unix.PidfdOpen, l)
		assert.Nil(t, lo)
		require.ErrorIs(t, err, errLeftoverGone)
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid))

		lo, err = openLeftover(cmd.Process.Pid, stat.Starttime, unix.PidfdOpen, l)
		require.NoError(t, err)
		lo.close()
	})
}

func TestIsVMAgentExecutable(t *testing.T) {
	t.Parallel()

	const vmagent = "/usr/local/percona/pmm/exporters/vmagent"

	for _, tc := range []struct {
		name     string
		exe      string
		expected bool
	}{
		{"this pmm-agent's vmagent", vmagent, true},
		{"path not clean", "/usr/local/percona/pmm/exporters//vmagent", true},
		{"binary replaced since it started", vmagent + " (deleted)", true},

		{"another vmagent binary", "/usr/bin/vmagent", false},
		{"another exporter", "/usr/local/percona/pmm/exporters/node_exporter", false},
		{"a sibling sharing the prefix", vmagent + "-old", false},
		{"no executable", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, isVMAgentExecutable(tc.exe, vmagent))
		})
	}

	t.Run("nothing configured", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isVMAgentExecutable(vmagent, ""))
	})
}

func TestComm(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "vmagent", comm("/usr/local/percona/pmm/exporters/vmagent"))
	assert.Equal(t, "vmagent-1234567", comm("/opt/vmagent-1234567"))
	assert.Equal(t, "vmagent-1234567", comm("/opt/vmagent-12345678"))
}

func TestTmpDataPath(t *testing.T) {
	t.Parallel()

	const queue = "/usr/local/percona/pmm/tmp/vmagent-temp-dir"
	args := func(rest ...string) []string {
		return append([]string{"/usr/local/percona/pmm/exporters/vmagent"}, rest...)
	}

	for _, tc := range []struct {
		name     string
		cmdline  []string
		expected string
	}{
		{"the flag pmm-agent passes", args("-envflag.enable=true", "-remoteWrite.tmpDataPath="+queue), queue},
		{"double-dash flag", args("--remoteWrite.tmpDataPath=" + queue), queue},
		{"value in the next argument", args("-remoteWrite.tmpDataPath", queue, "-envflag.enable=true"), queue},
		{"double-dash flag, value in the next argument", args("--remoteWrite.tmpDataPath", queue), queue},
		{"last of several", args("-remoteWrite.tmpDataPath=/tmp/other", "-remoteWrite.tmpDataPath", queue), queue},

		{"queue only as a value of another flag", args("-promscrape.config=" + queue), ""},
		{"queue as a positional argument", args("remoteWrite.tmpDataPath=" + queue), ""},
		{"empty flag value", args("-remoteWrite.tmpDataPath="), ""},
		{"empty last of several", args("-remoteWrite.tmpDataPath="+queue, "-remoteWrite.tmpDataPath="), ""},
		{"no value after the flag", args("-remoteWrite.tmpDataPath"), ""},
		{"no queue flag", args("-envflag.enable=true"), ""},
		{"no arguments", args(), ""},
		{"no command line", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path, ok := tmpDataPath(tc.cmdline)
			assert.Equal(t, tc.expected, path)
			assert.Equal(t, tc.expected != "", ok)
		})
	}
}

func TestQueueLockHolders(t *testing.T) {
	t.Parallel()

	t.Run("finds the process holding the queue lock", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "lock")

		assert.Equal(t, []int{cmd.Process.Pid}, queueLockHolders(queueDir))
	})

	t.Run("finds none without a lock", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		startLeftover(t, queueDir, "run")

		assert.Empty(t, queueLockHolders(queueDir))
	})

	t.Run("warns once about a holder that is not a child of this pmm-agent", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "lock")
		logger, hook := logrustest.NewNullLogger()
		l := logrus.NewEntry(logger)
		reported := make(map[int]bool)

		// The stand-in is this test's child, so a self of 0 makes it a stranger's.
		warnQueueLockHolders(queueDir, 0, reported, l)
		warnQueueLockHolders(queueDir, 0, reported, l)

		require.Len(t, hook.AllEntries(), 1)
		assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
		assert.Contains(t, hook.LastEntry().Message, "PID "+strconv.Itoa(cmd.Process.Pid)+" (")
		assert.Contains(t, hook.LastEntry().Message, queueDir)
	})

	t.Run("leaves the vmagent of this pmm-agent alone", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		startLeftover(t, queueDir, "lock")
		logger, hook := logrustest.NewNullLogger()

		warnQueueLockHolders(queueDir, os.Getpid(), make(map[int]bool), logrus.NewEntry(logger))

		assert.Empty(t, hook.AllEntries())
	})
}
