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
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

func TestService_GetQueryExample(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	var want qanpb.QueryExampleReply
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.QueryExampleRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.QueryExampleReply
		wantErr bool
	}{
		{
			"no_period_start_from",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.QueryExampleRequest{
					PeriodStartTo: &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:       "queryid",
					FilterBy:      "B305F6354FA21F2A",
					Limit:         5,
				},
			},
			nil,
			true,
		},
		{
			"no_period_start_to",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.QueryExampleRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "B305F6354FA21F2A",
					Limit:           5,
				},
			},
			nil,
			true,
		},
		{
			"no_group",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.QueryExampleRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					FilterBy:        "B305F6354FA21F2A",
					Limit:           5,
				},
			},
			&want,
			false,
		},
		{
			"no_limit",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.QueryExampleRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "B305F6354FA21F2A",
				},
			},
			&want,
			false,
		},
		{
			"invalid_group_name",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.QueryExampleRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "invalid_group_name",
					FilterBy:        "B305F6354FA21F2A",
				},
			},
			nil,
			true,
		},
		{
			"not_found",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.QueryExampleRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "unexist",
				},
			},
			&want,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				rm: tt.fields.rm,
				mm: tt.fields.mm,
			}
			got, err := s.GetQueryExample(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetQueryExample() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want = nil
			expectedData(t, got, &tt.want, "../../test_data/GetQueryExample_"+tt.name+".json")

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.GetQueryExample() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetMetrics(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	var want qanpb.MetricsReply

	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.MetricsRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.MetricsReply
		wantErr bool
	}{
		{
			"group_by_queryid",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.MetricsRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "B305F6354FA21F2A",
				},
			},
			&want,
			false,
		},
		{
			"group_by_queryid_total",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.MetricsRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "",
				},
			},
			&want,
			false,
		},
		{
			"not_found",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.MetricsRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "unexist",
				},
			},
			nil,
			true,
		},
		{
			"no_period_start_from",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.MetricsRequest{
					PeriodStartTo: &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:       "queryid",
					FilterBy:      "B305F6354FA21F2A",
				},
			},
			nil,
			true,
		},
		{
			"no_period_start_to",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.MetricsRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					GroupBy:         "queryid",
					FilterBy:        "B305F6354FA21F2A",
				},
			},
			nil,
			true,
		},
		{
			"invalid_group_name",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.MetricsRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
					GroupBy:         "no_group_name",
					FilterBy:        "B305F6354FA21F2A",
				},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				rm: tt.fields.rm,
				mm: tt.fields.mm,
			}
			got, err := s.GetMetrics(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want = nil
			expectedData(t, got, &tt.want, "../../test_data/GetMetrics_"+tt.name+".json")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.GetMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}
