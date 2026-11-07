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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/api/qan/v1/json/client/qan_service"
)

const (
	reportPath  = "/v1/qan/metrics:getReport"
	reportOne   = "report_by_service_one.json"
	reportTwo   = "report_by_service_two.json"
	exampleDone = "example.json"
)

func resolveRoutes(report string) map[string]fixture {
	routes := qanRoutes()
	routes["GET /v1/inventory/services"] = fixture{file: "services_with_mongo.json"}
	routes["POST "+reportPath] = fixture{file: report}
	return routes
}

func TestResolveQuery(t *testing.T) {
	t.Parallel()

	dayOld := queryRef{
		QueryID:    "QID-AAA",
		PeriodFrom: testNow.Add(-26 * time.Hour),
		PeriodTo:   testNow.Add(-25 * time.Hour),
	}
	hour := queryRef{QueryID: "QID-AAA", PeriodFrom: testNow.Add(-time.Hour), PeriodTo: testNow}

	withService := func(ref queryRef, id string) queryRef {
		ref.ServiceID = id
		return ref
	}
	with := func(routes map[string]fixture, kv ...string) map[string]fixture {
		for i := 0; i < len(kv); i += 2 {
			routes[kv[i]] = fixture{file: kv[i+1]}
		}
		return routes
	}

	tests := []struct {
		name       string
		routes     map[string]fixture
		ref        queryRef
		wantErr    errorCode
		wantText   []string
		notText    []string
		wantReport int
		check      func(t *testing.T, got *resolvedQuery)
	}{
		{
			name:       "one candidate",
			routes:     resolveRoutes(reportOne),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, "svc-1", got.Service.ServiceID)
				assert.Equal(t, "svc-1", got.Ref.ServiceID)
				assert.Equal(t, hour.PeriodFrom, got.Ref.PeriodFrom)
				assert.Equal(t, "shop", got.Database)
				assert.Empty(t, got.Tables)
				assert.Contains(t, got.Example, "customers")
				assert.NoError(t, got.DetailErr)
			},
		},
		{
			name:       "failed example call is kept",
			routes:     withRoutes(resolveRoutes(reportOne), map[string]fixture{"POST /v1/qan/query:getExample": failed}),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, "svc-1", got.Service.ServiceID)
				assert.Empty(t, got.Example)
				assert.Equal(t, "shop", got.Database, "the metrics call still answered")
				require.Error(t, got.DetailErr)
				assert.Equal(t, codePMMUnavailable, mapError(got.DetailErr).code)
			},
		},
		{
			name:       "two candidates",
			routes:     resolveRoutes(reportTwo),
			ref:        hour,
			wantErr:    codeInvalidInput,
			wantText:   []string{"shop-mysql (svc-1)", "shop-postgres (svc-2)", "service_id"},
			notText:    []string{"more"},
			wantReport: 1,
		},
		{
			name:       "many candidates",
			routes:     with(resolveRoutes("report_by_service_many.json"), "GET /v1/inventory/services", "services_many.json"),
			ref:        hour,
			wantErr:    codeInvalidInput,
			wantText:   []string{"bulk-mysql-00 (svc-bulk-00)", "bulk-mysql-19 (svc-bulk-19)", "and 110 more"},
			notText:    []string{"bulk-mysql-20"},
			wantReport: 1,
		},
		{
			name:       "foreign row dropped",
			routes:     resolveRoutes("report_by_service_foreign.json"),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, "svc-1", got.Service.ServiceID)
			},
		},
		{
			name:     "service_id outside the list",
			routes:   resolveRoutes(reportOne),
			ref:      withService(hour, "svc-elsewhere"),
			wantErr:  codeNotFound,
			wantText: []string{notInServiceList},
			notText:  []string{"svc-elsewhere", "QID-AAA"},
		},
		{
			name:       "service_id in the list but not a candidate",
			routes:     resolveRoutes(reportOne),
			ref:        withService(hour, "svc-2"),
			wantErr:    codeNotFound,
			wantText:   []string{"the query was not seen on that service in", hour.PeriodFrom.Format(time.RFC3339)},
			notText:    []string{"svc-2", "QID-AAA"},
			wantReport: 1,
		},
		{
			name:       "no candidates",
			routes:     resolveRoutes("report_by_service_empty.json"),
			ref:        hour,
			wantErr:    codeNotFound,
			wantText:   []string{hour.PeriodFrom.Format(time.RFC3339), hour.PeriodTo.Format(time.RFC3339), "query_ref"},
			notText:    []string{"QID-AAA"},
			wantReport: 1,
		},
		{
			name:       "example of another service",
			routes:     with(resolveRoutes(reportOne), "POST /v1/qan/query:getExample", "example_other_service.json", "POST /v1/qan:getMetrics", "metrics_empty.json"),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Empty(t, got.Example)
				assert.Empty(t, got.Tables)
				assert.Empty(t, got.Database)
			},
		},
		{
			name:       "mysql database falls back to the example schema",
			routes:     with(resolveRoutes(reportOne), "POST /v1/qan:getMetrics", "metrics_empty.json"),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, "shop", got.Database)
			},
		},
		{
			name:       "postgresql database and tables",
			routes:     with(resolveRoutes("report_by_service_postgres.json"), "POST /v1/qan:getMetrics", "metrics_pg.json", "POST /v1/qan/query:getExample", "example_pg.json"),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, "svc-2", got.Service.ServiceID)
				assert.Equal(t, "shop_pg", got.Database)
				assert.Equal(t, []string{"orders", "customers"}, got.Tables)
			},
		},
		{
			name:       "mongodb database",
			routes:     with(resolveRoutes("report_by_service_mongo.json"), "POST /v1/qan:getMetrics", "metrics_mongo.json", "POST /v1/qan/query:getExample", "example_empty.json"),
			ref:        hour,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, "svc-mongo", got.Service.ServiceID)
				assert.Equal(t, "shop_mongo", got.Database)
				assert.Empty(t, got.Tables)
			},
		},
		{
			name:       "window of an old query_ref",
			routes:     resolveRoutes(reportOne),
			ref:        dayOld,
			wantReport: 1,
			check: func(t *testing.T, got *resolvedQuery) {
				t.Helper()
				assert.Equal(t, dayOld.PeriodFrom, got.Ref.PeriodFrom)
				assert.Equal(t, dayOld.PeriodTo, got.Ref.PeriodTo)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := newFakePMM(t, tt.routes)
			s := newQANService(t, fake, true)

			got, err := s.resolveQuery(t.Context(), callerAuthFromHeader(http.Header{"Authorization": {testToken}}), tt.ref)

			reports := fake.requestsTo(reportPath)
			assert.Len(t, reports, tt.wantReport)
			for _, r := range reports {
				body := unmarshalBody[qan_service.GetReportBody](t, r)
				assert.Equal(t, "service_name", body.GroupBy)
				assert.EqualValues(t, 100, body.Limit)
				assert.Equal(t, tt.ref.PeriodFrom, time.Time(body.PeriodStartFrom).UTC())
				assert.Equal(t, tt.ref.PeriodTo, time.Time(body.PeriodStartTo).UTC())
				labels := map[string][]string{}
				for _, l := range body.Labels {
					labels[l.Key] = l.Value
				}
				assert.Equal(t, []string{"QID-AAA"}, labels["queryid"])
				if tt.ref.ServiceID != "" {
					assert.Equal(t, []string{tt.ref.ServiceID}, labels["service_id"])
				}
				assert.NotContains(t, string(r.body), "other-tenant-service")
			}
			if tt.name == "one candidate" {
				body := unmarshalBody[qan_service.GetReportBody](t, reports[0])
				var ids []string
				for _, l := range body.Labels {
					if l.Key == "service_id" {
						ids = l.Value
					}
				}
				assert.Equal(t, []string{"svc-1", "svc-mongo", "svc-2", "svc-pmm-pg"}, sortedByEngine(ids))
			}

			if tt.wantErr != "" {
				require.Error(t, err)
				te := mapError(err)
				assert.Equal(t, tt.wantErr, te.code)
				for _, text := range tt.wantText {
					assert.Contains(t, te.message, text)
				}
				for _, text := range tt.notText {
					assert.NotContains(t, te.message, text)
				}
				if tt.name == "service_id outside the list" {
					assert.Equal(t, notInServiceList, te.message)
				}
				return
			}
			require.NoError(t, err)
			tt.check(t, got)
		})
	}
}

// sortedByEngine orders the ids as the service list does: by engine, then name.
func sortedByEngine(ids []string) []string {
	order := []string{"svc-1", "svc-mongo", "svc-2", "svc-pmm-pg"}
	var out []string
	for _, id := range order {
		for _, got := range ids {
			if got == id {
				out = append(out, id)
			}
		}
	}
	return out
}

// refProbeResult is the result of the probe tool that runs refFromInput.
type refProbeResult struct {
	Ref queryRef `json:"ref"`
}

func (r refProbeResult) render() string {
	return "ref: " + r.Ref.QueryID
}

type refProbeInput struct {
	QueryRef   *queryRefInput `json:"query_ref,omitempty"`
	QueryID    string         `json:"queryid,omitempty"`
	ServiceID  string         `json:"service_id,omitempty"`
	PeriodFrom string         `json:"period_from,omitempty"`
	PeriodTo   string         `json:"period_to,omitempty"`
}

func TestRefFromInput(t *testing.T) {
	t.Parallel()

	fake := newFakePMM(t, qanRoutes())
	s := newQANService(t, fake, true)

	server := mcp.NewServer(&mcp.Implementation{Name: "probe", Version: "0"}, nil)
	addTool(s, server, &mcp.Tool{Name: "probe"}, func(_ context.Context, _ *mcp.CallToolRequest, in refProbeInput) (refProbeResult, error) {
		ref, err := refFromInput(in.QueryRef, in.QueryID, in.ServiceID, in.PeriodFrom, in.PeriodTo, testNow)
		return refProbeResult{Ref: ref}, err
	})
	srv := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	t.Cleanup(srv.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: srv.URL}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	from := testNow.Add(-2 * time.Hour).Format(time.RFC3339)
	to := testNow.Add(-time.Hour).Format(time.RFC3339)
	tests := []struct {
		name string
		args map[string]any
		want *queryRef
	}{
		{
			name: "query_ref alone",
			args: map[string]any{"query_ref": map[string]any{"queryid": "QID-AAA", "service_id": "svc-1", "period_from": from, "period_to": to}},
			want: &queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: testNow.Add(-2 * time.Hour), PeriodTo: testNow.Add(-time.Hour)},
		},
		{
			name: "queryid with a window",
			args: map[string]any{"queryid": "QID-AAA", "service_id": "svc-1", "period_from": from, "period_to": to},
			want: &queryRef{QueryID: "QID-AAA", ServiceID: "svc-1", PeriodFrom: testNow.Add(-2 * time.Hour), PeriodTo: testNow.Add(-time.Hour)},
		},
		{
			name: "queryid without a window",
			args: map[string]any{"queryid": "QID-AAA"},
			want: &queryRef{QueryID: "QID-AAA", PeriodFrom: testNow.Add(-time.Hour), PeriodTo: testNow},
		},
		{
			name: "both forms",
			args: map[string]any{"queryid": "QID-AAA", "query_ref": map[string]any{"queryid": "QID-AAA", "period_from": from, "period_to": to}},
		},
		{
			name: "query_ref beside service_id",
			args: map[string]any{"service_id": "svc-1", "query_ref": map[string]any{"queryid": "QID-AAA", "period_from": from, "period_to": to}},
		},
		{
			name: "query_ref beside period_from",
			args: map[string]any{"period_from": from, "query_ref": map[string]any{"queryid": "QID-AAA", "period_from": from, "period_to": to}},
		},
		{name: "neither", args: map[string]any{}},
		{name: "query_ref without queryid", args: map[string]any{"query_ref": map[string]any{"period_from": from, "period_to": to}}},
		{name: "query_ref without a window", args: map[string]any{"query_ref": map[string]any{"queryid": "QID-AAA"}}},
		{name: "service_id alone", args: map[string]any{"service_id": "svc-1"}},
		{name: "period_from alone", args: map[string]any{"period_from": from}},
		{
			name: "query_ref with a bad time",
			args: map[string]any{"query_ref": map[string]any{"queryid": "QID-AAA", "period_from": "yesterday", "period_to": to}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := callTool(t, session, "probe", tt.args)
			if tt.want != nil {
				got := structured[refProbeResult](t, res)
				assert.Equal(t, *tt.want, got.Ref)
				return
			}
			require.True(t, res.IsError)
			assert.True(t, strings.HasPrefix(textOf(t, res), "error: invalid_input\n"), textOf(t, res))
			assert.NotEmpty(t, res.Meta)
		})
	}
}
