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

	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

// Service implements gRPC service to communicate with QAN-APP.
type Service struct {
	rm models.Reporter
	mm models.Metrics
}

// NewService create new insstance of Service.
func NewService(rm models.Reporter, mm models.Metrics) *Service {
	return &Service{rm, mm}
}

// GetReport implements rpc to get report for given filtering.
func (s *Service) GetReport(ctx context.Context, in *qanpb.ReportRequest) (*qanpb.ReportReply, error) {

	if in.PeriodStartFrom == nil || in.PeriodStartTo == nil {
		err := fmt.Errorf("from-date: %s or to-date: %s cannot be empty", in.PeriodStartFrom, in.PeriodStartTo)
		return &qanpb.ReportReply{}, err
	}

	from := time.Unix(in.PeriodStartFrom.Seconds, 0)
	to := time.Unix(in.PeriodStartTo.Seconds, 0)
	if from.After(to) {
		err := fmt.Errorf("from-date %s cannot be bigger then to-date %s", from.UTC(), to.UTC())
		return &qanpb.ReportReply{}, err
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

	boolColumnNames := map[string]struct{}{
		"qc_hit":                 {},
		"full_scan":              {},
		"full_join":              {},
		"tmp_table":              {},
		"tmp_table_on_disk":      {},
		"filesort":               {},
		"filesort_on_disk":       {},
		"select_full_range_join": {},
		"select_range":           {},
		"select_range_check":     {},
		"sort_range":             {},
		"sort_rows":              {},
		"sort_scan":              {},
		"no_index_used":          {},
		"no_good_index_used":     {},
	}

	boolColumns := []string{}
	commonColumns := []string{}
	for _, col := range columns {
		if _, ok := boolColumnNames[col]; ok {
			boolColumns = append(boolColumns, col)
			continue
		}
		commonColumns = append(commonColumns, col)
	}

	order := "m_query_time_sum"
	if in.OrderBy != "" {
		col := in.OrderBy
		direction := "ASC"
		if col[0] == '-' {
			col = col[1:]
			direction = "DESC"
		}

		if _, ok := boolColumnNames[col]; ok {
			switch col {
			case "load", "latency":
				col = "m_query_time_sum"
			case "count":
				col = "num_queries"
			default:
				col = fmt.Sprintf("m_%s_sum", col)
			}
			order = fmt.Sprintf("%s %s", col, direction)
		}
	}

	resp := &qanpb.ReportReply{}
	results, err := s.rm.Select(
		ctx,
		from,
		to,
		dQueryids,
		dServers,
		dDatabases,
		dSchemas,
		dUsernames,
		dClientHosts,
		dbLabels,
		in.GroupBy,
		order,
		in.Offset,
		in.Limit,
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
			Load:        interfaceToFloat32(total["m_query_time_sum"]) / float32(intervalTime),
			Metrics:     make(map[string]*qanpb.Metric),
		}

		sparklines, err := s.rm.SelectSparklines(
			ctx,
			row.Dimension,
			from,
			to,
			dQueryids,
			dServers,
			dDatabases,
			dSchemas,
			dUsernames,
			dClientHosts,
			dbLabels,
			in.GroupBy,
			columns,
		)
		if err != nil {
			return resp, err
		}
		row.Sparkline = sparklines
		for _, c := range columns {
			rate := float32(0)
			divider := interfaceToFloat32(total["m_"+c+"_sum"])
			if divider != 0 {
				rate = interfaceToFloat32(res["m_"+c+"_sum"]) / divider
			}
			stats := &qanpb.Stat{
				Rate: rate,
				Cnt:  interfaceToFloat32(res["m_"+c+"_cnt"]),
				Sum:  interfaceToFloat32(res["m_"+c+"_sum"]),
			}
			if val, ok := res["m_"+c+"_min"]; ok {
				stats.Min = interfaceToFloat32(val)
			}
			if val, ok := res["m_"+c+"_max"]; ok {
				stats.Max = interfaceToFloat32(val)
			}
			if val, ok := res["m_"+c+"_p99"]; ok {
				stats.P99 = interfaceToFloat32(val)
			}
			row.Metrics[c] = &qanpb.Metric{
				Stats: stats,
			}
		}
		resp.Rows = append(resp.Rows, row)
	}
	return resp, nil
}

func interfaceToFloat32(unk interface{}) float32 {
	switch i := unk.(type) {
	case float64:
		return float32(i)
	case float32:
		return i
	case int64:
		return float32(i)
	default:
		return float32(0)
	}
}
