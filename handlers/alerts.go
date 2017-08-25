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

	"github.com/percona/pmm-managed/api"
	"github.com/percona/pmm-managed/services/prometheus"
)

type AlertsServer struct {
	Prometheus *prometheus.Service
}

func convertAlertRule(r *prometheus.AlertRule) *api.AlertRule {
	return &api.AlertRule{
		Name:     r.Name,
		Text:     r.Text,
		Disabled: r.Disabled,
	}
}

func (s *AlertsServer) List(ctx context.Context, req *api.AlertsListRequest) (*api.AlertsListResponse, error) {
	rules, err := s.Prometheus.ListAlertRules(ctx)
	if err != nil {
		return nil, err
	}

	res := &api.AlertsListResponse{
		AlertRules: make([]*api.AlertRule, len(rules)),
	}
	for i, r := range rules {
		res.AlertRules[i] = convertAlertRule(&r)
	}
	return res, nil
}

func (s *AlertsServer) Get(ctx context.Context, req *api.AlertsGetRequest) (*api.AlertsGetResponse, error) {
	rule, err := s.Prometheus.GetAlert(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &api.AlertsGetResponse{
		AlertRule: convertAlertRule(rule),
	}, nil
}

// check interface
var _ api.AlertsServer = (*AlertsServer)(nil)
