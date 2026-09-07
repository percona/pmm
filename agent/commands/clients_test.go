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

package commands

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The subtests share the package level API clients, so they cannot run in parallel.
func TestServerNodeOfAgent(t *testing.T) {
	const (
		agentID  = "5a2b8a4b-2b9d-4a5f-9a11-2b6a3f6f9a11"
		nodeID   = "8c1e0d3a-6f2b-4c8e-9a0d-3b7f2e1c5d44"
		nodeName = "test-node"
	)

	for _, tc := range []struct {
		name        string
		agentStatus int
		nodeStatus  int
		hangs       bool
		nodeName    string
		err         error
		unknowable  bool
	}{
		{
			name:        "PMM Server knows the Agent",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusOK,
			nodeName:    nodeName,
		},
		{
			name:        "PMM Server does not know the Agent",
			agentStatus: http.StatusNotFound,
			err:         errAgentNotFound,
		},
		{
			name:        "PMM Server rejects the Agent ID",
			agentStatus: http.StatusBadRequest,
			err:         errAgentNotFound,
		},
		{
			name:        "PMM Server does not accept the credentials",
			agentStatus: http.StatusUnauthorized,
			err:         errCredentialsRejected,
		},
		{
			name:        "PMM Server forbids the request",
			agentStatus: http.StatusForbidden,
			err:         errCredentialsRejected,
		},
		{
			name:        "PMM Server cannot answer",
			agentStatus: http.StatusServiceUnavailable,
			unknowable:  true,
		},
		{
			name:        "PMM Server cannot answer about the Node",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusServiceUnavailable,
			unknowable:  true,
		},
		{
			name:       "PMM Server hangs",
			hangs:      true,
			unknowable: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hangs {
				// The request has to give up long before the default 30 seconds.
				defaultTimeout := httptransport.DefaultTimeout
				httptransport.DefaultTimeout = 100 * time.Millisecond
				t.Cleanup(func() { httptransport.DefaultTimeout = defaultTimeout })
			}

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if tc.hangs {
					<-req.Context().Done()
					return
				}
				rw.Header().Set("Content-Type", "application/json")
				switch {
				case strings.HasPrefix(req.URL.Path, "/v1/inventory/agents/"):
					rw.WriteHeader(tc.agentStatus)
					if tc.agentStatus == http.StatusOK {
						_, _ = rw.Write([]byte(`{"pmm_agent": {"agent_id": "` + agentID + `", "runs_on_node_id": "` + nodeID + `"}}`))
						return
					}
				case strings.HasPrefix(req.URL.Path, "/v1/inventory/nodes/"):
					rw.WriteHeader(tc.nodeStatus)
					if tc.nodeStatus == http.StatusOK {
						_, _ = rw.Write([]byte(`{"generic": {"node_id": "` + nodeID + `", "node_name": "` + nodeName + `"}}`))
						return
					}
				default:
					rw.WriteHeader(http.StatusNotFound)
				}
				_, _ = rw.Write([]byte(`{"message": "` + tc.name + `"}`))
			}))
			t.Cleanup(server.Close)

			u, err := url.Parse(server.URL)
			require.NoError(t, err)
			setServerTransport(u, true, logrus.WithField("test", t.Name()))

			name, err := serverNodeOfAgent(agentID)
			switch {
			case tc.err != nil:
				require.ErrorIs(t, err, tc.err)
			case tc.unknowable:
				require.Error(t, err)
				require.NotErrorIs(t, err, errAgentNotFound)
				require.NotErrorIs(t, err, errCredentialsRejected)
			default:
				require.NoError(t, err)
			}
			assert.Equal(t, tc.nodeName, name)
		})
	}
}
