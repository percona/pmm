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

// Package ia contains Integrated Alerting APIs implementations.
package ia

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/percona-platform/saas/pkg/common"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// AlertsService represents integrated alerting alerts API.
// Deprecated. Do not use.
type AlertsService struct {
	db               *reform.DB
	l                *logrus.Entry
	alertManager     alertManager
	templatesService templatesService

	iav1beta1.UnimplementedAlertsServer
}

// NewAlertsService creates new alerts API service.
func NewAlertsService(db *reform.DB, alertManager alertManager, templatesService templatesService) *AlertsService {
	return &AlertsService{
		l:                logrus.WithField("component", "management/ia/alerts"),
		db:               db,
		alertManager:     alertManager,
		templatesService: templatesService,
	}
}

// Enabled returns if service is enabled and can be used.
// Deprecated. Do not use.
func (s *AlertsService) Enabled() bool {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		s.l.WithError(err).Error("can't get settings")
		return false
	}
	return !settings.Alerting.Disabled
}

// ListAlerts returns list of existing alerts.
// Deprecated. Do not use.
func (s *AlertsService) ListAlerts(ctx context.Context, req *iav1beta1.ListAlertsRequest) (*iav1beta1.ListAlertsResponse, error) { //nolint:staticcheck
	filter := &services.FilterParams{
		IsIA: true,
	}
	alerts, err := s.alertManager.GetAlerts(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get alerts from alertmanager")
	}

	var res []*iav1beta1.Alert //nolint:prealloc,staticcheck
	for _, alert := range alerts {
		updatedAt := timestamppb.New(time.Time(*alert.UpdatedAt))
		if err := updatedAt.CheckValid(); err != nil {
			return nil, errors.Wrap(err, "failed to convert timestamp")
		}

		createdAt := timestamppb.New(time.Time(*alert.StartsAt))
		if err := updatedAt.CheckValid(); err != nil {
			return nil, errors.Wrap(err, "failed to convert timestamp")
		}

		st := iav1beta1.Status_STATUS_INVALID
		if *alert.Status.State == "active" {
			st = iav1beta1.Status_TRIGGERING
		}

		if len(alert.Status.SilencedBy) != 0 {
			st = iav1beta1.Status_SILENCED
		}

		var rule *iav1beta1.Rule //nolint:staticcheck
		// Rules files created by user in directory /srv/prometheus/rules/ doesn't have associated rules in DB.
		// So alertname field will be empty or will keep invalid value. Don't fill rule field in that case.
		ruleID, ok := alert.Labels["alertname"]
		if ok && strings.HasPrefix(ruleID, "/rule_id/") {
			var r *models.Rule
			var channels []*models.Channel
			e := s.db.InTransaction(func(tx *reform.TX) error {
				var err error
				r, err = models.FindRuleByID(tx.Querier, ruleID)
				if err != nil {
					return err
				}

				channels, err = models.FindChannelsByIDs(tx.Querier, r.ChannelIDs)
				return err
			})
			if e != nil {
				// The codes.NotFound code can be returned just only by the FindRulesByID func
				// from the transaction above.
				if st, ok := status.FromError(e); ok && st.Code() == codes.NotFound {
					s.l.Warnf("The related alert rule was most likely removed: %s", st.Message())
					continue
				}

				return nil, e
			}

			rule, err = convertRule(s.l, r, channels)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert alert rule")
			}
		}

		if rule != nil && len(rule.Filters) != 0 {
			pass, err := satisfiesFilters(alert, rule.Filters)
			if err != nil {
				return nil, err
			}

			if !pass {
				continue
			}
		}

		res = append(res, &iav1beta1.Alert{ //nolint:staticcheck
			AlertId:   getAlertID(alert),
			Summary:   alert.Annotations["summary"],
			Severity:  managementpb.Severity(common.ParseSeverity(alert.Labels["severity"])),
			Status:    st,
			Labels:    alert.Labels,
			Rule:      rule,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	pageTotals := &managementpb.PageTotals{
		TotalPages: 1,
	}

	var pageIndex int
	var pageSize int
	if req.PageParams != nil {
		pageIndex = int(req.PageParams.Index)
		pageSize = int(req.PageParams.PageSize)
	}

	from, to := pageIndex*pageSize, (pageIndex+1)*pageSize
	if to > len(res) || to == 0 {
		to = len(res)
	}

	if from > len(res) {
		from = len(res)
	}

	if pageSize > 0 {
		pageTotals.TotalPages = int32(len(res) / pageSize)
		if len(res)%pageSize > 0 {
			pageTotals.TotalPages++
		}
	}
	pageTotals.TotalItems = int32(len(res))

	return &iav1beta1.ListAlertsResponse{Alerts: res[from:to], Totals: pageTotals}, nil //nolint:staticcheck
}

// satisfiesFilters checks that alert passes filters, returns true in case of success.
func satisfiesFilters(alert *ammodels.GettableAlert, filters []*iav1beta1.Filter) (bool, error) { //nolint:staticcheck
	for _, filter := range filters {
		value, ok := alert.Labels[filter.Key]
		if !ok {
			return false, nil
		}

		switch filter.Type {
		case iav1beta1.FilterType_EQUAL:
			if filter.Value != value {
				return false, nil
			}
		case iav1beta1.FilterType_REGEX:
			match, err := regexp.MatchString(filter.Value, value)
			if err != nil {
				return false, status.Errorf(codes.InvalidArgument, "bad regular expression: +%v", err)
			}

			if !match {
				return false, nil
			}
		case iav1beta1.FilterType_FILTER_TYPE_INVALID:
			fallthrough
		default:
			return false, status.Error(codes.Internal, "Unexpected filter type.")
		}
	}

	return true, nil
}

// getAlertID returns the alert's ID.
// Deprecated. Do not use.
func getAlertID(alert *ammodels.GettableAlert) string {
	return *alert.Fingerprint
}

// ToggleAlerts allows to silence/unsilence specified alerts.
// Deprecated. Do not use.
func (s *AlertsService) ToggleAlerts(ctx context.Context, req *iav1beta1.ToggleAlertsRequest) (*iav1beta1.ToggleAlertsResponse, error) { //nolint:staticcheck
	var err error
	var alerts []*ammodels.GettableAlert

	filters := &services.FilterParams{
		IsIA: true,
	}
	if len(req.AlertIds) == 0 {
		alerts, err = s.alertManager.GetAlerts(ctx, filters)
	} else {
		alerts, err = s.alertManager.FindAlertsByID(ctx, filters, req.AlertIds)
	}
	if err != nil {
		return nil, err
	}

	switch req.Silenced {
	case managementpb.BooleanFlag_DO_NOT_CHANGE:
		// nothing
	case managementpb.BooleanFlag_TRUE:
		err = s.alertManager.SilenceAlerts(ctx, alerts)
	case managementpb.BooleanFlag_FALSE:
		err = s.alertManager.UnsilenceAlerts(ctx, alerts)
	}
	if err != nil {
		return nil, err
	}

	return &iav1beta1.ToggleAlertsResponse{}, nil //nolint:staticcheck
}

// Check interfaces.
var (
	_ iav1beta1.AlertsServer = (*AlertsService)(nil)
)
