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

// Package analytics contains business logic of preparing query analytics for UI.
package analytics

import (
	"context"
	"fmt"
	"strings"

	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan-api2/models"
)

// GetReport implements rpc to get report for given filtering.
func (s *Service) GetReport(ctx context.Context, in *qanpb.GetReportRequest) (*qanpb.GetReportResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, fmt.Errorf("from-date: %s or to-date: %s cannot be empty", in.PeriodStartFrom, in.PeriodStartTo)
	}

	periodStartFromSec := in.PeriodStartFrom.Seconds
	periodStartToSec := in.PeriodStartTo.Seconds
	if periodStartFromSec > periodStartToSec {
		return nil, fmt.Errorf("from-date %s cannot be later then to-date %s", in.PeriodStartFrom, in.PeriodStartTo)
	}
	periodDurationSec := periodStartToSec - periodStartFromSec

	if _, ok := standartDimensions[in.GroupBy]; !ok {
		return nil, fmt.Errorf("unknown group dimension: %s", in.GroupBy)
	}
	group := in.GroupBy

	limit := in.Limit
	if limit == 0 {
		limit = 10
	}

	labels := make(map[string][]string)
	dimensions := make(map[string][]string)

	for _, label := range in.GetLabels() {
		if models.IsDimension(label.Key) {
			dimensions[label.Key] = label.Value
			continue
		}
		labels[label.Key] = label.Value
	}

	var columns []string //nolint:prealloc
	for _, col := range in.Columns {
		// TODO: remove when UI starts using num_queries instead.
		if col == "count" {
			col = "num_queries"
		}
		columns = append(columns, col)
	}

	if strings.TrimPrefix(in.OrderBy, "-") == "load" {
		columns = append([]string{"load", "num_queries", "query_time"}, columns...)
	}

	if strings.TrimPrefix(in.OrderBy, "-") == "count" {
		columns = append([]string{"load", "num_queries"}, columns...)
	}

	mainMetric := in.MainMetric
	if mainMetric == "" {
		mainMetric = columns[0]
	}
	uniqColumnsMap := make(map[string]struct{})
	for _, column := range columns {
		if _, ok := uniqColumnsMap[column]; !ok {
			uniqColumnsMap[column] = struct{}{}
		}
	}

	uniqColumns := make([]string, 0, len(uniqColumnsMap))
	for key := range uniqColumnsMap {
		uniqColumns = append(uniqColumns, key)
	}

	var sumColumns, commonColumns, specialColumns []string
	for _, col := range uniqColumns {
		if isBoolMetric(col) {
			sumColumns = append(sumColumns, col)
			continue
		}
		if isCommonMetric(col) {
			commonColumns = append(commonColumns, col)
			continue
		}
		if isSpecialMetric(col) {
			specialColumns = append(specialColumns, col)
			continue
		}
	}

	order, orderCol := getOrderBy(in.OrderBy, uniqColumns[0])
	if _, ok := uniqColumnsMap[orderCol]; !ok {
		return nil, fmt.Errorf("order column '%s' not in selected columns: [%s]", orderCol, strings.Join(uniqColumns, ", "))
	}

	resp := &qanpb.GetReportResponse{}
	results, err := s.rm.Select(
		ctx,
		periodStartFromSec,
		periodStartToSec,
		dimensions,
		labels,
		group,
		order,
		in.Search,
		in.Offset,
		limit,
		specialColumns,
		commonColumns,
		sumColumns)
	if err != nil {
		return nil, err
	}

	total := results[0]
	resp.TotalRows = uint32(total["total_rows"].(uint64)) //nolint:forcetypeassert,gosec // TODO: fix it
	resp.Offset = in.Offset
	resp.Limit = in.Limit

	for i, res := range results {
		numQueries := interfaceToFloat32(res["num_queries"])
		//nolint:forcetypeassert
		row := &qanpb.Row{
			Rank:        uint32(i) + in.Offset,
			Dimension:   res["dimension"].(string),
			Database:    res["database_name"].(string),
			Fingerprint: res["fingerprint"].(string),
			NumQueries:  uint32(numQueries),                                                       // TODO: deprecated, remove it when UI stop use it.
			Qps:         numQueries / float32(periodDurationSec),                                  // TODO: deprecated, remove it when UI stop use it.
			Load:        interfaceToFloat32(res["m_query_time_sum"]) / float32(periodDurationSec), // TODO: deprecated, remove it when UI stop use it.
			Metrics:     make(map[string]*qanpb.Metric),
		}

		// set TOTAL for Fingerprint instead of "any" if result is not empty.
		if i == 0 && row.Fingerprint != "" {
			row.Fingerprint = "TOTAL"
		}

		// The row with index 0 is total.
		isTotal := i == 0

		sparklines, err := s.rm.SelectSparklines(
			ctx,
			row.Dimension,
			periodStartFromSec,
			periodStartToSec,
			dimensions,
			labels,
			group,
			mainMetric,
			isTotal)
		if err != nil {
			return nil, err
		}
		row.Sparkline = sparklines
		for _, c := range columns {
			stats := makeStats(c, total, res, numQueries, periodDurationSec)
			row.Metrics[c] = &qanpb.Metric{
				Stats: stats,
			}
		}
		resp.Rows = append(resp.Rows, row)
	}
	return resp, nil
}

func makeStats(metricNameRoot string, total, res models.M, numQueries float32, periodDurationSec int64) *qanpb.Stat {
	var stat qanpb.Stat
	durSec := float32(periodDurationSec)
	switch metricNameRoot {
	case "load":
		stat.SumPerSec = interfaceToFloat32(res["load"])
	case "num_queries":
		stat.Sum = numQueries
		stat.SumPerSec = numQueries / durSec
	case "num_queries_with_errors":
		stat.Sum = interfaceToFloat32(res["num_queries_with_errors"])
		stat.SumPerSec = interfaceToFloat32(res["num_queries_with_errors"]) / durSec
	case "num_queries_with_warnings":
		stat.Sum = interfaceToFloat32(res["num_queries_with_warnings"])
		stat.SumPerSec = interfaceToFloat32(res["num_queries_with_warnings"]) / durSec
	default:
		rate := float32(0)
		divider := interfaceToFloat32(total["m_"+metricNameRoot+"_sum"])
		sum := interfaceToFloat32(res["m_"+metricNameRoot+"_sum"])
		if divider != 0 {
			rate = sum / divider
		}

		stat.Rate = rate
		stat.Cnt = interfaceToFloat32(res["m_"+metricNameRoot+"_cnt"])
		stat.Sum = sum
		stat.SumPerSec = sum / durSec

		if val, ok := res["m_"+metricNameRoot+"_min"]; ok {
			stat.Min = interfaceToFloat32(val)
		}

		if val, ok := res["m_"+metricNameRoot+"_max"]; ok {
			stat.Max = interfaceToFloat32(val)
		}

		if val, ok := res["m_"+metricNameRoot+"_avg"]; ok {
			stat.Avg = interfaceToFloat32(val)
		}

		if val, ok := res["m_"+metricNameRoot+"_p99"]; ok {
			stat.P99 = interfaceToFloat32(val)
		}
	}

	return &stat
}

// getOrderBy creates an order by string to use in query and column name to check if it in select column list.
func getOrderBy(reqOrder, defaultOrder string) (string, string) {
	var queryOrder, orderCol string
	direction := "ASC"
	if strings.HasPrefix(reqOrder, "-") {
		reqOrder = strings.TrimPrefix(reqOrder, "-")
		direction = "DESC"
	}

	switch {
	case reqOrder == "count":
		orderCol = "num_queries"
		queryOrder = fmt.Sprintf("%s %s", orderCol, direction)
	case reqOrder == "load":
		orderCol = "query_time"
		queryOrder = fmt.Sprintf("m_%s_sum %s", orderCol, direction)
	// order by average for time metrics.
	case isTimeMetric(reqOrder):
		orderCol = reqOrder
		queryOrder = fmt.Sprintf("m_%s_avg %s", orderCol, direction)
	// order by sum for all common not time metrics.
	case isCommonMetric(reqOrder) || isBoolMetric(reqOrder):
		orderCol = reqOrder
		queryOrder = fmt.Sprintf("m_%s_sum %s", orderCol, direction)
	case isSpecialMetric(reqOrder):
		orderCol = reqOrder
		queryOrder = fmt.Sprintf("%s %s", orderCol, direction)
	// on empty - order by the first column.
	default:
		orderCol = defaultOrder
		queryOrder = fmt.Sprintf("%s %s", orderCol, direction)
	}

	return queryOrder, orderCol
}
