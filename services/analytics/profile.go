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
	"strings"

	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

// GetReport implements rpc to get report for given filtering.
func (s *Service) GetReport(ctx context.Context, in *qanpb.ReportRequest) (*qanpb.ReportReply, error) {

	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		return nil, fmt.Errorf("from-date: %s or to-date: %s cannot be empty", in.PeriodStartFrom, in.PeriodStartTo)
	}

	periodStartFromSec := in.PeriodStartFrom.Seconds
	periodStartToSec := in.PeriodStartTo.Seconds
	if periodStartFromSec > periodStartToSec {
		return nil, fmt.Errorf("from-date %s cannot be bigger then to-date %s", in.PeriodStartFrom, in.PeriodStartTo)
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

	labels := in.GetLabels()
	dQueryids := []string{}
	dServers := []string{}
	dDatabases := []string{}
	dSchemas := []string{}
	dUsernames := []string{}
	dClientHosts := []string{}
	dbLabels := map[string][]string{}
	columns := in.Columns
	for _, label := range labels {
		switch label.Key {
		case "queryid":
			dQueryids = label.Value
		case "server":
			dServers = label.Value
		case "database":
			dDatabases = label.Value
		case "schema":
			dSchemas = label.Value
		case "username":
			dUsernames = label.Value
		case "client_host":
			dClientHosts = label.Value
		default:
			dbLabels[label.Key] = label.Value
		}
	}

	orderCol := in.OrderBy

	// TODO: remove this when UI done.
	if strings.TrimPrefix(in.OrderBy, "-") == "load" {
		columns = append([]string{"load", "num_queries"}, columns...)
	}
	// TODO: remove this when UI done.
	if strings.TrimPrefix(in.OrderBy, "-") == "count" {
		orderCol = "num_queries"
		columns = append([]string{"load", "num_queries"}, columns...)
	}

	mainMetric := in.MainMetric
	if mainMetric == "" {
		mainMetric = columns[0]
	}
	uniqColumnsMap := map[string]struct{}{}
	for _, column := range columns {
		if _, ok := uniqColumnsMap[column]; !ok {
			uniqColumnsMap[column] = struct{}{}
		}
	}

	uniqColumns := []string{}
	for key := range uniqColumnsMap {
		uniqColumns = append(uniqColumns, key)
	}

	boolColumns := []string{}
	commonColumns := []string{}
	specialColumns := []string{}
	for _, col := range uniqColumns {
		if _, ok := boolColumnNames[col]; ok {
			boolColumns = append(boolColumns, col)
			continue
		}
		if _, ok := commonColumnNames[col]; ok {
			commonColumns = append(commonColumns, col)
			continue
		}
		if _, ok := specialColumnNames[col]; ok {
			specialColumns = append(specialColumns, col)
			continue
		}
	}

	if orderCol == "" {
		orderCol = uniqColumns[0]
	}

	direction := "ASC"
	if orderCol[0] == '-' {
		orderCol = orderCol[1:]
		direction = "DESC"
	}

	if _, ok := uniqColumnsMap[orderCol]; !ok {
		return nil, fmt.Errorf("order column '%s' not in selected columns: [%s]", orderCol, strings.Join(uniqColumns, ", "))
	}

	_, isBoolCol := boolColumnNames[orderCol]
	_, isCommonCol := commonColumnNames[orderCol]

	if isBoolCol || isCommonCol {
		orderCol = fmt.Sprintf("m_%s_sum", orderCol)
	}
	order := fmt.Sprintf("%s %s", orderCol, direction)

	resp := &qanpb.ReportReply{}
	results, err := s.rm.Select(
		ctx,
		periodStartFromSec,
		periodStartToSec,
		dQueryids,
		dServers,
		dDatabases,
		dSchemas,
		dUsernames,
		dClientHosts,
		dbLabels,
		group,
		order,
		in.Offset,
		limit,
		specialColumns,
		commonColumns,
		boolColumns,
	)

	if err != nil {
		return nil, err
	}

	total := results[0]
	resp.TotalRows = uint32(total["total_rows"].(uint64))
	resp.Offset = in.Offset
	resp.Limit = in.Limit

	for i, res := range results {
		numQueries := interfaceToFloat32(res["num_queries"])
		row := &qanpb.Row{
			Rank:        uint32(i) + in.Offset,
			Dimension:   res["dimension"].(string),
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

		sparklines, err := s.rm.SelectSparklines(
			ctx,
			row.Dimension,
			periodStartFromSec,
			periodStartToSec,
			dQueryids,
			dServers,
			dDatabases,
			dSchemas,
			dUsernames,
			dClientHosts,
			dbLabels,
			group,
			mainMetric,
		)
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
	if metricNameRoot == "load" {
		return &qanpb.Stat{
			SumPerSec: interfaceToFloat32(res["load"]),
		}
	}
	if metricNameRoot == "num_queries" {
		return &qanpb.Stat{
			Sum:       numQueries,
			SumPerSec: numQueries / float32(periodDurationSec),
		}
	}
	rate := float32(0)
	divider := interfaceToFloat32(total["m_"+metricNameRoot+"_sum"])
	sum := interfaceToFloat32(res["m_"+metricNameRoot+"_sum"])
	if divider != 0 {
		rate = sum / divider
	}
	stat := &qanpb.Stat{
		Rate:      rate,
		Cnt:       interfaceToFloat32(res["m_"+metricNameRoot+"_cnt"]),
		Sum:       sum,
		Avg:       sum / numQueries,
		SumPerSec: sum / float32(periodDurationSec),
	}
	if val, ok := res["m_"+metricNameRoot+"_min"]; ok {
		stat.Min = interfaceToFloat32(val)
	}
	if val, ok := res["m_"+metricNameRoot+"_max"]; ok {
		stat.Max = interfaceToFloat32(val)
	}
	if val, ok := res["m_"+metricNameRoot+"_p99"]; ok {
		stat.P99 = interfaceToFloat32(val)
	}
	return stat

}
