// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	reporter "github.com/percona-platform/saas/gen/telemetry/reporter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/managed/models"
)

func TestService_Run(t *testing.T) {
	type fields struct {
		db                  func() *reform.DB
		l                   *logrus.Entry
		start               time.Time
		config              ServiceConfig
		dsRegistry          func() DataSourceLocator
		pmmVersion          string
		os                  string
		sDistributionMethod serverpb.DistributionMethod
		tDistributionMethod pmmv1.DistributionMethod
		dus                 func(l *logrus.Entry) DistributionUtilService
	}
	const testSourceName = "PMMDB_SELECT"
	const pmmVersion = "2.29"

	now := time.Now()

	expectedServerMetrics_Metrics := []*pmmv1.ServerMetric_Metric{
		{
			Key:   "key",
			Value: "value",
		},
		{
			Key:   "key2",
			Value: "value2",
		},
		{
			Key:   "key3",
			Value: "value3",
		},
	}
	expetedReport := &reporter.ReportRequest{
		Metrics: []*pmmv1.ServerMetric{
			{
				DistributionMethod: pmmv1.DistributionMethod_AMI,
				Metrics:            expectedServerMetrics_Metrics,
			},
		},
	}
	tests := []struct {
		name                string
		fields              fields
		mockTelemetrySender func() Sender
		testTimeout         time.Duration
	}{
		{
			name:        "should send metrics only once during start",
			testTimeout: 2 * time.Second,
			fields: fields{
				db:         initMockDB(t, now, 1),
				start:      now,
				config:     getTestConfig(true, testSourceName, 10*time.Second),
				dsRegistry: mockDataSourceLocator(t, [][]*pmmv1.ServerMetric_Metric{expectedServerMetrics_Metrics}, testSourceName, 1),
				pmmVersion: pmmVersion,
				dus: func(l *logrus.Entry) DistributionUtilService {
					var dusMock MockDistributionUtilService
					dusMock.Test(t)
					dusMock.On("getDistributionMethodAndOS", l).Return(serverpb.DistributionMethod_AMI, pmmv1.DistributionMethod_AMI, "ami").
						Times(1)
					return &dusMock
				},
			},
			mockTelemetrySender: initMockTelemetrySender(t, expetedReport, 1),
		},
		{
			name:        "should send metrics only once and not send during start",
			testTimeout: 3 * time.Second,
			fields: fields{
				db:         initMockDB(t, now, 1),
				start:      now,
				config:     getTestConfig(false, testSourceName, 500*time.Millisecond+2*time.Second),
				dsRegistry: mockDataSourceLocator(t, [][]*pmmv1.ServerMetric_Metric{expectedServerMetrics_Metrics}, testSourceName, 1),
				pmmVersion: pmmVersion,
				dus: func(l *logrus.Entry) DistributionUtilService {
					var dusMock MockDistributionUtilService
					dusMock.Test(t)
					dusMock.On("getDistributionMethodAndOS", l).Return(serverpb.DistributionMethod_AMI, pmmv1.DistributionMethod_AMI, "ami").
						Times(1)
					return &dusMock
				},
			},
			mockTelemetrySender: initMockTelemetrySender(t, expetedReport, 1),
		},
		{
			name:        "should send metrics during start and once timer is ticked",
			testTimeout: 3 * time.Second,
			fields: fields{
				db:     initMockDB(t, now, 2),
				start:  now,
				config: getTestConfig(true, testSourceName, 500*time.Millisecond+2*time.Second),
				dsRegistry: mockDataSourceLocator(t, [][]*pmmv1.ServerMetric_Metric{expectedServerMetrics_Metrics},
					testSourceName, 2),
				pmmVersion: pmmVersion,
				dus: func(l *logrus.Entry) DistributionUtilService {
					var dusMock MockDistributionUtilService
					dusMock.Test(t)
					dusMock.On("getDistributionMethodAndOS", l).
						Return(serverpb.DistributionMethod_AMI, pmmv1.DistributionMethod_AMI, "ami").
						Times(2)
					t.Cleanup(func() {
						dusMock.AssertExpectations(t)
					})
					return &dusMock
				},
			},
			mockTelemetrySender: initMockTelemetrySender(t, expetedReport, 2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.testTimeout)
			defer cancel()

			logger := logrus.StandardLogger()
			logger.SetLevel(logrus.DebugLevel)
			logEntry := logrus.NewEntry(logger)

			s := Service{
				db:                  tt.fields.db(),
				l:                   logEntry,
				start:               tt.fields.start,
				config:              tt.fields.config,
				dsRegistry:          tt.fields.dsRegistry(),
				pmmVersion:          "",
				os:                  tt.fields.os,
				sDistributionMethod: 0,
				tDistributionMethod: 0,
				dus:                 tt.fields.dus(logEntry),
				portalClient:        tt.mockTelemetrySender(),
			}

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				s.Run(ctx)
				wg.Done()
			}()

			wg.Wait()
			fmt.Println("Test is done")
		})
	}
}

func initMockTelemetrySender(t *testing.T, expetedReport *reporter.ReportRequest, timesCall int) func() Sender {
	return func() Sender {
		var mockTelemetrySender MockSender
		mockTelemetrySender.Test(t)
		mockTelemetrySender.On("SendTelemetry",
			mock.AnythingOfType(reflect.TypeOf(context.TODO()).Name()),
			mock.MatchedBy(func(report *reporter.ReportRequest) bool {
				return equalReports(report, expetedReport)
			}),
		).
			Return(nil).
			Times(timesCall)
		t.Cleanup(func() {
			mockTelemetrySender.AssertExpectations(t)
		})

		return &mockTelemetrySender
	}
}

func equalReports(report *reporter.ReportRequest, expectedReport *reporter.ReportRequest) bool {
	if len(expectedReport.Metrics) != len(report.Metrics) {
		return false
	}

	for i, m := range expectedReport.Metrics {
		gotMetric := report.Metrics[i]

		if m.PmmServerVersion != gotMetric.PmmServerVersion {
			return false
		}
		if m.DistributionMethod != gotMetric.DistributionMethod {
			return false
		}
		for j, mm := range m.Metrics {
			if mm.Key != gotMetric.Metrics[j].Key {
				return false
			}
			if mm.Value != gotMetric.Metrics[j].Value {
				return false
			}
		}
	}
	return true
}

//func mockServer(t *testing.T, expectedReport *reporter.ReportRequest) func() *httptest.Server {
//	return func() *httptest.Server {
//		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			fmt.Println("Call metric server")
//			b, err := ioutil.ReadAll(r.Body)
//			assert.NoError(t, err)
//
//			var reportRequest reporter.ReportRequest
//			err = protojson.Unmarshal(b, &reportRequest)
//			assert.NoError(t, err)
//
//			assert.True(t, len(expectedReport.Metrics) == len(reportRequest.Metrics))
//
//			for i, m := range expectedReport.Metrics {
//				gotMetric := reportRequest.Metrics[i]
//
//				assert.Equal(t, m.PmmServerVersion, gotMetric.PmmServerVersion)
//				assert.Equal(t, m.DistributionMethod, gotMetric.DistributionMethod)
//				for j, mm := range m.Metrics {
//					assert.Equal(t, mm.Key, gotMetric.Metrics[j].Key)
//					assert.Equal(t, mm.Value, gotMetric.Metrics[j].Value)
//				}
//			}
//
//			w.WriteHeader(http.StatusOK)
//		}))
//
//	}
//}

func mockDataSourceLocator(t *testing.T, metrics [][]*pmmv1.ServerMetric_Metric, testSourceName string, times int) func() DataSourceLocator {
	return func() DataSourceLocator {
		var mockSource MockDataSource
		mockSource.Mock.Test(t)
		mockSource.On("Enabled").Return(true)

		mockSource.On("FetchMetrics",
			mock.AnythingOfType(reflect.TypeOf(context.TODO()).Name()),
			mock.AnythingOfType(reflect.TypeOf(Config{}).Name())).
			Times(times).
			Return(metrics, nil)

		var mockDsl MockDataSourceLocator
		mockDsl.Test(t)
		mockDsl.On("LocateTelemetryDataSource", testSourceName).
			Times(times).
			Return(&mockSource, nil)
		t.Cleanup(func() {
			mockSource.AssertExpectations(t)
			mockDsl.AssertExpectations(t)
		})
		return &mockDsl
	}
}

func mockReturnErrorWhenFetchMetricsFromDataSource(t *testing.T, testSourceName string, times int) func() DataSourceLocator {
	return func() DataSourceLocator {
		var mockDsl MockDataSourceLocator
		mockDsl.Test(t)
		mockDsl.On("LocateTelemetryDataSource", testSourceName).
			Times(times).
			Return(nil, errors.New("cannot localte telemetry data source"))
		t.Cleanup(func() {
			mockDsl.AssertExpectations(t)
		})
		return &mockDsl
	}
}

func getTestConfig(sendOnStart bool, testSourceName string, reportingInterval time.Duration) ServiceConfig {
	return ServiceConfig{
		l:            nil,
		Enabled:      true,
		LoadDefaults: false,
		telemetry: []Config{
			{
				ID:      "1",
				Source:  testSourceName,
				Query:   "",
				Summary: "",
				Data:    nil,
			},
		},
		SaasHostname: "",
		DataSources: struct {
			VM          *DataSourceVictoriaMetrics `yaml:"VM"`
			QanDBSelect *DSConfigQAN               `yaml:"QANDB_SELECT"`
			PmmDBSelect *DSConfigPMMDB             `yaml:"PMMDB_SELECT"`
		}{},
		Reporting: ReportingConfig{
			SendOnStart:     sendOnStart,
			IntervalEnv:     "",
			Interval:        reportingInterval,
			RetryBackoffEnv: "",
			RetryBackoff:    0,
			SendTimeout:     0,
			RetryCount:      3,
		},
	}
}

func initMockDB(t *testing.T, now time.Time, callTimes int) func() *reform.DB {
	return func() *reform.DB {
		db, mock, err := sqlmock.New()
		assert.NoError(t, err)

		l := log.New(os.Stderr, "SQL: ", log.Flags())

		var s models.Settings
		b, err := json.Marshal(s)
		assert.NoError(t, err)

		for i := 0; i < callTimes; i++ {
			initGetSettingsSQLMock(mock, b, now)
		}

		// we make sure that all expectations were met
		t.Cleanup(func() {
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations for DB mock: %s", err)
			}
		})
		return reform.NewDB(db, postgresql.Dialect, reform.NewPrintfLogger(l.Printf))
	}
}

func initGetSettingsSQLMock(mock sqlmock.Sqlmock, b []byte, now time.Time) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT settings FROM settings").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(b))
	mock.ExpectQuery(`
						SELECT 
							"percona_sso_details"."pmm_managed_client_id",
							"percona_sso_details"."pmm_managed_client_secret", 
							"percona_sso_details"."grafana_client_id", 
							"percona_sso_details"."issuer_url", 
							"percona_sso_details"."scope", 
							"percona_sso_details"."access_token", 
							"percona_sso_details"."organization_id", 
							"percona_sso_details"."pmm_server_name",
							"percona_sso_details"."created_at" 
						FROM "percona_sso_details" `).
		WillReturnRows(
			sqlmock.NewRows(
				[]string{
					"pmm_managed_client_id",
					"pmm_managed_client_secret",
					"grafana_client_id",
					"issuer_url",
					"scope",
					"access_token",
					"organization_id",
					"pmm_server_name",
					"created_at",
				}).AddRow(
				"id",
				"secret",
				"client_id",
				"url",
				"scope",
				fmt.Sprintf(`
							{
								"token_type": "type",
								"expires_in": 10000,
								"expires_at": "%s",
								"access_token":"token",
								"scope": "scope"
							}`, now.Add(time.Hour).Format(time.RFC3339)),
				"id",
				"server_name",
				now.Add(-1*time.Hour)))
	mock.ExpectCommit()
}
