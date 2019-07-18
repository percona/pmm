// qan-api2
// Copyright (C) 2019 Percona LLC
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

package analitycs

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

// GetMetrics implements rpc to get metrics for specific filtering.
func (s *Service) GetMetrics(ctx context.Context, in *qanpb.MetricsRequest) (*qanpb.MetricsReply, error) {
	if in.PeriodStartFrom == nil {
		return nil, fmt.Errorf("period_start_from is required:%v", in.PeriodStartFrom)
	}
	periodStartFromSec := in.PeriodStartFrom.Seconds
	if in.PeriodStartTo == nil {
		return nil, fmt.Errorf("period_start_to is required:%v", in.PeriodStartTo)
	}
	periodStartToSec := in.PeriodStartTo.Seconds

	labels := map[string][]string{}
	dimensions := map[string][]string{}

	for _, label := range in.GetLabels() {
		if isDimension(label.Key) {
			dimensions[label.Key] = label.Value
			continue
		}
		labels[label.Key] = label.Value
	}

	m := make(map[string]*qanpb.MetricValues)
	t := make(map[string]*qanpb.MetricValues)
	resp := &qanpb.MetricsReply{
		Metrics: m,
		Totals:  t,
	}

	var metrics models.M
	// skip on TOTAL request.
	if in.FilterBy != "" {
		metricsList, err := s.mm.Get(
			ctx,
			periodStartFromSec,
			periodStartToSec,
			in.FilterBy, // filter by queryid, or other.
			in.GroupBy,
			dimensions,
			labels,
		)
		if err != nil {
			return nil, fmt.Errorf("error in quering metrics:%v", err)
		}

		if len(metricsList) < 2 {
			return nil, fmt.Errorf("metrics not found for filter: %s and group: %s in given time range", in.FilterBy, in.GroupBy)
		}
		// Get metrics of one queryid, server etc. without totals
		metrics = metricsList[0]
	}

	// to get totals - pass empty filter by (with same main labels and time range).
	totalsList, err := s.mm.Get(
		ctx,
		periodStartFromSec,
		periodStartToSec,
		"", // empty filter by (queryid, or other)
		in.GroupBy,
		dimensions,
		labels,
	)
	if err != nil {
		return nil, fmt.Errorf("error in quering totals:%v", err)
	}

	totalLen := len(totalsList)
	if totalLen < 2 {
		return nil, fmt.Errorf("totals not found for filter: %s and group: %s in given time range", in.FilterBy, in.GroupBy)
	}

	// Get totals for given filter
	totals := totalsList[totalLen-1]

	durationSec := periodStartToSec - periodStartFromSec

	// skip on TOTAL request.
	if in.FilterBy != "" {
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
		labels,
	)
	if err != nil {
		return resp, err
	}
	resp.Sparkline = sparklines

	return resp, err
}

func makeMetrics(mm, t models.M, durationSec int64) map[string]*qanpb.MetricValues {
	m := make(map[string]*qanpb.MetricValues)
	m["num_queries"] = &qanpb.MetricValues{
		Sum: interfaceToFloat32(mm["num_queries"]),
	}
	m["num_queries_with_errors"] = &qanpb.MetricValues{
		Sum: interfaceToFloat32(mm["num_queries_with_errors"]),
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
		if cnt > 0 && sum > 0 {
			mv.Avg = sum / cnt
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
		if cnt > 0 && sum > 0 {
			mv.Avg = sum / cnt
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
func (s *Service) GetQueryExample(ctx context.Context, in *qanpb.QueryExampleRequest) (*qanpb.QueryExampleReply, error) {
	if in.PeriodStartFrom == nil {
		return nil, fmt.Errorf("period_start_from is required:%v", in.PeriodStartFrom)
	}
	if in.PeriodStartTo == nil {
		return nil, fmt.Errorf("period_start_to is required:%v", in.PeriodStartTo)
	}

	from := time.Unix(in.PeriodStartFrom.Seconds, 0)
	to := time.Unix(in.PeriodStartTo.Seconds, 0)
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
	)
	if err != nil {
		return nil, errors.Wrap(err, "error in selecting query examples")
	}
	return resp, nil
}

// GetLabels gets labels in given time range for object.
func (s *Service) GetLabels(ctx context.Context, in *qanpb.ObjectDetailsLabelsRequest) (*qanpb.ObjectDetailsLabelsReply, error) {
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
		in.GroupBy,
	)
	if err != nil {
		return nil, fmt.Errorf("error in selecting object details labels:%v", err)
	}
	return resp, nil
}
