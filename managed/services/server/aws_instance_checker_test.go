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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	serverpb "github.com/percona/pmm/api/serverpb/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
)

func TestAWSInstanceChecker(t *testing.T) {
	setup := func(t *testing.T) (db *reform.DB, teardown func()) {
		t.Helper()
		sqlDB := testdb.Open(t, models.SkipFixtures, nil)
		db = reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

		teardown = func() {
			t.Helper()
			require.NoError(t, sqlDB.Close())
		}

		return
	}

	t.Run("Docker", func(t *testing.T) {
		db, teardown := setup(t)
		defer teardown()

		telemetry := &mockTelemetryService{}
		telemetry.Test(t)
		telemetry.On("DistributionMethod").Return(serverpb.DistributionMethod_DISTRIBUTION_METHOD_DOCKER)
		defer telemetry.AssertExpectations(t)

		checker := NewAWSInstanceChecker(db, telemetry)
		assert.False(t, checker.MustCheck())
		assert.NoError(t, checker.check("foo"))
	})

	t.Run("AMI", func(t *testing.T) {
		db, teardown := setup(t)
		defer teardown()

		telemetry := &mockTelemetryService{}
		telemetry.Test(t)
		telemetry.On("DistributionMethod").Return(serverpb.DistributionMethod_DISTRIBUTION_METHOD_AMI)
		defer telemetry.AssertExpectations(t)

		checker := NewAWSInstanceChecker(db, telemetry)
		assert.True(t, checker.MustCheck())
		tests.AssertGRPCError(t, status.New(codes.Unavailable, `cannot get instance metadata`), checker.check("foo"))
	})

	t.Run("AMI/Checked", func(t *testing.T) {
		db, teardown := setup(t)
		defer teardown()

		settings, err := models.GetSettings(db.Querier)
		require.NoError(t, err)
		settings.AWSInstanceChecked = true
		err = models.SaveSettings(db.Querier, settings)
		require.NoError(t, err)

		telemetry := &mockTelemetryService{}
		telemetry.Test(t)
		telemetry.On("DistributionMethod").Return(serverpb.DistributionMethod_DISTRIBUTION_METHOD_AMI)
		defer telemetry.AssertExpectations(t)

		checker := NewAWSInstanceChecker(db, telemetry)
		assert.False(t, checker.MustCheck())
		assert.NoError(t, checker.check("foo"))
	})
}
