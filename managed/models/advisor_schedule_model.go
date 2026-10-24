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

package models

import (
	"slices"
	"time"

	"github.com/lib/pq"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/pi/common"
)

//go:generate go tool reform

// AdvisorScheduleEmailMode selects who the runs of an Advisor schedule are emailed to.
type AdvisorScheduleEmailMode string

// Available Advisor schedule email modes.
const (
	// AdvisorScheduleEmailSettings emails the default recipients from the Advisor settings.
	AdvisorScheduleEmailSettings AdvisorScheduleEmailMode = "settings"
	// AdvisorScheduleEmailCustom emails the schedule's own recipients.
	AdvisorScheduleEmailCustom AdvisorScheduleEmailMode = "custom"
	// AdvisorScheduleEmailOff sends no email.
	AdvisorScheduleEmailOff AdvisorScheduleEmailMode = "off"
)

// AdvisorSchedule runs Advisor checks on the given weekdays at a time of day, both in UTC.
//
//reform:advisor_schedules
type AdvisorSchedule struct {
	ID      string `reform:"id,pk"`
	Name    string `reform:"name"`
	Enabled bool   `reform:"enabled"`
	// DaysOfWeek holds time.Weekday values, Sunday being 0.
	DaysOfWeek pq.Int64Array `reform:"days_of_week"`
	// TimeOfDay is the UTC start time as HH:MM.
	TimeOfDay string `reform:"time_of_day"`
	// CheckNames and ServiceIDs narrow the runs; empty means all.
	CheckNames pq.StringArray           `reform:"check_names"`
	ServiceIDs pq.StringArray           `reform:"service_ids"`
	EmailMode  AdvisorScheduleEmailMode `reform:"email_mode"`
	// EmailAddresses are the recipients of the custom email mode.
	EmailAddresses pq.StringArray `reform:"email_addresses"`
	// EmailSeverities hold common.Severity values; empty means the default severities from the settings.
	EmailSeverities pq.Int64Array `reform:"email_severities"`
	CreatedAt       time.Time     `reform:"created_at"`
	UpdatedAt       time.Time     `reform:"updated_at"`
}

// BeforeInsert implements reform.BeforeInserter interface.
func (s *AdvisorSchedule) BeforeInsert() error {
	now := Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	s.fillEmptyArrays()
	return nil
}

// BeforeUpdate implements reform.BeforeUpdater interface.
func (s *AdvisorSchedule) BeforeUpdate() error {
	s.UpdatedAt = Now()
	s.fillEmptyArrays()
	return nil
}

// AfterFind implements reform.AfterFinder interface.
func (s *AdvisorSchedule) AfterFind() error {
	s.CreatedAt = s.CreatedAt.UTC()
	s.UpdatedAt = s.UpdatedAt.UTC()
	return nil
}

// fillEmptyArrays replaces nil arrays, which are stored as NULL, with empty ones.
func (s *AdvisorSchedule) fillEmptyArrays() {
	for _, a := range []*pq.StringArray{&s.CheckNames, &s.ServiceIDs, &s.EmailAddresses} {
		if *a == nil {
			*a = pq.StringArray{}
		}
	}
	for _, a := range []*pq.Int64Array{&s.DaysOfWeek, &s.EmailSeverities} {
		if *a == nil {
			*a = pq.Int64Array{}
		}
	}
}

// Severities returns the severities emailed by the schedule's runs; empty means the default ones.
func (s *AdvisorSchedule) Severities() []common.Severity {
	res := make([]common.Severity, 0, len(s.EmailSeverities))
	for _, v := range s.EmailSeverities {
		res = append(res, common.Severity(v))
	}
	return res
}

// NextRunAfter returns the first time after t the schedule falls due, whether it
// is enabled or not. It returns the zero time when the schedule has no valid
// days or time of day.
func (s *AdvisorSchedule) NextRunAfter(t time.Time) time.Time {
	hour, minute, ok := parseTimeOfDay(s.TimeOfDay)
	if !ok {
		return time.Time{}
	}

	t = t.UTC()
	// a week and a day cover a single weekday whose time today has passed
	for d := range 8 {
		at := time.Date(t.Year(), t.Month(), t.Day()+d, hour, minute, 0, 0, time.UTC)
		if at.After(t) && slices.Contains(s.DaysOfWeek, int64(at.Weekday())) {
			return at
		}
	}
	return time.Time{}
}

// parseTimeOfDay parses an HH:MM time of day.
func parseTimeOfDay(v string) (int, int, bool) {
	t, err := time.Parse("15:04", v)
	if err != nil || len(v) != len("15:04") {
		return 0, 0, false
	}
	return t.Hour(), t.Minute(), true
}

// check interfaces.
var (
	_ reform.BeforeInserter = (*AdvisorSchedule)(nil)
	_ reform.BeforeUpdater  = (*AdvisorSchedule)(nil)
	_ reform.AfterFinder    = (*AdvisorSchedule)(nil)
)
