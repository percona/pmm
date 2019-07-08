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

package grafana

import (
	"net/http"
	"net/http/httputil"
	"path"

	"github.com/sirupsen/logrus"
)

// AuthServer authenticates incoming requests via Grafana API.
type AuthServer struct {
	l *logrus.Entry
}

// NewAuthServer creates new AuthServer.
func NewAuthServer() *AuthServer {
	return &AuthServer{
		l: logrus.WithField("component", "grafana/auth"),
	}
}

// ServeHTTP serves internal location /auth/<role>.
func (s *AuthServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b, err := httputil.DumpRequest(req, true)
	if err != nil {
		s.l.Warnf("%v", err)
	}
	s.l.Debugf("Request:\n%s", b)

	_, role := path.Split(req.URL.Path)
	switch role {
	case "admin", "editor", "viewer":
		s.l.Debugf("Role: %s", role)
		rw.WriteHeader(200)
	default:
		s.l.Errorf("Unexpected role %q.", role)
		rw.WriteHeader(500)
	}
}
