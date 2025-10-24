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

package agents

import (
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	"github.com/percona/pmm/managed/models"
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
	// Name returns string name, one of defined in the models package.
	Name() models.SoftwareName
	// GetVersionRequest returns prepared struct for gRPC request.
	GetVersionRequest() *agentv1.GetVersionsRequest_Software
}

// Mysqld represents mysqld software.
type Mysqld struct{}

func (*Mysqld) isSoftware() {}

// Name returns the software name for Mysqld.
func (*Mysqld) Name() models.SoftwareName { return models.MysqldSoftwareName }

// GetVersionRequest constructs a request for MySQL software.
func (*Mysqld) GetVersionRequest() *agentv1.GetVersionsRequest_Software {
	return &agentv1.GetVersionsRequest_Software{
		Software: &agentv1.GetVersionsRequest_Software_Mysqld{},
	}
}

// Xtrabackup represents xtrabackup software.
type Xtrabackup struct{}

func (*Xtrabackup) isSoftware() {}

// Name returns the software name for Xtrabackup.
func (*Xtrabackup) Name() models.SoftwareName { return models.XtrabackupSoftwareName }

// GetVersionRequest constructs a request for XtraBackup software.
func (*Xtrabackup) GetVersionRequest() *agentv1.GetVersionsRequest_Software {
	return &agentv1.GetVersionsRequest_Software{
		Software: &agentv1.GetVersionsRequest_Software_Xtrabackup{},
	}
}

// Xbcloud represents xbcloud software.
type Xbcloud struct{}

func (*Xbcloud) isSoftware() {}

// Name returns the software name for xbcloud.
func (*Xbcloud) Name() models.SoftwareName { return models.XbcloudSoftwareName }

// GetVersionRequest constructs a request for Xbcloud software.
func (*Xbcloud) GetVersionRequest() *agentv1.GetVersionsRequest_Software {
	return &agentv1.GetVersionsRequest_Software{
		Software: &agentv1.GetVersionsRequest_Software_Xbcloud{},
	}
}

// Qpress represents qpress software.
type Qpress struct{}

func (*Qpress) isSoftware() {}

// Name returns the software name for Qpress.
func (*Qpress) Name() models.SoftwareName { return models.QpressSoftwareName }

// GetVersionRequest constructs a request for Qpress software.
func (*Qpress) GetVersionRequest() *agentv1.GetVersionsRequest_Software {
	return &agentv1.GetVersionsRequest_Software{
		Software: &agentv1.GetVersionsRequest_Software_Qpress{},
	}
}

// MongoDB represents mongod software.
type MongoDB struct{}

func (*MongoDB) isSoftware() {}

// Name returns the software name for MongoDB.
func (*MongoDB) Name() models.SoftwareName { return models.MongoDBSoftwareName }

// GetVersionRequest constructs a request for MongoDB software.
func (*MongoDB) GetVersionRequest() *agentv1.GetVersionsRequest_Software {
	return &agentv1.GetVersionsRequest_Software{
		Software: &agentv1.GetVersionsRequest_Software_Mongod{},
	}
}

// PBM represents pbm software.
type PBM struct{}

func (*PBM) isSoftware() {}

// Name returns the software name for PBM.
func (*PBM) Name() models.SoftwareName { return models.PBMSoftwareName }

// GetVersionRequest constructs a request for PBM software.
func (*PBM) GetVersionRequest() *agentv1.GetVersionsRequest_Software {
	return &agentv1.GetVersionsRequest_Software{
		Software: &agentv1.GetVersionsRequest_Software_Pbm{},
	}
}

// getMysqlSoftwareList returns list of software required for MySQL backups.
func getMysqlSoftwareList() []Software {
	return []Software{&Mysqld{}, &Xtrabackup{}, &Xbcloud{}}
}

// getMongodbSoftwareList returns list of software required for MongoDB backups.
func getMongodbSoftwareList() []Software {
	return []Software{&MongoDB{}, &PBM{}}
}

// GetRequiredBackupSoftwareList maps service type into list of software required for backups. Returns empty list if no software specified for the type.
func GetRequiredBackupSoftwareList(serviceType models.ServiceType) []Software {
	switch serviceType {
	case models.MySQLServiceType:
		return getMysqlSoftwareList()
	case models.MongoDBServiceType:
		return getMongodbSoftwareList()
	default:
		return nil
	}
}

// GetVersions retrieves software versions.
func (s *VersionerService) GetVersions(pmmAgentID string, softwareList []Software) ([]Version, error) {
	if err := models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
		"versions retrieving", pmmAgentMinVersionForSoftwareVersions); err != nil {
		return nil, err
	}

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	softwareRequest := make([]*agentv1.GetVersionsRequest_Software, 0, len(softwareList))
	for _, software := range softwareList {
		softwareRequest = append(softwareRequest, software.GetVersionRequest())
	}

	request := &agentv1.GetVersionsRequest{Softwares: softwareRequest}
	response, err := agent.channel.SendAndWaitResponse(request)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	versionsResponse := response.(*agentv1.GetVersionsResponse).Versions //nolint:forcetypeassert
	if len(versionsResponse) != len(softwareRequest) {
		return nil, errors.Errorf("response and request slice length mismatch %d != %d",
			len(versionsResponse), len(softwareRequest))
	}

	versions := make([]Version, 0, len(softwareRequest))
	for _, v := range versionsResponse {
		versions = append(versions, Version{
			Version: v.Version,
			Error:   v.Error,
		})
	}

	return versions, nil
}
