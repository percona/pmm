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
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/kong"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/inventory/v1/json/client"
	agents "github.com/percona/pmm/api/inventory/v1/json/client/agents_service"
)

// setupAddAgentTestServer stands in for the inventory API on the add-agent
// path, capturing the request body so a test can assert what the command built.
// The returned cleanup restores the original client and must be called.
func setupAddAgentTestServer(t *testing.T, responseJSON string, capturedRequestBody *string) func() {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/inventory/agents", r.URL.Path)

		if capturedRequestBody != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			*capturedRequestBody = string(body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte(responseJSON))
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

func TestAddAgentRTAMySQLAgentCommand(t *testing.T) {
	t.Parallel()

	const okResponse = `{"rta_mysql_agent": {"agent_id": "new-agent-id"}}`

	t.Run("AllFlags", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		caFile := filepath.Join(dir, "ca.pem")
		certFile := filepath.Join(dir, "cert.pem")
		keyFile := filepath.Join(dir, "key.pem")
		require.NoError(t, os.WriteFile(caFile, []byte("ca-contents"), 0o600))
		require.NoError(t, os.WriteFile(certFile, []byte("cert-contents"), 0o600))
		require.NoError(t, os.WriteFile(keyFile, []byte("key-contents"), 0o600))

		var capturedRequestBody string
		cleanup := setupAddAgentTestServer(t, okResponse, &capturedRequestBody)
		defer cleanup()

		var cmd AddAgentRTAMySQLAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse([]string{
			"pmm-agent-id", "service-id", "testuser",
			"--password=testpass",
			"--tls",
			"--tls-skip-verify",
			"--tls-ca-file=" + caFile,
			"--tls-cert-file=" + certFile,
			"--tls-key-file=" + keyFile,
			"--collect-interval=7s",
			"--log-level=info",
			"--custom-labels=env=test,team=rta",
			"--skip-connection-check",
		})
		require.NoError(t, err)

		result, err := cmd.RunCmd()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// The TLS files are sent as contents, not as paths.
		assert.JSONEq(t, `{
			"rta_mysql_agent": {
				"pmm_agent_id": "pmm-agent-id",
				"service_id": "service-id",
				"username": "testuser",
				"password": "testpass",
				"tls": true,
				"tls_skip_verify": true,
				"tls_ca": "ca-contents",
				"tls_cert": "cert-contents",
				"tls_key": "key-contents",
				"log_level": "LOG_LEVEL_INFO",
				"skip_connection_check": true,
				"custom_labels": {
					"env": "test",
					"team": "rta"
				},
				"rta_options": {
					"collect_interval": "7s"
				}
			}
		}`, capturedRequestBody)
	})

	// Without --collect-interval no RTA options are sent at all, which is what
	// leaves the agent on the collector's default.
	t.Run("OmittedCollectIntervalSendsNoRTAOptions", func(t *testing.T) {
		t.Parallel()

		var capturedRequestBody string
		cleanup := setupAddAgentTestServer(t, okResponse, &capturedRequestBody)
		defer cleanup()

		var cmd AddAgentRTAMySQLAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse([]string{"pmm-agent-id", "service-id"})
		require.NoError(t, err)

		_, err = cmd.RunCmd()
		require.NoError(t, err)

		assert.NotContains(t, capturedRequestBody, "rta_options")
		assert.JSONEq(t, `{
			"rta_mysql_agent": {
				"pmm_agent_id": "pmm-agent-id",
				"service_id": "service-id",
				"log_level": "LOG_LEVEL_WARN"
			}
		}`, capturedRequestBody)
	})

	t.Run("MissingTLSFileIsReported", func(t *testing.T) {
		t.Parallel()

		var cmd AddAgentRTAMySQLAgentCommand
		parser, err := kong.New(&cmd)
		require.NoError(t, err)

		_, err = parser.Parse([]string{
			"pmm-agent-id", "service-id",
			"--tls-ca-file=/non/existent/ca.pem",
		})
		require.NoError(t, err)

		_, err = cmd.RunCmd()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "/non/existent/ca.pem")
	})

	t.Run("ResultTemplate", func(t *testing.T) {
		t.Parallel()

		t.Run("WithCollectInterval", func(t *testing.T) {
			t.Parallel()

			res := &addAgentRTAMySQLAgentResult{
				Agent: &agents.AddAgentOKBodyRtaMysqlAgent{
					AgentID:  "agent-id",
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
					RtaOptions: &agents.AddAgentOKBodyRtaMysqlAgentRtaOptions{
						CollectInterval: "5s",
					},
				},
			}

			assert.Contains(t, res.String(), "Collect interval      : 5s")
		})

		// ToAPIRTAOptions returns nil for empty options, and RenderTemplate
		// panics on a nil dereference, so the field's absence has to render -
		// this is the shape produced by an add without --collect-interval.
		t.Run("WithoutRTAOptions", func(t *testing.T) {
			t.Parallel()

			res := &addAgentRTAMySQLAgentResult{
				Agent: &agents.AddAgentOKBodyRtaMysqlAgent{
					AgentID:  "agent-id",
					LogLevel: new("LOG_LEVEL_UNSPECIFIED"),
				},
			}

			out := res.String()
			assert.NotContains(t, out, "Collect interval")
			assert.Contains(t, out, "Agent ID              : agent-id")
			assert.Contains(t, out, "Log level")
		})
	})
}
