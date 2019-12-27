// Package sqlmetrics provides Prometheus metrics for database/sql.
package sqlmetrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "go_sql"
	subsystem = "connections"
)

// Collector is a prometheus.Collector that exposes *sql.DB metrics.
type Collector struct {
	db *sql.DB

	maxOpenConnections *prometheus.Desc

	openConnections *prometheus.Desc
	inUse           *prometheus.Desc
	idle            *prometheus.Desc

	waitCount         *prometheus.Desc
	waitDuration      *prometheus.Desc
	maxIdleClosed     *prometheus.Desc
	maxLifetimeClosed *prometheus.Desc
}

// NewCollector creates a new collector.
func NewCollector(driver, dbName string, db *sql.DB) *Collector {
	constLabels := prometheus.Labels{
		"driver": driver,
		"db":     dbName,
	}

	return &Collector{
		db: db,

		maxOpenConnections: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "max_open"),
			"Maximum number of open connections to the database.",
			nil, constLabels,
		),

		openConnections: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "open"),
			"The number of established connections both in use and idle.",
			nil, constLabels,
		),
		inUse: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "in_use"),
			"The number of connections currently in use.",
			nil, constLabels,
		),
		idle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "idle"),
			"The number of idle connections.",
			nil, constLabels,
		),

		waitCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "wait_count"),
			"The total number of connections waited for.",
			nil, constLabels,
		),
		waitDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "wait_duration_seconds"),
			"The total time blocked waiting for a new connection.",
			nil, constLabels,
		),
		maxIdleClosed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "max_idle_closed"),
			"The total number of connections closed due to SetMaxIdleConns.",
			nil, constLabels,
		),
		maxLifetimeClosed: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "max_lifetime_closed"),
			"The total number of connections closed due to SetConnMaxLifetime.",
			nil, constLabels,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.maxOpenConnections

	ch <- c.openConnections
	ch <- c.inUse
	ch <- c.idle

	ch <- c.waitCount
	ch <- c.waitDuration
	ch <- c.maxIdleClosed
	ch <- c.maxLifetimeClosed
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(
		c.maxOpenConnections,
		prometheus.GaugeValue,
		float64(stats.MaxOpenConnections),
	)

	ch <- prometheus.MustNewConstMetric(
		c.openConnections,
		prometheus.GaugeValue,
		float64(stats.OpenConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		c.inUse,
		prometheus.GaugeValue,
		float64(stats.InUse),
	)
	ch <- prometheus.MustNewConstMetric(
		c.idle,
		prometheus.GaugeValue,
		float64(stats.Idle),
	)

	ch <- prometheus.MustNewConstMetric(
		c.waitCount,
		prometheus.CounterValue,
		float64(stats.WaitCount),
	)
	ch <- prometheus.MustNewConstMetric(
		c.waitDuration,
		prometheus.CounterValue,
		stats.WaitDuration.Seconds(),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxIdleClosed,
		prometheus.CounterValue,
		float64(stats.MaxIdleClosed),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxLifetimeClosed,
		prometheus.CounterValue,
		float64(stats.MaxLifetimeClosed),
	)
}

// check interfaces
var (
	_ prometheus.Collector = (*Collector)(nil)
)
