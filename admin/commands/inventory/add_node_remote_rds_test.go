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

package inventory

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/inventory/v1/json/client"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
)

func TestAddNodeRemoteRDS(t *testing.T) {
	res := &addNodeRemoteRDSResult{
		Node: &nodes.AddNodeOKBodyRemoteRDS{
			NodeID:       "1",
			NodeName:     "rds1",
			Address:      "rds-mysql57.abcdef.us-east-1.rds.amazonaws.com",
			InstanceID:   "rds-mysql57",
			NodeModel:    "db.t3.micro",
			Region:       "us-east-1",
			Az:           "us-east-1b",
			CustomLabels: map[string]string{"foo": "bar"},
		},
	}
	expected := strings.TrimSpace(`
Remote RDS Node added.
Node ID  : 1
Node name: rds1

Address       : rds-mysql57.abcdef.us-east-1.rds.amazonaws.com
Instance ID   : rds-mysql57
Model         : db.t3.micro
Custom labels : map[foo:bar]

Region    : us-east-1
Az        : us-east-1b
	`)
	assert.Equal(t, expected, strings.TrimSpace(res.String()))
}

// setupAddNodeTestServer serves POST /v1/inventory/nodes, captures the request body and
// restores the default client on cleanup.
func setupAddNodeTestServer(t *testing.T, responseJSON string, capturedRequestBody *string) func() {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/inventory/nodes", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		*capturedRequestBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(responseJSON))
		if err != nil {
			t.Error(err)
		}
	}))

	clientMutex.Lock()
	originalClient := client.Default

	serverURL, _ := url.Parse(server.URL)
	transport := httptransport.New(serverURL.Host, serverURL.Path, []string{serverURL.Scheme})
	client.Default = client.New(transport, nil)

	return func() {
		server.Close()
		client.Default = originalClient
		clientMutex.Unlock()
	}
}

func TestAddNodeRemoteRDSCommand(t *testing.T) {
	t.Run("InstanceIDRequired", func(t *testing.T) {
		var cmd AddNodeRemoteRDSCommand
		parser := kong.Must(&cmd)

		_, err := parser.Parse([]string{"rds1", "--address=rds-mysql57.abcdef.us-east-1.rds.amazonaws.com", "--region=us-east-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--instance-id")
	})

	t.Run("InstanceIDInRequest", func(t *testing.T) {
		var capturedRequestBody string
		cleanup := setupAddNodeTestServer(t, `{"remote_rds": {"node_id": "1", "node_name": "rds1", "instance_id": "rds-mysql57"}}`, &capturedRequestBody)
		defer cleanup()

		var cmd AddNodeRemoteRDSCommand
		parser := kong.Must(&cmd)
		_, err := parser.Parse([]string{"rds1", "--address=rds-mysql57.abcdef.us-east-1.rds.amazonaws.com", "--region=us-east-1", "--instance-id=rds-mysql57"})
		require.NoError(t, err)

		res, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.Contains(t, res.String(), "Instance ID   : rds-mysql57")
		assert.Contains(t, capturedRequestBody, `"instance_id":"rds-mysql57"`)
	})
}
