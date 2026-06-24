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
	"crypto/subtle"
	"io"
	"net/http"
	"strings"

	"github.com/percona/pmm/managed/models"
)

// maxAlertWebhookBytes bounds the alert webhook payload size.
const maxAlertWebhookBytes = 1 << 20 // 1 MiB

// alertWebhookSem bounds concurrent webhook processing so a burst of deliveries (or retries) can't
// spawn unbounded goroutines. When saturated the delivery is dropped and the reconciliation poll
// picks the alert up instead.
var alertWebhookSem = make(chan struct{}, 8)

// PostAlertWebhook handles POST /v1/adre/alert-webhook. It authenticates Grafana's alert webhook with
// the shared secret (Authorization: Bearer <secret>) and hands the payload to the auto-investigate
// sink. Without a configured sink or secret it is unavailable. Processing is asynchronous so a slow
// Holmes run never blocks Grafana's webhook delivery.
func (h *Handlers) PostAlertWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.alertSink == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "auto-investigate is not available")
		return
	}
	prov, err := models.GetAdreProvisioning(h.db)
	if err != nil || prov.AlertWebhookSecret == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "alert webhook is not configured")
		return
	}
	if !validBearerSecret(r.Header.Get("Authorization"), prov.AlertWebhookSecret) {
		writeJSONError(w, http.StatusUnauthorized, "invalid webhook credential")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxAlertWebhookBytes))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	sink := h.alertSink
	ctx := context.WithoutCancel(r.Context())
	select {
	case alertWebhookSem <- struct{}{}:
		go func() {
			defer func() { <-alertWebhookSem }()
			sink.ProcessWebhook(ctx, body)
		}()
	default:
		h.l.Warn("alert webhook processing saturated; relying on the reconciliation poll")
	}
	w.WriteHeader(http.StatusAccepted)
}

// validBearerSecret reports whether the Authorization header carries the expected bearer secret,
// using a constant-time comparison.
func validBearerSecret(header, secret string) bool {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
}
