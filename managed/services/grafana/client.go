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

// Package grafana provides stub implementations for Grafana API interactions.
// All methods return static/no-op values since Grafana is not part of this build.
package grafana

import (
	"context"
	"fmt"
	"net/http"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/services"
)

// ErrFailedToGetToken means it failed to get the user token.
var ErrFailedToGetToken = fmt.Errorf("failed to get the user token")

const (
	pmmServiceTokenName   = "pmm-agent-st" //nolint:gosec
	pmmServiceAccountName = "pmm-agent-sa" //nolint:gosec
)

// Client is a stub for the Grafana API client. It satisfies all interfaces
// that expect *grafana.Client without making any HTTP calls.
type Client struct {
	addr string
}

// NewClient creates a new stub client for given Grafana address.
func NewClient(addr string) *Client {
	return &Client{addr: addr}
}

// Describe implements prometheus.Collector (no-op).
func (c *Client) Describe(ch chan<- *prom.Desc) {}

// Collect implements prometheus.Collector (no-op).
func (c *Client) Collect(ch chan<- prom.Metric) {}

// clientError contains error response details.
type clientError struct {
	Method       string
	URL          string
	Code         int
	Body         string
	ErrorMessage string `json:"message"`
}

// Error implements error interface.
func (e *clientError) Error() string {
	return fmt.Sprintf("clientError: %s %s -> %d %s", e.Method, e.URL, e.Code, e.Body)
}

type authUser struct {
	role   role
	userID int
}

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
		return "Viewer"
	case editor:
		return "Editor"
	case admin:
		return "Admin"
	case grafanaAdmin:
		return "GrafanaAdmin"
	default:
		return fmt.Sprintf("unexpected role %d", int(r))
	}
}

func (c *Client) GetUserID(ctx context.Context) (int, error) {
	return 1, nil
}

func (c *Client) getAuthUser(ctx context.Context, authHeaders http.Header, l *logrus.Entry) (authUser, error) {
	return authUser{role: grafanaAdmin, userID: 1}, nil
}

func (c *Client) CreateServiceAccount(ctx context.Context, nodeName string, reregister bool) (int, string, error) {
	return 1, "no-grafana-stub-token", nil
}

func (c *Client) DeleteServiceAccount(ctx context.Context, nodeName string, force bool) (string, error) {
	return "", nil
}

func (c *Client) CreateAlertRule(ctx context.Context, folderUID, groupName, interval string, rule *services.Rule) error {
	return nil
}

func (c *Client) GetDatasourceUIDByID(ctx context.Context, id int64) (string, error) {
	return "no-grafana-ds", nil
}

func (c *Client) CreateFolder(ctx context.Context, title string) (*gapi.Folder, error) {
	return &gapi.Folder{
		ID:    1,
		UID:   "no-grafana-folder",
		Title: title,
	}, nil
}

func (c *Client) DeleteFolder(ctx context.Context, id string, force bool) error {
	return nil
}

func (c *Client) GetFolderByUID(ctx context.Context, uid string) (*gapi.Folder, error) {
	return &gapi.Folder{
		ID:    1,
		UID:   uid,
		Title: "stub-folder",
	}, nil
}

func (c *Client) IsReady(ctx context.Context) error {
	return nil
}

func (c *Client) CreateAnnotation(ctx context.Context, tags []string, from time.Time, text, authorization string) (string, error) {
	return "Annotation skipped (no Grafana)", nil
}

func (c *Client) GetCurrentUserAccessToken(ctx context.Context) (string, error) {
	return "", ErrFailedToGetToken
}

func (c *Client) testCreateUser(ctx context.Context, login string, role role, authHeaders http.Header) (int, error) {
	return 1, nil
}

func (c *Client) testDeleteUser(ctx context.Context, userID int, authHeaders http.Header) error {
	return nil
}

// check interfaces.
var (
	_ prom.Collector = (*Client)(nil)
	_ error          = (*clientError)(nil)
	_ fmt.Stringer   = role(0)
)
