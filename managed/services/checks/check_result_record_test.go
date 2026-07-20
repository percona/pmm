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

	c := check.Check{Name: "chk", Summary: "Check title", Description: "Check description", Category: "Performance", Subcategory: "Adv", Interval: check.Standard}
	target := services.Target{
		ServiceID:      "sid",
		ServiceName:    "sname",
		ServiceType:    models.MySQLServiceType,
		NodeID:         "nid",
		NodeName:       "nname",
		Environment:    "prod",
		Cluster:        "cluster-1",
		ReplicationSet: "rs-1",
	}
	checkedAt := models.Now()
	ri := runInfo{batchID: "batch-1", triggeredBy: models.CheckTriggeredByUser}

	t.Run("failed finding maps all fields", func(t *testing.T) {
		t.Parallel()

		result := check.Result{
			Summary:     "sum",
			Description: "desc",
			ReadMoreURL: "https://example.com",
			Severity:    common.Error,
			Labels:      map[string]string{"k": "v"},
		}

		rec := newCheckResultRecord(c, target, models.CheckResultFailed, result, checkedAt, ri)

		assert.Equal(t, "chk", rec.CheckName)
		assert.Equal(t, "Performance", rec.Category)
		assert.Equal(t, "Adv", rec.Subcategory)
		assert.Equal(t, models.Interval(check.Standard), rec.Interval)
		assert.Equal(t, "sid", rec.ServiceID)
		assert.Equal(t, "sname", rec.ServiceName)
		assert.Equal(t, models.MySQLServiceType, rec.ServiceType)
		assert.Equal(t, "nid", rec.NodeID)
		assert.Equal(t, "nname", rec.NodeName)
		assert.Equal(t, models.CheckResultFailed, rec.Status)
		assert.Equal(t, "sum", rec.Summary)
		assert.Equal(t, "Check description", rec.Description)
		assert.Equal(t, "desc", rec.Outcome)
		assert.Equal(t, "prod", rec.Environment)
		assert.Equal(t, "cluster-1", rec.Cluster)
		assert.Equal(t, "rs-1", rec.ReplicationSet)
		assert.Equal(t, "https://example.com", rec.ReadMoreURL)
		assert.Equal(t, models.Severity(common.Error), rec.Severity)
		assert.Equal(t, checkedAt, rec.CheckedAt)
		assert.Equal(t, "batch-1", rec.BatchID)
		assert.Equal(t, models.CheckTriggeredByUser, rec.TriggeredBy)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"k": "v"}, labels)
	})

	t.Run("ok outcome falls back to check summary and info severity", func(t *testing.T) {
		t.Parallel()

		rec := newCheckResultRecord(c, target, models.CheckResultOK, check.Result{}, checkedAt, ri)

		assert.Equal(t, models.CheckResultOK, rec.Status)
		assert.Equal(t, "Check title", rec.Summary)
		assert.Equal(t, "Check passed", rec.Outcome)
		assert.Equal(t, models.Severity(common.Info), rec.Severity)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Empty(t, labels)
	})

	t.Run("error outcome falls back to check summary and debug severity", func(t *testing.T) {
		t.Parallel()

		result := check.Result{Description: "execution failed"}

		rec := newCheckResultRecord(c, target, models.CheckResultError, result, checkedAt, ri)

		assert.Equal(t, models.CheckResultError, rec.Status)
		assert.Equal(t, "Check title", rec.Summary)
		assert.Equal(t, "Check description", rec.Description)
		assert.Equal(t, "execution failed", rec.Outcome)
		assert.Equal(t, models.Severity(common.Debug), rec.Severity)
	})
}
