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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestValidateAdvisorNotificationRecipients(t *testing.T) {
	t.Parallel()

	settings := func(enabled bool, addresses ...string) *models.Settings {
		s := &models.Settings{}
		s.AdvisorNotifications.Enabled = new(enabled)
		s.AdvisorNotifications.EmailAddresses = addresses
		return s
	}

	t.Run("disabled needs no recipients", func(t *testing.T) {
		t.Parallel()

		err := validateAdvisorNotificationRecipients(settings(false), &models.ChangeSettingsParams{})
		require.NoError(t, err)
	})

	t.Run("disabling drops the requirement even with no recipients", func(t *testing.T) {
		t.Parallel()

		err := validateAdvisorNotificationRecipients(settings(true, "a@example.com"),
			&models.ChangeSettingsParams{
				EnableAdvisorNotifications:        new(false),
				AdvisorNotificationEmailAddresses: []string{},
			})
		require.NoError(t, err)
	})

	t.Run("enabling without recipients is rejected", func(t *testing.T) {
		t.Parallel()

		err := validateAdvisorNotificationRecipients(settings(false),
			&models.ChangeSettingsParams{EnableAdvisorNotifications: new(true)})
		require.ErrorContains(t, err, "at least one recipient is required")
	})

	t.Run("enabling with recipients in the same request is accepted", func(t *testing.T) {
		t.Parallel()

		err := validateAdvisorNotificationRecipients(settings(false),
			&models.ChangeSettingsParams{
				EnableAdvisorNotifications:        new(true),
				AdvisorNotificationEmailAddresses: []string{"a@example.com"},
			})
		require.NoError(t, err)
	})

	t.Run("already enabled, stored recipients satisfy an unrelated change", func(t *testing.T) {
		t.Parallel()

		err := validateAdvisorNotificationRecipients(settings(true, "a@example.com"),
			&models.ChangeSettingsParams{})
		require.NoError(t, err)
	})

	t.Run("clearing recipients while enabled is rejected", func(t *testing.T) {
		t.Parallel()

		err := validateAdvisorNotificationRecipients(settings(true, "a@example.com"),
			&models.ChangeSettingsParams{AdvisorNotificationEmailAddresses: []string{}})
		require.ErrorContains(t, err, "at least one recipient is required")
	})
}
