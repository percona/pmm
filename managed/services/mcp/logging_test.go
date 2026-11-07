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

package mcp

import (
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoCredentialsInLogs calls every tool with each kind of caller credential against a PMM that answers,
// answers 401 and answers 500, and checks that the credential reached PMM and that no log entry, at any level, carries it.
func TestNoCredentialsInLogs(t *testing.T) {
	t.Parallel()

	const (
		token    = "glsa_LEAK_token_7f3a9c"
		password = "pw_LEAK_password_9d41b2"
		cookie   = "sess_LEAK_cookie_55e0aa"
	)
	encoded := base64.StdEncoding.EncodeToString([]byte("alice:" + password))
	secrets := []string{token, password, encoded, cookie}

	credentials := map[string]http.Header{
		"Bearer": {"Authorization": {"Bearer " + token}},
		"Basic":  {"Authorization": {"Basic " + encoded}},
		"Cookie": {"Cookie": {"pmm_session=" + cookie}},
	}

	// Each tool's calls, by its name. Every call but pmm_catalog's reaches its backing calls, so a failing PMM fails it
	// there; pmm_catalog has no backing call and refuses the invalid kind.
	calls := map[string][]map[string]any{
		"pmm_version":      {nil},
		"pmm_inventory":    {nil},
		"pmm_top_queries":  {nil},
		"pmm_query_detail": {{"queryid": "QID-AAA"}},
		"pmm_get_explain":  {{"service_id": "svc-1", "queryid": "QID-AAA"}},
		"pmm_get_schema": {
			{"service_id": "svc-1", "database": "shop", "tables": []string{"customers"}},
			{"queryid": "QID-AAA"},
		},
		"pmm_get_config":    {{"service_id": "svc-1"}},
		"pmm_catalog":       {{"kind": "bogus"}},
		"pmm_query_metrics": {{"query": "mysql_up"}},
	}
	requireEntries(t, listedTools(t), slices.Collect(maps.Keys(calls)))

	backends := map[string]func(*testing.T) *fakePMM{
		"Answered": func(t *testing.T) *fakePMM {
			t.Helper()
			fake, stub := newActionStub(t, 0)
			// A plan, so that pmm_get_schema's query form goes on to its tables.
			stub.explain = explainOutput(t, readFixture(t, "explain/mysql-8.0-join-aliases.json"), "SELECT 1")
			return fake
		},
		"Unauthorized": func(t *testing.T) *fakePMM {
			t.Helper()
			fake := newFakePMM(t, nil)
			body := readFixture(t, "error_401.json")
			fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusUnauthorized)
				_, _ = rw.Write(body)
			})
			return fake
		},
		"ServerError": func(t *testing.T) *fakePMM {
			t.Helper()
			fake := newFakePMM(t, qanRoutes())
			failWhen(t, fake, func(*http.Request) bool { return true })
			return fake
		},
	}

	for credName, header := range credentials {
		for backendName, backend := range backends {
			t.Run(credName+"/"+backendName, func(t *testing.T) {
				t.Parallel()

				fake := backend(t)
				var forwarded atomic.Int64
				next := fake.server.Config.Handler
				fake.server.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					if carries(req.Header, header) {
						forwarded.Add(1)
					}
					next.ServeHTTP(rw, req)
				})

				s := newQANService(t, fake, true)
				logger, hook := test.NewNullLogger()
				logger.SetLevel(logrus.TraceLevel)
				s.l = logrus.NewEntry(logger).WithField("component", "mcp")

				session := connectWithHeaders(t, s, header)
				for name, toolCalls := range calls {
					for _, args := range toolCalls {
						callTool(t, session, name, args)
					}
				}

				assert.Positive(t, forwarded.Load(), "no backing call carried the credential, so the logs could not have leaked it")

				entries := hook.AllEntries()
				require.NotEmpty(t, entries, "the tool calls must log, or this test checks nothing")
				for _, e := range entries {
					var text strings.Builder
					text.WriteString(e.Message)
					for k, v := range e.Data {
						fmt.Fprintf(&text, " %s=%v", k, v)
					}
					for _, secret := range secrets {
						assert.NotContains(t, text.String(), secret, "%s entry carries a credential", e.Level)
					}
				}
			})
		}
	}
}

// carries reports whether got holds every header of want with the same value.
func carries(got, want http.Header) bool {
	for key := range want {
		if got.Get(key) != want.Get(key) {
			return false
		}
	}
	return true
}
