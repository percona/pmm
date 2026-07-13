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

package otel

// LogSourceEntry defines a path and preset name for server-side filelog receivers.
type LogSourceEntry struct {
	Path   string
	Preset string
}

// DefaultServerOtelLogSources is the default list of (path, preset name) for the server's
// OTEL collector filelog receivers. Matches actual PMM server layout in /srv/logs/.
var DefaultServerOtelLogSources = []LogSourceEntry{
	{Path: "/srv/logs/nginx.log", Preset: "nginx_access"},
	{Path: "/srv/logs/grafana.log", Preset: "grafana"},
	{Path: "/srv/logs/pmm-managed.log", Preset: "pmm_managed"},
	{Path: "/srv/logs/pmm-agent.log", Preset: "pmm_agent"},
	{Path: "/srv/logs/postgresql14.log", Preset: "postgres"},
	{Path: "/srv/logs/clickhouse-server.log", Preset: "clickhouse_server"},
	{Path: "/srv/logs/otel-collector.log", Preset: "otel_collector"},
	{Path: "/srv/logs/supervisord.log", Preset: "supervisord"},
	{Path: "/srv/logs/qan-api2.log", Preset: "pmm_agent"},
	{Path: "/srv/logs/vmproxy.log", Preset: "pmm_agent"},
}
