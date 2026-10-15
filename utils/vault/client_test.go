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

package vault

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeVault is a minimal Vault API: logins issue numbered tokens, and reads return the secrets
// set in it, failing with 403 for any token it has not issued or has revoked.
type fakeVault struct {
	t *testing.T

	m        sync.Mutex
	tokens   map[string]bool
	issued   int
	logins   []map[string]any
	reads    int
	tokenTTL int64
	secrets  map[string]func() map[string]any
	headers  http.Header
}

func newFakeVault(t *testing.T) (*fakeVault, *httptest.Server) {
	t.Helper()

	fv := &fakeVault{
		t:       t,
		tokens:  map[string]bool{"static-token": true},
		secrets: make(map[string]func() map[string]any),
	}
	srv := httptest.NewServer(http.HandlerFunc(fv.serve))
	t.Cleanup(srv.Close)
	return fv, srv
}

func (fv *fakeVault) serve(w http.ResponseWriter, r *http.Request) {
	fv.m.Lock()
	defer fv.m.Unlock()

	fv.headers = r.Header.Clone()
	path := strings.TrimPrefix(r.URL.Path, "/v1/")

	if strings.HasSuffix(path, "/login") {
		var body map[string]any
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body["mount"] = strings.TrimSuffix(strings.TrimPrefix(path, "auth/"), "/login")
		fv.logins = append(fv.logins, body)

		fv.issued++
		token := "token-" + string(rune('0'+fv.issued))
		fv.tokens[token] = true
		writeJSON(fv.t, w, http.StatusOK, map[string]any{
			"auth": map[string]any{"client_token": token, "lease_duration": fv.tokenTTL},
		})
		return
	}

	if !fv.tokens[r.Header.Get("X-Vault-Token")] {
		writeJSON(fv.t, w, http.StatusForbidden, map[string]any{"errors": []string{"permission denied"}})
		return
	}

	secret, ok := fv.secrets[path]
	if !ok {
		writeJSON(fv.t, w, http.StatusNotFound, map[string]any{"errors": []string{}})
		return
	}

	fv.reads++
	writeJSON(fv.t, w, http.StatusOK, secret())
}

func (fv *fakeVault) revokeAll() {
	fv.m.Lock()
	defer fv.m.Unlock()
	clear(fv.tokens)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(v))
}

func kv2(data map[string]any) func() map[string]any {
	return func() map[string]any {
		return map[string]any{
			"lease_duration": 0,
			"data":           map[string]any{"data": data, "metadata": map[string]any{"version": 1}},
		}
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(path, []byte(content+"\n"), 0o600))
	return path
}

func TestClientTokenKV2(t *testing.T) {
	t.Parallel()

	fv, srv := newFakeVault(t)
	fv.secrets["secret/data/pmm/mysql"] = kv2(map[string]any{"username": "pmm", "password": "s3cr3t", "port": 3306})

	c, err := New(Config{Address: srv.URL, Token: "static-token", Namespace: "team"})
	require.NoError(t, err)

	data, err := c.Read(t.Context(), "/secret/data/pmm/mysql/")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"username": "pmm", "password": "s3cr3t", "port": "3306"}, data)
	assert.Equal(t, "team", fv.headers.Get("X-Vault-Namespace"))
	assert.Equal(t, "static-token", fv.headers.Get("X-Vault-Token"))

	v, err := c.GetRef(t.Context(), "vault:secret/data/pmm/mysql#password")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", v)

	_, err = c.Get(t.Context(), "secret/data/pmm/mysql", "missing")
	require.EqualError(t, err, "vault secret secret/data/pmm/mysql has no key missing")

	_, err = c.Read(t.Context(), "secret/data/pmm/nothing")
	require.EqualError(t, err, "vault GET secret/data/pmm/nothing: 404 Not Found")

	assert.Empty(t, fv.logins)
}

func TestClientCache(t *testing.T) {
	t.Parallel()

	t.Run("StaticSecret", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		password := "first"
		fv.secrets["secret/data/db"] = func() map[string]any { return kv2(map[string]any{"password": password})() }

		c, err := New(Config{Address: srv.URL, Token: "static-token", CacheTTL: time.Minute})
		require.NoError(t, err)
		now := time.Now()
		c.now = func() time.Time { return now }

		v, err := c.Get(t.Context(), "secret/data/db", "password")
		require.NoError(t, err)
		assert.Equal(t, "first", v)

		password = "second"
		now = now.Add(59 * time.Second)
		v, err = c.Get(t.Context(), "secret/data/db", "password")
		require.NoError(t, err)
		assert.Equal(t, "first", v)

		now = now.Add(time.Second)
		v, err = c.Get(t.Context(), "secret/data/db", "password")
		require.NoError(t, err)
		assert.Equal(t, "second", v)
		assert.Equal(t, 2, fv.reads)
	})

	t.Run("DynamicSecret", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		n := 0
		fv.secrets["database/creds/pmm"] = func() map[string]any {
			n++
			return map[string]any{
				"lease_id":       "database/creds/pmm/abc",
				"lease_duration": 3600,
				"data":           map[string]any{"username": "v-pmm-" + string(rune('0'+n)), "password": "p"},
			}
		}

		// The lease, not the far shorter cache TTL, decides: every read of a dynamic secret creates
		// new credentials, and readers must keep sharing one set of them.
		c, err := New(Config{Address: srv.URL, Token: "static-token", CacheTTL: time.Second})
		require.NoError(t, err)
		now := time.Now()
		c.now = func() time.Time { return now }

		u, err := c.Get(t.Context(), "database/creds/pmm", "username")
		require.NoError(t, err)
		assert.Equal(t, "v-pmm-1", u)

		now = now.Add(39 * time.Minute)
		u, err = c.Get(t.Context(), "database/creds/pmm", "username")
		require.NoError(t, err)
		assert.Equal(t, "v-pmm-1", u)

		now = now.Add(time.Minute)
		u, err = c.Get(t.Context(), "database/creds/pmm", "username")
		require.NoError(t, err)
		assert.Equal(t, "v-pmm-2", u)
	})
}

func TestClientLogin(t *testing.T) {
	t.Parallel()

	t.Run("AppRole", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.tokenTTL = 300
		fv.secrets["kv/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"password": "p"}} }

		c, err := New(Config{
			Address:      srv.URL,
			AuthMethod:   AuthMethodAppRole,
			AuthMount:    "/pmm-approle/",
			RoleID:       "role",
			SecretIDFile: writeFile(t, "secret-id"),
			CacheTTL:     time.Nanosecond,
		})
		require.NoError(t, err)
		now := time.Now()
		c.now = func() time.Time { return now }

		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)
		require.Len(t, fv.logins, 1)
		assert.Equal(t, map[string]any{"mount": "pmm-approle", "role_id": "role", "secret_id": "secret-id"}, fv.logins[0])
		assert.Equal(t, "token-1", fv.headers.Get("X-Vault-Token"))

		// The token is reused until two thirds of its TTL have passed.
		now = now.Add(199 * time.Second)
		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)
		assert.Len(t, fv.logins, 1)

		now = now.Add(time.Second)
		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)
		assert.Len(t, fv.logins, 2)
		assert.Equal(t, "token-2", fv.headers.Get("X-Vault-Token"))
	})

	t.Run("Kubernetes", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.secrets["kv/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"password": "p"}} }

		c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodKubernetes, Role: "pmm", JWTFile: writeFile(t, "jwt")})
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)
		require.Len(t, fv.logins, 1)
		assert.Equal(t, map[string]any{"mount": "kubernetes", "role": "pmm", "jwt": "jwt"}, fv.logins[0])
	})

	t.Run("RevokedTokenLogsInAgain", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.secrets["kv/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"password": "p"}} }

		c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodAppRole, RoleID: "role", SecretID: "id", CacheTTL: time.Nanosecond})
		require.NoError(t, err)

		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)

		fv.revokeAll()
		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)
		assert.Len(t, fv.logins, 2)
	})

	t.Run("TokenFileIsReadAgainWhenRejected", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.secrets["kv/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"password": "p"}} }
		tokenFile := writeFile(t, "old-token")

		c, err := New(Config{Address: srv.URL, TokenFile: tokenFile})
		require.NoError(t, err)
		now := time.Now()
		c.now = func() time.Time { return now }

		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.EqualError(t, err, "vault GET kv/pmm: 403 Forbidden: permission denied")

		// The failure is remembered briefly, so that a broken Vault is not asked again by every reader.
		require.NoError(t, os.WriteFile(tokenFile, []byte("static-token"), 0o600))
		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.EqualError(t, err, "vault GET kv/pmm: 403 Forbidden: permission denied")

		now = now.Add(errorCacheTTL)
		_, err = c.Get(t.Context(), "kv/pmm", "password")
		require.NoError(t, err)
	})
}

func TestClientDeletedKV2Version(t *testing.T) {
	t.Parallel()

	fv, srv := newFakeVault(t)
	fv.secrets["secret/data/gone"] = func() map[string]any {
		return map[string]any{"data": map[string]any{"data": nil, "metadata": map[string]any{"deletion_time": "now"}}}
	}

	c, err := New(Config{Address: srv.URL, Token: "static-token"})
	require.NoError(t, err)

	_, err = c.Read(t.Context(), "secret/data/gone")
	require.EqualError(t, err, "vault secret secret/data/gone: the secret version has no data, it may be deleted or destroyed")
}

func TestAWSLoginData(t *testing.T) {
	t.Parallel()

	provider := credentials.NewStaticCredentialsProvider("AKIDEXAMPLE", "secret", "session")
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)

	decode := func(t *testing.T, s any) string {
		t.Helper()
		b, err := base64.StdEncoding.DecodeString(s.(string))
		require.NoError(t, err)
		return string(b)
	}

	t.Run("Global", func(t *testing.T) {
		t.Parallel()

		data, err := awsLoginData(t.Context(), provider, "", "vault.example.com", now)
		require.NoError(t, err)
		assert.Equal(t, "POST", data["iam_http_request_method"])
		assert.Equal(t, "https://sts.amazonaws.com/", decode(t, data["iam_request_url"]))
		assert.Equal(t, "Action=GetCallerIdentity&Version=2011-06-15", decode(t, data["iam_request_body"]))

		var headers map[string][]string
		require.NoError(t, json.Unmarshal([]byte(decode(t, data["iam_request_headers"])), &headers))
		assert.Equal(t, []string{"vault.example.com"}, headers["X-Vault-Aws-Iam-Server-Id"])
		assert.Equal(t, []string{"session"}, headers["X-Amz-Security-Token"])
		require.Len(t, headers["Authorization"], 1)
		auth := headers["Authorization"][0]
		assert.True(t, strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20261015/us-east-1/sts/aws4_request"), auth)
		assert.Contains(t, auth, "x-vault-aws-iam-server-id")
	})

	t.Run("Regional", func(t *testing.T) {
		t.Parallel()

		data, err := awsLoginData(t.Context(), provider, "eu-west-1", "", now)
		require.NoError(t, err)
		assert.Equal(t, "https://sts.eu-west-1.amazonaws.com/", decode(t, data["iam_request_url"]))
		assert.Contains(t, decode(t, data["iam_request_headers"]), "/eu-west-1/sts/aws4_request")
	})
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		cfg Config
		err string
	}{
		"Disabled":        {cfg: Config{}},
		"Token":           {cfg: Config{Address: "https://vault:8200", Token: "t"}},
		"BadAddress":      {cfg: Config{Address: "vault:8200", Token: "t"}, err: "vault address must be an http(s) URL, got 'vault:8200'"},
		"NoToken":         {cfg: Config{Address: "https://vault:8200"}, err: "vault auth method token requires a token or a token file"},
		"UnknownMethod":   {cfg: Config{Address: "https://vault:8200", AuthMethod: "ldap"}, err: "unsupported vault auth method 'ldap', expected one of [token approle kubernetes aws]"},
		"AppRoleNoSecret": {cfg: Config{Address: "https://vault:8200", AuthMethod: AuthMethodAppRole, RoleID: "r"}, err: "vault auth method approle requires a role ID and a secret ID or a secret ID file"},
		"KubernetesNoRole": {
			cfg: Config{Address: "https://vault:8200", AuthMethod: AuthMethodKubernetes},
			err: "vault auth method kubernetes requires a role",
		},
		"AWS":         {cfg: Config{Address: "http://vault:8200", AuthMethod: AuthMethodAWS, Role: "pmm"}},
		"NegativeTTL": {cfg: Config{Address: "https://vault:8200", Token: "t", CacheTTL: -1}, err: "vault cache TTL must not be negative"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.err == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.err)
		})
	}
}

func TestConfigString(t *testing.T) {
	t.Parallel()

	cfg := Config{Address: "https://vault:8200", Token: "hvs.secret", RoleID: "role", SecretID: "secret-id"}
	s := fmt.Sprintf("%+v", struct{ Vault Config }{cfg})
	assert.NotContains(t, s, "hvs.secret")
	assert.NotContains(t, s, "secret-id")
	assert.Contains(t, s, "Address:https://vault:8200")
	assert.Contains(t, s, "RoleID:role")
	assert.Equal(t, "hvs.secret", cfg.Token)
}
