// pmm-agent
// Copyright 2019 Percona LLC
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

package supervisor

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/percona/pmm/api/agentlocalpb"
	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/agents/process"
	"github.com/percona/pmm-agent/config"
)

// assertChanges checks expected changes in any order.
func assertChanges(t *testing.T, s *Supervisor, expected ...agentpb.StateChangedRequest) {
	t.Helper()

	actual := make([]agentpb.StateChangedRequest, len(expected))
	for i := range expected {
		actual[i] = <-s.Changes()
	}

	sort.Slice(expected, func(i, j int) bool { return expected[i].AgentId < expected[j].AgentId })
	sort.Slice(actual, func(i, j int) bool { return actual[i].AgentId < actual[j].AgentId })
	assert.Equal(t, expected, actual)
}

func TestSupervisor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewSupervisor(ctx, nil, &config.Ports{Min: 65000, Max: 65099})

	t.Run("Start13", func(t *testing.T) {
		expectedList := []*agentlocalpb.AgentInfo{}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentpb.SetStateRequest{
			AgentProcesses: map[string]*agentpb.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentpb.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "30"},
			},
		})

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_STARTING},
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_STARTING, ListenPort: 65000},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_STARTING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_STARTING},
		}
		assert.Equal(t, expectedList, s.AgentsList())

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING, ListenPort: 65000},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
		}
		assert.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Restart1Start2", func(t *testing.T) {
		expectedList := []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentpb.SetStateRequest{
			AgentProcesses: map[string]*agentpb.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"20"}},
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentpb.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "30"},
			},
		})

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_STOPPING, ListenPort: 65000},
		)
		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_DONE, ListenPort: 65000},
		)

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_STARTING, ListenPort: 65000},
			agentpb.StateChangedRequest{AgentId: "sleep2", Status: inventorypb.AgentStatus_STARTING, ListenPort: 65001},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_STARTING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_STARTING},
		}
		assert.Equal(t, expectedList, s.AgentsList())

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING, ListenPort: 65000},
			agentpb.StateChangedRequest{AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING, ListenPort: 65001},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		assert.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Restart3Start4", func(t *testing.T) {
		expectedList := []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentpb.SetStateRequest{
			AgentProcesses: map[string]*agentpb.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"20"}},
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentpb.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "20"},
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_STOPPING},
		)
		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_DONE},
		)

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_STARTING},
			agentpb.StateChangedRequest{AgentId: "noop4", Status: inventorypb.AgentStatus_STARTING},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_STARTING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_STARTING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		assert.Equal(t, expectedList, s.AgentsList())

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			agentpb.StateChangedRequest{AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		assert.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Stop1", func(t *testing.T) {
		expectedList := []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentpb.SetStateRequest{
			AgentProcesses: map[string]*agentpb.SetStateRequest_AgentProcess{
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentpb.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "20"},
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_STOPPING, ListenPort: 65000},
		)
		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep1", Status: inventorypb.AgentStatus_DONE, ListenPort: 65000},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Stop3", func(t *testing.T) {
		expectedList := []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentpb.SetStateRequest{
			AgentProcesses: map[string]*agentpb.SetStateRequest_AgentProcess{
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentpb.SetStateRequest_BuiltinAgent{
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_STOPPING},
		)
		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "noop3", Status: inventorypb.AgentStatus_DONE},
		)
		expectedList = []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Exit", func(t *testing.T) {
		expectedList := []*agentlocalpb.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventorypb.AgentStatus_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventorypb.AgentStatus_RUNNING},
		}
		require.Equal(t, expectedList, s.AgentsList())

		cancel()

		assertChanges(t, s,
			agentpb.StateChangedRequest{AgentId: "sleep2", Status: inventorypb.AgentStatus_STOPPING, ListenPort: 65001},
			agentpb.StateChangedRequest{AgentId: "sleep2", Status: inventorypb.AgentStatus_DONE, ListenPort: 65001},
			agentpb.StateChangedRequest{AgentId: "noop4", Status: inventorypb.AgentStatus_STOPPING},
			agentpb.StateChangedRequest{AgentId: "noop4", Status: inventorypb.AgentStatus_DONE},
		)
		assertChanges(t, s,
			agentpb.StateChangedRequest{Status: inventorypb.AgentStatus_AGENT_STATUS_INVALID},
		)
		expectedList = []*agentlocalpb.AgentInfo{}
		require.Equal(t, expectedList, s.AgentsList())
	})
}

func TestFilter(t *testing.T) {
	t.Parallel()

	existingParams := map[string]agentpb.AgentParams{
		"toRestart":  &agentpb.SetStateRequest_AgentProcess{Type: inventorypb.AgentType_NODE_EXPORTER},
		"toStop":     &agentpb.SetStateRequest_AgentProcess{},
		"notChanged": &agentpb.SetStateRequest_AgentProcess{},
	}

	newParams := map[string]agentpb.AgentParams{
		"toStart":    &agentpb.SetStateRequest_AgentProcess{},
		"toRestart":  &agentpb.SetStateRequest_AgentProcess{Type: inventorypb.AgentType_MYSQLD_EXPORTER},
		"notChanged": &agentpb.SetStateRequest_AgentProcess{},
	}
	toStart, toRestart, toStop := filter(existingParams, newParams)
	assert.Equal(t, []string{"toStart"}, toStart)
	assert.Equal(t, []string{"toRestart"}, toRestart)
	assert.Equal(t, []string{"toStop"}, toStop)
}

func TestSupervisorProcessParams(t *testing.T) {
	setup := func(t *testing.T) (*Supervisor, func()) {
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
		s, teardown := setup(t)
		defer teardown()

		p := &agentpb.SetStateRequest_AgentProcess{
			Type: inventorypb.AgentType_MYSQLD_EXPORTER,
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
		actual, err := s.processParams("ID", p, 12345)
		require.NoError(t, err)

		expected := process.Params{
			Path: "/path/to/mysql_exporter",
			Args: []string{
				"-web.listen-address=:12345",
				"-web.ssl-cert-file=" + filepath.Join(s.paths.TempDir, "mysqld_exporter-ID", "Cert"),
			},
			Env: []string{
				"HTTP_AUTH=pmm:secret",
				"TEST=:12345",
			},
		}
		assert.Equal(t, expected, *actual)
	})

	t.Run("BadTemplate", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		defer teardown()

		p := &agentpb.SetStateRequest_AgentProcess{
			Type: inventorypb.AgentType_MYSQLD_EXPORTER,
			Args: []string{"-foo=:{{ .bar }}"},
		}
		_, err := s.processParams("ID", p, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "bar"`, err.Error())

		p = &agentpb.SetStateRequest_AgentProcess{
			Type:      inventorypb.AgentType_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"foo": "{{ .bar }}"},
		}
		_, err = s.processParams("ID", p, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "bar"`, err.Error())

		p = &agentpb.SetStateRequest_AgentProcess{
			Type:      inventorypb.AgentType_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"bar": "{{ .listen_port }}"},
			Args:      []string{"-foo=:{{ .TextFiles.baz }}"},
		}
		_, err = s.processParams("ID", p, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "baz"`, err.Error())
	})

	t.Run("InsecureName", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		defer teardown()

		process := &agentpb.SetStateRequest_AgentProcess{
			Type:      inventorypb.AgentType_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"../bar": "hax0r"},
		}
		_, err := s.processParams("ID", process, 0)
		require.Error(t, err)
		assert.Regexp(t, `invalid text file name "../bar"`, err.Error())
	})
}
