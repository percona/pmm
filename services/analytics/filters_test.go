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
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/jmoiron/sqlx"
	_ "github.com/kshvakov/clickhouse"
	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/api/qanpb"

	"github.com/percona/qan-api2/models"
)

func TestService_GetFilters(t *testing.T) {
	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	if !ok {
		dsn = "clickhouse://127.0.0.1:19000?database=pmm_test"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
	}

	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T00:01:00Z")
	var want qanpb.FiltersReply

	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	type args struct {
		ctx context.Context
		in  *qanpb.FiltersRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *qanpb.FiltersReply
		wantErr bool
	}{
		{
			"success",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.FiltersRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t1.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t2.Unix()},
				},
			},
			&want,
			false,
		},
		{
			"fail",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.FiltersRequest{
					PeriodStartFrom: &timestamp.Timestamp{Seconds: t2.Unix()},
					PeriodStartTo:   &timestamp.Timestamp{Seconds: t1.Unix()},
				},
			},
			nil,
			true,
		},
		{
			"fail",
			fields{rm: rm, mm: mm},
			args{
				context.TODO(),
				&qanpb.FiltersRequest{},
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
			got, err := s.Get(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				assert.Errorf(t, err, "Service.GetFilters() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want == nil {
				assert.Nil(t, got, "Service.GetFilters() return not nil")
				return
			}
			expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetFilters_"+tt.name+".json")
			marshaler := jsonpb.Marshaler{Indent: "	"}
			gotJSON, err := marshaler.MarshalToString(got)
			if err != nil {
				t.Errorf("cannot marshal:%v", err)
			}
			assert.JSONEq(t, string(expectedJSON), string(gotJSON))
		})
	}
}
