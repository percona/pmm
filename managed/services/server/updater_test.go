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
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/version"
)

func TestUpdater(t *testing.T) {
	gRPCMessageMaxSize := uint32(100 * 1024 * 1024)
	watchtowerURL, _ := url.Parse("http://watchtower:8080")

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
						{Name: "3.0.0"},
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
						{Name: "3.2.0"},
						{Name: "3.1.0"},
						{Name: "3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.2.0",
					DockerImage: "3.2.0",
				},
			},
			{
				name: "new patch version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "3.0.0"},
						{Name: "3.0.1", TagLastPushed: time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC)},
					},
				},
				want: &versionInfo{
					Version:     "3.0.1",
					DockerImage: "3.0.1",
					BuildTime:   pointer.To(time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC)),
				},
			},
			{
				name: "new major version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "4.0.0"},
						{Name: "3.0.0"},
					},
				},
				want: &versionInfo{
					Version:     "4.0.0",
					DockerImage: "4.0.0",
				},
			},
			{
				name: "new major version with rc version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "4.0.0"},
						{Name: "3.0.0"},
						{Name: "4.0.0-rc"},
					},
				},
				want: &versionInfo{
					Version:     "4.0.0",
					DockerImage: "4.0.0",
				},
			},
			{
				name: "multiple new major versions",
				args: args{
					currentVersion: "3.3.0",
					results: []result{
						{Name: "4.1.0"},
						{Name: "4.0.0"},
						{Name: "3.0.0"},
						{Name: "5.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "4.1.0",
					DockerImage: "4.1.0",
				},
			},
			{
				name: "new major version with minor version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "4.1.0"},
						{Name: "4.0.0"},
						{Name: "3.0.0"},
						{Name: "3.1.0"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "3.1.0",
				},
			},
			{
				name: "invalid version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "3.0.0"},
						{Name: "3.1.0"},
						{Name: "invalid"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "3.1.0",
				},
			},
			{
				name: "non semver version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "3.0.0"},
						{Name: "3.1"},
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
						{Name: "3.0.0"},
						{Name: "3.1.0-rc"},
						{Name: "3.1.0-rc757"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0-rc757",
					DockerImage: "3.1.0-rc757",
				},
			},
			{
				name: "rc version and release version",
				args: args{
					currentVersion: "3.0.0",
					results: []result{
						{Name: "3.0.0"},
						{Name: "3.1.0"},
						{Name: "3.1.0-rc"},
						{Name: "3.1.0-rc757"},
					},
				},
				want: &versionInfo{
					Version:     "3.1.0",
					DockerImage: "3.1.0",
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

	t.Run("TestLatest", func(t *testing.T) {
		// Used PMM 2, because PMM 3 is not released yet.
		version.Version = "2.40.0"
		u := NewUpdater(watchtowerURL, gRPCMessageMaxSize)
		_, latest, err := u.latest(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, latest)
		assert.True(t, strings.HasPrefix(latest.Version.String(), "2.41."), "latest version of PMM 2 should have prefix 2.41.")
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
}
