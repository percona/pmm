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

package cli

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/commands/base"
	"github.com/percona/pmm/admin/commands/inventory"
	"github.com/percona/pmm/admin/pkg/flags"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	"github.com/percona/pmm/utils/servererror"
)

func TestPrintResponseTLSError(t *testing.T) {
	t.Parallel()

	serverURL, err := url.Parse("https://admin:admin@pmm-server-second:8443/")
	require.NoError(t, err)

	certErr := &url.Error{
		Op:  "Put",
		URL: "https://pmm-server-second:8443/v1/inventory/agents/722fbfc8",
		Err: x509.HostnameError{Certificate: &x509.Certificate{}, Host: "pmm-server-second"},
	}

	t.Run("hint added", func(t *testing.T) {
		t.Parallel()

		opts := &flags.GlobalFlags{ServerURL: serverURL} //nolint:exhaustruct

		wrapped := printResponse(opts, nil, certErr)
		require.Error(t, wrapped)
		assert.Contains(t, wrapped.Error(), servererror.InsecureTLSFlag)
	})

	t.Run("no hint when validation is already disabled", func(t *testing.T) {
		t.Parallel()

		opts := &flags.GlobalFlags{ServerURL: serverURL, SkipTLSCertificateCheck: true} //nolint:exhaustruct

		assert.Equal(t, certErr, printResponse(opts, nil, certErr))
	})

	t.Run("unrelated errors are untouched", func(t *testing.T) {
		t.Parallel()

		opts := &flags.GlobalFlags{ServerURL: serverURL} //nolint:exhaustruct
		other := errors.New("connection refused")

		assert.Equal(t, other, printResponse(opts, nil, other))
	})
}

// redirectToTestServer makes the configured PMM Server clients dial srv while still using the
// host name from --server-url for the TLS handshake, so that a certificate mismatch can be
// reproduced without touching DNS.
func redirectToTestServer(t *testing.T, srv *httptest.Server) {
	t.Helper()

	target, err := url.Parse(srv.URL)
	require.NoError(t, err)

	runtime, ok := inventoryClient.Default.Transport.(*httptransport.Runtime)
	require.True(t, ok)

	transport, ok := runtime.Transport.(*http.Transport)
	require.True(t, ok)

	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, target.Host)
	}
}

// TestChangeAgentAgainstMismatchedCertificate reproduces PMM-15186 end to end: pointing
// --server-url at a PMM Server whose certificate is issued for a different host name must
// fail with a message naming --server-insecure-tls, and must succeed once that flag is set.
func TestChangeAgentAgainstMismatchedCertificate(t *testing.T) {
	// Not parallel: SetupClients configures the package-level API clients.
	const agentID = "722fbfc8-8497-4acc-839b-ec53983cf398"

	// Written by the handler goroutine, read by the test goroutine.
	var (
		mu      sync.Mutex
		gotAuth string
	)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"postgres_exporter": map[string]any{"agent_id": agentID},
		}))
	}))
	t.Cleanup(srv.Close)

	// The httptest certificate is issued for example.com and 127.0.0.1 only, mirroring the
	// localhost-only certificate PMM Server ships with.
	serverURL, err := url.Parse("https://admin:admin@pmm-server-second:8443/")
	require.NoError(t, err)

	t.Run("without --server-insecure-tls", func(t *testing.T) {
		opts := &flags.GlobalFlags{ServerURL: cloneURL(t, serverURL)} //nolint:exhaustruct
		base.SetupClients(opts)
		redirectToTestServer(t, srv)

		cmd := &inventory.ChangeAgentPostgresExporterCommand{AgentID: agentID} //nolint:exhaustruct
		_, cmdErr := cmd.RunCmd()
		require.Error(t, cmdErr)

		wrapped := printResponse(opts, nil, cmdErr)
		require.Error(t, wrapped)

		msg := wrapped.Error()
		assert.True(t, servererror.IsTLSCertificateError(wrapped), msg)
		assert.Contains(t, msg, "PMM Server TLS certificate could not be verified")
		assert.Contains(t, msg, `not valid for host "pmm-server-second"`)
		assert.Contains(t, msg, servererror.InsecureTLSFlag)
	})

	t.Run("with --server-insecure-tls", func(t *testing.T) {
		opts := &flags.GlobalFlags{ //nolint:exhaustruct
			ServerURL:               cloneURL(t, serverURL),
			SkipTLSCertificateCheck: true,
		}
		base.SetupClients(opts)
		redirectToTestServer(t, srv)

		cmd := &inventory.ChangeAgentPostgresExporterCommand{AgentID: agentID} //nolint:exhaustruct
		res, cmdErr := cmd.RunCmd()
		require.NoError(t, cmdErr)
		require.NotNil(t, res)

		// Credentials from --server-url must reach PMM Server; the ticket comment
		// reported them being rejected.
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, "Basic YWRtaW46YWRtaW4=", gotAuth)
	})
}

// cloneURL returns a copy of u, since SetupClients mutates the URL it is given.
func cloneURL(t *testing.T, u *url.URL) *url.URL {
	t.Helper()

	c, err := url.Parse(u.String())
	require.NoError(t, err)

	return c
}
