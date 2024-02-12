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

	"github.com/percona-platform/saas/pkg/check"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/api/managementpb"
	"github.com/percona/pmm/managed/services"
)

// ChecksAPIService represents security checks service API.
type ChecksAPIService struct {
	checksService checksService
	l             *logrus.Entry

	managementpb.UnimplementedSecurityChecksServer
}

// NewChecksAPIService creates new Checks API Service.
func NewChecksAPIService(checksService checksService) *ChecksAPIService {
	return &ChecksAPIService{
		checksService: checksService,
		l:             logrus.WithField("component", "management/checks"),
	}
}

// ListFailedServices returns a list of services with failed checks and their summaries.
func (s *ChecksAPIService) ListFailedServices(_ context.Context, _ *managementpb.ListFailedServicesRequest) (*managementpb.ListFailedServicesResponse, error) {
	results, err := s.checksService.GetSecurityCheckResults()
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

	failedServices := make([]*managementpb.CheckResultSummary, 0, len(summaries))
	for _, result := range summaries {
		failedServices = append(failedServices, &managementpb.CheckResultSummary{
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

	return &managementpb.ListFailedServicesResponse{Result: failedServices}, nil
}

// GetFailedChecks returns details of failed checks for a given service.
func (s *ChecksAPIService) GetFailedChecks(ctx context.Context, req *managementpb.GetFailedChecksRequest) (*managementpb.GetFailedChecksResponse, error) {
	results, err := s.checksService.GetChecksResults(ctx, req.ServiceId)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrapf(err, "failed to get check results for service '%s'", req.ServiceId)
	}

	failedChecks := make([]*managementpb.CheckResult, 0, len(results))
	for _, result := range results {
		labels := make(map[string]string, len(result.Target.Labels)+len(result.Result.Labels))
		for k, v := range result.Result.Labels {
			labels[k] = v
		}
		for k, v := range result.Target.Labels {
			labels[k] = v
		}

		failedChecks = append(failedChecks, &managementpb.CheckResult{
			Summary:     result.Result.Summary,
			CheckName:   result.CheckName,
			Description: result.Result.Description,
			ReadMoreUrl: result.Result.ReadMoreURL,
			Severity:    managementpb.Severity(result.Result.Severity),
			Labels:      labels,
			ServiceName: result.Target.ServiceName,
			ServiceId:   result.Target.ServiceID,
		})
	}

	pageTotals := &managementpb.PageTotals{
		TotalPages: 1,
		TotalItems: int32(len(failedChecks)),
	}
	var pageIndex int
	var pageSize int
	if req.PageParams != nil {
		pageIndex = int(req.PageParams.Index)
		pageSize = int(req.PageParams.PageSize)
	}

	from, to := pageIndex*pageSize, (pageIndex+1)*pageSize
	if to > len(failedChecks) || to == 0 {
		to = len(failedChecks)
	}
	if from > len(failedChecks) {
		from = len(failedChecks)
	}

	if pageSize > 0 {
		pageTotals.TotalPages = int32(len(failedChecks) / pageSize)
		if len(failedChecks)%pageSize > 0 {
			pageTotals.TotalPages++
		}
	}

	return &managementpb.GetFailedChecksResponse{Results: failedChecks[from:to], PageTotals: pageTotals}, nil
}

// ToggleCheckAlert toggles the silence state of the check with the provided alertID.
func (s *ChecksAPIService) ToggleCheckAlert(ctx context.Context, req *managementpb.ToggleCheckAlertRequest) (*managementpb.ToggleCheckAlertResponse, error) { //nolint:revive,lll
	return nil, status.Error(codes.NotFound, "Advisor alerts silencing is not supported anymore.")
}

// GetSecurityCheckResults returns Security Thread Tool's latest checks results.
func (s *ChecksAPIService) GetSecurityCheckResults(_ context.Context, _ *managementpb.GetSecurityCheckResultsRequest) (*managementpb.GetSecurityCheckResultsResponse, error) { //nolint:staticcheck,lll
	results, err := s.checksService.GetSecurityCheckResults()
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrap(err, "failed to get security check results")
	}

	checkResults := make([]*managementpb.SecurityCheckResult, 0, len(results))
	for _, result := range results {
		checkResults = append(checkResults, &managementpb.SecurityCheckResult{
			Summary:     result.Result.Summary,
			Description: result.Result.Description,
			ReadMoreUrl: result.Result.ReadMoreURL,
			Severity:    managementpb.Severity(result.Result.Severity),
			Labels:      result.Result.Labels,
			ServiceName: result.Target.ServiceName,
		})
	}

	return &managementpb.GetSecurityCheckResultsResponse{Results: checkResults}, nil //nolint:staticcheck
}

// StartSecurityChecks executes Security Thread Tool checks and returns when all checks are executed.
func (s *ChecksAPIService) StartSecurityChecks(_ context.Context, req *managementpb.StartSecurityChecksRequest) (*managementpb.StartSecurityChecksResponse, error) {
	// Start only specified checks from any group.
	err := s.checksService.StartChecks(req.Names)
	if err != nil {
		if errors.Is(err, services.ErrAdvisorsDisabled) {
			return nil, status.Errorf(codes.FailedPrecondition, "%v.", err)
		}

		return nil, errors.Wrap(err, "failed to start security checks")
	}

	return &managementpb.StartSecurityChecksResponse{}, nil
}

// ListSecurityChecks returns a list of available advisor checks and their statuses.
func (s *ChecksAPIService) ListSecurityChecks(_ context.Context, _ *managementpb.ListSecurityChecksRequest) (*managementpb.ListSecurityChecksResponse, error) {
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

	res := make([]*managementpb.SecurityCheck, 0, len(checks))
	for _, c := range checks {
		_, disabled := m[c.Name]
		res = append(res, &managementpb.SecurityCheck{
			Name:        c.Name,
			Disabled:    disabled,
			Summary:     c.Summary,
			Family:      convertFamily(c.GetFamily()),
			Description: c.Description,
			Interval:    convertInterval(c.Interval),
		})
	}

	return &managementpb.ListSecurityChecksResponse{Checks: res}, nil
}

// ListAdvisors retrieves a list of advisors based on the provided request.
func (s *ChecksAPIService) ListAdvisors(_ context.Context, _ *managementpb.ListAdvisorsRequest) (*managementpb.ListAdvisorsResponse, error) {
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

	res := make([]*managementpb.Advisor, 0, len(advisors))
	for _, a := range advisors {
		checks := make([]*managementpb.SecurityCheck, 0, len(a.Checks))
		for _, c := range a.Checks {
			_, disabled := m[c.Name]
			checks = append(checks, &managementpb.SecurityCheck{
				Name:        c.Name,
				Disabled:    disabled,
				Summary:     c.Summary,
				Family:      convertFamily(c.GetFamily()),
				Description: c.Description,
				Interval:    convertInterval(c.Interval),
			})
		}

		res = append(res, &managementpb.Advisor{
			Name:        a.Name,
			Description: a.Description,
			Summary:     a.Summary,
			Comment:     createComment(a.Checks),
			Category:    a.Category,
			Checks:      checks,
		})
	}

	return &managementpb.ListAdvisorsResponse{Advisors: res}, nil
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

// ChangeSecurityChecks enables/disables Security Thread Tool checks by names or changes its execution interval.
func (s *ChecksAPIService) ChangeSecurityChecks(_ context.Context, req *managementpb.ChangeSecurityChecksRequest) (*managementpb.ChangeSecurityChecksResponse, error) {
	var enableChecks, disableChecks []string
	changeIntervalParams := make(map[string]check.Interval)
	for _, check := range req.Params {
		if check.Enable && check.Disable {
			return nil, status.Errorf(codes.InvalidArgument, "Check %s has enable and disable parameters set to the true.", check.Name)
		}

		if check.Interval != managementpb.SecurityCheckInterval_SECURITY_CHECK_INTERVAL_INVALID {
			interval, err := convertAPIInterval(check.Interval)
			if err != nil {
				return nil, errors.Wrap(err, "failed to change security check interval")
			}
			changeIntervalParams[check.Name] = interval
		}

		if check.Enable {
			enableChecks = append(enableChecks, check.Name)
		}

		if check.Disable {
			disableChecks = append(disableChecks, check.Name)
		}
	}

	if len(changeIntervalParams) != 0 {
		err := s.checksService.ChangeInterval(changeIntervalParams)
		if err != nil {
			return nil, errors.Wrap(err, "failed to change security check interval")
		}
	}

	err := s.checksService.EnableChecks(enableChecks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to enable disabled security checks")
	}

	err = s.checksService.DisableChecks(disableChecks)
	if err != nil {
		return nil, errors.Wrap(err, "failed to disable security checks")
	}

	return &managementpb.ChangeSecurityChecksResponse{}, nil
}

// convertInterval converts check.Interval type to managementpb.SecurityCheckInterval.
func convertInterval(interval check.Interval) managementpb.SecurityCheckInterval {
	switch interval {
	case check.Standard, "": // empty interval means standard
		return managementpb.SecurityCheckInterval_STANDARD
	case check.Frequent:
		return managementpb.SecurityCheckInterval_FREQUENT
	case check.Rare:
		return managementpb.SecurityCheckInterval_RARE
	default:
		return managementpb.SecurityCheckInterval_SECURITY_CHECK_INTERVAL_INVALID
	}
}

// convertFamily converts check.Family type to managementpb.AdvisorCheckFamily.
func convertFamily(family check.Family) managementpb.AdvisorCheckFamily {
	switch family {
	case check.MySQL:
		return managementpb.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MYSQL
	case check.PostgreSQL:
		return managementpb.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_POSTGRESQL
	case check.MongoDB:
		return managementpb.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_MONGODB
	default:
		return managementpb.AdvisorCheckFamily_ADVISOR_CHECK_FAMILY_INVALID
	}
}

// convertAPIInterval converts managementpb.SecurityCheckInterval type to check.Interval.
func convertAPIInterval(interval managementpb.SecurityCheckInterval) (check.Interval, error) {
	switch interval {
	case managementpb.SecurityCheckInterval_STANDARD:
		return check.Standard, nil
	case managementpb.SecurityCheckInterval_FREQUENT:
		return check.Frequent, nil
	case managementpb.SecurityCheckInterval_RARE:
		return check.Rare, nil
	case managementpb.SecurityCheckInterval_SECURITY_CHECK_INTERVAL_INVALID:
		return check.Interval(""), errors.New("invalid security check interval")
	default:
		return check.Interval(""), errors.New("unknown security check interval")
	}
}
