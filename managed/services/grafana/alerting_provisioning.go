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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// alertAnnotationsReceiver is the name of both the webhook contact point and the
// notification-policy route used to deliver alert notifications to PMM's annotations webhook.
const alertAnnotationsReceiver = "pmm-alert-annotations"

// embeddedContactPoint is a subset of Grafana's provisioning contact point object.
type embeddedContactPoint struct {
	UID                   string         `json:"uid,omitempty"`
	Name                  string         `json:"name"`
	Type                  string         `json:"type"`
	Settings              map[string]any `json:"settings"`
	DisableResolveMessage bool           `json:"disableResolveMessage"`
}

// EnsureAlertAnnotationsContactPoint idempotently provisions a webhook contact point pointing at
// webhookURL and a continue=true notification-policy route that delivers all alerts to it, so PMM
// can turn alert notifications into Grafana annotations. It uses the Grafana provisioning API with
// the X-Disable-Provenance header so the resources remain editable in the Grafana UI, and the
// continue=true route preserves delivery to any other (user-defined) receivers.
func (c *Client) EnsureAlertAnnotationsContactPoint(ctx context.Context, webhookURL string) error {
	headers := make(http.Header)
	headers.Set("Authorization", adminAuthorization())
	headers.Set("X-Disable-Provenance", "true")

	err := c.ensureAnnotationsContactPoint(ctx, headers, webhookURL)
	if err != nil {
		return err
	}
	return c.ensureAnnotationsPolicyRoute(ctx, headers)
}

func (c *Client) ensureAnnotationsContactPoint(ctx context.Context, headers http.Header, webhookURL string) error {
	var existing []embeddedContactPoint
	err := c.do(ctx, http.MethodGet, "/api/v1/provisioning/contact-points", "", headers, nil, &existing)
	if err != nil {
		return fmt.Errorf("failed to list contact points: %w", err)
	}
	for _, cp := range existing {
		if cp.Name == alertAnnotationsReceiver {
			return nil
		}
	}

	cp := embeddedContactPoint{
		Name: alertAnnotationsReceiver,
		Type: "webhook",
		Settings: map[string]any{
			"url":        webhookURL,
			"httpMethod": http.MethodPost,
		},
	}
	b, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("failed to marshal contact point: %w", err)
	}
	err = c.do(ctx, http.MethodPost, "/api/v1/provisioning/contact-points", "", headers, b, nil)
	if err != nil {
		return fmt.Errorf("failed to create contact point: %w", err)
	}
	return nil
}

// ensureAnnotationsPolicyRoute adds a continue=true child route for our receiver to the root
// notification policy. The tree is read into a generic map and written back so unrelated
// (user-configured) fields are preserved.
func (c *Client) ensureAnnotationsPolicyRoute(ctx context.Context, headers http.Header) error {
	var tree map[string]any
	err := c.do(ctx, http.MethodGet, "/api/v1/provisioning/policies", "", headers, nil, &tree)
	if err != nil {
		return fmt.Errorf("failed to get notification policy tree: %w", err)
	}

	routes, _ := tree["routes"].([]any)
	for _, r := range routes {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if recv, _ := m["receiver"].(string); recv == alertAnnotationsReceiver {
			return nil
		}
	}

	routes = append(routes, map[string]any{
		"receiver":        alertAnnotationsReceiver,
		"continue":        true,
		"object_matchers": []any{},
	})
	tree["routes"] = routes

	b, err := json.Marshal(tree)
	if err != nil {
		return fmt.Errorf("failed to marshal notification policy tree: %w", err)
	}
	err = c.do(ctx, http.MethodPut, "/api/v1/provisioning/policies", "", headers, b, nil)
	if err != nil {
		return fmt.Errorf("failed to update notification policy tree: %w", err)
	}
	return nil
}
