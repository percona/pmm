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

	c := check.Check{Name: "chk", Summary: "Check title", Description: "Check description", Category: "Performance", Interval: check.Standard}
	target := services.Target{
		ServiceID:      "sid",
		ServiceName:    "sname",
		ServiceType:    models.MySQLServiceType,
		NodeID:         "nid",
		NodeName:       "nname",
		Environment:    "prod",
		Cluster:        "cluster-1",
		ReplicationSet: "rs-1",
		Region:         "us-east-1",
		AZ:             "us-east-1f",
		Labels:         map[string]string{"az": "us-east-1f", "region": "us-east-1", "k": "target-wins"},
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

		rec := newInsightRecord(c, target, models.CheckResultFailed, result, checkedAt, ri)

		assert.Equal(t, "chk", rec.CheckName)
		assert.Equal(t, "Performance", rec.Category)
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
		assert.Equal(t, "us-east-1", rec.Region)
		assert.Equal(t, "us-east-1f", rec.AZ)
		assert.Equal(t, "https://example.com", rec.ReadMoreURL)
		assert.Equal(t, models.Severity(common.Error), rec.Severity)
		assert.Equal(t, checkedAt, rec.CheckedAt)
		assert.Equal(t, "run-1", rec.RunID)
		assert.Equal(t, models.CheckTriggeredByUser, rec.TriggeredBy)

		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, map[string]string{
			"az":     "us-east-1f",
			"region": "us-east-1",
			"k":      "target-wins",
		}, labels)
	})

	t.Run("ok outcome falls back to check summary and info severity", func(t *testing.T) {
		t.Parallel()

		rec := newInsightRecord(c, target, models.CheckResultOK, check.Result{}, checkedAt, ri)

		assert.Equal(t, models.CheckResultOK, rec.Status)
		assert.Equal(t, "Check title", rec.Summary)
		assert.Equal(t, "Check passed", rec.Outcome)
		assert.Equal(t, models.Severity(common.Info), rec.Severity)

		// target labels are carried even when the check reports none of its own
		labels, err := rec.GetLabels()
		require.NoError(t, err)
		assert.Equal(t, target.Labels, labels)
	})

	t.Run("error outcome falls back to check summary and debug severity", func(t *testing.T) {
		t.Parallel()

		result := check.Result{Description: "execution failed"}

		rec := newInsightRecord(c, target, models.CheckResultError, result, checkedAt, ri)

		assert.Equal(t, models.CheckResultError, rec.Status)
		assert.Equal(t, "Check title", rec.Summary)
		assert.Equal(t, "Check description", rec.Description)
		assert.Equal(t, "execution failed", rec.Outcome)
		assert.Equal(t, models.Severity(common.Info), rec.Severity)
	})
}
