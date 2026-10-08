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

// env holds the fixture: a viewer restricted to a label set that matches nothing, the two
// identifiers Grafana serves the Metrics data source under, and a folder for alert rules.
type env struct {
	viewer    *url.Userinfo
	dsID      int64
	dsUID     string
	folderUID string
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
	require.NotZero(t, results(body),
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
			assert.Zerof(t, results(body), "filters were not applied, response: %s", trim(body))
		})
	}
}

// TestLBACFiltersAlertingPreview covers the alerting routes that run a rule's queries on the
// caller's behalf. Grafana forwards the filter header on the ones it evaluates in-process; the
// rule test of a data source-managed rule sends a request of its own without it, so a filtered
// user is refused that one instead.
func TestLBACFiltersAlertingPreview(t *testing.T) {
	t.Parallel()

	eval := fmt.Sprintf(`{"condition":"A","data":[%s]}`, alertQuery())
	ruleTest := fmt.Sprintf(`{"folderUid":%q,"rule_group":"api-tests","rule":{"for":"0s","grafana_alert":`+
		`{"title":"api-tests","condition":"A","no_data_state":"OK","exec_err_state":"OK","data":[%s]}}}`,
		testEnv.folderUID, alertQuery())

	for _, tc := range []struct {
		path   string
		body   string
		series func([]byte) int
	}{
		{"/graph/api/v1/eval", eval, evalSeries},
		{"/graph/api/v1/rule/test/grafana", ruleTest, alertSeries},
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			code, body := requestBody(t, http.MethodPost, tc.path, adminUser(), tc.body)
			require.Equal(t, http.StatusOK, code, "%s", trim(body))
			require.Positive(t, tc.series(body), "admin preview selected nothing: %s", trim(body))

			code, body = requestBody(t, http.MethodPost, tc.path, testEnv.viewer, tc.body)
			require.Equal(t, http.StatusOK, code, "%s", trim(body))
			assert.Zerof(t, tc.series(body), "filters were not applied, response: %s", trim(body))
		})
	}

	t.Run("/graph/api/v1/rule/test/<uid>", func(t *testing.T) {
		t.Parallel()

		path := "/graph/api/v1/rule/test/" + testEnv.dsUID

		code, body := requestBody(t, http.MethodPost, path, adminUser(), `{"expr":"up"}`)
		assert.Equalf(t, http.StatusOK, code, "response: %s", trim(body))

		code, body = requestBody(t, http.MethodPost, path, testEnv.viewer, `{"expr":"up"}`)
		assert.Equalf(t, http.StatusForbidden, code, "response: %s", trim(body))
	})
}

// TestEncodedDelimiterInPath guards the paths that legitimately carry an encoded '?' or '#':
// Grafana puts an alert rule group name in the path, so refusing them made every such group
// impossible to open, edit or delete.
func TestEncodedDelimiterInPath(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"api-tests #1", "api-tests ?2"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			folderPath := "/graph/api/ruler/grafana/api/v1/rules/" + testEnv.folderUID
			group := fmt.Sprintf(`{"name":%q,"interval":"1m","rules":[{"for":"1m","grafana_alert":`+
				`{"title":%q,"condition":"A","no_data_state":"OK","exec_err_state":"OK","data":[%s]}}]}`,
				name, name, alertQuery())

			code, body := requestBody(t, http.MethodPost, folderPath, adminUser(), group)
			require.Equal(t, http.StatusAccepted, code, "%s", trim(body))

			groupPath := folderPath + "/" + url.PathEscape(name)
			t.Cleanup(func() {
				code, body, err := send(http.MethodDelete, groupPath, adminUser())
				if err != nil || code != http.StatusAccepted {
					t.Logf("Failed to delete rule group %s: %d %s %v", name, code, trim(body), err)
				}
			})

			code, body = request(t, http.MethodGet, groupPath, adminUser())
			require.Equal(t, http.StatusAccepted, code, "%s", trim(body))
			assert.Contains(t, string(body), name)
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

	// pmm-managed resolved the ".." and read /graph/api/api/v1/query, outside the data source
	// prefix, while Grafana routed the path as sent and vmproxy cleaned the sub-path back onto
	// /api/v1/query: every series, unfiltered.
	for _, path := range []string{
		dsPath("/api/v1/query" + traversal + "/api/v1/query?query=up"),
		dsPath("/api/v1/query" + strings.ReplaceAll(traversal, "..", "%2E%2E") + "/api/v1/query?query=up"),
		dsPath("/api/v1/query" + strings.ReplaceAll(traversal, "/", "%2F") + "%2Fapi/v1/query?query=up"),
	} {
		t.Run("a traversal in the path is refused: "+path, func(t *testing.T) {
			t.Parallel()

			code, body := request(t, http.MethodGet, path, testEnv.viewer)
			assert.Equalf(t, http.StatusForbidden, code, "response: %s", trim(body))
		})
	}

	t.Run("a traversal in the query string does not drop the filters", func(t *testing.T) {
		t.Parallel()

		path := fmt.Sprintf("/graph/api/datasources/proxy/%d/api/v1/query?query=up&x=%s/ping", testEnv.dsID, traversal)
		code, body := request(t, http.MethodGet, path, testEnv.viewer)
		require.Equal(t, http.StatusOK, code, "%s", body)
		assert.Zerof(t, results(body), "filters were not applied, response: %s", trim(body))
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
		// the upstream URL was parsed out of an already decoded string. pmm-managed now
		// refuses the ".." before it gets there; vmproxy's own tests cover the proxy.
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
		assert.NotZero(t, results(body))
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

	folderUID, err := createFolder(fmt.Sprintf("api-tests-lbac-%d", suffix))
	if err != nil {
		return teardown, err
	}
	cleanups = append(cleanups, func() {
		err := deleteFolder(folderUID)
		if err != nil {
			logrus.Errorf("Failed to delete folder %s: %s", folderUID, err)
		}
	})
	testEnv.folderUID = folderUID

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

	err = waitForFilters()
	if err != nil {
		return teardown, err
	}

	return teardown, nil
}

// waitForFilters blocks until the restricted viewer is actually filtered. pmm-managed caches
// the access control setting for a few seconds, so on a server where it was off -- every fresh
// one -- requests keep coming back unfiltered for a moment after it is switched on, and every
// assertion below would read that as a missing filter.
//
// It probes the data source route that has always been filtered, never one of the routes under
// test, so a genuine regression cannot satisfy it. Running out of time is not fatal for the
// same reason: the assertions report what the server does, with the response body, which is
// more useful than aborting the package here.
func waitForFilters() error {
	const timeout = 20 * time.Second

	deadline := time.Now().Add(timeout)
	for {
		code, body, err := send(http.MethodGet, dsPath("/api/v1/query?query=up"), testEnv.viewer)
		if err != nil {
			return err
		}

		count := results(body)
		if code == http.StatusOK && count == 0 {
			return nil
		}

		if time.Now().After(deadline) {
			logrus.Warnf("Label-based access control has not taken effect within %s: the restricted viewer still reads %d series (HTTP %d). Running the tests anyway.",
				timeout, count, code)

			return nil
		}

		time.Sleep(time.Second)
	}
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

// createFolder creates a Grafana folder for alert rules and returns its uid.
func createFolder(title string) (string, error) {
	code, body, err := sendBody(http.MethodPost, "/graph/api/folders", adminUser(), fmt.Sprintf(`{"title":%q}`, title))
	if err != nil {
		return "", err
	}
	if code != http.StatusOK {
		return "", fmt.Errorf("failed to create folder: %d %s", code, trim(body))
	}

	var created struct {
		UID string `json:"uid"`
	}
	err = json.Unmarshal(body, &created)
	if err != nil {
		return "", fmt.Errorf("failed to parse created folder: %w", err)
	}

	return created.UID, nil
}

// deleteFolder deletes a Grafana folder together with any alert rules left in it.
func deleteFolder(uid string) error {
	code, body, err := send(http.MethodDelete, fmt.Sprintf("/graph/api/folders/%s?forceDeleteRules=true", uid), adminUser())
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to delete folder: %d %s", code, trim(body))
	}

	return nil
}

// alertQuery returns an alert rule query that selects every 'up' series of the Metrics data
// source.
func alertQuery() string {
	return fmt.Sprintf(`{"refId":"A","datasourceUid":%q,"relativeTimeRange":{"from":600,"to":0},`+
		`"model":{"refId":"A","expr":"up","instant":true}}`, testEnv.dsUID)
}

// request sends rawPath as written and fails the test if the request cannot be made.
func request(t *testing.T, method, rawPath string, user *url.Userinfo) (int, []byte) {
	t.Helper()

	return requestBody(t, method, rawPath, user, "")
}

// requestBody is request with a JSON body.
func requestBody(t *testing.T, method, rawPath string, user *url.Userinfo, reqBody string) (int, []byte) {
	t.Helper()

	code, body, err := sendBody(method, rawPath, user, reqBody)
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

// results reports how much data a VictoriaMetrics answer carries, whatever shape it came in:
// query and series answer with JSON, export with newline-delimited JSON, and an empty body is
// a result of its own.
func results(body []byte) int {
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

	var list []json.RawMessage
	err = json.Unmarshal(answer.Data, &list)
	if err == nil {
		return len(list)
	}

	var vector struct {
		Data struct {
			Result []json.RawMessage `json:"result"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &vector)
	if err != nil {
		return len(strings.Split(trimmed, "\n"))
	}

	return len(vector.Data.Result)
}

// evalSeries counts the series an alerting query evaluation answered with: a frame per series
// holding its samples, and a single empty frame when the query selected nothing.
func evalSeries(body []byte) int {
	var answer struct {
		Results map[string]struct {
			Frames []struct {
				Data struct {
					Values [][]json.RawMessage `json:"values"`
				} `json:"data"`
			} `json:"frames"`
		} `json:"results"`
	}
	err := json.Unmarshal(body, &answer)
	if err != nil {
		return -1
	}

	var count int
	for _, result := range answer.Results {
		for _, frame := range result.Frames {
			if len(frame.Data.Values) > 0 && len(frame.Data.Values[0]) > 0 {
				count++
			}
		}
	}

	return count
}

// alertSeries counts the alerts a rule preview answered with that carry a series' labels. A
// query that selected nothing still yields one alert for the rule itself, without them.
func alertSeries(body []byte) int {
	var alerts []struct {
		Labels map[string]string `json:"labels"`
	}
	err := json.Unmarshal(body, &alerts)
	if err != nil {
		return -1
	}

	var count int
	for _, a := range alerts {
		if a.Labels["job"] != "" {
			count++
		}
	}

	return count
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
