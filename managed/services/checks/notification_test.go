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

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
)

func TestBuildAdvisorEmailReport(t *testing.T) {
	t.Parallel()

	sCounts := map[common.Severity]int{
		common.Error:   1,
		common.Warning: 1,
	}
	tCounts := map[models.ServiceType]int{
		models.MySQLServiceType:   1,
		models.MongoDBServiceType: 1,
	}
	insights := []string{"Insight A", "Insight B"}

	want := `Percona Monitoring and Management runs Advisor checks against your monitored databases to surface potential issues. This report covers run 7394c7d1-53ae-4094-98b3-b7fe561dbac1, which was triggered manually by an operator. It found 2 insight(s) with the selected severities (Error, Warning) that may need your attention.

Findings by severity:
  Error: 1
  Warning: 1

Findings by technology:
  MySQL: 1
  PostgreSQL: 0
  MongoDB: 1

Next steps:
  - Review the insights below, addressing the most severe findings first.
  - Follow each insight's "Read More" link for remediation guidance.
  - Prioritize issues affecting production services.
  - See full details in PMM under Advisors -> Insights.

Advisor Insights (2):

Insight A

Insight B`

	run := &models.AdvisorRun{ID: "7394c7d1-53ae-4094-98b3-b7fe561dbac1", TriggeredBy: models.CheckTriggeredByUser}
	got := buildAdvisorEmailReport(run, []common.Severity{common.Error, common.Warning}, sCounts, tCounts, insights)
	require.Equal(t, want, got)
}

func TestBuildSharedInsightsEmail(t *testing.T) {
	t.Parallel()

	t.Run("filters", func(t *testing.T) {
		t.Parallel()

		failed := models.CheckResultFailed
		severity := models.Severity(common.Warning)
		filters := models.InsightFilters{RunID: "run-1", ServiceName: "mysql", Status: &failed, Severity: &severity}
		counts := map[models.CheckResultStatus]int{models.CheckResultFailed: 2}

		want := `Percona Monitoring and Management runs Advisor checks against your monitored databases to surface potential issues. A PMM user shared 2 Advisor insight(s) with you.

Filters:
  Run ID: run-1
  Service Name: mysql
  Status: Failed
  Severity: Warning

Insights by status:
  Failed: 2
  Error: 0
  OK: 0
  Pending: 0
  Not run: 0

See full details in PMM under Advisors -> Insights.

Advisor Insights (2):

Insight A

Insight B`

		got := buildSharedInsightsEmail(filters, counts, []string{"Insight A", "Insight B"})
		require.Equal(t, want, got)
	})

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()

		counts := map[models.CheckResultStatus]int{models.CheckResultOK: 1}
		got := buildSharedInsightsEmail(models.InsightFilters{}, counts, []string{"Insight A"})
		assert.Contains(t, got, "Filters:\n  None, all insights are included\n")
		assert.Contains(t, got, "  OK: 1\n")
	})
}

func TestEmailInsights(t *testing.T) {
	t.Parallel()

	// the insights query returns no rows, and the count query returns total
	setup := func(t *testing.T, total int) *Service {
		t.Helper()
		sqlDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
			require.NoError(t, mock.ExpectationsWereMet())
		})

		mock.ExpectQuery(`SELECT .* FROM "?advisor_insights"?`).
			WithArgs(maxEmailedInsights, 0).
			WillReturnRows(sqlmock.NewRows(models.InsightTable.Columns()))
		mock.ExpectQuery(`SELECT count\(\*\) FROM "?advisor_insights"?`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(total))
		mock.ExpectClose()

		return &Service{db: reform.NewDB(sqlDB, postgresql.Dialect, nil)}
	}

	t.Run("no insights match", func(t *testing.T) {
		t.Parallel()

		count, err := setup(t, 0).EmailInsights(t.Context(), models.InsightFilters{}, []string{"dba@example.com"})
		require.ErrorIs(t, err, services.ErrNoInsightsToEmail)
		assert.Zero(t, count)
	})

	t.Run("too many insights match", func(t *testing.T) {
		t.Parallel()

		count, err := setup(t, maxEmailedInsights+1).EmailInsights(t.Context(), models.InsightFilters{}, []string{"dba@example.com"})
		require.ErrorIs(t, err, services.ErrTooManyInsightsToEmail)
		require.EqualError(t, err, "too many insights match the filters: 201 match, narrow them to 200 or fewer")
		assert.Zero(t, count)
	})
}
