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

func TestTipsServiceGetUserTip(t *testing.T) {
	tests := []struct {
		name           string
		tipRequest     *onboardingpb.GetTipRequest
		mockInvService func() *mockInventoryService

		expectedTipResponse *onboardingpb.GetTipResponse
		expectedError       error
	}{
		{
			name: "retrieve system tip status of valid tip of install pmm server",
			tipRequest: &onboardingpb.GetTipRequest{
				TipId:   1,
				TipType: onboardingpb.TipType_SYSTEM,
				UserId:  1,
			},
			mockInvService: func() *mockInventoryService {
				return &mockInventoryService{}
			},
			expectedTipResponse: &onboardingpb.GetTipResponse{
				TipId:       1,
				IsCompleted: true,
			},
		},
		{
			name: "retrieve system tip status of valid tip of connected service to pmm (only one service is connected)",
			tipRequest: &onboardingpb.GetTipRequest{
				TipId:   3,
				TipType: onboardingpb.TipType_SYSTEM,
				UserId:  1,
			},
			mockInvService: func() *mockInventoryService {
				service := &mockInventoryService{}
				service.On("List", mock.Anything, models.ServiceFilters{}).
					Return([]inventorypb.Service{
						&inventorypb.ExternalService{},
					}, nil)
				return service
			},
			expectedTipResponse: &onboardingpb.GetTipResponse{
				TipId:       3,
				IsCompleted: false,
			},
		},
		{
			name: "retrieve system tip status of valid tip of connected service to pmm (only two services are connected)",
			tipRequest: &onboardingpb.GetTipRequest{
				TipId:   3,
				TipType: onboardingpb.TipType_SYSTEM,
				UserId:  1,
			},
			mockInvService: func() *mockInventoryService {
				service := &mockInventoryService{}
				service.On("List", mock.Anything, models.ServiceFilters{}).
					Return([]inventorypb.Service{
						&inventorypb.ExternalService{},
						&inventorypb.ExternalService{},
					}, nil)
				return service
			},
			expectedTipResponse: &onboardingpb.GetTipResponse{
				TipId:       3,
				IsCompleted: true,
			},
		},
		{
			name: "retrieve system tip status of not valid tip",
			tipRequest: &onboardingpb.GetTipRequest{
				TipId:   20,
				TipType: onboardingpb.TipType_SYSTEM,
				UserId:  1,
			},
			mockInvService: func() *mockInventoryService {
				service := &mockInventoryService{}
				return service
			},
			expectedError: errors.New("system tip doesn't exist: 20"),
		},
		{
			name: "retrieve user tip status which doesn't exist, it should be not completed by default",
			tipRequest: &onboardingpb.GetTipRequest{
				TipId:   2000,
				TipType: onboardingpb.TipType_USER,
				UserId:  1,
			},
			mockInvService: func() *mockInventoryService {
				service := &mockInventoryService{}
				return service
			},
			expectedTipResponse: &onboardingpb.GetTipResponse{
				TipId:       2000,
				IsCompleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			sqlDB := testdb.Open(t, models.SetupFixtures, nil)
			db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
			defer func() {
				require.NoError(t, sqlDB.Close())
			}()

			ctx := context.Background()

			is := tt.mockInvService()
			tipService := NewTipService(db, is)
			status, err := tipService.GetTipStatus(ctx, tt.tipRequest)

			if tt.expectedError == nil {
				require.NoError(t, err)

				require.Equal(t, tt.expectedTipResponse.TipId, status.TipId)
				require.Equal(t, tt.expectedTipResponse.IsCompleted, status.IsCompleted)
			} else {
				require.Error(t, err, tt.expectedError.Error())
			}
		})
	}
}

func TestTipsServiceCompleteUserTip(t *testing.T) {
	t.Run("complete user tip when it doesn't exist", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		defer func() {
			require.NoError(t, sqlDB.Close())
		}()

		ctx := context.Background()

		is := &mockInventoryService{}
		tipService := NewTipService(db, is)
		_, err := tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
			TipId:  2000,
			UserId: 2,
		})

		require.NoError(t, err)
	})

	t.Run("complete user tip when it's already completed", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		defer func() {
			require.NoError(t, sqlDB.Close())
		}()

		ctx := context.Background()

		is := &mockInventoryService{}
		tipService := NewTipService(db, is)
		_, err := tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
			TipId:  2000,
			UserId: 2,
		})

		require.NoError(t, err)

		_, err = tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
			TipId:  2000,
			UserId: 2,
		})

		require.NoError(t, err)
	})

	t.Run("return error when user tries to complete system tip", func(t *testing.T) {
		t.Helper()

		sqlDB := testdb.Open(t, models.SetupFixtures, nil)
		db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))
		defer func() {
			require.NoError(t, sqlDB.Close())
		}()

		ctx := context.Background()

		is := &mockInventoryService{}
		tipService := NewTipService(db, is)
		_, err := tipService.CompleteUserTip(ctx, &onboardingpb.CompleteUserTipRequest{
			TipId:  1,
			UserId: 2,
		})

		require.Error(t, err, "Tip ID is not correct, it's system tip")
	})
}
