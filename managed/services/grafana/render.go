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
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultRenderCacheDir = "/srv/pmm/grafana_render_cache"

// isoToEpochMs parses from/to as ISO 8601 and returns epoch milliseconds for Grafana dashboard URL. If either parse fails, returns false.
func isoToEpochMs(from, to string) (fromMs, toMs int64, ok bool) { //nolint:nonamedreturns
	parse := func(s string) (int64, bool) {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, s)
		}
		if err != nil {
			return 0, false
		}
		return t.UnixMilli(), true
	}
	fms, okFrom := parse(from)
	tms, okTo := parse(to)
	if !okFrom || !okTo {
		return 0, 0, false
	}
	return fms, tms, true
}

// fromToForGrafanaImageRendererQuery returns from/to for /graph/render/d-solo/ (and downstream image-renderer) URLs.
//
// When Grafana forwards the panel URL to grafana-image-renderer as a nested query parameter, absolute RFC3339
// values get URL-encoded; a second encoding pass turns ":" into "%253A". Chromium then loads timestamps that
// Grafana cannot parse, the panel never reports ready, and the renderer hits 408 at the default timeout.
// Epoch millisecond strings avoid ":" and survive that chain; Grafana accepts digit-only from/to.
func fromToForGrafanaImageRendererQuery(fromNorm, toNorm string) (string, string) {
	fromNorm = strings.TrimSpace(fromNorm)
	toNorm = strings.TrimSpace(toNorm)
	if looksLikeGrafanaRelative(fromNorm) || looksLikeGrafanaRelative(toNorm) {
		return fromNorm, toNorm
	}
	if fromMs, toMs, ok := isoToEpochMs(fromNorm, toNorm); ok {
		return strconv.FormatInt(fromMs, 10), strconv.FormatInt(toMs, 10)
	}
	return fromNorm, toNorm
}

// renderCacheKey returns a stable hash key from url.Values (sorted keys); same logical params => same key.
func renderCacheKey(params url.Values) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "format" || k == "cache" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		for _, v := range params[k] {
			_, _ = h.Write([]byte(k))
			_, _ = h.Write([]byte("\x00"))
			_, _ = h.Write([]byte(v))
			_, _ = h.Write([]byte("\x00"))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
