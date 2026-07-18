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

	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// advisorContactPointName is the fixed-convention Grafana email contact point that PMM delivers
// Advisor batch summaries to. The user creates and configures it in Grafana.
const advisorContactPointName = "PMM Advisor Insights"

// syncAdvisorContactPoint resolves the Advisor email contact point and caches its recipients in
// settings when the notification enablement changes or notifications are enabled (to pick up
// changes). Failures are logged and swallowed - they must not fail the settings change.
func (s *Server) syncAdvisorContactPoint(ctx context.Context, oldSettings, newSettings *models.Settings) {
	enabledNow := newSettings.IsAdvisorNotificationsEnabled()
	if !enabledNow && oldSettings.IsAdvisorNotificationsEnabled() == enabledNow {
		return
	}

	addresses, err := s.reconcileAdvisorContactPoint(ctx, newSettings)
	if err != nil {
		s.l.Errorf("Failed to reconcile Advisor contact point: %v", err)
		return
	}

	err = s.saveAdvisorContactPointCache(ctx, addresses)
	if err != nil {
		s.l.Errorf("Failed to cache Advisor contact point: %v", err)
	}
}

// reconcileAdvisorContactPoint resolves the Advisor email contact point's recipients from Grafana
// when notifications are enabled, returning them for the caller to cache in settings.
//
// It talks to Grafana on the caller's behalf, so it must be called within an authenticated request
// context (e.g. ChangeSettings) - background contexts have no Grafana credentials.
func (s *Server) reconcileAdvisorContactPoint(ctx context.Context, settings *models.Settings) ([]string, error) {
	if !settings.IsAdvisorNotificationsEnabled() {
		return nil, nil
	}

	addresses, err := s.grafanaClient.GetEmailContactPoint(ctx, advisorContactPointName)
	if err != nil {
		return nil, fmt.Errorf("failed to read advisor contact point: %w", err)
	}
	if len(addresses) == 0 {
		s.l.Warnf("Advisor notifications are enabled but no email contact point named %q was found in "+
			"Grafana. Create one to start sending batch summaries.", advisorContactPointName)
	}

	return addresses, nil
}

// saveAdvisorContactPointCache persists the resolved recipient addresses into settings so the
// background batch-completion path (which cannot reach Grafana) can email without further Grafana calls.
func (s *Server) saveAdvisorContactPointCache(ctx context.Context, addresses []string) error {
	return s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		settings, err := models.GetSettings(tx)
		if err != nil {
			return err
		}

		settings.AdvisorNotifications.EmailAddresses = addresses

		return models.SaveSettings(tx, settings)
	})
}
