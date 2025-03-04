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
	"sort"
	"testing"

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

	sort.Slice(expected, func(i, j int) bool { return expected[i].AgentId < expected[j].AgentId })
	sort.Slice(actual, func(i, j int) bool { return actual[i].AgentId < actual[j].AgentId })
	assert.Equal(t, expected, actual)
}

func TestSupervisor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
		expectedList := []*agentlocal.AgentInfo{}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "30"},
			},
		})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		}
		assert.Equal(t, expectedList, s.AgentsList())

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		}
		assert.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Restart1Start2", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"20"}},
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "30"},
			},
		})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65000, ProcessExecPath: "sleep"})
		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65000, ProcessExecPath: "sleep"})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		assert.Equal(t, expectedList, s.AgentsList())

		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		assert.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Restart3Start4", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())

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

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING})
		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		assert.Equal(t, expectedList, s.AgentsList())

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		assert.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Stop1", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65000, ProcessExecPath: "sleep"},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop3": {Type: type_TEST_NOOP, Dsn: "20"},
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65000, ProcessExecPath: "sleep"})
		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65000, ProcessExecPath: "sleep"})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Stop3", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep2": {Type: type_TEST_SLEEP, Args: []string{"10"}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop4": {Type: type_TEST_NOOP, Dsn: "10"},
			},
		})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING})
		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "noop3", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE})
		expectedList = []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())
	})

	t.Run("Exit", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{
			{AgentType: type_TEST_NOOP, AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
			{AgentType: type_TEST_SLEEP, AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65001, ProcessExecPath: "sleep"},
		}
		require.Equal(t, expectedList, s.AgentsList())

		cancel()

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop4", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE})
		assertChanges(t, s, nil)
		expectedList = []*agentlocal.AgentInfo{}
		require.Equal(t, expectedList, s.AgentsList())
	})
}

func TestStartProcessFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tempDir := t.TempDir()
	cfgStorage := config.NewStorage(&config.Config{
		Paths:         config.Paths{TempDir: tempDir},
		Ports:         config.Ports{Min: 65000, Max: 65099},
		Server:        config.Server{Address: "localhost:443"},
		LogLinesCount: 1,
	})
	s := NewSupervisor(ctx, nil, cfgStorage)
	go s.Run(ctx)

	t.Run("Start", func(t *testing.T) {
		expectedList := []*agentlocal.AgentInfo{}
		require.Equal(t, expectedList, s.AgentsList())

		s.SetState(&agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: type_TEST_SLEEP, Args: []string{"wrong format"}},
			},
		})

		assertChanges(t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65000, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65001, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65002, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_INITIALIZATION_ERROR, ListenPort: 65002, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65002, ProcessExecPath: "sleep"})
		expectedList = []*agentlocal.AgentInfo{}
		require.Equal(t, expectedList, s.AgentsList())
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

func TestSupervisorProcessParams(t *testing.T) {
	t.Parallel()
	setup := func(t *testing.T) (*Supervisor, func()) {
		t.Helper()

		temp, err := os.MkdirTemp("", "pmm-agent-")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		paths := config.Paths{
			MySQLdExporter: "/path/to/mysql_exporter",
			Nomad:          "/path/to/nomad",
			TempDir:        temp,
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

		configFilePath := filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_mysqld_exporter", "ID", "Config")
		expected := process.Params{
			Path: "/path/to/mysql_exporter",
			Args: []string{
				"-web.listen-address=:12345",
				"-web.ssl-cert-file=" + filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_mysqld_exporter", "ID", "Cert"),
				"-web.config=" + configFilePath,
			},
			Env: []string{
				"MONGODB_URI=mongodb://username:s3cur3%20p%40$$w0r4.@1.2.3.4:12345/?connectTimeoutMS=1000&ssl=true&" +
					"sslCaFile=" + filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_mysqld_exporter", "ID", "caFilePlaceholder") +
					"&sslCertificateKeyFile=" + filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_mysqld_exporter", "ID", "certificateKeyFilePlaceholder"),
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
		defer teardown()

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
    agent_type = "nomad-client"
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

  # optional lables set to Nomad Client, may be the same as for PMM Agent.
  meta {
    pmm-agent = "1"
    agent_type = "nomad-client"
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
  ca_file   = "` + filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_nomad_agent", "ID", "caCert") + `" # filled by PMM Agent
  cert_file = "` + filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_nomad_agent", "ID", "certFile") + `" # filled by PMM Agent
  key_file  = "` + filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_nomad_agent", "ID", "keyFile") + `" # filled by PMM Agent

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

		configFilePath := filepath.Join(s.cfg.Get().Paths.TempDir, "agent_type_nomad_agent", "ID", "nomadConfig")
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

	t.Run("BadTemplate", func(t *testing.T) {
		t.Parallel()
		s, teardown := setup(t)
		defer teardown()

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
		defer teardown()

		agentProcess := &agentv1.SetStateRequest_AgentProcess{
			Type:      inventoryv1.AgentType_AGENT_TYPE_MYSQLD_EXPORTER,
			TextFiles: map[string]string{"../bar": "hax0r"},
		}
		_, err := s.processParams("ID", agentProcess, 0)
		require.Error(t, err)
		assert.Regexp(t, `invalid text file name "../bar"`, err.Error())
	})
}
