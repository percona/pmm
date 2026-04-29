// Copyright (C) 2025 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package grafana

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"time"
)

const defaultRenderCacheDir = "/srv/pmm/grafana_render_cache"

// isoToEpochMs parses from/to as ISO 8601 and returns epoch milliseconds for Grafana dashboard URL. If either parse fails, returns false.
func isoToEpochMs(from, to string) (fromMs, toMs int64, ok bool) {
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
