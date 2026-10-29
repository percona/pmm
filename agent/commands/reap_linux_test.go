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
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

const leftoverHelperEnv = "PMM_TEST_LEFTOVER_VMAGENT"

// TestLeftoverVMAgentHelper is not a test: it is the stand-in vmagent the tests below run, the test
// binary itself started with a -remoteWrite.tmpDataPath flag. The test binary plays the configured
// vmagent binary, so the real executable check is exercised as it is.
func TestLeftoverVMAgentHelper(t *testing.T) {
	mode := os.Getenv(leftoverHelperEnv)
	if mode == "" {
		t.Skip("helper process for the leftover vmagent tests")
	}
	if mode == "ignore-sigterm" {
		signal.Ignore(syscall.SIGTERM)
	}
	_, _ = os.Stdout.WriteString("ready\n")
	time.Sleep(time.Hour)
}

// startLeftover starts a stand-in vmagent using queueDir, waits until it is up, and returns it.
func startLeftover(t *testing.T, queueDir, mode string) *exec.Cmd {
	t.Helper()

	if mode == "" {
		mode = "run" // an empty one tells the helper it is not one
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLeftoverVMAgentHelper$", "--",
		"-remoteWrite.tmpDataPath="+queueDir)
	cmd.Env = append(os.Environ(), leftoverHelperEnv+"="+mode)
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL} // not left sleeping if the test binary dies
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

// state returns the process state of a child of the test: "Z" once it has exited, as nothing reaps
// it before its Cleanup does.
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
	short := leftoverTimeouts{scan: 5 * time.Second, stop: 300 * time.Millisecond, kill: 5 * time.Second}

	// Each subtest asserts the process has exited the moment stopLeftovers returns, without waiting:
	// the lock is released only then, and pmm-agent starts its own vmagent right after.
	t.Run("stops a vmagent holding this queue with SIGTERM", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftovers returned")
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, cmd))
	})

	t.Run("kills one that ignores SIGTERM", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "ignore-sigterm")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftovers returned")
		assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
	})

	// They share the wait after SIGTERM, and the one after SIGKILL too, which no test process can
	// outlast to show.
	t.Run("kills several that ignore SIGTERM, within one budget", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		first := startLeftover(t, queueDir, "ignore-sigterm")
		second := startLeftover(t, queueDir, "ignore-sigterm")
		timeouts := leftoverTimeouts{scan: 5 * time.Second, stop: time.Second, kill: 5 * time.Second}

		start := time.Now()
		stopLeftovers(t.Context(), testExecutable(t), tempDir, timeouts, unix.PidfdOpen, l)
		assert.Less(t, time.Since(start), 2*timeouts.stop-200*time.Millisecond, "each waited out the SIGTERM budget on its own")
		for _, cmd := range []*exec.Cmd{first, second} {
			assert.Equal(t, "Z", state(t, cmd.Process.Pid), "still running when stopLeftovers returned")
			assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
		}
	})

	// Reading /proc can block on a process stuck in the kernel. pmm-agent starts anyway, and the
	// stuck search signals nothing when it gets going again: pmm-agent may have a vmagent of its own
	// by then.
	t.Run("does not wait for a search that is stuck, nor let it signal later", func(t *testing.T) {
		t.Parallel()

		tempDir, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "")

		stuck := make(chan struct{})
		stuckOpen := func(pid, flags int) (int, error) {
			<-stuck
			return unix.PidfdOpen(pid, flags)
		}
		timeouts := leftoverTimeouts{scan: 100 * time.Millisecond, stop: 100 * time.Millisecond, kill: 100 * time.Millisecond}

		start := time.Now()
		done := stopLeftoversWithin(t.Context(), testExecutable(t), tempDir, timeouts, stuckOpen, l)
		assert.Less(t, time.Since(start), 5*time.Second)

		close(stuck)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("the search did not finish")
		}
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid), "signaled after it was given up on")
	})

	// Linux before 5.3 has no pidfd_open; a seccomp profile older than pidfd refuses it.
	t.Run("without pidfd", func(t *testing.T) {
		t.Parallel()

		for _, errno := range []unix.Errno{unix.ENOSYS, unix.EPERM} {
			noPidfd := func(int, int) (int, error) { return -1, errno }

			tempDir, queueDir := newQueue(t)
			term := startLeftover(t, queueDir, "")
			stopLeftovers(t.Context(), testExecutable(t), tempDir, short, noPidfd, l)
			assert.Equal(t, "Z", state(t, term.Process.Pid), "%s: still running when stopLeftovers returned", errno)
			assert.Equal(t, syscall.SIGTERM, exitedBy(t, term))

			tempDir, queueDir = newQueue(t)
			kill := startLeftover(t, queueDir, "ignore-sigterm")
			stopLeftovers(t.Context(), testExecutable(t), tempDir, short, noPidfd, l)
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
		cmd := startLeftover(t, filepath.Join(link, vmagentQueueDirName), "")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, cmd))
	})

	t.Run("leaves a vmagent with another queue alone", func(t *testing.T) {
		t.Parallel()

		_, otherQueue := newQueue(t)
		cmd := startLeftover(t, otherQueue, "")
		tempDir, _ := newQueue(t)

		stopLeftovers(t.Context(), testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		// Not Signal(0), which also succeeds on a zombie nobody has reaped yet.
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid), "the other queue's vmagent must keep running")
	})

	t.Run("does nothing before the queue exists", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cmd := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "")

		stopLeftovers(t.Context(), testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		assert.NotEqual(t, "Z", state(t, cmd.Process.Pid))
	})

	// The PID may be reused between listing a process and opening it.
	t.Run("does not take another process for the one listed", func(t *testing.T) {
		t.Parallel()

		_, queueDir := newQueue(t)
		cmd := startLeftover(t, queueDir, "")
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
