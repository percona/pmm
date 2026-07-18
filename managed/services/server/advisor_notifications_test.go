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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestReconcileAdvisorContactPoint(t *testing.T) {
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
		enabled := true
		s := &models.Settings{}
		s.AdvisorNotifications.Enabled = &enabled
		return s
	}

	t.Run("enabled resolves the email contact point recipients", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)
		gc.On("GetEmailContactPoint", mock.Anything, advisorContactPointName).
			Return([]string{"a@example.com", "b@example.com"}, nil)

		addresses, err := s.reconcileAdvisorContactPoint(t.Context(), enabledSettings())
		require.NoError(t, err)
		require.Equal(t, []string{"a@example.com", "b@example.com"}, addresses)
	})

	t.Run("disabled returns nil without calling Grafana", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)

		addresses, err := s.reconcileAdvisorContactPoint(t.Context(), &models.Settings{})
		require.NoError(t, err)
		require.Nil(t, addresses)
		gc.AssertNotCalled(t, "GetEmailContactPoint")
	})

	t.Run("enabled with no matching contact point returns nil", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)
		gc.On("GetEmailContactPoint", mock.Anything, advisorContactPointName).
			Return([]string(nil), nil)

		addresses, err := s.reconcileAdvisorContactPoint(t.Context(), enabledSettings())
		require.NoError(t, err)
		require.Empty(t, addresses)
	})

	t.Run("enabled propagates a Grafana error", func(t *testing.T) {
		t.Parallel()

		s, gc := newServer(t)
		gc.On("GetEmailContactPoint", mock.Anything, advisorContactPointName).
			Return([]string(nil), errors.New("boom"))

		_, err := s.reconcileAdvisorContactPoint(t.Context(), enabledSettings())
		require.Error(t, err)
	})
}
