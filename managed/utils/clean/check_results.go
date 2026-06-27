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

// CheckResults cleans up Advisor check results history past the configured retention.
type CheckResults struct {
	db *reform.DB
}

// NewCheckResults returns a new CheckResults cleaner.
func NewCheckResults(db *reform.DB) *CheckResults {
	return &CheckResults{db: db}
}

// Run starts the Advisor check results history cleanup process.
func (c *CheckResults) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	l := logrus.WithField("component", "advisor-history-cleaner")

	for {
		settings, err := models.GetSettings(c.db)
		if err != nil {
			l.Error(err)
		} else {
			olderThanTS := models.Now().Add(-1 * settings.AdvisorHistoryRetention)
			if err := models.CleanupOldCheckResults(c.db.Querier, olderThanTS); err != nil {
				l.Error(err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
