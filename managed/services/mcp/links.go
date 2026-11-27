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
// settings.PMMPublicAddress when set, else the incoming X-Forwarded-Host and
// X-Forwarded-Proto (set by nginx for /mcp) with https unless that is http,
// else empty, in which case links are relative.
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
		scheme := "https"
		if strings.EqualFold(strings.TrimSpace(h.Get("X-Forwarded-Proto")), "http") {
			scheme = "http"
		}
		return scheme + "://" + host + "/"
	}
	return ""
}

// windowQuery is a dashboard's time range [from, to] and, when serviceName is set, its service_name variable.
func windowQuery(serviceName string, from, to time.Time) url.Values {
	q := url.Values{}
	q.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	q.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	if serviceName != "" {
		q.Set("var-service_name", serviceName)
	}
	return q
}

// qanOverviewURL links to the QAN ranked overview for a service and window.
func qanOverviewURL(base, serviceName string, from, to time.Time) string {
	return base + qanPath + "?" + windowQuery(serviceName, from, to).Encode()
}

// serviceURL links to the engine's instance summary dashboard for a service,
// or returns "" for a service type without one.
func serviceURL(base string, svc serviceInfo) string {
	q := url.Values{}
	switch svc.Engine {
	case engineMySQL, enginePostgreSQL, engineMongoDB, engineHAProxy:
		q.Set("var-service_name", svc.ServiceName)
	case engineProxySQL:
		if svc.NodeName == "" {
			return ""
		}
		q.Set("var-proxysql", svc.NodeName)
	default:
		return ""
	}
	return base + "graph/d/" + svc.Engine + "-instance-summary?" + q.Encode()
}

// serviceWindowURL is serviceURL with the dashboard's time range set to [from, to].
func serviceWindowURL(base string, svc serviceInfo, from, to time.Time) string {
	u := serviceURL(base, svc)
	if u == "" {
		return ""
	}
	return u + "&" + windowQuery("", from, to).Encode()
}

// qanQueryURL links to QAN with one query pre-selected; detailsTab may be
// "" (default), "explain" or "tables".
func qanQueryURL(base, queryID, serviceName string, from, to time.Time, detailsTab string) string {
	q := windowQuery(serviceName, from, to)
	q.Set("filter_by", queryID)
	q.Set("query_selected", "true")
	if detailsTab != "" {
		q.Set("details_tab", detailsTab)
	}
	return base + qanPath + "?" + q.Encode()
}

// alertRuleURL links to an alert rule's page in PMM, as PMM's alert list does, or to the alert list when the rule is unknown.
func alertRuleURL(base, ruleUID string) string {
	if ruleUID == "" {
		return base + "graph/alerting/list"
	}
	return base + "graph/alerting/grafana/" + url.PathEscape(ruleUID) + "/view"
}

// link renders a markdown link to PMM; u is relative when there is no base.
func link(label, u string) string {
	return "[" + label + " ↗](" + u + ")"
}
