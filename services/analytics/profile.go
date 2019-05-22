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

	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

const defaultOrder = "m_query_time_sum"

// GetReport implements rpc to get report for given filtering.
func (s *Service) GetReport(ctx context.Context, in *qanpb.ReportRequest) (*qanpb.ReportReply, error) {

	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		err := fmt.Errorf("from-date: %s or to-date: %s cannot be empty", in.PeriodStartFrom, in.PeriodStartTo)
		return &qanpb.ReportReply{}, err
	}

	periodStartFromSec := in.PeriodStartFrom.Seconds
	periodStartToSec := in.PeriodStartTo.Seconds
	if periodStartFromSec > periodStartToSec {
		err := fmt.Errorf("from-date %s cannot be bigger then to-date %s", in.PeriodStartFrom, in.PeriodStartTo)
		return &qanpb.ReportReply{}, err
	}

	if _, ok := standartDimensions[in.GroupBy]; !ok {
		return &qanpb.ReportReply{}, fmt.Errorf("unknown group dimension: %s", in.GroupBy)
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
		case "d_server":
			dServers = label.Value
		case "d_database":
			dDatabases = label.Value
		case "d_schema":
			dSchemas = label.Value
		case "d_username":
			dUsernames = label.Value
		case "d_client_host":
			dClientHosts = label.Value
		default:
			dbLabels[label.Key] = label.Value
		}
	}

	boolColumns := []string{}
	commonColumns := []string{}
	for _, col := range columns {
		if _, ok := boolColumnNames[col]; ok {
			boolColumns = append(boolColumns, col)
			continue
		}
		if _, ok := commonColumnNames[col]; ok {
			commonColumns = append(commonColumns, col)
			continue
		}
	}

	order := defaultOrder
	if in.OrderBy != "" {
		col := in.OrderBy
		direction := "ASC"
		if col[0] == '-' {
			col = col[1:]
			direction = "DESC"
		}

		_, isBoolCol := boolColumnNames[col]
		_, isCommonCol := commonColumnNames[col]

		switch {
		case isBoolCol || isCommonCol:
			col = fmt.Sprintf("m_%s_sum", col)
		case col == "load" || col == "latency":
			col = defaultOrder
		case col == "count":
			col = "num_queries"
		default:
			col = defaultOrder
		}
		order = fmt.Sprintf("%s %s", col, direction)
	}

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
		commonColumns,
		boolColumns,
	)

	if err != nil {
		return resp, err
	}

	total := results[0]
	resp.TotalRows = uint32(total["total_rows"].(uint64))
	resp.Offset = in.Offset
	resp.Limit = in.Limit
	intervalTime := in.PeriodStartTo.Seconds - in.PeriodStartFrom.Seconds

	for i, res := range results {
		numQueries := interfaceToFloat32(res["num_queries"])
		row := &qanpb.Row{
			Rank:        uint32(i) + in.Offset,
			Dimension:   res["dimension"].(string),
			Fingerprint: res["fingerprint"].(string),
			NumQueries:  uint32(numQueries),
			Qps:         float32(int64(numQueries) / intervalTime),
			Load:        interfaceToFloat32(res["m_query_time_sum"]) / float32(intervalTime),
			Metrics:     make(map[string]*qanpb.Metric),
		}
		// Add latency as default column.
		stats := makeStats("query_time", total, res, numQueries)
		row.Metrics["latency"] = &qanpb.Metric{
			Stats: stats,
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
			columns,
		)
		if err != nil {
			return resp, err
		}
		row.Sparkline = sparklines
		for _, c := range columns {
			stats := makeStats(c, total, res, numQueries)
			row.Metrics[c] = &qanpb.Metric{
				Stats: stats,
			}
		}
		resp.Rows = append(resp.Rows, row)
	}
	return resp, nil
}

func makeStats(metricNameRoot string, total, res models.M, numQueries float32) *qanpb.Stat {
	rate := float32(0)
	divider := interfaceToFloat32(total["m_"+metricNameRoot+"_sum"])
	sum := interfaceToFloat32(res["m_"+metricNameRoot+"_sum"])
	if divider != 0 {
		rate = sum / divider
	}
	stat := &qanpb.Stat{
		Rate: rate,
		Cnt:  interfaceToFloat32(res["m_"+metricNameRoot+"_cnt"]),
		Sum:  sum,
		Avg:  sum / numQueries,
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
