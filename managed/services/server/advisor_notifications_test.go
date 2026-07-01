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
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/pi/common"
	"github.com/percona/pmm/managed/services"
)

func TestAdvisorNotificationSeverityRegex(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "emergency", advisorNotificationSeverityRegex(common.Emergency))
	assert.Equal(t, "emergency|alert|critical|error", advisorNotificationSeverityRegex(common.Error))
	assert.Equal(t, "emergency|alert|critical|error|warning|notice|info|debug", advisorNotificationSeverityRegex(common.Debug))
	// Unknown falls back to the Error threshold.
	assert.Equal(t, "emergency|alert|critical|error", advisorNotificationSeverityRegex(common.Unknown))
}

func TestBuildAdvisorNotificationRule(t *testing.T) {
	t.Parallel()

	rule := buildAdvisorNotificationRule("ds-uid", common.Error)

	require.Len(t, rule.GrafanaAlert.Data, 1)
	assert.Equal(t, advisorNotificationsRuleTitle, rule.GrafanaAlert.Title)
	assert.Equal(t, "ds-uid", rule.GrafanaAlert.Data[0].DatasourceUID)
	assert.Equal(t, `pmm_managed_advisor_check_insights{severity=~"emergency|alert|critical|error"} > 0`, rule.GrafanaAlert.Data[0].Model.Expr)
	assert.True(t, rule.GrafanaAlert.Data[0].Model.Instant)
	assert.Equal(t, "1", rule.Labels[advisorNotificationLabel])
}

func TestReconcileAdvisorNotifications(t *testing.T) {
	t.Parallel()

	newServer := func(t *testing.T) (*Server, *mockGrafanaClient) {
		t.Helper()
		gc := newMockGrafanaClient(t)
		s := &Server{
			grafanaClient: gc,
			l:             logrus.WithField("test", t.Name()),
		}
		return s, gc
	}

	enabledSettings := func() *models.Settings {
		s := &models.Settings{}
		s.AdvisorNotifications.Enabled = new(true)
		s.AdvisorNotifications.SeverityThreshold = common.Error
		return s
	}

	t.Run("enabled creates folder and rule", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)
		gc.On("CreateFolderWithUID", mock.Anything, advisorNotificationsFolderTitle, advisorNotificationsFolderUID).Return(nil)
		gc.On("DeleteAlertRuleGroup", mock.Anything, advisorNotificationsFolderUID, advisorNotificationsRuleGroup).Return(nil)
		gc.On("GetDatasourceUIDByID", mock.Anything, int64(1)).Return("ds-uid", nil)
		gc.On(
			"CreateAlertRule", mock.Anything, advisorNotificationsFolderUID, advisorNotificationsRuleGroup, advisorNotificationsRuleInterval,
			mock.MatchedBy(func(rule *services.Rule) bool {
				return rule.GrafanaAlert.Data[0].Model.Expr == `pmm_managed_advisor_check_insights{severity=~"emergency|alert|critical|error"} > 0` &&
					rule.Labels[advisorNotificationLabel] == "1"
			}),
		).Return(nil)

		err := s.reconcileAdvisorNotifications(t.Context(), enabledSettings())
		require.NoError(t, err)
	})

	t.Run("disabled deletes rule group only", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)
		gc.On("DeleteAlertRuleGroup", mock.Anything, advisorNotificationsFolderUID, advisorNotificationsRuleGroup).Return(nil)

		err := s.reconcileAdvisorNotifications(t.Context(), &models.Settings{})
		require.NoError(t, err)
		gc.AssertNotCalled(t, "CreateFolderWithUID")
		gc.AssertNotCalled(t, "CreateAlertRule")
	})

	t.Run("enabled returns error when datasource lookup fails", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)
		gc.On("CreateFolderWithUID", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gc.On("DeleteAlertRuleGroup", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gc.On("GetDatasourceUIDByID", mock.Anything, int64(1)).Return("", errors.New("boom"))

		err := s.reconcileAdvisorNotifications(t.Context(), enabledSettings())
		require.Error(t, err)
		gc.AssertNotCalled(t, "CreateAlertRule")
	})
}
