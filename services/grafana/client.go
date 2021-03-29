// pmm-managed
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

// Package grafana provides facilities for working with Grafana.
package grafana

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm-managed/utils/irt"
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

// do makes HTTP request with given parameters, and decodes JSON response with 200 OK status
// to respBody. It returns wrapped clientError on any other status, or other fatal errors.
// ctx is used only for cancelation.
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
	defer resp.Body.Close() //nolint:errcheck

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != 200 {
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

// getRole returns grafanaAdmin if currently authenticated user is a Grafana (super) admin.
// Otherwise, it returns a role in the default organization (with ID 1).
// ctx is used only for cancelation.
func (c *Client) getRole(ctx context.Context, authHeaders http.Header) (role, error) {
	// Check if it's API Key
	if c.isAPIKeyAuth(authHeaders.Get("Authorization")) {
		return c.getRoleForAPIKey(ctx, authHeaders)
	}

	// https://grafana.com/docs/http_api/user/#actual-user - works only with Basic Auth
	var m map[string]interface{}
	err := c.do(ctx, "GET", "/api/user", "", authHeaders, nil, &m)
	if err != nil {
		return none, err
	}

	if a, _ := m["isGrafanaAdmin"].(bool); a {
		return grafanaAdmin, nil
	}

	// works only with Basic auth
	var s []interface{}
	if err := c.do(ctx, "GET", "/api/user/orgs", "", authHeaders, nil, &s); err != nil {
		return none, err
	}

	for _, el := range s {
		m, _ := el.(map[string]interface{})
		if m == nil {
			continue
		}

		// check only default organization (with ID 1)
		if id, _ := m["orgId"].(float64); id == 1 {
			role, _ := m["role"].(string)
			return c.convertRole(role), nil
		}
	}

	return none, nil
}

func (c *Client) isAPIKeyAuth(authHeader string) bool {
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

func (c *Client) getRoleForAPIKey(ctx context.Context, authHeaders http.Header) (role, error) {
	var k map[string]interface{}
	if err := c.do(ctx, "GET", "/api/auth/key", "", authHeaders, nil, &k); err != nil {
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
	userID := int(m["id"].(float64))

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

func (c *Client) testCreateAPIKey(ctx context.Context, name string, role role, authHeaders http.Header) (int, string, error) {
	// https://grafana.com/docs/grafana/latest/http_api/auth/#create-api-key
	b, err := json.Marshal(map[string]string{
		"name": name,
		"role": role.String(),
	})
	if err != nil {
		return 0, "", errors.WithStack(err)
	}
	var m map[string]interface{}
	if err = c.do(ctx, "POST", "/api/auth/keys", "", authHeaders, b, &m); err != nil {
		return 0, "", err
	}
	apiKey := m["key"].(string)

	apiAuthHeaders := http.Header{}
	apiAuthHeaders.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	var k map[string]interface{}
	if err := c.do(ctx, "GET", "/api/auth/key", "", apiAuthHeaders, nil, &k); err != nil {
		return 0, "", err
	}
	apiKeyID := int(k["id"].(float64))

	return apiKeyID, apiKey, nil
}

func (c *Client) testDeleteAPIKey(ctx context.Context, apiKeyID int, authHeaders http.Header) error {
	// https://grafana.com/docs/grafana/latest/http_api/auth/#delete-api-key
	return c.do(ctx, "DELETE", "/api/auth/keys/"+strconv.Itoa(apiKeyID), "", authHeaders, nil, nil)
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
	if err := c.do(ctx, "GET", "/api/annotations", params, headers, nil, &response); err != nil {
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

// IsReady calls Grafana API to check its status
func (c *Client) IsReady(ctx context.Context) error {
	var status grafanaHealthResponse
	if err := c.do(ctx, "GET", "/api/health", "", nil, nil, &status); err != nil {
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

// check interfaces
var (
	_ prom.Collector = (*Client)(nil)
	_ error          = (*clientError)(nil)
	_ fmt.Stringer   = role(0)
)
