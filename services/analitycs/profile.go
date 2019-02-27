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
	dbServers := []string{}
	dbSchemas := []string{}
	dbUsernames := []string{}
	clientHosts := []string{}
	dbLabels := map[string][]string{}
	for _, label := range labels {
		switch label.Key {
		case "db_server":
			dbServers = label.Value
		case "db_schema":
			dbSchemas = label.Value
		case "db_username":
			dbUsernames = label.Value
		case "client_host":
			clientHosts = label.Value
		default:
			dbLabels[label.Key] = label.Value
		}
	}
	total, _ := s.rm.GetTotal(in.PeriodStartFrom, in.PeriodStartTo, dbServers, dbSchemas, dbUsernames, clientHosts, dbLabels)
	classes, _ := s.rm.Select(in.PeriodStartFrom, in.PeriodStartTo, in.Keyword, in.FirstSeen, dbServers, dbSchemas, dbUsernames, clientHosts, dbLabels, in.OrderBy, in.Offset, in.Limit)

	fromDate, _ := time.Parse("2006-01-02 15:04:05", in.PeriodStartFrom)
	toDate, _ := time.Parse("2006-01-02 15:04:05", in.PeriodStartTo)
	timeInterval := float32(toDate.Unix() - fromDate.Unix())

	reply := &pbqan.ReportReply{}
	reply.Rows = append(reply.Rows, &pbqan.ProfileRow{
		Rank:       0,
		Percentage: 1, // 100%
		Digest:     "TOTAL",
		DigestText: "",
		FirstSeen:  "",
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

	for i, class := range classes {
		reply.Rows = append(reply.Rows, &pbqan.ProfileRow{
			Rank:       uint32(int(in.Offset) + i + 1),
			Percentage: class.MQueryTimeSum / total.MQueryTimeSum,
			Digest:     class.Digest1,
			DigestText: class.DigestText1,
			FirstSeen:  class.FirstSeen,
			Qps:        float32(class.NumQueries) / timeInterval,
			Load:       class.MQueryTimeSum / timeInterval,
			Stats: &pbqan.Stats{
				NumQueries:    class.NumQueries,
				MQueryTimeSum: class.MQueryTimeSum,
				MQueryTimeMin: class.MQueryTimeMin,
				MQueryTimeMax: class.MQueryTimeMax,
				MQueryTimeP99: class.MQueryTimeP99,
			},
		})
	}

	return &pbqan.ReportReply{Rows: reply.Rows}, nil
}
