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
	// TODO: add validator/sanitazer
	labels := in.GetLabels()
	dQueryids := []string{}
	dServers := []string{}
	dDatabases := []string{}
	dSchemas := []string{}
	dUsernames := []string{}
	dClientHosts := []string{}
	dbLabels := map[string][]string{}
	columns := in.Columns
	if len(columns) == 0 {
		columns = append(columns, "lock_time")
	}
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

	resp := &qanpb.ReportReply{}
	results, err := s.rm.Select(
		in.PeriodStartFrom,
		in.PeriodStartTo,
		in.Keyword,
		in.FirstSeen,
		dQueryids,
		dServers,
		dDatabases,
		dSchemas,
		dUsernames,
		dClientHosts,
		dbLabels,
		in.GroupBy,
		in.OrderBy,
		in.Offset,
		in.Limit,
		columns,
	)

	if err != nil {
		return resp, err
	}

	total := results[0]
	resp.TotalRows = uint32(total["total_rows"].(uint64))
	resp.Offset = in.Offset
	resp.Limit = in.Limit

	for i, res := range results {
		row := &qanpb.Row{
			Rank:      uint32(i) + in.Offset,
			Dimension: res["dimension"].(string),
			Metrics:   make(map[string]*qanpb.Metric),
		}

		sparklines, err := s.rm.SelectSparklines(
			row.Dimension,
			in.PeriodStartFrom,
			in.PeriodStartTo,
			in.Keyword,
			in.FirstSeen,
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
			row.Metrics[c] = &qanpb.Metric{
				Stats: &qanpb.Stat{
					Rate: rate,
					Cnt:  interfaceToFloat32(res["m_"+c+"_cnt"]),
					Sum:  interfaceToFloat32(res["m_"+c+"_sum"]),
					Min:  interfaceToFloat32(res["m_"+c+"_min"]),
					Max:  interfaceToFloat32(res["m_"+c+"_max"]),
					P99:  interfaceToFloat32(res["m_"+c+"_p99"]),
				},
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
