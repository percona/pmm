// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package grafana

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RenderCanonicalParams holds inputs included in the content-addressable cache key for panel renders.
// All string fields should be in canonical form before calling ContentHashFromRenderParams.
type RenderCanonicalParams struct {
	DashboardUID string
	PanelID      string // numeric string, no "panel-" prefix
	From         string // RFC3339 UTC ms-normalized, or Grafana relative e.g. now-6h
	To           string
	OrgID        int
	Width        int
	Height       int
	Scale        int
	TZ           string
	Theme        string
	// Vars keys must be Grafana-style "var-<name>" with sorted iteration handled in hashing.
	Vars map[string]string
}

// normalizeFromToForCanonical returns stable from/to strings for hashing and Grafana queries.
// Absolute times are normalized to UTC RFC3339Nano when parseable; relative Grafana tokens are preserved.
func normalizeFromToForCanonical(from, to string) (fromOut, toOut string) { //nolint:nonamedreturns
	fromOut = normalizeTimeForCanonical(strings.TrimSpace(from))
	toOut = normalizeTimeForCanonical(strings.TrimSpace(to))
	return fromOut, toOut
}

func normalizeTimeForCanonical(s string) string {
	if looksLikeGrafanaRelative(s) {
		return s
	}
	if t, ok := parseRFC3339MillisUTC(s); ok {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return s
}

func looksLikeGrafanaRelative(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "now") {
		return true
	}
	// Epoch ms as digits only
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	const epochMsMinDigits = 10
	return len(s) >= epochMsMinDigits
}

func parseRFC3339MillisUTC(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, s)
	}
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// NormalizePanelID strips a leading "panel-" prefix if present.
func NormalizePanelID(panelID string) string {
	s := strings.TrimSpace(panelID)
	s = strings.TrimPrefix(s, "panel-")
	return s
}

// ContentHashFromRenderParams builds a deterministic SHA-256 hex key from canonical render inputs.
//
// Migration from renderCacheKey(url.Values built from the old GET /v1/grafana/render query): that path
// hashed whatever the client sent (including duplicate keys and ordering noise). Tier 1 hashes only
// resolved inputs — dashboard UID (lower), normalized panel id, UTC-normalized absolute times (see
// normalizeFromToForCanonical), explicit org/width/height/scale/tz/theme defaults, and lexicographically
// sorted var-* entries — using the same digest primitive as renderCacheKey (sorted keys, stable iteration).
func ContentHashFromRenderParams(p RenderCanonicalParams) string {
	v := url.Values{}
	v.Set("dashboard_uid", strings.ToLower(strings.TrimSpace(p.DashboardUID)))
	v.Set("panel_id", p.PanelID)
	fromN, toN := normalizeFromToForCanonical(p.From, p.To)
	v.Set("from", fromN)
	v.Set("to", toN)
	v.Set("org_id", strconv.Itoa(p.OrgID))
	v.Set("width", strconv.Itoa(p.Width))
	v.Set("height", strconv.Itoa(p.Height))
	v.Set("scale", strconv.Itoa(p.Scale))
	v.Set("tz", p.TZ)
	v.Set("theme", p.Theme)

	keys := make([]string, 0, len(p.Vars))
	for k := range p.Vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v.Set(k, p.Vars[k])
	}

	return renderCacheKey(v)
}
