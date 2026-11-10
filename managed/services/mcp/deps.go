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
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/percona/pmm/api/actions/v1/json/client/actions_service"
	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
	"github.com/percona/pmm/api/server/v1/json/client/server_service"
	"github.com/percona/pmm/managed/utils/auth"
)

// callerAuth carries the credentials of the MCP caller in one canonical form: a
// token as a Bearer Authorization without a cookie, anything else as received.
// Every backing PMM API call sends them again, so nginx authorizes each call
// against the caller's own role.
type callerAuth struct {
	authorization string
	cookie        string
}

// callerAuthFromHeader extracts the caller's credentials from the MCP request headers.
func callerAuthFromHeader(h http.Header) callerAuth {
	token := auth.GetTokenFromHeaders(h)
	if token != "" {
		return callerAuth{authorization: "Bearer " + token}
	}

	return callerAuth{
		authorization: h.Get("Authorization"),
		cookie:        h.Get("Cookie"),
	}
}

// datasource is a Grafana datasource as returned by GET /graph/api/datasources/name/{name}.
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

// lenientFloat decodes a JSON number or a numeric string: qan-api2 (grpc-gateway
// with protojson) encodes NaN and infinite float values as the strings "NaN",
// "Infinity" and "-Infinity" - live-observed in sparkline fields on 3.8.1 -
// which the generated go-swagger client rejects for float32 fields.
type lenientFloat float64

// UnmarshalJSON implements json.Unmarshaler.
func (f *lenientFloat) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*f = 0
		return nil
	}
	if b[0] == '"' {
		var s string
		err := json.Unmarshal(b, &s)
		if err != nil {
			return err
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("lenientFloat: %w", err)
		}
		*f = lenientFloat(v)
		return nil
	}
	var v float64
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	*f = lenientFloat(v)
	return nil
}

// finite returns the value, or 0 for NaN and infinities.
func finite(f lenientFloat) float64 {
	v := nullable(float64(f))
	if v == nil {
		return 0
	}
	return *v
}

// nullable returns a pointer to the value, or nil for NaN and infinities.
func nullable(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &v
}

// metricStats is the per-metric statistics block of QAN responses
// (Stat in qan.v1 profile.proto, MetricValues in qan.v1 object_details.proto).
type metricStats struct {
	Rate           lenientFloat `json:"rate"`
	Cnt            lenientFloat `json:"cnt"`
	Sum            lenientFloat `json:"sum"`
	Min            lenientFloat `json:"min"`
	Max            lenientFloat `json:"max"`
	P99            lenientFloat `json:"p99"`
	Avg            lenientFloat `json:"avg"`
	SumPerSec      lenientFloat `json:"sum_per_sec"`
	PercentOfTotal lenientFloat `json:"percent_of_total"`
}

// qanReportRow is one row of POST /v1/qan/metrics:getReport.
type qanReportRow struct {
	Rank        int64        `json:"rank"`
	Dimension   string       `json:"dimension"`
	Database    string       `json:"database"`
	Fingerprint string       `json:"fingerprint"`
	NumQueries  lenientFloat `json:"num_queries"`
	QPS         lenientFloat `json:"qps"`
	Load        lenientFloat `json:"load"`
	Metrics     map[string]struct {
		Stats *metricStats `json:"stats"`
	} `json:"metrics"`
}

// qanReport is the response of POST /v1/qan/metrics:getReport, without sparklines.
type qanReport struct {
	TotalRows int64          `json:"total_rows"`
	Rows      []qanReportRow `json:"rows"`
}

// queryMetadata is the metadata block of POST /v1/qan:getMetrics.
type queryMetadata struct {
	ServiceName    string `json:"service_name"`
	Database       string `json:"database"`
	Schema         string `json:"schema"`
	Username       string `json:"username"`
	ReplicationSet string `json:"replication_set"`
	Cluster        string `json:"cluster"`
	ServiceType    string `json:"service_type"`
	ServiceID      string `json:"service_id"`
	Environment    string `json:"environment"`
	NodeID         string `json:"node_id"`
	NodeName       string `json:"node_name"`
	NodeType       string `json:"node_type"`
}

// queryMetrics is the response of POST /v1/qan:getMetrics, without sparklines.
type queryMetrics struct {
	Metrics     map[string]metricStats `json:"metrics"`
	TextMetrics map[string]string      `json:"text_metrics"`
	Fingerprint string                 `json:"fingerprint"`
	Metadata    *queryMetadata         `json:"metadata"`
}

// grafanaAlert is one firing alert of a rule that Grafana's rules API returned to the caller.
type grafanaAlert struct {
	RuleUID string
	// PMMRule is true for a rule PMM provisions: file provenance and a UID of PMM's catalog.
	PMMRule     bool
	Labels      map[string]string
	Annotations map[string]string
	StartsAt    time.Time
	Silenced    bool
	// SilencedUntil is the latest end of the silences that mute the alert; nil when it is not silenced or no end is known.
	SilencedUntil *time.Time
}

// amAlertStatus is the status of an alert of Grafana's Alertmanager.
type amAlertStatus struct {
	SilencedBy []string `json:"silencedBy"`
}

// amAlert is an alert of Grafana's Alertmanager API (/api/v2/alerts), as far as the silence join reads it.
type amAlert struct {
	Labels map[string]string `json:"labels"`
	Status amAlertStatus     `json:"status"`
}

// amSilence is a silence of Grafana's Alertmanager API (/api/v2/silences), as far as the silence join reads it.
type amSilence struct {
	ID     string    `json:"id"`
	EndsAt time.Time `json:"endsAt"`
}

// pmmAPI is the subset of PMM's public REST API the tools depend on. Every
// method goes through the nginx loopback with the caller's credentials.
//
//nolint:interfacebloat // one method per backing PMM endpoint, by design
type pmmAPI interface {
	Version(ctx context.Context, auth callerAuth) (*server_service.VersionOKBody, error)
	GetReport(ctx context.Context, auth callerAuth, body qan_service.GetReportBody) (*qanReport, error)
	GetMetrics(ctx context.Context, auth callerAuth, body qan_service.GetMetricsBody) (*queryMetrics, error)
	GetQueryExample(ctx context.Context, auth callerAuth, body qan_service.GetQueryExampleBody) (*qan_service.GetQueryExampleOKBody, error)
	GetQueryPlan(ctx context.Context, auth callerAuth, queryID, serviceID string) (*qan_service.GetQueryPlanOKBody, error)
	QANServiceTypes(ctx context.Context, auth callerAuth, from, to time.Time) ([]string, error)
	StartServiceAction(ctx context.Context, auth callerAuth, body actions_service.StartServiceActionBody) (*actions_service.StartServiceActionOKBody, error)
	GetAction(ctx context.Context, auth callerAuth, actionID string) (*actions_service.GetActionOKBody, error)
	CancelAction(ctx context.Context, auth callerAuth, actionID string) error
	GetDatasourceByName(ctx context.Context, auth callerAuth, name string) (*datasource, error)
	QueryInstant(ctx context.Context, auth callerAuth, uid string, r promRequest) ([]metricSample, error)
	QueryRange(ctx context.Context, auth callerAuth, uid string, r promRequest) ([]metricSeries, error)
	LabelValues(ctx context.Context, auth callerAuth, uid, label string, match []string, start, end time.Time, limit int) ([]string, error)
	Labels(ctx context.Context, auth callerAuth, uid string, match []string, start, end time.Time, limit int) ([]string, error)
	Series(ctx context.Context, auth callerAuth, uid string, match []string, start, end time.Time, limit int) ([]map[string]string, error)
	Alerts(ctx context.Context, auth callerAuth) ([]grafanaAlert, error)
	SilencedAlerts(ctx context.Context, auth callerAuth) ([]amAlert, error)
	Silences(ctx context.Context, auth callerAuth) ([]amSilence, error)
}
