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

// Package accesscontrol contains PMM Server label-based access control tests.
//
// They belong here rather than in a unit test because every defect they guard against was a
// disagreement between components -- nginx, pmm-managed, Grafana and vmproxy each normalize a
// URL differently, and each layer's own tests stayed green while the composition leaked. Only
// a live server exercises all four at once.
package accesscontrol

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmmapitests "github.com/percona/pmm/api-tests"
	accesscontrolClient "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client"
	accesscontrol "github.com/percona/pmm/api/accesscontrol/v1beta1/json/client/access_control_service"
	serverClient "github.com/percona/pmm/api/server/v1/json/client"
	server "github.com/percona/pmm/api/server/v1/json/client/server_service"
	"github.com/percona/pmm/utils/tlsconfig"
)

// viewerPassword is fixed because the user exists only for the lifetime of this test binary.
const viewerPassword = "lbac-viewer-password"

//nolint:gochecknoglobals
var (
	// Fixture shared by every test: the restricted viewer and the data source they drive.
	testEnv env

	// Client for the payloads the generated clients cannot express.
	httpClient *http.Client
)

// env holds the fixture: a viewer restricted to a label set that matches nothing, and the two
// identifiers Grafana serves the Metrics data source under.
type env struct {
	viewer *url.Userinfo
	dsID   int64
	dsUID  string
}

func TestMain(m *testing.M) {
	httpClient = newHTTPClient()

	teardown, err := setup()
	if err != nil {
		logrus.Errorf("Failed to set up access control tests: %s", err)
		if teardown != nil {
			teardown()
		}
		os.Exit(1)
	}

	code := m.Run()
	teardown()
	os.Exit(code)
}

// TestLBACFiltersEveryDataSourceRoute covers the route shapes Grafana serves the same data
// source under. A shape missing from the prefix list in pmm-managed is served unfiltered,
// which is how PMM-15379 started.
func TestLBACFiltersEveryDataSourceRoute(t *testing.T) {
	t.Parallel()

	code, body := request(t, http.MethodGet, dsPath("/api/v1/query?query=up"), adminUser())
	require.Equal(t, http.StatusOK, code, "%s", body)
	require.NotZero(t, seriesCount(t, body),
		"PMM Server reports no 'up' series, so a filtered response is indistinguishable from an empty one")

	for _, path := range []string{
		fmt.Sprintf("/graph/api/datasources/proxy/%d/api/v1/query?query=up", testEnv.dsID),
		fmt.Sprintf("/graph/api/datasources/proxy/uid/%s/api/v1/query?query=up", testEnv.dsUID),
		fmt.Sprintf("/graph/api/datasources/%d/resources/api/v1/query?query=up", testEnv.dsID),
		fmt.Sprintf("/graph/api/datasources/uid/%s/resources/api/v1/query?query=up", testEnv.dsUID),
		// An escaped separator must not evade the prefix match either.
		fmt.Sprintf("/graph/api/datasources%%2Fproxy/%d/api/v1/query?query=up", testEnv.dsID),
		fmt.Sprintf("/graph/api/datasources/proxy/%d/api/v1/series?match%%5B%%5D=up", testEnv.dsID),
		fmt.Sprintf("/graph/api/datasources/proxy/%d/api/v1/export?match%%5B%%5D=up", testEnv.dsID),
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			code, body := request(t, http.MethodGet, path, testEnv.viewer)
			require.Equal(t, http.StatusOK, code, "%s", body)
			assert.Zerof(t, resultLen(t, body), "filters were not applied, response: %s", trim(body))
		})
	}
}

// TestVMSurfaceIsGated covers the VictoriaMetrics endpoints a dashboard user must not reach
// through the data source, whether or not access control is enabled.
func TestVMSurfaceIsGated(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/api/v1/targets",
		"/targets",
		"/api/v1/status/config",
		"/metrics",
		"/flags",
		"/debug/pprof/heap",
		"/snapshot/create",
		"/api/v1/admin/tsdb/delete_series?match%5B%5D=up",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			code, body := request(t, http.MethodGet, dsPath(path), testEnv.viewer)
			assert.Equalf(t, http.StatusForbidden, code, "response: %s", trim(body))
		})
	}

	t.Run("POST /api/v1/admin/tsdb/delete_series", func(t *testing.T) {
		t.Parallel()

		code, body := request(t, http.MethodPost, dsPath("/api/v1/admin/tsdb/delete_series?match%5B%5D=up"), testEnv.viewer)
		assert.Equalf(t, http.StatusForbidden, code, "response: %s", trim(body))
	})
}

// TestPathConfusion covers the URL shapes that once slipped past the layer in front of
// VictoriaMetrics because two components disagreed on what the path was. Each case names the
// disagreement it guards.
func TestPathConfusion(t *testing.T) {
	t.Parallel()

	// Six is the count at which the payload leaves the data source prefix in pmm-managed
	// while still cleaning back onto an allow-listed path in vmproxy.
	traversal := strings.Repeat("/..", 6)

	t.Run("a traversal in the query string does not authenticate the request", func(t *testing.T) {
		t.Parallel()

		// pmm-managed cleaned the path and query together, so this resolved to /ping,
		// which requires no role, while nginx forwarded the original URI to
		// VictoriaMetrics. Note that it only returned data with access control
		// disabled: with it enabled, the filter lookup for an anonymous caller failed
		// and refused the request by accident, so this case guards the contract rather
		// than that one build.
		code, body := request(t, http.MethodGet, "/prometheus/api/v1/query?query=up&x=/../../../../ping", nil)
		assert.Equalf(t, http.StatusUnauthorized, code, "response: %s", trim(body))
	})

	t.Run("a traversal in the query string does not drop the filters", func(t *testing.T) {
		t.Parallel()

		path := fmt.Sprintf("/graph/api/datasources/proxy/%d/api/v1/query?query=up&x=%s/ping", testEnv.dsID, traversal)
		code, body := request(t, http.MethodGet, path, testEnv.viewer)
		require.Equal(t, http.StatusOK, code, "%s", body)
		assert.Zerof(t, resultLen(t, body), "filters were not applied, response: %s", trim(body))
	})

	for _, separator := range []string{"%3F", "%23"} {
		t.Run("an encoded "+separator+" does not split the path", func(t *testing.T) {
			t.Parallel()

			// The separator decodes to a real delimiter, and the traversal behind it
			// walks out of the data source prefix in pmm-managed while Grafana keeps
			// routing the request to the data source.
			path := fmt.Sprintf("/graph/api/datasources/proxy/%d/api/v1/query%sa=%s/api/v1/query?query=up",
				testEnv.dsID, separator, traversal)
			code, body := request(t, http.MethodGet, path, testEnv.viewer)
			assert.Equalf(t, http.StatusForbidden, code, "response: %s", trim(body))
		})
	}

	t.Run("an encoded separator does not name a second path upstream", func(t *testing.T) {
		t.Parallel()

		// vmproxy checked the path it was given and forwarded a different one, because
		// the upstream URL was parsed out of an already decoded string.
		code, body := request(t, http.MethodGet, dsPath("/metrics%23/../api/v1/query"), testEnv.viewer)
		assert.Equalf(t, http.StatusForbidden, code, "response: %s", trim(body))
		assert.NotContainsf(t, string(body), "vm_promscrape", "VictoriaMetrics metrics leaked: %s", trim(body))
	})
}

// TestAdminKeepsFullSurface guards the other direction: the gate above must not cost an admin
// the diagnostics they reach through the admin-only nginx locations.
func TestAdminKeepsFullSurface(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/prometheus/api/v1/query?query=up",
		"/prometheus/api/v1/targets",
		"/prometheus/api/v1/status/tsdb",
		"/victoriametrics/targets",
		"/victoriametrics/api/v1/status/config",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			code, body := request(t, http.MethodGet, path, adminUser())
			assert.Equalf(t, http.StatusOK, code, "response: %s", trim(body))
		})
	}

	t.Run("data source query", func(t *testing.T) {
		t.Parallel()

		code, body := request(t, http.MethodGet, dsPath("/api/v1/query?query=up"), adminUser())
		require.Equal(t, http.StatusOK, code, "%s", body)
		assert.NotZero(t, seriesCount(t, body))
	})
}

// setup enables access control, creates a role whose filter matches nothing and a viewer
// holding it, and returns a teardown that undoes all three.
func setup() (func(), error) {
	var cleanups []func()
	teardown := func() {
		for _, cleanup := range slices.Backward(cleanups) {
			cleanup()
		}
	}

	accesscontrolClient.Default = accesscontrolClient.New(pmmapitests.Transport(pmmapitests.BaseURL, pmmapitests.ServerInsecureTLS), nil)

	settings, err := serverClient.Default.ServerService.GetSettings(&server.GetSettingsParams{Context: pmmapitests.Context})
	if err != nil {
		return teardown, fmt.Errorf("failed to read settings: %w", err)
	}

	if wasEnabled := settings.Payload.Settings.EnableAccessControl; !wasEnabled {
		err := setAccessControl(true)
		if err != nil {
			return teardown, err
		}
		cleanups = append(cleanups, func() {
			err := setAccessControl(false)
			if err != nil {
				logrus.Errorf("Failed to disable access control: %s", err)
			}
		})
	}

	dsID, dsUID, err := metricsDataSource()
	if err != nil {
		return teardown, err
	}
	testEnv.dsID, testEnv.dsUID = dsID, dsUID

	suffix := time.Now().UnixNano()

	role, err := accesscontrolClient.Default.AccessControlService.CreateRole(&accesscontrol.CreateRoleParams{
		Body: accesscontrol.CreateRoleBody{
			Title:       fmt.Sprintf("api-tests-lbac-%d", suffix),
			Filter:      fmt.Sprintf("{environment=\"api-tests-no-such-environment-%d\"}", suffix),
			Description: "PMM-15379 label-based access control tests",
		},
		Context: pmmapitests.Context,
	})
	if err != nil {
		return teardown, fmt.Errorf("failed to create role: %w", err)
	}
	roleID := role.Payload.RoleID
	cleanups = append(cleanups, func() {
		_, err := accesscontrolClient.Default.AccessControlService.DeleteRole(&accesscontrol.DeleteRoleParams{
			RoleID:  roleID,
			Context: pmmapitests.Context,
		})
		if err != nil {
			logrus.Errorf("Failed to delete role %d: %s", roleID, err)
		}
	})

	login := fmt.Sprintf("api-tests-lbac-viewer-%d", suffix)
	userID, err := createGrafanaUser(login)
	if err != nil {
		return teardown, err
	}
	cleanups = append(cleanups, func() {
		err := deleteGrafanaUser(userID)
		if err != nil {
			logrus.Errorf("Failed to delete Grafana user %d: %s", userID, err)
		}
	})
	testEnv.viewer = url.UserPassword(login, viewerPassword)

	_, err = accesscontrolClient.Default.AccessControlService.AssignRoles(&accesscontrol.AssignRolesParams{
		Body: accesscontrol.AssignRolesBody{
			RoleIds: []int64{roleID},
			UserID:  userID,
		},
		Context: pmmapitests.Context,
	})
	if err != nil {
		return teardown, fmt.Errorf("failed to assign role %d to user %d: %w", roleID, userID, err)
	}

	return teardown, nil
}

func setAccessControl(enabled bool) error {
	_, err := serverClient.Default.ServerService.ChangeSettings(&server.ChangeSettingsParams{
		Body:    server.ChangeSettingsBody{EnableAccessControl: new(enabled)},
		Context: pmmapitests.Context,
	})
	if err != nil {
		return fmt.Errorf("failed to set access control to %t: %w", enabled, err)
	}

	return nil
}

// metricsDataSource returns the numeric id and the uid Grafana serves the Metrics data source
// under. Both are route parameters, and the tests drive every shape built from them.
func metricsDataSource() (int64, string, error) {
	code, body, err := send(http.MethodGet, "/graph/api/datasources", adminUser())
	if err != nil {
		return 0, "", err
	}
	if code != http.StatusOK {
		return 0, "", fmt.Errorf("failed to list data sources: %d %s", code, trim(body))
	}

	var sources []struct {
		ID   int64  `json:"id"`
		UID  string `json:"uid"`
		Name string `json:"name"`
		Type string `json:"type"`
	}
	err = json.Unmarshal(body, &sources)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse data sources: %w", err)
	}

	for _, s := range sources {
		if s.Type == "prometheus" {
			return s.ID, s.UID, nil
		}
	}

	return 0, "", fmt.Errorf("no prometheus data source among %d", len(sources))
}

func createGrafanaUser(login string) (int64, error) {
	body := fmt.Sprintf(`{"name":%q,"login":%q,"password":%q}`, login, login, viewerPassword)

	code, respBody, err := sendBody(http.MethodPost, "/graph/api/admin/users", adminUser(), body)
	if err != nil {
		return 0, err
	}
	if code != http.StatusOK {
		return 0, fmt.Errorf("failed to create Grafana user: %d %s", code, trim(respBody))
	}

	var created struct {
		ID int64 `json:"id"`
	}
	err = json.Unmarshal(respBody, &created)
	if err != nil {
		return 0, fmt.Errorf("failed to parse created user: %w", err)
	}

	return created.ID, nil
}

func deleteGrafanaUser(userID int64) error {
	code, body, err := send(http.MethodDelete, fmt.Sprintf("/graph/api/admin/users/%d", userID), adminUser())
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to delete Grafana user: %d %s", code, trim(body))
	}

	return nil
}

// request sends rawPath as written and fails the test if the request cannot be made.
func request(t *testing.T, method, rawPath string, user *url.Userinfo) (int, []byte) {
	t.Helper()

	code, body, err := sendBody(method, rawPath, user, "")
	require.NoError(t, err)
	t.Logf("%s %s -> %d", method, rawPath, code)

	return code, body
}

func send(method, rawPath string, user *url.Userinfo) (int, []byte, error) {
	return sendBody(method, rawPath, user, "")
}

// sendBody sends rawPath to PMM Server byte for byte. The URL is assembled by concatenation
// on purpose: url.URL.ResolveReference resolves dot-segments and re-encodes the path, which
// would rewrite every payload here into a harmless one before it reaches the server.
func sendBody(method, rawPath string, user *url.Userinfo, body string) (int, []byte, error) {
	base := *pmmapitests.BaseURL
	base.User = nil
	base.Path = ""

	req, err := http.NewRequestWithContext(pmmapitests.Context, method, strings.TrimSuffix(base.String(), "/")+rawPath, strings.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to build request for %s: %w", rawPath, err)
	}
	if user != nil {
		password, _ := user.Password()
		req.SetBasicAuth(user.Username(), password)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to send request to %s: %w", rawPath, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read response from %s: %w", rawPath, err)
	}

	return resp.StatusCode, respBody, nil
}

// dsPath builds a data source proxy path for the sub-path VictoriaMetrics is asked for.
func dsPath(subPath string) string {
	return fmt.Sprintf("/graph/api/datasources/proxy/%d%s", testEnv.dsID, subPath)
}

func adminUser() *url.Userinfo {
	return pmmapitests.BaseURL.User
}

// seriesCount reports how many series a VictoriaMetrics query answer carries.
func seriesCount(t *testing.T, body []byte) int {
	t.Helper()

	var answer struct {
		Data struct {
			Result []json.RawMessage `json:"result"`
		} `json:"data"`
	}
	require.NoErrorf(t, json.Unmarshal(body, &answer), "response: %s", trim(body))

	return len(answer.Data.Result)
}

// resultLen reports how much a response carries, whatever shape VictoriaMetrics answered in:
// query and series return JSON, export returns newline-delimited JSON, and an empty body is a
// result of its own.
func resultLen(t *testing.T, body []byte) int {
	t.Helper()

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return 0
	}

	var answer struct {
		Data json.RawMessage `json:"data"`
	}
	err := json.Unmarshal(body, &answer)
	if err != nil {
		return len(strings.Split(trimmed, "\n"))
	}

	var series []json.RawMessage
	err = json.Unmarshal(answer.Data, &series)
	if err == nil {
		return len(series)
	}

	return seriesCount(t, body)
}

func trim(body []byte) string {
	const limit = 512
	if len(body) > limit {
		return string(body[:limit]) + "..."
	}

	return string(body)
}

func newHTTPClient() *http.Client {
	transport := &http.Transport{}
	if pmmapitests.BaseURL.Scheme == "https" {
		tlsConfig := tlsconfig.Get()
		tlsConfig.ServerName = pmmapitests.BaseURL.Hostname()
		tlsConfig.InsecureSkipVerify = pmmapitests.ServerInsecureTLS
		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{Transport: transport}
}
