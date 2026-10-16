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

package base

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/go-openapi/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

// fakeLocalAgent is the local pmm-agent, with the ID pmm-agent-1, which runs the Agent exporter-1.
type fakeLocalAgent struct {
	m       sync.Mutex
	staged  []map[string]string
	staging bool
}

func (a *fakeLocalAgent) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.m.Lock()
	defer a.m.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/local/Status":
		_, _ = w.Write([]byte(`{"agent_id":"pmm-agent-1","runs_on_node_id":"node-1",` +
			`"server_info":{"url":"https://pmm/","connected":true},"agents_info":[{"agent_id":"exporter-1"}]}`))
	case "/local/StageSecrets":
		if !a.staging {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req struct {
			Data map[string]string `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		a.staged = append(a.staged, req.Data)
		refs := make(map[string]string)
		for k := range req.Data {
			refs[k] = "vault:pmm-agent-staged/h" + strconv.Itoa(len(a.staged)) + "#" + k
		}
		err := json.NewEncoder(w).Encode(struct {
			Refs map[string]string `json:"refs"`
		}{refs})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// fakeServer records the requests to PMM Server, and answers GetAgent with an Agent that has a Vault path.
type fakeServer struct {
	submitted []any
}

func (s *fakeServer) Submit(op *runtime.ClientOperation) (any, error) {
	if p, ok := op.Params.(*agents_service.GetAgentParams); ok {
		return &agents_service.GetAgentOK{Payload: &agents_service.GetAgentOKBody{
			MysqldExporter: &agents_service.GetAgentOKBodyMysqldExporter{AgentID: p.AgentID, VaultPath: "secret/data/pmm/mysql"},
		}}, nil
	}
	s.submitted = append(s.submitted, op.Params)
	return struct{}{}, nil
}

// Not parallel: the address of the local pmm-agent is process-wide.
func TestStagingTransport(t *testing.T) {
	local := &fakeLocalAgent{staging: true}
	srv := httptest.NewServer(local)
	t.Cleanup(srv.Close)
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	require.NoError(t, err)
	p, err := strconv.ParseUint(port, 10, 32)
	require.NoError(t, err)
	agentlocal.SetTransport(false, uint32(p))
	t.Cleanup(func() { agentlocal.SetTransport(false, agentlocal.DefaultPMMAgentListenPort) })

	addMySQL := func(pmmAgentID, vaultPath string) *mservice.AddServiceParams {
		return &mservice.AddServiceParams{Body: mservice.AddServiceBody{Mysql: &mservice.AddServiceParamsBodyMysql{
			PMMAgentID: pmmAgentID, VaultPath: vaultPath, Username: "pmm", Password: "s3cr3t",
			TLSKey: "KEY", TLSCa: "vault:secret/data/other#ca", Port: 3306, MaxQueryLength: 2048,
		}}}
	}
	submit := func(t *testing.T, params runtime.ClientRequestWriter) *fakeServer {
		t.Helper()
		s := &fakeServer{}
		_, err := stagingTransport{s}.Submit(&runtime.ClientOperation{Params: params})
		require.NoError(t, err)
		require.Len(t, s.submitted, 1)
		return s
	}

	t.Run("added with a Vault path, on this pmm-agent", func(t *testing.T) {
		s := submit(t, addMySQL("pmm-agent-1", "secret/data/pmm/mysql"))
		mysql := s.submitted[0].(*mservice.AddServiceParams).Body.Mysql
		assert.Regexp(t, `^vault:pmm-agent-staged/h\d+#password$`, mysql.Password)
		assert.Regexp(t, `^vault:pmm-agent-staged/h\d+#tls_key$`, mysql.TLSKey)
		assert.Equal(t, "vault:secret/data/other#ca", mysql.TLSCa, "in Vault already")
		assert.Equal(t, "pmm", mysql.Username)
		assert.Equal(t, int64(3306), mysql.Port)
		assert.Equal(t, int32(2048), mysql.MaxQueryLength)
		assert.Equal(t, map[string]string{"password": "s3cr3t", "tls_key": "KEY"}, local.staged[len(local.staged)-1])
	})

	t.Run("sent as they are", func(t *testing.T) {
		for name, params := range map[string]*mservice.AddServiceParams{
			"on another pmm-agent": addMySQL("pmm-agent-2", "secret/data/pmm/mysql"),
			"without a Vault path": addMySQL("pmm-agent-1", ""),
		} {
			s := submit(t, params)
			assert.Equal(t, "s3cr3t", s.submitted[0].(*mservice.AddServiceParams).Body.Mysql.Password, name)
		}
	})

	t.Run("a change of an Agent of this pmm-agent with a Vault path", func(t *testing.T) {
		s := submit(t, &agents_service.ChangeAgentParams{
			AgentID: "exporter-1",
			Body: agents_service.ChangeAgentBody{MysqldExporter: &agents_service.ChangeAgentParamsBodyMysqldExporter{
				Password: new("rotated"),
			}},
		})
		change := s.submitted[0].(*agents_service.ChangeAgentParams).Body.MysqldExporter
		assert.Regexp(t, `^vault:pmm-agent-staged/h\d+#password$`, pointer.GetString(change.Password))
	})

	t.Run("a pmm-agent that does not stage", func(t *testing.T) {
		local.m.Lock()
		local.staging = false
		local.m.Unlock()

		s := submit(t, addMySQL("pmm-agent-1", "secret/data/pmm/mysql"))
		assert.Equal(t, "s3cr3t", s.submitted[0].(*mservice.AddServiceParams).Body.Mysql.Password)
	})
}
