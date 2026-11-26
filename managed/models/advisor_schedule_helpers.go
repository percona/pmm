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
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/pi/common"
)

// maxAdvisorScheduleNameLength limits schedule names, which head Run history rows and email subjects.
const maxAdvisorScheduleNameLength = 100

// DefaultAdvisorSchedule returns the schedule PMM creates when there is none: all
// checks against all services every day at 06:00 UTC, emailed as set in the settings.
func DefaultAdvisorSchedule() *AdvisorSchedule {
	return &AdvisorSchedule{
		Name:       "All checks daily",
		Enabled:    true,
		DaysOfWeek: pq.Int64Array{0, 1, 2, 3, 4, 5, 6},
		TimeOfDay:  "06:00",
		EmailMode:  AdvisorScheduleEmailSettings,
	}
}

// FindAdvisorSchedules returns all Advisor schedules ordered by time of day, then name.
func FindAdvisorSchedules(ctx context.Context, q *reform.Querier) ([]*AdvisorSchedule, error) {
	rows, err := q.WithContext(ctx).SelectAllFrom(AdvisorScheduleTable, "ORDER BY time_of_day, name")
	if err != nil {
		return nil, fmt.Errorf("failed to select advisor schedules: %w", err)
	}

	schedules := make([]*AdvisorSchedule, 0, len(rows))
	for _, r := range rows {
		schedules = append(schedules, r.(*AdvisorSchedule)) //nolint:forcetypeassert
	}
	return schedules, nil
}

// FindAdvisorScheduleByID returns the Advisor schedule with the given ID, or a NotFound error.
func FindAdvisorScheduleByID(ctx context.Context, q *reform.Querier, id string) (*AdvisorSchedule, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Advisor schedule ID.")
	}

	s := &AdvisorSchedule{ID: id}
	err := q.WithContext(ctx).Reload(s)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Advisor schedule with ID %s not found.", id)
		}
		return nil, fmt.Errorf("failed to select advisor schedule %s: %w", id, err)
	}
	return s, nil
}

// CreateAdvisorSchedule validates and stores a new Advisor schedule.
func CreateAdvisorSchedule(ctx context.Context, q *reform.Querier, s *AdvisorSchedule) error {
	err := normalizeAdvisorSchedule(s)
	if err != nil {
		return err
	}

	s.ID = uuid.NewString()
	err = q.WithContext(ctx).Insert(s)
	if err != nil {
		return advisorScheduleWriteError(s, err)
	}
	return nil
}

// UpdateAdvisorSchedule validates the schedule and replaces the stored one with the same ID.
func UpdateAdvisorSchedule(ctx context.Context, q *reform.Querier, s *AdvisorSchedule) error {
	existing, err := FindAdvisorScheduleByID(ctx, q, s.ID)
	if err != nil {
		return err
	}

	err = normalizeAdvisorSchedule(s)
	if err != nil {
		return err
	}

	s.CreatedAt = existing.CreatedAt
	err = q.WithContext(ctx).Update(s)
	if err != nil {
		return advisorScheduleWriteError(s, err)
	}
	return nil
}

// RemoveAdvisorSchedule deletes the Advisor schedule with the given ID. Its runs
// keep the schedule name but lose the link to it.
func RemoveAdvisorSchedule(ctx context.Context, q *reform.Querier, id string) error {
	s, err := FindAdvisorScheduleByID(ctx, q, id)
	if err != nil {
		return err
	}

	err = q.WithContext(ctx).Delete(s)
	if err != nil {
		return fmt.Errorf("failed to delete advisor schedule %s: %w", id, err)
	}
	return nil
}

// CreateDefaultAdvisorSchedule creates the default schedule unless any schedule
// exists, and reports whether it did.
func CreateDefaultAdvisorSchedule(ctx context.Context, q *reform.Querier) (bool, error) {
	var exists bool
	err := q.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM "+AdvisorScheduleTable.Name()+")").Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to look for advisor schedules: %w", err)
	}
	if exists {
		return false, nil
	}

	err = CreateAdvisorSchedule(ctx, q, DefaultAdvisorSchedule())
	if status.Code(err) == codes.AlreadyExists {
		// another HA node has just created it
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// advisorScheduleWriteError turns a violation of the unique name into an AlreadyExists error.
func advisorScheduleWriteError(s *AdvisorSchedule, err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "advisor_schedules_name_idx" {
		return status.Errorf(codes.AlreadyExists, "Advisor schedule '%s' already exists.", s.Name)
	}
	return fmt.Errorf("failed to save advisor schedule '%s': %w", s.Name, err)
}

// normalizeAdvisorSchedule validates the schedule, returning InvalidArgument
// errors, and puts its lists in canonical form: sorted days, no duplicates,
// severities most severe first, and no recipients or severities the email mode
// does not use.
func normalizeAdvisorSchedule(s *AdvisorSchedule) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return status.Error(codes.InvalidArgument, "Advisor schedule name is required.")
	}
	if utf8.RuneCountInString(s.Name) > maxAdvisorScheduleNameLength {
		return status.Errorf(codes.InvalidArgument, "Advisor schedule name must be at most %d characters.", maxAdvisorScheduleNameLength)
	}

	if len(s.DaysOfWeek) == 0 {
		return status.Error(codes.InvalidArgument, "At least one day of the week is required.")
	}
	for _, d := range s.DaysOfWeek {
		if d < 0 || d > 6 {
			return status.Errorf(codes.InvalidArgument, "Invalid day of the week %d: use 0 (Sunday) to 6 (Saturday).", d)
		}
	}
	slices.Sort(s.DaysOfWeek)
	s.DaysOfWeek = slices.Compact(s.DaysOfWeek)

	_, _, ok := parseTimeOfDay(s.TimeOfDay)
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Invalid time of day '%s': use HH:MM in UTC.", s.TimeOfDay)
	}

	s.CheckNames = compactStrings(s.CheckNames)
	s.ServiceIDs = compactStrings(s.ServiceIDs)

	switch s.EmailMode {
	case AdvisorScheduleEmailCustom:
		s.EmailAddresses = compactStrings(s.EmailAddresses)
		if len(s.EmailAddresses) == 0 {
			return status.Error(codes.InvalidArgument, "At least one email address is required for custom recipients.")
		}
		err := validateEmailAddresses(s.EmailAddresses)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "Invalid email addresses: %s.", err)
		}
	case AdvisorScheduleEmailSettings:
		s.EmailAddresses = nil
	case AdvisorScheduleEmailOff:
		s.EmailAddresses = nil
		s.EmailSeverities = nil
	default:
		return status.Errorf(codes.InvalidArgument, "Unknown email mode '%s'.", s.EmailMode)
	}

	for _, v := range s.EmailSeverities {
		switch common.Severity(v) {
		case common.Critical, common.Error, common.Warning, common.Info:
		default:
			return status.Errorf(codes.InvalidArgument,
				"Unsupported email severity %s: use critical, error, warning or info.", common.Severity(v))
		}
	}
	// lower values are more severe
	slices.Sort(s.EmailSeverities)
	s.EmailSeverities = slices.Compact(s.EmailSeverities)

	return nil
}

// compactStrings drops empty and repeated values, keeping the first occurrence of each.
func compactStrings(values []string) []string {
	res := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" && !slices.Contains(res, v) {
			res = append(res, v)
		}
	}
	return res
}
