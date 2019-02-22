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
	"path/filepath"
	"sort"
	"testing"

	"github.com/percona/pmm/api/agent"
	"github.com/percona/pmm/api/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/config"
)

// assertChanges checks expected changes in any order.
func assertChanges(t *testing.T, s *Supervisor, expected ...agent.StateChangedRequest) {
	t.Helper()

	actual := make([]agent.StateChangedRequest, len(expected))
	for i := range expected {
		actual[i] = <-s.Changes()
	}

	sort.Slice(expected, func(i, j int) bool { return expected[i].String() < expected[j].String() })
	sort.Slice(actual, func(i, j int) bool { return actual[i].String() < actual[j].String() })
	assert.Equal(t, expected, actual)
}

func TestSupervisor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewSupervisor(ctx, nil, &config.Ports{Min: 10000, Max: 20000})

	t.Run("Start1", func(t *testing.T) {
		s.SetState(map[string]*agent.SetStateRequest_AgentProcess{
			"sleep1": {Type: type_TEST_SLEEP, Args: []string{"10"}},
		})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_STARTING, ListenPort: 10000})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_RUNNING, ListenPort: 10000})
	})

	t.Run("Restart1Start2", func(t *testing.T) {
		s.SetState(map[string]*agent.SetStateRequest_AgentProcess{
			"sleep1": {Type: type_TEST_SLEEP, Args: []string{"20"}},
			"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
		})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_STOPPING, ListenPort: 10000})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_DONE, ListenPort: 10000})

		// the order of those two is not defined
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_STARTING, ListenPort: 10000},
			agent.StateChangedRequest{AgentId: "sleep2", Status: inventory.AgentStatus_STARTING, ListenPort: 10001})

		// the order of those two is not defined
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_RUNNING, ListenPort: 10000},
			agent.StateChangedRequest{AgentId: "sleep2", Status: inventory.AgentStatus_RUNNING, ListenPort: 10001},
		)
	})

	t.Run("Stop1", func(t *testing.T) {
		s.SetState(map[string]*agent.SetStateRequest_AgentProcess{
			"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
		})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_STOPPING, ListenPort: 10000})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep1", Status: inventory.AgentStatus_DONE, ListenPort: 10000})
	})

	t.Run("Exit", func(t *testing.T) {
		cancel()
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep2", Status: inventory.AgentStatus_STOPPING, ListenPort: 10001})
		assertChanges(t, s, agent.StateChangedRequest{AgentId: "sleep2", Status: inventory.AgentStatus_DONE, ListenPort: 10001})
		assertChanges(t, s, agent.StateChangedRequest{Status: inventory.AgentStatus_AGENT_STATUS_INVALID})
	})
}

func TestSupervisorFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewSupervisor(ctx, nil, &config.Ports{Min: 10000, Max: 20000})

	t.Run("Normal", func(t *testing.T) {
		s.agents = map[string]*agentInfo{
			"toRestart": {
				cancel: cancel,
				requestedState: &agent.SetStateRequest_AgentProcess{
					Type: agent.Type_NODE_EXPORTER,
				},
			},
			"toStop": {
				cancel:         cancel,
				requestedState: &agent.SetStateRequest_AgentProcess{},
			},
			"notChanged": {
				cancel:         cancel,
				requestedState: &agent.SetStateRequest_AgentProcess{},
			},
		}

		agentProcesses := map[string]*agent.SetStateRequest_AgentProcess{
			"toStart":    {},
			"toRestart":  {Type: agent.Type_MYSQLD_EXPORTER},
			"notChanged": {},
		}
		toStart, toRestart, toStop := s.filter(agentProcesses)
		assert.Equal(t, []string{"toStart"}, toStart)
		assert.Equal(t, []string{"toRestart"}, toRestart)
		assert.Equal(t, []string{"toStop"}, toStop)
	})
}

func TestSupervisorProcessParams(t *testing.T) {
	setup := func() (*Supervisor, func()) {
		temp, err := ioutil.TempDir("", "pmm-agent-")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		paths := &config.Paths{
			MySQLdExporter: "/path/to/mysql_exporter",
			TempDir:        temp,
		}
		s := NewSupervisor(ctx, paths, new(config.Ports))

		teardown := func() {
			cancel()
			if t.Failed() {
				t.Logf("%s is kept.", paths.TempDir)
			} else {
				require.NoError(t, os.RemoveAll(paths.TempDir))
			}
		}
		return s, teardown
	}

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup()
		defer teardown()

		process := &agent.SetStateRequest_AgentProcess{
			Type: agent.Type_MYSQLD_EXPORTER,
			Args: []string{
				"-web.listen-address=:{{ .listen_port }}",
				"-web.ssl-cert-file={{ .TextFiles.Cert }}",
			},
			Env: []string{
				"HTTP_AUTH=pmm:secret",
				"TEST=:{{ .listen_port }}",
			},
			TextFiles: map[string]string{
				"Cert":   "-----BEGIN CERTIFICATE-----\n...",
				"Config": "test={{ .listen_port }}",
			},
		}
		actual, err := s.processParams("ID", process, 12345)
		require.NoError(t, err)

		expected := processParams{
			path: "/path/to/mysql_exporter",
			args: []string{
				"-web.listen-address=:12345",
				"-web.ssl-cert-file=" + filepath.Join(s.paths.TempDir, "mysqld_exporter-ID", "Cert"),
			},
			env: []string{
				"HTTP_AUTH=pmm:secret",
				"TEST=:12345",
			},
		}
		assert.Equal(t, expected, *actual)
	})

	t.Run("BadTemplate", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup()
		defer teardown()

		process := &agent.SetStateRequest_AgentProcess{
			Type: agent.Type_MYSQLD_EXPORTER,
			Args: []string{"-foo=:{{ .bar }}"},
		}
		_, err := s.processParams("ID", process, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "bar"`, err.Error())

		process = &agent.SetStateRequest_AgentProcess{
			Type:      agent.Type_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"foo": "{{ .bar }}"},
		}
		_, err = s.processParams("ID", process, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "bar"`, err.Error())

		process = &agent.SetStateRequest_AgentProcess{
			Type:      agent.Type_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"bar": "{{ .listen_port }}"},
			Args:      []string{"-foo=:{{ .TextFiles.baz }}"},
		}
		_, err = s.processParams("ID", process, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "baz"`, err.Error())
	})

	t.Run("InsecureName", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup()
		defer teardown()

		process := &agent.SetStateRequest_AgentProcess{
			Type:      agent.Type_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"../bar": "hax0r"},
		}
		_, err := s.processParams("ID", process, 0)
		require.Error(t, err)
		assert.Regexp(t, `invalid text file name "../bar"`, err.Error())
	})
}
