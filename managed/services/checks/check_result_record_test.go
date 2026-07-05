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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
)

func TestNewCheckResultRecord(t *testing.T) {
	t.Parallel()

	c := check.Check{Name: "chk", Summary: "Check title", Advisor: "adv", Interval: check.Standard}
	target := services.Target{
		ServiceID:   "sid",
		ServiceName: "sname",
		ServiceType: models.MySQLServiceType,
		NodeID:      "nid",
		NodeName:    "nname",
	}
	checkedAt := models.Now()
	ri := runInfo{runID: "run-1", triggeredBy: models.CheckTriggeredByUser}

	t.Run("failed finding maps all fields", func(t *testing.T) {
		t.Parallel()

		result := check.Result{
			Summary:     "sum",
			Description: "desc",
			ReadMoreURL: "https://example.com",
			Severity:    common.Error,
			Labels:      map[string]string{"k": "v"},
		}

		rec := newCheckResultRecord(c, target, "performance", models.CheckResultFailed, result, checkedAt, ri)

		assert.Equal(t, "chk", rec.CheckName)
		assert.Equal(t, "adv", rec.AdvisorName)
		assert.Equal(t, "performance", rec.Category)
		assert.Equal(t, models.Interval(check.Standard), rec.Interval)
		assert.Equal(t, "sid", rec.ServiceID)
		assert.Equal(t, "sname", rec.ServiceName)
		assert.Equal(t, models.MySQLServiceType, rec.ServiceType)
		assert.Equal(t, "nid", rec.NodeID)
		assert.Equal(t, "nname", rec.NodeName)
		assert.Equal(t, models.CheckResultFailed, rec.Status)
		assert.Equal(t, "sum", rec.Summary)
		assert.Equal(t, "desc", rec.Description)
		assert.Equal(t, "https://example.com", rec.ReadMoreURL)
		assert.Equal(t, models.CheckSeverityError, rec.Severity)
		assert.Equal(t, checkedAt, rec.CheckedAt)
		assert.Equal(t, "run-1", rec.RunID)
		assert.Equal(t, models.CheckTriggeredByUser, rec.TriggeredBy)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"k": "v"}, labels)
	})

	t.Run("ok outcome falls back to check summary and info severity", func(t *testing.T) {
		t.Parallel()

		rec := newCheckResultRecord(c, target, "performance", models.CheckResultOK, check.Result{}, checkedAt, ri)

		assert.Equal(t, models.CheckResultOK, rec.Status)
		assert.Equal(t, "Check title", rec.Summary)
		assert.Equal(t, models.CheckSeverityInfo, rec.Severity)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Empty(t, labels)
	})

	t.Run("error outcome falls back to check summary and debug severity", func(t *testing.T) {
		t.Parallel()

		result := check.Result{Description: "execution failed"}

		rec := newCheckResultRecord(c, target, "performance", models.CheckResultError, result, checkedAt, ri)

		assert.Equal(t, models.CheckResultError, rec.Status)
		assert.Equal(t, "Check title", rec.Summary)
		assert.Equal(t, "execution failed", rec.Description)
		assert.Equal(t, models.CheckSeverityDebug, rec.Severity)
	})
}
