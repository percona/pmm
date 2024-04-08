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
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/version"
)

func TestUpdater(t *testing.T) {
	gRPCMessageMaxSize := uint32(100 * 1024 * 1024)
	gaReleaseDate := time.Date(2019, 9, 18, 0, 0, 0, 0, time.UTC)
	watchtowerURL, _ := url.Parse("http://watchtower:8080")

	t.Run("TestNextVersion", func(t *testing.T) {
		type args struct {
			currentVersion string
			results        []result
		}
		type versionInfo struct {
			Version     string
			DockerImage string
		}
		tests := []struct {
			name    string
			args    args
			want    *versionInfo
			wantErr assert.ErrorAssertionFunc
		}{
			{
				name: "no results",
				args: args{
					currentVersion: "3.0.0",
					results:        nil,
				},
				want: &versionInfo{
					Version:     "3.0.0",
					DockerImage: "",
				},
				wantErr: nil,
			},
			{
				name: "no new version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.0.0",
					DockerImage: "",
				},
				wantErr: nil,
			},
			{
				name: "new minor version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:3.1.0"},
						{Name: "percona/pmm-server:3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
				wantErr: nil,
			},
			{
				name: "new major version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:4.0.0"},
						{Name: "percona/pmm-server:3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "4.0.0",
					DockerImage: "percona/pmm-server:4.0.0",
				},
				wantErr: nil,
			},
			{
				name: "new major version with rc version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:4.0.0"},
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:4.0.0-rc"},
					},
				},
				want: &versionInfo{
					Version:     "4.0.0",
					DockerImage: "percona/pmm-server:4.0.0",
				},
				wantErr: nil,
			},
			{
				name: "multiple new major versions",
				args: args{
					currentVersion: "3.3.0",
					results: []result{
						{Name: "percona/pmm-server:4.0.0"},
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:4.1.0"},
						{Name: "percona/pmm-server:5.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "4.1.0",
					DockerImage: "percona/pmm-server:4.1.0",
				},
				wantErr: nil,
			},
			{
				name: "new major version with minor version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:4.1.0"},
						{Name: "percona/pmm-server:4.0.0"},
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:3.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
				wantErr: nil,
			},
			{
				name: "invalid version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:3.1.0"},
						{Name: "percona/pmm-server:invalid"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
				wantErr: nil,
			},
			{
				name: "non semver version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:3.1"},
					},
				},
				want: &versionInfo{
					Version:     "3.0.0",
					DockerImage: "",
				},
				wantErr: nil,
			},
			{
				name: "rc version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:3.1.0-rc"},
						{Name: "percona/pmm-server:3.1.0-rc757"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0-rc757",
					DockerImage: "percona/pmm-server:3.1.0-rc757",
				},
				wantErr: nil,
			},
			{
				name: "rc version and release version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "percona/pmm-server:3.0.0"},
						{Name: "percona/pmm-server:3.1.0-rc"},
						{Name: "percona/pmm-server:3.1.0-rc757"},
						{Name: "percona/pmm-server:3.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
				wantErr: nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				u := NewUpdater(nil, gRPCMessageMaxSize)
				parsed, err := version.Parse(tt.args.currentVersion)
				require.NoError(t, err)
				next, err := u.next(*parsed, tt.args.results)
				if tt.wantErr != nil {
					tt.wantErr(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.want.Version, next.Version.String())
				assert.Equal(t, tt.want.DockerImage, next.DockerImage)
			})
		}
	})

	t.Run("Installed", func(t *testing.T) {
		t.Skip("This test is to be deprecated or completely rewritten")
		checker := NewUpdater(watchtowerURL, gRPCMessageMaxSize)

		info := checker.InstalledPMMVersion()
		require.NotNil(t, info)

		assert.True(t, strings.HasPrefix(info.Version, "3."), "version should start with `3.`. Actual value is: %s", info.Version)
		fullVersion, _ := normalizeFullversion(&info)
		assert.True(t, strings.HasPrefix(fullVersion, "3."), "full version should start with `3.`. Actual value is: %s", fullVersion)
		require.NotEmpty(t, info.BuildTime)
		assert.True(t, info.BuildTime.After(gaReleaseDate), "BuildTime = %s", info.BuildTime)
	})

	t.Run("Check", func(t *testing.T) {
		t.Skip("This test is to be deprecated or completely rewritten")

		ctx := context.TODO()
		checker := NewUpdater(watchtowerURL, gRPCMessageMaxSize)

		res, resT := checker.LastCheckUpdatesResult(ctx)
		assert.WithinDuration(t, time.Now(), resT, time.Second)

		assert.True(t, strings.HasPrefix(res.Installed.Version, "3."), "installed version should start with `3.`. Actual value is: %s", res.Installed.Version)
		installedFullVersion, _ := normalizeFullversion(&res.Installed)
		assert.True(t, strings.HasPrefix(installedFullVersion, "3."), "installed full version should start with `3.`. Actual value is: %s", installedFullVersion)
		require.NotEmpty(t, res.Installed.BuildTime)
		assert.True(t, res.Installed.BuildTime.After(gaReleaseDate), "Installed.BuildTime = %s", res.Installed.BuildTime)
		assert.Equal(t, "local", res.Installed.Repo)

		assert.True(t, strings.HasPrefix(res.Latest.Version, "3."), "The latest available version should start with `3.`. Actual value is: %s", res.Latest.Version)
		latestFullVersion, isFeatureBranch := normalizeFullversion(&res.Latest)
		if isFeatureBranch {
			t.Skip("Skipping check latest version.")
		}
		assert.True(t, strings.HasPrefix(latestFullVersion, "3."), "The latest available versions full value should start with `3.`. Actual value is: %s", latestFullVersion)
		require.NotEmpty(t, res.Latest.BuildTime)
		assert.True(t, res.Latest.BuildTime.After(gaReleaseDate), "Latest.BuildTime = %s", res.Latest.BuildTime)
		assert.NotEmpty(t, res.Latest.Repo)

		// We assume that the latest perconalab/pmm-server:3-dev-latest image
		// always contains the latest pmm-update package versions.
		// If this test fails, re-pull them and recreate devcontainer.
		t.Log("Assuming the latest pmm-update version.")
		assert.False(t, res.UpdateAvailable, "update should not be available")
		assert.Empty(t, res.LatestNewsURL, "latest_news_url should be empty")
		assert.Equal(t, res.Installed, res.Latest, "version should be the same (latest)")
		assert.Equal(t, *res.Installed.BuildTime, *res.Latest.BuildTime, "build times should be the same")
		assert.Equal(t, "local", res.Latest.Repo)

		// cached result
		res2, resT2 := checker.checkResult(ctx)
		assert.Equal(t, res, res2)
		assert.Equal(t, resT, resT2)

		time.Sleep(100 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), updateDefaultTimeout)
		defer cancel()
		go checker.run(ctx)
		time.Sleep(100 * time.Millisecond)

		// should block and wait for run to finish one iteration
		res3, resT3 := checker.checkResult(ctx)
		assert.Equal(t, res2, res3)
		assert.NotEqual(t, resT2, resT3, "%s", resT2)
		assert.WithinDuration(t, resT2, resT3, 10*time.Second)
	})
}

func normalizeFullversion(info *version.PackageInfo) (version string, isFeatureBranch bool) {
	fullVersion := info.FullVersion

	epochPrefix := "1:" // set by RPM_EPOCH in PMM Server build scripts
	isFeatureBranch = strings.HasPrefix(fullVersion, epochPrefix)
	if isFeatureBranch {
		fullVersion = strings.TrimPrefix(fullVersion, epochPrefix)
	}

	return fullVersion, isFeatureBranch
}
