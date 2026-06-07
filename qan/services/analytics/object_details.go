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

package analytics

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	qanv1 "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan/models"
	"github.com/percona/pmm/utils/ddsketch"
)

// metricsNames maps metric roots to human-readable labels for the UI catalog.
var metricsNames = map[string]string{
	"load":          "Load",
	"num_queries":   "Number of Queries",
	"query_time":    "Query Time",
	"lock_time":     "Lock Time",
	"rows_sent":     "Rows Sent",
	"rows_examined": "Rows Examined",
	"rows_affected": "Rows Affected",
	"bytes_sent":    "Bytes Sent",
}

// detailColumns are the metrics returned by GetMetrics.
var detailColumns = []string{"query_time", "lock_time", "rows_sent", "rows_examined", "rows_affected", "bytes_sent"}

// GetMetricsNames returns the catalog of metric names.
func (s *Service) GetMetricsNames(_ context.Context, _ *qanv1.GetMetricsNamesRequest) (*qanv1.GetMetricsNamesResponse, error) { //nolint:unparam
	return &qanv1.GetMetricsNamesResponse{Data: metricsNames}, nil
}

// GetMetrics returns per-metric statistics for one dimension value plus the grand totals.
func (s *Service) GetMetrics(ctx context.Context, in *qanv1.GetMetricsRequest) (*qanv1.GetMetricsResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, status.Error(codes.InvalidArgument, "period_start_from and period_start_to are required")
	}
	if _, ok := models.GroupByColumn[in.GroupBy]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported group_by: %q", in.GroupBy)
	}
	from, to := in.PeriodStartFrom.GetSeconds(), in.PeriodStartTo.GetSeconds()

	dimensions := make(map[string][]string)
	for _, l := range in.GetLabels() {
		dimensions[l.Key] = l.Value
	}

	value, total, err := s.reporter.Metrics(ctx, models.ReportParams{
		FromSec: from, ToSec: to, GroupBy: in.GroupBy, Dimensions: dimensions,
	}, in.FilterBy)
	if err != nil {
		s.l.Errorf("GetMetrics: %v", err)
		return nil, status.Errorf(codes.Internal, "metrics failed: %v", err)
	}

	resp := &qanv1.GetMetricsResponse{
		Metrics: make(map[string]*qanv1.MetricValues),
		Totals:  make(map[string]*qanv1.MetricValues),
	}
	for _, c := range detailColumns {
		resp.Metrics[c] = metricValues(c, &value, &total)
		resp.Totals[c] = metricValues(c, &total, &total)
	}
	resp.Metrics["num_queries"] = numQueriesValues(value.NumQueries, total.NumQueries)
	resp.Totals["num_queries"] = numQueriesValues(total.NumQueries, total.NumQueries)

	if in.GroupBy == "queryid" {
		fp, err := s.reporter.Fingerprint(ctx, in.FilterBy)
		if err == nil {
			resp.Fingerprint = fp
		}
	}
	meta, err := s.reporter.Metadata(ctx, in.GroupBy, in.FilterBy, from, to)
	if err == nil {
		resp.Metadata = &qanv1.GetSelectedQueryMetadataResponse{
			ServiceName:    meta.ServiceName,
			ServiceId:      meta.ServiceID,
			ServiceType:    meta.ServiceType,
			Database:       meta.Database,
			Schema:         meta.Schema,
			Cluster:        meta.Cluster,
			Environment:    meta.Environment,
			ReplicationSet: meta.ReplicationSet,
			NodeName:       meta.NodeName,
		}
	}
	spark, err := s.reporter.Sparklines(ctx, models.ReportParams{FromSec: from, ToSec: to, GroupBy: in.GroupBy, Dimensions: dimensions}, in.GroupBy, in.FilterBy)
	if err == nil {
		resp.Sparkline = spark
	}
	return resp, nil
}

func metricValues(col string, row, total *models.ReportRow) *qanv1.MetricValues {
	sum, cnt, mn, mx, sketch := row.Metric(col)
	totalSum, _, _, _, _ := total.Metric(col) //nolint:dogsled
	mv := &qanv1.MetricValues{Sum: float32(sum), Cnt: float32(cnt), Min: mn, Max: mx}
	if cnt > 0 {
		mv.Avg = float32(sum / float64(cnt))
	}
	if totalSum > 0 {
		mv.Rate = float32(sum / totalSum)
		mv.PercentOfTotal = float32(sum / totalSum)
	}
	if models.ReportMetrics[col].IsTime && len(sketch) > 0 {
		mv.P99 = float32(ddsketch.QuantileFromMap(sketch, 0.99))
	}
	return mv
}

func numQueriesValues(sum, total float64) *qanv1.MetricValues {
	mv := &qanv1.MetricValues{Sum: float32(sum)}
	if total > 0 {
		mv.Rate = float32(sum / total)
		mv.PercentOfTotal = float32(sum / total)
	}
	return mv
}

// GetLabels returns the distinct dimension values a query appears with.
func (s *Service) GetLabels(ctx context.Context, in *qanv1.GetLabelsRequest) (*qanv1.GetLabelsResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, status.Error(codes.InvalidArgument, "period_start_from and period_start_to are required")
	}
	if _, ok := models.GroupByColumn[in.GroupBy]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported group_by: %q", in.GroupBy)
	}
	labels, err := s.reporter.LabelsForQuery(ctx, in.GroupBy, in.FilterBy, in.PeriodStartFrom.GetSeconds(), in.PeriodStartTo.GetSeconds())
	if err != nil {
		s.l.Errorf("GetLabels: %v", err)
		return nil, status.Errorf(codes.Internal, "labels failed: %v", err)
	}
	resp := &qanv1.GetLabelsResponse{Labels: make(map[string]*qanv1.ListLabelValues, len(labels))}
	for k, vals := range labels {
		resp.Labels[k] = &qanv1.ListLabelValues{Values: vals}
	}
	return resp, nil
}

// GetHistogram returns the query_time latency histogram, sourced directly from the DDSketch buckets.
func (s *Service) GetHistogram(ctx context.Context, in *qanv1.GetHistogramRequest) (*qanv1.GetHistogramResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, status.Error(codes.InvalidArgument, "period_start_from and period_start_to are required")
	}
	sketch, err := s.reporter.Histogram(ctx, in.Queryid, in.PeriodStartFrom.GetSeconds(), in.PeriodStartTo.GetSeconds())
	if err != nil {
		s.l.Errorf("GetHistogram: %v", err)
		return nil, status.Errorf(codes.Internal, "histogram failed: %v", err)
	}
	idxs := make([]int, 0, len(sketch))
	for idx := range sketch {
		idxs = append(idxs, int(idx))
	}
	sort.Ints(idxs)

	resp := &qanv1.GetHistogramResponse{}
	for _, idx := range idxs {
		lo, hi := ddsketch.BucketBounds(idx)
		resp.HistogramItems = append(resp.HistogramItems, &qanv1.HistogramItem{
			Range:     fmt.Sprintf("%.4gs - %.4gs", lo, hi),
			Frequency: uint32(sketch[uint16(idx)]), //nolint:gosec
		})
	}
	return resp, nil
}

// ExplainFingerprintByQueryID returns the stored explain fingerprint for a queryid.
func (s *Service) ExplainFingerprintByQueryID(ctx context.Context, in *qanv1.ExplainFingerprintByQueryIDRequest) (*qanv1.ExplainFingerprintByQueryIDResponse, error) {
	fp, placeholders, err := s.reporter.ExplainFingerprint(ctx, in.QueryId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "explain fingerprint failed: %v", err)
	}
	return &qanv1.ExplainFingerprintByQueryIDResponse{ExplainFingerprint: fp, PlaceholdersCount: placeholders}, nil
}

// GetQueryPlan returns the most recent stored query plan for a queryid.
func (s *Service) GetQueryPlan(ctx context.Context, in *qanv1.GetQueryPlanRequest) (*qanv1.GetQueryPlanResponse, error) {
	planid, plan, err := s.reporter.QueryPlan(ctx, in.Queryid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query plan failed: %v", err)
	}
	return &qanv1.GetQueryPlanResponse{Planid: planid, QueryPlan: plan}, nil
}

// QueryExists reports whether a queryid has been seen.
func (s *Service) QueryExists(ctx context.Context, in *qanv1.QueryExistsRequest) (*qanv1.QueryExistsResponse, error) {
	exists, err := s.reporter.QueryExists(ctx, in.Query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query exists failed: %v", err)
	}
	return &qanv1.QueryExistsResponse{Exists: exists}, nil
}

// SchemaByQueryID returns the schema a query ran against for a service.
func (s *Service) SchemaByQueryID(ctx context.Context, in *qanv1.SchemaByQueryIDRequest) (*qanv1.SchemaByQueryIDResponse, error) {
	schema, err := s.reporter.SchemaForQuery(ctx, in.ServiceId, in.QueryId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "schema failed: %v", err)
	}
	return &qanv1.SchemaByQueryIDResponse{Schema: schema}, nil
}
