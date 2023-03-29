// Copyright (C) 2022 Percona LLC
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

// Package team provides API for team related tasks
package team

import (
	"context"

	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/teampb"
	"github.com/percona/pmm/managed/models"
)

// Service is responsible for team related APIs.
type Service struct {
	db *reform.DB
	l  *logrus.Entry

	teampb.UnimplementedTeamServer
}

// NewTeamService return a team service.
func NewTeamService(db *reform.DB) *Service {
	l := logrus.WithField("component", "team")

	s := Service{
		db: db,
		l:  l,
	}
	return &s
}

// ListTeams lists all teams and their details.
func (s *Service) ListTeams(_ context.Context, _ *teampb.ListTeamsRequest) (*teampb.ListTeamsResponse, error) {
	teamRoles, err := models.ListTeams(s.db.Querier)
	if err != nil {
		return nil, err
	}

	resp := &teampb.ListTeamsResponse{
		Teams: make([]*teampb.ListTeamsResponse_TeamDetail, 0, len(teamRoles)),
	}
	for teamID, roleIDs := range teamRoles {
		resp.Teams = append(resp.Teams, &teampb.ListTeamsResponse_TeamDetail{
			TeamId:  uint32(teamID),
			RoleIds: roleIDs,
		})
	}

	return resp, nil
}
