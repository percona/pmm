// pmm-managed
// Copyright (C) 2017 Percona LLC
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
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona-platform/saas/pkg/common"
	"github.com/percona/pmm/api/alertmanager/ammodels"
	"github.com/percona/pmm/api/managementpb"
	iav1beta1 "github.com/percona/pmm/api/managementpb/ia"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services"
)

// AlertsService represents integrated alerting alerts API.
type AlertsService struct {
	db               *reform.DB
	l                *logrus.Entry
	alertManager     alertManager
	templatesService *TemplatesService
}

// NewAlertsService creates new alerts API service.
func NewAlertsService(db *reform.DB, alertManager alertManager, templatesService *TemplatesService) *AlertsService {
	return &AlertsService{
		l:                logrus.WithField("component", "management/ia/alerts"),
		db:               db,
		alertManager:     alertManager,
		templatesService: templatesService,
	}
}

// ListAlerts returns list of existing alerts.
func (s *AlertsService) ListAlerts(ctx context.Context, req *iav1beta1.ListAlertsRequest) (*iav1beta1.ListAlertsResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IntegratedAlerting.Enabled {
		return nil, status.Errorf(codes.FailedPrecondition, "%v.", services.ErrAlertingDisabled)
	}

	alerts, err := s.alertManager.GetAlerts(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get alerts form alertmanager")
	}

	res := make([]*iav1beta1.Alert, 0, len(alerts))
	for _, alert := range alerts {

		if _, ok := alert.Labels["ia"]; !ok { // Skip non-IA alerts
			continue
		}

		updatedAt, err := ptypes.TimestampProto(time.Time(*alert.UpdatedAt))
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert timestamp")
		}

		createdAt, err := ptypes.TimestampProto(time.Time(*alert.StartsAt))
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert timestamp")
		}

		st := iav1beta1.Status_STATUS_INVALID
		if *alert.Status.State == "active" {
			st = iav1beta1.Status_TRIGGERING
		}

		if len(alert.Status.SilencedBy) != 0 {
			st = iav1beta1.Status_SILENCED
		}

		var rule *iav1beta1.Rule
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
				return nil, e
			}

			template, ok := s.templatesService.getTemplates()[r.TemplateName]
			if !ok {
				return nil, status.Errorf(codes.NotFound, "Failed to find template with name: %s", r.TemplateName)
			}

			rule, err = convertRule(s.l, r, template, channels)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to convert alert rule")
			}
		}

		res = append(res, &iav1beta1.Alert{
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

	return &iav1beta1.ListAlertsResponse{Alerts: res}, nil
}

func getAlertID(alert *ammodels.GettableAlert) string {
	return *alert.Fingerprint
}

// ToggleAlert allows to silence/unsilence specified alerts.
func (s *AlertsService) ToggleAlert(ctx context.Context, req *iav1beta1.ToggleAlertRequest) (*iav1beta1.ToggleAlertResponse, error) {
	settings, err := models.GetSettings(s.db)
	if err != nil {
		return nil, err
	}

	if !settings.IntegratedAlerting.Enabled {
		return nil, status.Errorf(codes.FailedPrecondition, "%v.", services.ErrAlertingDisabled)
	}

	switch req.Silenced {
	case iav1beta1.BooleanFlag_DO_NOT_CHANGE:
		// nothing
	case iav1beta1.BooleanFlag_TRUE:
		err := s.alertManager.Silence(ctx, req.AlertId)
		if err != nil {
			return nil, err
		}
	case iav1beta1.BooleanFlag_FALSE:
		err := s.alertManager.Unsilence(ctx, req.AlertId)
		if err != nil {
			return nil, err
		}
	}

	return &iav1beta1.ToggleAlertResponse{}, nil
}

// Check interfaces.
var (
	_ iav1beta1.AlertsServer = (*AlertsService)(nil)
)
