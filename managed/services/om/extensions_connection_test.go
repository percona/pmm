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
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/utils/env"
)

const (
	testSecretKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	// What the side-car's Settings.derive_internal_token returns for testSecretKey.
	testInternalToken = "6bf9f1933df380ff0f8666dbfcc1bab90b269098c5fdb3046ac786c7afb5effa"
)

func writeSecretKey(t *testing.T, dir, content string) string {
	t.Helper()

	path := filepath.Join(dir, "SECRET_KEY")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestDeriveInternalToken(t *testing.T) {
	t.Parallel()

	t.Run("matches the side-car, ignoring surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		token, err := deriveInternalToken(writeSecretKey(t, t.TempDir(), testSecretKey+"\n"))
		require.NoError(t, err)
		assert.Equal(t, testInternalToken, token)
	})

	t.Run("blank key", func(t *testing.T) {
		t.Parallel()

		_, err := deriveInternalToken(writeSecretKey(t, t.TempDir(), " \n"))
		require.ErrorContains(t, err, "is empty")
	})

	t.Run("missing key", func(t *testing.T) {
		t.Parallel()

		_, err := deriveInternalToken(filepath.Join(t.TempDir(), "absent"))
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestResolveExtensions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		enabled   bool
		address   string
		url       string
		token     string
		noKey     bool
		wantURL   string
		wantToken string
		wantWarn  string
	}{
		{
			name: "PMM Extensions disabled",
		},
		{
			name:      "documented deployment",
			enabled:   true,
			wantURL:   "http://pmm-extensions:9000",
			wantToken: testInternalToken,
		},
		{
			name:      "custom side-car address",
			enabled:   true,
			address:   "extensions:9100",
			wantURL:   "http://extensions:9100",
			wantToken: testInternalToken,
		},
		{
			name:      "explicit values win",
			enabled:   true,
			url:       "http://127.0.0.1:8000",
			token:     "explicit",
			wantURL:   "http://127.0.0.1:8000",
			wantToken: "explicit",
		},
		{
			name:      "explicit token with the default URL",
			enabled:   true,
			token:     "explicit",
			wantURL:   "http://pmm-extensions:9000",
			wantToken: "explicit",
		},
		{
			name:    "the local token never goes to an explicit URL",
			enabled: true,
			url:     "http://127.0.0.1:8000",
			wantURL: "http://127.0.0.1:8000",
		},
		{
			name:     "secrets not published, as under HA",
			enabled:  true,
			noKey:    true,
			wantURL:  "http://pmm-extensions:9000",
			wantWarn: "Cannot derive the PMM Extensions token",
		},
		{
			name:      "explicit plain-HTTP URL off this host",
			url:       "http://extensions.example.com:9000",
			token:     "explicit",
			wantURL:   "http://extensions.example.com:9000",
			wantToken: "explicit",
			wantWarn:  "will cross the network in clear text",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enabled := ""
			if tc.enabled {
				enabled = "1"
			}
			t.Setenv(env.EnableExtensions, enabled)
			t.Setenv("PMM_EXTENSIONS_ADDRESS", tc.address)
			secretsDir := t.TempDir()
			t.Setenv("PMM_DEV_EXTENSIONS_SECRETS_DIR", secretsDir)
			if !tc.noKey {
				writeSecretKey(t, secretsDir, testSecretKey)
			}
			logger, hook := logrustest.NewNullLogger()

			gotURL, gotToken := ResolveExtensions(tc.url, tc.token, logrus.NewEntry(logger))

			assert.Equal(t, tc.wantURL, gotURL)
			assert.Equal(t, tc.wantToken, gotToken)
			var warnings []string
			for _, entry := range hook.AllEntries() {
				if entry.Level == logrus.WarnLevel {
					warnings = append(warnings, entry.Message)
				}
			}
			if tc.wantWarn == "" {
				assert.Empty(t, warnings)
			} else {
				require.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], tc.wantWarn)
			}
		})
	}
}

func TestCheckAvailable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		code  int
		body  string
		token string
		want  codes.Code
		text  string
	}{
		{name: "available", code: http.StatusOK, body: `{"items":[],"total":0}`, token: "test-token", want: codes.OK},
		{
			name:  "token rejected",
			code:  http.StatusUnauthorized,
			body:  `{"detail":"Not authenticated"}`,
			token: "test-token",
			want:  codes.Internal,
			text:  "PMM's token does not match EXTENSIONS_INTERNAL_TOKEN on PMM Extensions (401 Unauthorized: Not authenticated)",
		},
		{
			name: "no token",
			code: http.StatusUnauthorized,
			body: `{"detail":"Not authenticated"}`,
			want: codes.Internal,
			text: "PMM has no token for PMM Extensions, set PMM_EXTENSIONS_TOKEN (401 Unauthorized",
		},
		{
			name:  "request refused",
			code:  http.StatusForbidden,
			body:  `{"detail":"Forbidden"}`,
			token: "test-token",
			want:  codes.Internal,
			text:  "PMM Extensions refused PMM's request (403 Forbidden)",
		},
		{
			name:  "app not served",
			code:  http.StatusNotFound,
			body:  `{"detail":"Not Found"}`,
			token: "test-token",
			want:  codes.FailedPrecondition,
			text:  "PMM Extensions does not serve the om_inventory app: Not Found",
		},
		{
			name:  "not the inventory app",
			code:  http.StatusOK,
			body:  `<html></html>`,
			token: "test-token",
			want:  codes.Internal,
			text:  "failed to decode the inventory app's answer",
		},
		{
			name:  "other failure",
			code:  http.StatusInternalServerError,
			body:  `{"detail":"boom"}`,
			token: "test-token",
			want:  codes.Internal,
			text:  "the inventory app answered 500 Internal Server Error: boom",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stub := newSEPStub(t, tc.code, tc.body)
			svc := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource(stub.server.URL, tc.token)

			err := svc.CheckAvailable(t.Context())

			assert.Equal(t, tc.want, status.Code(err))
			assert.Contains(t, status.Convert(err).Message(), tc.text)
			assert.Equal(t, "/api/apps/om_inventory/services", stub.path)
		})
	}

	t.Run("not configured", func(t *testing.T) {
		t.Parallel()

		err := (&Service{l: logrus.WithField("test", t.Name())}).WithProbeSource("", "").CheckAvailable(t.Context())
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Equal(t, extensionsNotConfigured, status.Convert(err).Message())
	})

	t.Run("unreachable", func(t *testing.T) {
		t.Parallel()

		stub := newSEPStub(t, http.StatusOK, `{}`)
		svc := stub.service(t)
		stub.server.Close()

		err := svc.CheckAvailable(t.Context())
		assert.Equal(t, codes.Unavailable, status.Code(err))
		assert.Contains(t, status.Convert(err).Message(), "PMM Extensions is unreachable: ")
	})
}
