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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan-api2/models"
)

// GetMetrics implements rpc to get metrics for specific filtering.
func (s *Service) GetMetrics(ctx context.Context, in *qanpb.GetMetricsRequest) (*qanpb.GetMetricsResponse, error) {
	if in.PeriodStartFrom == nil {
		return nil, fmt.Errorf("period_start_from is required:%v", in.PeriodStartFrom)
	}
	periodStartFromSec := in.PeriodStartFrom.Seconds
	if in.PeriodStartTo == nil {
		return nil, fmt.Errorf("period_start_to is required:%v", in.PeriodStartTo)
	}
	periodStartToSec := in.PeriodStartTo.Seconds

	labels := make(map[string][]string)
	dimensions := make(map[string][]string)

	for _, label := range in.GetLabels() {
		if isDimension(label.Key) {
			dimensions[label.Key] = label.Value
			continue
		}
		labels[label.Key] = label.Value
	}

	m := make(map[string]*qanpb.MetricValues)
	t := make(map[string]*qanpb.MetricValues)
	resp := &qanpb.GetMetricsResponse{
		Metrics: m,
		Totals:  t,
	}

	var metrics models.M
	// skip on TOTAL request.
	if !in.Totals {
		metricsList, err := s.mm.Get(
			ctx,
			periodStartFromSec,
			periodStartToSec,
			in.FilterBy, // filter by queryid, or other.
			in.GroupBy,
			dimensions,
			labels,
			in.Totals)
		if err != nil {
			return nil, fmt.Errorf("error in quering metrics:%w", err)
		}

		if len(metricsList) < 2 {
			logrus.Debugf("metrics not found for filter: %s and group: %s in given time range", in.FilterBy, in.GroupBy)
			return &qanpb.GetMetricsResponse{}, nil
		}
		// Get metrics of one queryid, server etc. without totals
		metrics = metricsList[0]
	}

	totalsList, err := s.mm.Get(
		ctx,
		periodStartFromSec,
		periodStartToSec,
		"", // empty filter by (queryid, or other)
		in.GroupBy,
		dimensions,
		labels,
		true) // get Totals
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get metrics totals")
	}

	totalLen := len(totalsList)
	if totalLen < 2 {
		logrus.Debugf("totals not found for filter: %s and group: %s in given time range", in.FilterBy, in.GroupBy)
		return &qanpb.GetMetricsResponse{}, nil
	}

	// Get totals for given filter
	totals := totalsList[totalLen-1]

	durationSec := periodStartToSec - periodStartFromSec

	// skip on TOTAL request.
	if !in.Totals {
		// populate metrics and totals.
		resp.Metrics = makeMetrics(metrics, totals, durationSec)
	}
	resp.Totals = makeMetrics(totals, totals, durationSec)

	sparklines, err := s.mm.SelectSparklines(
		ctx,
		periodStartFromSec,
		periodStartToSec,
		in.FilterBy,
		in.GroupBy,
		dimensions,
		labels)
	if err != nil {
		return resp, err
	}
	resp.Sparkline = sparklines

	resp.TextMetrics = makeTextMetrics(metrics)

	if in.GroupBy == "queryid" {
		fp, err := s.mm.GetFingerprintByQueryID(ctx, in.FilterBy)
		if err != nil {
			return resp, err
		}
		resp.Fingerprint = fp
	}

	metadata, err := s.mm.GetSelectedQueryMetadata(ctx, periodStartFromSec,
		periodStartToSec,
		in.FilterBy, // filter by queryid, or other.
		in.GroupBy,
		dimensions,
		labels,
		in.Totals)
	if err != nil {
		return resp, err
	}
	resp.Metadata = metadata

	return resp, err
}

func makeTextMetrics(mm models.M) map[string]string {
	m := make(map[string]string)

	m["top_queryid"] = interfaceToString(mm["top_queryid"])
	m["top_query"] = interfaceToString(mm["top_query"])
	m["plan_summary"] = interfaceToString(mm["plan_summary"])

	return m
}

func makeMetrics(mm, t models.M, durationSec int64) map[string]*qanpb.MetricValues {
	m := make(map[string]*qanpb.MetricValues)
	sumNumQueries := interfaceToFloat32(mm["num_queries"])
	m["num_queries"] = &qanpb.MetricValues{
		Sum:  sumNumQueries,
		Rate: sumNumQueries / float32(durationSec),
	}

	sumNumQueriesWithErrors := interfaceToFloat32(mm["num_queries_with_errors"])
	m["num_queries_with_errors"] = &qanpb.MetricValues{
		Sum:  sumNumQueriesWithErrors,
		Rate: sumNumQueriesWithErrors / float32(durationSec),
	}

	sumNumQueriesWithWarnings := interfaceToFloat32(mm["num_queries_with_warnings"])
	m["num_queries_with_warnings"] = &qanpb.MetricValues{
		Sum:  sumNumQueriesWithWarnings,
		Rate: sumNumQueriesWithWarnings / float32(durationSec),
	}

	for k := range commonColumnNames {
		cnt := interfaceToFloat32(mm["m_"+k+"_cnt"])
		sum := interfaceToFloat32(mm["m_"+k+"_sum"])
		totalSum := interfaceToFloat32(mm["m_"+k+"sum"])
		mv := qanpb.MetricValues{
			Cnt: cnt,
			Sum: sum,
			Min: interfaceToFloat32(mm["m_"+k+"_min"]),
			Max: interfaceToFloat32(mm["m_"+k+"_max"]),
			P99: interfaceToFloat32(mm["m_"+k+"_p99"]),
		}
		if sumNumQueries > 0 && sum > 0 {
			mv.Avg = sum / sumNumQueries
		}
		if sum > 0 && totalSum > 0 {
			mv.PercentOfTotal = sum / totalSum
		}
		if sum > 0 && durationSec > 0 {
			mv.Rate = sum / float32(durationSec)
		}
		m[k] = &mv
	}

	for k := range sumColumnNames {
		cnt := interfaceToFloat32(mm["m_"+k+"_cnt"])
		sum := interfaceToFloat32(mm["m_"+k+"_sum"])
		totalSum := interfaceToFloat32(t["m_"+k+"sum"])
		mv := qanpb.MetricValues{
			Cnt: cnt,
			Sum: sum,
		}
		if sumNumQueries > 0 && sum > 0 {
			mv.Avg = sum / sumNumQueries
		}
		if sum > 0 && totalSum > 0 {
			mv.PercentOfTotal = sum / totalSum
		}
		if sum > 0 && durationSec > 0 {
			mv.Rate = sum / float32(durationSec)
		}
		m[k] = &mv
	}
	return m
}

// GetQueryExample gets query examples in given time range for queryid.
func (s *Service) GetQueryExample(ctx context.Context, in *qanpb.GetQueryExampleRequest) (*qanpb.GetQueryExampleResponse, error) {
	if in.PeriodStartFrom == nil {
		return nil, fmt.Errorf("period_start_from is required:%v", in.PeriodStartFrom)
	}
	if in.PeriodStartTo == nil {
		return nil, fmt.Errorf("period_start_to is required:%v", in.PeriodStartTo)
	}

	from := time.Unix(in.PeriodStartFrom.Seconds, 0)
	to := time.Unix(in.PeriodStartTo.Seconds, 0)

	labels := make(map[string][]string)
	dimensions := make(map[string][]string)

	for _, label := range in.GetLabels() {
		if isDimension(label.Key) {
			dimensions[label.Key] = label.Value
			continue
		}
		labels[label.Key] = label.Value
	}

	limit := uint32(1)
	if in.Limit > 1 {
		limit = in.Limit
	}

	group := "queryid"
	if in.GroupBy != "" {
		group = in.GroupBy
	}
	resp, err := s.mm.SelectQueryExamples(
		ctx,
		from,
		to,
		in.FilterBy,
		group,
		limit,
		dimensions,
		labels)
	if err != nil {
		return nil, errors.Wrap(err, "error in selecting query examples")
	}
	return resp, nil
}

// GetLabels gets labels in given time range for object.
func (s *Service) GetLabels(ctx context.Context, in *qanpb.GetLabelsRequest) (*qanpb.GetLabelsResponse, error) {
	if in.PeriodStartFrom == nil {
		return nil, fmt.Errorf("period_start_from is required: %v", in.PeriodStartFrom)
	}
	if in.PeriodStartTo == nil {
		return nil, fmt.Errorf("period_start_to is required: %v", in.PeriodStartTo)
	}
	if in.FilterBy != "" && in.GroupBy == "" {
		return nil, fmt.Errorf("group_by is required if filter_by is not empty %v = %v", in.GroupBy, in.FilterBy)
	}

	from := time.Unix(in.PeriodStartFrom.Seconds, 0)
	to := time.Unix(in.PeriodStartTo.Seconds, 0)
	if from.After(to) {
		return nil, fmt.Errorf("from time (%s) cannot be after to (%s)", in.PeriodStartFrom, in.PeriodStartTo)
	}

	resp, err := s.mm.SelectObjectDetailsLabels(
		ctx,
		from,
		to,
		in.FilterBy,
		in.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("error in selecting object details labels:%w", err)
	}
	return resp, nil
}

// GetQueryPlan gets query plan and plan ID for given queryid.
func (s *Service) GetQueryPlan(ctx context.Context, in *qanpb.GetQueryPlanRequest) (*qanpb.GetQueryPlanResponse, error) {
	resp, err := s.mm.SelectQueryPlan(
		ctx,
		in.Queryid)
	if err != nil {
		return nil, errors.Wrap(err, "error in selecting query plans")
	}
	return resp, nil
}

// GetHistogram gets histogram for given queryid.
func (s *Service) GetHistogram(ctx context.Context, in *qanpb.GetHistogramRequest) (*qanpb.GetHistogramResponse, error) {
	if in.PeriodStartFrom == nil {
		return nil, fmt.Errorf("period_start_from is required:%v", in.PeriodStartFrom)
	}
	periodStartFromSec := in.PeriodStartFrom.Seconds
	if in.PeriodStartTo == nil {
		return nil, fmt.Errorf("period_start_to is required:%v", in.PeriodStartTo)
	}
	periodStartToSec := in.PeriodStartTo.Seconds

	if in.Queryid == "" {
		return nil, fmt.Errorf("queryid is required:%v", in.Queryid)
	}

	labels := make(map[string][]string)
	dimensions := make(map[string][]string)

	for _, label := range in.GetLabels() {
		if isDimension(label.Key) {
			dimensions[label.Key] = label.Value
			continue
		}
		labels[label.Key] = label.Value
	}

	resp, err := s.mm.SelectHistogram(
		ctx,
		periodStartFromSec,
		periodStartToSec,
		dimensions,
		labels,
		in.Queryid)
	if err != nil {
		return nil, fmt.Errorf("error in selecting histogram:%w", err)
	}

	return resp, nil
}

// QueryExists check if query value in request exists in clickhouse.
func (s *Service) QueryExists(ctx context.Context, in *qanpb.QueryExistsRequest) (*qanpb.QueryExistsResponse, error) {
	resp, err := s.mm.QueryExists(
		ctx,
		in.Serviceid,
		in.Query)
	if err != nil {
		return nil, fmt.Errorf("error in checking query:%w", err)
	}

	return &qanpb.QueryExistsResponse{Exists: resp}, nil
}

// ExplainFingerprintByQueryID get explain fingerprint and placeholders count by query ID.
func (s *Service) ExplainFingerprintByQueryID(ctx context.Context, in *qanpb.ExplainFingerprintByQueryIDRequest) (*qanpb.ExplainFingerprintByQueryIDResponse, error) {
	res, err := s.mm.ExplainFingerprintByQueryID(
		ctx,
		in.Serviceid,
		in.QueryId)
	if err != nil {
		return nil, fmt.Errorf("error in checking query:%w", err)
	}

	return res, nil
}

// SchemaByQueryID returns schema for given queryID and serviceID.
func (s *Service) SchemaByQueryID(ctx context.Context, in *qanpb.SchemaByQueryIDRequest) (*qanpb.SchemaByQueryIDResponse, error) {
	res, err := s.mm.SchemaByQueryID(
		ctx,
		in.ServiceId,
		in.QueryId)
	if err != nil {
		return nil, fmt.Errorf("error in checking query:%w", err)
	}

	return res, nil
}
