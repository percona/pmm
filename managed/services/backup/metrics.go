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

package backup

import (
	"context"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

const (
	requestTimeout              = 3 * time.Second
	artifactExists      float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "backups"
)

// MetricsCollector is responsible for collecting metrics related to backup.
type MetricsCollector struct {
	db *reform.DB
	l  *logrus.Entry

	mArtifactsDesc *prom.Desc
}

// NewMetricsCollector creates a new instance of MetricsCollector.
func NewMetricsCollector(db *reform.DB) *MetricsCollector {
	return &MetricsCollector{
		db: db,
		l:  logrus.WithField("component", "backups/metrics"),
		mArtifactsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "artifacts"),
			"Artifacts",
			[]string{
				"artifact_id", "artifact_name", "artifact_vendor", "service_id", "service_name",
				"type", "db_version", "data_model", "mode", "status",
			},
			nil),
	}
}

// Describe sends the metrics descriptions to the provided channel.
func (c *MetricsCollector) Describe(ch chan<- *prom.Desc) {
	ch <- c.mArtifactsDesc
}

// Collect sends the collected metrics to the provided channel.
func (c *MetricsCollector) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelCtx()

	var artifacts []*models.Artifact
	var services map[string]*models.Service
	errTx := c.db.InTransactionContext(ctx, nil, func(t *reform.TX) error {
		var err error
		artifacts, err = models.FindArtifacts(t.Querier, models.ArtifactFilters{})
		if err != nil {
			return errors.WithStack(err)
		}

		serviceIDs := make([]string, 0, len(artifacts))
		for _, artifact := range artifacts {
			serviceIDs = append(serviceIDs, artifact.ServiceID)
		}

		services, err = models.FindServicesByIDs(t.Querier, serviceIDs)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if errTx != nil {
		c.l.Warnf("Failed to get artifacts: %v", errTx)
		return
	}

	for _, artifact := range artifacts {
		var serviceName string
		if service, ok := services[artifact.ServiceID]; ok {
			serviceName = service.ServiceName
		}

		ch <- prom.MustNewConstMetric(
			c.mArtifactsDesc,
			prom.GaugeValue,
			artifactExists,
			artifact.ID,
			artifact.Name,
			artifact.Vendor,
			artifact.ServiceID,
			serviceName,
			string(artifact.Type),
			artifact.DBVersion,
			string(artifact.DataModel),
			string(artifact.Mode),
			string(artifact.Status))
	}
}
