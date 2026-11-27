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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	metrics "github.com/prometheus/client_golang/api"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// fakeGrafana serves its items, in the given order, to GET /api/annotations, filtered the way Grafana's SQL does,
// written independently of the handler's matcher.
type fakeGrafana struct {
	items  []string
	status int

	mu       sync.Mutex
	requests []*http.Request
}

func grafanaSQLPair(t string) (string, string) {
	parts := strings.Split(t, ":")
	if len(parts) == 1 {
		return strings.Trim(parts[0], " "), ""
	}
	return strings.Trim(parts[0], " "), strings.Trim(parts[1], " ")
}

func (g *fakeGrafana) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	g.mu.Lock()
	g.requests = append(g.requests, req)
	g.mu.Unlock()
	if g.status != 0 {
		rw.WriteHeader(g.status)
		_, _ = rw.Write([]byte(`{"message":"` + http.StatusText(g.status) + `"}`))
		return
	}

	q := req.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	matches := func(stored []string) bool {
		if len(q["tags"]) == 0 {
			return true
		}
		for _, want := range q["tags"] {
			wk, wv := grafanaSQLPair(want)
			for _, s := range stored {
				k, v := grafanaSQLPair(s)
				if wk != "" && k == wk && (wv == "" || v == wv) {
					return true
				}
			}
		}
		return false
	}

	var res []string
	for _, item := range g.items {
		var a struct {
			AlertID int      `json:"alertId"`
			Tags    []string `json:"tags"`
		}
		_ = json.Unmarshal([]byte(item), &a)
		if (q.Get("type") == "annotation" && a.AlertID != 0) || !matches(a.Tags) {
			continue
		}
		res = append(res, item)
		if len(res) == limit {
			break
		}
	}
	_, _ = rw.Write([]byte("[" + strings.Join(res, ",") + "]"))
}

// fakeVM serves label values; a selector containing "bad" is rejected, and a set status fails every request.
type fakeVM struct {
	values map[string][]string
	status int

	mu       sync.Mutex
	requests []*http.Request
}

func (v *fakeVM) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	v.mu.Lock()
	v.requests = append(v.requests, req)
	v.mu.Unlock()
	if v.status != 0 {
		rw.WriteHeader(v.status)
		return
	}
	label := strings.TrimSuffix(strings.TrimPrefix(req.URL.Path, "/api/v1/label/"), "/values")
	if strings.Contains(req.URL.Query().Get("match[]"), "bad") {
		rw.WriteHeader(http.StatusBadRequest)
		_, _ = rw.Write([]byte(`{"status":"error","error":"cannot parse"}`))
		return
	}
	err := json.NewEncoder(rw).Encode(struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}{Status: "success", Data: v.values[label]})
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
}

func (g *fakeGrafana) seen() []*http.Request {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.requests)
}

func (v *fakeVM) seen() []*http.Request {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.requests)
}

func annotationJSON(t *testing.T, id int, tags ...string) string {
	t.Helper()

	return itemJSON(t, id, 0, tags)
}

// alertJSON is an alert state history row, which Grafana stores as an annotation too.
func alertJSON(t *testing.T, id int, tags ...string) string {
	t.Helper()

	return itemJSON(t, id, 1, tags)
}

func itemJSON(t *testing.T, id, alertID int, tags []string) string {
	t.Helper()

	b, err := json.Marshal(struct {
		ID           int      `json:"id"`
		AlertID      int      `json:"alertId"`
		DashboardUID string   `json:"dashboardUID"`
		Time         int      `json:"time"`
		TimeEnd      int      `json:"timeEnd"`
		Text         string   `json:"text"`
		Tags         []string `json:"tags"`
	}{ID: id, AlertID: alertID, Time: 1700000000000 + id, TimeEnd: 1700000000000 + id, Text: fmt.Sprintf("annotation %d", id), Tags: tags})
	require.NoError(t, err)
	return string(b)
}

func setupScopedAnnotations(t *testing.T, g *fakeGrafana, v *fakeVM) (*ScopedAnnotationsHandler, *mockCallerAuthenticator) {
	t.Helper()

	gs := httptest.NewServer(g)
	t.Cleanup(gs.Close)
	vs := httptest.NewServer(v)
	t.Cleanup(vs.Close)

	vm, err := metrics.NewClient(metrics.Config{Address: vs.URL})
	require.NoError(t, err)

	auth := newMockCallerAuthenticator(t)
	h := &ScopedAnnotationsHandler{
		auth: auth,
		c:    NewClient(strings.TrimPrefix(gs.URL, "http://")),
		vm:   vm,
		l:    logrus.WithField("test", t.Name()),
	}
	return h, auth
}

func newScopedRequest(t *testing.T, method string, q url.Values) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, "/scoped-annotations?"+q.Encode(), nil)
	req.Header.Set("Cookie", "grafana_session=abc")
	req.Header.Set("Authorization", "Bearer glsa_abc")
	req.Header.Set("X-Grafana-Org-Id", "2")
	return req
}

// serveScoped sends a GET to h, which must ask auth once for that caller's filters and may expand its scope tags.
func serveScoped(t *testing.T, h *ScopedAnnotationsHandler, auth *mockCallerAuthenticator, q url.Values, filters []string, authErr *authError) *httptest.ResponseRecorder {
	t.Helper()

	req := newScopedRequest(t, http.MethodGet, q)
	auth.On("callerFilters", mock.Anything, req, h.l).Return(filters, authErr == nil, authErr).Once()

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	return rw
}

// serveRejected sends a request that h must refuse before authenticating the caller.
func serveRejected(t *testing.T, h *ScopedAnnotationsHandler, method string, q url.Values) *httptest.ResponseRecorder {
	t.Helper()

	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, newScopedRequest(t, method, q))
	return rw
}

func responseIDs(t *testing.T, rw *httptest.ResponseRecorder) []int {
	t.Helper()

	var items []struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &items))
	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}

func TestScopedAnnotationsHandler(t *testing.T) {
	t.Parallel()

	const (
		nodeScope    = `pmm_scope:node_name:mysql_up{environment=~"prod",node_name=~"node\.1"}`
		serviceScope = `pmm_scope:service_name:mysql_up{environment=~"prod",service_name=~".+"}`
	)

	t.Run("expands scopes and returns Grafana's items", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{items: []string{
			annotationJSON(t, 4, "node.1"),
			annotationJSON(t, 3, "svc-9"),
			annotationJSON(t, 2, "svc-1"),
			annotationJSON(t, 1, "pmm_annotation"),
		}}
		v := &fakeVM{values: map[string][]string{
			"node_name":    {"node.1"},
			"service_name": {"svc-2", "svc-1"},
		}}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{
			"tags":     {"pmm_annotation", nodeScope, serviceScope},
			"from":     {"1700000000000"},
			"to":       {"1700003600000"},
			"limit":    {"50"},
			"matchAny": {"true"},
		}, nil, nil)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.Equal(t, []int{4, 2, 1}, responseIDs(t, rw))
		var items []json.RawMessage
		require.NoError(t, json.Unmarshal(rw.Body.Bytes(), &items))
		assert.JSONEq(t, g.items[0], string(items[0]))

		require.Len(t, v.seen(), 2)
		nodeQuery := v.seen()[0].URL.Query()
		assert.Equal(t, "/api/v1/label/node_name/values", v.seen()[0].URL.Path)
		assert.Equal(t, `mysql_up{environment=~"prod",node_name=~"node\\.1"}`, nodeQuery.Get("match[]"))
		assert.Equal(t, "1700000000.000", nodeQuery.Get("start"))
		assert.Equal(t, "1700003600.000", nodeQuery.Get("end"))
		assert.Empty(t, nodeQuery["extra_filters[]"])

		require.Len(t, g.seen(), 1)
		gq := g.seen()[0].URL.Query()
		assert.Equal(t, []string{"pmm_annotation", "node.1", "svc-1", "svc-2"}, gq["tags"])
		assert.Equal(t, "true", gq.Get("matchAny"))
		assert.Equal(t, "50", gq.Get("limit"))
		assert.Equal(t, "1700000000000", gq.Get("from"))
		assert.Equal(t, "grafana_session=abc", g.seen()[0].Header.Get("Cookie"))
		assert.Equal(t, "Bearer glsa_abc", g.seen()[0].Header.Get("Authorization"))
		assert.Equal(t, "2", g.seen()[0].Header.Get("X-Grafana-Org-Id"))
	})

	t.Run("limits names with the caller's LBAC filters", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{}
		v := &fakeVM{values: map[string][]string{"service_name": {"svc-1"}}}
		filters := []string{`{environment="env-1"}`, `{environment="env-2"}`}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, filters, nil)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		require.Len(t, v.seen(), 1)
		assert.Equal(t, filters, v.seen()[0].URL.Query()["extra_filters[]"])
	})

	t.Run("matches in memory from 500 names on and keeps the newest", func(t *testing.T) {
		t.Parallel()

		names := make([]string, 0, maxNamesPerQuery+1)
		names = append(names, "host:a:b")
		for i := range maxNamesPerQuery {
			names = append(names, fmt.Sprintf("svc-%03d", i))
		}
		g := &fakeGrafana{items: []string{
			annotationJSON(t, 13, "host:a"),
			alertJSON(t, 14, "svc-001"),
			annotationJSON(t, 11, "other"),
			annotationJSON(t, 10, "svc-001"),
			annotationJSON(t, 12, "pmm_annotation"),
		}}
		v := &fakeVM{values: map[string][]string{"service_name": names}}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {"pmm_annotation", serviceScope}, "matchAny": {"true"}}, nil, nil)
		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.Equal(t, []int{13, 12, 10}, responseIDs(t, rw))

		require.Len(t, g.seen(), 1)
		gq := g.seen()[0].URL.Query()
		assert.Empty(t, gq["tags"])
		assert.Empty(t, gq["matchAny"])
		assert.Equal(t, "annotation", gq.Get("type"))
		assert.Equal(t, strconv.Itoa(windowReadLimit), gq.Get("limit"))

		rw = serveScoped(t, h, auth, url.Values{"tags": {"pmm_annotation", serviceScope}, "limit": {"2"}}, nil, nil)
		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.Equal(t, []int{13, 12}, responseIDs(t, rw))
	})

	t.Run("matches like Grafana's tag query on both paths", func(t *testing.T) {
		t.Parallel()

		items := []string{
			annotationJSON(t, 25, "svc-9"),
			annotationJSON(t, 24, "other:svc-7"),
			annotationJSON(t, 23, "svc-8"),
			// Stored as typed by an update through Grafana's API.
			annotationJSON(t, 22, " svc-7"),
			annotationJSON(t, 21, "db:primary"),
		}
		names := []string{"db", "svc-7", "svc-8"}
		fillers := make([]string, 0, maxNamesPerQuery)
		for i := range maxNamesPerQuery {
			fillers = append(fillers, fmt.Sprintf("filler-%03d", i))
		}

		for name, values := range map[string][]string{
			"tag query":       names,
			"in-memory match": append(slices.Clone(names), fillers...),
		} {
			g := &fakeGrafana{items: items}
			v := &fakeVM{values: map[string][]string{"service_name": values}}
			h, auth := setupScopedAnnotations(t, g, v)

			rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, nil)

			require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
			assert.Equal(t, []int{23, 22, 21}, responseIDs(t, rw), name)
		}
	})

	t.Run("serves plain tags when VictoriaMetrics rejects the selector", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{items: []string{annotationJSON(t, 2, "svc-1"), annotationJSON(t, 1, "pmm_annotation")}}
		v := &fakeVM{}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {"pmm_annotation", `pmm_scope:service_name:bad{`}}, nil, nil)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.Equal(t, []int{1}, responseIDs(t, rw))
		require.Len(t, g.seen(), 1)
		assert.Equal(t, []string{"pmm_annotation"}, g.seen()[0].URL.Query()["tags"])
	})

	t.Run("serves plain tags when VictoriaMetrics fails", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{items: []string{annotationJSON(t, 2, "svc-1"), annotationJSON(t, 1, "pmm_annotation")}}
		v := &fakeVM{status: http.StatusServiceUnavailable}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {"pmm_annotation", serviceScope}}, nil, nil)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.Equal(t, []int{1}, responseIDs(t, rw))
		require.Len(t, v.seen(), 1)
	})

	t.Run("returns an empty list when nothing is selected", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{}
		v := &fakeVM{}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, nil)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.JSONEq(t, `[]`, rw.Body.String())
		assert.Empty(t, g.seen())
	})

	t.Run("rejects an invalid scope tag", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{}
		v := &fakeVM{}
		h, _ := setupScopedAnnotations(t, g, v)

		rw := serveRejected(t, h, http.MethodGet, url.Values{"tags": {`pmm_scope:environment:up{}`}})

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.JSONEq(t, `{"message":"invalid scope tag"}`, rw.Body.String())
		assert.Empty(t, v.seen())
		assert.Empty(t, g.seen())
	})

	t.Run("rejects an invalid limit", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{}
		v := &fakeVM{}
		h, _ := setupScopedAnnotations(t, g, v)

		rw := serveRejected(t, h, http.MethodGet, url.Values{"tags": {serviceScope}, "limit": {"ten"}})

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.JSONEq(t, `{"message":"invalid limit"}`, rw.Body.String())
		assert.Empty(t, v.seen())
		assert.Empty(t, g.seen())
	})

	t.Run("rejects an invalid time range", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{}
		v := &fakeVM{}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}, "from": {"now-6h"}}, nil, nil)

		assert.Equal(t, http.StatusBadRequest, rw.Code)
		assert.JSONEq(t, `{"message":"invalid time range"}`, rw.Body.String())
		assert.Empty(t, v.seen())
		assert.Empty(t, g.seen())
	})

	t.Run("returns the authentication error", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{}
		v := &fakeVM{}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, &authError{code: codes.Unauthenticated, message: "Unauthorized"})

		assert.Equal(t, http.StatusUnauthorized, rw.Code)
		assert.JSONEq(t, `{"message":"Unauthorized"}`, rw.Body.String())
		assert.Empty(t, v.seen())
	})

	t.Run("passes Grafana's permission error through", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{status: http.StatusForbidden}
		v := &fakeVM{values: map[string][]string{"service_name": {"svc-1"}}}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, nil)

		assert.Equal(t, http.StatusForbidden, rw.Code)
		assert.JSONEq(t, `{"message":"Forbidden"}`, rw.Body.String())
	})

	t.Run("returns Bad Gateway when Grafana fails", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{status: http.StatusInternalServerError}
		v := &fakeVM{values: map[string][]string{"service_name": {"svc-1"}}}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, nil)

		assert.Equal(t, http.StatusBadGateway, rw.Code)
		assert.JSONEq(t, `{"message":"Bad Gateway"}`, rw.Body.String())
		require.Len(t, g.seen(), 1)
	})

	t.Run("serves plain tags only to a caller whose scope tags may not be expanded", func(t *testing.T) {
		t.Parallel()

		g := &fakeGrafana{items: []string{annotationJSON(t, 2, "svc-1"), annotationJSON(t, 1, "pmm_annotation")}}
		v := &fakeVM{values: map[string][]string{"service_name": {"svc-1"}}}
		h, auth := setupScopedAnnotations(t, g, v)
		req := newScopedRequest(t, http.MethodGet, url.Values{"tags": {"pmm_annotation", serviceScope}})
		auth.On("callerFilters", mock.Anything, req, h.l).Return(nil, false, nil).Once()

		rw := httptest.NewRecorder()
		h.ServeHTTP(rw, req)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		assert.Equal(t, []int{1}, responseIDs(t, rw))
		assert.Empty(t, v.seen())
		require.Len(t, g.seen(), 1)
		assert.Equal(t, []string{"pmm_annotation"}, g.seen()[0].URL.Query()["tags"])
	})

	t.Run("returns the caller's refusal with its status", func(t *testing.T) {
		t.Parallel()

		for code, status := range map[codes.Code]int{
			codes.PermissionDenied: http.StatusForbidden,
			codes.Internal:         http.StatusInternalServerError,
		} {
			v := &fakeVM{}
			h, auth := setupScopedAnnotations(t, &fakeGrafana{}, v)

			rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, &authError{code: code, message: "Refused"})

			assert.Equal(t, status, rw.Code, code)
			assert.JSONEq(t, `{"message":"Refused"}`, rw.Body.String())
			assert.Empty(t, v.seen())
		}
	})

	t.Run("returns Bad Gateway for an annotation it can't decode", func(t *testing.T) {
		t.Parallel()

		names := make([]string, 0, maxNamesPerQuery)
		for i := range maxNamesPerQuery {
			names = append(names, fmt.Sprintf("svc-%03d", i))
		}
		g := &fakeGrafana{items: []string{annotationJSON(t, 2, "svc-001"), `{"id":"two","tags":["svc-001"]}`}}
		v := &fakeVM{values: map[string][]string{"service_name": names}}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}}, nil, nil)

		assert.Equal(t, http.StatusBadGateway, rw.Code)
		assert.JSONEq(t, `{"message":"Bad Gateway"}`, rw.Body.String())
	})

	t.Run("uses Grafana's default limit and caps large ones", func(t *testing.T) {
		t.Parallel()

		for limit, sent := range map[string]string{
			"0":          "100",
			"-5":         "100",
			"2000000000": strconv.Itoa(windowReadLimit),
		} {
			g := &fakeGrafana{}
			v := &fakeVM{values: map[string][]string{"service_name": {"svc-1"}}}
			h, auth := setupScopedAnnotations(t, g, v)

			rw := serveScoped(t, h, auth, url.Values{"tags": {serviceScope}, "limit": {limit}}, nil, nil)

			require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
			require.Len(t, g.seen(), 1)
			assert.Equal(t, sent, g.seen()[0].URL.Query().Get("limit"), "limit=%s", limit)
		}
	})

	t.Run("chooses the window read by names, not plain tags", func(t *testing.T) {
		t.Parallel()

		tags := make([]string, 0, maxNamesPerQuery+2)
		for i := range maxNamesPerQuery + 1 {
			tags = append(tags, fmt.Sprintf("plain-%03d", i))
		}
		tags = append(tags, serviceScope)
		g := &fakeGrafana{}
		v := &fakeVM{values: map[string][]string{"service_name": {"svc-1"}}}
		h, auth := setupScopedAnnotations(t, g, v)

		rw := serveScoped(t, h, auth, url.Values{"tags": tags}, nil, nil)

		require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
		require.Len(t, g.seen(), 1)
		gq := g.seen()[0].URL.Query()
		assert.Len(t, gq["tags"], maxNamesPerQuery+2)
		assert.Equal(t, "true", gq.Get("matchAny"))
	})

	t.Run("allows only GET", func(t *testing.T) {
		t.Parallel()

		h, _ := setupScopedAnnotations(t, &fakeGrafana{}, &fakeVM{})

		rw := serveRejected(t, h, http.MethodPost, url.Values{"tags": {serviceScope}})

		assert.Equal(t, http.StatusMethodNotAllowed, rw.Code)
		assert.JSONEq(t, `{"message":"Method Not Allowed"}`, rw.Body.String())
		assert.Equal(t, http.MethodGet, rw.Header().Get("Allow"))
	})
}

func TestParseScopeTags(t *testing.T) {
	t.Parallel()

	plain, scopes, err := parseScopeTags([]string{"pmm_annotation", `pmm_scope:node_name:up{a=~"b:c"}`})
	require.NoError(t, err)
	assert.Equal(t, []string{"pmm_annotation"}, plain)
	assert.Equal(t, []scope{{label: "node_name", selector: `up{a=~"b:c"}`}}, scopes)

	// A multi-select of a few hundred names must still fit.
	_, scopes, err = parseScopeTags([]string{`pmm_scope:service_name:up{service_name=~"(` + strings.Repeat(`svc\-0001|`, 1000) + `)"}`})
	require.NoError(t, err)
	assert.Len(t, scopes, 1)

	tooMany := slices.Repeat([]string{`pmm_scope:node_name:up`}, maxScopeTags+1)
	for name, tags := range map[string][]string{
		"unknown label":  {`pmm_scope:environment:up`},
		"no selector":    {`pmm_scope:node_name:`},
		"no label":       {`pmm_scope:node_name`},
		"long selector":  {`pmm_scope:node_name:` + strings.Repeat("x", maxScopeSelectorLength+1)},
		"too many tags":  tooMany,
		"empty prefixed": {`pmm_scope:`},
	} {
		_, _, err := parseScopeTags(tags)
		assert.ErrorIs(t, err, errInvalidScopeTag, name)
	}
}

func TestParseTag(t *testing.T) {
	t.Parallel()

	for in, expected := range map[string]tagPair{
		"svc-1":     {key: "svc-1"},
		" svc-1 ":   {key: "svc-1"},
		"host:a":    {key: "host", value: "a"},
		"host:a:b":  {key: "host", value: "a"},
		"host:":     {key: "host"},
		"host: a ":  {key: "host", value: "a"},
		"node.1.db": {key: "node.1.db"},
		":a":        {value: "a"},
	} {
		assert.Equal(t, expected, parseTag(in), in)
	}
}
