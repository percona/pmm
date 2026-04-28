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

	switch req.URL.Path {
	case "/v1/users/current":
		user, err := h.c.GetCurrentUser(req.Context(), authHeaders)
		if err != nil {
			h.l.Errorf("failed to get current user: %v", err)
			status, body := grafana.CurrentUserHTTPResponse(err)
			rw.WriteHeader(status)
			_ = json.NewEncoder(rw).Encode(body)
			return
		}
		_ = json.NewEncoder(rw).Encode(user)
	case "/v1/users/current/orgs":
		orgs, err := h.c.GetCurrentUserOrgs(req.Context(), authHeaders)
		if err != nil {
			h.l.Errorf("failed to get current user orgs: %v", err)
			status, body := grafana.CurrentUserHTTPResponse(err)
			rw.WriteHeader(status)
			_ = json.NewEncoder(rw).Encode(body)
			return
		}
		_ = json.NewEncoder(rw).Encode(orgs)
	default:
		http.NotFound(rw, req)
	}
}
