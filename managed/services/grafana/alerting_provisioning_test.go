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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureAlertAnnotationsContactPoint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const webhookURL = "http://127.0.0.1:7772/internal/webhook"

	t.Run("provisions when missing and preserves existing policy", func(t *testing.T) {
		t.Parallel()

		var (
			postedCP    embeddedContactPoint
			cpCreated   bool
			puttedTree  map[string]any
			policyPut   bool
			disableProv string
		)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v := r.Header.Get("X-Disable-Provenance"); v != "" {
				disableProv = v
			}
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/provisioning/contact-points":
				_, _ = fmt.Fprint(w, `[{"name":"grafana-default-email","type":"email"}]`)
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/provisioning/contact-points":
				cpCreated = true
				b, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(b, &postedCP)
				w.WriteHeader(http.StatusAccepted)
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/provisioning/policies":
				_, _ = fmt.Fprint(w, `{"receiver":"grafana-default-email","group_by":["alertname"],"routes":[{"receiver":"user-slack"}]}`)
			case r.Method == http.MethodPut && r.URL.Path == "/api/v1/provisioning/policies":
				policyPut = true
				b, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(b, &puttedTree)
				w.WriteHeader(http.StatusAccepted)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		require.NoError(t, c.EnsureAlertAnnotationsContactPoint(ctx, webhookURL))

		assert.Equal(t, "true", disableProv)

		require.True(t, cpCreated)
		assert.Equal(t, alertAnnotationsReceiver, postedCP.Name)
		assert.Equal(t, "webhook", postedCP.Type)
		assert.Equal(t, webhookURL, postedCP.Settings["url"])

		require.True(t, policyPut)
		assert.Equal(t, "grafana-default-email", puttedTree["receiver"], "default receiver preserved")
		routes, _ := puttedTree["routes"].([]any)
		require.Len(t, routes, 2)
		assert.Equal(t, "user-slack", routes[0].(map[string]any)["receiver"], "existing user route preserved")
		ours := routes[1].(map[string]any)
		assert.Equal(t, alertAnnotationsReceiver, ours["receiver"])
		assert.Equal(t, true, ours["continue"])
	})

	t.Run("idempotent when already provisioned", func(t *testing.T) {
		t.Parallel()

		var cpCreated, policyPut bool
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/provisioning/contact-points":
				_, _ = fmt.Fprintf(w, `[{"name":%q,"type":"webhook"}]`, alertAnnotationsReceiver)
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/provisioning/contact-points":
				cpCreated = true
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/provisioning/policies":
				_, _ = fmt.Fprintf(w, `{"receiver":"grafana-default-email","routes":[{"receiver":%q,"continue":true}]}`, alertAnnotationsReceiver)
			case r.Method == http.MethodPut && r.URL.Path == "/api/v1/provisioning/policies":
				policyPut = true
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		c := NewClient(strings.TrimPrefix(ts.URL, "http://"))
		require.NoError(t, c.EnsureAlertAnnotationsContactPoint(ctx, webhookURL))
		assert.False(t, cpCreated, "contact point should not be recreated")
		assert.False(t, policyPut, "policy tree should not be rewritten")
	})
}
