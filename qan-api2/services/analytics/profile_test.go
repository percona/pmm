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
	"log"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	jsonpb "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan-api2/models"
	"github.com/percona/pmm/qan-api2/utils/logger"
)

func setup() *sqlx.DB {
	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	if !ok {
		dsn = "clickhouse://127.0.0.1:19000/pmm_test"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
	}

	return db
}

func makeContext(t *testing.T) context.Context {
	t.Helper()
	ctx := metadata.NewIncomingContext(context.TODO(), metadata.Pairs("x-request-id", "test"))
	return logger.SetEntry(ctx, logrus.WithField("test", t.Name()))
}

func getExpectedJSON(t *testing.T, got proto.Message, filename string) []byte {
	t.Helper()
	if os.Getenv("REFRESH_TEST_DATA") != "" {
		marshaler := jsonpb.MarshalOptions{
			Indent: "\t",
		}
		json, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		err = os.WriteFile(filename, json, 0o644) //nolint:gosec
		if err != nil {
			t.Errorf("cannot write:%v", err)
		}
	}
	data, err := os.ReadFile(filename) //nolint:gosec
	if err != nil {
		t.Errorf("cannot read data from file:%v", err)
	}
	return data
}

func TestService_GetReport(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	var want qanpb.GetReportResponse
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	tests := []struct {
		name    string
		fields  fields
		in      *qanpb.GetReportRequest
		want    *qanpb.GetReportResponse
		wantErr bool
	}{
		{
			"success",
			fields{rm: rm, mm: mm},
			&qanpb.GetReportRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				Columns:         []string{"query_time", "lock_time", "sort_scan"},
				OrderBy:         "query_time",
				Offset:          0,
				Limit:           10,
			},
			&want,
			false,
		},
		{
			"load without query_time",
			fields{rm: rm, mm: mm},
			&qanpb.GetReportRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				GroupBy:         "queryid",
				Columns:         []string{"load", "lock_time", "sort_scan"},
				OrderBy:         "-load",
				Offset:          0,
				Limit:           10,
			},
			&want,
			false,
		},
		{
			"wrong_time_range",
			fields{rm: rm, mm: mm},
			&qanpb.GetReportRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t2.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t1.Unix()},
			},
			nil,
			true,
		},
		{
			"empty_fail",
			fields{rm: rm, mm: mm},
			&qanpb.GetReportRequest{},
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

			got, err := s.GetReport(makeContext(t), tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil {
				assert.Nil(t, got, "Service.GetReport() return not nil")
				return
			}
			expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_"+tt.name+".json")
			marshaler := jsonpb.MarshalOptions{Indent: "\t"}
			gotJSON, err := marshaler.Marshal(got)
			if err != nil {
				t.Errorf("cannot marshal:%v", err)
			}
			assert.JSONEq(t, string(expectedJSON), string(gotJSON))
		})
	}
}

func TestService_GetReport_Mix(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	var want qanpb.GetReportResponse
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	test := struct {
		fields  fields
		in      *qanpb.GetReportRequest
		want    *qanpb.GetReportResponse
		wantErr bool
	}{
		fields{rm: rm, mm: mm},
		&qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns:         []string{"query_time", "lock_time", "sort_scan"},
			OrderBy:         "-query_time",
			Offset:          10,
			Limit:           10,
			Labels: []*qanpb.ReportMapFieldEntry{
				{
					Key:   "label1",
					Value: []string{"value1", "value2"},
				},
				{
					Key:   "service_name",
					Value: []string{"server1", "server2", "server3", "server4", "server5", "server6", "server7"},
				},
			},
		},
		&want,
		false,
	}
	t.Run("reverse_order", func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}

		got, err := s.GetReport(makeContext(t), test.in)
		if (err != nil) != test.wantErr {
			t.Errorf("Service.GetReport() error = %v, wantErr %v", err, test.wantErr)
			return
		}
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Mix_reverce_order.json")
		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("correct_load", func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}

		got, err := s.GetReport(makeContext(t), test.in)
		if (err != nil) != test.wantErr {
			t.Errorf("Service.GetReport() error = %v, wantErr %v", err, test.wantErr)
			return
		}
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Mix_correct_load.json")
		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("correct_latency", func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}

		got, err := s.GetReport(makeContext(t), test.in)
		if (err != nil) != test.wantErr {
			t.Errorf("Service.GetReport() error = %v, wantErr %v", err, test.wantErr)
			return
		}
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Mix_correct_latency.json")
		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("no error on limit is 0", func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}

		test.in.Limit = 0
		_, err := s.GetReport(makeContext(t), test.in)
		if err != nil {
			t.Errorf("Service.GetReport() error = %v, wantErr %v", err, test.wantErr)
			return
		}
	})

	t.Run("Limit is 0", func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}

		test.in.GroupBy = "unknown dimension"
		expectedErr := fmt.Errorf("unknown group dimension: %s", "unknown dimension")
		_, err := s.GetReport(makeContext(t), test.in)
		if err.Error() != expectedErr.Error() {
			t.Errorf("Service.GetReport() unexpected error = %v, wantErr %v", err, expectedErr)
			return
		}
	})
}

func TestService_GetReport_Groups(t *testing.T) {
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

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Groups_group_by_queryid.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("group_by_service_name", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "service_name",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Groups_group_by_service_name.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("group_by_database", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "database",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Groups_group_by_database.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("group_by_schema", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "schema",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Groups_group_by_schema.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("group_by_username", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "username",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Groups_group_by_username.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("group_by_client_host", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "client_host",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Groups_group_by_client_host.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})
}

func TestService_GetReport_AllLabels(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")
	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}

	genDimensionvalues := func(dimKey string, amount int) []string {
		arr := []string{}
		for i := range amount {
			arr = append(arr, fmt.Sprintf("%s%d", dimKey, i))
		}
		return arr
	}
	test := struct {
		name    string
		fields  fields
		in      *qanpb.GetReportRequest
		wantErr bool
	}{
		"",
		fields{rm: rm, mm: mm},
		&qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns:         []string{"query_time", "lock_time", "sort_scan"},
			OrderBy:         "-query_time",
			Offset:          10,
			Limit:           10,
			Labels: []*qanpb.ReportMapFieldEntry{
				{
					Key:   "label1",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label2",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label3",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label4",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label5",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label6",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label7",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label8",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "label9",
					Value: genDimensionvalues("value", 100),
				},
				{
					Key:   "service_name",
					Value: genDimensionvalues("server", 10),
				},
				{
					Key:   "database",
					Value: []string{},
				},
				{
					Key:   "schema",
					Value: genDimensionvalues("schema", 100),
				},
				{
					Key:   "username",
					Value: genDimensionvalues("user", 100),
				},
				{
					Key:   "client_host",
					Value: genDimensionvalues("10.11.12.", 100),
				},
			},
		},
		false,
	}
	t.Run("Use all label keys", func(t *testing.T) {
		s := &Service{
			rm: test.fields.rm,
			mm: test.fields.mm,
		}

		got, err := s.GetReport(makeContext(t), test.in)
		if (err != nil) != test.wantErr {
			t.Errorf("Service.GetReport() error = %v, wantErr %v", err, test.wantErr)
			return
		}

		expectedRows := 1
		gotRows := len(got.Rows)
		if gotRows != expectedRows {
			t.Errorf("Got rows count: %d - expected, %d", gotRows, expectedRows)
		}
	})
}

func TestService_GetReport_Sparklines(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T01:00:00Z")

	t.Run("sparklines_60_points", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_sparklines_60_points.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t3, _ := time.Parse(time.RFC3339, "2019-01-01T01:30:00Z")
	t.Run("sparklines_90_points", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t3.Unix()},
			GroupBy:         "queryid",
			Columns: []string{
				"query_time", "lock_time", "sort_scan", "rows_sent", "rows_examined", "rows_affected",
				"rows_read", "merge_passes", "innodb_io_r_ops", "innodb_io_r_bytes",
				"innodb_io_r_wait", "innodb_rec_lock_wait", "innodb_queue_wait",
				"innodb_pages_distinct", "query_length", "bytes_sent", "tmp_tables",
				"tmp_disk_tables", "tmp_table_sizes", "qc_hit", "full_scan", "full_join",
				"tmp_table", "tmp_table_on_disk", "filesort", "filesort_on_disk",
				"select_full_range_join", "select_range", "select_range_check",
				"sort_range", "sort_rows", "sort_scan", "no_index_used", "no_good_index_used",
				"no_good_index_used", "docs_returned", "response_length", "docs_scanned",
			},
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_sparklines_90_points.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})
}

func TestService_GetReport_Search(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")

	t.Run("search_queryid", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns: []string{
				"query_time",
			},
			Search:  "F6760F2D2E",
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Search_search_queryid.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("search_fingerprint", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns: []string{
				"query_time",
			},
			Search:  "SeLeCt",
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Search_search_fingerprint.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})

	t.Run("search_service_name", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "service_name",
			Columns: []string{
				"query_time",
			},
			Search:  "server",
			OrderBy: "-query_time",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetReport_Search_search_service_name.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})
}

func TestServiceGetReportSpecialMetrics(t *testing.T) {
	db := setup()
	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T10:00:00Z")

	t.Run("num_queries_with_errors", func(t *testing.T) {
		s := &Service{
			rm: rm,
			mm: mm,
		}

		in := qanpb.GetReportRequest{
			PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
			PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			GroupBy:         "queryid",
			Columns: []string{
				"num_queries_with_errors", "num_queries_with_warnings", "num_queries", "load",
			},
			OrderBy: "-num_queries_with_errors",
			Offset:  0,
			Limit:   10,
		}

		got, err := s.GetReport(makeContext(t), &in)
		assert.NoError(t, err, "Unexpected error in Service.GetReport()")
		expectedJSON := getExpectedJSON(t, got, "../../test_data/TestServiceGetReportSpecialMetrics_num_queries_with_errors.json")

		marshaler := jsonpb.MarshalOptions{Indent: "\t"}
		gotJSON, err := marshaler.Marshal(got)
		if err != nil {
			t.Errorf("cannot marshal:%v", err)
		}
		assert.JSONEq(t, string(expectedJSON), string(gotJSON))
	})
}
