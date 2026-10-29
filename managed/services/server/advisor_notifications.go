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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	managementv1 "github.com/percona/pmm/api/management/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
)

// validateAdvisorNotificationRecipients rejects a settings change that would leave Advisor
// notifications enabled with nobody to notify, which would otherwise fail silently once a check
// run completed.
//
// It resolves the effective post-change state rather than looking at the request alone, because
// enablement and recipients are separate fields and either one may already be stored.
func validateAdvisorNotificationRecipients(oldSettings *models.Settings, params *models.ChangeSettingsParams) error {
	enabled := oldSettings.IsAdvisorNotificationsEnabled()
	if params.EnableAdvisorNotifications != nil {
		enabled = *params.EnableAdvisorNotifications
	}
	if !enabled {
		return nil
	}

	addresses := oldSettings.AdvisorNotifications.EmailAddresses
	if params.AdvisorNotificationEmailAddresses != nil {
		addresses = params.AdvisorNotificationEmailAddresses
	}
	if len(addresses) == 0 {
		return status.Error(codes.InvalidArgument, "Invalid argument: advisor_notification_email_addresses: "+
			"at least one recipient is required while Advisor notifications are enabled.")
	}

	return nil
}

// advisorSeveritiesToAPI converts the stored notification severities to their API values.
func advisorSeveritiesToAPI(severities []common.Severity) []managementv1.Severity {
	res := make([]managementv1.Severity, 0, len(severities))
	for _, severity := range severities {
		// severity is a small bounded enum value
		res = append(res, managementv1.Severity(severity)) //nolint:gosec
	}
	return res
}

// advisorSeveritiesFromAPI converts the requested notification severities to their stored values.
func advisorSeveritiesFromAPI(severities []managementv1.Severity) []common.Severity {
	res := make([]common.Severity, 0, len(severities))
	for _, severity := range severities {
		res = append(res, common.Severity(severity))
	}
	return res
}
