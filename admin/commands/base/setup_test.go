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
	"net/http"
	"net/url"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/pkg/flags"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
)

func TestApplyAgentServerParams(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		flagInsecureTLS  bool
		agentInsecureTLS bool
		expected         bool
	}{
		"both secure":                  {false, false, false},
		"agent configured as insecure": {false, true, true},
		// PMM-15186: --server-insecure-tls is opt-in only, so it must survive the
		// parameters read from a pmm-agent which validates certificates.
		"flag wins over secure agent": {true, false, true},
		"both insecure":               {true, true, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			globals := &flags.GlobalFlags{SkipTLSCertificateCheck: tc.flagInsecureTLS} //nolint:exhaustruct
			status := &agentlocal.Status{                                              //nolint:exhaustruct
				ServerURL:         "https://admin:admin@pmm-server:8443/",
				ServerInsecureTLS: tc.agentInsecureTLS,
			}

			require.NoError(t, applyAgentServerParams(globals, status))

			assert.Equal(t, tc.expected, globals.SkipTLSCertificateCheck)
			require.NotNil(t, globals.ServerURL)
			assert.Equal(t, "https://admin:admin@pmm-server:8443/", globals.ServerURL.String())
		})
	}
}

// TestApplyAgentServerParamsInvalidURL checks that an unparseable URL is reported instead of
// silently leaving ServerURL nil for SetupClients to dereference.
func TestApplyAgentServerParamsInvalidURL(t *testing.T) {
	t.Parallel()

	globals := &flags.GlobalFlags{}                                        //nolint:exhaustruct
	status := &agentlocal.Status{ServerURL: "https://pmm-server:8443/%zz"} //nolint:exhaustruct

	require.Error(t, applyAgentServerParams(globals, status))
	assert.Nil(t, globals.ServerURL)
}

// tlsConfigOf returns the TLS configuration the PMM Server API clients were set up with.
func tlsConfigOf(t *testing.T) *http.Transport {
	t.Helper()

	runtime, ok := inventoryClient.Default.Transport.(*httptransport.Runtime)
	require.True(t, ok, "expected the inventory client to use a go-openapi runtime")

	transport, ok := runtime.Transport.(*http.Transport)
	require.True(t, ok, "expected an *http.Transport")

	return transport
}

// TestSetupClientsServerURL covers PMM-15186: a --server-url pointing at a host the PMM
// Server certificate was not issued for must be reachable with --server-insecure-tls, and
// must keep validating certificates without it.
func TestSetupClientsServerURL(t *testing.T) {
	// Not parallel: SetupClients configures the package-level API clients.
	for name, tc := range map[string]struct {
		serverURL   string
		insecureTLS bool

		expectedInsecure   bool
		expectedServerName string
		expectedAuth       bool
	}{
		"https with insecure tls": {
			serverURL:          "https://admin:admin@pmm-server-second:8443/",
			insecureTLS:        true,
			expectedInsecure:   true,
			expectedServerName: "pmm-server-second",
			expectedAuth:       true,
		},
		"https validating certificates": {
			serverURL:          "https://admin:admin@pmm-server-second:8443/",
			insecureTLS:        false,
			expectedInsecure:   false,
			expectedServerName: "pmm-server-second",
			expectedAuth:       true,
		},
		"https without credentials": {
			serverURL:          "https://pmm-server-second:8443/",
			insecureTLS:        true,
			expectedInsecure:   true,
			expectedServerName: "pmm-server-second",
			expectedAuth:       false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			u, err := url.Parse(tc.serverURL)
			require.NoError(t, err)

			globals := &flags.GlobalFlags{ //nolint:exhaustruct
				ServerURL:               u,
				SkipTLSCertificateCheck: tc.insecureTLS,
			}

			SetupClients(globals)

			transport := tlsConfigOf(t)
			require.NotNil(t, transport.TLSClientConfig)
			assert.Equal(t, tc.expectedInsecure, transport.TLSClientConfig.InsecureSkipVerify)
			// ServerName is taken from the URL host, which is what makes the
			// certificate mismatch reported in PMM-15186 detectable at all.
			assert.Equal(t, tc.expectedServerName, transport.TLSClientConfig.ServerName)
			// HTTP/2 must stay disabled.
			assert.NotNil(t, transport.TLSNextProto)
			assert.Empty(t, transport.TLSNextProto)

			runtime, ok := inventoryClient.Default.Transport.(*httptransport.Runtime)
			require.True(t, ok)
			if tc.expectedAuth {
				assert.NotNil(t, runtime.DefaultAuthentication, "credentials from --server-url must be sent")
			} else {
				assert.Nil(t, runtime.DefaultAuthentication)
			}
		})
	}
}

// TestSetupClientsAddsTrailingPath documents that a --server-url without a path is usable:
// go-openapi requires a base path.
func TestSetupClientsAddsTrailingPath(t *testing.T) {
	u, err := url.Parse("https://admin:admin@pmm-server-second:8443")
	require.NoError(t, err)

	globals := &flags.GlobalFlags{ServerURL: u, SkipTLSCertificateCheck: true} //nolint:exhaustruct
	SetupClients(globals)

	assert.Equal(t, "/", globals.ServerURL.Path)
}

// TestSetupClientsClonesTransport guards against reconfiguring TLS on http.DefaultTransport.
// Because go-openapi hands that global out, mutating it in place would leak PMM's TLS settings
// into every other HTTP client in the process - and, in tests, into every later test in the binary.
func TestSetupClientsClonesTransport(t *testing.T) {
	// Not parallel: SetupClients configures the package-level API clients.
	def, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	u, err := url.Parse("https://admin:admin@pmm-server-second:8443/")
	require.NoError(t, err)

	globals := &flags.GlobalFlags{ServerURL: u, SkipTLSCertificateCheck: true} //nolint:exhaustruct
	SetupClients(globals)

	assert.NotSame(t, def, tlsConfigOf(t), "SetupClients must configure a clone of http.DefaultTransport")

	// The HTTP/2 machinery may install an empty TLS config on the global, but none of PMM's
	// settings may end up there.
	if c := def.TLSClientConfig; c != nil {
		assert.Empty(t, c.ServerName)
		assert.False(t, c.InsecureSkipVerify)
	}
}
