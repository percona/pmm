// qan-api
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
	"time"

	"github.com/Percona-Lab/qan-api/models"
	pbqan "github.com/percona/pmm/api/qan"
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

// DataInterchange implements rpc to exchange data between API and agent.
func (s *Service) GetReport(ctx context.Context, in *pbqan.ReportRequest) (*pbqan.ReportReply, error) {
	// TODO: add validator/sanitazer
	labels := in.GetLabels()
	dQueryids := []string{}
	dServers := []string{}
	dDatabases := []string{}
	dSchemas := []string{}
	dUsernames := []string{}
	dClientHosts := []string{}
	dbLabels := map[string][]string{}
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
	results, _ := s.rm.Select(in.PeriodStartFrom, in.PeriodStartTo, in.Keyword, in.FirstSeen, dQueryids, dServers, dDatabases, dSchemas, dUsernames, dClientHosts, dbLabels, in.GroupBy, in.OrderBy, in.Offset, in.Limit)

	fromDate, _ := time.Parse("2006-01-02 15:04:05", in.PeriodStartFrom)
	toDate, _ := time.Parse("2006-01-02 15:04:05", in.PeriodStartTo)
	timeInterval := float32(toDate.Unix() - fromDate.Unix())

	reply := &pbqan.ReportReply{}

	var total models.DimensionReport
	for i, result := range results {
		if i == 0 {
			total = result
			reply.Rows = append(reply.Rows, &pbqan.ProfileRow{
				Rank:       0,
				Percentage: 1, // 100%
				Dimension:  total.Dimension,
				RowNumber:  total.RowNumber,
				Qps:        float32(total.NumQueries) / timeInterval,
				Load:       total.MQueryTimeSum / timeInterval,
				Stats: &pbqan.Stats{
					NumQueries:    total.NumQueries,
					MQueryTimeSum: total.MQueryTimeSum,
					MQueryTimeMin: total.MQueryTimeMin,
					MQueryTimeMax: total.MQueryTimeMax,
					MQueryTimeP99: total.MQueryTimeP99,
				},
			})
			continue
		}

		reply.Rows = append(reply.Rows, &pbqan.ProfileRow{
			Rank:        uint32(int(in.Offset) + i),
			Percentage:  result.MQueryTimeSum / total.MQueryTimeSum,
			Dimension:   result.Dimension,
			Fingerprint: result.Fingerprint,
			Qps:         float32(result.NumQueries) / timeInterval,
			Load:        result.MQueryTimeSum / timeInterval,
			Stats: &pbqan.Stats{
				NumQueries:    result.NumQueries,
				MQueryTimeSum: result.MQueryTimeSum,
				MQueryTimeMin: result.MQueryTimeMin,
				MQueryTimeMax: result.MQueryTimeMax,
				MQueryTimeP99: result.MQueryTimeP99,
			},
		})
	}

	return &pbqan.ReportReply{Rows: reply.Rows}, nil
}
