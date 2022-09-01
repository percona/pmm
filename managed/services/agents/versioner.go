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

package agents

import (
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"

	"github.com/percona/pmm/api/agentpb"
)

var pmmAgentMinVersionForSoftwareVersions = version.Must(version.NewVersion("2.22"))

// VersionerService provides methods for retrieving versions of different software.
type VersionerService struct {
	r *Registry
}

// NewVersionerService returns new versioner service.
func NewVersionerService(registry *Registry) *VersionerService {
	return &VersionerService{
		r: registry,
	}
}

// Version contains version and error fields.
// Both Version and Error fields could be empty if requested software is not installed.
type Version struct {
	Version string
	Error   string
}

// Software interface.
type Software interface {
	isSoftware()
}

// Mysqld represents mysqld software.
type Mysqld struct{}

func (*Mysqld) isSoftware() {}

// Xtrabackup represents xtrabackup software.
type Xtrabackup struct{}

func (*Xtrabackup) isSoftware() {}

// Xbcloud represents xbcloud software.
type Xbcloud struct{}

func (*Xbcloud) isSoftware() {}

// Qpress represents qpress software.
type Qpress struct{}

func (*Qpress) isSoftware() {}

func convertSoftwares(softwares []Software) ([]*agentpb.GetVersionsRequest_Software, error) {
	softwaresRequest := make([]*agentpb.GetVersionsRequest_Software, 0, len(softwares))
	for _, software := range softwares {
		switch s := software.(type) {
		case *Mysqld:
			softwaresRequest = append(softwaresRequest, &agentpb.GetVersionsRequest_Software{
				Software: &agentpb.GetVersionsRequest_Software_Mysqld{
					Mysqld: &agentpb.GetVersionsRequest_MySQLd{},
				},
			})
		case *Xtrabackup:
			softwaresRequest = append(softwaresRequest, &agentpb.GetVersionsRequest_Software{
				Software: &agentpb.GetVersionsRequest_Software_Xtrabackup{
					Xtrabackup: &agentpb.GetVersionsRequest_Xtrabackup{},
				},
			})
		case *Xbcloud:
			softwaresRequest = append(softwaresRequest, &agentpb.GetVersionsRequest_Software{
				Software: &agentpb.GetVersionsRequest_Software_Xbcloud{
					Xbcloud: &agentpb.GetVersionsRequest_Xbcloud{},
				},
			})
		case *Qpress:
			softwaresRequest = append(softwaresRequest, &agentpb.GetVersionsRequest_Software{
				Software: &agentpb.GetVersionsRequest_Software_Qpress{
					Qpress: &agentpb.GetVersionsRequest_Qpress{},
				},
			})
		default:
			return nil, errors.Errorf("unexpected software type: %T", s)
		}
	}

	return softwaresRequest, nil
}

// GetVersions retrieves software versions.
func (s *VersionerService) GetVersions(pmmAgentID string, softwares []Software) ([]Version, error) {
	if err := PMMAgentSupportedByAgentIdd(s.r.db.Querier, pmmAgentID,
		"versions retrieving", pmmAgentMinVersionForSoftwareVersions); err != nil {
		return nil, err
	}

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	softwaresRequest, err := convertSoftwares(softwares)
	if err != nil {
		return nil, err
	}

	request := &agentpb.GetVersionsRequest{Softwares: softwaresRequest}
	response, err := agent.channel.SendAndWaitResponse(request)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	versionsResponse := response.(*agentpb.GetVersionsResponse).Versions
	if len(versionsResponse) != len(softwaresRequest) {
		return nil, errors.Errorf("response and request slice length mismatch %d != %d",
			len(versionsResponse), len(softwaresRequest))
	}

	versions := make([]Version, 0, len(softwaresRequest))
	for _, v := range versionsResponse {
		versions = append(versions, Version{
			Version: v.Version,
			Error:   v.Error,
		})
	}

	return versions, nil
}
