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

package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	"github.com/percona/pmm/api/inventorypb"
)

// assertStates checks expected statuses in the same order.
func assertStates(t *testing.T, sa *Process, expected ...inventorypb.AgentStatus) {
	t.Helper()

	actual := make([]inventorypb.AgentStatus, len(expected))
	for i := range expected {
		actual[i] = <-sa.Changes()
	}
	assert.Equal(t, expected, actual)
}

// builds helper app.
func build(t *testing.T, tag string, fileName string, outputFile string) *exec.Cmd { //nolint:unparam
	t.Helper()

	t.Logf("building to %s", outputFile)
	args := []string{"build"}
	if tag != "" {
		args = append(args, "-tags", tag)
	}
	args = append(args, "-o", outputFile, fileName)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), "failed to build %s", fileName)
	return cmd
}

func setup(t *testing.T) (context.Context, context.CancelFunc, *logrus.Entry) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	l := logrus.WithField("test", t.Name())
	return ctx, cancel, l
}

func TestProcess(t *testing.T) {
	t.Run("Normal", func(t *testing.T) {
		ctx, cancel, l := setup(t)
		p := New(&Params{Path: "sleep", Args: []string{"100500"}}, nil, l)
		go p.Run(ctx)

		assertStates(t, p, inventorypb.AgentStatus_STARTING, inventorypb.AgentStatus_RUNNING)
		cancel()
		assertStates(t, p, inventorypb.AgentStatus_STOPPING, inventorypb.AgentStatus_DONE, inventorypb.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("FailedToStart", func(t *testing.T) {
		ctx, _, l := setup(t)
		p := New(&Params{Path: "no_such_command"}, nil, l)
		go p.Run(ctx)

		assertStates(t, p, inventorypb.AgentStatus_STARTING, inventorypb.AgentStatus_INITIALIZATION_ERROR,
			inventorypb.AgentStatus_DONE, inventorypb.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("ExitedEarly", func(t *testing.T) {
		sleep := strconv.FormatFloat(runningT.Seconds()-0.5, 'f', -1, 64)
		ctx, _, l := setup(t)
		p := New(&Params{Path: "sleep", Args: []string{sleep}}, nil, l)
		go p.Run(ctx)

		assertStates(t, p, inventorypb.AgentStatus_STARTING, inventorypb.AgentStatus_INITIALIZATION_ERROR,
			inventorypb.AgentStatus_DONE, inventorypb.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("Exited", func(t *testing.T) {
		sleep := strconv.FormatFloat(runningT.Seconds()+0.5, 'f', -1, 64)
		ctx, cancel, l := setup(t)
		p := New(&Params{Path: "sleep", Args: []string{sleep}}, nil, l)
		go p.Run(ctx)

		assertStates(t, p, inventorypb.AgentStatus_STARTING, inventorypb.AgentStatus_RUNNING, inventorypb.AgentStatus_WAITING)
		cancel()
		assertStates(t, p, inventorypb.AgentStatus_DONE, inventorypb.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("Killed", func(t *testing.T) {
		f, err := os.CreateTemp("", "pmm-agent-process-test-noterm")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() {
			require.NoError(t, os.Remove(f.Name()))
		}()

		build(t, "", "process_noterm.go", f.Name())

		ctx, cancel, l := setup(t)
		p := New(&Params{Path: f.Name()}, nil, l)
		go p.Run(ctx)

		assertStates(t, p, inventorypb.AgentStatus_STARTING, inventorypb.AgentStatus_RUNNING)
		cancel()
		assertStates(t, p, inventorypb.AgentStatus_STOPPING, inventorypb.AgentStatus_DONE, inventorypb.AgentStatus_AGENT_STATUS_INVALID)
	})

	t.Run("KillChild", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Pdeathsig is implemented only on Linux")
		}

		f, err := os.CreateTemp("", "pmm-agent-process-test-child")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() {
			require.NoError(t, os.Remove(f.Name()))
		}()

		build(t, "child", "process_child.go", f.Name())

		ctx, cancel, l := setup(t)
		defer cancel()

		logger := newProcessLogger(l, 2, nil)

		pCmd := exec.CommandContext(ctx, f.Name()) //nolint:gosec
		pCmd.Stdout = logger
		err = pCmd.Start()
		require.NoError(t, err)

		var logs []string
		for ; len(logs) == 0; logs = logger.Latest() {
			time.Sleep(50 * time.Millisecond)
		}
		pid, err := strconv.Atoi(logs[0])
		require.NoError(t, err)

		err = pCmd.Process.Kill()
		require.NoError(t, err)
		err = pCmd.Wait()
		require.EqualError(t, err, "signal: killed")
		time.Sleep(200 * time.Millisecond) // Waiting to be sure that child process is killed.

		proc, err := os.FindProcess(pid)
		require.NoError(t, err)

		err = pCmd.Process.Signal(unix.Signal(0))
		require.EqualError(t, err, "os: process already finished", "process with pid %v is not killed", pCmd.Process.Pid)

		err = proc.Signal(unix.Signal(0))
		require.EqualError(t, err, "os: process already finished", "child process with pid %v is not killed", pid)
	})
}

func TestExtractLogLevel(t *testing.T) {
	tests := []struct {
		testName      string
		line          string
		expectedLevel logrus.Level
		expectedFound bool
		expectedErr   error
	}{
		{"info", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492 level=info msg="Starting mysqld_exporter"`, logrus.InfoLevel, true, nil},
		{"panic", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492 level=panic msg="Starting mysqld_exporter"`, logrus.PanicLevel, true, nil},
		{"trace", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492 level=trace msg="Starting mysqld_exporter"`, logrus.TraceLevel, true, nil},
		{"duplicate level key", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492 level=warn msg="Starting mysqld_exporter" duplicate=" level=debug "`, logrus.WarnLevel, true, nil},
		{"missing level key", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492             msg="Starting mysqld_exporter"`, 0, false, nil},
		{"level key with empty value", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492 level= msg="Starting mysqld_exporter"`, 0, false, nil},
		{"level key with incorrect value", `ts=2022-06-14T21:43:42.984Z caller=mysqld_exporter.go:492 level=info123 msg="Starting mysqld_exporter"`, 0, false, fmt.Errorf(`not a valid logrus Level: "info123"`)},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			level, found, err := extractLogLevel(tt.line)

			require.Equal(t, tt.expectedLevel, level)
			require.Equal(t, tt.expectedFound, found)

			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
