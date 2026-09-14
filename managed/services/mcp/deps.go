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
	"time"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	"github.com/percona/pmm/api/inventory/v1/json/client/nodes_service"
	"github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
)

// callerAuth carries the credentials of the MCP caller, copied verbatim from the
// incoming HTTP request. Every backing PMM API call sends them again, so nginx
// authorizes each call against the caller's own role.
type callerAuth struct {
	authorization string
	cookie        string
}

// callerAuthFromHeader extracts the caller's credentials from the MCP request headers.
func callerAuthFromHeader(h http.Header) callerAuth {
	return callerAuth{
		authorization: h.Get("Authorization"),
		cookie:        h.Get("Cookie"),
	}
}

// datasource is a Grafana datasource as returned by GET /graph/api/datasources.
type datasource struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// metricSample is one element of a Prometheus instant-vector result.
type metricSample struct {
	Labels map[string]string
	Value  float64
}

// pmmAPI is the subset of PMM's public REST API the tools depend on. Every
// method goes through the nginx loopback with the caller's credentials.
//
//nolint:interfacebloat // one method per backing PMM endpoint, by design
type pmmAPI interface {
	Version(ctx context.Context, auth callerAuth) (*server_service.VersionOKBody, error)
	ListServices(ctx context.Context, auth callerAuth) (*services_service.ListServicesOKBody, error)
	ListNodes(ctx context.Context, auth callerAuth) (*nodes_service.ListNodesOKBody, error)
	GetReport(ctx context.Context, auth callerAuth, body qan_service.GetReportBody) (*qan_service.GetReportOKBody, error)
	GetMetrics(ctx context.Context, auth callerAuth, body qan_service.GetMetricsBody) (*qan_service.GetMetricsOKBody, error)
	GetQueryExample(ctx context.Context, auth callerAuth, body qan_service.GetQueryExampleBody) (*qan_service.GetQueryExampleOKBody, error)
	GetQueryPlan(ctx context.Context, auth callerAuth, queryID string) (*qan_service.GetQueryPlanOKBody, error)
	StartServiceAction(ctx context.Context, auth callerAuth, body actions_service.StartServiceActionBody) (*actions_service.StartServiceActionOKBody, error)
	GetAction(ctx context.Context, auth callerAuth, actionID string) (*actions_service.GetActionOKBody, error)
	ListDatasources(ctx context.Context, auth callerAuth) ([]datasource, error)
	QueryInstant(ctx context.Context, auth callerAuth, datasourceUID, promql string, at time.Time) ([]metricSample, error)
}
