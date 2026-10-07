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
	"errors"
	"fmt"
	"time"

	"github.com/percona/pmm/managed/models"
)

var (
	// ErrAdvisorsDisabled means that advisors checks are disabled and can't be called.
	ErrAdvisorsDisabled = errors.New("advisor checks are disabled")

	// ErrSMTPNotConfigured means PMM Server has no usable SMTP configuration, so no email can be
	// sent. The settings are inherited from Grafana's environment and cannot be changed from the
	// PMM UI, so callers surface it as a precondition failure naming the variable to set.
	ErrSMTPNotConfigured = errors.New("SMTP is not configured in PMM Server")

	// ErrLocationFolderPairAlreadyUsed returned when location-folder pair already in use and cannot be used for backup.
	ErrLocationFolderPairAlreadyUsed = errors.New("location-folder pair already used")

	// ErrAlertingDisabled means Percona Alerting is disabled and its APIs can't be called.
	ErrAlertingDisabled = errors.New("alerting is disabled")

	// ErrAzureDisabled means Azure Monitoring is disabled and its APIs can't be called.
	ErrAzureDisabled = errors.New("azure monitoring is disabled")

	// ErrPMMUpdatesDisabled means PMM server updates are disabled and calls to query/start updates are not allowed.
	ErrPMMUpdatesDisabled = errors.New("PMM updates are disabled")
)

// AdvisorRunInProgressError is returned when Advisor checks are requested while
// another run is queued or running.
type AdvisorRunInProgressError struct {
	// Run is the run in progress; nil if it finished before it could be read.
	Run *models.AdvisorRun
	// Now is when the request was rejected, to tell how long ago Run started.
	Now time.Time
}

// Error implements error. The text is shown to the user as is.
func (e *AdvisorRunInProgressError) Error() string {
	const retry = "Try again when the run finishes."
	if e.Run == nil {
		return "Advisor checks are already running. " + retry
	}

	by := "a user"
	if e.Run.TriggeredBy == models.CheckTriggeredByScheduler {
		by = "the scheduler"
	}
	return fmt.Sprintf("Advisor checks are already running (started %s ago by %s). %s",
		formatAge(e.Now.Sub(e.Run.StartedAt)), by, retry)
}

// formatAge spells out a duration in its largest whole unit, e.g. "3 minutes".
func formatAge(d time.Duration) string {
	const day = 24 * time.Hour

	n, unit := int(d/time.Second), "second"
	switch {
	case d >= day:
		n, unit = int(d/day), "day"
	case d >= time.Hour:
		n, unit = int(d/time.Hour), "hour"
	case d >= time.Minute:
		n, unit = int(d/time.Minute), "minute"
	}
	// a run that has just started reads "1 second ago", not "0 seconds ago"
	n = max(n, 1)
	if n != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s", n, unit)
}
