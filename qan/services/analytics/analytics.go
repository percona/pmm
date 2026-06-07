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

// Package analytics implements the QANService read API over the rollup tables.
package analytics

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan/models"
	"github.com/percona/pmm/utils/ddsketch"
)

// Service implements the gRPC QANService. RPCs not yet implemented are inherited
// from UnimplementedQANServiceServer and return codes.Unimplemented.
type Service struct {
	conn     driver.Conn
	reporter *models.Reporter
	cache    *resultCache
	l        *logrus.Entry

	qanv1.UnimplementedQANServiceServer
}

// NewService returns a QANService backed by conn.
func NewService(conn driver.Conn) *Service {
	return &Service{
		conn:     conn,
		reporter: models.NewReporter(conn),
		cache:    newResultCache(resultCacheTTL),
		l:        logrus.WithField("component", "analytics"),
	}
}

// HealthCheck reports readiness by pinging ClickHouse.
func (s *Service) HealthCheck(ctx context.Context, _ *qanv1.HealthCheckRequest) (*qanv1.HealthCheckResponse, error) {
	err := s.conn.Ping(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "clickhouse not ready: %v", err)
	}
	return &qanv1.HealthCheckResponse{}, nil
}

// GetReport returns the profile: a grand-total row followed by the page of top dimensions.
func (s *Service) GetReport(ctx context.Context, in *qanv1.GetReportRequest) (*qanv1.GetReportResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, status.Error(codes.InvalidArgument, "period_start_from and period_start_to are required")
	}
	from, to := in.PeriodStartFrom.GetSeconds(), in.PeriodStartTo.GetSeconds()
	if from > to {
		return nil, status.Error(codes.InvalidArgument, "period_start_from cannot be after period_start_to")
	}
	if _, ok := models.GroupByColumn[in.GroupBy]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported group_by: %q", in.GroupBy)
	}

	cacheK := cacheKey("report", in)
	if cacheK != "" {
		cached, ok := s.cache.get(cacheK)
		if ok {
			return cached.(*qanv1.GetReportResponse), nil //nolint:forcetypeassert
		}
	}

	durSec := float32(to - from)
	if durSec <= 0 {
		durSec = 1
	}
	limit := in.Limit
	if limit == 0 {
		limit = 10
	}
	columns := reportColumns(in.Columns)

	dimensions := make(map[string][]string)
	for _, l := range in.GetLabels() {
		dimensions[l.Key] = l.Value
	}

	params := models.ReportParams{
		FromSec: from, ToSec: to, GroupBy: in.GroupBy,
		Dimensions: dimensions, OrderBy: in.OrderBy, Offset: in.Offset, Limit: limit,
		Search: in.Search,
	}
	res, err := s.reporter.Report(ctx, params)
	if err != nil {
		s.l.Errorf("GetReport: %v", err)
		return nil, status.Errorf(codes.Internal, "report failed: %v", err)
	}

	resp := &qanv1.GetReportResponse{
		TotalRows: uint32(res.Total.TotalRows),
		Offset:    in.Offset,
		Limit:     limit,
	}
	// Row 0 is the grand total (fingerprint "TOTAL").
	totalRow := buildRow(in.Offset, "TOTAL", &res.Total, &res.Total, columns, durSec)
	totalRow.Sparkline, err = s.reporter.Sparklines(ctx, params, "", "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sparkline failed: %v", err)
	}
	resp.Rows = append(resp.Rows, totalRow)
	for i := range res.Rows {
		row := &res.Rows[i]
		r := buildRow(in.Offset+uint32(i)+1, res.Fingerprints[row.Dimension], row, &res.Total, columns, durSec)
		r.Sparkline, err = s.reporter.Sparklines(ctx, params, in.GroupBy, row.Dimension)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "sparkline failed: %v", err)
		}
		resp.Rows = append(resp.Rows, r)
	}

	if cacheK != "" {
		s.cache.set(cacheK, resp)
	}
	return resp, nil
}

func buildRow(rank uint32, fingerprint string, row, total *models.ReportRow, columns []string, durSec float32) *qanv1.Row {
	r := &qanv1.Row{
		Rank:        rank,
		Dimension:   row.Dimension,
		Database:    row.Database,
		Fingerprint: fingerprint,
		NumQueries:  uint32(row.NumQueries),
		Qps:         float32(row.NumQueries) / durSec,
		Load:        float32(row.QueryTimeSum) / durSec,
		Metrics:     make(map[string]*qanv1.Metric, len(columns)),
	}
	for _, c := range columns {
		r.Metrics[c] = &qanv1.Metric{Stats: makeStat(c, row, total, durSec)}
	}
	return r
}

func makeStat(col string, row, total *models.ReportRow, durSec float32) *qanv1.Stat {
	st := &qanv1.Stat{}
	switch col {
	case "load":
		st.SumPerSec = float32(row.QueryTimeSum) / durSec
	case "num_queries":
		st.Sum = float32(row.NumQueries)
		st.SumPerSec = float32(row.NumQueries) / durSec
	case "num_queries_with_errors":
		st.Sum = float32(row.NumQueriesWithErrors)
		st.SumPerSec = float32(row.NumQueriesWithErrors) / durSec
	case "num_queries_with_warnings":
		st.Sum = float32(row.NumQueriesWithWarnings)
		st.SumPerSec = float32(row.NumQueriesWithWarnings) / durSec
	default:
		metric, ok := models.ReportMetrics[col]
		if !ok {
			return st
		}
		sum, cnt, mn, mx, sketch := row.Metric(col)
		totalSum, _, _, _, _ := total.Metric(col)
		st.Sum = float32(sum)
		st.Cnt = float32(cnt)
		st.Min = mn
		st.Max = mx
		st.SumPerSec = float32(sum) / durSec
		if cnt > 0 {
			st.Avg = float32(sum / float64(cnt))
		}
		if totalSum > 0 {
			st.Rate = float32(sum / totalSum)
		}
		if metric.IsTime && len(sketch) > 0 {
			st.P99 = float32(ddsketch.QuantileFromMap(sketch, 0.99))
		}
	}
	return st
}

// reportColumns resolves the requested columns (with the legacy count->num_queries
// alias), defaulting to the standard dashboard set, deduplicated and order-preserving.
func reportColumns(requested []string) []string {
	if len(requested) == 0 {
		requested = []string{"load", "num_queries", "query_time"}
	}
	seen := make(map[string]struct{}, len(requested))
	cols := make([]string, 0, len(requested))
	for _, c := range requested {
		if c == "count" {
			c = "num_queries"
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		cols = append(cols, c)
	}
	return cols
}

// GetFilteredMetricsNames returns each dimension's values from the precomputed
// dim_values table (no fact-table scan).
func (s *Service) GetFilteredMetricsNames(ctx context.Context, in *qanv1.GetFilteredMetricsNamesRequest) (*qanv1.GetFilteredMetricsNamesResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, status.Error(codes.InvalidArgument, "period_start_from and period_start_to are required")
	}

	cacheK := cacheKey("filters", in)
	if cacheK != "" {
		cached, ok := s.cache.get(cacheK)
		if ok {
			return cached.(*qanv1.GetFilteredMetricsNamesResponse), nil //nolint:forcetypeassert
		}
	}

	filters, err := s.reporter.Filters(ctx, in.PeriodStartFrom.GetSeconds(), in.PeriodStartTo.GetSeconds())
	if err != nil {
		s.l.Errorf("GetFilteredMetricsNames: %v", err)
		return nil, status.Errorf(codes.Internal, "filters failed: %v", err)
	}

	resp := &qanv1.GetFilteredMetricsNamesResponse{Labels: make(map[string]*qanv1.ListLabels, len(filters))}
	for dim, vals := range filters {
		list := &qanv1.ListLabels{Name: make([]*qanv1.Values, 0, len(vals))}
		for _, v := range vals {
			list.Name = append(list.Name, &qanv1.Values{
				Value:             v.Value,
				MainMetricPercent: v.Percent,
				MainMetricPerSec:  v.PerSec,
			})
		}
		resp.Labels[dim] = list
	}
	if cacheK != "" {
		s.cache.set(cacheK, resp)
	}
	return resp, nil
}

// GetQueryExample returns stored examples for a queryid.
func (s *Service) GetQueryExample(ctx context.Context, in *qanv1.GetQueryExampleRequest) (*qanv1.GetQueryExampleResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, status.Error(codes.InvalidArgument, "period_start_from and period_start_to are required")
	}
	if in.GroupBy != "queryid" {
		return nil, status.Errorf(codes.InvalidArgument, "group_by %q not supported for examples; use queryid", in.GroupBy)
	}
	examples, err := s.reporter.QueryExamples(ctx, in.FilterBy, in.PeriodStartFrom.GetSeconds(), in.PeriodStartTo.GetSeconds(), in.Limit)
	if err != nil {
		s.l.Errorf("GetQueryExample: %v", err)
		return nil, status.Errorf(codes.Internal, "examples failed: %v", err)
	}

	resp := &qanv1.GetQueryExampleResponse{}
	for _, e := range examples {
		resp.QueryExamples = append(resp.QueryExamples, &qanv1.QueryExample{
			Example:     e.Example,
			ExampleType: qanv1.ExampleType(qanv1.ExampleType_value[e.ExampleType]),
			IsTruncated: uint32(e.IsTruncated),
			QueryId:     in.FilterBy,
		})
	}
	return resp, nil
}
