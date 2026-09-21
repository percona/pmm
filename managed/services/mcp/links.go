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
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// qanPath is PMM's Query Analytics dashboard. Deep links open it with an
// absolute time window (Grafana from/to in epoch milliseconds), optionally a
// service filter and a pre-selected query.
//
// Confirmed against the QAN frontend source (percona/grafana-dashboards,
// pmm-app/src/pmm-qan/panel/provider/provider.tools.ts): a query is selected
// by filter_by=<queryid> plus query_selected=true; details_tab picks the tab.
const qanPath = "graph/d/pmm-qan/pmm-query-analytics"

// publicBaseURL returns the browser-facing PMM base URL, with a trailing slash:
// settings.PMMPublicAddress when set, else the incoming X-Forwarded-Host (set
// by nginx for /mcp), else empty, in which case links are relative.
func (s *Service) publicBaseURL(ctx context.Context, h http.Header) string {
	if s.publicAddress != nil {
		if addr := strings.TrimSpace(s.publicAddress(ctx)); addr != "" {
			if !strings.Contains(addr, "://") {
				addr = "https://" + addr
			}
			return strings.TrimRight(addr, "/") + "/"
		}
	}
	if host := h.Get("X-Forwarded-Host"); host != "" {
		scheme := h.Get("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "https"
		}
		return scheme + "://" + host + "/"
	}
	return ""
}

// qanOverviewURL links to the QAN ranked overview for a service and window.
func qanOverviewURL(base, serviceName string, from, to time.Time) string {
	q := url.Values{}
	q.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	q.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	if serviceName != "" {
		q.Set("var-service_name", serviceName)
	}
	return base + qanPath + "?" + q.Encode()
}

// qanQueryURL links to QAN with one query pre-selected; detailsTab may be
// "" (default), "explain" or "tables".
func qanQueryURL(base, queryID, serviceName string, from, to time.Time, detailsTab string) string {
	q := url.Values{}
	q.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	q.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	if serviceName != "" {
		q.Set("var-service_name", serviceName)
	}
	q.Set("filter_by", queryID)
	q.Set("query_selected", "true")
	if detailsTab != "" {
		q.Set("details_tab", detailsTab)
	}
	return base + qanPath + "?" + q.Encode()
}

// link renders a markdown "View in PMM" line; relative when there is no base.
func link(label, u string) string {
	return "[" + label + " ↗](" + u + ")"
}
