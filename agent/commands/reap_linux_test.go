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
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

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

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLeftoverVMAgentHelper$", "--", //nolint:gosec
		"-remoteWrite.tmpDataPath="+queueDir)
	cmd.Env = append(os.Environ(), leftoverHelperEnv+"="+mode)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)

	return cmd
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
	short := leftoverTimeouts{stop: 300 * time.Millisecond, kill: 5 * time.Second}

	t.Run("stops a vmagent holding this queue with SIGTERM", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cmd := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "")

		stopLeftovers(testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, cmd))
	})

	t.Run("kills one that ignores SIGTERM", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cmd := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "ignore-sigterm")

		stopLeftovers(testExecutable(t), tempDir, short, unix.PidfdOpen, l)
		assert.Equal(t, syscall.SIGKILL, exitedBy(t, cmd))
	})

	// Linux before 5.3 has no pidfd_open.
	t.Run("without pidfd support", func(t *testing.T) {
		t.Parallel()

		noPidfd := func(int, int) (int, error) { return -1, unix.ENOSYS }

		tempDir := t.TempDir()
		term := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "")
		stopLeftovers(testExecutable(t), tempDir, short, noPidfd, l)
		assert.Equal(t, syscall.SIGTERM, exitedBy(t, term))

		tempDir = t.TempDir()
		kill := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "ignore-sigterm")
		stopLeftovers(testExecutable(t), tempDir, short, noPidfd, l)
		assert.Equal(t, syscall.SIGKILL, exitedBy(t, kill))
	})

	t.Run("leaves a vmagent with another queue alone", func(t *testing.T) {
		t.Parallel()

		cmd := startLeftover(t, filepath.Join(t.TempDir(), vmagentQueueDirName), "")

		stopLeftovers(testExecutable(t), t.TempDir(), short, unix.PidfdOpen, l)
		assert.NoError(t, cmd.Process.Signal(syscall.Signal(0)), "the other queue's vmagent must keep running")
	})

	t.Run("does not signal a process that is already gone", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		cmd := startLeftover(t, filepath.Join(tempDir, vmagentQueueDirName), "")
		pid := cmd.Process.Pid
		require.NoError(t, cmd.Process.Kill())
		exitedBy(t, cmd) // reaped: the PID is free for reuse from here on

		lo, err := openLeftover(pid, 0, unix.PidfdOpen)
		assert.Nil(t, lo)
		assert.Error(t, err)
		assert.False(t, errors.Is(err, unix.ENOSYS))
	})
}
