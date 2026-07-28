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

package servererror

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tlsServerError performs a request against a TLS test server without trusting its
// certificate, returning the error a PMM Server call would fail with. The hostname argument
// selects the name the client uses in the TLS handshake, which is how the two distinct
// failures are produced: an untrusted issuer, and a certificate issued for a different host.
func tlsServerError(t *testing.T, hostname string) error {
	t.Helper()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	client := &http.Client{ //nolint:exhaustruct
		Transport: &http.Transport{ //nolint:exhaustruct
			// httptest certificates are issued for "example.com" and 127.0.0.1, so
			// dialing the server while claiming a different name mismatches the SANs.
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, u.Host)
			},
		},
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+hostname+"/v1/inventory/agents", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}

	require.Error(t, err)

	return err
}

func TestIsTLSCertificateError(t *testing.T) {
	t.Parallel()

	t.Run("untrusted issuer", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsTLSCertificateError(tlsServerError(t, "127.0.0.1")))
	})

	t.Run("host name mismatch", func(t *testing.T) {
		t.Parallel()

		assert.True(t, IsTLSCertificateError(tlsServerError(t, "pmm-server-second")))
	})

	t.Run("bare x509 errors", func(t *testing.T) {
		t.Parallel()

		// macOS and Windows verify through the system verifier and do not always wrap
		// the x509 error in tls.CertificateVerificationError.
		for name, err := range map[string]error{
			"hostname":  x509.HostnameError{Certificate: &x509.Certificate{}, Host: "pmm-server-second"}, //nolint:exhaustruct
			"authority": x509.UnknownAuthorityError{},                                                    //nolint:exhaustruct
			"invalid":   x509.CertificateInvalidError{Cert: &x509.Certificate{}, Reason: x509.Expired},   //nolint:exhaustruct
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				assert.True(t, IsTLSCertificateError(err))
				// Errors reach us wrapped in *url.Error by net/http.
				assert.True(t, IsTLSCertificateError(&url.Error{Op: "Put", URL: "https://pmm/", Err: err}))
			})
		}
	})

	t.Run("unrelated errors", func(t *testing.T) {
		t.Parallel()

		assert.False(t, IsTLSCertificateError(nil))
		assert.False(t, IsTLSCertificateError(errors.New("connection refused")))
		assert.False(t, IsTLSCertificateError(&url.Error{
			Op: "Put", URL: "https://pmm/", Err: errors.New("EOF"),
		}))
		// A handshake failure which is not about the certificate must not be reported
		// as one, otherwise --server-insecure-tls would be suggested for nothing.
		assert.False(t, IsTLSCertificateError(tls.RecordHeaderError{ //nolint:exhaustruct
			Msg: "first record does not look like a TLS handshake",
		}))
	})
}

func TestWrapTLSError(t *testing.T) {
	t.Parallel()

	certErr := &url.Error{
		Op:  "Put",
		URL: "https://pmm-server-second:8443/v1/inventory/agents/722fbfc8",
		Err: x509.HostnameError{Certificate: &x509.Certificate{}, Host: "pmm-server-second"}, //nolint:exhaustruct
	}

	t.Run("adds hint", func(t *testing.T) {
		t.Parallel()

		wrapped := WrapTLSError(certErr, "pmm-server-second", false)
		require.Error(t, wrapped)

		msg := wrapped.Error()
		// The original error stays first so existing output remains greppable.
		assert.True(t, strings.HasPrefix(msg, certErr.Error()+"."), msg)
		assert.Contains(t, msg, `not valid for host "pmm-server-second"`)
		assert.Contains(t, msg, InsecureTLSFlag)
		// The wrapped error must stay inspectable.
		require.ErrorIs(t, wrapped, certErr)
		assert.True(t, IsTLSCertificateError(wrapped))
	})

	t.Run("no hint once validation is disabled", func(t *testing.T) {
		t.Parallel()

		// Suggesting the flag the user already passed would be nonsense.
		assert.Equal(t, certErr, WrapTLSError(certErr, "pmm-server-second", true))
	})

	t.Run("no hint for unrelated errors", func(t *testing.T) {
		t.Parallel()

		other := errors.New("connection refused")
		assert.Equal(t, other, WrapTLSError(other, "pmm-server-second", false))
		assert.NoError(t, WrapTLSError(nil, "pmm-server-second", false))
	})

	t.Run("without a host", func(t *testing.T) {
		t.Parallel()

		msg := WrapTLSError(certErr, "", false).Error()
		assert.Contains(t, msg, "not valid for the requested host")
		assert.Contains(t, msg, InsecureTLSFlag)
	})
}

func TestAuthHint(t *testing.T) {
	t.Parallel()

	const (
		grpcUnauthenticated  = 16
		grpcInternal         = 13
		grpcPermissionDenied = 7
		grpcNotFound         = 5
	)

	for name, tc := range map[string]struct {
		httpCode int
		grpcCode int32
		expected string
	}{
		// What PMM Server actually returns for a wrong password.
		"rejected credentials": {401, grpcUnauthenticated, "Please check username and password"},
		// nginx auth_request accepts 401/403 only, so PMM Server reports internal auth
		// failures with HTTP 401 too. PMM-15186: those were misreported as bad credentials.
		"internal error mapped to 401": {401, grpcInternal, "Please check PMM Server logs"},
		// Older PMM Servers, and responses which never reach the API, carry no code.
		"401 without a gRPC code": {401, 0, "Please check username and password"},
		// Unauthenticated is conclusive on its own, whatever the status.
		"unauthenticated behind another status": {500, grpcUnauthenticated, "Please check username and password"},
		"permission denied":                     {403, grpcPermissionDenied, ""},
		"not found":                             {404, grpcNotFound, ""},
		"conflict":                              {409, 6, ""},
		"success":                               {200, 0, ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, AuthHint(tc.httpCode, tc.grpcCode))
		})
	}
}
