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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMetadata serves the given handler as a metadata service, and returns its URL.
func fakeMetadata(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestClientLoginAzure(t *testing.T) {
	t.Parallel()

	azureMetadata := func(t *testing.T, vmss string) (string, *[]*http.Request) {
		t.Helper()
		var requests []*http.Request
		u := fakeMetadata(t, func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Clone(r.Context()))
			if r.Header.Get("Metadata") != "true" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			switch r.URL.Path {
			case "/identity/oauth2/token":
				writeJSON(t, w, http.StatusOK, map[string]any{"access_token": "azure-token", "token_type": "Bearer"})
			case "/instance/compute":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"subscriptionId":    "sub-1",
					"resourceGroupName": "rg-1",
					"name":              "vm-1",
					"vmScaleSetName":    vmss,
				})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		})
		return u, &requests
	}

	t.Run("virtual machine with the system-assigned identity", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.secrets["secret/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"k": "v"}} }
		metadata, requests := azureMetadata(t, "")

		c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodAzure, Role: "pmm"})
		require.NoError(t, err)
		c.azureMetadataURL = metadata
		c.getenv = func(string) string { return "" } // not in a pod with a workload identity

		v, err := c.Get(t.Context(), "secret/pmm", "k")
		require.NoError(t, err)
		assert.Equal(t, "v", v)

		require.Len(t, fv.logins, 1)
		assert.Equal(t, map[string]any{
			"mount":               "azure",
			"role":                "pmm",
			"jwt":                 "azure-token",
			"subscription_id":     "sub-1",
			"resource_group_name": "rg-1",
			"vm_name":             "vm-1",
		}, fv.logins[0])

		require.Len(t, *requests, 2)
		token := (*requests)[0].URL.Query()
		assert.Equal(t, DefaultAzureResource, token.Get("resource"))
		assert.Empty(t, token.Get("client_id"))
	})

	t.Run("scale set with a user-assigned identity", func(t *testing.T) {
		t.Parallel()

		fv, srv := newFakeVault(t)
		fv.secrets["secret/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"k": "v"}} }
		metadata, requests := azureMetadata(t, "vmss-1")

		c, err := New(Config{
			Address:       srv.URL,
			AuthMethod:    AuthMethodAzure,
			AuthMount:     "azure-prod",
			Role:          "pmm",
			AzureResource: "https://vault.example.com",
			AzureClientID: "6d0c1f2e",
		})
		require.NoError(t, err)
		c.azureMetadataURL = metadata
		c.getenv = func(string) string { return "" } // not in a pod with a workload identity

		_, err = c.Get(t.Context(), "secret/pmm", "k")
		require.NoError(t, err)

		require.Len(t, fv.logins, 1)
		assert.Equal(t, "azure-prod", fv.logins[0]["mount"])
		assert.Equal(t, "vmss-1", fv.logins[0]["vmss_name"])
		assert.NotContains(t, fv.logins[0], "vm_name")

		token := (*requests)[0].URL.Query()
		assert.Equal(t, "https://vault.example.com", token.Get("resource"))
		assert.Equal(t, "6d0c1f2e", token.Get("client_id"))
	})

	t.Run("no managed identity", func(t *testing.T) {
		t.Parallel()

		_, srv := newFakeVault(t)
		metadata := fakeMetadata(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) })

		c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodAzure, Role: "pmm"})
		require.NoError(t, err)
		c.azureMetadataURL = metadata
		c.getenv = func(string) string { return "" } // not in a pod with a workload identity

		_, err = c.Get(t.Context(), "secret/pmm", "k")
		require.ErrorContains(t, err, "failed to get an access token of the managed identity for vault login")
		require.ErrorContains(t, err, "400 Bad Request")
	})
}

func TestClientLoginAzureWorkloadIdentity(t *testing.T) {
	t.Parallel()

	fv, srv := newFakeVault(t)
	fv.secrets["secret/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"k": "v"}} }

	var form url.Values
	var path string
	entra := fakeMetadata(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		form = r.PostForm
		writeJSON(t, w, http.StatusOK, map[string]any{"access_token": "entra-token", "token_type": "Bearer"})
	})

	tokenFile := writeFile(t, "service-account-token\n")
	env := map[string]string{
		"AZURE_FEDERATED_TOKEN_FILE": tokenFile,
		"AZURE_CLIENT_ID":            "client-1",
		"AZURE_TENANT_ID":            "tenant-1",
		"AZURE_AUTHORITY_HOST":       entra + "/",
	}
	c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodAzure, Role: "pmm", AzureResource: "https://vault.example.com/"})
	require.NoError(t, err)
	c.getenv = func(name string) string { return env[name] }
	c.azureMetadataURL = "http://127.0.0.1:1" // not asked

	v, err := c.Get(t.Context(), "secret/pmm", "k")
	require.NoError(t, err)
	assert.Equal(t, "v", v)

	assert.Equal(t, "/tenant-1/oauth2/v2.0/token", path)
	assert.Equal(t, url.Values{
		"grant_type":            []string{"client_credentials"},
		"client_id":             []string{"client-1"},
		"scope":                 []string{"https://vault.example.com/.default"},
		"client_assertion_type": []string{"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      []string{"service-account-token"},
	}, form)
	require.Len(t, fv.logins, 1)
	assert.Equal(t, map[string]any{"mount": "azure", "role": "pmm", "jwt": "entra-token"}, fv.logins[0])

	t.Run("without the tenant", func(t *testing.T) {
		t.Parallel()

		_, srv := newFakeVault(t)
		c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodAzure, Role: "pmm"})
		require.NoError(t, err)
		c.getenv = func(name string) string {
			return map[string]string{"AZURE_FEDERATED_TOKEN_FILE": tokenFile, "AZURE_CLIENT_ID": "client-1"}[name]
		}
		_, err = c.Get(t.Context(), "secret/pmm", "k")
		require.ErrorContains(t, err, "AZURE_CLIENT_ID and AZURE_TENANT_ID are needed")
	})
}

func TestClientLoginGCP(t *testing.T) {
	t.Parallel()

	fv, srv := newFakeVault(t)
	fv.tokenTTL = 3600
	fv.secrets["secret/pmm"] = func() map[string]any { return map[string]any{"data": map[string]any{"k": "v"}} }

	var audience string
	metadata := fakeMetadata(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" || r.URL.Path != "/instance/service-accounts/default/identity" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		audience = r.URL.Query().Get("audience")
		assert.Equal(t, "full", r.URL.Query().Get("format"))
		_, _ = w.Write([]byte("gce-identity-token"))
	})

	c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodGCP, Role: "pmm-agent", CacheTTL: time.Hour})
	require.NoError(t, err)
	c.gcpMetadataURL = metadata

	v, err := c.Get(t.Context(), "secret/pmm", "k")
	require.NoError(t, err)
	assert.Equal(t, "v", v)

	assert.Equal(t, "http://vault/pmm-agent", audience)
	require.Len(t, fv.logins, 1)
	assert.Equal(t, map[string]any{"mount": "gcp", "role": "pmm-agent", "jwt": "gce-identity-token"}, fv.logins[0])

	t.Run("not on Google Compute Engine", func(t *testing.T) {
		t.Parallel()

		_, srv := newFakeVault(t)
		c, err := New(Config{Address: srv.URL, AuthMethod: AuthMethodGCP, Role: "pmm-agent"})
		require.NoError(t, err)
		c.gcpMetadataURL = fakeMetadata(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })

		_, err = c.Get(t.Context(), "secret/pmm", "k")
		require.ErrorContains(t, err, "failed to get an identity token of the instance for vault login")
	})
}

func TestConfigValidateCloudMethods(t *testing.T) {
	t.Parallel()

	for _, method := range []string{AuthMethodAzure, AuthMethodGCP} {
		err := Config{Address: "https://vault:8200", AuthMethod: method}.Validate()
		require.EqualError(t, err, "vault auth method "+method+" requires a role")

		require.NoError(t, Config{Address: "https://vault:8200", AuthMethod: method, Role: "pmm"}.Validate())
	}
}
