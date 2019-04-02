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
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jmoiron/sqlx"
	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

const expectedDataFile = "../../test_data/profile.json"

func setup() *sqlx.DB {
	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	if !ok {
		dsn = "clickhouse://127.0.0.1:19000?database=pmm_test"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
	}

	return db
}

func TestService_GetReport(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	var want qanpb.ReportReply
	expectedData, err := ioutil.ReadFile(expectedDataFile)
	err = json.Unmarshal(expectedData, &want)
	if err != nil {
		log.Fatal("cannot unmarshal expected result: ", err)
	}
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.ReportRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.ReportReply
		wantErr bool
	}{
		{
			"success",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.ReportRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "queryid",
					Columns:         []string{"lock_time", "sort_scan"},
					OrderBy:         "load",
					Offset:          10,
					Limit:           10,
					Labels: []*qanpb.ReportMapFieldEntry{
						{
							Key:   "label1",
							Value: []string{"value1", "value2"},
						},
						{
							Key:   "d_server",
							Value: []string{"db1", "db2", "db3", "db4", "db5", "db6", "db7"},
						},
					},
				},
			},
			&want,
			false,
		},
		{
			"wrong time range",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.ReportRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t2.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t1.Unix()},
				},
			},
			&qanpb.ReportReply{},
			true,
		},
		{
			"empty fail",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.ReportRequest{},
			},
			&qanpb.ReportReply{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				rm: tt.fields.rm,
				mm: tt.fields.mm,
			}
			got, err := s.GetReport(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// TODO: why travis-ci return other values then expected?
			if got.TotalRows != tt.want.TotalRows {
				t.Errorf("got.TotalRows (%v) != *tt.want.TotalRows (%v)", got.TotalRows, tt.want.TotalRows)
			}

			for i, v := range got.Rows {
				if v.NumQueries != tt.want.Rows[i].NumQueries {
					t.Errorf("got.Rows[0].NumQueries (%v) != *tt.want.Rows[0].NumQueries (%v)", v.NumQueries, tt.want.Rows[i].NumQueries)
				}
			}
		})
	}
}

func TestService_GetReport_DescOrder(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	var want qanpb.ReportReply
	expectedData, err := ioutil.ReadFile(expectedDataFile)
	err = json.Unmarshal(expectedData, &want)
	if err != nil {
		log.Fatal("cannot unmarshal expected result: ", err)
	}
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.ReportRequest
	}
	test := struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.ReportReply
		wantErr bool
	}{
		"reverce order",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			&qanpb.ReportRequest{
				PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				Columns:         []string{"lock_time", "sort_scan"},
				OrderBy:         "-load",
				Offset:          10,
				Limit:           10,
				Labels: []*qanpb.ReportMapFieldEntry{
					{
						Key:   "label1",
						Value: []string{"value1", "value2"},
					},
					{
						Key:   "d_server",
						Value: []string{"db1", "db2", "db3", "db4", "db5", "db6", "db7"},
					},
				},
			},
		},
		&want,
		false,
	}
	t.Run(test.name, func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}
		got, err := s.GetReport(test.args.ctx, test.args.in)
		if (err != nil) != test.wantErr {
			t.Errorf("Service.GetReport() error = %v, wantErr %v", err, test.wantErr)
			return
		}

		for i, v := range got.Rows {
			if v.NumQueries != test.want.Rows[i].NumQueries {
				t.Errorf("got.Rows[0].NumQueries (%v) != *tt.want.Rows[0].NumQueries (%v)", v.Load, test.want.Rows[i].Load)
			}
		}
	})
}
