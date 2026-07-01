// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package adre

import (
	"context"
	"net/http"
)

// GrafanaAuth resolves the current Grafana user and calls Grafana HTTP APIs on behalf of the request.
type GrafanaAuth interface {
	GetAlertmanagerAlerts(ctx context.Context, authHeaders http.Header) ([]byte, error)
	GetCurrentUserLogin(ctx context.Context, authHeaders http.Header) (string, error)
	// IsCurrentUserAdmin gates admin-only deployment-config endpoints (org Admin or Grafana super-admin).
	IsCurrentUserAdmin(ctx context.Context, authHeaders http.Header) (bool, error)
	// CreateServiceAccount mints the Grafana service-account token PMM injects as Holmes's PMM_API_TOKEN.
	CreateServiceAccount(ctx context.Context, nodeName string, reregister bool) (int, string, error)
	// EnsureAlertWebhookContactPoint provisions the auto-investigate webhook contact point + route.
	EnsureAlertWebhookContactPoint(ctx context.Context, webhookURL, secret string) error
}
