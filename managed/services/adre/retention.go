// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// RunAdreChatRetentionLoop deletes stale ADRE conversations once per interval until ctx is done.
func RunAdreChatRetentionLoop(ctx context.Context, db reform.DBTX, l *logrus.Entry, interval time.Duration) {
	if interval <= 0 {
		interval = 24 * time.Hour //nolint:mnd
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	runOnce := func() {
		settings, err := models.GetSettings(db)
		if err != nil {
			l.Warnf("retention: GetSettings: %v", err)
			return
		}
		days := settings.GetAdreChatRetentionDays()
		if days <= 0 {
			return
		}
		cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		n, err := models.PurgeAdreConversationsOlderThan(db, cutoff)
		if err != nil {
			l.Warnf("retention: purge: %v", err)
			return
		}
		if n > 0 {
			l.Infof("ADRE chat retention: removed %d conversation(s) older than %d days", n, days)
		}
	}

	runOnce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runOnce()
		}
	}
}
