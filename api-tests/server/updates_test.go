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
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	pmmapitests "github.com/percona/pmm/api-tests"
	serverClient "github.com/percona/pmm/api/serverpb/json/client"
	"github.com/percona/pmm/api/serverpb/json/client/server"
)

func TestCheckUpdates(t *testing.T) {
	// do not run this test in parallel with other tests as it also tests timings

	const fast, slow = 5 * time.Second, 60 * time.Second

	if !pmmapitests.RunUpdateTest {
		t.Skip("skipping PMM Server check update test")
	}

	// that call should always be fast
	version, err := serverClient.Default.Server.Version(server.NewVersionParamsWithTimeout(fast))
	require.NoError(t, err)
	if version.Payload.Server == nil || version.Payload.Server.Version == "" {
		t.Skip("skipping test in developer's environment")
	}

	params := &server.CheckUpdatesParams{
		Context: pmmapitests.Context,
	}
	params.SetTimeout(slow) // that call can be slow with a cold cache
	res, err := serverClient.Default.Server.CheckUpdates(params)
	require.NoError(t, err)

	require.NotEmpty(t, res.Payload.Installed)
	assert.True(t, strings.HasPrefix(res.Payload.Installed.Version, "2."),
		"installed.version = %q should have '2.' prefix", res.Payload.Installed.Version)
	assert.NotEmpty(t, res.Payload.Installed.FullVersion)
	require.NotEmpty(t, res.Payload.Installed.Timestamp)
	ts := time.Time(res.Payload.Installed.Timestamp)
	hour, min, _ := ts.Clock()
	assert.Zero(t, hour, "installed.timestamp should contain only date")
	assert.Zero(t, min, "installed.timestamp should contain only date")

	require.NotEmpty(t, res.Payload.Latest)
	assert.True(t, strings.HasPrefix(res.Payload.Latest.Version, "2."),
		"latest.version = %q should have '2.' prefix", res.Payload.Latest.Version)
	assert.NotEmpty(t, res.Payload.Latest.FullVersion)
	require.NotEmpty(t, res.Payload.Latest.Timestamp)
	ts = time.Time(res.Payload.Latest.Timestamp)
	hour, min, _ = ts.Clock()
	assert.Zero(t, hour, "latest.timestamp should contain only date")
	assert.Zero(t, min, "latest.timestamp should contain only date")

	if res.Payload.UpdateAvailable {
		assert.NotEqual(t, res.Payload.Installed.FullVersion, res.Payload.Latest.FullVersion)
		assert.NotEqual(t, res.Payload.Installed.Timestamp, res.Payload.Latest.Timestamp)
		assert.True(t, strings.HasPrefix(res.Payload.LatestNewsURL, "https://per.co.na/pmm/2."), "latest_news_url = %q", res.Payload.LatestNewsURL)
	} else {
		assert.Equal(t, res.Payload.Installed.FullVersion, res.Payload.Latest.FullVersion)
		assert.Equal(t, res.Payload.Installed.Timestamp, res.Payload.Latest.Timestamp)
		assert.Empty(t, res.Payload.LatestNewsURL, "latest_news_url should be empty")
	}
	assert.NotEmpty(t, res.Payload.LastCheck)

	t.Run("HotCache", func(t *testing.T) {
		params = &server.CheckUpdatesParams{
			Context: pmmapitests.Context,
		}
		params.SetTimeout(fast) // that call should be fast with hot cache
		resAgain, err := serverClient.Default.Server.CheckUpdates(params)
		require.NoError(t, err)

		assert.Equal(t, res.Payload, resAgain.Payload)
	})

	t.Run("Force", func(t *testing.T) {
		params = &server.CheckUpdatesParams{
			Body: server.CheckUpdatesBody{
				Force: true,
			},
			Context: pmmapitests.Context,
		}
		params.SetTimeout(slow) // that call with force can be slow
		resForce, err := serverClient.Default.Server.CheckUpdates(params)
		require.NoError(t, err)

		assert.Equal(t, res.Payload.Installed, resForce.Payload.Installed)
		assert.Equal(t, resForce.Payload.Installed.FullVersion != resForce.Payload.Latest.FullVersion, resForce.Payload.UpdateAvailable)
		assert.NotEqual(t, res.Payload.LastCheck, resForce.Payload.LastCheck)
	})
}

func TestUpdate(t *testing.T) {
	// do not run this test in parallel with other tests

	if !pmmapitests.RunUpdateTest {
		t.Skip("skipping PMM Server update test")
	}

	// check that pmm-managed and pmm-update versions match
	version, err := serverClient.Default.Server.Version(nil)
	require.NoError(t, err)
	require.NotNil(t, version.Payload)
	t.Logf("Before update: %s", spew.Sdump(version.Payload))
	assert.True(t, strings.HasPrefix(version.Payload.Managed.Version, version.Payload.Version),
		"managed.version = %q should have %q prefix", version.Payload.Managed.Version, version.Payload.Version)
	assert.True(t, strings.HasPrefix(version.Payload.Server.Version, version.Payload.Version),
		"server.version = %q should have %q prefix", version.Payload.Server.Version, version.Payload.Version)

	// make a new client without authentication
	baseURL, err := url.Parse(pmmapitests.BaseURL.String())
	require.NoError(t, err)
	baseURL.User = nil
	noAuthClient := serverClient.New(pmmapitests.Transport(baseURL, true), nil)

	// without authentication
	_, err = noAuthClient.Server.StartUpdate(nil)
	pmmapitests.AssertAPIErrorf(t, err, 401, codes.Unauthenticated, "Unauthorized")

	// with authentication
	startRes, err := serverClient.Default.Server.StartUpdate(nil)
	require.NoError(t, err)
	authToken := startRes.Payload.AuthToken
	logOffset := startRes.Payload.LogOffset
	require.NotEmpty(t, authToken)
	assert.Zero(t, logOffset)

	_, err = serverClient.Default.Server.StartUpdate(nil)
	pmmapitests.AssertAPIErrorf(t, err, 400, codes.FailedPrecondition, "Update is already running.")

	// without token
	_, err = noAuthClient.Server.UpdateStatus(&server.UpdateStatusParams{
		Body: server.UpdateStatusBody{
			LogOffset: logOffset,
		},
		Context: pmmapitests.Context,
	})
	pmmapitests.AssertAPIErrorf(t, err, 403, codes.PermissionDenied, "Invalid authentication token.")

	// read log lines like UI would do, but without delays to increase a chance for race detector to spot something
	var lastLine string
	var retries int
	for {
		start := time.Now()
		statusRes, err := noAuthClient.Server.UpdateStatus(&server.UpdateStatusParams{
			Body: server.UpdateStatusBody{
				AuthToken: authToken,
				LogOffset: logOffset,
			},
			Context: pmmapitests.Context,
		})
		if err != nil {
			// check that we know and understand all possible errors
			switch err := err.(type) {
			case *url.Error:
				// *net.OpError, http.nothingWrittenError, or just io.EOF
			case *pmmapitests.ErrFromNginx:
				// nothing
			case *server.UpdateStatusDefault:
				assert.Equal(t, 503, err.Code(), "%[1]T %[1]s", err)
			default:
				t.Fatalf("%#v", err)
			}
			continue
		}
		dur := time.Since(start)
		t.Logf("%s, offset = %d->%d, done = %t:\n%s", dur, logOffset, statusRes.Payload.LogOffset,
			statusRes.Payload.Done, strings.Join(statusRes.Payload.LogLines, "\n"))

		if statusRes.Payload.LogOffset == logOffset {
			// pmm-managed waits up to 30 seconds for new log lines. Usually, that's more than enough for
			// Ansible playbook to produce a new output, and that test checks that. However, our Jenkins node
			// is very slow, so we try several times.
			// That code should be removed once Jenkins performance is fixed.
			t.Logf("retries = %d", retries)
			if !statusRes.Payload.Done {
				retries++
				if retries < 5 {
					assert.InDelta(t, (30 * time.Second).Seconds(), dur.Seconds(), (7 * time.Second).Seconds())
					continue
				}
			}

			assert.Empty(t, statusRes.Payload.LogLines, "lines should be empty for the same offset")
			require.True(t, statusRes.Payload.Done, "lines should be empty only when done")
			break
		}

		retries = 0
		assert.True(t, statusRes.Payload.LogOffset > logOffset,
			"expected statusRes.Payload.LogOffset (%d) > logOffset (%d)",
			statusRes.Payload.LogOffset, logOffset)
		require.NotEmpty(t, statusRes.Payload.LogLines, "pmm-managed should delay response until some lines are available")
		logOffset = statusRes.Payload.LogOffset
		lastLine = statusRes.Payload.LogLines[len(statusRes.Payload.LogLines)-1]
	}

	t.Logf("lastLine = %q", lastLine)
	assert.Contains(t, lastLine, "Waiting for Grafana dashboards update to finish...")

	// extra check for done
	statusRes, err := noAuthClient.Server.UpdateStatus(&server.UpdateStatusParams{
		Body: server.UpdateStatusBody{
			AuthToken: authToken,
			LogOffset: logOffset,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.True(t, statusRes.Payload.Done, "should be done")
	assert.Empty(t, statusRes.Payload.LogLines, "lines should be empty when done")
	assert.Equal(t, logOffset, statusRes.Payload.LogOffset)

	// whole log
	statusRes, err = noAuthClient.Server.UpdateStatus(&server.UpdateStatusParams{
		Body: server.UpdateStatusBody{
			AuthToken: authToken,
		},
		Context: pmmapitests.Context,
	})
	require.NoError(t, err)
	assert.True(t, statusRes.Payload.Done, "should be done")
	assert.Equal(t, int(logOffset), len(strings.Join(statusRes.Payload.LogLines, "\n")+"\n"))
	assert.Equal(t, logOffset, statusRes.Payload.LogOffset)
	lastLine = statusRes.Payload.LogLines[len(statusRes.Payload.LogLines)-1]
	t.Logf("lastLine = %q", lastLine)
	assert.Contains(t, lastLine, "Waiting for Grafana dashboards update to finish...")

	// check that both pmm-managed and pmm-update were updated
	version, err = serverClient.Default.Server.Version(nil)
	require.NoError(t, err)
	require.NotNil(t, version.Payload)
	t.Logf("After update: %s", spew.Sdump(version.Payload))
	assert.True(t, strings.HasPrefix(version.Payload.Managed.Version, version.Payload.Version),
		"managed.version = %q should have %q prefix", version.Payload.Managed.Version, version.Payload.Version)
	assert.True(t, strings.HasPrefix(version.Payload.Server.Version, version.Payload.Version),
		"server.version = %q should have %q prefix", version.Payload.Server.Version, version.Payload.Version)
}
