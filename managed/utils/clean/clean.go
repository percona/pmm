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

// Package clean has the old actions results cleaner.
package clean

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// CleanResults has unexported fields for the results cleanup function.
type CleanResults struct {
	db *reform.DB
}

// New returns a new CleanResults instance.
func New(db *reform.DB) *CleanResults {
	return &CleanResults{db: db}
}

// Run starts the clean process.
func (c *CleanResults) Run(ctx context.Context, interval time.Duration, olderThan time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	l := logrus.WithField("component", "cleaner")

	for {
		olderThanTS := models.Now().Add(-1 * olderThan)
		if err := models.CleanupOldActionResults(c.db.Querier, olderThanTS); err != nil {
			l.Error(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
