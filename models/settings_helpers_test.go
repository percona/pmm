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

package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
	"github.com/percona/pmm-managed/utils/tests"
)

func TestSettings(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	t.Run("Defaults", func(t *testing.T) {
		actual, err := models.GetSettings(sqlDB)
		require.NoError(t, err)
		expected := &models.Settings{
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Second,
				MR: 5 * time.Second,
				LR: time.Minute,
			},
			DataRetention: 30 * 24 * time.Hour,
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("SaveWithDefaults", func(t *testing.T) {
		s := &models.Settings{}
		err := models.SaveSettings(sqlDB, s)
		require.NoError(t, err)
		expected := &models.Settings{
			MetricsResolutions: models.MetricsResolutions{
				HR: 5 * time.Second,
				MR: 5 * time.Second,
				LR: time.Minute,
			},
			DataRetention: 30 * 24 * time.Hour,
		}
		assert.Equal(t, expected, s)
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("MetricsResolutions", func(t *testing.T) {
			s := &models.Settings{
				MetricsResolutions: models.MetricsResolutions{
					HR: 500 * time.Millisecond,
				},
			}
			err := models.SaveSettings(sqlDB, s)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "hr: minimal resolution is 1s"), err)

			s = &models.Settings{
				MetricsResolutions: models.MetricsResolutions{
					LR: 1500 * time.Millisecond,
				},
			}
			err = models.SaveSettings(sqlDB, s)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "lr: should be a natural number of seconds"), err)
		})

		t.Run("DataRetention", func(t *testing.T) {
			s := &models.Settings{
				DataRetention: 12 * time.Hour,
			}
			err := models.SaveSettings(sqlDB, s)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "data_retention: minimal resolution is 24h"), err)

			s = &models.Settings{
				DataRetention: 36 * time.Hour,
			}
			err = models.SaveSettings(sqlDB, s)
			tests.AssertGRPCError(t, status.New(codes.InvalidArgument, "data_retention: should be a natural number of days"), err)
		})
	})
}
