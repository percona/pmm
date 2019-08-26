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

package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/utils/testdb"
)

func TestServer(t *testing.T) {
	sqlDB := testdb.Open(t, models.SkipFixtures)
	defer func() {
		require.NoError(t, sqlDB.Close())
	}()

	t.Run("UpdateSettingsFromEnv", func(t *testing.T) {
		t.Run("Typical", func(t *testing.T) {
			s, err := NewServer(reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), nil, nil)
			require.NoError(t, err)

			err = s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY=1",
				"METRICS_RESOLUTION=2s",
				"DATA_RETENTION=240h",
			})
			require.NoError(t, err)
			assert.Equal(t, true, s.envDisableTelemetry)
			assert.Equal(t, 2*time.Second, s.envMetricsResolution)
			assert.Equal(t, 10*24*time.Hour, s.envDataRetention)
		})

		t.Run("Untypical", func(t *testing.T) {
			s, err := NewServer(reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), nil, nil)
			require.NoError(t, err)

			err = s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY=TrUe",
				"METRICS_RESOLUTION=3S",
				"DATA_RETENTION=360H",
			})
			require.NoError(t, err)
			assert.Equal(t, true, s.envDisableTelemetry)
			assert.Equal(t, 3*time.Second, s.envMetricsResolution)
			assert.Equal(t, 15*24*time.Hour, s.envDataRetention)
		})

		t.Run("NoValue", func(t *testing.T) {
			s, err := NewServer(reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), nil, nil)
			require.NoError(t, err)

			err = s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY",
			})
			require.NoError(t, err)
			assert.Equal(t, false, s.envDisableTelemetry)
		})

		t.Run("InvalidValue", func(t *testing.T) {
			s, err := NewServer(reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf)), nil, nil)
			require.NoError(t, err)

			err = s.UpdateSettingsFromEnv([]string{
				"DISABLE_TELEMETRY=",
			})
			require.NoError(t, err)
			assert.Equal(t, false, s.envDisableTelemetry)
		})
	})
}
