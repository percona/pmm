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

type MetricsCollector struct {
	db *reform.DB
	l  *logrus.Entry

	mArtifactsDesc *prom.Desc
}

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

func (c *MetricsCollector) Describe(ch chan<- *prom.Desc) {
	ch <- c.mArtifactsDesc
}

func (c *MetricsCollector) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelCtx()

	var artifacts []*models.Artifact
	var services map[string]*models.Service
	errTx := c.db.InTransactionContext(ctx, nil, func(t *reform.TX) error {
		var err error
		artifacts, err = models.FindArtifacts(t.Querier, models.ArtifactFilters{})
		if err != nil {
			return errors.Wrapf(err, "failed to find artifacts")
		}

		serviceIDs := make([]string, len(artifacts))
		for _, artifact := range artifacts {
			serviceIDs = append(serviceIDs, artifact.ServiceID)
		}

		services, err = models.FindServicesByIDs(t.Querier, serviceIDs)
		if err != nil {
			return errors.Wrapf(err, "failed to find services")
		}
		return nil
	})
	if errTx != nil {
		c.l.Warnf("Failed to get artifacts")
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
