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
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	metrics "github.com/prometheus/client_golang/api"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
)

const (
	// A scope tag carries a series selector: "pmm_scope:<label>:<selector>".
	scopeTagPrefix = "pmm_scope:"
	maxScopeTags   = 4
	// Nginx refuses a longer request line (large_client_header_buffers in pmm.conf), so this
	// only bounds direct calls; a multi-select of many names must still fit.
	maxScopeSelectorLength = 64 << 10

	// From this many names on, one read of the time window matched here is faster than a tag query.
	maxNamesPerQuery = 500
	// That read takes at most this many of the newest annotations in the window, and no caller gets more.
	windowReadLimit = 100000

	// Grafana's own default for a query without a limit.
	defaultAnnotationsLimit = 100
)

// scopeLabels are the labels a scope tag may expand.
var scopeLabels = []string{"node_name", "service_name"}

var (
	errInvalidScopeTag  = errors.New("invalid scope tag")
	errInvalidTimeRange = errors.New("invalid time range")
	errInvalidLimit     = errors.New("invalid limit")
)

type scope struct {
	label    string
	selector string
}

// ScopedAnnotationsHandler expands scope tags into the names the caller may see and returns Grafana's own annotations for them.
type ScopedAnnotationsHandler struct {
	auth callerAuthenticator
	c    *Client
	vm   metrics.Client
	l    *logrus.Entry
}

// NewScopedAnnotationsHandler creates a handler for scoped annotation queries.
func NewScopedAnnotationsHandler(auth *AuthServer, c *Client, vm metrics.Client) *ScopedAnnotationsHandler {
	return &ScopedAnnotationsHandler{
		auth: auth,
		c:    c,
		vm:   vm,
		l:    logrus.WithField("component", "grafana/scoped-annotations"),
	}
}

// ServeHTTP serves GET /graph/api/annotations requests that nginx routes here when they carry a scope tag.
func (h *ScopedAnnotationsHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()

	if req.Method != http.MethodGet {
		rw.Header().Set("Allow", http.MethodGet)
		h.writeError(rw, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	ctx := req.Context()
	q := req.URL.Query()

	tags, scopes, err := parseScopeTags(q["tags"])
	if err != nil {
		h.writeError(rw, http.StatusBadRequest, err.Error())
		return
	}

	limit, err := parseLimit(q.Get("limit"))
	if err != nil {
		h.writeError(rw, http.StatusBadRequest, err.Error())
		return
	}

	filters, expand, authErr := h.auth.callerFilters(ctx, req, h.l)
	if authErr != nil {
		h.writeError(rw, statusForAuthError(authErr.code), authErr.message)
		return
	}

	var names []string
	if expand {
		names, err = h.expandScopes(ctx, scopes, q.Get("from"), q.Get("to"), filters)
		if errors.Is(err, errInvalidTimeRange) {
			h.writeError(rw, http.StatusBadRequest, err.Error())
			return
		}
		if err != nil {
			// Global annotations don't need VictoriaMetrics, so keep serving the plain tags.
			h.l.Warnf("Serving plain tags only, failed to expand scope tags: %s", err)
			names = nil
		}
	}

	items, err := h.findAnnotations(ctx, req, q, tags, names, limit)
	if err != nil {
		cErr, ok := errors.AsType[*clientError](err)
		if ok && (cErr.Code == http.StatusUnauthorized || cErr.Code == http.StatusForbidden) {
			h.writeError(rw, cErr.Code, cErr.ErrorMessage)
			return
		}
		h.l.Errorf("Failed to find annotations: %s", err)
		h.writeError(rw, http.StatusBadGateway, "Bad Gateway")
		return
	}

	h.l.WithFields(logrus.Fields{
		"scopes":   len(scopes),
		"lbac":     len(filters) != 0,
		"names":    len(names),
		"returned": len(items),
		"duration": time.Since(start),
	}).Debug("Served scoped annotations.")

	h.writeJSON(rw, http.StatusOK, items)
}

// parseScopeTags splits tags into plain tags and scopes.
func parseScopeTags(tags []string) ([]string, []scope, error) {
	var plain []string
	var scopes []scope
	for _, t := range tags {
		rest, ok := strings.CutPrefix(t, scopeTagPrefix)
		if !ok {
			plain = append(plain, t)
			continue
		}

		label, selector, ok := strings.Cut(rest, ":")
		if !ok || !slices.Contains(scopeLabels, label) || selector == "" || len(selector) > maxScopeSelectorLength {
			return nil, nil, errInvalidScopeTag
		}
		scopes = append(scopes, scope{label: label, selector: selector})
	}

	if len(scopes) > maxScopeTags {
		return nil, nil, errInvalidScopeTag
	}

	return plain, scopes, nil
}

// parseLimit returns the query's limit, at most windowReadLimit, or Grafana's default when it is missing or not positive.
func parseLimit(v string) (int, error) {
	if v == "" {
		return defaultAnnotationsLimit, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, errInvalidLimit
	}
	if n <= 0 {
		return defaultAnnotationsLimit, nil
	}

	return min(n, windowReadLimit), nil
}

// expandScopes returns the sorted, distinct label values the scopes select in the time range,
// limited by the caller's LBAC filters.
func (h *ScopedAnnotationsHandler) expandScopes(ctx context.Context, scopes []scope, from, to string, filters []string) ([]string, error) {
	params := url.Values{}
	for _, p := range []struct{ name, ms string }{{"start", from}, {"end", to}} {
		if p.ms == "" {
			continue
		}
		n, err := strconv.ParseInt(p.ms, 10, 64)
		if err != nil {
			return nil, errInvalidTimeRange
		}
		params.Set(p.name, strconv.FormatFloat(float64(n)/1000, 'f', 3, 64))
	}
	for _, f := range filters {
		// The same filter vmproxy adds to the caller's data source queries.
		params.Add("extra_filters[]", f)
	}

	seen := make(map[string]struct{})
	for _, s := range scopes {
		values, err := h.labelValues(ctx, s, params)
		if err != nil {
			return nil, err
		}
		for _, v := range values {
			seen[v] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	slices.Sort(names)

	return names, nil
}

func (h *ScopedAnnotationsHandler) labelValues(ctx context.Context, s scope, params url.Values) ([]string, error) {
	u := h.vm.URL("/api/v1/label/:name/values", map[string]string{"name": s.label})
	q := maps.Clone(params)
	// Grafana's regex format escapes with one backslash, which a PromQL string rejects.
	// Scope selectors have no backslashes of their own, so doubling all of them is exact.
	q.Set("match[]", strings.ReplaceAll(s.selector, `\`, `\\`))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create label values request: %w", err)
	}

	// The client reads and closes the body itself.
	resp, body, err := h.vm.Do(ctx, req) //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("failed to get label values: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("label values for %s: %s: %s", s.label, resp.Status, body)
	}

	var res struct {
		Data []string `json:"data"`
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode label values: %w", err)
	}

	return res.Data, nil
}

// findAnnotations asks Grafana, as the caller, for annotations carrying any of the plain tags or names.
func (h *ScopedAnnotationsHandler) findAnnotations(ctx context.Context, req *http.Request, q url.Values, plain, names []string, limit int) ([]json.RawMessage, error) {
	tags := slices.Concat(plain, names)
	if len(tags) == 0 {
		return []json.RawMessage{}, nil
	}

	authHeaders := make(http.Header)
	for _, k := range []string{"Authorization", "Cookie", "X-Grafana-Org-Id"} {
		if v := req.Header.Get(k); v != "" {
			authHeaders.Set(k, v)
		}
	}

	params := maps.Clone(q)
	params.Del("tags")

	if len(names) < maxNamesPerQuery {
		params["tags"] = tags
		params.Set("matchAny", "true")
		params.Set("limit", strconv.Itoa(limit))
		return h.c.queryAnnotations(ctx, authHeaders, params)
	}

	params.Del("matchAny")
	params.Set("limit", strconv.Itoa(windowReadLimit))
	// Alert state history is stored as annotations too; PMM annotations never are.
	params.Set("type", "annotation")

	matcher := newTagMatcher(tags)
	var kept []annotationItem
	err := h.c.eachAnnotation(ctx, authHeaders, params, func(raw json.RawMessage) error {
		item := annotationItem{raw: raw}
		err := json.Unmarshal(raw, &item)
		if err != nil {
			return fmt.Errorf("failed to decode annotation: %w", err)
		}
		if !matcher.matches(item.Tags) {
			return nil
		}

		kept = append(kept, item)
		// Trimming at twice the limit bounds memory however many annotations the window holds.
		if len(kept) == 2*limit {
			kept = newestFirst(kept, limit)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	kept = newestFirst(kept, limit)
	res := make([]json.RawMessage, 0, len(kept))
	for _, item := range kept {
		res = append(res, item.raw)
	}
	return res, nil
}

// annotationItem is an annotation from Grafana with the fields that decide whether and where it is served.
type annotationItem struct {
	raw     json.RawMessage
	ID      int64    `json:"id"`
	Time    int64    `json:"time"`
	TimeEnd int64    `json:"timeEnd"`
	Tags    []string `json:"tags"`
}

// newestFirst returns at most limit of the items, sorted the way Grafana's annotation query sorts them.
func newestFirst(items []annotationItem, limit int) []annotationItem {
	slices.SortFunc(items, func(a, b annotationItem) int {
		return cmp.Or(cmp.Compare(b.TimeEnd, a.TimeEnd), cmp.Compare(b.Time, a.Time), cmp.Compare(b.ID, a.ID))
	})
	return items[:min(len(items), limit)]
}

// tagMatcher matches tags the way Grafana's tag query does (pkg/services/annotations/annotationsimpl/xorm_store.go).
type tagMatcher struct {
	// A requested tag without a value matches every value of its key.
	anyValue map[string]struct{}
	exact    map[tagPair]struct{}
}

func newTagMatcher(tags []string) tagMatcher {
	m := tagMatcher{
		anyValue: make(map[string]struct{}),
		exact:    make(map[tagPair]struct{}),
	}
	for _, t := range tags {
		p := parseTag(t)
		if p.value == "" {
			m.anyValue[p.key] = struct{}{}
			continue
		}
		m.exact[p] = struct{}{}
	}
	// Grafana drops tags with an empty key.
	delete(m.anyValue, "")

	return m
}

// matches reports whether any stored tag matches; Grafana returns the tags as last written, but matches its parsed copy of them.
func (m tagMatcher) matches(stored []string) bool {
	return slices.ContainsFunc(stored, func(t string) bool {
		p := parseTag(t)
		_, key := m.anyValue[p.key]
		_, pair := m.exact[p]
		return key || pair
	})
}

// tagPair is a tag as Grafana stores it in its tag table.
type tagPair struct {
	key   string
	value string
}

// parseTag splits a tag the way Grafana's tag.ParseTagPairs does: "key" or "key:value", each
// part trimmed of spaces, anything after a second colon dropped.
func parseTag(t string) tagPair {
	key, rest, found := strings.Cut(t, ":")
	p := tagPair{key: strings.Trim(key, " ")}
	if found {
		value, _, _ := strings.Cut(rest, ":")
		p.value = strings.Trim(value, " ")
	}

	return p
}

func statusForAuthError(code codes.Code) int {
	switch code {
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func (h *ScopedAnnotationsHandler) writeError(rw http.ResponseWriter, status int, message string) {
	h.writeJSON(rw, status, map[string]string{"message": message})
}

func (h *ScopedAnnotationsHandler) writeJSON(rw http.ResponseWriter, status int, v any) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	err := json.NewEncoder(rw).Encode(v)
	if err != nil {
		h.l.Warnf("Failed to write response: %s", err)
	}
}
