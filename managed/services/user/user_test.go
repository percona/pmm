// Copyright (C) 2025 Percona LLC
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

package user

import (
	"context"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	userv1 "github.com/percona/pmm/api/user/v1"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/utils/logger"
)

// mockGrafanaClient is a mock implementation of grafanaClient interface
type mockGrafanaClient struct {
	mock.Mock
}

func (m *mockGrafanaClient) GetUserID(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func TestSnoozeUpdate(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures, nil)

	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
	ctx := logger.Set(context.Background(), t.Name())
	userID := 123

	setup := func(t *testing.T) (*Service, *mockGrafanaClient, func()) {
		t.Helper()

		mockClient := &mockGrafanaClient{}
		mockClient.Test(t)

		service := NewUserService(db, mockClient)

		cleanup := func() {
			// Clean up any test data
			_, err := db.Exec("DELETE FROM user_flags WHERE id = $1", userID)
			require.NoError(t, err)
		}

		return service, mockClient, cleanup
	}

	t.Run("snooze an update", func(t *testing.T) {
		service, mockClient, cleanup := setup(t)
		defer cleanup()

		// Mock GetUserID to return our test user ID
		mockClient.On("GetUserID", ctx).Return(userID, nil)

		// Create a user first to simulate existing user
		userInfo, err := models.GetOrCreateUser(db.Querier, userID)
		require.NoError(t, err)
		require.NotNil(t, userInfo)

		// Verify initial state
		assert.Equal(t, "", userInfo.SnoozedPMMVersion)
		assert.Nil(t, userInfo.SnoozedAt)
		assert.Equal(t, 0, userInfo.SnoozeCount)

		req := &userv1.SnoozeUpdateRequest{
			SnoozedPmmVersion: "1.0.0",
		}

		resp, err := service.SnoozeUpdate(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify the response
		assert.Equal(t, "1.0.0", resp.SnoozedPmmVersion)
		assert.WithinDuration(t, time.Now(), resp.SnoozedAt.AsTime(), 1*time.Second)
		assert.Equal(t, uint32(1), resp.SnoozeCount)

		// Verify the database was updated
		updatedUser, err := models.FindUser(db.Querier, userID)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", updatedUser.SnoozedPMMVersion)
		assert.WithinDuration(t, time.Now(), *updatedUser.SnoozedAt, 1*time.Second)
		assert.Equal(t, 1, updatedUser.SnoozeCount)

		mockClient.AssertExpectations(t)
	})

	t.Run("snooze the same update to increase snooze count", func(t *testing.T) {
		service, mockClient, cleanup := setup(t)
		defer cleanup()

		// Mock GetUserID to return our test user ID
		mockClient.On("GetUserID", ctx).Return(userID, nil)

		// Create a user with existing snooze data
		userInfo, err := models.GetOrCreateUser(db.Querier, userID)
		require.NoError(t, err)

		// Set up existing snooze data
		params := &models.UpdateUserParams{
			UserID:            userInfo.ID,
			SnoozedPMMVersion: pointer.ToString("1.0.0"),
			SnoozedAt:         pointer.ToTime(time.Now().Add(-1 * time.Hour)),
			SnoozeCount:       pointer.ToInt(2),
		}
		userInfo, err = models.UpdateUser(db.Querier, params)
		require.NoError(t, err)

		// Verify initial state
		assert.Equal(t, "1.0.0", userInfo.SnoozedPMMVersion)
		assert.Equal(t, 2, userInfo.SnoozeCount)

		// Subsequent snooze with same version
		req := &userv1.SnoozeUpdateRequest{
			SnoozedPmmVersion: "1.0.0",
		}

		resp, err := service.SnoozeUpdate(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify the response - count should be incremented
		assert.Equal(t, "1.0.0", resp.SnoozedPmmVersion)
		assert.WithinDuration(t, time.Now(), resp.SnoozedAt.AsTime(), 1*time.Second)
		assert.Equal(t, uint32(3), resp.SnoozeCount)

		// Verify the database was updated
		updatedUser, err := models.FindUser(db.Querier, userID)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", updatedUser.SnoozedPMMVersion)
		assert.WithinDuration(t, time.Now(), *updatedUser.SnoozedAt, 1*time.Second)
		assert.Equal(t, 3, updatedUser.SnoozeCount)

		mockClient.AssertExpectations(t)
	})

	t.Run("resets snooze count when called with different version", func(t *testing.T) {
		service, mockClient, cleanup := setup(t)
		defer cleanup()

		// Mock GetUserID to return our test user ID
		mockClient.On("GetUserID", ctx).Return(userID, nil)

		// Create a user with existing snooze data for different version
		userInfo, err := models.GetOrCreateUser(db.Querier, userID)
		require.NoError(t, err)

		// Set up existing snooze data for version 1.0.0
		params := &models.UpdateUserParams{
			UserID:            userInfo.ID,
			SnoozedPMMVersion: pointer.ToString("1.0.0"),
			SnoozedAt:         pointer.ToTime(time.Now().Add(-1 * time.Hour)),
			SnoozeCount:       pointer.ToInt(5),
		}
		userInfo, err = models.UpdateUser(db.Querier, params)
		require.NoError(t, err)

		// Verify initial state
		assert.Equal(t, "1.0.0", userInfo.SnoozedPMMVersion)
		assert.Equal(t, 5, userInfo.SnoozeCount)

		// Snooze with different version
		req := &userv1.SnoozeUpdateRequest{
			SnoozedPmmVersion: "2.0.0",
		}

		resp, err := service.SnoozeUpdate(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify the response - version changed and count reset to 1
		assert.Equal(t, "2.0.0", resp.SnoozedPmmVersion)
		assert.WithinDuration(t, time.Now(), resp.SnoozedAt.AsTime(), 1*time.Second)
		assert.Equal(t, uint32(1), resp.SnoozeCount)

		// Verify the database was updated
		updatedUser, err := models.FindUser(db.Querier, userID)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", updatedUser.SnoozedPMMVersion)
		assert.WithinDuration(t, time.Now(), *updatedUser.SnoozedAt, 1*time.Second)
		assert.Equal(t, 1, updatedUser.SnoozeCount)

		mockClient.AssertExpectations(t)
	})

	t.Run("multiple subsequent snoozes with same version", func(t *testing.T) {
		service, mockClient, cleanup := setup(t)
		defer cleanup()

		// Mock GetUserID to return our test user ID
		mockClient.On("GetUserID", ctx).Return(userID, nil)

		// Create a user
		_, err := models.GetOrCreateUser(db.Querier, userID)
		require.NoError(t, err)

		// Perform multiple snoozes with the same version
		version := "1.5.0"
		for i := 1; i <= 3; i++ {
			req := &userv1.SnoozeUpdateRequest{
				SnoozedPmmVersion: version,
			}

			resp, err := service.SnoozeUpdate(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			// Verify the response
			assert.Equal(t, version, resp.SnoozedPmmVersion)
			assert.WithinDuration(t, time.Now(), resp.SnoozedAt.AsTime(), 1*time.Second)
			assert.Equal(t, uint32(i), resp.SnoozeCount)
		}

		// Verify final state in database
		updatedUser, err := models.FindUser(db.Querier, userID)
		require.NoError(t, err)
		assert.Equal(t, version, updatedUser.SnoozedPMMVersion)
		assert.Equal(t, 3, updatedUser.SnoozeCount)

		mockClient.AssertExpectations(t)
	})
}
