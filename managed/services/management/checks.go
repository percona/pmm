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
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/percona/saas/pkg/check"
	"github.com/percona/saas/pkg/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	advisorsv1 "github.com/percona/pmm/api/advisors/v1"
	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/services"
)

// ChecksAPIService represents advisor service API.
type ChecksAPIService struct {
	checksService checksService
	l             *logrus.Entry

	advisorsv1.UnimplementedAdvisorServiceServer
}

// NewChecksAPIService creates new Checks API Service.
func NewChecksAPIService(checksService checksService) *ChecksAPIService {
	return &ChecksAPIService{
		checksService: checksService,
		l:             logrus.WithField("component", "management/checks"),
	}
}

// ListFailedServices returns a list of services with failed checks and their summaries.
func (s *ChecksAPIService) ListFailedServices(ctx context.Context, _ *advisorsv1.ListFailedServicesRequest) (*advisorsv1.ListFailedServicesResponse, error) {
	results, err := s.checksService.GetChecksResults(ctx, "")
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrap(err, "failed to get check results")
	}

	summaries := make(map[string]*services.CheckResultSummary)
	var svcSummary *services.CheckResultSummary
	var exists bool
	for _, result := range results {
		if svcSummary, exists = summaries[result.Target.ServiceID]; !exists {
			svcSummary = &services.CheckResultSummary{
				ServiceName: result.Target.ServiceName,
				ServiceID:   result.Target.ServiceID,
			}
			summaries[result.Target.ServiceID] = svcSummary
		}
		switch result.Result.Severity {
		case common.Emergency:
			svcSummary.EmergencyCount++
		case common.Alert:
			svcSummary.AlertCount++
		case common.Critical:
			svcSummary.CriticalCount++
		case common.Error:
			svcSummary.ErrorCount++
		case common.Warning:
			svcSummary.WarningCount++
		case common.Notice:
			svcSummary.NoticeCount++
		case common.Info:
			svcSummary.InfoCount++
		case common.Debug:
			svcSummary.DebugCount++
		case common.Unknown:
			continue
		}
	}

	failedServices := make([]*advisorsv1.CheckResultSummary, 0, len(summaries))
	for _, result := range summaries {
		failedServices = append(failedServices, &advisorsv1.CheckResultSummary{
			ServiceId:      result.ServiceID,
			ServiceName:    result.ServiceName,
			EmergencyCount: result.EmergencyCount,
			AlertCount:     result.AlertCount,
			CriticalCount:  result.CriticalCount,
			ErrorCount:     result.ErrorCount,
			WarningCount:   result.WarningCount,
			NoticeCount:    result.NoticeCount,
			InfoCount:      result.InfoCount,
			DebugCount:     result.DebugCount,
		})
	}

	return &advisorsv1.ListFailedServicesResponse{Result: failedServices}, nil
}

// GetFailedChecks returns details of failed checks for a given service.
func (s *ChecksAPIService) GetFailedChecks(ctx context.Context, req *advisorsv1.GetFailedChecksRequest) (*advisorsv1.GetFailedChecksResponse, error) {
	results, err := s.checksService.GetChecksResults(ctx, req.ServiceId)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrapf(err, "failed to get check results for service '%s'", req.ServiceId)
	}

	failedChecks := make([]*advisorsv1.CheckResult, 0, len(results))
	for _, result := range results {
		labels := make(map[string]string, len(result.Target.Labels)+len(result.Result.Labels))
		for k, v := range result.Result.Labels {
			labels[k] = v
		}
		for k, v := range result.Target.Labels {
			labels[k] = v
		}

		failedChecks = append(failedChecks, &advisorsv1.CheckResult{
			Summary:     result.Result.Summary,
			CheckName:   result.CheckName,
			Description: result.Result.Description,
			ReadMoreUrl: result.Result.ReadMoreURL,
			Severity:    managementv1.Severity(result.Result.Severity),
			Labels:      labels,
			ServiceName: result.Target.ServiceName,
			ServiceId:   result.Target.ServiceID,
		})
	}

	var pageIndex, pageSize int
	totalPages := int32(1)
	totalItems := int32(len(failedChecks))

	if req.PageIndex != nil {
		pageIndex = int(pointer.GetInt32(req.PageIndex))
	}
	if req.PageSize != nil {
		pageSize = int(pointer.GetInt32(req.PageSize))
	}

	from, to := pageIndex*pageSize, (pageIndex+1)*pageSize
	if to > len(failedChecks) || to == 0 {
		to = len(failedChecks)
	}
	if from > len(failedChecks) {
		from = len(failedChecks)
	}

	if pageSize > 0 {
		totalPages = int32(len(failedChecks) / pageSize)
		if len(failedChecks)%pageSize > 0 {
			totalPages++
		}
	}

	return &advisorsv1.GetFailedChecksResponse{
		Results:    failedChecks[from:to],
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

// StartAdvisorChecks executes advisor checks and returns when all checks are executed.
func (s *ChecksAPIService) StartAdvisorChecks(_ context.Context, req *advisorsv1.StartAdvisorChecksRequest) (*advisorsv1.StartAdvisorChecksResponse, error) {
	// Start only specified checks from any group.
	err := s.checksService.StartChecks(req.Names)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrap(err, "failed to start advisor checks")
	}

	return &advisorsv1.StartAdvisorChecksResponse{}, nil
}

// ListAdvisorChecks returns a list of available advisor checks and their statuses.
func (s *ChecksAPIService) ListAdvisorChecks(_ context.Context, _ *advisorsv1.ListAdvisorChecksRequest) (*advisorsv1.ListAdvisorChecksResponse, error) {
	disChecks, err := s.checksService.GetDisabledChecks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get disabled checks list")
	}

	m := make(map[string]struct{}, len(disChecks))
	for _, c := range disChecks {
		m[c] = struct{}{}
	}

	checks, err := s.checksService.GetChecks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available checks list")
	}

	res := make([]*advisorsv1.AdvisorCheck, 0, len(checks))
	for _, c := range checks {
		_, disabled := m[c.Name]
		res = append(res, &advisorsv1.AdvisorCheck{
			Name:        c.Name,
			Enabled:     !disabled,
			Summary:     c.Summary,
			Family:      convertFamily(c.GetFamily()),
			Description: c.Description,
			Interval:    convertInterval(c.Interval),
		})
	}

	return &advisorsv1.ListAdvisorChecksResponse{Checks: res}, nil
}

// ListAdvisors retrieves a list of advisors based on the provided request.
func (s *ChecksAPIService) ListAdvisors(_ context.Context, _ *advisorsv1.ListAdvisorsRequest) (*advisorsv1.ListAdvisorsResponse, error) {
	disChecks, err := s.checksService.GetDisabledChecks()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get disabled checks list")
	}

	m := make(map[string]struct{}, len(disChecks))
	for _, c := range disChecks {
		m[c] = struct{}{}
	}

	advisors, err := s.checksService.GetAdvisors()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available checks list")
	}

	res := make([]*advisorsv1.Advisor, 0, len(advisors))
	for _, a := range advisors {
		checks := make([]*advisorsv1.AdvisorCheck, 0, len(a.Checks))
		for _, c := range a.Checks {
			_, disabled := m[c.Name]
			checks = append(checks, &advisorsv1.AdvisorCheck{
				Name:        c.Name,
				Enabled:     !disabled,
				Summary:     c.Summary,
				Family:      convertFamily(c.GetFamily()),
				Description: c.Description,
				Interval:    convertInterval(c.Interval),
			})
		}

		res = append(res, &advisorsv1.Advisor{
			Name:        a.Name,
			Description: a.Description,
			Summary:     a.Summary,
			Comment:     createComment(a.Checks),
			Category:    a.Category,
			Checks:      checks,
		})
	}

	return &advisorsv1.ListAdvisorsResponse{Advisors: res}, nil
}

func createComment(checks []check.Check) string {
	var mySQL, postgreSQL, mongoDB bool
	for _, c := range checks {
		switch c.GetFamily() {
		case check.MySQL:
			mySQL = true
		case check.PostgreSQL:
			postgreSQL = true
		case check.MongoDB:
			mongoDB = true
		}
	}

	b := make([]string, 0, 3)
	if mySQL {
		b = append(b, "MySQL")
	}
	if postgreSQL {
		b = append(b, "PostgreSQL")
	}
	if mongoDB {
		b = append(b, "MongoDB")
	}

	if len(b) == 3 {
		return "All technologies supported"
	}

	return "Partial support (" + strings.Join(b, ", ") + ")"
}

// ChangeAdvisorChecks enables/disables advisor checks by names or changes its execution interval.
func (s *ChecksAPIService) ChangeAdvisorChecks(_ context.Context, req *advisorsv1.ChangeAdvisorChecksRequest) (*advisorsv1.ChangeAdvisorChecksResponse, error) {
	var enableChecks, disableChecks []string
	changeIntervalParams := make(map[string]check.Interval)
	for _, check := range req.Params {
		if check.Interval != advisorsv1.AdvisorCheckInterval_ADVISOR_CHECK_INTERVAL_UNSPECIFIED {
			interval, err := convertAPIInterval(check.Interval)
			if err != nil {
				return nil, err
			}
			changeIntervalParams[check.Name] = interval
		}

		if check.Enable != nil {
			if *check.Enable {
				enableChecks = append(enableChecks, check.Name)
			} else {
				disableChecks = append(disableChecks, check.Name)
			}
		}
	}

	if len(changeIntervalParams) != 0 {
		err := s.checksService.ChangeInterval(changeIntervalParams)
		if err != nil {
			return nil, errors.Wrap(err, "failed to change advisor check interval")
		}
	}

	err := s.checksService.EnableChecks(enableChecks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enable disabled advisor checks")
	}

	err = s.checksService.DisableChecks(disableChecks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to disable advisor checks")
	}

	return &advisorsv1.ChangeAdvisorChecksResponse{}, nil
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

// convertFamily converts check.Family type to advisorsv1.AdvisorCheckFamily.
func convertFamily(family check.Family) advisorsv1.AdvisorCheckFamily {
	switch family {
	case check.MySQL:
		return advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MYSQL
	case check.PostgreSQL:
		return advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_POSTGRESQL
	case check.MongoDB:
		return advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MONGODB
	default:
		return advisorsv1.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_UNSPECIFIED
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
