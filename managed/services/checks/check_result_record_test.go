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

	c := check.Check{Name: "chk", Advisor: "adv", Interval: check.Standard}
	target := services.Target{
		ServiceID:   "sid",
		ServiceName: "sname",
		ServiceType: models.MySQLServiceType,
		NodeID:      "nid",
		NodeName:    "nname",
	}
	checkedAt := models.Now()

	t.Run("failed finding maps all fields", func(t *testing.T) {
		t.Parallel()

		result := check.Result{
			Summary:     "sum",
			Description: "desc",
			ReadMoreURL: "https://example.com",
			Severity:    common.Error,
			Labels:      map[string]string{"k": "v"},
		}

		rec := newCheckResultRecord(c, target, "performance", models.CheckResultFailed, result, checkedAt)

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
		assert.Equal(t, int(common.Error), rec.Severity)
		assert.Equal(t, checkedAt, rec.CheckedAt)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"k": "v"}, labels)
	})

	t.Run("ok outcome has empty finding fields", func(t *testing.T) {
		t.Parallel()

		rec := newCheckResultRecord(c, target, "performance", models.CheckResultOK, check.Result{}, checkedAt)

		assert.Equal(t, models.CheckResultOK, rec.Status)
		assert.Empty(t, rec.Summary)
		assert.Zero(t, rec.Severity)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Empty(t, labels)
	})
}
