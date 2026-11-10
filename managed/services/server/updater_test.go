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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/utils/env"
	"github.com/percona/pmm/managed/utils/envvars"
	"github.com/percona/pmm/managed/utils/testdb"
	"github.com/percona/pmm/managed/utils/tests"
	"github.com/percona/pmm/version"
)

func TestUpdater(t *testing.T) {
	const tmpDistributionFile = "/tmp/distribution"

	sqlDB := testdb.Open(t, models.SkipFixtures, nil)
	db := reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))

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
					BuildTime:   new(time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC)),
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
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				u := NewUpdater(db)
				parsed, err := version.Parse(tt.args.currentVersion)
				require.NoError(t, err)
				_, next := u.next(t.Context(), *parsed, tt.args.results)
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
		u := NewUpdater(db)

		t.Run("LatestFromProduction", func(t *testing.T) {
			_, latest, err := u.latest(t.Context())
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
				t.Setenv(env.PlatformAddress, versionServiceURL)
			}()
			t.Setenv(env.PlatformAddress, "https://check-dev.percona.com")
			_, latest, err := u.latest(t.Context())
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

		u := NewUpdater(db)
		_, latest, err := u.latest(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "2.41.1", latest.Version.String())
		assert.Equal(t, "2.41.1", latest.DockerImage)
		assert.Equal(t, time.Date(2024, 3, 20, 15, 48, 7, 145620000, time.UTC), latest.BuildTime)
	})

	t.Run("with disabled updates check", func(t *testing.T) {
		_, err := models.UpdateSettings(db.Querier, &models.ChangeSettingsParams{
			EnableUpdates: new(false),
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		u := NewUpdater(db)
		err = u.check(ctx)
		require.Error(t, err)
		tests.AssertGRPCError(t, grpcstatus.New(codes.FailedPrecondition, "PMM updates are disabled"), err)
	})
}

// trackingReadCloser records whether Close was called.
type trackingReadCloser struct {
	io.Reader

	closed bool
}

func (rc *trackingReadCloser) Close() error {
	rc.closed = true
	return nil
}

type stubRoundTripper struct {
	statusCode int
	body       io.ReadCloser
}

func (rt stubRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: rt.statusCode,
		Body:       rt.body,
		Header:     make(http.Header),
	}, nil
}

func TestGetReleaseNotesClosesResponseBody(t *testing.T) {
	t.Setenv(env.PlatformAddress, "https://version.test")

	body := &trackingReadCloser{Reader: strings.NewReader("")}
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = stubRoundTripper{statusCode: http.StatusNotFound, body: body}
	t.Cleanup(func() { http.DefaultClient.Transport = origTransport })

	u := NewUpdater(nil)
	text, err := u.getReleaseNotesText(t.Context(), *version.MustParse("3.0.0"))
	require.NoError(t, err)
	assert.Empty(t, text)
	assert.True(t, body.closed, "response body must be closed on non-200 status")
}

// newOfflineUpdater returns an Updater whose version service is served by handler,
// with settings expected to be read checks times.
func newOfflineUpdater(t *testing.T, handler http.HandlerFunc, checks int) (*Updater, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for range checks {
		mock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow([]byte("{}")))
	}

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Setenv(env.PlatformAddress, srv.URL)
	t.Setenv("PMM_DEV_UPDATE_DOCKER_IMAGE", "")

	oldFileName, oldTimeout, oldVersion := fileName, versionServiceTimeout, version.Version
	fileName = filepath.Join(t.TempDir(), "absent.json")
	versionServiceTimeout = 200 * time.Millisecond
	version.Version = "3.0.0"
	t.Cleanup(func() {
		fileName, versionServiceTimeout, version.Version = oldFileName, oldTimeout, oldVersion
	})

	return NewUpdater(reform.NewDB(sqlDB, postgresql.Dialect, reform.NewPrintfLogger(t.Logf))), mock
}

func TestUpdaterVersionServiceUnreachable(t *testing.T) {
	const failedCheck = "Failed to check for PMM Server updates."

	// bounds a single call, so a missing timeout fails the test instead of hanging it;
	// each call gets its own, so one call running out cannot hide a retry made by the next
	callCtx := func(t *testing.T) context.Context {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		t.Cleanup(cancel)
		return ctx
	}

	blocking := func(hits *atomic.Int32) http.HandlerFunc {
		return func(_ http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			<-r.Context().Done()
		}
	}

	t.Run("check returns Unavailable within the timeout", func(t *testing.T) {
		var hits atomic.Int32
		u, _ := newOfflineUpdater(t, blocking(&hits), 1)

		start := time.Now()
		err := u.check(callCtx(t))

		tests.AssertGRPCError(t, grpcstatus.New(codes.Unavailable, failedCheck), err)
		assert.Less(t, time.Since(start), 2*time.Second)
		assert.True(t, u.lastCheckTime.IsZero())
		assert.False(t, u.lastCheckAttempt.IsZero())
	})

	t.Run("check abandoned by its caller is not an attempt", func(t *testing.T) {
		var hits atomic.Int32
		var once sync.Once
		received := make(chan struct{})
		handler := func(w http.ResponseWriter, r *http.Request) {
			once.Do(func() { close(received) })
			blocking(&hits)(w, r)
		}
		u, _ := newOfflineUpdater(t, handler, 1)
		// long enough that only the caller's cancellation can end the request
		versionServiceTimeout = 5 * time.Second
		ctx, cancel := context.WithCancel(callCtx(t))
		go func() {
			<-received
			cancel()
		}()

		err := u.check(ctx)

		tests.AssertGRPCError(t, grpcstatus.New(codes.Unavailable, failedCheck), err)
		assert.True(t, u.lastCheckAttempt.IsZero())
	})

	t.Run("checkResult does not retry a fresh failed attempt", func(t *testing.T) {
		var hits atomic.Int32
		// a second settings read lets an unwanted retry reach the version service
		u, _ := newOfflineUpdater(t, blocking(&hits), 2)

		_ = u.check(callCtx(t))
		attempt := u.lastCheckAttempt
		res, last := u.checkResult(callCtx(t))

		assert.Nil(t, res)
		assert.True(t, last.IsZero())
		assert.Equal(t, attempt, u.lastCheckAttempt)
		assert.Equal(t, int32(1), hits.Load())
	})

	t.Run("checkResult retries a stale failed attempt", func(t *testing.T) {
		var hits atomic.Int32
		u, mock := newOfflineUpdater(t, blocking(&hits), 2)

		_ = u.check(callCtx(t))
		u.lastCheckAttempt = time.Now().Add(-updateCheckResultFresh - time.Minute)
		u.checkResult(callCtx(t))

		assert.Equal(t, int32(2), hits.Load())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("checkResult checks again once updates are re-enabled", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/metadata/v2/pmm-server" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = io.WriteString(w, `{"versions":[{"version":"3.99.0","imageInfo":{"imagePath":"percona/pmm-server:3.99.0"}}]}`)
		}
		u, mock := newOfflineUpdater(t, handler, 0)
		mock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow([]byte(`{"updates":{"enabled":false}}`)))
		mock.ExpectQuery("SELECT settings FROM settings").
			WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow([]byte("{}")))

		err := u.check(callCtx(t))
		tests.AssertGRPCError(t, grpcstatus.New(codes.FailedPrecondition, "PMM updates are disabled"), err)
		res, _ := u.checkResult(callCtx(t))

		require.NotNil(t, res)
		assert.Equal(t, "3.99.0", res.Version.String())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("checkResult waits for an in-flight check", func(t *testing.T) {
		var hits atomic.Int32
		var once sync.Once
		received := make(chan struct{})
		handler := func(w http.ResponseWriter, r *http.Request) {
			once.Do(func() { close(received) })
			blocking(&hits)(w, r)
		}
		// a second settings read lets an unwanted retry reach the version service
		u, _ := newOfflineUpdater(t, handler, 2)
		// shorter than callCtx, so checkResult still has time left to retry once this check ends
		checkCtx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		go func() { _ = u.check(checkCtx) }()
		<-received
		u.checkResult(callCtx(t))

		assert.Equal(t, int32(1), hits.Load())
	})

	t.Run("concurrent callers that find the attempt stale run one check", func(t *testing.T) {
		var hits atomic.Int32
		// a second settings read lets an unwanted second check reach the version service
		u, _ := newOfflineUpdater(t, blocking(&hits), 2)

		// both callers queue behind an earlier lock holder while the last attempt is stale
		u.checkMu.Lock()
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() { u.checkResult(callCtx(t)) })
		}
		time.Sleep(100 * time.Millisecond)
		u.checkMu.Unlock()
		wg.Wait()

		assert.Equal(t, int32(1), hits.Load())
	})

	t.Run("release notes timing out do not fail the check", func(t *testing.T) {
		var hits atomic.Int32
		handler := func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/metadata/v2/pmm-server" {
				blocking(&hits)(w, r)
				return
			}
			_, _ = io.WriteString(w, `{"versions":[{"version":"3.99.0","imageInfo":{"imagePath":"percona/pmm-server:3.99.0"}}]}`)
		}
		u, _ := newOfflineUpdater(t, handler, 1)

		start := time.Now()
		err := u.check(callCtx(t))

		require.NoError(t, err)
		assert.Less(t, time.Since(start), 2*time.Second)
		require.NotNil(t, u.lastCheckResult)
		assert.Equal(t, "3.99.0", u.lastCheckResult.Version.String())
		assert.Equal(t, "percona/pmm-server:3.99.0", u.lastCheckResult.DockerImage)
		assert.Empty(t, u.lastCheckResult.ReleaseNotesText)
		assert.Equal(t, int32(1), hits.Load())
	})
}
