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

package telemetry

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
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

const (
	envPGHostPort = "TEST_PG_HOST_PORT"
	envQanDSN     = "TEST_QAN_DSN"
	envVMDSN      = "TEST_VM_DSN"
)

func TestRunTelemetryService(t *testing.T) {
	t.Parallel()
	pgHostPort := "127.0.0.1:5432"
	pgHostPortFromEnv, ok := os.LookupEnv(envPGHostPort)
	if ok {
		pgHostPort = pgHostPortFromEnv
	}
	qanDSN := "tcp://localhost:9000?database=pmm"
	qanDSNFromEnv, ok := os.LookupEnv(envQanDSN)
	if ok {
		qanDSN = qanDSNFromEnv
	}
	vmDSN := "http://localhost:9090/prometheus/"
	vmDSNFromEnv, ok := os.LookupEnv(envVMDSN)
	if ok {
		vmDSN = vmDSNFromEnv
	}

	type fields struct {
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
				start:      now,
				config:     getTestConfig(true, testSourceName, 500*time.Millisecond+2*time.Second),
				pmmVersion: pmmVersion,
				dus:        getDistributionUtilService(t, logEntry),
			},
			mockTelemetrySender: initMockTelemetrySender(t, expectedReport, 2),
		},
	}

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), tt.testTimeout)
			defer cancel()

			serviceConfig := getServiceConfig(pgHostPort, qanDSN, vmDSN)

			registry, err := NewDataSourceRegistry(serviceConfig, logEntry)
			assert.NoError(t, err)

			s := Service{
				db:                  db,
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
				sendCh:              make(chan *pmmv1.ServerMetric, sendChSize),
			}
			s.Run(ctx)
		})
	}
}

func getServiceConfig(pgPortHost string, qanDSN string, vmDSN string) ServiceConfig {
	serviceConfig := ServiceConfig{
		Enabled:      true,
		SaasHostname: "check.localhost",
		Reporting: ReportingConfig{
			SendOnStart:  true,
			Interval:     time.Second * 10,
			RetryBackoff: time.Second * 1,
			RetryCount:   2,
			SendTimeout:  time.Second * 10,
		},
		DataSources: DataSources{
			VM: &DSConfigVM{
				Enabled: true,
				Timeout: time.Second * 2,
				Address: vmDSN,
			},
			QanDBSelect: &DSConfigQAN{
				Enabled: true,
				Timeout: time.Second * 2,
				DSN:     qanDSN,
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
					Host:   pgPortHost,
					DB:     "pmm-managed-dev",
					Params: "sslmode=disable",
				},
			},
			GrafanaDBSelect: &DSConfigGrafanaDB{
				Enabled:                true,
				Timeout:                time.Second * 2,
				UseSeparateCredentials: true,
				SeparateCredentials: struct {
					Username string `yaml:"username"`
					Password string `yaml:"password"`
				}{
					Username: "grafana",
					Password: "grafana",
				},
				DSN: struct {
					Scheme string
					Host   string
					DB     string
					Params string
				}{
					Scheme: "postgres",
					Host:   pgPortHost,
					DB:     "grafana",
					Params: "sslmode=disable",
				},
			},
			EnvVars: &DSConfigEnvVars{
				Enabled: true,
			},
		},
	}
	return serviceConfig
}

func getDistributionUtilService(t *testing.T, l *logrus.Entry) *distributionUtilServiceImpl {
	t.Helper()
	const (
		tmpDistributionFile = "/tmp/distribution"
		ami                 = "ami"
	)
	err := os.WriteFile(tmpDistributionFile, []byte(ami), fs.ModePerm)
	if err != nil {
		assert.Fail(t, "cannot write to file: ", err)
		return nil
	}
	dus := newDistributionUtilServiceImpl(tmpDistributionFile, osInfoFilePath, l)
	return dus
}

func initMockTelemetrySender(t *testing.T, expectedReport *reporter.ReportRequest, timesCall int) func() sender {
	t.Helper()
	return func() sender {
		var mockTelemetrySender mockSender
		mockTelemetrySender.Test(t)
		mockTelemetrySender.On("SendTelemetry",
			mock.Anything,
			mock.MatchedBy(func(report *reporter.ReportRequest) bool {
				return matchExpectedReport(report, expectedReport)
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
		l:       nil,
		Enabled: true,
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
		DataSources:  DataSources{},
		Reporting: ReportingConfig{
			Send:         true,
			SendOnStart:  sendOnStart,
			Interval:     reportingInterval,
			RetryBackoff: 0,
			SendTimeout:  0,
			RetryCount:   3,
		},
	}
}
