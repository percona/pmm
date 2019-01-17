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
	"testing"

	"github.com/percona/pmm/api/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm-agent/config"
)

func TestSupervisorFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := NewSupervisor(ctx, nil, new(config.Ports))

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
				"-web.listen-address=:{{ .ListenPort }}",
				"-web.ssl-cert-file={{ .TextFiles.Cert }}",
			},
			Env: []string{
				"HTTP_AUTH=pmm:secret",
				"TEST=:{{ .ListenPort }}",
			},
			TextFiles: map[string]string{
				"Cert":   "-----BEGIN CERTIFICATE-----\n...",
				"Config": "test={{ .ListenPort }}",
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
			TextFiles: map[string]string{"bar": "{{ .ListenPort }}"},
			Args:      []string{"-foo=:{{ .TextFiles.baz }}"},
		}
		_, err = s.processParams("ID", process, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "baz"`, err.Error())
	})
}
