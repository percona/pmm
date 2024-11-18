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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/version"
)

func TestUpdater(t *testing.T) {
	gRPCMessageMaxSize := uint32(100 * 1024 * 1024)
	watchtowerURL, _ := url.Parse("http://watchtower:8080")
	const tmpDistributionFile = "/tmp/distribution"

	t.Run("TestNextVersion", func(t *testing.T) {
		type args struct {
			currentVersion string
			results        []result
		}
		type versionInfo struct {
			Version     string
			DockerImage string
			BuildTime   *time.Time
		}
		tests := []struct {
			name string
			args args
			want *versionInfo
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
			},
			{
				name: "no new version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.0.0",
					DockerImage: "",
				},
			},
			{
				name: "new minor versions",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.2.0"},
						{Version: "3.1.0"},
						{Version: "3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.2.0",
					DockerImage: "percona/pmm-server:3.2.0",
				},
			},
			{
				name: "new patch version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.0.0"},
						{
							Version: "3.0.1",
							ImageInfo: imageInfo{
								ImageReleaseTimestamp: time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC),
							},
						},
					},
				},
				want: &versionInfo{
					Version:     "3.0.1",
					DockerImage: "percona/pmm-server:3.0.1",
					BuildTime:   pointer.To(time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC)),
				},
			},
			{
				name: "new major version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "4.0.0"},
						{Version: "3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "4.0.0",
					DockerImage: "percona/pmm-server:4.0.0",
				},
			},
			{
				name: "new major version with rc version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "4.0.0"},
						{Version: "3.0.0"},
						{Version: "4.0.0-rc"},
					},
				},
				want: &versionInfo{
					Version:     "4.0.0",
					DockerImage: "percona/pmm-server:4.0.0",
				},
			},
			{
				name: "multiple new major versions",
				args: args{
					currentVersion: "3.3.0",
					results: []result{
						{Version: "4.1.0"},
						{Version: "4.0.0"},
						{Version: "3.0.0"},
						{Version: "5.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "4.1.0",
					DockerImage: "percona/pmm-server:4.1.0",
				},
			},
			{
				name: "new major version with minor version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "4.1.0"},
						{Version: "4.0.0"},
						{Version: "3.0.0"},
						{Version: "3.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
			},
			{
				name: "invalid version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.0.0"},
						{Version: "3.1.0"},
						{Version: "invalid"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
			},
			{
				name: "non semver version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.0.0"},
						{Version: "3.1"},
					},
				},
				want: &versionInfo{
					Version:     "3.0.0",
					DockerImage: "",
				},
			},
			{
				name: "rc version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.0.0"},
						{Version: "3.1.0-rc"},
						{Version: "3.1.0-rc757"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0-rc757",
					DockerImage: "percona/pmm-server:3.1.0-rc757",
				},
			},
			{
				name: "rc version and release version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Version: "3.0.0"},
						{Version: "3.1.0"},
						{Version: "3.1.0-rc"},
						{Version: "3.1.0-rc757"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "percona/pmm-server:3.1.0",
				},
			},
		}
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				u := NewUpdater(watchtowerURL, gRPCMessageMaxSize)
				parsed, err := version.Parse(tt.args.currentVersion)
				require.NoError(t, err)
				_, next := u.next(*parsed, tt.args.results)
				require.NoError(t, err)
				assert.Equal(t, tt.want.Version, next.Version.String())
				assert.Equal(t, tt.want.DockerImage, next.DockerImage)
				if tt.want.BuildTime != nil {
					assert.NotNil(t, next.BuildTime)
					assert.Equal(t, *tt.want.BuildTime, next.BuildTime)
				}
			})
		}
	})

	t.Run("TestSortedVersionList", func(t *testing.T) {
		versions := version.DockerVersionsInfo{
			{Version: *version.MustParse("3.0.0")},
			{Version: *version.MustParse("3.1.0")},
			{Version: *version.MustParse("3.0.1")},
			{Version: *version.MustParse("3.0.0-rc")},
		}

		sort.Sort(versions)
		assert.Equal(t, "3.0.0-rc", versions[0].Version.String())
		assert.Equal(t, "3.0.0", versions[1].Version.String())
		assert.Equal(t, "3.0.1", versions[2].Version.String())
		assert.Equal(t, "3.1.0", versions[3].Version.String())
	})

	t.Run("TestLatest", func(t *testing.T) {
		version.Version = "2.41.0"
		u := NewUpdater(watchtowerURL, gRPCMessageMaxSize)

		t.Run("LatestFromProduction", func(t *testing.T) {
			_, latest, err := u.latest(context.Background())
			require.NoError(t, err)
			if latest != nil {
				assert.True(t, strings.HasPrefix(latest.Version.String(), "3."),
					"latest version of PMM should start with a '3.' prefix")
			}
		})
		t.Run("LatestFromStaging", func(t *testing.T) {
			versionServiceURL, err := envvars.GetPlatformAddress() // defaults to production
			require.NoError(t, err)
			defer func() {
				t.Setenv(envvars.EnvPlatformAddress, versionServiceURL)
			}()
			t.Setenv(envvars.EnvPlatformAddress, "https://check-dev.percona.com")
			_, latest, err := u.latest(context.Background())
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(latest.Version.String(), "3."),
				"latest version of PMM should start with a '3.' prefix")
		})
	})

	t.Run("TestParseFile", func(t *testing.T) {
		fileBody := `{ "version": "2.41.1" , "docker_image": "2.41.1" , "build_time": "2024-03-20T15:48:07.14562Z" }`
		oldFileName := fileName
		fileName = filepath.Join(os.TempDir(), "pmm-update.json")
		defer func() { fileName = oldFileName }()

		err := os.WriteFile(fileName, []byte(fileBody), 0o600)
		require.NoError(t, err)

		u := NewUpdater(watchtowerURL, gRPCMessageMaxSize)
		_, latest, err := u.latest(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "2.41.1", latest.Version.String())
		assert.Equal(t, "2.41.1", latest.DockerImage)
		assert.Equal(t, time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC), latest.BuildTime)
	})

	t.Run("TestUpdateEnvFile", func(t *testing.T) {
		u := NewUpdater(watchtowerURL, gRPCMessageMaxSize)
		tmpFile := filepath.Join(os.TempDir(), "pmm-service.env")
		content := `PMM_WATCHTOWER_HOST=http://watchtower:8080
PMM_WATCHTOWER_TOKEN=123
PMM_SERVER_UPDATE_VERSION=docker.io/perconalab/pmm-server:3-dev-container
PMM_IMAGE=docker.io/perconalab/pmm-server:3-dev-latest
PMM_DISTRIBUTION_METHOD=ami`
		err := os.WriteFile(tmpFile, []byte(content), 0o644)
		require.NoError(t, err)

		err = u.updatePodmanEnvironmentVariables(tmpFile, "perconalab/pmm-server:3-dev-container")
		require.NoError(t, err)
		newContent, err := os.ReadFile(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, `PMM_WATCHTOWER_HOST=http://watchtower:8080
PMM_WATCHTOWER_TOKEN=123
PMM_SERVER_UPDATE_VERSION=docker.io/perconalab/pmm-server:3-dev-container
PMM_IMAGE=docker.io/perconalab/pmm-server:3-dev-container
PMM_DISTRIBUTION_METHOD=ami`, string(newContent))
	})
}
