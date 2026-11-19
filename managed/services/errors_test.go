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

package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/models"
)

func TestAdvisorRunInProgressError(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name        string
		triggeredBy models.CheckTriggeredBy
		age         time.Duration
		expected    string
	}{
		{
			name:        "just started",
			triggeredBy: models.CheckTriggeredByUser,
			age:         200 * time.Millisecond,
			expected:    "Advisor checks are already running (started 1 second ago by a user). Try again when the run finishes.",
		},
		{
			name:        "seconds",
			triggeredBy: models.CheckTriggeredByUser,
			age:         40 * time.Second,
			expected:    "Advisor checks are already running (started 40 seconds ago by a user). Try again when the run finishes.",
		},
		{
			name:        "minutes",
			triggeredBy: models.CheckTriggeredByScheduler,
			age:         3*time.Minute + 59*time.Second,
			expected:    "Advisor checks are already running (started 3 minutes ago by the scheduler). Try again when the run finishes.",
		},
		{
			name:        "one hour",
			triggeredBy: models.CheckTriggeredByScheduler,
			age:         time.Hour + 5*time.Minute,
			expected:    "Advisor checks are already running (started 1 hour ago by the scheduler). Try again when the run finishes.",
		},
		{
			name:        "days",
			triggeredBy: models.CheckTriggeredByUser,
			age:         50 * time.Hour,
			expected:    "Advisor checks are already running (started 2 days ago by a user). Try again when the run finishes.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := &AdvisorRunInProgressError{
				Run: &models.AdvisorRun{TriggeredBy: tc.triggeredBy, StartedAt: now.Add(-tc.age)},
				Now: now,
			}
			assert.Equal(t, tc.expected, err.Error())
			// the UI snackbar cuts longer messages off
			assert.LessOrEqual(t, len(err.Error()), 120)
		})
	}

	t.Run("run already gone", func(t *testing.T) {
		t.Parallel()
		err := &AdvisorRunInProgressError{Now: now}
		assert.Equal(t, "Advisor checks are already running. Try again when the run finishes.", err.Error())
	})
}
