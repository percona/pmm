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

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
)

func TestBuildAdvisorEmailReport(t *testing.T) {
	t.Parallel()

	counts := map[common.Severity]int{
		common.Error:   1,
		common.Warning: 1,
	}
	insights := []string{"Insight A", "Insight B"}

	want := `Percona Monitoring and Management runs Advisor checks against your monitored databases to surface potential issues. This report covers batch batch-123, which was triggered manually by an operator. It found 2 insight(s) at or above the "Warning" severity level that may need your attention.

Findings by severity:
  Emergency: 0
  Alert: 0
  Critical: 0
  Error: 1
  Warning: 1

Next steps:
  - Review the insights below, addressing the most severe findings first.
  - Follow each insight's "Read More" link for remediation guidance.
  - Prioritize issues affecting production services.
  - See full details in PMM under Advisors -> Insights.

Advisor Insights (2):

Insight A

Insight B`

	got := buildAdvisorEmailReport("batch-123", models.CheckTriggeredByUser, common.Warning, counts, insights)
	require.Equal(t, want, got)
}
