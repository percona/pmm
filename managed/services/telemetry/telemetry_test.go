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
	"io/fs"
	"log"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pmmv1 "github.com/percona-platform/saas/gen/telemetry/events/pmm"
	reporter "github.com/percona-platform/saas/gen/telemetry/reporter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/serverpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestRunTelemetryService(t *testing.T) {
	type fields struct {
		db                  func() *reform.DB
		l                   *logrus.Entry
		start               time.Time
		config              ServiceConfig
		pmmVersion          string
		os                  string
		sDistributionMethod serverpb.DistributionMethod
		tDistributionMethod pmmv1.DistributionMethod
		dus                 distributionUtilService
	}
	const (
		testSourceName = "VM"
		pmmVersion     = "2.29"
	)

	now := time.Now()
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.DebugLevel)
	logEntry := logrus.NewEntry(logger)

	expectedServerMetrics := []*pmmv1.ServerMetric_Metric{
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
	expectedReport := &reporter.ReportRequest{
		Metrics: []*pmmv1.ServerMetric{
			{
				DistributionMethod: pmmv1.DistributionMethod_AMI,
				Metrics:            expectedServerMetrics,
			},
		},
	}
	tests := []struct {
		name                string
		fields              fields
		mockTelemetrySender func() sender
		testTimeout         time.Duration
	}{
		{
			name:        "should send metrics only once during start",
			testTimeout: 2 * time.Second,
			fields: fields{
				db:         initMockDB(t, now, 1),
				start:      now,
				config:     getTestConfig(true, testSourceName, 10*time.Second),
				pmmVersion: pmmVersion,
				dus:        getDistributionUtilService(t, logEntry),
			},
			mockTelemetrySender: initMockTelemetrySender(t, expectedReport, 1),
		},
		{
			name:        "should send metrics only once and not send during start",
			testTimeout: 3 * time.Second,
			fields: fields{
				db:         initMockDB(t, now, 1),
				start:      now,
				config:     getTestConfig(false, testSourceName, 500*time.Millisecond+2*time.Second),
				pmmVersion: pmmVersion,
				dus:        getDistributionUtilService(t, logEntry),
			},
			mockTelemetrySender: initMockTelemetrySender(t, expectedReport, 1),
		},
		{
			name:        "should send metrics during start and once timer is ticked",
			testTimeout: 3 * time.Second,
			fields: fields{
				db:         initMockDB(t, now, 2),
				start:      now,
				config:     getTestConfig(true, testSourceName, 500*time.Millisecond+2*time.Second),
				pmmVersion: pmmVersion,
				dus:        getDistributionUtilService(t, logEntry),
			},
			mockTelemetrySender: initMockTelemetrySender(t, expectedReport, 2),
		},
	}

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), tt.testTimeout)
			defer cancel()

			serviceConfig := getServiceConfig()

			registry, err := NewDataSourceRegistry(serviceConfig, logEntry)
			assert.NoError(t, err)

			s := Service{
				db:                  tt.fields.db(),
				l:                   logEntry,
				start:               tt.fields.start,
				config:              tt.fields.config,
				dsRegistry:          registry,
				pmmVersion:          "",
				os:                  tt.fields.os,
				sDistributionMethod: 0,
				tDistributionMethod: 0,
				dus:                 tt.fields.dus,
				portalClient:        tt.mockTelemetrySender(),
			}

			var wg sync.WaitGroup

			wg.Add(1)
			go func() {
				s.Run(ctx)
				wg.Done()
			}()

			wg.Wait()
		})
	}
}

func getServiceConfig() ServiceConfig {
	serviceConfig := ServiceConfig{
		Enabled:      true,
		LoadDefaults: true,
		SaasHostname: "check.localhost",
		Reporting: ReportingConfig{
			SendOnStart:     true,
			Interval:        time.Second * 10,
			IntervalEnv:     "PERCONA_TEST_TELEMETRY_INTERVAL",
			RetryBackoff:    time.Second * 1,
			RetryBackoffEnv: "PERCONA_TEST_TELEMETRY_RETRY_BACKOFF",
			RetryCount:      2,
			SendTimeout:     time.Second * 10,
		},
		DataSources: struct {
			VM          *DataSourceVictoriaMetrics `yaml:"VM"`
			QanDBSelect *DSConfigQAN               `yaml:"QANDB_SELECT"` //nolint:tagliatelle
			PmmDBSelect *DSConfigPMMDB             `yaml:"PMMDB_SELECT"` //nolint:tagliatelle
		}{
			VM: &DataSourceVictoriaMetrics{
				Enabled: true,
				Timeout: time.Second * 2,
				Address: "http://localhost:9090/prometheus/",
			},
			QanDBSelect: &DSConfigQAN{
				Enabled: true,
				Timeout: time.Second * 2,
				DSN:     "tcp://localhost:9000?database=pmm&block_size=10000&pool_size=",
			},
			PmmDBSelect: &DSConfigPMMDB{
				Enabled:                true,
				Timeout:                time.Second * 2,
				UseSeparateCredentials: true,
				SeparateCredentials: struct {
					Username string `yaml:"username"`
					Password string `yaml:"password"`
				}{
					Username: "postgres",
					Password: "",
				},
				DSN: struct {
					Scheme string
					Host   string
					DB     string
					Params string
				}{
					Scheme: "postgres",
					Host:   "127.0.0.1:5432",
					DB:     "pmm-managed-dev",
					Params: "sslmode=disable",
				},
			},
		},
	}
	return serviceConfig
}

func getDistributionUtilService(t *testing.T, l *logrus.Entry) distributionUtilService {
	const (
		tmpDistributionFile = "/tmp/distribution"
		ami                 = "ami"
	)
	err := os.WriteFile(tmpDistributionFile, []byte(ami), fs.ModePerm)
	if err != nil {
		assert.Fail(t, "cannot write to file: ", err)
		return nil
	}
	dus := NewDistributionUtilServiceImpl(tmpDistributionFile, osInfoFilePath, l)
	return dus
}

func initMockTelemetrySender(t *testing.T, expetedReport *reporter.ReportRequest, timesCall int) func() sender {
	return func() sender {
		var mockTelemetrySender mockSender
		mockTelemetrySender.Test(t)
		mockTelemetrySender.On("SendTelemetry",
			mock.AnythingOfType(reflect.TypeOf(context.TODO()).Name()),
			mock.MatchedBy(func(report *reporter.ReportRequest) bool {
				return matchExpectedReport(report, expetedReport)
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

func matchExpectedReport(report *reporter.ReportRequest, expectedReport *reporter.ReportRequest) bool {
	return len(report.Metrics) == 1 &&
		expectedReport.Metrics[0].DistributionMethod.String() == "AMI"
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
				Query:   "pg_static{service_type=\"postgresql\"}",
				Summary: "Monitored PostgreSQL services version",
				Data: []ConfigData{
					{
						MetricName: "postgresql_version",
						Label:      "short_version",
					},
				},
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
			defer db.Close() //nolint:errcheck
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
