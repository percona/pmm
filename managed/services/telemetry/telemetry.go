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

// Package telemetry provides telemetry functionality.
package telemetry

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pmmv1 "github.com/percona/saas/gen/telemetry/events/pmm"
	telemetryv1 "github.com/percona/saas/gen/telemetry/generic"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	serverv1 "github.com/percona/pmm/api/server/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/platform"
)

const (
	sendChSize = 10
)

// Service reports telemetry.
type Service struct {
	db                  *reform.DB
	l                   *logrus.Entry
	portalClient        sender
	start               time.Time
	config              ServiceConfig
	dsRegistry          DataSourceLocator
	pmmVersion          string
	os                  string
	sDistributionMethod serverv1.DistributionMethod
	tDistributionMethod pmmv1.DistributionMethod
	sendCh              chan *telemetryv1.GenericReport
	dataSourcesMap      map[DataSourceName]DataSource

	extensions map[ExtensionType]Extension

	dus distributionUtilService
}

// check interfaces.
var (
	_ DataSourceLocator = (*Service)(nil)
)

// NewService creates a new service.
func NewService(db *reform.DB, portalClient *platform.Client, pmmVersion string,
	dus distributionUtilService, config ServiceConfig, extensions map[ExtensionType]Extension,
) (*Service, error) {
	if config.SaasHostname == "" {
		return nil, errors.New("empty host")
	}

	l := logrus.WithField("component", "telemetry")

	registry, err := NewDataSourceRegistry(config, l)
	if err != nil {
		return nil, err
	}
	s := &Service{
		db:           db,
		l:            l,
		portalClient: portalClient,
		pmmVersion:   pmmVersion,
		start:        time.Now(),
		config:       config,
		dsRegistry:   registry,
		dus:          dus,
		sendCh:       make(chan *telemetryv1.GenericReport, sendChSize),
		extensions:   extensions,
	}

	s.sDistributionMethod, s.tDistributionMethod, s.os = dus.GetDistributionMethodAndOS()
	s.dataSourcesMap = s.locateDataSources(config.telemetry)

	return s, nil
}

// LocateTelemetryDataSource retrieves DataSource by name.
func (s *Service) LocateTelemetryDataSource(name string) (DataSource, error) {
	return s.dsRegistry.LocateTelemetryDataSource(name)
}

// Run start sending telemetry to SaaS.
func (s *Service) Run(ctx context.Context) {
	if !s.config.Enabled {
		s.l.Warn("service is disabled, skip Run")
		return
	}

	ticker := time.NewTicker(s.config.Reporting.Interval)
	defer ticker.Stop()

	doSend := func() {
		var settings *models.Settings
		err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
			var e error
			if settings, e = models.GetSettings(tx); e != nil {
				return e
			}
			return nil
		})
		if err != nil {
			s.l.Debugf("Failed to retrieve settings: %s.", err)
			return
		}
		if !settings.IsTelemetryEnabled() {
			s.l.Info("Disabled via settings.")
			return
		}

		report := s.prepareReport(ctx)

		s.l.Debugf("\nTelemetry captured:\n%s\n", s.Format(report))

		if s.config.Reporting.Send {
			s.sendCh <- report
		} else {
			s.l.Info("Sending telemetry is disabled.")
		}
	}

	if s.config.Reporting.SendOnStart {
		s.l.Debug("Sending telemetry on start is enabled, in progress...")
		doSend()
	}

	go s.processSendCh(ctx)

	for {
		select {
		case <-ticker.C:
			doSend()
		case <-ctx.Done():
			return
		}
	}
}

// DistributionMethod returns PMM Server distribution method where this pmm-managed runs.
func (s *Service) DistributionMethod() serverv1.DistributionMethod {
	return s.sDistributionMethod
}

func (s *Service) processSendCh(ctx context.Context) {
	var reportsBufSync sync.Mutex
	var reportsBuf []*telemetryv1.GenericReport
	var sendCtx context.Context //nolint:contextcheck
	var cancel context.CancelFunc

	for {
		select {
		case report, ok := <-s.sendCh:
			if ok {
				s.l.Debug("Processing telemetry report.")
				if sendCtx != nil {
					cancel()
				}
				sendCtx, cancel = context.WithTimeout(ctx, s.config.Reporting.SendTimeout)

				reportsBufSync.Lock()
				reportsBuf = append(reportsBuf, report)
				reportsToSend := reportsBuf
				reportsBuf = []*telemetryv1.GenericReport{}
				reportsBufSync.Unlock()

				go func(ctx context.Context) {
					err := s.send(ctx, &telemetryv1.ReportRequest{
						Reports: reportsToSend,
					})
					if err != nil {
						s.l.Debugf("Telemetry info not sent, due to error: %s.", err)
						reportsBufSync.Lock()
						reportsBuf = append(reportsBuf, reportsToSend...)
						reportsBufSync.Unlock()
						return
					}

					s.l.Debug("Telemetry info sent.")
				}(sendCtx)
			}
		case <-ctx.Done():
			if cancel != nil {
				cancel()
			}
			return
		}
	}
}

func (s *Service) prepareReport(ctx context.Context) *telemetryv1.GenericReport {
	initializedDataSources := make(map[DataSourceName]DataSource)
	telemetryMetric, _ := s.makeMetric(ctx)
	var totalTime time.Duration

	// initialize datasources
	for sourceName, dataSource := range s.dataSourcesMap {
		if !dataSource.Enabled() {
			s.l.Warnf("Datasource %s is disabled, skipping initialization.", sourceName)
			continue
		}
		err := dataSource.Init(ctx)
		if err != nil {
			s.l.Warnf("Telemetry datasource %s init failed: %s", sourceName, err)
			continue
		}
		initializedDataSources[sourceName] = dataSource
	}

	for _, telemetry := range s.config.telemetry {
		if telemetry.Extension != "" {
			extension, ok := s.extensions[telemetry.Extension]
			if !ok {
				s.l.Errorf("telemetry extension [%s] is not supported", telemetry.Extension)
				continue
			}

			metrics, err := extension.FetchMetrics(ctx, telemetry)
			if err != nil {
				s.l.Debugf("failed while calling extension [%s]:%s", telemetry.Extension, err)
				continue
			}
			telemetryMetric.Metrics = append(telemetryMetric.Metrics, metrics...)

			continue
		}

		// locate DS in initialized state
		ds := initializedDataSources[DataSourceName(telemetry.Source)]
		if ds == nil {
			s.l.Debugf("Cannot find initialized telemetry datasource: %s", telemetry.Source)
			continue
		}
		if !ds.Enabled() {
			s.l.Debugf("Datasource %s is disabled", telemetry.Source)
			continue
		}

		// fetch metrics from DS
		metricFetchStartTime := time.Now()
		metrics, err := ds.FetchMetrics(ctx, telemetry)
		metricFetchTook := time.Since(metricFetchStartTime)
		s.l.Debugf("fetching [%s] took [%s]", telemetry.ID, metricFetchTook)
		totalTime += metricFetchTook
		if err != nil {
			s.l.Debugf("Failed to extract metric from datasource for [%s]:[%s]: %s", telemetry.Source, telemetry.ID, err)
			continue
		}

		if telemetry.Transform != nil {
			switch telemetry.Transform.Type {
			case JSONTransform:
				telemetryCopy := telemetry // G601: Implicit memory aliasing in for loop. (gosec)
				metrics, err = transformToJSON(&telemetryCopy, metrics)
				if err != nil {
					s.l.Debugf("Failed to transform to JSON: %s", err)
					continue
				}
			case StripValuesTransform:
				telemetryCopy := telemetry // G601: Implicit memory aliasing in for loop. (gosec)
				metrics, err = transformExportValues(&telemetryCopy, metrics)
				if err != nil {
					s.l.Debugf("failed to strip values: %s", err)
					continue
				}
			default:
				s.l.Errorf("unsupported transform type: %s", telemetry.Transform.Type)
			}
		}

		telemetryMetric.Metrics = append(telemetryMetric.Metrics, metrics...)
	}

	// datasources disposal
	for sourceName, dataSource := range initializedDataSources {
		err := dataSource.Dispose(ctx)
		if err != nil {
			s.l.Debugf("Disposing of %s datasource failed: %s", sourceName, err)
			continue
		}
	}

	telemetryMetric.Metrics = removeEmpty(telemetryMetric.Metrics)

	s.l.Debugf("Fetching all metrics took [%s]", totalTime)

	return telemetryMetric
}

func (s *Service) locateDataSources(telemetryConfig []Config) map[DataSourceName]DataSource {
	dataSources := make(map[DataSourceName]DataSource)
	for _, telemetry := range telemetryConfig {
		ds, err := s.LocateTelemetryDataSource(telemetry.Source)
		if err != nil {
			s.l.Debugf("Failed to lookup telemetry datasource for [%s]:[%s]", telemetry.Source, telemetry.ID)
			continue
		}
		dataSources[DataSourceName(telemetry.Source)] = ds
	}

	return dataSources
}

func (s *Service) makeMetric(ctx context.Context) (*telemetryv1.GenericReport, error) {
	var settings *models.Settings
	useServerID := false
	err := s.db.InTransaction(func(tx *reform.TX) error {
		var e error
		if settings, e = models.GetSettings(tx); e != nil {
			return e
		}

		if _, err := models.GetPerconaSSODetails(ctx, s.db.Querier); err == nil {
			useServerID = true
		} else if settings.Telemetry.UUID == "" {
			settings.Telemetry.UUID, e = generateUUID()
			if e != nil {
				return e
			}
			return models.SaveSettings(tx, settings)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var serverID string
	if useServerID {
		serverID = settings.PMMServerID
	} else {
		serverID = settings.Telemetry.UUID
	}

	_, distMethod, _ := s.dus.GetDistributionMethodAndOS()

	return &telemetryv1.GenericReport{
		Id:            uuid.New().String(),
		CreateTime:    timestamppb.New(time.Now()),
		InstanceId:    uuid.MustParse(serverID).String(),
		ProductFamily: telemetryv1.ProductFamily_PRODUCT_FAMILY_PMM,
		Metrics: []*telemetryv1.GenericReport_Metric{
			{Key: "PMMServerVersion", Value: s.pmmVersion},
			{Key: "UpDuration", Value: durationpb.New(time.Since(s.start)).String()},
			{Key: "DistributionMethod", Value: distMethod.String()},
		},
	}, nil
}

func generateUUID() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "can't generate UUID")
	}

	return uuid.String(), nil
}

func (s *Service) send(ctx context.Context, report *telemetryv1.ReportRequest) error {
	var err error
	var attempt int
	for {
		s.l.Debugf("Using %s as telemetry host.", s.config.SaasHostname)
		err = s.portalClient.SendTelemetry(ctx, report)
		attempt++
		s.l.Debugf("SendV2Request (attempt %d/%d) result: %s", attempt, s.config.Reporting.RetryCount, err)
		if err == nil {
			return nil
		}

		if attempt >= s.config.Reporting.RetryCount {
			s.l.Debug("Failed to send v2 event, will not retry (too many attempts).")
			return err
		}

		retryCtx, retryCancel := context.WithTimeout(ctx, s.config.Reporting.RetryBackoff)
		<-retryCtx.Done()
		retryCancel()

		if err = ctx.Err(); err != nil {
			s.l.Debugf("Will not retry sending v2 event: %s.", err)
			return err
		}
	}
}

// Format returns the formatted representation of the provided server metric.
func (s *Service) Format(report *telemetryv1.GenericReport) string {
	var builder strings.Builder
	for _, m := range report.Metrics {
		builder.WriteString(m.Key)
		builder.WriteString(": ")
		builder.WriteString(m.Value)
		builder.WriteString("\n")
	}

	return builder.String()
}

// GetSummaries returns the list of gathered telemetry.
func (s *Service) GetSummaries() []string {
	result := make([]string, 0, len(s.config.telemetry))
	for _, c := range s.config.telemetry {
		result = append(result, c.Summary)
	}
	return result
}
