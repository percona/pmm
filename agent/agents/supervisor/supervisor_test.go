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

package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/agents/process"
	"github.com/percona/pmm/agent/config"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	agentlocal "github.com/percona/pmm/api/agentlocal/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
)

// assertChanges checks expected changes in any order.
func assertChanges(t *testing.T, s *Supervisor, expected ...*agentv1.StateChangedRequest) {
	t.Helper()

	actual := make([]*agentv1.StateChangedRequest, len(expected))
	for i := range expected {
		actual[i] = <-s.Changes()
	}

	require.ElementsMatch(t, expected, actual)
}

func TestSupervisor(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	tempDir := t.TempDir()
	cfgStorage := config.NewStorage(&config.Config{
		Paths:         config.Paths{TempDir: tempDir},
		Ports:         config.Ports{Min: 65000, Max: 65099},
		Server:        config.Server{Address: "localhost:8443"},
		LogLinesCount: 1,
	})
	s := NewSupervisor(ctx, nil, cfgStorage)
	go s.Run(ctx)

	t.Run("Start13", func(t *testing.T) {
		require.Empty(t, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "30"},
			},
		})

		assertChanges(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		)

		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		}
		assert.ElementsMatch(t, expectedList, s.AgentsList())

		assertChanges(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
		)

		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		}
		assert.ElementsMatch(t, expectedList, s.AgentsList())
	})

	t.Run("Restart1Start2", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"20"}},
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "30"},
			},
		})

		assertChanges(
			t, s,
			// stop sleep1 agent
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65000, ProcessExecPath: "sleep"},
			// start sleep1 agent
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			// start sleep2 agent
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
			// nothing for noop3 agent is expected
		)

		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		assert.ElementsMatch(t, expectedList, s.AgentsList())
	})

	t.Run("Restart3Start4", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"20"}},
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "20"},
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(
			t, s,
			// stop noop3 agent
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE},
			// start noop3 agent
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			// start noop4 agent
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			// noop3 and noop4 are running
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			// nothing for sleep1 and sleep2 is expected
		)

		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		assert.ElementsMatch(t, expectedList, s.AgentsList())
	})

	t.Run("Stop1", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "20"},
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(
			t, s,
			// sleep1 agent is terminated
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65000, ProcessExecPath: "sleep"},
		)
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())
	})

	t.Run("Stop3", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(
			t, s,
			// noop3 agent is terminated
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE},
		)
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())
	})

	t.Run("Exit", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.ElementsMatch(t, expectedList, s.AgentsList())

		cancel()

		assertChanges(
			t, s,
			// all agents are terminated
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE},
		)
		require.Empty(t, s.AgentsList())
	})
}

func TestStartProcessFail(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tempDir := t.TempDir()
	cfgStorage := config.NewStorage(&config.Config{
		Paths:         config.Paths{TempDir: tempDir},
		Ports:         config.Ports{Min: 65100, Max: 65199},
		Server:        config.Server{Address: "localhost:443"},
		LogLinesCount: 1,
	})
	s := NewSupervisor(ctx, nil, cfgStorage)
	go s.Run(ctx)

	t.Run("Start", func(t *testing.T) {
		require.Empty(t, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"wrong format"}},
			},
		})

		assertChanges(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65101, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, ListenPort: 65101, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65101, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65102, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, ListenPort: 65102, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65102, ProcessExecPath: "sleep"},
		)
		require.Empty(t, s.AgentsList())
	})
}

func TestFilter(t *testing.T) {
	t.Parallel()

	existingParams := map[string]agentv1.AgentParams{
		"toRestart":  &agentv1.SetStateRequest_AgentProcess{Type: inventoryv1.AgentType_AGENT_TYPE_NODE_EXPORTER},
		"toStop":     &agentv1.SetStateRequest_AgentProcess{},
		"notChanged": &agentv1.SetStateRequest_AgentProcess{},
	}

	newParams := map[string]agentv1.AgentParams{
		"toStart":    &agentv1.SetStateRequest_AgentProcess{},
		"toRestart":  &agentv1.SetStateRequest_AgentProcess{Type: inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER},
		"notChanged": &agentv1.SetStateRequest_AgentProcess{},
	}
	toStart, toRestart, toStop := filter(existingParams, newParams)
	assert.Equal(t, []string{"toStart"}, toStart)
	assert.Equal(t, []string{"toRestart"}, toRestart)
	assert.Equal(t, []string{"toStop"}, toStop)
}

func TestSupervisorSendQANRequest(t *testing.T) {
	t.Parallel()

	request := &agentv1.QANCollectRequest{
		MetricsBucket: []*agentv1.MetricsBucket{{}},
	}
	l := logrus.NewEntry(logrus.New())

	t.Run("sends without delay", func(t *testing.T) {
		t.Parallel()

		s := &Supervisor{
			qanRequests: make(chan *agentv1.QANCollectRequest, 1),
		}

		sent := s.sendQANRequest(t.Context(), l, request, 0)
		require.True(t, sent)
		assert.Same(t, request, <-s.QANRequests())
	})

	t.Run("cancels pending delivery", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		s := &Supervisor{
			qanRequests: make(chan *agentv1.QANCollectRequest, 1),
		}

		done := make(chan bool, 1)
		go func() {
			done <- s.sendQANRequest(ctx, l, request, time.Hour)
		}()

		cancel()

		select {
		case sent := <-done:
			assert.False(t, sent)
		case <-time.After(time.Second):
			t.Fatal("sendQANRequest did not stop after context cancellation")
		}

		assert.Empty(t, s.qanRequests)
	})
}

func TestSupervisorProcessParams(t *testing.T) {
	t.Parallel()
	setup := func(t *testing.T) (*Supervisor, func()) {
		t.Helper()

		ctx, cancel := context.WithCancel(t.Context())
		paths := config.Paths{
			MySQLdExporter: "/path/to/mysql_exporter",
			Nomad:          "/path/to/nomad",
			TempDir:        t.TempDir(),
			NomadDataDir:   "/path/to/nomad/data",
		}

		cfgStorage := config.NewStorage(&config.Config{
			Paths:         paths,
			Ports:         config.Ports{},
			Server:        config.Server{Address: "server:443", Username: "admin", Password: "admin"},
			LogLinesCount: 1,
		})
		s := NewSupervisor(ctx, nil, cfgStorage) //nolint:varnamelen
		go s.Run(ctx)

		return s, cancel
	}

	t.Run("Normal", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		t.Cleanup(teardown)

		p := &agentv1.SetStateRequest_AgentProcess{
			Type: inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			Args: []string{
				"-web.listen-address=:{{ .listen_port }}",
				"-web.ssl-cert-file={{ .TextFiles.Cert }}",
				"-web.config={{ .TextFiles.Config }}",
			},
			Env: []string{
				"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true&sslCaFile={{.TextFiles.caFilePlaceholder}}&sslCertificateKeyFile={{.TextFiles.certificateKeyFilePlaceholder}}",
				"HTTP_AUTH=pmm:secret",
				"TEST=:{{ .listen_port }}",
			},
			TextFiles: map[string]string{
				"Cert":                          "-----BEGIN CERTIFICATE-----\n...",
				"Config":                        "test={{ .listen_port }}",
				"caFilePlaceholder":             "ca",
				"certificateKeyFilePlaceholder": "certificate",
			},
		}
		actual, err := s.processParams("ID", p, 12345)
		require.NoError(t, err)

		configFilePath := filepath.Join(s.cfg.Get().Paths.TempDir, "mysqld_exporter", "ID", "Config")
		expected := process.Params{
			Path: "/path/to/mysql_exporter",
			Args: []string{
				"-web.listen-address=:12345",
				"-web.ssl-cert-file=" + filepath.Join(s.cfg.Get().Paths.TempDir, "mysqld_exporter", "ID", "Cert"),
				"-web.config=" + configFilePath,
			},
			Env: []string{
				"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true&" +
					"sslCaFile=" + filepath.Join(s.cfg.Get().Paths.TempDir, "mysqld_exporter", "ID", "caFilePlaceholder") +
					"&sslCertificateKeyFile=" + filepath.Join(s.cfg.Get().Paths.TempDir, "mysqld_exporter", "ID", "certificateKeyFilePlaceholder"),
				"HTTP_AUTH=pmm:secret",
				"TEST=:12345",
			},
		}
		assert.Equal(t, expected.Path, actual.Path)
		assert.Equal(t, expected.Args, actual.Args)
		assert.Equal(t, expected.Env, actual.Env)
		assert.NotEmpty(t, actual.TemplateParams)
		assert.NotEmpty(t, actual.TemplateRenderer)
		require.FileExists(t, configFilePath)
		b, err := os.ReadFile(configFilePath) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "test=12345", string(b))
	})

	t.Run("Nomad", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		t.Cleanup(teardown)

		configTemplate := `log_level = "DEBUG"

disable_update_check = true
data_dir = "{{.nomad_data_dir}}" # it shall be persistent
region = "global"
datacenter = "PMM Deployment"
name = "PMM Agent node-name"

ui {
  enabled = false
}

addresses {
  http = "127.0.0.1"
  rpc = "127.0.0.1"
}

advertise {
  # 127.0.0.1 is not applicable here
  http = "node-address" # filled by PMM Server
  rpc = "node-address"  # filled by PMM Server
}

client {
  enabled = true
  cpu_total_compute = 1000

  servers = ["{{.server_host}}:4647"] # filled by PMM Server

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec"
    "driver.allowlist" = "raw_exec"
  }

  # optional labels assigned to Nomad Client, can be the same as PMM Agent's.
  meta {
    pmm-agent = "1"
    agent_type = "nomad-agent"
    node_id = "node-id"
    node_name = "node-name"
  }
}

server {
  enabled = false
}

tls {
  http = true
  rpc  = true
  ca_file   = "{{ .TextFiles.caCert }}" # filled by PMM Agent
  cert_file = "{{ .TextFiles.certFile }}" # filled by PMM Agent
  key_file  = "{{ .TextFiles.keyFile }}" # filled by PMM Agent

  verify_server_hostname = true
}

# Enabled plugins
plugin "raw_exec" {
  config {
      enabled = true
  }
}
`

		expectedConfig := `log_level = "DEBUG"

disable_update_check = true
data_dir = "/path/to/nomad/data" # it shall be persistent
region = "global"
datacenter = "PMM Deployment"
name = "PMM Agent node-name"

ui {
  enabled = false
}

addresses {
  http = "127.0.0.1"
  rpc = "127.0.0.1"
}

advertise {
  # 127.0.0.1 is not applicable here
  http = "node-address" # filled by PMM Server
  rpc = "node-address"  # filled by PMM Server
}

client {
  enabled = true
  cpu_total_compute = 1000

  servers = ["server:4647"] # filled by PMM Server

  # disable Docker plugin
  options = {
    "driver.denylist" = "docker,qemu,java,exec"
    "driver.allowlist" = "raw_exec"
  }

  # optional labels assigned to Nomad Client, can be the same as PMM Agent's.
  meta {
    pmm-agent = "1"
    agent_type = "nomad-agent"
    node_id = "node-id"
    node_name = "node-name"
  }
}

server {
  enabled = false
}

tls {
  http = true
  rpc  = true
  ca_file   = "` + filepath.Join(s.cfg.Get().Paths.TempDir, "nomad_agent", "ID", "caCert") + `" # filled by PMM Agent
  cert_file = "` + filepath.Join(s.cfg.Get().Paths.TempDir, "nomad_agent", "ID", "certFile") + `" # filled by PMM Agent
  key_file  = "` + filepath.Join(s.cfg.Get().Paths.TempDir, "nomad_agent", "ID", "keyFile") + `" # filled by PMM Agent

  verify_server_hostname = true
}

# Enabled plugins
plugin "raw_exec" {
  config {
      enabled = true
  }
}
`

		p := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_NOMAD_AGENT,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"agent",
				"-client",
				"-config",
				"{{ .TextFiles.nomadConfig }}",
			},
			TextFiles: map[string]string{
				"nomadConfig": configTemplate,
				"caCert":      "-----BEGIN CERTIFICATE-----\n...",
				"certFile":    "---BEGIN CERTIFICATE---\n...",
				"keyFile":     "---BEGIN PRIVATE",
			},
		}
		actual, err := s.processParams("ID", p, 12345)
		require.NoError(t, err)

		configFilePath := filepath.Join(s.cfg.Get().Paths.TempDir, "nomad_agent", "ID", "nomadConfig")
		expected := process.Params{
			Path: "/path/to/nomad",
			Args: []string{
				"agent",
				"-client",
				"-config",
				configFilePath,
			},
		}
		assert.Equal(t, expected.Path, actual.Path)
		assert.Equal(t, expected.Args, actual.Args)
		assert.NotEmpty(t, actual.TemplateParams)
		assert.NotEmpty(t, actual.TemplateRenderer)
		require.FileExists(t, configFilePath)
		b, err := os.ReadFile(configFilePath) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, expectedConfig, string(b))
	})

	t.Run("VMAgent", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		t.Cleanup(teardown)

		// Update the config to include VMAgent path
		cfg := s.cfg.Get()
		cfg.Paths.VMAgent = "/path/to/vmagent"

		p := &agentv1.SetStateRequest_AgentProcess{
			Type:               inventoryv1.AgentType_AGENT_TYPE_VM_AGENT,
			TemplateLeftDelim:  "{{",
			TemplateRightDelim: "}}",
			Args: []string{
				"-envflag.enable=true",
				"-envflag.prefix=VMAGENT_",
				"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
				"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
				"-httpListenAddr=127.0.0.1:{{.listen_port}}",
			},
			Env: []string{
				"VMAGENT_remoteWrite_url={{.server_url}}/victoriametrics/api/v1/write",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify={{.server_insecure}}",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_remoteWrite_basicAuth_username={{.server_username}}",
				"VMAGENT_remoteWrite_basicAuth_password={{.server_password}}",
			},
			TextFiles: map[string]string{
				"vmagentscrapecfg": "global:\n  scrape_interval: 15s\n",
			},
		}
		actual, err := s.processParams("vmagent-id", p, 12345)
		require.NoError(t, err)

		configFilePath := filepath.Join(s.cfg.Get().Paths.TempDir, "vm_agent", "vmagent-id", "vmagentscrapecfg")
		tempDir := s.cfg.Get().Paths.TempDir
		expected := process.Params{
			Path: "/path/to/vmagent",
			Args: []string{
				"-envflag.enable=true",
				"-envflag.prefix=VMAGENT_",
				"-remoteWrite.tmpDataPath=" + tempDir + "/vmagent-temp-dir",
				"-promscrape.config=" + configFilePath,
				"-httpListenAddr=127.0.0.1:12345",
			},
			Env: []string{
				"VMAGENT_remoteWrite_url=https://server:443/victoriametrics/api/v1/write",
				"VMAGENT_remoteWrite_tlsInsecureSkipVerify=false",
				"VMAGENT_promscrape_maxScrapeSize=64MiB",
				"VMAGENT_remoteWrite_maxDiskUsagePerURL=1073741824",
				"VMAGENT_loggerLevel=INFO",
				"VMAGENT_remoteWrite_basicAuth_username=admin",
				"VMAGENT_remoteWrite_basicAuth_password=admin",
			},
		}
		assert.Equal(t, expected.Path, actual.Path)
		assert.ElementsMatch(t, expected.Args, actual.Args)
		assert.ElementsMatch(t, expected.Env, actual.Env)
		assert.NotEmpty(t, actual.TemplateParams)
		assert.NotEmpty(t, actual.TemplateRenderer)
		require.FileExists(t, configFilePath)
		b, err := os.ReadFile(configFilePath) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "global:\n  scrape_interval: 15s\n", string(b))
	})

	t.Run("BadTemplate", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		t.Cleanup(teardown)

		p := &agentv1.SetStateRequest_AgentProcess{
			Type: inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			Args: []string{"-foo=:{{ .bar }}"},
		}
		_, err := s.processParams("ID", p, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "bar"`, err.Error())

		p = &agentv1.SetStateRequest_AgentProcess{
			Type:      inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"foo": "{{ .bar }}"},
		}
		_, err = s.processParams("ID", p, 0)
		require.Error(t, err)
		assert.Regexp(t, `map has no entry for key "bar"`, err.Error())

		p = &agentv1.SetStateRequest_AgentProcess{
			Type:      inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
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
		t.Cleanup(teardown)

		agentProcess := &agentv1.SetStateRequest_AgentProcess{
			Type:      inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"../bar": "hax0r"},
		}
		_, err := s.processParams("ID", agentProcess, 0)
		require.Error(t, err)
		assert.Regexp(t, `invalid text file name "../bar"`, err.Error())
	})

	t.Run("TrimPrefix", func(t *testing.T) {
		t.Parallel()

		actual := trimPrefix(inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER.String())
		expected := "mysqld_exporter"
		assert.Equal(t, expected, actual)
	})
}
