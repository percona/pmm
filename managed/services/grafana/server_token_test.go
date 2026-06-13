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

package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	mu     sync.Mutex
	stored string
	saved  []string
}

func (f *fakeStore) Load(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stored, nil
}

func (f *fakeStore) Save(_ context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, token)
	f.stored = token
	return nil
}

// serviceAccountMux serves the Grafana service-account endpoints used by mintServerServiceToken,
// returning the given token key.
func serviceAccountMux(t *testing.T, tokenKey string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/serviceaccounts":
			_, _ = fmt.Fprint(w, `{"id":7}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/serviceaccounts/7":
			_, _ = fmt.Fprint(w, `{}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/serviceaccounts/7/tokens":
			_, _ = fmt.Fprint(w, `[]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/serviceaccounts/7/tokens":
			_, _ = fmt.Fprintf(w, `{"id":1,"key":%q}`, tokenKey)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestMintServerServiceToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var gotAuth, saName, tokenName string
	nameOf := func(r *http.Request) string {
		var body struct {
			Name string `json:"name"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		return body.Name
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/serviceaccounts":
			saName = nameOf(r)
			_, _ = fmt.Fprint(w, `{"id":7}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/api/serviceaccounts/7":
			_, _ = fmt.Fprint(w, `{}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/serviceaccounts/7/tokens":
			_, _ = fmt.Fprint(w, `[]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/serviceaccounts/7/tokens":
			tokenName = nameOf(r)
			_, _ = fmt.Fprint(w, `{"id":1,"key":"glsa_minted"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
	token, err := c.mintServerServiceToken(ctx)
	require.NoError(t, err)
	assert.Equal(t, "glsa_minted", token)
	assert.Equal(t, "Basic YWRtaW46YWRtaW4=", gotAuth) // admin:admin
	assert.Equal(t, "pmm-managed-sa", saName)
	assert.Equal(t, "pmm-managed-st", tokenName)
}

func TestServerAuthorization(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("uses stored token without minting", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError) // any mint attempt would fail the test
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		c.SetServerTokenStore(&fakeStore{stored: "stored_tok"})

		authz, err := c.serverAuthorization(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Bearer stored_tok", authz)
	})

	t.Run("mints and persists when none stored", func(t *testing.T) {
		t.Parallel()
		ts := httptest.NewServer(serviceAccountMux(t, "glsa_minted"))
		defer ts.Close()

		store := &fakeStore{}
		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		c.SetServerTokenStore(store)

		authz, err := c.serverAuthorization(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Bearer glsa_minted", authz)
		assert.Equal(t, []string{"glsa_minted"}, store.saved)
	})

	t.Run("falls back to admin basic-auth without a store", func(t *testing.T) {
		t.Parallel()
		c := NewClient("127.0.0.1:3000")
		authz, err := c.serverAuthorization(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Basic YWRtaW46YWRtaW4=", authz)
	})
}

func TestServerAuthRefreshOnUnauthorized(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var annAuths []string
	firstCall := true
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/annotations" && r.Method == http.MethodPost {
			annAuths = append(annAuths, r.Header.Get("Authorization"))
			if firstCall {
				firstCall = false
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"message":"unauthorized"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"id":55,"message":"Annotation added"}`)
			return
		}
		serviceAccountMux(t, "glsa_new")(w, r)
	}))
	defer ts.Close()

	store := &fakeStore{stored: "old_tok"}
	c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
	c.SetServerTokenStore(store)

	id, err := c.CreateAlertAnnotation(ctx, []string{"pmm_alert"}, time.Unix(1700000000, 0), "MySQL down")
	require.NoError(t, err)
	assert.Equal(t, 55, id)

	require.Len(t, annAuths, 2)
	assert.Equal(t, "Bearer old_tok", annAuths[0])
	assert.Equal(t, "Bearer glsa_new", annAuths[1])
	assert.Contains(t, store.saved, "glsa_new")
}
