// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package supervisor

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/agent/config"
	"github.com/percona/pmm/agent/utils/secrets"
	agentv1 "github.com/percona/pmm/api/agent/v1"
	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/utils/vault"
)

type fakeVault struct {
	m       sync.Mutex
	secrets map[string]string
	err     error
}

func (f *fakeVault) Get(_ context.Context, path, key string) (string, error) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.err != nil {
		return "", f.err
	}
	v, ok := f.secrets[path+"#"+key]
	if !ok {
		return "", errors.New("no such secret")
	}
	return v, nil
}

func (f *fakeVault) setPassword(value string, err error) {
	f.m.Lock()
	defer f.m.Unlock()

	f.secrets["kv/db#password"] = value
	f.err = err
}

// assertChangesIgnoringPorts is assertChanges for Agents whose ports depend on reservations given up earlier.
func assertChangesIgnoringPorts(t *testing.T, s *Supervisor, expected ...*agentv1.StateChangedRequest) {
	t.Helper()

	actual := make([]*agentv1.StateChangedRequest, len(expected))
	for i := range expected {
		actual[i] = <-s.Changes()
		actual[i].ListenPort = 0
	}

	require.ElementsMatch(t, expected, actual)
}

// Not parallel: it configures the process-wide Vault client.
func TestSupervisorVaultSecrets(t *testing.T) {
	fv := &fakeVault{secrets: map[string]string{"kv/db#password": "first"}}
	secrets.Configure(fv)
	t.Cleanup(func() { secrets.Configure(nil) })

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	cfgStorage := config.NewStorage(&config.Config{
		Paths:         config.Paths{TempDir: t.TempDir()},
		Ports:         config.Ports{Min: 65100, Max: 65199},
		Server:        config.Server{Address: "localhost:8443"},
		LogLinesCount: 1,
	})
	s := NewSupervisor(ctx, nil, cfgStorage)
	go s.Run(ctx)

	token := vault.Token("kv/db", "password", vault.EncodingRaw)

	t.Run("RestartOnChange", func(t *testing.T) {
		s.SetState(t.Context(), &agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep1": {Type: typeTestSleep, Args: []string{"10"}, Env: []string{"DB_PASSWORD=" + token}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop1": {Type: typeTestNoop, Dsn: "user:" + token + "@tcp(127.0.0.1:3306)/"},
			},
		})
		assertChanges(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
		)

		s.rw.RLock()
		before := s.agentProcesses["sleep1"].secretsFingerprint
		s.rw.RUnlock()
		require.NotEmpty(t, before)

		// Unchanged secrets restart nothing.
		s.refreshSecrets()
		s.rw.RLock()
		assert.Equal(t, before, s.agentProcesses["sleep1"].secretsFingerprint)
		s.rw.RUnlock()
		assert.Empty(t, s.changes)

		// Neither does an unreachable Vault: the Agents keep running with what they have.
		fv.setPassword("second", errors.New("vault is down"))
		s.refreshSecrets()
		assert.Empty(t, s.changes)

		fv.setPassword("second", nil)
		s.refreshSecrets()
		assertChanges(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ListenPort: 65100, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
		)

		s.rw.RLock()
		assert.NotEqual(t, before, s.agentProcesses["sleep1"].secretsFingerprint)
		s.rw.RUnlock()
	})

	t.Run("RetryWhenVaultIsBack", func(t *testing.T) {
		fv.setPassword("third", errors.New("vault is down"))

		s.SetState(t.Context(), &agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep2": {Type: typeTestSleep, Args: []string{"10"}, Env: []string{"DB_PASSWORD=" + token}},
			},
			BuiltinAgents: map[string]*agentv1.SetStateRequest_BuiltinAgent{
				"noop2": {Type: typeTestNoop, Dsn: token},
			},
		})
		assertChangesIgnoringPorts(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop1", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE},
		)

		s.rw.RLock()
		assert.Empty(t, s.agentProcesses)
		assert.Empty(t, s.builtinAgents)
		assert.Contains(t, s.pendingProcesses, "sleep2")
		assert.Contains(t, s.pendingBuiltins, "noop2")
		s.rw.RUnlock()

		// Still down: nothing starts, and the Agents stay pending.
		s.refreshSecrets()
		assert.Empty(t, s.changes)

		fv.setPassword("third", nil)
		s.refreshSecrets()
		assertChangesIgnoringPorts(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STARTING},
			&agentv1.StateChangedRequest{AgentId: "noop2", Status: inventoryv1.AgentStatus_AGENT_STATUS_RUNNING},
		)

		s.rw.RLock()
		assert.Empty(t, s.pendingProcesses)
		assert.Empty(t, s.pendingBuiltins)
		s.rw.RUnlock()
	})

	t.Run("SetStateDropsPending", func(t *testing.T) {
		fv.setPassword("fourth", errors.New("vault is down"))
		s.SetState(t.Context(), &agentv1.SetStateRequest{
			AgentProcesses: map[string]*agentv1.SetStateRequest_AgentProcess{
				"sleep3": {Type: typeTestSleep, Args: []string{"10"}, Env: []string{"DB_PASSWORD=" + token}},
			},
		})
		assertChangesIgnoringPorts(
			t, s,
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "sleep2", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE, ProcessExecPath: "sleep"},
			&agentv1.StateChangedRequest{AgentId: "noop2", Status: inventoryv1.AgentStatus_AGENT_STATUS_STOPPING},
			&agentv1.StateChangedRequest{AgentId: "noop2", Status: inventoryv1.AgentStatus_AGENT_STATUS_DONE},
		)

		s.SetState(t.Context(), &agentv1.SetStateRequest{})
		s.rw.RLock()
		assert.Empty(t, s.pendingProcesses)
		s.rw.RUnlock()

		fv.setPassword("fourth", nil)
		s.refreshSecrets()
		assert.Empty(t, s.changes)
	})
}
