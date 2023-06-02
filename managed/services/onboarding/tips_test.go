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

package onboarding

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/api/inventorypb"
	"github.com/percona/pmm/api/onboardingpb"
	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/testdb"
)

func TestGetOnboardingStatus(t *testing.T) {
	tests := []struct {
		name           string
		tipRequest     *onboardingpb.GetOnboardingStatusRequest
		mockInvService func() *mockInventoryService

		tipResponse   func(*onboardingpb.GetOnboardingStatusResponse)
		expectedError error
	}{
		{
			name:           "retrieve system tip status of valid tip of install pmm server",
			tipRequest:     &onboardingpb.GetOnboardingStatusRequest{},
			mockInvService: getDefaultMockService(),
		},
		{
			name:           "retrieve system tip status of valid tip of install pmm server and connected service",
			tipRequest:     &onboardingpb.GetOnboardingStatusRequest{},
			mockInvService: getDefaultMockServiceWithConnectedTwoServices(),
			tipResponse: func(resp *onboardingpb.GetOnboardingStatusResponse) {
				resp.SystemTips[2].IsCompleted = true // third system tip should be completed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
			t.Cleanup(func() {
				require.NoError(t, sqlDB.Close())
			})

			ctx := context.Background()

			c := &mockGrafanaClient{}
			c.On("GetUserID", mock.Anything).
				Return(1, nil)

			is := tt.mockInvService()
			tipService := NewTipService(db, is, c)

			expectedTipResponse := getDefaultResponse()
			if tt.tipResponse != nil {
				tt.tipResponse(expectedTipResponse)
			}
			status, err := tipService.GetOnboardingStatus(ctx, tt.tipRequest)

			if tt.expectedError == nil {
				require.NoError(t, err)

				require.Equal(t, expectedTipResponse.UserTips, status.UserTips)
				require.Equal(t, expectedTipResponse.SystemTips, status.SystemTips)
			} else {
				require.Error(t, err, tt.expectedError.Error())
			}
		})
	}
}

func getDefaultMockService() func() *mockInventoryService {
	return func() *mockInventoryService {
		service := &mockInventoryService{}
		service.On("List", mock.Anything, models.ServiceFilters{}).
			Return([]inventorypb.Service{
				&inventorypb.ExternalService{},
			}, nil)

		return service
	}
}

func getDefaultMockServiceWithConnectedTwoServices() func() *mockInventoryService {
	return func() *mockInventoryService {
		service := &mockInventoryService{}
		service.On("List", mock.Anything, models.ServiceFilters{}).
			Return([]inventorypb.Service{
				&inventorypb.ExternalService{},
				&inventorypb.ExternalService{},
			}, nil)

		return service
	}
}

func getDefaultResponse() *onboardingpb.GetOnboardingStatusResponse {
	return &onboardingpb.GetOnboardingStatusResponse{
		SystemTips: []*onboardingpb.TipModel{
			{
				TipId:       1,
				IsCompleted: true,
			},
			{
				TipId:       2,
				IsCompleted: false,
			},
			{
				TipId:       3,
				IsCompleted: false,
			},
		},
		UserTips: []*onboardingpb.TipModel{
			{
				TipId:       1000,
				IsCompleted: false,
			},
		},
	}
}

func TestTipsServiceCompleteUserTip(t *testing.T) {
	t.Run("return error when user tip doesn't exist", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		ctx := context.Background()

		c := &mockGrafanaClient{}
		c.On("GetUserID", mock.Anything).
			Return(2, nil)

		is := &mockInventoryService{}
		tipService := NewTipService(db, is, c)
		_, err := tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
			TipId: 2000,
		})

		require.Error(t, err, "should not complete tip")
	})

	t.Run("should complete user tip successfully", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db.InTransaction(func(tx *reform.TX) error {
			ctx := context.Background()

			c := &mockGrafanaClient{}
			c.On("GetUserID", mock.Anything).
				Return(2, nil)

			is := &mockInventoryService{}
			tipService := NewTipService(db, is, c)

			_, err := tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
				TipId: 1000,
			})

			require.NoError(t, err)
			return errors.New("rollback changes")
		})
	})

	t.Run("return error when user tries to complete system tip", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		ctx := context.Background()

		c := &mockGrafanaClient{}
		c.On("GetUserID", mock.Anything).
			Return(2, nil)

		is := &mockInventoryService{}
		tipService := NewTipService(db, is, c)
		_, err := tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
			TipId: 1,
		})

		require.Error(t, err, "Tip ID is not correct, it's system tip")
	})

	t.Run("return error when user tries to user tip which is already completed", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		t.Cleanup(func() {
			require.NoError(t, sqlDB.Close())
		})

		db.InTransaction(func(tx *reform.TX) error {
			ctx := context.Background()

			c := &mockGrafanaClient{}
			c.On("GetUserID", mock.Anything).
				Return(2, nil)

			is := &mockInventoryService{}
			tipService := NewTipService(db, is, c)

			// user tip will be added to the user_tips table once it's requested by user
			err := db.Querier.Save(&models.OnboardingUserTip{
				UserID:      2,
				TipID:       1000,
				IsCompleted: true,
			})
			require.NoError(t, err)

			_, err = tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
				TipId: 1000,
			})

			require.Error(t, err, "should not complete an already completed tip")
			return errors.New("rollback changes")
		})
	})
}
