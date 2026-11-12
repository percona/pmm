// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package om

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

func TestCleartextToken(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		baseURL   string
		cleartext bool
	}{
		{"https://sep.example.com:8000", false},
		{"http://127.0.0.1:8000", false},
		{"http://[::1]:8000", false},
		{"http://localhost:8000", false},
		{"http://sep.example.com:8000", true},
		{"http://10.0.0.7:8000", true},
		{"http://host.docker.internal:8000", true},
		{"://nonsense", false},
	} {
		t.Run(tc.baseURL, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.cleartext, cleartextToken(tc.baseURL))
		})
	}
}

func TestSEPClientRefusesRedirects(t *testing.T) {
	t.Parallel()

	// net/http keeps the Authorization header on a scheme-only redirect, which could leak the bearer token.
	elsewhere := newSEPStub(t, http.StatusOK, `{}`)
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.server.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirecting.Close)

	svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(redirecting.URL, "test-token")

	err := svc.probe.app.setEnabled(asService(t.Context()), true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "answered 307")
	assert.Empty(t, elsewhere.calls, "the redirect target must never see the request, let alone the bearer")
}

func TestSEPClientBypassesProxy(t *testing.T) {
	t.Parallel()

	client := newExtensionsClient("http://pmm-extensions:9000", "test-token")

	transport, ok := client.http.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Nil(t, transport.Proxy, "the proxy environment must not see what is sent to PMM Extensions")
}

func TestSEPClientTransportErrorIsNotDoublePrefixed(t *testing.T) {
	t.Parallel()

	// http.Client.Do's *url.Error leads with the method and URL, so only its cause is kept.
	stub := newSEPStub(t, http.StatusOK, `{}`)
	svc := stub.service(t)
	stub.server.Close()

	err := svc.probe.app.setEnabled(asService(t.Context()), true)

	require.Error(t, err)
	assert.Equal(t, "PMM Extensions is unreachable: connection refused", status.Convert(err).Message())
}

func TestReadEnabledRefusesAConfigWithoutABooleanSwitch(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"missing":     `[{"key": "AUTO_SCAN", "value": true}]`,
		"not boolean": `[{"key": "ENABLED", "value": "true"}]`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc := newSEPStub(t, http.StatusOK, body).service(t)

			_, err := svc.probe.app.readEnabled(asService(t.Context()))

			require.Error(t, err)
		})
	}
}
