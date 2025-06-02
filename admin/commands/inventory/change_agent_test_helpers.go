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
	"sync"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/inventory/v1/json/client"
)

var clientMutex sync.Mutex

// setupChangeAgentTestServer creates a test HTTP server for change agent tests.
// If capturedRequestBody is provided, the request body will be captured for verification.
// If responseJSON is empty, a default minimal response is used.
// Returns the server and a cleanup function that must be called to restore the original client.
func setupChangeAgentTestServer(t *testing.T, agentID string, responseJSON string, capturedRequestBody *string) (*httptest.Server, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the actual API method and path
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v1/inventory/agents/"+agentID, r.URL.Path)

		// Capture request body if requested
		if capturedRequestBody != nil {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			*capturedRequestBody = string(body)
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")

		// Handle error cases
		if strings.Contains(responseJSON, `"error"`) || strings.Contains(responseJSON, `"code"`) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		// Use provided response or default minimal response
		response := responseJSON
		if response == "" {
			response = `{"node_exporter": {"agent_id": "` + agentID + `"}}`
		}

		_, err := w.Write([]byte(response))
		require.NoError(t, err)
	}))

	// Setup client to use test server - all operations under mutex to avoid race conditions
	clientMutex.Lock()
	originalClient := client.Default
	serverURL, _ := url.Parse(server.URL)
	transport := httptransport.New(serverURL.Host, serverURL.Path, []string{serverURL.Scheme})
	client.Default = client.New(transport, nil)

	cleanup := func() {
		server.Close()
		client.Default = originalClient
		clientMutex.Unlock()
	}

	return server, cleanup
}
