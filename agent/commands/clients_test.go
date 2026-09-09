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
	"fmt"
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
	"google.golang.org/grpc/codes"
)

// The subtests share the package level API clients, so they cannot run in parallel.
func TestServerNodeOfAgent(t *testing.T) {
	const (
		agentID     = "5a2b8a4b-2b9d-4a5f-9a11-2b6a3f6f9a11"
		nodeID      = "8c1e0d3a-6f2b-4c8e-9a0d-3b7f2e1c5d44"
		nodeName    = "test-node"
		nodeAddress = "10.20.30.40"
	)

	for _, tc := range []struct {
		name        string
		agentStatus int
		agentCode   codes.Code
		agentBody   string
		nodeStatus  int
		nodeCode    codes.Code
		nodeBody    string
		hangs       bool
		boundedOnce bool
		node        serverNode
		err         error
		unknowable  bool
	}{
		{
			name:        "PMM Server knows the Agent",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusOK,
			node:        serverNode{Name: nodeName, Type: "generic", Address: nodeAddress},
		},
		{
			name:        "the ID belongs to another kind of Agent",
			agentStatus: http.StatusOK,
			agentBody:   `{"node_exporter": {"agent_id": "` + agentID + `"}}`,
			err:         errAgentNotFound,
		},
		{
			name:        "PMM Server does not know the Agent",
			agentStatus: http.StatusNotFound,
			agentCode:   codes.NotFound,
			err:         errAgentNotFound,
		},
		{
			name:        "PMM Server rejects the Agent ID",
			agentStatus: http.StatusBadRequest,
			agentCode:   codes.InvalidArgument,
			err:         errAgentNotFound,
		},
		{
			name:        "PMM Server does not accept the credentials",
			agentStatus: http.StatusUnauthorized,
			agentCode:   codes.Unauthenticated,
			err:         errCredentialsRejected,
		},
		{
			// A service account below the admin role holds credentials PMM Server accepts, so registering
			// the Node again over this would only add a second one.
			name:        "the credentials are not allowed to read the inventory",
			agentStatus: http.StatusForbidden,
			agentCode:   codes.PermissionDenied,
			unknowable:  true,
		},
		{
			// PMM Server answers 401 for a failure of its own, a Grafana restart among them.
			name:        "PMM Server failed while authenticating",
			agentStatus: http.StatusUnauthorized,
			agentCode:   codes.Internal,
			unknowable:  true,
		},
		{
			// A proxy whose path rules predate this call answers the same status with a body of its own.
			name:        "the 404 is not PMM Server's",
			agentStatus: http.StatusNotFound,
			unknowable:  true,
		},
		{
			name:        "PMM Server cannot answer",
			agentStatus: http.StatusServiceUnavailable,
			agentCode:   codes.Unavailable,
			unknowable:  true,
		},
		{
			name:        "PMM Server does not know the Node",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusNotFound,
			nodeCode:    codes.NotFound,
			err:         errAgentNotFound,
		},
		{
			name:        "PMM Server does not accept the credentials for the Node",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusUnauthorized,
			nodeCode:    codes.Unauthenticated,
			err:         errCredentialsRejected,
		},
		{
			name:        "the credentials are not allowed to read the Node",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusForbidden,
			nodeCode:    codes.PermissionDenied,
			unknowable:  true,
		},
		{
			name:        "PMM Server cannot answer about the Node",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusServiceUnavailable,
			nodeCode:    codes.Unavailable,
			unknowable:  true,
		},
		{
			// A newer PMM Server may hold the Agent on a Node type this pmm-agent has never heard of.
			name:        "the Node has a type this pmm-agent does not know",
			agentStatus: http.StatusOK,
			nodeStatus:  http.StatusOK,
			nodeBody:    `{"remote_valkey": {"node_id": "` + nodeID + `", "node_name": "` + nodeName + `"}}`,
			unknowable:  true,
		},
		{
			name:       "PMM Server hangs",
			hangs:      true,
			unknowable: true,
		},
		{
			// The check as a whole gives up, not each of its two requests on its own.
			name:        "PMM Server hangs for longer than the check is allowed",
			hangs:       true,
			boundedOnce: true,
			unknowable:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hangs {
				// The request has to give up long before the default 30 seconds.
				defaultTimeout := httptransport.DefaultTimeout
				httptransport.DefaultTimeout = 100 * time.Millisecond
				t.Cleanup(func() { httptransport.DefaultTimeout = defaultTimeout })
			}
			if tc.boundedOnce {
				// Leave the per-request timeout long, so that only the bound on the whole check can end
				// this. Without it the two requests would run to the per-request timeout one after another.
				httptransport.DefaultTimeout = time.Minute
				checkTimeout := registrationCheckTimeout
				registrationCheckTimeout = 100 * time.Millisecond
				t.Cleanup(func() { registrationCheckTimeout = checkTimeout })
			}

			started := time.Now()
			t.Cleanup(func() {
				if tc.hangs {
					assert.Less(t, time.Since(started), 30*time.Second, "the check has to give up on a hung PMM Server")
				}
			})

			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if tc.hangs {
					<-req.Context().Done()
					return
				}
				rw.Header().Set("Content-Type", "application/json")
				// A failure of PMM Server's own carries the gRPC code, which is what the client reads.
				// codes.OK stands for an answer from something else on the path, which carries none.
				failure := func(code codes.Code) {
					if code == codes.OK {
						_, _ = rw.Write([]byte(`{"message": "` + tc.name + `"}`))
						return
					}
					_, _ = fmt.Fprintf(rw, `{"code": %d, "message": %q}`, code, tc.name)
				}

				switch {
				case strings.HasPrefix(req.URL.Path, "/v1/inventory/agents/"):
					rw.WriteHeader(tc.agentStatus)
					if tc.agentStatus == http.StatusOK {
						body := tc.agentBody
						if body == "" {
							body = `{"pmm_agent": {"agent_id": "` + agentID + `", "runs_on_node_id": "` + nodeID + `"}}`
						}
						_, _ = rw.Write([]byte(body))
						return
					}
					failure(tc.agentCode)
				case strings.HasPrefix(req.URL.Path, "/v1/inventory/nodes/"):
					rw.WriteHeader(tc.nodeStatus)
					if tc.nodeStatus == http.StatusOK {
						body := tc.nodeBody
						if body == "" {
							body = `{"generic": {"node_id": "` + nodeID + `", "node_name": "` + nodeName +
								`", "address": "` + nodeAddress + `"}}`
						}
						_, _ = rw.Write([]byte(body))
						return
					}
					failure(tc.nodeCode)
				default:
					rw.WriteHeader(http.StatusNotFound)
					failure(codes.OK)
				}
			}))
			t.Cleanup(server.Close)

			u, err := url.Parse(server.URL)
			require.NoError(t, err)
			setServerTransport(u, true, logrus.WithField("test", t.Name()))

			node, err := serverNodeOfAgent(agentID)
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
			assert.Equal(t, tc.node, node)
		})
	}
}
