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

package apitransport

import (
	"errors"
	"net/http"
	"testing"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertDefaultTransportUntouched checks that none of PMM's settings ended up on the
// process-wide transport. The HTTP/2 machinery may install an empty TLS config on it, so the
// check is on the individual settings rather than on the config being absent.
func assertDefaultTransportUntouched(t *testing.T, def *http.Transport) {
	t.Helper()

	// Disabling HTTP/2 means installing an empty TLSNextProto. The HTTP/2 machinery installs
	// a populated one of its own, and an untouched transport has none at all, so an empty map
	// on the global could only be ours.
	if def.TLSNextProto != nil {
		assert.NotEmpty(t, def.TLSNextProto, "HTTP/2 must not be disabled on the global transport")
	}

	if c := def.TLSClientConfig; c != nil {
		assert.Empty(t, c.ServerName)
		assert.False(t, c.InsecureSkipVerify)
	}
}

// TestConfigureClonesDefaultTransport is the guard for PMM-15186: go-openapi hands out
// http.DefaultTransport, so configuring it in place would disable certificate validation for
// every other HTTP client in the process.
func TestConfigureClonesDefaultTransport(t *testing.T) {
	t.Parallel()

	def, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	rt := httptransport.New("pmm-server-second:8443", "/", []string{"https"})

	original, ok := rt.Transport.(*http.Transport)
	require.True(t, ok)
	require.Same(t, def, original, "go-openapi is expected to hand out the global transport")

	Configure(rt, "https", "pmm-server-second", true)

	configured, ok := rt.Transport.(*http.Transport)
	require.True(t, ok)
	assert.NotSame(t, original, configured, "the transport must be a clone")

	require.NotNil(t, configured.TLSClientConfig)
	// ServerName is what makes the certificate mismatch reported in PMM-15186 detectable.
	assert.Equal(t, "pmm-server-second", configured.TLSClientConfig.ServerName)
	assert.True(t, configured.TLSClientConfig.InsecureSkipVerify)
	// HTTP/2 must be disabled on the clone, and only there.
	assert.NotNil(t, configured.TLSNextProto)
	assert.Empty(t, configured.TLSNextProto)

	assertDefaultTransportUntouched(t, def)
}

// TestConfigureValidatesCertificates checks that the certificate check stays on without
// --server-insecure-tls, which is the whole point of pinning ServerName.
func TestConfigureValidatesCertificates(t *testing.T) {
	t.Parallel()

	rt := httptransport.New("pmm-server-second:8443", "/", []string{"https"})
	Configure(rt, "https", "pmm-server-second", false)

	configured, ok := rt.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, configured.TLSClientConfig)
	assert.False(t, configured.TLSClientConfig.InsecureSkipVerify)
	assert.Equal(t, "pmm-server-second", configured.TLSClientConfig.ServerName)
}

// TestConfigureLocal covers the plain HTTP targets: the pmm-agent local API needs HTTP/1.1
// without any TLS configuration, and still must not touch the global transport.
func TestConfigureLocal(t *testing.T) {
	t.Parallel()

	def, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	rt := httptransport.New("127.0.0.1:7777", "/", []string{"http"})
	ConfigureLocal(rt)

	configured, ok := rt.Transport.(*http.Transport)
	require.True(t, ok)
	assert.NotSame(t, def, configured)
	assert.NotNil(t, configured.TLSNextProto)
	assert.Empty(t, configured.TLSNextProto)

	// Nil rather than merely empty: Clone inherits DefaultTransport's ["h2", "http/1.1"]
	// ALPN list, which an emptied TLSNextProto does not clear - see
	// TestNoTLSConfigurationLeavesNoALPNAdvertisement.
	assert.Nil(t, configured.TLSClientConfig)

	assertDefaultTransportUntouched(t, def)
}

// TestConfigureHTTPSchemeSkipsTLS documents that Configure only installs a TLS configuration
// for https, so that an http:// PMM Server URL is not silently given one.
func TestConfigureHTTPSchemeSkipsTLS(t *testing.T) {
	t.Parallel()

	rt := httptransport.New("pmm-server:80", "/", []string{"http"})
	Configure(rt, "http", "pmm-server", true)

	configured, ok := rt.Transport.(*http.Transport)
	require.True(t, ok)

	// Nil rather than merely empty: Clone inherits DefaultTransport's ["h2", "http/1.1"]
	// ALPN list, which an emptied TLSNextProto does not clear - see
	// TestNoTLSConfigurationLeavesNoALPNAdvertisement.
	assert.Nil(t, configured.TLSClientConfig)
}

// TestNoTLSConfigurationLeavesNoALPNAdvertisement is the guard for the ALPN half of disabling
// HTTP/2. Clone runs DefaultTransport's lazy HTTP/2 setup first, which puts an
// ["h2", "http/1.1"] NextProtos list on its TLS config; the clone inherits it, and emptying
// TLSNextProto does not remove it. A transport left in that state offers h2 with no handler
// registered for it, so a server which accepts h2 - reachable from an http:// URL that
// redirects to https - would be spoken to in HTTP/1.1 framing over an h2 connection.
func TestNoTLSConfigurationLeavesNoALPNAdvertisement(t *testing.T) {
	t.Parallel()

	for name, configure := range map[string]func(*httptransport.Runtime){
		"ConfigureLocal": ConfigureLocal,
		"Configure(http)": func(rt *httptransport.Runtime) {
			Configure(rt, "http", "pmm-server", false)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rt := httptransport.New("pmm-server:80", "/", []string{"http"})
			configure(rt)

			configured, ok := rt.Transport.(*http.Transport)
			require.True(t, ok)
			assert.Empty(t, configured.TLSNextProto)

			if c := configured.TLSClientConfig; c != nil {
				assert.NotContains(t, c.NextProtos, "h2")
			}
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestConfigureRejectsForeignTransport documents the panic: a runtime built on something other
// than an *http.Transport cannot be configured, and must not be left half-configured.
func TestConfigureRejectsForeignTransport(t *testing.T) {
	t.Parallel()

	rt := httptransport.New("127.0.0.1:7777", "/", []string{"http"})
	rt.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("not implemented")
	})

	assert.Panics(t, func() { ConfigureLocal(rt) })
	assert.Panics(t, func() { Configure(rt, "https", "pmm-server", false) })
}
