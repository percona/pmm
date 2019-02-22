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
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/percona/pmm/api/inventory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// assertStates checks expected statuses in the same order.
func assertStates(t *testing.T, sa *process, expected ...inventory.AgentStatus) {
	t.Helper()

	actual := make([]inventory.AgentStatus, len(expected))
	for i := range expected {
		actual[i] = <-sa.Changes()
	}
	assert.Equal(t, expected, actual)
}

// builds helper app.
func build(t *testing.T, tag string, fileName string, outputFile string) *exec.Cmd {
	t.Helper()

	t.Logf("building to %s", outputFile)
	args := []string{"build"}
	if tag != "" {
		args = append(args, "-tags", tag)
	}
	args = append(args, "-o", outputFile, fileName)
	cmd := exec.Command("go", args...) //nolint:gosec
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), "failed to build %s", fileName)
	return cmd
}

func setup(t *testing.T) (context.Context, context.CancelFunc, *logrus.Entry) {
	ctx, cancel := context.WithCancel(context.Background())
	l := logrus.WithField("test", t.Name())
	return ctx, cancel, l
}

func TestProcess(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{"100500"}}, l)

		assertStates(t, p, inventory.AgentStatus_STARTING, inventory.AgentStatus_RUNNING)
		cancel()
		assertStates(t, p, inventory.AgentStatus_STOPPING, inventory.AgentStatus_DONE, inventory.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("FailedToStart", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: "no_such_command"}, l)

		assertStates(t, p, inventory.AgentStatus_STARTING, inventory.AgentStatus_WAITING, inventory.AgentStatus_STARTING, inventory.AgentStatus_WAITING)
		cancel()
		assertStates(t, p, inventory.AgentStatus_DONE, inventory.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("ExitedEarly", func(t *testing.T) {
		t.Parallel()
		sleep := strconv.FormatFloat(runningT.Seconds()-0.5, 'f', -1, 64)

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{sleep}}, l)

		assertStates(t, p, inventory.AgentStatus_STARTING, inventory.AgentStatus_WAITING, inventory.AgentStatus_STARTING, inventory.AgentStatus_WAITING)
		cancel()
		assertStates(t, p, inventory.AgentStatus_DONE, inventory.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("CancelStarting", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		sleep := strconv.FormatFloat(runningT.Seconds()-0.5, 'f', -1, 64)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{sleep}}, l)

		assertStates(t, p, inventory.AgentStatus_STARTING, inventory.AgentStatus_WAITING, inventory.AgentStatus_STARTING)
		cancel()
		assertStates(t, p, inventory.AgentStatus_WAITING, inventory.AgentStatus_DONE, inventory.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("Exited", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		sleep := strconv.FormatFloat(runningT.Seconds()+0.5, 'f', -1, 64)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{sleep}}, l)

		assertStates(t, p, inventory.AgentStatus_STARTING, inventory.AgentStatus_RUNNING, inventory.AgentStatus_WAITING)
		cancel()
		assertStates(t, p, inventory.AgentStatus_DONE, inventory.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("Kill child", func(t *testing.T) {
		t.Parallel()

		f, err := ioutil.TempFile("", "pmm-agent-process-test-child")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() {
			require.NoError(t, os.Remove(f.Name()))
		}()

		build(t, "child", "process_child.go", f.Name())

		ctx, cancel, l := setup(t)
		defer cancel()

		logger := newProcessLogger(l, 2)

		pCmd := exec.CommandContext(ctx, f.Name()) //nolint:gosec
		pCmd.Stdout = logger
		err = pCmd.Start()
		require.NoError(t, err)

		var logs []string
		for ; len(logs) == 0; logs = logger.Latest() {
			time.Sleep(50 * time.Millisecond)
		}
		err = pCmd.Process.Kill()
		require.NoError(t, err)
		err = pCmd.Wait()
		require.EqualError(t, err, "signal: killed")
		time.Sleep(200 * time.Millisecond) // Waiting to be sure that child process is killed.

		pid, err := strconv.Atoi(logs[0])
		require.NoError(t, err)
		proc, err := os.FindProcess(pid)
		require.NoError(t, err)

		err = pCmd.Process.Signal(unix.Signal(0))
		require.EqualError(t, err, "os: process already finished", "process with pid %v is not killed", pCmd.Process.Pid)

		err = proc.Signal(unix.Signal(0))
		require.EqualError(t, err, "os: process already finished", "child process with pid %v is not killed", logs[0])
	})

	t.Run("Killed", func(t *testing.T) {
		t.Parallel()

		f, err := ioutil.TempFile("", "pmm-agent-process-test-noterm")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() {
			require.NoError(t, os.Remove(f.Name()))
		}()

		build(t, "", "process_noterm.go", f.Name())

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: f.Name()}, l)

		assertStates(t, p, inventory.AgentStatus_STARTING, inventory.AgentStatus_RUNNING)
		cancel()
		assertStates(t, p, inventory.AgentStatus_STOPPING, inventory.AgentStatus_DONE, inventory.AgentStatus_AGENT_STATUS_INVALID)
	})
}
