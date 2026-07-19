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

package checks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

// TestInsightToText locks the Go output to the UI's "Copy to text" format
// (insightToText in ui/apps/pmm/src/pages/advisors/insights/AdvisorInsights.utils.ts).
func TestInsightToText(t *testing.T) {
	t.Parallel()

	t.Run("all fields", func(t *testing.T) {
		t.Parallel()

		r := &models.CheckResult{
			ID:             "insight-1",
			BatchID:        "batch-1",
			CheckName:      "mysql_version",
			Subcategory:    "version_advisor",
			Category:       "Performance",
			ServiceName:    "mysql-prod",
			ServiceType:    "mysql",
			NodeName:       "node-a",
			Environment:    "prod",
			Cluster:        "cluster-1",
			ReplicationSet: "rs0",
			Interval:       models.Standard,
			TriggeredBy:    models.CheckTriggeredByScheduler,
			IsRead:         false,
			Summary:        "Outdated MySQL version",
			Description:    "The MySQL version is old",
			Outcome:        "Upgrade recommended",
			Severity:       models.CheckSeverityWarning,
			ReadMoreURL:    "https://example.com/more",
			Status:         models.CheckResultFailed,
			CheckedAt:      time.Date(2026, 7, 16, 10, 30, 0, 0, time.UTC),
		}
		require.NoError(t, r.SetLabels(map[string]string{"tier": "db", "env": "prod"}))

		want := `The Advisor Check "Outdated MySQL version" completed at 2026-07-16 10:30:00 with status "Failed".

Check Details:
  ID: insight-1
  Batch ID: batch-1
  Check Name: mysql_version
  Category: Performance
  Sub category: version_advisor
  Service Name: mysql-prod
  Service Type: mysql
  Node Name: node-a
  Environment: prod
  Cluster: cluster-1
  Replication Set: rs0
  Interval: Standard
  Triggered By: Scheduler
  Read: Unread
  Summary: Outdated MySQL version
  Description: The MySQL version is old
  Outcome: Upgrade recommended
  Severity: Warning
  Read More: https://example.com/more
  Labels: env=prod, tier=db`

		got, err := insightToText(r)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("empty fields are omitted", func(t *testing.T) {
		t.Parallel()

		r := &models.CheckResult{
			ID:          "insight-2",
			BatchID:     "batch-2",
			CheckName:   "pg_check",
			Summary:     "Issue found",
			Severity:    models.CheckSeverityError,
			Status:      models.CheckResultFailed,
			TriggeredBy: models.CheckTriggeredByUser,
			IsRead:      true,
			CheckedAt:   time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
		}

		want := `The Advisor Check "Issue found" completed at 2026-07-16 12:00:00 with status "Failed".

Check Details:
  ID: insight-2
  Batch ID: batch-2
  Check Name: pg_check
  Interval: Standard
  Triggered By: User
  Read: Read
  Summary: Issue found
  Severity: Error`

		got, err := insightToText(r)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})
}
