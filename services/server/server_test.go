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
)

func TestServer(t *testing.T) {
	t.Run("ParseEnv", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			s, err := NewServer(nil, nil, nil, []string{
				"DISABLE_TELEMETRY=1",
				"METRICS_RESOLUTION=2",
			})
			require.NoError(t, err)
			assert.Equal(t, true, s.envDisableTelemetry)
			assert.Equal(t, 2*time.Second, s.envMetricsResolution)

			s, err = NewServer(nil, nil, nil, []string{
				"DISABLE_TELEMETRY=TrUe",
				"METRICS_RESOLUTION=3S",
			})
			require.NoError(t, err)
			assert.Equal(t, true, s.envDisableTelemetry)
			assert.Equal(t, 3*time.Second, s.envMetricsResolution)
		})

		t.Run("Invalid", func(t *testing.T) {
			s, err := NewServer(nil, nil, nil, []string{
				"DISABLE_TELEMETRY=YES",
				"METRICS_RESOLUTION=0.1s",
			})
			require.NoError(t, err)
			assert.Equal(t, false, s.envDisableTelemetry)
			assert.Equal(t, time.Duration(0), s.envMetricsResolution)

			s, err = NewServer(nil, nil, nil, []string{
				"DISABLE_TELEMETRY=on",
				"METRICS_RESOLUTION=-1",
			})
			require.NoError(t, err)
			assert.Equal(t, false, s.envDisableTelemetry)
			assert.Equal(t, time.Duration(0), s.envMetricsResolution)
		})
	})
}
