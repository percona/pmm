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
	"maps"
	"slices"

	"github.com/AlekSi/pointer"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/check"
	"github.com/percona/pmm/managed/services"
)

// ChecksAPIService represents advisor service API.
type ChecksAPIService struct {
	advisorsv1.UnimplementedAdvisorServiceServer

	checksService checksService
	l             *logrus.Entry
}

// NewChecksAPIService creates new Checks API Service.
func NewChecksAPIService(checksService checksService) *ChecksAPIService {
	return &ChecksAPIService{
		checksService: checksService,
		l:             logrus.WithField("component", "management/checks"),
	}
}

// ListInsights returns the paginated history of Advisor check results (insights) matching the filters.
func (s *ChecksAPIService) ListInsights(
	ctx context.Context,
	req *advisorsv1.ListInsightsRequest,
) (*advisorsv1.ListInsightsResponse, error) {
	var pageIndex, pageSize int
	if req.PageIndex != nil {
		pageIndex = int(pointer.GetInt32(req.PageIndex))
	}
	if req.PageSize != nil {
		pageSize = int(pointer.GetInt32(req.PageSize))
	}

	filters := models.InsightFilters{
		ServiceID:   req.ServiceId,
		ServiceName: req.ServiceName,
		NodeName:    req.NodeName,
		Category:    req.Category,
		CheckName:   req.CheckName,
		RunID:       req.RunId,
		IsRead:      req.IsRead,
	}
	if req.Status != nil {
		if st := convertAPIResultStatus(*req.Status); st != "" {
			filters.Status = &st
		}
	}
	if req.TriggeredBy != nil {
		if tb := convertAPITriggeredBy(*req.TriggeredBy); tb != "" {
			filters.TriggeredBy = &tb
		}
	}
	if req.Severity != nil && *req.Severity != managementv1.Severity_SEVERITY_UNSPECIFIED {
		severity := models.Severity(*req.Severity)
		filters.Severity = &severity
	}
	if req.From != nil {
		from := req.From.AsTime()
		filters.From = &from
	}
	if req.To != nil {
		to := req.To.AsTime()
		filters.To = &to
	}

	results, totalItems, err := s.checksService.GetInsights(ctx, filters, pageIndex, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get insights: %w", err)
	}

	items := make([]*advisorsv1.Insight, 0, len(results))
	for _, r := range results {
		labels, err := r.GetLabels()
		if err != nil {
			return nil, fmt.Errorf("failed to decode labels for insight '%s': %w", r.ID, err)
		}

		items = append(items, &advisorsv1.Insight{
			Id:             r.ID,
			CheckName:      r.CheckName,
			RunId:          r.RunID,
			Category:       r.Category,
			Subcategory:    r.Subcategory,
			Severity:       managementv1.Severity(r.Severity), //nolint:gosec // severity is a bounded enum (0-8), no overflow
			Interval:       convertModelInterval(r.Interval),
			ServiceId:      r.ServiceID,
			ServiceName:    r.ServiceName,
			ServiceType:    string(r.ServiceType),
			NodeId:         r.NodeID,
			NodeName:       r.NodeName,
			Environment:    r.Environment,
			Cluster:        r.Cluster,
			ReplicationSet: r.ReplicationSet,
			Region:         r.Region,
			Az:             r.AZ,
			Status:         convertModelResultStatus(r.Status),
			Summary:        r.Summary,
			Description:    r.Description,
			Outcome:        r.Outcome,
			ReadMoreUrl:    r.ReadMoreURL,
			Labels:         labels,
			CheckedAt:      timestamppb.New(r.CheckedAt),
			IsRead:         r.IsRead,
			TriggeredBy:    convertModelTriggeredBy(r.TriggeredBy),
		})
	}

	totalPages := 1
	if pageSize > 0 {
		totalPages = totalItems / pageSize
		if totalItems%pageSize > 0 {
			totalPages++
		}
	}

	return &advisorsv1.ListInsightsResponse{
		Results:    items,
		TotalItems: int32(totalItems), //nolint:gosec
		TotalPages: int32(totalPages),
	}, nil
}

// ListRuns returns the paginated history of Advisor check executions.
func (s *ChecksAPIService) ListRuns(
	ctx context.Context,
	req *advisorsv1.ListRunsRequest,
) (*advisorsv1.ListRunsResponse, error) {
	var pageIndex, pageSize int
	if req.PageIndex != nil {
		pageIndex = int(pointer.GetInt32(req.PageIndex))
	}
	if req.PageSize != nil {
		pageSize = int(pointer.GetInt32(req.PageSize))
	}

	var filters models.AdvisorRunFilters
	if req.TriggeredBy != nil {
		if tb := convertAPITriggeredBy(*req.TriggeredBy); tb != "" {
			filters.TriggeredBy = &tb
		}
	}
	if req.From != nil {
		from := req.From.AsTime()
		filters.From = &from
	}
	if req.To != nil {
		to := req.To.AsTime()
		filters.To = &to
	}

	runs, totalItems, err := s.checksService.GetRuns(ctx, filters, pageIndex, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get advisor runs: %w", err)
	}

	items := make([]*advisorsv1.AdvisorRun, 0, len(runs))
	for _, r := range runs {
		severityCounts, err := r.GetSeverityCounts()
		if err != nil {
			return nil, fmt.Errorf("failed to decode severity counts for advisor run '%s': %w", r.ID, err)
		}

		item := &advisorsv1.AdvisorRun{
			Id:             r.ID,
			TriggeredBy:    convertModelTriggeredBy(r.TriggeredBy),
			StartedAt:      timestamppb.New(r.StartedAt),
			ChecksCount:    int32(r.ChecksCount),   //nolint:gosec
			ServicesCount:  int32(r.ServicesCount), //nolint:gosec
			FindingsCount:  int32(r.FindingsCount), //nolint:gosec
			ErrorsCount:    int32(r.ErrorsCount),   //nolint:gosec
			SeverityCounts: convertSeverityCounts(severityCounts),
		}
		// left unset while the run is still going
		if r.FinishedAt != nil {
			item.FinishedAt = timestamppb.New(*r.FinishedAt)
		}

		items = append(items, item)
	}

	totalPages := 1
	if pageSize > 0 {
		totalPages = totalItems / pageSize
		if totalItems%pageSize > 0 {
			totalPages++
		}
	}

	return &advisorsv1.ListRunsResponse{
		Results:    items,
		TotalItems: int32(totalItems), //nolint:gosec
		TotalPages: int32(totalPages),
	}, nil
}

// ListInsightsFilterValues returns the distinct values usable as insights filters.
func (s *ChecksAPIService) ListInsightsFilterValues(
	ctx context.Context,
	_ *advisorsv1.ListInsightsFilterValuesRequest,
) (*advisorsv1.ListInsightsFilterValuesResponse, error) {
	serviceNames, nodeNames, err := s.checksService.GetInsightsFilterValues(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get insights filter values: %w", err)
	}

	return &advisorsv1.ListInsightsFilterValuesResponse{
		ServiceNames: serviceNames,
		NodeNames:    nodeNames,
	}, nil
}

// MarkInsightsRead sets the read state on the specified Advisor insights.
func (s *ChecksAPIService) MarkInsightsRead(
	ctx context.Context,
	req *advisorsv1.MarkInsightsReadRequest,
) (*advisorsv1.MarkInsightsReadResponse, error) {
	switch {
	case len(req.Ids) > 0:
		err := s.checksService.MarkInsightsRead(ctx, req.Ids, req.IsRead)
		if err != nil {
			return nil, fmt.Errorf("failed to mark insights read: %w", err)
		}
	case req.Filters != nil:
		filters := models.InsightFilters{
			ServiceName: req.Filters.ServiceName,
			NodeName:    req.Filters.NodeName,
			Category:    req.Filters.Category,
			CheckName:   req.Filters.CheckName,
			RunID:       req.Filters.RunId,
			IsRead:      req.Filters.IsRead,
		}
		if req.Filters.Status != nil {
			if st := convertAPIResultStatus(*req.Filters.Status); st != "" {
				filters.Status = &st
			}
		}
		if req.Filters.Severity != nil && *req.Filters.Severity != managementv1.Severity_SEVERITY_UNSPECIFIED {
			severity := models.Severity(*req.Filters.Severity)
			filters.Severity = &severity
		}
		err := s.checksService.MarkInsightsReadByFilters(ctx, filters, req.IsRead)
		if err != nil {
			return nil, fmt.Errorf("failed to mark insights read by filters: %w", err)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "Either ids or filters must be provided.")
	}

	return &advisorsv1.MarkInsightsReadResponse{}, nil
}

// StartAdvisorChecks executes advisor checks and returns the ID assigned to this run.
func (s *ChecksAPIService) StartAdvisorChecks(_ context.Context, req *advisorsv1.StartAdvisorChecksRequest) (*advisorsv1.StartAdvisorChecksResponse, error) {
	// Start only specified checks from any group.
	runID, err := s.checksService.StartChecks(req.Names, req.ServiceIds)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, fmt.Errorf("failed to start advisor checks: %w", err)
	}

	return &advisorsv1.StartAdvisorChecksResponse{RunId: runID}, nil
}

// ListAdvisorChecks returns a list of available advisor checks and their statuses.
func (s *ChecksAPIService) ListAdvisorChecks(ctx context.Context, _ *advisorsv1.ListAdvisorChecksRequest) (*advisorsv1.ListAdvisorChecksResponse, error) {
	disChecks, err := s.checksService.GetDisabledChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disabled checks list: %w", err)
	}

	m := make(map[string]struct{}, len(disChecks))
	for _, c := range disChecks {
		m[c] = struct{}{}
	}

	disServices, err := s.checksService.GetDisabledServicesForChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disabled services list: %w", err)
	}

	checks, err := s.checksService.GetChecks()
	if err != nil {
		return nil, fmt.Errorf("failed to get available checks list: %w", err)
	}

	res := make([]*advisorsv1.AdvisorCheck, 0, len(checks))
	for _, c := range checks {
		_, disabled := m[c.Name]
		res = append(res, &advisorsv1.AdvisorCheck{
			Name:               c.Name,
			Enabled:            !disabled,
			Summary:            c.Summary,
			Technology:         convertTechnology(c.Technology),
			Description:        c.Description,
			Interval:           convertInterval(c.Interval),
			Category:           c.Category,
			Subcategory:        c.Subcategory,
			UserDefined:        c.UserDefined,
			DisabledServiceIds: disServices[c.Name],
		})
	}

	return &advisorsv1.ListAdvisorChecksResponse{Checks: res}, nil
}

// ListAdvisors retrieves a list of advisors based on the provided request.
func (s *ChecksAPIService) ListAdvisors(ctx context.Context, _ *advisorsv1.ListAdvisorsRequest) (*advisorsv1.ListAdvisorsResponse, error) {
	disChecks, err := s.checksService.GetDisabledChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disabled checks list: %w", err)
	}

	m := make(map[string]struct{}, len(disChecks))
	for _, c := range disChecks {
		m[c] = struct{}{}
	}

	disServices, err := s.checksService.GetDisabledServicesForChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disabled services list: %w", err)
	}

	advisors, err := s.checksService.GetAdvisors()
	if err != nil {
		return nil, fmt.Errorf("failed to get available checks list: %w", err)
	}

	res := make([]*advisorsv1.Advisor, 0, len(advisors))
	for _, a := range advisors {
		checks := make([]*advisorsv1.AdvisorCheck, 0, len(a.Checks))
		for _, c := range a.Checks {
			_, disabled := m[c.Name]
			checks = append(checks, &advisorsv1.AdvisorCheck{
				Name:               c.Name,
				Enabled:            !disabled,
				Summary:            c.Summary,
				Technology:         convertTechnology(c.Technology),
				Description:        c.Description,
				Interval:           convertInterval(c.Interval),
				Category:           c.Category,
				Subcategory:        c.Subcategory,
				UserDefined:        c.UserDefined,
				DisabledServiceIds: disServices[c.Name],
			})
		}

		res = append(res, &advisorsv1.Advisor{
			Category:    a.Category,
			Subcategory: a.Subcategory,
			Checks:      checks,
		})
	}

	return &advisorsv1.ListAdvisorsResponse{Advisors: res}, nil
}

// ChangeAdvisorChecks enables/disables advisor checks — globally or for specific
// services — by names, or changes their execution interval.
func (s *ChecksAPIService) ChangeAdvisorChecks(ctx context.Context, req *advisorsv1.ChangeAdvisorChecksRequest) (*advisorsv1.ChangeAdvisorChecksResponse, error) {
	var enableChecks, disableChecks []string
	changeIntervalParams := make(map[string]check.Interval)
	for _, p := range req.Params {
		if len(p.ServiceIds) != 0 {
			err := s.changeChecksForServices(ctx, p)
			if err != nil {
				return nil, err
			}
			continue
		}

		if p.Interval != advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED {
			interval, err := convertAPIInterval(p.Interval)
			if err != nil {
				return nil, err
			}
			changeIntervalParams[p.Name] = interval
		}

		if p.Enable != nil {
			if *p.Enable {
				enableChecks = append(enableChecks, p.Name)
			} else {
				disableChecks = append(disableChecks, p.Name)
			}
		}
	}

	if len(changeIntervalParams) != 0 {
		err := s.checksService.ChangeInterval(ctx, changeIntervalParams)
		if err != nil {
			return nil, fmt.Errorf("failed to change advisor check interval: %w", err)
		}
	}

	err := s.checksService.EnableChecks(ctx, enableChecks)
	if err != nil {
		return nil, fmt.Errorf("failed to enable disabled advisor checks: %w", err)
	}

	err = s.checksService.DisableChecks(ctx, disableChecks)
	if err != nil {
		return nil, fmt.Errorf("failed to disable advisor checks: %w", err)
	}

	return &advisorsv1.ChangeAdvisorChecksResponse{}, nil
}

// changeChecksForServices applies a per-service enable/disable params entry:
// the change affects only the given services. The interval stays check-wide
// and cannot be mixed into such an entry.
func (s *ChecksAPIService) changeChecksForServices(ctx context.Context, p *advisorsv1.ChangeAdvisorCheckParams) error {
	if p.Interval != advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED {
		return status.Errorf(codes.InvalidArgument, "Interval of check %s cannot be changed per service.", p.Name)
	}
	if p.Enable == nil {
		return status.Errorf(codes.InvalidArgument, "Enable flag is required to change check %s per service.", p.Name)
	}

	if *p.Enable {
		return s.checksService.EnableChecksForServices(ctx, p.Name, p.ServiceIds)
	}
	return s.checksService.DisableChecksForServices(ctx, p.Name, p.ServiceIds)
}

// GetAdvisorCheck returns a single advisor check by name, including its queries and script.
func (s *ChecksAPIService) GetAdvisorCheck(ctx context.Context, req *advisorsv1.GetAdvisorCheckRequest) (*advisorsv1.GetAdvisorCheckResponse, error) {
	c, enabled, disabledServiceIDs, err := s.getCheck(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &advisorsv1.GetAdvisorCheckResponse{Check: advisorCheckToAPI(c, enabled, disabledServiceIDs)}, nil
}

// CreateAdvisorCheck creates a new user-authored advisor check.
func (s *ChecksAPIService) CreateAdvisorCheck(ctx context.Context, req *advisorsv1.CreateAdvisorCheckRequest) (*advisorsv1.CreateAdvisorCheckResponse, error) {
	if req.Check == nil {
		return nil, status.Error(codes.InvalidArgument, "Check is required.")
	}

	err := s.checksService.CreateAdvisorCheck(ctx, apiToAdvisorCheck(req.Check))
	if err != nil {
		return nil, err
	}

	c, enabled, disabledServiceIDs, err := s.getCheck(ctx, req.Check.Name)
	if err != nil {
		return nil, err
	}

	return &advisorsv1.CreateAdvisorCheckResponse{Check: advisorCheckToAPI(c, enabled, disabledServiceIDs)}, nil
}

// UpdateAdvisorCheck updates an existing user-authored advisor check.
func (s *ChecksAPIService) UpdateAdvisorCheck(ctx context.Context, req *advisorsv1.UpdateAdvisorCheckRequest) (*advisorsv1.UpdateAdvisorCheckResponse, error) {
	if req.Check == nil {
		return nil, status.Error(codes.InvalidArgument, "Check is required.")
	}

	// a check cannot be renamed: its name is the primary key and is denormalized
	// into insight history, so reject a body name that disagrees with the path
	// instead of silently updating the check the path points at. An empty body
	// name means "not specified" and keeps working.
	if req.Check.Name != "" && req.Check.Name != req.Name {
		return nil, status.Errorf(codes.InvalidArgument,
			"Advisor check cannot be renamed: name '%s' in the request body does not match '%s'.",
			req.Check.Name, req.Name)
	}

	c := apiToAdvisorCheck(req.Check)
	c.Name = req.Name

	err := s.checksService.UpdateAdvisorCheck(ctx, c)
	if err != nil {
		return nil, err
	}

	updated, enabled, disabledServiceIDs, err := s.getCheck(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &advisorsv1.UpdateAdvisorCheckResponse{Check: advisorCheckToAPI(updated, enabled, disabledServiceIDs)}, nil
}

// DeleteAdvisorCheck deletes a user-authored advisor check.
func (s *ChecksAPIService) DeleteAdvisorCheck(ctx context.Context, req *advisorsv1.DeleteAdvisorCheckRequest) (*advisorsv1.DeleteAdvisorCheckResponse, error) {
	err := s.checksService.DeleteAdvisorCheck(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &advisorsv1.DeleteAdvisorCheckResponse{}, nil
}

// TestAdvisorCheck executes an advisor check definition against a single service
// without saving the check or persisting its results.
func (s *ChecksAPIService) TestAdvisorCheck(ctx context.Context, req *advisorsv1.TestAdvisorCheckRequest) (*advisorsv1.TestAdvisorCheckResponse, error) {
	if req.Check == nil {
		return nil, status.Error(codes.InvalidArgument, "Check is required.")
	}

	results, scriptOutput, err := s.checksService.TestAdvisorCheck(ctx, apiToAdvisorCheck(req.Check), req.ServiceId)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}
		// pass errors through with their message intact (incl. query and
		// script failures), so the check can be debugged from the UI
		return nil, err
	}

	return &advisorsv1.TestAdvisorCheckResponse{
		Results:      convertTestCheckResults(results),
		ScriptOutput: scriptOutput,
	}, nil
}

// ListAdvisorCheckTestTargets returns the services an advisor check of the given technology can be tested against.
func (s *ChecksAPIService) ListAdvisorCheckTestTargets(
	ctx context.Context,
	req *advisorsv1.ListAdvisorCheckTestTargetsRequest,
) (*advisorsv1.ListAdvisorCheckTestTargetsResponse, error) {
	targets, err := s.checksService.ListTestTargets(ctx, convertAPITechnology(req.Technology))
	if err != nil {
		return nil, err
	}

	res := make([]*advisorsv1.AdvisorCheckTestTarget, len(targets))
	for i, target := range targets {
		res[i] = &advisorsv1.AdvisorCheckTestTarget{
			ServiceId:   target.ServiceID,
			ServiceName: target.ServiceName,
		}
	}

	return &advisorsv1.ListAdvisorCheckTestTargetsResponse{Targets: res}, nil
}

// convertTestCheckResults converts test (dry-run) check execution results to their API representation.
func convertTestCheckResults(results []services.CheckResult) []*advisorsv1.TestAdvisorCheckResult {
	converted := make([]*advisorsv1.TestAdvisorCheckResult, 0, len(results))
	for _, result := range results {
		labels := make(map[string]string, len(result.Target.Labels)+len(result.Result.Labels))
		maps.Copy(labels, result.Result.Labels)
		maps.Copy(labels, result.Target.Labels)

		converted = append(converted, &advisorsv1.TestAdvisorCheckResult{
			Summary:     result.Result.Summary,
			CheckName:   result.CheckName,
			Description: result.Result.Description,
			ReadMoreUrl: result.Result.ReadMoreURL,
			Severity:    managementv1.Severity(result.Result.Severity), //nolint:gosec // severity is a bounded enum (0-8), no overflow
			Labels:      labels,
			ServiceName: result.Target.ServiceName,
			ServiceId:   result.Target.ServiceID,
		})
	}

	return converted
}

// getCheck returns a check by name together with its enabled state and the
// service IDs for which it is disabled.
func (s *ChecksAPIService) getCheck(ctx context.Context, name string) (check.Check, bool, []string, error) {
	checks, err := s.checksService.GetChecks()
	if err != nil {
		return check.Check{}, false, nil, fmt.Errorf("failed to get available checks list: %w", err)
	}

	c, ok := checks[name]
	if !ok {
		return check.Check{}, false, nil, status.Errorf(codes.NotFound, "Advisor check %q not found.", name)
	}

	disabled, err := s.checksService.GetDisabledChecks(ctx)
	if err != nil {
		return check.Check{}, false, nil, fmt.Errorf("failed to get disabled checks list: %w", err)
	}

	disServices, err := s.checksService.GetDisabledServicesForChecks(ctx)
	if err != nil {
		return check.Check{}, false, nil, fmt.Errorf("failed to get disabled services list: %w", err)
	}

	enabled := !slices.Contains(disabled, name)

	return c, enabled, disServices[name], nil
}

// convertInterval converts check.Interval type to advisorsv1.AdvisorCheckInterval.
func convertInterval(interval check.Interval) advisorsv1.AdvisorCheckInterval {
	switch interval {
	case check.Standard, "": // empty interval means standard
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD
	case check.Frequent:
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_FREQUENT
	case check.Rare:
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE
	default:
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED
	}
}

// convertModelInterval converts models.Interval type to advisorsv1.AdvisorCheckInterval.
func convertModelInterval(interval models.Interval) advisorsv1.AdvisorCheckInterval {
	switch interval {
	case models.Standard, "": // empty interval means standard
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD
	case models.Frequent:
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_FREQUENT
	case models.Rare:
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE
	default:
		return advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED
	}
}

// convertModelResultStatus converts models.CheckResultStatus to advisorsv1.AdvisorCheckResultStatus.
func convertModelResultStatus(status models.CheckResultStatus) advisorsv1.AdvisorCheckResultStatus {
	switch status {
	case models.CheckResultOK:
		return advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_OK
	case models.CheckResultFailed:
		return advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_FAILED
	case models.CheckResultError:
		return advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_ERROR
	default:
		return advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_UNSPECIFIED
	}
}

// convertAPIResultStatus converts advisorsv1.AdvisorCheckResultStatus to models.CheckResultStatus.
// An empty value is returned for an unspecified status, meaning "no filter".
func convertAPIResultStatus(status advisorsv1.AdvisorCheckResultStatus) models.CheckResultStatus {
	switch status {
	case advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_OK:
		return models.CheckResultOK
	case advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_FAILED:
		return models.CheckResultFailed
	case advisorsv1.AdvisorCheckResultStatus_ADVISOR_CHECK_RESULT_STATUS_ERROR:
		return models.CheckResultError
	default:
		return ""
	}
}

// convertModelTriggeredBy converts models.CheckTriggeredBy to advisorsv1.AdvisorCheckTriggeredBy.
func convertModelTriggeredBy(triggeredBy models.CheckTriggeredBy) advisorsv1.AdvisorCheckTriggeredBy {
	switch triggeredBy {
	case models.CheckTriggeredByUser:
		return advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_USER
	case models.CheckTriggeredByScheduler:
		return advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER
	default:
		return advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_UNSPECIFIED
	}
}

// convertSeverityCounts converts a run's per-severity finding counts, ordered
// most severe first so the API output is stable across calls.
func convertSeverityCounts(counts map[models.Severity]int) []*advisorsv1.SeverityCount {
	res := make([]*advisorsv1.SeverityCount, 0, len(counts))
	for severity, count := range counts {
		res = append(res, &advisorsv1.SeverityCount{
			Severity: managementv1.Severity(severity), //nolint:gosec // severity is a bounded enum (0-8), no overflow
			Count:    int32(count),                    //nolint:gosec
		})
	}
	// lower enum values are more severe, emergency being 1
	slices.SortFunc(res, func(a, b *advisorsv1.SeverityCount) int {
		return int(a.Severity) - int(b.Severity)
	})
	return res
}

// convertAPITriggeredBy converts advisorsv1.AdvisorCheckTriggeredBy to models.CheckTriggeredBy.
// An empty string is returned for unknown values.
func convertAPITriggeredBy(triggeredBy advisorsv1.AdvisorCheckTriggeredBy) models.CheckTriggeredBy {
	switch triggeredBy {
	case advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_USER:
		return models.CheckTriggeredByUser
	case advisorsv1.AdvisorCheckTriggeredBy_ADVISOR_CHECK_TRIGGERED_BY_SCHEDULER:
		return models.CheckTriggeredByScheduler
	default:
		return ""
	}
}

// convertTechnology converts check.Technology type to advisorsv1.AdvisorCheckTechnology.
func convertTechnology(technology check.Technology) advisorsv1.AdvisorCheckTechnology {
	switch technology {
	case check.MySQL:
		return advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL
	case check.PostgreSQL:
		return advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_POSTGRESQL
	case check.MongoDB:
		return advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MONGODB
	default:
		return advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_UNSPECIFIED
	}
}

// convertAPIInterval converts advisorsv1.AdvisorCheckInterval type to check.Interval.
func convertAPIInterval(interval advisorsv1.AdvisorCheckInterval) (check.Interval, error) {
	switch interval {
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD:
		return check.Standard, nil
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_FREQUENT:
		return check.Frequent, nil
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE:
		return check.Rare, nil
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED:
		return "", errors.New("invalid advisor check interval")
	default:
		return "", errors.New("unknown advisor check interval")
	}
}

// convertAPITechnology converts advisorsv1.AdvisorCheckTechnology to check.Technology.
// An unspecified technology maps to an empty technology, which fails check validation.
func convertAPITechnology(technology advisorsv1.AdvisorCheckTechnology) check.Technology {
	switch technology {
	case advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MYSQL:
		return check.MySQL
	case advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_POSTGRESQL:
		return check.PostgreSQL
	case advisorsv1.AdvisorCheckTechnology_ADVISOR_CHECK_TECHNOLOGY_MONGODB:
		return check.MongoDB
	default:
		return ""
	}
}

// convertAPIIntervalOptional converts advisorsv1.AdvisorCheckInterval to check.Interval.
// Unlike convertAPIInterval, an unspecified interval maps to an empty interval
// (treated as standard by the loader) rather than an error.
func convertAPIIntervalOptional(interval advisorsv1.AdvisorCheckInterval) check.Interval {
	switch interval {
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_STANDARD:
		return check.Standard
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_FREQUENT:
		return check.Frequent
	case advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_RARE:
		return check.Rare
	default:
		return ""
	}
}

// advisorCheckToAPI converts a check.Check into its full API representation, including queries and script.
func advisorCheckToAPI(c check.Check, enabled bool, disabledServiceIDs []string) *advisorsv1.AdvisorCheck {
	return &advisorsv1.AdvisorCheck{
		Name:               c.Name,
		Enabled:            enabled,
		Summary:            c.Summary,
		Description:        c.Description,
		Technology:         convertTechnology(c.Technology),
		Interval:           convertInterval(c.Interval),
		Category:           c.Category,
		Subcategory:        c.Subcategory,
		UserDefined:        c.UserDefined,
		Queries:            convertQueriesToAPI(c.Queries),
		Script:             c.Script,
		DisabledServiceIds: disabledServiceIDs,
	}
}

// apiToAdvisorCheck converts an API advisor check (from a create/update request) into a check.Check.
func apiToAdvisorCheck(c *advisorsv1.AdvisorCheck) check.Check {
	return check.Check{
		Name:        c.Name,
		Summary:     c.Summary,
		Description: c.Description,
		Category:    c.Category,
		Subcategory: c.Subcategory,
		Technology:  convertAPITechnology(c.Technology),
		Interval:    convertAPIIntervalOptional(c.Interval),
		Queries:     convertAPIQueries(c.Queries),
		Script:      c.Script,
	}
}

// convertQueriesToAPI converts check queries to their API representation.
func convertQueriesToAPI(queries []check.Query) []*advisorsv1.AdvisorCheckQuery {
	if len(queries) == 0 {
		return nil
	}

	res := make([]*advisorsv1.AdvisorCheckQuery, 0, len(queries))
	for _, q := range queries {
		var params map[string]string
		if len(q.Parameters) != 0 {
			params = make(map[string]string, len(q.Parameters))
			for k, v := range q.Parameters {
				params[string(k)] = v
			}
		}

		res = append(res, &advisorsv1.AdvisorCheckQuery{
			Type:       string(q.Type),
			Query:      q.Query,
			Parameters: params,
		})
	}

	return res
}

// convertAPIQueries converts API queries into check queries.
func convertAPIQueries(queries []*advisorsv1.AdvisorCheckQuery) []check.Query {
	if len(queries) == 0 {
		return nil
	}

	res := make([]check.Query, 0, len(queries))
	for _, q := range queries {
		if q == nil {
			continue
		}

		var params map[check.Parameter]string
		if len(q.Parameters) != 0 {
			params = make(map[check.Parameter]string, len(q.Parameters))
			for k, v := range q.Parameters {
				params[check.Parameter(k)] = v
			}
		}

		res = append(res, check.Query{
			Type:       check.Type(q.Type),
			Query:      q.Query,
			Parameters: params,
		})
	}

	return res
}
