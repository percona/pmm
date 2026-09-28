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

package clean

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// Insights cleans up Advisor insights past the configured retention.
type Insights struct {
	db *reform.DB
}

// NewInsights returns a new Insights cleaner.
func NewInsights(db *reform.DB) *Insights {
	return &Insights{db: db}
}

// Run starts the Advisor insights cleanup process.
func (c *Insights) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	l := logrus.WithField("component", "advisor-history-cleaner")

	for {
		c.cleanup(ctx, l)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// cleanup performs a single cleanup pass, removing insights past the configured retention.
func (c *Insights) cleanup(ctx context.Context, l *logrus.Entry) {
	settings, err := models.GetSettings(c.db)
	if err != nil {
		l.Error(err)
		return
	}

	olderThanTS := models.Now().Add(-1 * settings.AdvisorHistoryRetention)
	err = models.CleanupOldInsights(ctx, c.db.Querier, olderThanTS)
	if err != nil {
		l.Error(err)
	}

	// Runs share the retention window but are pruned by their own start time, so
	// a run keeps reporting its stored totals until it ages out itself.
	err = models.CleanupOldAdvisorRuns(ctx, c.db.Querier, olderThanTS)
	if err != nil {
		l.Error(err)
	}
}
