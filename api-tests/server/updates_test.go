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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
)

func TestCheckUpdates(t *testing.T) {
	// do not run this test in parallel with other tests as it also tests timings

	const fast, slow = 5 * time.Second, 60 * time.Second

	if !pmmapitests.RunUpdateTest {
		t.Skip("skipping PMM Server check update test")
	}

	// that call should always be fast
	version, err := serverClient.Default.ServerService.Version(server.NewVersionParamsWithTimeout(fast))
	require.NoError(t, err)
	if version.Payload.Server == nil || version.Payload.Server.Version == "" {
		t.Skip("skipping test in developer's environment")
	}

	params := &server.CheckUpdatesParams{
		Context: pmmapitests.Context,
	}
	params.SetTimeout(slow) // that call can be slow with a cold cache
	res, err := serverClient.Default.ServerService.CheckUpdates(params)
	require.NoError(t, err)

	require.NotEmpty(t, res.Payload.Installed)
	assert.True(t, strings.HasPrefix(res.Payload.Installed.Version, "2.") || strings.HasPrefix(res.Payload.Installed.Version, "3."),
		"installed.version = %q should have '2.' or '3.' prefix", res.Payload.Installed.Version)
	assert.NotEmpty(t, res.Payload.Installed.FullVersion)
	require.NotEmpty(t, res.Payload.Installed.Timestamp)
	ts := time.Time(res.Payload.Installed.Timestamp)
	hour, min, _ := ts.Clock()
	assert.Zero(t, hour, "installed.timestamp should contain only date")
	assert.Zero(t, min, "installed.timestamp should contain only date")

	require.NotEmpty(t, res.Payload.Latest)
	assert.True(t, strings.HasPrefix(res.Payload.Installed.Version, "2.") || strings.HasPrefix(res.Payload.Installed.Version, "3."),
		"installed.version = %q should have '2.' or '3.' prefix", res.Payload.Installed.Version)
	assert.NotEmpty(t, res.Payload.Installed.FullVersion)

	if res.Payload.UpdateAvailable {
		require.NotEmpty(t, res.Payload.Latest)
		assert.True(t, strings.HasPrefix(res.Payload.Latest.Version, "2.") || strings.HasPrefix(res.Payload.Installed.Version, "3."),
			"latest.version = %q should have '2.' or '3.' prefix", res.Payload.Latest.Version)
		require.NotEmpty(t, res.Payload.Latest.Timestamp)
		ts = time.Time(res.Payload.Latest.Timestamp)
		hour, min, _ = ts.Clock()
		assert.Zero(t, hour, "latest.timestamp should contain only date")
		assert.Zero(t, min, "latest.timestamp should contain only date")

		assert.NotEmpty(t, res.Payload.Latest.Tag)
		require.NotEmpty(t, res.Payload.Latest.Timestamp)
		ts = time.Time(res.Payload.Latest.Timestamp)
		hour, min, _ = ts.Clock()
		assert.Zero(t, hour, "latest.timestamp should contain only date")
		assert.Zero(t, min, "latest.timestamp should contain only date")

		assert.NotEqual(t, res.Payload.Installed.FullVersion, res.Payload.Latest.Version)
		assert.NotEqual(t, res.Payload.Installed.Timestamp, res.Payload.Latest.Timestamp)
		assert.True(t, strings.HasPrefix(res.Payload.LatestNewsURL, "https://per.co.na/pmm/2."), "latest_news_url = %q", res.Payload.LatestNewsURL)
		assert.True(t, strings.HasPrefix(res.Payload.Latest.ReleaseNotesURL, "https://per.co.na/pmm/2."), "latest_news_url = %q", res.Payload.Latest.ReleaseNotesURL)
	}
	assert.NotEmpty(t, res.Payload.LastCheck)

	t.Run("HotCache", func(t *testing.T) {
		params = &server.CheckUpdatesParams{
			Context: pmmapitests.Context,
		}
		params.SetTimeout(fast) // that call should be fast with hot cache
		resAgain, err := serverClient.Default.ServerService.CheckUpdates(params)
		require.NoError(t, err)

		assert.Equal(t, res.Payload, resAgain.Payload)
	})

	t.Run("Force", func(t *testing.T) {
		params = &server.CheckUpdatesParams{
			Force:   new(true),
			Context: pmmapitests.Context,
		}
		params.SetTimeout(slow) // that call with force can be slow
		resForce, err := serverClient.Default.ServerService.CheckUpdates(params)
		require.NoError(t, err)

		assert.Equal(t, res.Payload.Installed, resForce.Payload.Installed)
		assert.Equal(t, resForce.Payload.Latest.Tag != "", resForce.Payload.UpdateAvailable)
		assert.NotEqual(t, res.Payload.LastCheck, resForce.Payload.LastCheck)
	})

	t.Run("forced with updates disabled", func(t *testing.T) {
		defer RestoreSettingsDefaults(t)
		settingsRes, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableUpdates: new(false),
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.False(t, settingsRes.Payload.Settings.UpdatesEnabled)
		params = &server.CheckUpdatesParams{
			Force:   new(true),
			Context: pmmapitests.Context,
		}
		params.SetTimeout(slow) // that call with force can be slow
		_, err = serverClient.Default.ServerService.CheckUpdates(params)
		require.Error(t, err)

		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `PMM updates are disabled`)
	})
}

func TestListUpdates(t *testing.T) {
	const fast, slow = 5 * time.Second, 60 * time.Second

	if !pmmapitests.RunUpdateTest {
		t.Skip("skipping PMM Server check update test")
	}

	version, err := serverClient.Default.ServerService.Version(server.NewVersionParamsWithTimeout(fast))
	require.NoError(t, err)
	if version.Payload.Server == nil || version.Payload.Server.Version == "" {
		t.Skip("skipping test in developer's environment")
	}

	params := &server.ListChangeLogsParams{
		Context: pmmapitests.Context,
	}
	params.SetTimeout(slow)
	res, err := serverClient.Default.ServerService.ListChangeLogs(params)
	require.NoError(t, err)

	if len(res.Payload.Updates) > 0 {
		assert.True(t, strings.HasPrefix(res.Payload.Updates[0].Version, "3."),
			"installed.version = %q should have '3.' prefix", res.Payload.Updates[0].Version)
	}

	t.Run("with updates disabled", func(t *testing.T) {
		defer RestoreSettingsDefaults(t)
		settingsRes, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
			Body: server.ChangeSettingsBody{
				EnableUpdates: new(false),
			},
			Context: pmmapitests.Context,
		})
		require.NoError(t, err)
		assert.False(t, settingsRes.Payload.Settings.UpdatesEnabled)
		params := &server.ListChangeLogsParams{
			Context: pmmapitests.Context,
		}
		params.SetTimeout(slow)
		_, err = serverClient.Default.ServerService.ListChangeLogs(params)
		require.Error(t, err)
		pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, `PMM updates are disabled`)
	})
}
