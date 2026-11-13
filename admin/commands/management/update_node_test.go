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

package management

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alecthomas/kong"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/commands"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	nodes "github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
)

// updateNodeTestBefore is the Node a test server holds before the update.
const updateNodeTestBefore = `{
	"generic": {"node_id": "node-id", "node_name": "node1", "region": "eu-west-1", "custom_labels": {"tier": "1"}}
}`

// runUpdateNode parses args as the command line of update node and runs the command against a test server
// that holds updateNodeTestBefore and answers the change with after. It returns the change request.
func runUpdateNode(t *testing.T, after string, args ...string) (string, *updateNodeResult) {
	t.Helper()

	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/inventory/nodes/node-id", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		response := updateNodeTestBefore
		if r.Method == http.MethodPut {
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			body = string(b)
			response = after
		}

		_, err := w.Write([]byte(response))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	originalClient := inventoryClient.Default
	inventoryClient.Default = inventoryClient.New(httptransport.New(serverURL.Host, serverURL.Path, []string{serverURL.Scheme}), nil)
	t.Cleanup(func() { inventoryClient.Default = originalClient })

	var cmd UpdateNodeCommand
	ctx, err := kong.Must(&cmd).Parse(append([]string{"--node-id=node-id"}, args...))
	require.NoError(t, err)
	cmd.SetTypedFlags(commands.TypedFlags(ctx))

	res, err := cmd.RunCmd()
	require.NoError(t, err)

	return body, res.(*updateNodeResult)
}

func TestUpdateNode(t *testing.T) {
	t.Run("SendsOnlyTypedFlags", func(t *testing.T) {
		body, res := runUpdateNode(t,
			`{"generic": {"node_id": "node-id", "node_name": "node1", "environment": "prod", "region": "eu-west-1", "custom_labels": {"team": "db"}}}`,
			"--environment=prod", "--custom-labels=team=db")

		assert.JSONEq(t, `{"environment": "prod", "custom_labels": {"values": {"team": "db"}}}`, body)
		assert.True(t, res.Changed)
		assert.Equal(t, `Node updated.
Node ID  : node-id
Node name: node1

Changed settings:
  --environment=prod (was empty)
  --custom-labels=team=db (was tier=1)

Current settings:
  --environment=prod
  --node-model=
  --region=eu-west-1
  --az=
  --custom-labels=team=db
`, res.String())
	})

	t.Run("EmptyValuesClear", func(t *testing.T) {
		body, res := runUpdateNode(t,
			`{"generic": {"node_id": "node-id", "node_name": "node1"}}`,
			"--region=", "--custom-labels=")

		assert.JSONEq(t, `{"region": "", "custom_labels": {}}`, body)
		assert.Equal(t, []settingChange{
			{Flag: "--region", Before: "eu-west-1"},
			{Flag: "--custom-labels", Before: "tier=1"},
		}, res.Changes)
	})

	t.Run("DryRun", func(t *testing.T) {
		body, res := runUpdateNode(t,
			`{"generic": {"node_id": "node-id", "node_name": "node1", "environment": "prod", "region": "eu-west-1", "custom_labels": {"tier": "1"}}}`,
			"--environment=prod", "--dry-run")

		assert.JSONEq(t, `{"environment": "prod", "dry_run": true}`, body)
		assert.Contains(t, res.String(), "Node not changed (dry run).")
		assert.Contains(t, res.String(), "Settings the update would change:\n  --environment=prod (was empty)")
	})

	t.Run("NoFlags", func(t *testing.T) {
		body, res := runUpdateNode(t, updateNodeTestBefore)

		assert.JSONEq(t, `{}`, body)
		assert.False(t, res.Changed)
		assert.Contains(t, res.String(), "No settings changed.")
	})
}

func TestNodeLabelsOf(t *testing.T) {
	labels, err := nodeLabelsOf(&nodes.GetNodeOKBody{
		RemoteRDS: &nodes.GetNodeOKBodyRemoteRDS{NodeID: "rds", NodeName: "rds1", Region: "us-east-1", Environment: "prod"},
	})
	require.NoError(t, err)
	assert.Equal(t, &nodeLabels{NodeID: "rds", NodeName: "rds1", Region: "us-east-1", Environment: "prod"}, labels)

	_, err = nodeLabelsOf(&nodes.GetNodeOKBody{})
	require.EqualError(t, err, "PMM Server returned no Node")
}
