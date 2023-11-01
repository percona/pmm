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

package dump

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
	dumpExists          float64 = 1
	prometheusNamespace         = "pmm_managed"
	prometheusSubsystem         = "dumps"
)

type MetricsCollector struct {
	db *reform.DB
	l  *logrus.Entry

	mDumpsDesc *prom.Desc
}

func NewMetricsCollector(db *reform.DB) *MetricsCollector {
	return &MetricsCollector{
		db: db,
		l:  logrus.WithField("component", "dumps/metrics"),
		mDumpsDesc: prom.NewDesc(
			prom.BuildFQName(prometheusNamespace, prometheusSubsystem, "dumps"),
			"Dumps",
			[]string{
				"dump_id", "status",
			},
			nil),
	}
}

func (c *MetricsCollector) Describe(ch chan<- *prom.Desc) {
	ch <- c.mDumpsDesc
}

func (c *MetricsCollector) Collect(ch chan<- prom.Metric) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), requestTimeout)
	defer cancelCtx()

	var dumps []*models.Dump
	errTx := c.db.InTransactionContext(ctx, nil, func(t *reform.TX) error {
		var err error
		dumps, err = models.FindDumps(t.Querier, models.DumpFilters{})
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if errTx != nil {
		c.l.Warnf("Failed to get dumps: %v", errTx)
		return
	}

	for _, dump := range dumps {
		ch <- prom.MustNewConstMetric(
			c.mDumpsDesc,
			prom.GaugeValue,
			dumpExists,
			dump.ID,
			string(dump.Status))
	}
}
