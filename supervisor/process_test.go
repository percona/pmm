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

	"github.com/percona/pmm/api/agent"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertStates(t *testing.T, sa *process, expected ...agent.Status) {
	t.Helper()

	actual := make([]agent.Status, len(expected))
	for i := range expected {
		actual[i] = <-sa.Changes()
	}
	assert.Equal(t, expected, actual)
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

		assertStates(t, p, agent.Status_STARTING, agent.Status_RUNNING)
		cancel()
		assertStates(t, p, agent.Status_STOPPING, agent.Status_DONE, agent.Status_STATUS_INVALID)
	})

	t.Run("FailedToStart", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: "no_such_command"}, l)

		assertStates(t, p, agent.Status_STARTING, agent.Status_WAITING, agent.Status_STARTING, agent.Status_WAITING)
		cancel()
		assertStates(t, p, agent.Status_DONE, agent.Status_STATUS_INVALID)
	})

	t.Run("ExitedEarly", func(t *testing.T) {
		t.Parallel()
		sleep := strconv.FormatFloat(runningT.Seconds()-0.5, 'f', -1, 64)

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{sleep}}, l)

		assertStates(t, p, agent.Status_STARTING, agent.Status_WAITING, agent.Status_STARTING, agent.Status_WAITING)
		cancel()
		assertStates(t, p, agent.Status_DONE, agent.Status_STATUS_INVALID)
	})

	t.Run("CancelStarting", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		sleep := strconv.FormatFloat(runningT.Seconds()-0.5, 'f', -1, 64)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{sleep}}, l)

		assertStates(t, p, agent.Status_STARTING, agent.Status_WAITING, agent.Status_STARTING)
		cancel()
		assertStates(t, p, agent.Status_WAITING, agent.Status_DONE, agent.Status_STATUS_INVALID)
	})

	t.Run("Exited", func(t *testing.T) {
		t.Parallel()

		ctx, cancel, l := setup(t)
		sleep := strconv.FormatFloat(runningT.Seconds()+0.5, 'f', -1, 64)
		p := newProcess(ctx, &processParams{path: "sleep", args: []string{sleep}}, l)

		assertStates(t, p, agent.Status_STARTING, agent.Status_RUNNING, agent.Status_WAITING)
		cancel()
		assertStates(t, p, agent.Status_DONE, agent.Status_STATUS_INVALID)
	})

	t.Run("Killed", func(t *testing.T) {
		t.Parallel()

		f, err := ioutil.TempFile("", "pmm-agent-process-test-noterm")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		defer func() {
			require.NoError(t, os.Remove(f.Name()))
		}()

		t.Logf("building to %s", f.Name())
		cmd := exec.Command("go", "build", "-o", f.Name(), "process_noterm.go") //nolint:gosec
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		require.NoError(t, cmd.Run(), "failed to build process_noterm.go")

		ctx, cancel, l := setup(t)
		p := newProcess(ctx, &processParams{path: f.Name()}, l)

		assertStates(t, p, agent.Status_STARTING, agent.Status_RUNNING)
		cancel()
		assertStates(t, p, agent.Status_STOPPING, agent.Status_DONE, agent.Status_STATUS_INVALID)
	})
}
