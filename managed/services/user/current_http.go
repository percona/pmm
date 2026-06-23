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

package user

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/services/grafana"
)

// currentUserClient provides methods for current user endpoints.
type currentUserClient interface {
	GetCurrentUser(ctx context.Context, authHeaders http.Header) (grafana.CurrentUser, error)
	GetCurrentUserOrgs(ctx context.Context, authHeaders http.Header) ([]grafana.CurrentUserOrg, error)
}

type currentHTTPHandler struct {
	l *logrus.Entry
	c currentUserClient
}

// NewCurrentHTTPHandler creates handler for current user JSON endpoints.
func NewCurrentHTTPHandler(c currentUserClient) http.Handler {
	return &currentHTTPHandler{
		c: c,
		l: logrus.WithField("component", "user/current-http"),
	}
}

func (h *currentHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	authHeaders := make(http.Header)
	for _, k := range []string{"Authorization", "Cookie"} {
		if v := req.Header.Get(k); v != "" {
			authHeaders.Set(k, v)
		}
	}

	rw.Header().Set("Content-Type", "application/json")

	if req.Method != http.MethodGet {
		rw.Header().Set("Allow", http.MethodGet)
		rw.WriteHeader(http.StatusMethodNotAllowed)
		err := json.NewEncoder(rw).Encode(map[string]string{"message": "Method Not Allowed"})
		if err != nil {
			h.l.Errorf("encode method-not-allowed body: %v", err)
		}
		return
	}

	switch req.URL.Path {
	case "/v1/users/current":
		user, err := h.c.GetCurrentUser(req.Context(), authHeaders)
		if err != nil {
			h.l.Errorf("failed to get current user: %v", err)
			status, body := grafana.CurrentUserHTTPResponse(err)
			rw.WriteHeader(status)
			encErr := json.NewEncoder(rw).Encode(body)
			if encErr != nil {
				h.l.Errorf("encode error body: %v", encErr)
			}
			return
		}
		err = json.NewEncoder(rw).Encode(user)
		if err != nil {
			h.l.Errorf("encode current user: %v", err)
		}
	case "/v1/users/current/orgs":
		orgs, err := h.c.GetCurrentUserOrgs(req.Context(), authHeaders)
		if err != nil {
			h.l.Errorf("failed to get current user orgs: %v", err)
			status, body := grafana.CurrentUserHTTPResponse(err)
			rw.WriteHeader(status)
			encErr := json.NewEncoder(rw).Encode(body)
			if encErr != nil {
				h.l.Errorf("encode error body: %v", encErr)
			}
			return
		}
		err = json.NewEncoder(rw).Encode(orgs)
		if err != nil {
			h.l.Errorf("encode current user orgs: %v", err)
		}
	default:
		http.NotFound(rw, req)
	}
}
