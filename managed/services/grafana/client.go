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

// Package grafana provides facilities for working with Grafana.
package grafana

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/percona/pmm/managed/services"
	"github.com/percona/pmm/managed/utils/irt"
)

// ErrFailedToGetToken means it failed to get user's token. Most likely due to the fact user is not logged in using Percona Account.
var ErrFailedToGetToken = errors.New("failed to get token")

const defaultEvaluationInterval = time.Minute

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
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
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
type clientError struct { //nolint:musttag
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

// do makes HTTP request with given parameters, and decodes JSON response with 200 OK status
// to respBody. It returns wrapped clientError on any other status, or other fatal errors.
// Ctx is used only for cancelation.
func (c *Client) do(ctx context.Context, method, path, rawQuery string, headers http.Header, body []byte, respBody interface{}) error {
	u := url.URL{
		Scheme:   "http",
		Host:     c.addr,
		Path:     path,
		RawQuery: rawQuery,
	}
	req, err := http.NewRequest(method, u.String(), bytes.NewReader(body))
	if err != nil {
		return errors.WithStack(err)
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
		return errors.WithStack(err)
	}
	defer resp.Body.Close() //nolint:gosec,errcheck,nolintlint

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		cErr := &clientError{
			Method: req.Method,
			URL:    req.URL.String(),
			Code:   resp.StatusCode,
			Body:   string(b),
		}
		_ = json.Unmarshal(b, cErr) // add ErrorMessage
		return errors.WithStack(cErr)
	}

	if respBody != nil {
		if err = json.Unmarshal(b, respBody); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

type authUser struct {
	role   role
	userID int
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
	authHeaders, err := c.authHeadersFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var m map[string]interface{}
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

// getAuthUser returns grafanaAdmin if currently authenticated user is a Grafana (super) admin.
// Otherwise, it returns a role in the default organization (with ID 1).
// Ctx is used only for cancelation.
func (c *Client) getAuthUser(ctx context.Context, authHeaders http.Header) (authUser, error) {
	// Check if it's API Key
	if c.IsAPIKeyAuth(authHeaders) {
		role, err := c.getRoleForAPIKey(ctx, authHeaders)
		return authUser{
			role:   role,
			userID: 0,
		}, err
	}

	// https://grafana.com/docs/http_api/user/#actual-user - works only with Basic Auth
	var m map[string]interface{}
	err := c.do(ctx, http.MethodGet, "/api/user", "", authHeaders, nil, &m)
	if err != nil {
		return authUser{
			role:   none,
			userID: 0,
		}, err
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
	var s []interface{}
	if err := c.do(ctx, http.MethodGet, "/api/user/orgs", "", authHeaders, nil, &s); err != nil {
		return authUser{
			role:   none,
			userID: userID,
		}, err
	}

	for _, el := range s {
		m, _ := el.(map[string]interface{})
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

// IsAPIKeyAuth checks if the request is made using an API Key.
func (c *Client) IsAPIKeyAuth(headers http.Header) bool {
	authHeader := headers.Get("Authorization")
	switch {
	case strings.HasPrefix(authHeader, "Bearer"):
		return true
	case strings.HasPrefix(authHeader, "Basic"):
		h := strings.TrimPrefix(authHeader, "Basic")
		d, err := base64.StdEncoding.DecodeString(strings.TrimSpace(h))
		if err != nil {
			return false
		}
		return strings.HasPrefix(string(d), "api_key:")
	}
	return false
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

type apiKey struct {
	ID         int64      `json:"id"`
	OrgID      int64      `json:"orgId,omitempty"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	Expiration *time.Time `json:"expiration,omitempty"`
}

func (c *Client) getRoleForAPIKey(ctx context.Context, authHeaders http.Header) (role, error) {
	var k map[string]interface{}
	if err := c.do(ctx, http.MethodGet, "/api/auth/key", "", authHeaders, nil, &k); err != nil {
		return none, err
	}

	if id, _ := k["orgId"].(float64); id != 1 {
		return none, nil
	}

	role, _ := k["role"].(string)
	return c.convertRole(role), nil
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
		return 0, errors.WithStack(err)
	}
	var m map[string]interface{}
	if err = c.do(ctx, "POST", "/api/admin/users", "", authHeaders, b, &m); err != nil {
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
		return 0, errors.WithStack(err)
	}
	if err = c.do(ctx, "PATCH", "/api/org/users/"+strconv.Itoa(userID), "", authHeaders, b, nil); err != nil {
		return 0, err
	}
	return userID, nil
}

func (c *Client) testDeleteUser(ctx context.Context, userID int, authHeaders http.Header) error {
	// https://grafana.com/docs/http_api/admin/#delete-global-user
	return c.do(ctx, "DELETE", "/api/admin/users/"+strconv.Itoa(userID), "", authHeaders, nil, nil)
}

// CreateAdminAPIKey creates API key with Admin role and provided name.
func (c *Client) CreateAdminAPIKey(ctx context.Context, name string) (int64, string, error) {
	authHeaders, err := c.authHeadersFromContext(ctx)
	if err != nil {
		return 0, "", err
	}
	return c.createAPIKey(ctx, name, admin, authHeaders)
}

// DeleteAPIKeysWithPrefix deletes all API keys with provided prefix. If there is no api key with provided prefix just ignores it.
func (c *Client) DeleteAPIKeysWithPrefix(ctx context.Context, prefix string) error {
	authHeaders, err := c.authHeadersFromContext(ctx)
	if err != nil {
		return err
	}
	var keys []apiKey
	if err := c.do(ctx, http.MethodGet, "/api/auth/keys", "", authHeaders, nil, &keys); err != nil {
		return err
	}

	for _, k := range keys {
		if strings.HasPrefix(k.Name, prefix) {
			err := c.deleteAPIKey(ctx, k.ID, authHeaders)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteAPIKeyByID deletes API key by ID.
func (c *Client) DeleteAPIKeyByID(ctx context.Context, id int64) error {
	authHeaders, err := c.authHeadersFromContext(ctx)
	if err != nil {
		return err
	}
	return c.deleteAPIKey(ctx, id, authHeaders)
}

// CreateAlertRule creates Grafana alert rule.
func (c *Client) CreateAlertRule(ctx context.Context, folderName, groupName string, rule *services.Rule) error {
	authHeaders, err := c.authHeadersFromContext(ctx)
	if err != nil {
		return err
	}

	type AlertRuleGroup struct {
		Name     string            `json:"name"`
		Interval string            `json:"interval"`
		Rules    []json.RawMessage `json:"rules"`
	}

	var group AlertRuleGroup
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/ruler/grafana/api/v1/rules/%s/%s", folderName, groupName), "", authHeaders, nil, &group); err != nil {
		return err
	}

	b, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	group.Rules = append(group.Rules, b)

	if group.Interval == "" {
		// TODO: align it with grafanas default value: https://grafana.com/docs/grafana/v9.0/setup-grafana/configure-grafana/#min_interval
		group.Interval = defaultEvaluationInterval.String()
	}

	if err = validateDurations(group.Interval, rule.For); err != nil {
		return err
	}

	body, err := json.Marshal(group)
	if err != nil {
		return err
	}

	if err := c.do(ctx, "POST", fmt.Sprintf("/api/ruler/grafana/api/v1/rules/%s", folderName), "", authHeaders, body, nil); err != nil {
		if err != nil {
			if cErr, ok := errors.Cause(err).(*clientError); ok { //nolint:errorlint
				return status.Error(codes.InvalidArgument, cErr.ErrorMessage)
			}
			return err
		}
	}

	return nil
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
		return "", errors.Wrap(err, "failed to create grafana client")
	}

	ds, err := grafanaClient.DataSource(id)
	if err != nil {
		return "", err
	}
	return ds.UID, nil
}

// GetFolderByUID returns folder with given UID.
func (c *Client) GetFolderByUID(ctx context.Context, uid string) (*gapi.Folder, error) {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create grafana client")
	}

	folder, err := grafanaClient.FolderByUID(uid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find folder")
	}

	return folder, nil
}

func (c *Client) createGrafanaClient(ctx context.Context) (*gapi.Client, error) {
	authHeaders, err := c.authHeadersFromContext(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	headers := make(map[string]string, len(authHeaders))
	for k := range authHeaders {
		headers[k] = authHeaders.Get(k)
	}

	grafanaClient, err := gapi.New("http://"+c.addr, gapi.Config{HTTPHeaders: headers})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return grafanaClient, nil
}

func (c *Client) authHeadersFromContext(ctx context.Context) (http.Header, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("cannot get headers from metadata")
	}
	// get authorization from headers.
	authorizationHeaders := headers.Get("Authorization")
	cookieHeaders := headers.Get("grpcgateway-cookie")
	if len(authorizationHeaders) == 0 && len(cookieHeaders) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Authorization error.")
	}

	authHeaders := make(http.Header)
	if len(authorizationHeaders) != 0 {
		authHeaders.Add("Authorization", authorizationHeaders[0])
	}
	if len(cookieHeaders) != 0 {
		for _, header := range cookieHeaders {
			authHeaders.Add("Cookie", header)
		}
	}
	return authHeaders, nil
}

func (c *Client) createAPIKey(ctx context.Context, name string, role role, authHeaders http.Header) (int64, string, error) {
	// https://grafana.com/docs/grafana/latest/http_api/auth/#create-api-key
	b, err := json.Marshal(apiKey{Name: name, Role: role.String()})
	if err != nil {
		return 0, "", errors.WithStack(err)
	}
	var m map[string]interface{}
	if err = c.do(ctx, "POST", "/api/auth/keys", "", authHeaders, b, &m); err != nil {
		return 0, "", err
	}
	key := m["key"].(string) //nolint:forcetypeassert

	apiAuthHeaders := http.Header{}
	apiAuthHeaders.Set("Authorization", fmt.Sprintf("Bearer %s", key))

	var k apiKey
	if err := c.do(ctx, http.MethodGet, "/api/auth/key", "", apiAuthHeaders, nil, &k); err != nil {
		return 0, "", err
	}
	apiKeyID := k.ID

	return apiKeyID, key, nil
}

func (c *Client) deleteAPIKey(ctx context.Context, apiKeyID int64, authHeaders http.Header) error {
	// https://grafana.com/docs/grafana/latest/http_api/auth/#delete-api-key
	return c.do(ctx, "DELETE", "/api/auth/keys/"+strconv.FormatInt(apiKeyID, 10), "", authHeaders, nil, nil)
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
		return "", errors.Wrap(err, "failed to marshal request")
	}

	headers := make(http.Header)
	headers.Add("Authorization", authorization)

	var response struct {
		Message string `json:"message"`
	}

	if err := c.do(ctx, "POST", "/api/annotations", "", headers, b, &response); err != nil {
		return "", errors.Wrap(err, "failed to create annotation")
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
	if err := c.do(ctx, http.MethodGet, "/api/annotations", params, headers, nil, &response); err != nil {
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
	if err := c.do(ctx, http.MethodGet, "/api/health", "", nil, nil, &status); err != nil {
		// since we don't return the error to the user, log it to help debugging
		logrus.Errorf("grafana status check failed: %s", err)
		return fmt.Errorf("cannot reach Grafana API")
	}

	if strings.ToLower(status.Database) != "ok" {
		logrus.Errorf("grafana is up but the database is not ok. Database status is %s", status.Database)
		return fmt.Errorf("grafana is running with errors")
	}

	return nil
}

const grpcGatewayCookie = "grpcgateway-cookie"

type currentUser struct {
	AccessToken string `json:"access_token"`
}

var errCookieIsNotSet = errors.Errorf("cookie %q is not set", grpcGatewayCookie)

// GetCurrentUserAccessToken return users access token from Grafana.
func (c *Client) GetCurrentUserAccessToken(ctx context.Context) (string, error) {
	// We need to set cookie to the request to make it execute in grafana user context.
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.Wrap(errCookieIsNotSet, "metada not set in the context")
	}
	cookies := md.Get(grpcGatewayCookie)
	if len(cookies) == 0 {
		return "", errCookieIsNotSet
	}
	headers := http.Header{}
	headers.Set("Cookie", strings.Join(cookies, "; "))

	var user currentUser
	if err := c.do(ctx, http.MethodGet, "/percona-api/user/oauth-token", "", headers, nil, &user); err != nil {
		var e *clientError
		if errors.As(err, &e) && e.ErrorMessage == "Failed to get token" && e.Code == http.StatusInternalServerError {
			return "", ErrFailedToGetToken
		}
		return "", errors.Wrap(err, "unknown error occurred during getting of user's token")
	}

	return user.AccessToken, nil
}

// check interfaces.
var (
	_ prom.Collector = (*Client)(nil)
	_ error          = (*clientError)(nil)
	_ fmt.Stringer   = role(0)
)
