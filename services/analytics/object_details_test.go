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
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
				assert.Errorf(t, err, "Service.GetQueryExample() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.want = nil
			expectedData(t, got, &tt.want, "../../test_data/GetQueryExample_"+tt.name+".json")
			assert.Equal(t, proto.MarshalTextString(got), proto.MarshalTextString(tt.want))
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
				assert.Errorf(t, err, "Service.GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.want = nil
			expectedData(t, got, &tt.want, "../../test_data/GetMetrics_"+tt.name+".json")
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Service.GetMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetLabels(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	want := qanpb.ObjectDetailsLabelsReply{}

	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.ObjectDetailsLabelsRequest
	}
	type testCase struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.ObjectDetailsLabelsReply
		wantErr error
	}

	tt := testCase{
		"success",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			&qanpb.ObjectDetailsLabelsRequest{
				PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "1D410B4BE5060972",
			},
		},
		&want,
		nil,
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		got, err := s.GetLabels(tt.args.ctx, tt.args.in)
		require.Equal(t, err, tt.wantErr)
		expectedJSON := getExpectedJSON(t, got, "../../test_data/GetLabels"+tt.name+".json")

		gotJSON, err := json.MarshalIndent(got, "", "\t")
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		require.Equal(t, expectedJSON, gotJSON)
	})

	tt = testCase{
		"required from",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			&qanpb.ObjectDetailsLabelsRequest{
				PeriodStartFrom: nil,
				PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "1D410B4BE5060972",
			},
		},
		nil,
		fmt.Errorf("period_start_from is required:%v", nil),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(tt.args.ctx, tt.args.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	tt = testCase{
		"required to",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			&qanpb.ObjectDetailsLabelsRequest{
				PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   nil,
				GroupBy:         "queryid",
				FilterBy:        "1D410B4BE5060972",
			},
		},
		nil,
		fmt.Errorf("period_start_to is required:%v", nil),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(tt.args.ctx, tt.args.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	request := &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
		PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
		GroupBy:         "",
		FilterBy:        "1D410B4BE5060972",
	}
	tt = testCase{
		"required group_by",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			request,
		},
		nil,
		fmt.Errorf("group_by is required:%v", request.GroupBy),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(tt.args.ctx, tt.args.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	request = &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
		PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
		GroupBy:         "queryid",
		FilterBy:        "",
	}
	tt = testCase{
		"required filter_by",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			request,
		},
		nil,
		fmt.Errorf("filter_by is required:%v", request.FilterBy),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(tt.args.ctx, tt.args.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	request = &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamp.Timestamp{Seconds: t2.Unix()},
		PeriodStartTo:   &timestamp.Timestamp{Seconds: t1.Unix()},
		GroupBy:         "queryid",
		FilterBy:        "1D410B4BE5060972",
	}
	tt = testCase{
		"invalid time range",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			request,
		},
		nil,
		fmt.Errorf("from time (%s) cannot be after to (%s)", request.PeriodStartFrom, request.PeriodStartTo),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(tt.args.ctx, tt.args.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	const selectError = `cannot select object details labels`

	request = &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
		PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
		GroupBy:         "invalid group",
		FilterBy:        "1D410B4BE5060972",
	}
	tt = testCase{
		"select error",
		fields{rm: rm, mm: mm},
		args{
			context.TODO(),
			request,
		},
		nil,
		fmt.Errorf("error in selecting object details labels:cannot select object details labels"),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(tt.args.ctx, tt.args.in)
		// errors start with same text.
		require.Regexp(t, "^error in selecting object details labels:cannot select object details labels.*", err.Error())

	})
}
