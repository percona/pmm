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
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// A redirect is where a bearer leaks: net/http keeps the Authorization header when
	// only the scheme changes, so an https SEP pointing at http would hand PMM_SEP_TOKEN
	// to the wire. Nothing in SEP's API redirects, so the client refuses to follow one
	// and the 3xx surfaces as an unexpected status instead.
	elsewhere := newSEPStub(t, http.StatusOK, `{}`)
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.server.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirecting.Close)

	svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(redirecting.URL, "test-token")

	err := svc.probe.app.triggerRun(t.Context())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 307")
	assert.Empty(t, elsewhere.calls, "the redirect target must never see the request, let alone the bearer")
}

func TestSEPClientTransportErrorIsNotDoublePrefixed(t *testing.T) {
	t.Parallel()

	// http.Client.Do fails with a *url.Error, whose message already names the verb and
	// the full URL. Wrapping that in another "POST <url>:" printed both of them twice.
	stub := newSEPStub(t, http.StatusOK, `{}`)
	svc := stub.service(t)
	stub.server.Close()

	err := svc.probe.app.triggerRun(t.Context())

	require.Error(t, err)
	assert.Equal(t, 1, strings.Count(err.Error(), stub.server.URL), "the URL belongs in the message once: %s", err)
}
