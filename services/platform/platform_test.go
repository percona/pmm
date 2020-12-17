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

package platform

import (
	"context"
	"testing"

	"github.com/brianvoe/gofakeit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

const devAuthHost = "check-dev.percona.com:443"

func TestPlatformService(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, nil)

	s, err := New(db)
	require.NoError(t, err)
	s.host = devAuthHost

	t.Run("SignUp", func(t *testing.T) {
		login := tests.GenEmail(t)
		password := gofakeit.Password(true, true, true, false, false, 14)

		err := s.SignUp(context.Background(), login, password)
		// Revert once https://jira.percona.com/browse/SAAS-370 is done.
		// require.NoError(t, err)
		tests.AssertGRPCError(t, status.New(codes.Internal, "Internal server error."), err)
		t.Skip("https://jira.percona.com/browse/SAAS-370")
	})

	t.Run("SignIn", func(t *testing.T) {
		login := tests.GenEmail(t)
		password := gofakeit.Password(true, true, true, false, false, 14)

		err := s.SignUp(context.Background(), login, password)
		// Revert once https://jira.percona.com/browse/SAAS-370 is done.
		// require.NoError(t, err)
		tests.AssertGRPCError(t, status.New(codes.Internal, "Internal server error."), err)
		t.Skip("https://jira.percona.com/browse/SAAS-370")

		settings, err := models.GetSettings(s.db)
		require.NoError(t, err)
		require.Empty(t, settings.SaaS.SessionID)
		require.Empty(t, settings.SaaS.Email)

		err = s.SignIn(context.Background(), login, password)
		require.NoError(t, err)

		settings, err = models.GetSettings(s.db)
		require.NoError(t, err)
		assert.NotEmpty(t, settings.SaaS.SessionID)
		assert.Equal(t, login, settings.SaaS.Email)
	})

	t.Run("SignOut", func(t *testing.T) {
		login := tests.GenEmail(t)
		password := gofakeit.Password(true, true, true, false, false, 14)

		err := s.SignUp(context.Background(), login, password)
		// Revert once https://jira.percona.com/browse/SAAS-370 is done.
		// require.NoError(t, err)
		tests.AssertGRPCError(t, status.New(codes.Internal, "Internal server error."), err)
		t.Skip("https://jira.percona.com/browse/SAAS-370")

		err = s.SignIn(context.Background(), login, password)
		require.NoError(t, err)

		err = s.SignOut(context.Background())
		require.NoError(t, err)

		settings, err := models.GetSettings(s.db)
		require.NoError(t, err)
		require.Empty(t, settings.SaaS.SessionID)
		require.Empty(t, settings.SaaS.Email)
	})

	t.Run("refreshSession", func(t *testing.T) {
		login := tests.GenEmail(t)
		password := gofakeit.Password(true, true, true, false, false, 14)

		err := s.SignUp(context.Background(), login, password)
		// Revert once https://jira.percona.com/browse/SAAS-370 is done.
		// require.NoError(t, err)
		tests.AssertGRPCError(t, status.New(codes.Internal, "Internal server error."), err)
		t.Skip("https://jira.percona.com/browse/SAAS-370")

		err = s.SignIn(context.Background(), login, password)
		require.NoError(t, err)

		err = s.refreshSession(context.Background())
		assert.NoError(t, err)
	})
}
