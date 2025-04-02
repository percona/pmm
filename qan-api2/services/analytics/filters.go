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

	qanpb "github.com/percona/pmm/api/qan/v1"
)

// GetFilteredMetricsNames implements rpc to get list of available metrics names.
//
//nolint:goconst
func (s *Service) GetFilteredMetricsNames(ctx context.Context, in *qanpb.GetFilteredMetricsNamesRequest) (*qanpb.GetFilteredMetricsNamesResponse, error) {
	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		err := fmt.Errorf("from-date: %s or to-date: %s cannot be empty", in.PeriodStartFrom, in.PeriodStartTo)
		return nil, err
	}

	periodStartFromSec := in.PeriodStartFrom.Seconds
	periodStartToSec := in.PeriodStartTo.Seconds
	if periodStartFromSec > periodStartToSec {
		err := fmt.Errorf("from-date %s cannot be bigger then to-date %s", in.PeriodStartFrom, in.PeriodStartTo)
		return nil, err
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

	var mainMetricName string
	switch in.MainMetricName {
	case "":
		mainMetricName = "m_query_time_sum"
	case "load":
		mainMetricName = "m_query_time_sum"
	case "num_queries":
		mainMetricName = "num_queries"
	case "count":
		mainMetricName = "num_queries"
	case "num_queries_with_errors":
		mainMetricName = "num_queries_with_errors"
	case "num_queries_with_warnings":
		mainMetricName = "num_queries_with_warnings"
	default:
		mainMetricName = fmt.Sprintf("m_%s_sum", in.MainMetricName)
	}

	return s.rm.SelectFilters(ctx, periodStartFromSec, periodStartToSec, mainMetricName, dimensions, labels)
}
