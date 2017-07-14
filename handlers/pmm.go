// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"golang.org/x/net/context"

	"github.com/Percona-Lab/pmm-managed/api"
	"github.com/Percona-Lab/pmm-managed/service"
)

type Server struct {
	Prometheus *service.Prometheus
}

func (s *Server) Version(context.Context, *api.BaseVersionRequest) (*api.BaseVersionResponse, error) {
	return &api.BaseVersionResponse{"pmm-managed v0.0.0-alpha"}, nil
}

func (s *Server) List(context.Context, *api.AlertsListRequest) (*api.AlertsListResponse, error) {
	rules, err := s.Prometheus.ListAlertRules()
	if err != nil {
		return nil, err
	}

	res := &api.AlertsListResponse{
		AlertRules: make([]*api.AlertRule, len(rules)),
	}
	for i, r := range rules {
		res.AlertRules[i] = &api.AlertRule{
			Name:     r.Name,
			Text:     r.Text,
			Disabled: r.Disabled,
		}
	}
	return res, nil
}

// check interfaces
var (
	_ api.AlertsServer = (*Server)(nil)
	_ api.BaseServer   = (*Server)(nil)
)
