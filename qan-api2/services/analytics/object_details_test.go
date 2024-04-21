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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	qanpb "github.com/percona/pmm/api/qanpb"
	"github.com/percona/pmm/qan-api2/models"
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
	tests := []struct {
		name    string
		fields  fields
		in      *qanpb.QueryExampleRequest
		want    *qanpb.QueryExampleReply
		wantErr bool
	}{
		{
			"no_period_start_from",
			fields{rm: rm, mm: mm},
			&qanpb.QueryExampleRequest{
				PeriodStartTo: &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:       "queryid",
				FilterBy:      "B305F6354FA21F2A",
				Limit:         5,
			},
			nil,
			true,
		},
		{
			"no_period_start_to",
			fields{rm: rm, mm: mm},
			&qanpb.QueryExampleRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "B305F6354FA21F2A",
				Limit:           5,
			},
			nil,
			true,
		},
		{
			"no_group",
			fields{rm: rm, mm: mm},
			&qanpb.QueryExampleRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				FilterBy:        "B305F6354FA21F2A",
				Limit:           5,
			},
			&want,
			false,
		},
		{
			"no_limit",
			fields{rm: rm, mm: mm},
			&qanpb.QueryExampleRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "B305F6354FA21F2A",
			},
			&want,
			false,
		},
		{
			"invalid_group_name",
			fields{rm: rm, mm: mm},
			&qanpb.QueryExampleRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "invalid_group_name",
				FilterBy:        "B305F6354FA21F2A",
			},
			nil,
			true,
		},
		{
			"not_found",
			fields{rm: rm, mm: mm},
			&qanpb.QueryExampleRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "unexist",
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
			got, err := s.GetQueryExample(context.TODO(), tt.in)
			if (err != nil) != tt.wantErr {
				assert.Errorf(t, err, "Service.GetQueryExample() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want == nil {
				assert.Nil(t, got, "Service.GetQueryExample() returned not nil")
				return
			}

			expectedJSON := getExpectedJSON(t, got, "../../test_data/GetQueryExample_"+tt.name+".json")
			marshaler := protojson.MarshalOptions{Indent: "\t"}
			gotJSON, err := marshaler.Marshal(got)
			if err != nil {
				t.Errorf("cannot marshal:%v", err)
			}
			require.JSONEq(t, string(expectedJSON), string(gotJSON))
		})
	}
}

func TestService_GetMetricsError(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")

	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	tests := []struct {
		name    string
		fields  fields
		in      *qanpb.MetricsRequest
		want    *qanpb.MetricsReply
		wantErr bool
	}{
		{
			"not_found",
			fields{rm: rm, mm: mm},
			&qanpb.MetricsRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "unexist",
			},
			nil,
			false,
		},
		{
			"no_period_start_from",
			fields{rm: rm, mm: mm},
			&qanpb.MetricsRequest{
				PeriodStartTo: &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:       "queryid",
				FilterBy:      "B305F6354FA21F2A",
			},
			nil,
			true,
		},
		{
			"no_period_start_to",
			fields{rm: rm, mm: mm},
			&qanpb.MetricsRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				GroupBy:         "queryid",
				FilterBy:        "B305F6354FA21F2A",
			},
			nil,
			true,
		},
		{
			"invalid_group_name",
			fields{rm: rm, mm: mm},
			&qanpb.MetricsRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "no_group_name",
				FilterBy:        "B305F6354FA21F2A",
			},
			nil,
			true,
		},
		{
			"not_found_labels",
			fields{rm: rm, mm: mm},
			&qanpb.MetricsRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "no_group_name",
				FilterBy:        "B305F6354FA21F2A",
				Labels: []*qanpb.MapFieldEntry{
					{
						Key:   "label1",
						Value: []string{"value1", "value2"},
					},
					{
						Key:   "server",
						Value: []string{"server1", "server2", "server3", "server4", "server5", "server6", "server7"},
					},
					{
						Key:   "client_host",
						Value: []string{"localhost"},
					},
					{
						Key:   "username",
						Value: []string{"john"},
					},
					{
						Key:   "schema",
						Value: []string{"my_schema"},
					},
					{
						Key:   "database",
						Value: []string{"test_database"},
					},
					{
						Key:   "queryid",
						Value: []string{"some_query_id"},
					},
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
			_, err := s.GetMetrics(context.TODO(), tt.in)
			if (err != nil) != tt.wantErr {
				assert.Errorf(t, err, "Service.GetMetrics() error = %v, wantErr %v", err, tt.wantErr)
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

	t.Run("group_by_queryid", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}
		in := &qanpb.MetricsRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			FilterBy:        "B305F6354FA21F2A",
		}
		got, err := s.GetMetrics(context.TODO(), in)
		assert.NoError(t, err, "Unexpected error in Service.GetMetrics()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/GetMetrics_group_by_queryid.json")

		marshaler := protojson.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		require.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t3, _ := time.Parse(time.RFC3339, "2019-01-01T01:30:00Z")
	t.Run("sparklines_90_points", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}
		in := &qanpb.MetricsRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t3.Unix()},
			GroupBy:         "queryid",
			FilterBy:        "B305F6354FA21F2A",
		}
		got, err := s.GetMetrics(context.TODO(), in)
		assert.NoError(t, err, "Unexpected error in Service.GetMetrics()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/GetMetrics_sparklines_90_points.json")

		marshaler := protojson.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		require.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("total", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}
		in := &qanpb.MetricsRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			FilterBy:        "", // Empty filter get all queries.
			Totals:          true,
		}
		got, err := s.GetMetrics(context.TODO(), in)
		assert.NoError(t, err, "Unexpected error in Service.GetMetrics()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/GetMetrics_total.json")

		marshaler := protojson.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})
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
	type testCase struct {
		name    string
		fields  fields
		in      *qanpb.ObjectDetailsLabelsRequest
		want    *qanpb.ObjectDetailsLabelsReply
		wantErr error
	}

	tt := testCase{
		"success",
		fields{rm: rm, mm: mm},
		&qanpb.ObjectDetailsLabelsRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			FilterBy:        "1D410B4BE5060972",
		},
		&want,
		nil,
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		got, err := s.GetLabels(context.TODO(), tt.in)
		require.Equal(t, tt.wantErr, err)
		expectedJSON := getExpectedJSON(t, got, "../../test_data/GetLabels"+tt.name+".json")

		marshaler := protojson.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	tt = testCase{
		"required from",
		fields{rm: rm, mm: mm},
		&qanpb.ObjectDetailsLabelsRequest{
			PeriodStartFrom: nil,
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			FilterBy:        "1D410B4BE5060972",
		},
		nil,
		fmt.Errorf("period_start_from is required: %v", nil),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(context.TODO(), tt.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	tt = testCase{
		"required to",
		fields{rm: rm, mm: mm},
		&qanpb.ObjectDetailsLabelsRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   nil,
			GroupBy:         "queryid",
			FilterBy:        "1D410B4BE5060972",
		},
		nil,
		fmt.Errorf("period_start_to is required: %v", nil),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(context.TODO(), tt.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	request := &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
		PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
		GroupBy:         "",
		FilterBy:        "1D410B4BE5060972",
	}
	tt = testCase{
		"required group_by",
		fields{rm: rm, mm: mm},
		request,
		nil,
		fmt.Errorf("group_by is required if filter_by is not empty %v = %v", request.GroupBy, request.FilterBy),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(context.TODO(), tt.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	request = &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
		PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
		GroupBy:         "queryid",
		FilterBy:        "",
	}
	tt = testCase{
		"required_filter_by",
		fields{rm: rm, mm: mm},
		request,
		nil,
		nil,
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		got, err := s.GetLabels(context.TODO(), tt.in)
		require.Equal(t, tt.wantErr, err)
		expectedJSON := getExpectedJSON(t, got, "../../test_data/GetLabels_"+tt.name+".json")

		marshaler := protojson.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	request = &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamppb.Timestamp{Seconds: t2.Unix()},
		PeriodStartTo:   &timestamppb.Timestamp{Seconds: t1.Unix()},
		GroupBy:         "queryid",
		FilterBy:        "1D410B4BE5060972",
	}
	tt = testCase{
		"invalid time range",
		fields{rm: rm, mm: mm},
		request,
		nil,
		fmt.Errorf("from time (%s) cannot be after to (%s)", request.PeriodStartFrom, request.PeriodStartTo),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(context.TODO(), tt.in)
		require.EqualError(t, err, tt.wantErr.Error())
	})

	request = &qanpb.ObjectDetailsLabelsRequest{
		PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
		PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
		GroupBy:         "invalid group",
		FilterBy:        "1D410B4BE5060972",
	}
	tt = testCase{
		"select error",
		fields{rm: rm, mm: mm},
		request,
		nil,
		fmt.Errorf("error in selecting object details labels:cannot select object details labels"),
	}

	t.Run(tt.name, func(t *testing.T) {
		s := &Service{
			rm: tt.fields.rm,
			mm: tt.fields.mm,
		}
		_, err := s.GetLabels(context.TODO(), tt.in)
		// errors start with same text.
		require.Regexp(t, "^error in selecting object details labels:cannot select object details labels.*", err.Error())
	})
}
