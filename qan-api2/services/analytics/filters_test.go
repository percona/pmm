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
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2" // register database/sql driver
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	qanpb "github.com/percona/pmm/api/qan/v1"
	"github.com/percona/pmm/qan-api2/models"
)

type expected struct {
	Labels map[string]listLabels `json:"labels,omitempty"`
}

type listLabels struct {
	Name []testValuesUnmarshal `json:"name,omitempty"`
}
type testValues struct {
	MainMetricPercent float32 `json:"mainMetricPercent,omitempty"`
	MainMetricPerSec  float32 `json:"mainMetricPerSec,omitempty"`
}

type testValuesUnmarshal struct {
	Value             string `json:"value,omitempty"`
	MainMetricPercent any    `json:"mainMetricPercent,omitempty"`
	MainMetricPerSec  any    `json:"mainMetricPerSec,omitempty"`
}

func TestService_GetFilters(t *testing.T) {
	dsn, ok := os.LookupEnv("QANAPI_DSN_TEST")
	if !ok {
		dsn = "clickhouse://127.0.0.1:19000/pmm_test"
	}
	db, err := sqlx.Connect("clickhouse", dsn)
	if err != nil {
		log.Fatal("Connection: ", err)
	}

	rm := models.NewReporter(db)
	mm := models.NewMetrics(db)
	t1, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	t2, _ := time.Parse(time.RFC3339, "2019-01-01T00:01:00Z")
	var want qanpb.GetFilteredMetricsNamesResponse

	type fields struct {
		rm models.Reporter
		mm models.Metrics
	}
	tests := []struct {
		name    string
		fields  fields
		in      *qanpb.GetFilteredMetricsNamesRequest
		want    *qanpb.GetFilteredMetricsNamesResponse
		wantErr bool
	}{
		{
			"success",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
			},
			&want,
			false,
		},
		{
			"success_with_dimensions_username",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				Labels: []*qanpb.MapFieldEntry{
					{Key: "username", Value: []string{"user1", "user2"}},
				},
			},
			&want,
			false,
		},
		{
			"success_with_dimensions_client_host_schema_service_name",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				Labels: []*qanpb.MapFieldEntry{
					{Key: "client_host", Value: []string{"10.11.12.1", "10.11.12.2", "10.11.12.3", "10.11.12.4", "10.11.12.5", "10.11.12.6", "10.11.12.7", "10.11.12.8", "10.11.12.9", "10.11.12.10", "10.11.12.11", "10.11.12.12", "10.11.12.13"}},
					{Key: "schema", Value: []string{"schema65", "schema6", "schema42", "schema76", "schema90", "schema39", "schema1", "schema17", "schema79", "schema10"}},
					{Key: "service_name", Value: []string{"server5", "server8", "server6", "server3", "server4", "server2", "server1"}},
				},
			},
			&want,
			false,
		},
		{
			"success_with_dimensions_multiple",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				Labels: []*qanpb.MapFieldEntry{
					{Key: "container_id", Value: []string{"container_id"}},
					{Key: "container_name", Value: []string{"container_name1"}},
					{Key: "machine_id", Value: []string{"machine_id1"}},
					{Key: "node_type", Value: []string{"node_type1"}},
					{Key: "node_name", Value: []string{"node_name1"}},
					{Key: "node_id", Value: []string{"node_id1"}},
					{Key: "node_model", Value: []string{"node_model1"}},
					{Key: "region", Value: []string{"region1"}},
					{Key: "az", Value: []string{"az1"}},
					{Key: "environment", Value: []string{"environment1"}},
					{Key: "service_id", Value: []string{"service_id1"}},
					{Key: "service_type", Value: []string{"service_type1"}},
					{Key: "cmd_type", Value: []string{"1"}},
					{Key: "top_queryid", Value: []string{"top_queryid1"}},
					{Key: "application_name", Value: []string{"psql"}},
					{Key: "planid", Value: []string{"planid1"}},
					{Key: "plan_summary", Value: []string{"COLLSCAN", "IXSCAN"}},
				},
			},
			&want,
			false,
		},
		{
			"success_with_labels",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t1.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t2.Unix()},
				Labels: []*qanpb.MapFieldEntry{
					{Key: "label0", Value: []string{"value1"}},
				},
			},
			&want,
			false,
		},
		{
			"fail",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{
				PeriodStartFrom: &timestamppb.Timestamp{Seconds: t2.Unix()},
				PeriodStartTo:   &timestamppb.Timestamp{Seconds: t1.Unix()},
			},
			nil,
			true,
		},
		{
			"fail",
			fields{rm: rm, mm: mm},
			&qanpb.GetFilteredMetricsNamesRequest{},
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
			got, err := s.GetFilteredMetricsNames(context.TODO(), tt.in)
			if (err != nil) != tt.wantErr {
				assert.Errorf(t, err, "Service.GetFilters() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want == nil {
				assert.Nil(t, got, "Service.GetFilters() return not nil")
				return
			}

			valuesGot := make(map[string]map[string]testValues)
			for k, l := range got.Labels {
				if _, ok := valuesGot[k]; !ok {
					valuesGot[k] = make(map[string]testValues)
				}
				for _, v := range l.Name {
					valuesGot[k][v.Value] = testValues{
						MainMetricPercent: v.MainMetricPercent,
						MainMetricPerSec:  v.MainMetricPerSec,
					}
				}
			}

			expectedJSON := getExpectedJSON(t, got, "../../test_data/TestService_GetFilters_"+tt.name+".json")
			var unmarshal expected
			err = json.Unmarshal(expectedJSON, &unmarshal)
			if err != nil {
				t.Errorf("cannot unmarshal:%v", err)
			}

			valuesExpected := make(map[string]map[string]testValues)
			for k, l := range unmarshal.Labels {
				if _, ok := valuesExpected[k]; !ok {
					valuesExpected[k] = make(map[string]testValues)
				}
				for _, v := range l.Name {
					percent := float32(0)
					if p, ok := v.MainMetricPercent.(float64); ok {
						percent = float32(p)
					}

					perSec := float32(0)
					if p, ok := v.MainMetricPerSec.(float64); ok {
						perSec = float32(p)
					}

					valuesExpected[k][v.Value] = testValues{
						MainMetricPercent: percent,
						MainMetricPerSec:  perSec,
					}
				}
			}

			assert.ObjectsAreEqual(valuesExpected, valuesGot)
		})
	}
}
