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

package backup

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
)

// CompatibilityService is responsible for checking software and artifacts compatibility during backup and restore.
type CompatibilityService struct {
	db *reform.DB
	v  versioner
	l  *logrus.Entry
}

// NewCompatibilityService creates new backups logic service.
func NewCompatibilityService(db *reform.DB, v versioner) *CompatibilityService {
	return &CompatibilityService{
		l:  logrus.WithField("component", "management/backup/compatibility"),
		db: db,
		v:  v,
	}
}

// checkSoftwareCompatibilityForService contains compatibility checking logic.
func (s *CompatibilityService) checkCompatibility(serviceModel *models.Service, agentModel *models.Agent) (string, error) {
	// Only MySQL compatibility checking implemented for now.
	if serviceModel.ServiceType != models.MySQLServiceType {
		return "", nil
	}

	softwares := []agents.Software{&agents.Mysqld{}, &agents.Xtrabackup{}, &agents.Xbcloud{}, &agents.Qpress{}}
	svs, err := s.v.GetVersions(agentModel.AgentID, softwares)
	if err != nil {
		return "", err
	}
	if len(svs) != len(softwares) {
		return "", errors.Wrapf(ErrComparisonImpossible, "response slice len %d != request len %d", len(svs), len(softwares))
	}

	svm := make(map[models.SoftwareName]string, len(softwares))
	for i, software := range softwares {
		name, err := convertSoftwareName(software)
		if err != nil {
			return "", err
		}
		if svs[i].Error != "" {
			return "", errors.Wrapf(ErrComparisonImpossible, "failed to get software %s version: %s", name, svs[i].Error)
		}

		svm[name] = svs[i].Version
	}

	if err := mySQLSoftwaresInstalledAndCompatible(svm); err != nil {
		return "", err
	}

	return svm[models.MysqldSoftwareName], nil
}

// findCompatibleServiceIDs looks for services compatible to artifact in given array of services' software versions.
func (s *CompatibilityService) findCompatibleServiceIDs(artifactModel *models.Artifact, svs []*models.ServiceSoftwareVersions) []string {
	compatibleServiceIDs := make([]string, 0, len(svs))
	for _, sv := range svs {
		svm := softwareVersionsToMap(sv.SoftwareVersions)

		if err := mySQLSoftwaresInstalledAndCompatible(svm); err != nil {
			s.l.WithError(err).Debugf("skip incompatible service id %q", sv.ServiceID)
			continue
		}

		serviceDBVersion := svm[models.MysqldSoftwareName]
		if artifactModel.DBVersion != serviceDBVersion {
			s.l.Debugf("skip incompatible service id %q: artifact version %q != db version %q\"", sv.ServiceID,
				artifactModel.DBVersion, serviceDBVersion)
			continue
		}

		compatibleServiceIDs = append(compatibleServiceIDs, sv.ServiceID)
	}
	return compatibleServiceIDs
}

// CheckSoftwareCompatibilityForService checks if all the necessary backup tools are installed,
// and they are compatible with the db version, currently only supports backup tools for MySQL
// Returns db version.
func (s *CompatibilityService) CheckSoftwareCompatibilityForService(ctx context.Context, serviceID string) (string, error) {
	var serviceModel *models.Service
	var agentModel *models.Agent
	errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		serviceModel, err = models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}
		if len(pmmAgents) == 0 {
			return errors.Errorf("pmmAgent not found for service %q", serviceID)
		}
		agentModel = pmmAgents[0]
		return nil
	})
	if errTx != nil {
		return "", errTx
	}

	return s.checkCompatibility(serviceModel, agentModel)
}

// FindArtifactCompatibleServices searches compatible services which can be used to restore an artifact to.
func (s *CompatibilityService) FindArtifactCompatibleServices(
	ctx context.Context,
	artifactID string,
) ([]*models.Service, error) {
	var compatibleServices []*models.Service
	if err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		artifactModel, err := models.FindArtifactByID(tx.Querier, artifactID)
		if err != nil {
			return err
		}

		serviceType, err := vendorToServiceType(artifactModel.Vendor)
		if err != nil {
			return err
		}

		onlySameService := isOnlySameService(artifactModel.DBVersion, serviceType)

		if onlySameService {
			service, err := models.FindServiceByID(tx.Querier, artifactModel.ServiceID)
			if err != nil {
				s.l.WithError(err).Warnf("restore is not possible to the service id %q", artifactModel.ServiceID)
				return nil
			}
			compatibleServices = []*models.Service{service}
			return nil
		}

		filter := models.FindServicesSoftwareVersionsFilter{ServiceType: &serviceType}
		svs, err := models.FindServicesSoftwareVersions(tx.Querier, filter, models.SoftwareVersionsOrderByServiceID)
		if err != nil {
			return err
		}

		compatibleServiceIDs := s.findCompatibleServiceIDs(artifactModel, svs)
		if len(compatibleServiceIDs) == 0 {
			return nil
		}

		servicesMap, err := models.FindServicesByIDs(tx.Querier, compatibleServiceIDs)
		if err != nil {
			return err
		}

		compatibleServices = make([]*models.Service, len(compatibleServiceIDs))
		for i, id := range compatibleServiceIDs {
			compatibleServices[i] = servicesMap[id]
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return compatibleServices, nil
}
