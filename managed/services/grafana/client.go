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
	"github.com/percona/pmm/managed/utils/auth"
	"github.com/percona/pmm/managed/utils/irt"
	"github.com/percona/pmm/utils/grafana"
)

// ErrFailedToGetToken means it failed to get the user token. Most likely due to the fact the user is not logged in using Percona Account.
var ErrFailedToGetToken = errors.New("failed to get the user token")

const (
	pmmServiceTokenName   = "pmm-agent-st" //nolint:gosec
	pmmServiceAccountName = "pmm-agent-sa" //nolint:gosec
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
func (c *Client) do(ctx context.Context, method, path, rawQuery string, headers http.Header, body []byte, target interface{}) error {
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
	if resp.StatusCode < 200 || resp.StatusCode > 202 {
		cErr := &clientError{
			Method: req.Method,
			URL:    req.URL.String(),
			Code:   resp.StatusCode,
			Body:   string(b),
		}
		_ = json.Unmarshal(b, cErr) // add ErrorMessage
		return errors.WithStack(cErr)
	}

	if len(b) > 0 && target != nil {
		if err = json.Unmarshal(b, target); err != nil {
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
	authHeaders, err := auth.GetHeadersFromContext(ctx)
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
		if err == nil {
			return authUser{
				role:   role,
				userID: 0,
			}, nil
		}

		if strings.Contains(err.Error(), "Auth method is not service account token") {
			role, err := c.getRoleForAPIKey(ctx, authHeaders)
			if err == nil {
				l.Warning("you should migrate your API Key to a Service Account")
			}
			return authUser{
				role:   role,
				userID: 0,
			}, err
		}

		return emptyUser, err
	}

	// https://grafana.com/docs/http_api/user/#actual-user - works only with Basic Auth
	var m map[string]interface{}
	err := c.do(ctx, http.MethodGet, "/api/user", "", authHeaders, nil, &m)
	if err != nil {
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

func (c *Client) getRoleForServiceToken(ctx context.Context, token string) (role, error) {
	header := http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	var k map[string]interface{}
	if err := c.do(ctx, http.MethodGet, "/api/auth/serviceaccount", "", header, nil, &k); err != nil {
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
	if err := c.do(ctx, http.MethodGet, "/api/serviceaccounts/search", fmt.Sprintf("query=%s", serviceAccountName), authHeaders, nil, &res); err != nil {
		return 0, err
	}
	for _, serviceAccount := range res.ServiceAccounts {
		if serviceAccount.Name != serviceAccountName {
			continue
		}
		return serviceAccount.ID, nil
	}

	return 0, errors.Errorf("service account %s not found", serviceAccountName)
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, nil, &tokens); err != nil {
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/ruler/grafana/api/v1/rules/%s/%s", folderUID, groupName), "", authHeaders, nil, &group); err != nil {
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

	if err = validateDurations(group.Interval, rule.For); err != nil {
		return err
	}

	body, err := json.Marshal(group)
	if err != nil {
		return err
	}

	if err := c.do(ctx, "POST", fmt.Sprintf("/api/ruler/grafana/api/v1/rules/%s", folderUID), "", authHeaders, body, nil); err != nil {
		if cErr, ok := errors.Cause(err).(*clientError); ok { //nolint:errorlint
			return status.Error(codes.InvalidArgument, cErr.ErrorMessage)
		}
		return err
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

// CreateFolder creates grafana folder.
func (c *Client) CreateFolder(ctx context.Context, title string) (*gapi.Folder, error) {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create grafana client")
	}

	folder, err := grafanaClient.NewFolder(title)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create folder")
	}

	return &folder, nil
}

// DeleteFolder deletes grafana folder.
func (c *Client) DeleteFolder(ctx context.Context, id string, force bool) error {
	grafanaClient, err := c.createGrafanaClient(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create grafana client")
	}

	params := make(url.Values)
	if force {
		params.Add("forceDeleteRules", "true")
	}

	err = grafanaClient.DeleteFolder(id, params)
	if err != nil {
		return errors.Wrap(err, "failed to delete folder")
	}

	return nil
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
	authHeaders, err := auth.GetHeadersFromContext(ctx)
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
		return 0, errors.WithStack(err)
	}

	var m map[string]interface{}
	if err = c.do(ctx, "POST", "/api/serviceaccounts", "", authHeaders, b, &m); err != nil {
		return 0, err
	}

	serviceAccountID := int(m["id"].(float64)) //nolint:forcetypeassert

	// orgId is ignored during creating service account and default is -1
	// orgId should be set to 1
	if err = c.do(ctx, "PATCH", fmt.Sprintf("/api/serviceaccounts/%d", serviceAccountID), "", authHeaders, []byte("{\"orgId\": 1}"), &m); err != nil {
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
		return 0, "", errors.WithStack(err)
	}

	var m map[string]interface{}
	if err = c.do(ctx, "POST", fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, b, &m); err != nil {
		return 0, "", err
	}
	serviceTokenID := int(m["id"].(float64)) //nolint:forcetypeassert
	serviceTokenKey := m["key"].(string)     //nolint:forcetypeassert

	return serviceTokenID, serviceTokenKey, nil
}

func (c *Client) serviceTokenExists(ctx context.Context, serviceAccountID int, nodeName string, authHeaders http.Header) (bool, error) {
	var tokens []serviceToken
	if err := c.do(ctx, "GET", fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, nil, &tokens); err != nil {
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
	if err := c.do(ctx, "GET", fmt.Sprintf("/api/serviceaccounts/%d/tokens", serviceAccountID), "", authHeaders, nil, &tokens); err != nil {
		return err
	}

	serviceTokenName := fmt.Sprintf("%s-%s", pmmServiceTokenName, nodeName)
	for _, token := range tokens {
		if strings.HasPrefix(token.Name, grafana.SanitizeSAName(serviceTokenName)) {
			if err := c.do(ctx, "DELETE", fmt.Sprintf("/api/serviceaccounts/%d/tokens/%d", serviceAccountID, token.ID), "", authHeaders, nil, nil); err != nil {
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
	if err := c.do(ctx, http.MethodGet, "/graph/percona-api/user/oauth-token", "", headers, nil, &user); err != nil {
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
