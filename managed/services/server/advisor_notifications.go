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

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
)

const (
	advisorNotificationsFolderUID    = "pmm-advisor-notifications"
	advisorNotificationsFolderTitle  = "PMM Advisor Notifications"
	advisorNotificationsRuleGroup    = "pmm_advisor_notifications"
	advisorNotificationsRuleTitle    = "Advisor check failing"
	advisorNotificationsRuleInterval = "1m"
	advisorNotificationsRuleFor      = "5m"
	advisorInsightsMetric            = "pmm_managed_advisor_check_insights"
	// Look-back window (seconds) for the instant query.
	advisorNotificationsQueryRangeSeconds = 600
	// Added to fired alerts so operators can route them via a Grafana notification policy.
	advisorNotificationLabel = "advisor_notification"
)

// advisorNotificationSeverityRegex builds a label-value regex matching every severity at least as
// severe as the given threshold (e.g. Error -> "emergency|alert|critical|error").
func advisorNotificationSeverityRegex(threshold common.Severity) string {
	if threshold == common.Unknown {
		threshold = common.Error
	}

	var names []string
	for s := common.Emergency; s <= threshold; s++ {
		names = append(names, s.String())
	}
	return strings.Join(names, "|")
}

// buildAdvisorNotificationRule builds the Grafana alert rule that fires for advisor insights at or
// above the configured severity threshold.
func buildAdvisorNotificationRule(datasourceUID string, threshold common.Severity) *services.Rule {
	expr := fmt.Sprintf(`%s{severity=~"%s"} > 0`, advisorInsightsMetric, advisorNotificationSeverityRegex(threshold))

	return &services.Rule{
		GrafanaAlert: services.GrafanaAlert{
			Title:        advisorNotificationsRuleTitle,
			Condition:    "A",
			NoDataState:  "OK",
			ExecErrState: "Alerting",
			Data: []services.Data{
				{
					RefID:             "A",
					DatasourceUID:     datasourceUID,
					RelativeTimeRange: services.RelativeTimeRange{From: advisorNotificationsQueryRangeSeconds, To: 0},
					Model: services.Model{
						RefID:   "A",
						Expr:    expr,
						Instant: true,
					},
				},
			},
		},
		For: advisorNotificationsRuleFor,
		Annotations: map[string]string{
			"summary":     "Advisor check {{ $labels.check_name }} is failing on service {{ $labels.service_name }}",
			"description": "Advisor {{ $labels.advisor }} check {{ $labels.check_name }} reported a {{ $labels.severity }} issue on service {{ $labels.service_name }}.",
		},
		Labels: map[string]string{
			advisorNotificationLabel: "1",
		},
	}
}

// reconcileAdvisorNotifications creates or removes the Grafana alert rule that drives advisor email
// notifications, based on the current settings.
//
// It talks to Grafana on the caller's behalf, so it must be called within an authenticated request
// context (e.g. ChangeSettings) - background contexts have no Grafana credentials.
func (s *Server) reconcileAdvisorNotifications(ctx context.Context, settings *models.Settings) error {
	if !settings.IsAdvisorNotificationsEnabled() {
		err := s.grafanaClient.DeleteAlertRuleGroup(ctx, advisorNotificationsFolderUID, advisorNotificationsRuleGroup)
		if err != nil {
			s.l.Debugf("Failed to delete advisor notifications rule group (ignored): %v", err)
		}
		return nil
	}

	// Best-effort folder creation; an error here usually means the folder already exists. A genuine
	// problem (auth, connectivity) surfaces below when creating the rule.
	err := s.grafanaClient.CreateFolderWithUID(ctx, advisorNotificationsFolderTitle, advisorNotificationsFolderUID)
	if err != nil {
		s.l.Debugf("Failed to create advisor notifications folder (ignored, may already exist): %v", err)
	}

	// Remove any existing rule so we always reflect the latest settings (idempotent replace).
	err = s.grafanaClient.DeleteAlertRuleGroup(ctx, advisorNotificationsFolderUID, advisorNotificationsRuleGroup)
	if err != nil {
		s.l.Debugf("Failed to delete existing advisor notifications rule group (ignored): %v", err)
	}

	datasourceUID, err := s.grafanaClient.GetDatasourceUIDByID(ctx, 1) // 1 is the Metrics datasource ID in PMM.
	if err != nil {
		return fmt.Errorf("failed to get metrics datasource UID: %w", err)
	}

	rule := buildAdvisorNotificationRule(datasourceUID, settings.AdvisorNotifications.SeverityThreshold)
	err = s.grafanaClient.CreateAlertRule(ctx, advisorNotificationsFolderUID, advisorNotificationsRuleGroup, advisorNotificationsRuleInterval, rule)
	if err != nil {
		return fmt.Errorf("failed to create advisor notifications rule: %w", err)
	}

	return nil
}
