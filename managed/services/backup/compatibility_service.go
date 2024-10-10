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

package backup

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/agents"
)

// pmmAgentMinVersionForMongoBackupSoftwareCheck minimum agent version for getting required backup software versions.
var pmmAgentMinVersionForMongoBackupSoftwareCheck = version.Must(version.NewVersion("2.35.0-0"))

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

// checkCompatibility contains compatibility checking logic.
func (s *CompatibilityService) checkCompatibility(serviceModel *models.Service, agentModel *models.Agent) (string, error) {
	softwareList := agents.GetRequiredBackupSoftwareList(serviceModel.ServiceType)
	if len(softwareList) == 0 {
		s.l.Infof("Required backup software is not specified for %s service type.", serviceModel.ServiceType)
		return "", nil
	}

	svs, err := s.v.GetVersions(agentModel.AgentID, softwareList)
	if err != nil {
		return "", err
	}
	if len(svs) != len(softwareList) {
		s.l.Errorf("Response slice len %d != request len %d.", len(svs), len(softwareList))
		return "", ErrComparisonImpossible
	}

	svm := make(map[models.SoftwareName]string, len(softwareList))
	for i, software := range softwareList {
		name := software.Name()
		if svs[i].Error != "" {
			return "", errors.Wrapf(ErrComparisonImpossible, "failed to get software %s version: %s", name, svs[i].Error)
		}

		svm[name] = svs[i].Version
	}

	var binaryName models.SoftwareName
	switch serviceModel.ServiceType {
	case models.MySQLServiceType:
		binaryName = models.MysqldSoftwareName
		err = mySQLBackupSoftwareInstalledAndCompatible(svm)
	case models.MongoDBServiceType:
		binaryName = models.MongoDBSoftwareName
		err = mongoDBBackupSoftwareInstalledAndCompatible(svm)
	default:
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return svm[binaryName], nil
}

// findCompatibleServiceIDs looks for services compatible to artifact in given array of services' software versions.
func (s *CompatibilityService) findCompatibleServiceIDs(artifactModel *models.Artifact, svs []*models.ServiceSoftwareVersions) []string {
	compatibleServiceIDs := make([]string, 0, len(svs))
	for _, sv := range svs {
		svm := softwareVersionsToMap(sv.SoftwareVersions)
		var (
			serviceDBVersion string
			err              error
		)

		switch artifactModel.Vendor {
		case "mysql":
			serviceDBVersion = svm[models.MysqldSoftwareName]
			err = mySQLBackupSoftwareInstalledAndCompatible(svm)

		case "mongodb":
			serviceDBVersion = svm[models.MongoDBSoftwareName]
			err = mongoDBBackupSoftwareInstalledAndCompatible(svm)

		default:
			return nil
		}

		if err != nil {
			s.l.WithError(err).Debugf("skip incompatible service id %q", sv.ServiceID)
			continue
		}

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
// Returns version of the installed database.
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

	if serviceModel.ServiceType == models.MongoDBServiceType {
		if err := models.PMMAgentSupported(s.db.Querier, agentModel.AgentID, "get mongodb backup software versions",
			pmmAgentMinVersionForMongoBackupSoftwareCheck); err != nil {
			var agentNotSupportedError models.AgentNotSupportedError
			if errors.As(err, &agentNotSupportedError) {
				s.l.Warnf("Got versioner error message: %s.", err.Error())
				return "", nil
			}
			return "", err
		}
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

		onlySameService := isOnlySameService(artifactModel.DBVersion)

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

// CheckArtifactCompatibility check compatibility between artifact and target database.
func (s *CompatibilityService) CheckArtifactCompatibility(artifactID, targetDBVersion string) error {
	artifactModel, err := models.FindArtifactByID(s.db.Querier, artifactID)
	if err != nil {
		return err
	}

	serviceModel, err := models.FindServiceByID(s.db.Querier, artifactModel.ServiceID)
	if err != nil {
		return err
	}

	return s.artifactCompatibility(artifactModel, serviceModel, targetDBVersion)
}

// artifactCompatibility contains logic for CheckArtifactCompatibility.
func (s *CompatibilityService) artifactCompatibility(artifactModel *models.Artifact, serviceModel *models.Service, targetDBVersion string) error {
	var err error
	if artifactModel.DBVersion != "" && artifactModel.DBVersion != targetDBVersion {
		switch serviceModel.ServiceType {
		case models.MySQLServiceType:
			err = ErrIncompatibleTargetMySQL
		case models.MongoDBServiceType:
			err = ErrIncompatibleTargetMongoDB
		default:
			return nil
		}
		return errors.Wrapf(err, "backup artifact db version %q does not match the target db version %q", artifactModel.DBVersion, targetDBVersion)
	}

	return nil
}
