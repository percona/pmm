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

package management

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// ListAdvisorSchedules returns the Advisor schedules with their next and most recent runs.
func (s *ChecksAPIService) ListAdvisorSchedules(
	ctx context.Context,
	_ *advisorsv1.ListAdvisorSchedulesRequest,
) (*advisorsv1.ListAdvisorSchedulesResponse, error) {
	schedules, err := s.checksService.GetSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get advisor schedules: %w", err)
	}

	lastRuns, err := s.checksService.GetLastScheduleRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last runs of advisor schedules: %w", err)
	}

	now := models.Now()
	res := make([]*advisorsv1.AdvisorSchedule, 0, len(schedules))
	for _, sc := range schedules {
		item, err := advisorScheduleToAPI(sc, lastRuns[sc.ID], now)
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}

	return &advisorsv1.ListAdvisorSchedulesResponse{Schedules: res}, nil
}

// CreateAdvisorSchedule creates an Advisor schedule.
func (s *ChecksAPIService) CreateAdvisorSchedule(
	ctx context.Context,
	req *advisorsv1.CreateAdvisorScheduleRequest,
) (*advisorsv1.CreateAdvisorScheduleResponse, error) {
	sc := &models.AdvisorSchedule{
		Name:            req.Name,
		Enabled:         req.Enabled == nil || *req.Enabled,
		DaysOfWeek:      convertAPIDaysOfWeek(req.DaysOfWeek),
		TimeOfDay:       req.TimeOfDay,
		CheckNames:      req.CheckNames,
		ServiceIDs:      req.ServiceIds,
		EmailMode:       convertAPIEmailMode(req.EmailMode),
		EmailAddresses:  req.EmailAddresses,
		EmailSeverities: convertAPIEmailSeverities(req.EmailSeverities),
	}
	err := s.checksService.CreateSchedule(ctx, sc)
	if err != nil {
		return nil, err
	}

	item, err := advisorScheduleToAPI(sc, nil, models.Now())
	if err != nil {
		return nil, err
	}
	return &advisorsv1.CreateAdvisorScheduleResponse{Schedule: item}, nil
}

// UpdateAdvisorSchedule replaces an Advisor schedule.
func (s *ChecksAPIService) UpdateAdvisorSchedule(
	ctx context.Context,
	req *advisorsv1.UpdateAdvisorScheduleRequest,
) (*advisorsv1.UpdateAdvisorScheduleResponse, error) {
	sc := &models.AdvisorSchedule{
		ID:              req.ScheduleId,
		Name:            req.Name,
		Enabled:         req.Enabled == nil || *req.Enabled,
		DaysOfWeek:      convertAPIDaysOfWeek(req.DaysOfWeek),
		TimeOfDay:       req.TimeOfDay,
		CheckNames:      req.CheckNames,
		ServiceIDs:      req.ServiceIds,
		EmailMode:       convertAPIEmailMode(req.EmailMode),
		EmailAddresses:  req.EmailAddresses,
		EmailSeverities: convertAPIEmailSeverities(req.EmailSeverities),
	}
	err := s.checksService.UpdateSchedule(ctx, sc)
	if err != nil {
		return nil, err
	}

	lastRuns, err := s.checksService.GetLastScheduleRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the last runs of advisor schedules: %w", err)
	}

	item, err := advisorScheduleToAPI(sc, lastRuns[sc.ID], models.Now())
	if err != nil {
		return nil, err
	}
	return &advisorsv1.UpdateAdvisorScheduleResponse{Schedule: item}, nil
}

// DeleteAdvisorSchedule deletes an Advisor schedule.
func (s *ChecksAPIService) DeleteAdvisorSchedule(
	ctx context.Context,
	req *advisorsv1.DeleteAdvisorScheduleRequest,
) (*advisorsv1.DeleteAdvisorScheduleResponse, error) {
	err := s.checksService.DeleteSchedule(ctx, req.ScheduleId)
	if err != nil {
		return nil, err
	}

	return &advisorsv1.DeleteAdvisorScheduleResponse{}, nil
}

// RunAdvisorSchedule starts a run of an Advisor schedule now and returns its ID.
func (s *ChecksAPIService) RunAdvisorSchedule(
	ctx context.Context,
	req *advisorsv1.RunAdvisorScheduleRequest,
) (*advisorsv1.RunAdvisorScheduleResponse, error) {
	runID, err := s.checksService.RunSchedule(ctx, req.ScheduleId)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}
		_, inProgress := errors.AsType[*services.AdvisorRunInProgressError](err)
		if inProgress {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	return &advisorsv1.RunAdvisorScheduleResponse{RunId: runID}, nil
}

// advisorScheduleToAPI converts an Advisor schedule and its most recent run, if
// any, to its API representation; the next run is the first one after now.
func advisorScheduleToAPI(sc *models.AdvisorSchedule, lastRun *models.AdvisorRun, now time.Time) (*advisorsv1.AdvisorSchedule, error) {
	item := &advisorsv1.AdvisorSchedule{
		ScheduleId:     sc.ID,
		Name:           sc.Name,
		Enabled:        sc.Enabled,
		TimeOfDay:      sc.TimeOfDay,
		CheckNames:     sc.CheckNames,
		ServiceIds:     sc.ServiceIDs,
		EmailMode:      convertModelEmailMode(sc.EmailMode),
		EmailAddresses: sc.EmailAddresses,
		CreatedAt:      timestamppb.New(sc.CreatedAt),
		UpdatedAt:      timestamppb.New(sc.UpdatedAt),
	}
	for _, d := range sc.DaysOfWeek {
		item.DaysOfWeek = append(item.DaysOfWeek, int32(d)) //nolint:gosec // a weekday, 0-6
	}
	for _, severity := range sc.EmailSeverities {
		item.EmailSeverities = append(item.EmailSeverities, managementv1.Severity(severity)) //nolint:gosec // severity is a bounded enum (0-8), no overflow
	}

	if sc.Enabled {
		next := sc.NextRunAfter(now)
		if !next.IsZero() {
			item.NextRunAt = timestamppb.New(next)
		}
	}

	if lastRun != nil {
		run, err := advisorRunToAPI(lastRun, nil)
		if err != nil {
			return nil, err
		}
		item.LastRun = run
	}

	return item, nil
}

// convertAPIDaysOfWeek converts API weekdays to their stored form; validation rejects values out of range.
func convertAPIDaysOfWeek(days []int32) pq.Int64Array {
	res := make(pq.Int64Array, 0, len(days))
	for _, d := range days {
		res = append(res, int64(d))
	}
	return res
}

// convertAPIEmailSeverities converts API severities to their stored form; validation rejects unsupported ones.
func convertAPIEmailSeverities(severities []managementv1.Severity) pq.Int64Array {
	res := make(pq.Int64Array, 0, len(severities))
	for _, severity := range severities {
		res = append(res, int64(severity))
	}
	return res
}

// convertAPIEmailMode converts an API email mode; an unspecified or unknown one
// maps to an empty mode, which fails validation.
func convertAPIEmailMode(mode advisorsv1.AdvisorScheduleEmailMode) models.AdvisorScheduleEmailMode {
	switch mode {
	case advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_SETTINGS:
		return models.AdvisorScheduleEmailSettings
	case advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_CUSTOM:
		return models.AdvisorScheduleEmailCustom
	case advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_OFF:
		return models.AdvisorScheduleEmailOff
	default:
		return ""
	}
}

// convertModelEmailMode converts a stored email mode to its API representation.
func convertModelEmailMode(mode models.AdvisorScheduleEmailMode) advisorsv1.AdvisorScheduleEmailMode {
	switch mode {
	case models.AdvisorScheduleEmailSettings:
		return advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_SETTINGS
	case models.AdvisorScheduleEmailCustom:
		return advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_CUSTOM
	case models.AdvisorScheduleEmailOff:
		return advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_OFF
	default:
		return advisorsv1.AdvisorScheduleEmailMode_ADVISOR_SCHEDULE_EMAIL_MODE_UNSPECIFIED
	}
}
