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

package grafana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/auth"
	"github.com/percona/pmm/managed/utils/irt"
	"github.com/percona/pmm/utils/grafana"
)

// ErrFailedToGetToken means it failed to get the user token. Most likely due to the fact the user is not logged in using Percona Account.
var ErrFailedToGetToken = errors.New("failed to get the user token")

const (
	pmmServiceTokenName          = "pmm-agent-st" //nolint:gosec
	pmmServiceAccountName        = "pmm-agent-sa" //nolint:gosec
	defaultDialTimeout           = 3 * time.Second
	defaultKeepAliveTimeout      = 30 * time.Second
	defaultIdleConnTimeout       = 90 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	defaultMaxIdleConns          = 50
)

// Client represents a client for Grafana API.
type Client struct {
	addr string
	http *http.Client
	irtm prom.Collector
}

// NewClient creates a new client for given Grafana address.
func NewClient(addr string) *Client {
	var t http.RoundTripper = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAliveTimeout,
		}).DialContext,
		MaxIdleConns:          defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleConnTimeout,
		ExpectContinueTimeout: defaultExpectContinueTimeout,
	}

	if logrus.GetLevel() >= logrus.TraceLevel {
		t = irt.WithLogger(t, logrus.WithField("component", "grafana/client").Tracef)
	}
	t, irtm := irt.WithMetrics(t, "grafana_client")

	return &Client{
		addr: addr,
		http: &http.Client{
			Transport: t,
		},
		irtm: irtm,
	}
}

// Describe implements prometheus.Collector.
func (c *Client) Describe(ch chan<- *prom.Desc) {
	c.irtm.Describe(ch)
}

// Collect implements prometheus.Collector.
func (c *Client) Collect(ch chan<- prom.Metric) {
	c.irtm.Collect(ch)
}

// clientError contains error response details.
type clientError struct {
	Method       string
	URL          string
	Code         int
	Body         string
	ErrorMessage string `json:"message"` // from response JSON object, if any
}

// Error implements error interface.
func (e *clientError) Error() string {
	return fmt.Sprintf("clientError: %s %s -> %d %s", e.Method, e.URL, e.Code, e.Body)
}

// CurrentUserHTTPResponse maps errors returned by Client calls used from current-user HTTP handlers
// to an HTTP status and a small JSON body. Non-Grafana errors (e.g. dial failures) map to 502.
func CurrentUserHTTPResponse(err error) (int, map[string]string) {
	var cErr *clientError
	if !errors.As(err, &cErr) {
		return http.StatusBadGateway, map[string]string{"message": "Bad Gateway"}
	}

	switch cErr.Code {
	case http.StatusUnauthorized:
		msg := cErr.ErrorMessage
		if msg == "" {
			msg = "Unauthorized"
		}
		return http.StatusUnauthorized, map[string]string{"message": msg}
	case http.StatusForbidden:
		msg := cErr.ErrorMessage
		if msg == "" {
			msg = "Forbidden"
		}
		return http.StatusForbidden, map[string]string{"message": msg}
	default:
		if cErr.Code >= 500 {
			return http.StatusBadGateway, map[string]string{"message": "Bad Gateway"}
		}
		// Other Grafana 4xx responses are treated as upstream errors for this proxy endpoint.
		return http.StatusBadGateway, map[string]string{"message": "Bad Gateway"}
	}
}

// do makes HTTP request with given parameters, and decodes JSON response with 200 OK status
// to respBody. It returns wrapped clientError on any other status, or other fatal errors.
// Ctx is used only for cancelation.
func (c *Client) do(ctx context.Context, method, path, rawQuery string, headers http.Header, body []byte, target any) error { //nolint:funcorder
	u := url.URL{
		Scheme:   "http",
		Host:     c.addr,
		Path:     path,
		RawQuery: rawQuery,
	}
	req, err := http.NewRequest(method, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	if len(body) != 0 {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	for k := range headers {
		req.Header.Set(k, headers.Get(k))
	}

	req = req.WithContext(ctx)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute http request: %w", err)
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read http response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 202 {
		cErr := &clientError{
			Method: req.Method,
			URL:    req.URL.String(),
			Code:   resp.StatusCode,
			Body:   string(b),
		}
		_ = json.Unmarshal(b, cErr) // add ErrorMessage
		return cErr
	}

	if len(b) != 0 && target != nil {
		err = json.Unmarshal(b, target)
		if err != nil {
			return fmt.Errorf("failed to unmarshal http response body: %w", err)
		}
	}
	return nil
}

type authUser struct {
	role   role
	userID int
}

// CurrentUser represents Grafana user payload.
type CurrentUser struct {
	ID                             int    `json:"id"`
	Email                          string `json:"email"`
	Name                           string `json:"name"`
	Login                          string `json:"login"`
	CreatedAt                      string `json:"createdAt"`
	OrgID                          int    `json:"orgId"`
	IsAnonymous                    bool   `json:"isAnonymous"`
	IsDisabled                     bool   `json:"isDisabled"`
	IsExternal                     bool   `json:"isExternal"`
	IsExtarnallySynced             bool   `json:"isExtarnallySynced"`
	IsGrafanaAdmin                 bool   `json:"isGrafanaAdmin"`
	IsGrafanaAdminExternallySynced bool   `json:"isGrafanaAdminExternallySynced"`
	Theme                          string `json:"theme"`
}

// CurrentUserOrg represents Grafana org payload.
type CurrentUserOrg struct {
	OrgID int    `json:"orgId"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// role defines Grafana user role within the organization
// (except grafanaAdmin that is a global flag that is more important than any other role).
// Role with more permissions has larger numerical value: viewer < editor, admin < grafanaAdmin, etc.
type role int

const (
	none role = iota
	viewer
	editor
	admin
	grafanaAdmin
)

func (r role) String() string {
	switch r {
	case none:
		return "None"
	case viewer:
		return "Viewer" // as in Grafana API
	case editor:
		return "Editor" // as in Grafana API
	case admin:
		return "Admin" // as in Grafana API
	case grafanaAdmin:
		return "GrafanaAdmin"
	default:
		return fmt.Sprintf("unexpected role %d", int(r))
	}
}

// GetUserID returns user ID from Grafana for the current user.
func (c *Client) GetUserID(ctx context.Context) (int, error) {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var m map[string]any
	err = c.do(ctx, http.MethodGet, "/api/user", "", authHeaders, nil, &m)
	if err != nil {
		return 0, err
	}

	userID, ok := m["id"].(float64)
	if !ok {
		return 0, errors.New("Missing User ID in Grafana response")
	}

	return int(userID), nil
}

var emptyUser = authUser{
	role:   none,
	userID: 0,
}

// getAuthUser returns grafanaAdmin if currently authenticated user is a Grafana (super) admin.
// Otherwise, it returns a role in the default organization (with ID 1).
// Ctx is used only for cancelation.
func (c *Client) getAuthUser(ctx context.Context, authHeaders http.Header, l *logrus.Entry) (authUser, error) {
	// Check if API Key or Service Token is authorized.
	token := auth.GetTokenFromHeaders(authHeaders)
	if token != "" {
		role, err := c.getRoleForServiceToken(ctx, token)
		if err != nil {
			return emptyUser, err
		}

		return authUser{
			role:   role,
			userID: 0,
		}, nil
	}

	// https://grafana.com/docs/http_api/user/#actual-user - works only with Basic Auth
	var m map[string]any
	err := c.do(ctx, http.MethodGet, "/api/user", "", authHeaders, nil, &m)
	if err != nil {
		if hasAuthorizationHeader(authHeaders) {
			return emptyUser, err
		}
		var cErr *clientError
		if !errors.As(err, &cErr) {
			return emptyUser, err
		}
		if cErr.Code != http.StatusUnauthorized {
			return emptyUser, err
		}
		anonymousEnabled, anonymousRole := c.getAnonymousRoleFromSettings(ctx, l)
		if anonymousEnabled {
			l.Debugf("Grafana returned 401 for /api/user with no credentials; using anonymous role %q.", anonymousRole.String())
			return authUser{
				role:   anonymousRole,
				userID: 0,
			}, nil
		}
		return emptyUser, err
	}

	id, _ := m["id"].(float64)
	userID := int(id)
	if a, _ := m["isGrafanaAdmin"].(bool); a {
		return authUser{
			role:   grafanaAdmin,
			userID: userID,
		}, nil
	}

	// works only with Basic auth
	var s []any
	err = c.do(ctx, http.MethodGet, "/api/user/orgs", "", authHeaders, nil, &s)
	if err != nil {
		return authUser{
			role:   none,
			userID: userID,
		}, err
	}

	for _, el := range s {
		m, _ := el.(map[string]any)
		if m == nil {
			continue
		}

		// check only default organization (with ID 1)
		if id, _ := m["orgId"].(float64); id == 1 {
			role, _ := m["role"].(string)
			return authUser{
				role:   c.convertRole(role),
				userID: userID,
			}, nil
		}
	}

	return authUser{
		role:   none,
		userID: userID,
	}, nil
}

func (c *Client) convertRole(role string) role {
	switch role {
	case "Viewer":
		return viewer
	case "Editor":
		return editor
	case "Admin":
		return admin
	default:
		return none
	}
}

// Grafana /api/frontend/settings may include user org id/name; org role for anonymous comes from anonymousOrgRole.
type frontendUserSettingsFull struct {
	OrgID   int    `json:"orgId"`
	OrgName string `json:"orgName"`
}

type frontendSettingsFull struct {
	AnonymousEnabled bool                     `json:"anonymousEnabled"`
	AnonymousOrgRole string                   `json:"anonymousOrgRole"`
	User             frontendUserSettingsFull `json:"user"`
}

func (c *Client) getAnonymousRoleFromSettings(ctx context.Context, l *logrus.Entry) (bool, role) {
	settings, err := c.getFrontendSettings(ctx)
	if err != nil {
		return false, none
	}

	if !settings.AnonymousEnabled {
		return false, none
	}

	parsedRole := c.convertRole(resolveAnonymousOrgRole(settings.AnonymousOrgRole))
	if parsedRole == none {
		return false, none
	}
	l.Debugf("Grafana anonymous mode is enabled with role %q.", parsedRole.String())
	return true, parsedRole
}

func (c *Client) getFrontendSettings(ctx context.Context) (frontendSettingsFull, error) {
	var settings frontendSettingsFull
	err := c.do(ctx, http.MethodGet, "/api/frontend/settings", "", nil, nil, &settings)
	if err != nil {
		return frontendSettingsFull{}, err
	}

	return settings, nil
}

func hasAuthorizationHeader(authHeaders http.Header) bool {
	return authHeaders.Get("Authorization") != ""
}

// Grafana organization role strings as returned by /api/frontend/settings (camelCase JSON keys).
const (
	grafanaOrgRoleViewer       = "Viewer"
	grafanaOrgRoleEditor       = "Editor"
	grafanaOrgRoleAdmin        = "Admin"
	grafanaOrgRoleGrafanaAdmin = "GrafanaAdmin"
	grafanaOrgRoleNone         = "None"
)

// resolveAnonymousOrgRole maps anonymousOrgRole from /api/frontend/settings ([auth.anonymous] org_role)
// to the org role string PMM uses for anonymous Grafana auth. Grafana does not populate user.orgRole
// in this payload; only anonymousOrgRole is authoritative. Deprecated elevated roles are clamped to Viewer.
func resolveAnonymousOrgRole(anonymousOrgRole string) string {
	anonymousOrgRole = strings.TrimSpace(anonymousOrgRole)

	switch anonymousOrgRole {
	case grafanaOrgRoleViewer:
		return grafanaOrgRoleViewer
	case grafanaOrgRoleEditor, grafanaOrgRoleAdmin, grafanaOrgRoleGrafanaAdmin:
		return grafanaOrgRoleViewer
	default:
		return grafanaOrgRoleNone
	}
}

// GetCurrentUser returns current Grafana user.
// If anonymous mode is enabled and no auth headers are present, it returns
// a synthetic anonymous user when /api/user responds with 401.
func (c *Client) GetCurrentUser(ctx context.Context, authHeaders http.Header) (CurrentUser, error) {
	var user CurrentUser
	err := c.do(ctx, http.MethodGet, "/api/user", "", authHeaders, nil, &user)
	if err == nil {
		return user, nil
	}

	if hasAuthorizationHeader(authHeaders) {
		return CurrentUser{}, err
	}
	var cErr *clientError
	if !errors.As(err, &cErr) {
		return CurrentUser{}, err
	}
	if cErr.Code != http.StatusUnauthorized {
		return CurrentUser{}, err
	}

	settings, settingsErr := c.getFrontendSettings(ctx)
	if settingsErr != nil || !settings.AnonymousEnabled {
		return CurrentUser{}, err
	}
	role := resolveAnonymousOrgRole(settings.AnonymousOrgRole)
	if role == grafanaOrgRoleNone {
		return CurrentUser{}, err
	}

	orgID := settings.User.OrgID
	if orgID == 0 {
		orgID = 1
	}

	return CurrentUser{
		ID:             0,
		Email:          "",
		Name:           "Anonymous",
		Login:          "anonymous",
		OrgID:          orgID,
		IsAnonymous:    true,
		IsGrafanaAdmin: false,
	}, nil
}

// GetCurrentUserOrgs returns current Grafana user organizations.
// If anonymous mode is enabled and no auth headers are present, it returns
// a synthetic org list when /api/user/orgs responds with 401.
func (c *Client) GetCurrentUserOrgs(ctx context.Context, authHeaders http.Header) ([]CurrentUserOrg, error) {
	var orgs []CurrentUserOrg
	err := c.do(ctx, http.MethodGet, "/api/user/orgs", "", authHeaders, nil, &orgs)
	if err == nil {
		return orgs, nil
	}

	if hasAuthorizationHeader(authHeaders) {
		return nil, err
	}
	var cErr *clientError
	if !errors.As(err, &cErr) {
		return nil, err
	}
	if cErr.Code != http.StatusUnauthorized {
		return nil, err
	}

	settings, settingsErr := c.getFrontendSettings(ctx)
	if settingsErr != nil || !settings.AnonymousEnabled {
		return nil, err
	}
	role := resolveAnonymousOrgRole(settings.AnonymousOrgRole)
	if role == grafanaOrgRoleNone {
		return nil, err
	}

	orgID := settings.User.OrgID
	if orgID == 0 {
		orgID = 1
	}
	orgName := settings.User.OrgName
	if orgName == "" {
		orgName = "Main Org."
	}

	return []CurrentUserOrg{
		{
			OrgID: orgID,
			Name:  orgName,
			Role:  role,
		},
	}, nil
}

func (c *Client) getRoleForServiceToken(ctx context.Context, token string) (role, error) {
	header := http.Header{}
	header.Add("Authorization", "Bearer "+token)

	var k map[string]any
	err := c.do(ctx, http.MethodGet, "/api/auth/serviceaccount", "", header, nil, &k)
	if err != nil {
		return none, err
	}

	if id, _ := k["orgId"].(float64); id != 1 {
		return none, nil
	}

	role, _ := k["role"].(string)
	return c.convertRole(role), nil
}

type serviceAccountSearch struct {
	TotalCount      int              `json:"totalCount"`
	ServiceAccounts []serviceAccount `json:"serviceAccounts"`
}

func (c *Client) getServiceAccountIDFromName(ctx context.Context, nodeName string, authHeaders http.Header) (int, error) {
	var res serviceAccountSearch
	serviceAccountName := grafana.SanitizeSAName(fmt.Sprintf("%s-%s", pmmServiceAccountName, nodeName))
	err := c.do(ctx, http.MethodGet, "/api/serviceaccounts/search", "query="+serviceAccountName, authHeaders, nil, &res)
	if err != nil {
		return 0, err
	}
	for _, serviceAccount := range res.ServiceAccounts {
		if serviceAccount.Name != serviceAccountName {
			continue
		}
		return serviceAccount.ID, nil
	}

	return 0, fmt.Errorf("service account %s not found", serviceAccountName)
}

func (c *Client) getNotPMMAgentTokenCountForServiceAccount(ctx context.Context, nodeName string) (int, error) {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return 0, err
	}

	serviceAccountID, err := c.getServiceAccountIDFromName(ctx, nodeName, authHeaders)
	if err != nil {
		return 0, err
	}

	var tokens []serviceToken
	err = c.do(ctx, http.MethodGet, fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, nil, &tokens)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, token := range tokens {
		serviceTokenName := fmt.Sprintf("%s-%s", pmmServiceTokenName, nodeName)
		if !strings.HasPrefix(token.Name, grafana.SanitizeSAName(serviceTokenName)) {
			count++
		}
	}

	return count, nil
}

func (c *Client) testCreateUser(ctx context.Context, login string, role role, authHeaders http.Header) (int, error) {
	// https://grafana.com/docs/http_api/admin/#global-users
	b, err := json.Marshal(map[string]string{
		"name":     login,
		"email":    login + "@percona.invalid",
		"login":    login,
		"password": login,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal a new user request body: %w", err)
	}
	var m map[string]any
	err = c.do(ctx, "POST", "/api/admin/users", "", authHeaders, b, &m)
	if err != nil {
		return 0, err
	}
	userID := int(m["id"].(float64)) //nolint:forcetypeassert

	// settings in grafana.ini should make a viewer by default
	if role < editor {
		return userID, nil
	}

	// https://grafana.com/docs/http_api/org/#updates-the-given-user
	b, err = json.Marshal(map[string]string{
		"role": role.String(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal a new user role: %w", err)
	}
	err = c.do(ctx, "PATCH", "/api/org/users/"+strconv.Itoa(userID), "", authHeaders, b, nil)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (c *Client) testDeleteUser(ctx context.Context, userID int, authHeaders http.Header) error {
	// https://grafana.com/docs/http_api/admin/#delete-global-user
	return c.do(ctx, "DELETE", "/api/admin/users/"+strconv.Itoa(userID), "", authHeaders, nil, nil)
}

// CreateServiceAccount creates service account and token with Admin role.
func (c *Client) CreateServiceAccount(ctx context.Context, nodeName string, reregister bool) (int, string, error) {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return 0, "", err
	}

	serviceAccountID, err := c.createServiceAccount(ctx, admin, nodeName, reregister, authHeaders)
	if err != nil {
		return 0, "", err
	}

	_, serviceToken, err := c.createServiceToken(ctx, serviceAccountID, nodeName, reregister, authHeaders)
	if err != nil {
		return 0, "", err
	}

	return serviceAccountID, serviceToken, nil
}

// DeleteServiceAccount deletes service account by current service token.
func (c *Client) DeleteServiceAccount(ctx context.Context, nodeName string, force bool) (string, error) {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return "", err
	}

	warning := ""
	serviceAccountID, err := c.getServiceAccountIDFromName(ctx, nodeName, authHeaders)
	if err != nil {
		return warning, err
	}

	customsTokensCount, err := c.getNotPMMAgentTokenCountForServiceAccount(ctx, nodeName)
	if err != nil {
		return warning, err
	}

	if !force && customsTokensCount > 0 {
		warning = "Service account wont be deleted, because there are more not PMM agent related service tokens."
		err = c.deletePMMAgentServiceToken(ctx, serviceAccountID, nodeName, authHeaders)
	} else {
		err = c.deleteServiceAccount(ctx, serviceAccountID, authHeaders)
	}
	if err != nil {
		return warning, err
	}

	return warning, err
}

// CreateAlertRule creates Grafana alert rule.
func (c *Client) CreateAlertRule(ctx context.Context, folderUID, groupName, interval string, rule *services.Rule) error {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return err
	}

	type AlertRuleGroup struct {
		Name     string            `json:"name"`
		Interval string            `json:"interval"`
		Rules    []json.RawMessage `json:"rules"`
	}

	var group AlertRuleGroup
	err = c.do(ctx, http.MethodGet, fmt.Sprintf("/api/ruler/grafana/api/v1/rules/%s/%s", folderUID, groupName), "", authHeaders, nil, &group)
	clientErr := &clientError{}

	switch {
	// Initialize rule group if not present
	case errors.As(err, &clientErr) && clientErr.Code == http.StatusNotFound:
		group.Name = groupName
		group.Rules = []json.RawMessage{}
	case err != nil:
		return err
	}

	b, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	group.Rules = append(group.Rules, b)

	if group.Interval == "" {
		group.Interval = interval
	}

	err = validateDurations(group.Interval, rule.For)
	if err != nil {
		return err
	}

	body, err := json.Marshal(group)
	if err != nil {
		return err
	}

	err = c.do(ctx, "POST", "/api/ruler/grafana/api/v1/rules/"+folderUID, "", authHeaders, body, nil)
	if err != nil {
		cErr, ok := errors.AsType[*clientError](err)
		if ok {
			return status.Error(codes.InvalidArgument, cErr.ErrorMessage)
		}
		return err
	}

	return nil
}

// DeleteAlertRuleGroup deletes a Grafana-managed alert rule group from the given folder.
func (c *Client) DeleteAlertRuleGroup(ctx context.Context, folderUID, groupName string) error {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return err
	}

	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/ruler/grafana/api/v1/rules/%s/%s", folderUID, groupName), "", authHeaders, nil, nil)
}

func validateDurations(intervalD, forD string) error {
	i, err := time.ParseDuration(intervalD)
	if err != nil {
		return err
	}

	f, err := time.ParseDuration(forD)
	if err != nil {
		return err
	}

	if f < i {
		return status.Errorf(codes.InvalidArgument, "Duration (%s) can't be shorter than evaluation interval for the given group (%s).", forD, intervalD)
	}

	return nil
}

// GetDatasourceUIDByID returns grafana datasource UID.
func (c *Client) GetDatasourceUIDByID(ctx context.Context, id int64) (string, error) {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create grafana client: %w", err)
	}

	ds, err := grafanaClient.DataSource(id)
	if err != nil {
		return "", err
	}
	return ds.UID, nil
}

// CreateFolder creates grafana folder.
func (c *Client) CreateFolder(ctx context.Context, title string) (*gapi.Folder, error) {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create grafana client: %w", err)
	}

	folder, err := grafanaClient.NewFolder(title)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return &folder, nil
}

// CreateFolderWithUID creates a grafana folder with a specific UID.
func (c *Client) CreateFolderWithUID(ctx context.Context, title, uid string) error {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return err
	}

	body, err := json.Marshal(struct {
		UID   string `json:"uid"`
		Title string `json:"title"`
	}{UID: uid, Title: title})
	if err != nil {
		return err
	}

	return c.do(ctx, http.MethodPost, "/api/folders", "", authHeaders, body, nil)
}

// DeleteFolder deletes grafana folder.
func (c *Client) DeleteFolder(ctx context.Context, id string, force bool) error {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create grafana client: %w", err)
	}

	params := make(url.Values)
	if force {
		params.Add("forceDeleteRules", "true")
	}

	err = grafanaClient.DeleteFolder(id, params)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// GetFolderByUID returns folder with given UID.
func (c *Client) GetFolderByUID(ctx context.Context, uid string) (*gapi.Folder, error) {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create grafana client: %w", err)
	}

	folder, err := grafanaClient.FolderByUID(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to find folder: %w", err)
	}

	return folder, nil
}

func (c *Client) createGrafanaClient(ctx context.Context) (*gapi.Client, error) {
	authHeaders, err := auth.GetHeadersFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers from incoming context: %w", err)
	}

	headers := make(map[string]string, len(authHeaders))
	for k := range authHeaders {
		headers[k] = authHeaders.Get(k)
	}

	grafanaClient, err := gapi.New("http://"+c.addr, gapi.Config{HTTPHeaders: headers})
	if err != nil {
		return nil, fmt.Errorf("failed to create a new grafana client: %w", err)
	}

	return grafanaClient, nil
}

type serviceAccount struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Force bool   `json:"force"`
}
type serviceToken struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func (c *Client) createServiceAccount(ctx context.Context, role role, nodeName string, reregister bool, authHeaders http.Header) (int, error) {
	if role == none {
		return 0, errors.New("you cannot create service account with empty role")
	}

	serviceAccountName := fmt.Sprintf("%s-%s", pmmServiceAccountName, nodeName)
	b, err := json.Marshal(serviceAccount{Name: serviceAccountName, Role: role.String(), Force: reregister})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal service account: %w", err)
	}

	var m map[string]any
	err = c.do(ctx, "POST", "/api/serviceaccounts", "", authHeaders, b, &m)
	if err != nil {
		return 0, err
	}

	serviceAccountID := int(m["id"].(float64)) //nolint:forcetypeassert

	// orgId is ignored during creating service account and default is -1
	// orgId should be set to 1
	err = c.do(ctx, "PATCH", fmt.Sprintf("/api/serviceaccounts/%d", serviceAccountID), "", authHeaders, []byte("{\"orgId\": 1}"), &m)
	if err != nil {
		return 0, err
	}

	return serviceAccountID, nil
}

func (c *Client) createServiceToken(ctx context.Context, serviceAccountID int, nodeName string, reregister bool, authHeaders http.Header) (int, string, error) {
	serviceTokenName := fmt.Sprintf("%s-%s", pmmServiceTokenName, nodeName)
	exists, err := c.serviceTokenExists(ctx, serviceAccountID, nodeName, authHeaders)
	if err != nil {
		return 0, "", err
	}
	if exists && reregister {
		err := c.deletePMMAgentServiceToken(ctx, serviceAccountID, nodeName, authHeaders)
		if err != nil {
			return 0, "", err
		}
	}

	b, err := json.Marshal(serviceToken{Name: serviceTokenName, Role: admin.String()})
	if err != nil {
		return 0, "", fmt.Errorf("failed to marshal service token: %w", err)
	}

	var m map[string]any
	err = c.do(ctx, "POST", fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, b, &m)
	if err != nil {
		return 0, "", err
	}
	serviceTokenID := int(m["id"].(float64)) //nolint:forcetypeassert
	serviceTokenKey := m["key"].(string)     //nolint:forcetypeassert

	return serviceTokenID, serviceTokenKey, nil
}

func (c *Client) serviceTokenExists(ctx context.Context, serviceAccountID int, nodeName string, authHeaders http.Header) (bool, error) {
	var tokens []serviceToken
	err := c.do(ctx, "GET", fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, nil, &tokens)
	if err != nil {
		return false, err
	}

	serviceTokenName := fmt.Sprintf("%s-%s", pmmServiceTokenName, nodeName)
	for _, token := range tokens {
		if !strings.HasPrefix(token.Name, grafana.SanitizeSAName(serviceTokenName)) {
			continue
		}

		return true, nil
	}

	return false, nil
}

func (c *Client) deletePMMAgentServiceToken(ctx context.Context, serviceAccountID int, nodeName string, authHeaders http.Header) error {
	var tokens []serviceToken
	err := c.do(ctx, "GET", fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, nil, &tokens)
	if err != nil {
		return err
	}

	serviceTokenName := fmt.Sprintf("%s-%s", pmmServiceTokenName, nodeName)
	for _, token := range tokens {
		if strings.HasPrefix(token.Name, grafana.SanitizeSAName(serviceTokenName)) {
			err := c.do(ctx, "DELETE", fmt.Sprintf("/api/serviceaccounts/%d/tokens/%d", serviceAccountID, token.ID), "", authHeaders, nil, nil)
			if err != nil {
				return err
			}

			return nil
		}
	}

	return nil
}

func (c *Client) deleteServiceAccount(ctx context.Context, serviceAccountID int, authHeaders http.Header) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/api/serviceaccounts/%d", serviceAccountID), "", authHeaders, nil, nil)
}

// Annotation contains grafana annotation response.
type annotation struct {
	Time time.Time `json:"-"`
	Tags []string  `json:"tags,omitempty"`
	Text string    `json:"text,omitempty"`

	TimeInt int64 `json:"time,omitempty"`
}

// encode annotation before sending request.
func (a *annotation) encode() {
	var t int64
	if !a.Time.IsZero() {
		t = a.Time.UnixNano() / int64(time.Millisecond)
	}
	a.TimeInt = t
}

// decode annotation after receiving response.
func (a *annotation) decode() {
	var t time.Time
	if a.TimeInt != 0 {
		t = time.Unix(0, a.TimeInt*int64(time.Millisecond))
	}
	a.Time = t
}

// CreateAnnotation creates annotation with given text and tags ("pmm_annotation" is added automatically)
// and returns Grafana's response text which is typically "Annotation added" or "Failed to save annotation".
func (c *Client) CreateAnnotation(ctx context.Context, tags []string, from time.Time, text, authorization string) (string, error) {
	// http://docs.grafana.org/http_api/annotations/#create-annotation
	request := &annotation{
		Tags: tags,
		Text: text,
		Time: from,
	}
	request.encode()

	b, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	headers := make(http.Header)
	headers.Add("Authorization", authorization)

	var response struct {
		Message string `json:"message"`
	}

	err = c.do(ctx, "POST", "/api/annotations", "", headers, b, &response)
	if err != nil {
		return "", fmt.Errorf("failed to create annotation: %w", err)
	}

	return response.Message, nil
}

func (c *Client) findAnnotations(ctx context.Context, from, to time.Time, authorization string) ([]annotation, error) {
	// http://docs.grafana.org/http_api/annotations/#find-annotations

	headers := make(http.Header)
	headers.Add("Authorization", authorization)

	params := url.Values{
		"from": []string{strconv.FormatInt(from.UnixNano()/int64(time.Millisecond), 10)},
		"to":   []string{strconv.FormatInt(to.UnixNano()/int64(time.Millisecond), 10)},
	}.Encode()

	var response []annotation
	err := c.do(ctx, http.MethodGet, "/api/annotations", params, headers, nil, &response)
	if err != nil {
		return nil, err
	}

	for i, r := range response {
		r.decode()
		response[i] = r
	}
	return response, nil
}

type grafanaHealthResponse struct {
	Commit   string `json:"commit"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

// IsReady calls Grafana API to check its status.
func (c *Client) IsReady(ctx context.Context) error {
	var status grafanaHealthResponse
	err := c.do(ctx, http.MethodGet, "/api/health", "", nil, nil, &status)
	if err != nil {
		return fmt.Errorf("grafana health check failed: %w", err)
	}

	if strings.ToLower(status.Database) != "ok" {
		return fmt.Errorf("grafana health check failure: database status is %s", status.Database)
	}

	return nil
}

const grpcGatewayCookie = "grpcgateway-cookie"

type currentUser struct {
	AccessToken string `json:"access_token"`
}

var errCookieIsNotSet = fmt.Errorf("cookie %q is not set", grpcGatewayCookie)

// GetCurrentUserAccessToken return users access token from Grafana.
func (c *Client) GetCurrentUserAccessToken(ctx context.Context) (string, error) {
	// We need to set cookie to the request to make it execute in grafana user context.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("metadata not set in the context: %w", errCookieIsNotSet)
	}
	cookies := md.Get(grpcGatewayCookie)
	if len(cookies) == 0 {
		return "", errCookieIsNotSet
	}
	headers := http.Header{}
	headers.Set("Cookie", strings.Join(cookies, "; "))

	var user currentUser
	err := c.do(ctx, http.MethodGet, "/graph/percona-api/user/oauth-token", "", headers, nil, &user)
	if err != nil {
		var e *clientError
		if errors.As(err, &e) && e.ErrorMessage == "Failed to get token" && e.Code == http.StatusInternalServerError {
			return "", ErrFailedToGetToken
		}
		return "", fmt.Errorf("unknown error occurred during getting of user's token: %w", err)
	}

	return user.AccessToken, nil
}

// check interfaces.
var (
	_ prom.Collector = (*Client)(nil)
	_ error          = (*clientError)(nil)
	_ fmt.Stringer   = role(0)
)
