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

// Package backup provides backup functionality.
package backup

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
	"github.com/percona/pmm-managed/services/agents"
)

// Service represents core logic for db backup.
type Service struct {
	db          *reform.DB
	jobsService jobsService
	l           *logrus.Entry
	v           versioner
}

// NewService creates new backups logic service.
func NewService(db *reform.DB, jobsService jobsService, v versioner) *Service {
	return &Service{
		l:           logrus.WithField("component", "management/backup/backup"),
		db:          db,
		jobsService: jobsService,
		v:           v,
	}
}

type pmmAgentResult struct {
	id          string
	serviceType models.ServiceType
}

func (s *Service) findPMMAgentForService(ctx context.Context, serviceID string) (*pmmAgentResult, error) {
	var pmmAgentID string
	var serviceType models.ServiceType
	if err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		service, err := models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		serviceType = service.ServiceType

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}
		if len(pmmAgents) == 0 {
			return errors.Errorf("pmmAgent not found for service %q", serviceID)
		}

		pmmAgentID = pmmAgents[0].AgentID

		return nil
	}); err != nil {
		return nil, err
	}

	return &pmmAgentResult{
		id:          pmmAgentID,
		serviceType: serviceType,
	}, nil
}

// PerformBackupParams are params for performing backup.
type PerformBackupParams struct {
	ServiceID     string
	LocationID    string
	Name          string
	ScheduleID    string
	Retries       uint32
	RetryInterval time.Duration
}

// PerformBackup starts on-demand backup.
func (s *Service) PerformBackup(ctx context.Context, params PerformBackupParams) (string, error) {
	var err error
	dbVersion, err := s.checkSoftwareCompatibilityForService(ctx, params.ServiceID)
	if err != nil {
		return "", err
	}

	var artifact *models.Artifact
	var location *models.BackupLocation
	var svc *models.Service
	var job *models.Job
	var config *models.DBConfig

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		svc, err = models.FindServiceByID(tx.Querier, params.ServiceID)
		if err != nil {
			return err
		}

		location, err = models.FindBackupLocationByID(tx.Querier, params.LocationID)
		if err != nil {
			return err
		}

		var dataModel models.DataModel
		var jobType models.JobType
		switch svc.ServiceType {
		case models.MySQLServiceType:
			dataModel = models.PhysicalDataModel
			jobType = models.MySQLBackupJob
		case models.MongoDBServiceType:
			dataModel = models.LogicalDataModel
			jobType = models.MongoDBBackupJob
		case models.PostgreSQLServiceType,
			models.ProxySQLServiceType,
			models.HAProxyServiceType,
			models.ExternalServiceType:
			return status.Errorf(codes.Unimplemented, "unimplemented service: %s", svc.ServiceType)
		default:
			return status.Errorf(codes.Unknown, "unknown service: %s", svc.ServiceType)
		}

		artifact, err = models.CreateArtifact(tx.Querier, models.CreateArtifactParams{
			Name:       params.Name,
			Vendor:     string(svc.ServiceType),
			DBVersion:  dbVersion,
			LocationID: location.ID,
			ServiceID:  svc.ServiceID,
			DataModel:  dataModel,
			Status:     models.PendingBackupStatus,
			ScheduleID: params.ScheduleID,
		})
		if err != nil {
			return err
		}

		job, config, err = s.prepareBackupJob(tx.Querier, svc, artifact.ID, jobType, params.Retries, params.RetryInterval)
		if err != nil {
			return err
		}
		return nil
	})
	if errTX != nil {
		return "", errTX
	}

	locationConfig := &models.BackupLocationConfig{
		PMMServerConfig: location.PMMServerConfig,
		PMMClientConfig: location.PMMClientConfig,
		S3Config:        location.S3Config,
	}

	switch svc.ServiceType {
	case models.MySQLServiceType:
		err = s.jobsService.StartMySQLBackupJob(job.ID, job.PMMAgentID, 0, params.Name, config, locationConfig)
	case models.MongoDBServiceType:
		err = s.jobsService.StartMongoDBBackupJob(job.ID, job.PMMAgentID, 0, params.Name, config, locationConfig)
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		return "", status.Errorf(codes.Unimplemented, "unimplemented service: %s", svc.ServiceType)
	default:
		return "", status.Errorf(codes.Unknown, "unknown service: %s", svc.ServiceType)
	}
	if err != nil {
		return "", err
	}

	return artifact.ID, nil
}

type prepareRestoreJobParams struct {
	AgentID      string
	ArtifactName string
	DBVersion    string
	Location     *models.BackupLocation
	ServiceType  models.ServiceType
	DBConfig     *models.DBConfig
}

// RestoreBackup starts restore backup job.
func (s *Service) RestoreBackup(ctx context.Context, serviceID, artifactID string) (string, error) {
	dbVersion, err := s.checkSoftwareCompatibilityForService(ctx, serviceID)
	if err != nil {
		return "", err
	}

	var params *prepareRestoreJobParams
	var jobID, restoreID string
	if err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		params, err = s.prepareRestoreJob(tx.Querier, serviceID, artifactID)
		if err != nil {
			return err
		}

		if params.ServiceType == models.MySQLServiceType && params.DBVersion != "" {
			if params.DBVersion != dbVersion {
				return errors.Errorf("incompatible service: artifact db version %q != db version %q",
					params.DBVersion, dbVersion)
			}
		}

		restore, err := models.CreateRestoreHistoryItem(tx.Querier, models.CreateRestoreHistoryItemParams{
			ArtifactID: artifactID,
			ServiceID:  serviceID,
			Status:     models.InProgressRestoreStatus,
		})
		if err != nil {
			return err
		}

		restoreID = restore.ID

		service, err := models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		var jobType models.JobType
		var jobData *models.JobData
		switch service.ServiceType {
		case models.MySQLServiceType:
			jobType = models.MySQLRestoreBackupJob
			jobData = &models.JobData{
				MySQLRestoreBackup: &models.MySQLRestoreBackupJobData{
					RestoreID: restoreID,
				}}
		case models.MongoDBServiceType:
			jobType = models.MongoDBRestoreBackupJob
			jobData = &models.JobData{
				MongoDBRestoreBackup: &models.MongoDBRestoreBackupJobData{
					RestoreID: restoreID,
				}}
		case models.PostgreSQLServiceType,
			models.ProxySQLServiceType,
			models.HAProxyServiceType,
			models.ExternalServiceType:
			return errors.Errorf("backup restore unimplemented for service type: %s", service.ServiceType)
		default:
			return errors.Errorf("unsupported service type: %s", service.ServiceType)
		}

		job, err := models.CreateJob(tx.Querier, models.CreateJobParams{
			PMMAgentID: params.AgentID,
			Type:       jobType,
			Data:       jobData,
		})
		if err != nil {
			return err
		}

		jobID = job.ID

		return err
	}); err != nil {
		return "", err
	}

	if err := s.startRestoreJob(jobID, serviceID, params); err != nil {
		return "", err
	}

	return restoreID, nil
}

// FindArtifactCompatibleServices finds compatible services which can be used to restoring an artifact to.
func (s *Service) FindArtifactCompatibleServices(
	ctx context.Context,
	artifactID string,
) ([]*models.Service, error) {
	var compatibleServices []*models.Service
	if err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		artifact, err := models.FindArtifactByID(tx.Querier, artifactID)
		switch {
		case err == nil:
		case errors.Is(err, models.ErrNotFound):
			return status.Errorf(codes.NotFound, "Artifact with ID %q not found.", artifactID)
		default:
			return err
		}

		serviceType, err := vendorToServiceType(artifact.Vendor)
		if err != nil {
			return err
		}

		compatibleServices, err = s.findArtifactCompatibleServices(
			tx.Querier,
			artifact.ServiceID,
			serviceType,
			artifact.DBVersion,
		)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return compatibleServices, nil
}

func (s *Service) prepareRestoreJob(
	q *reform.Querier,
	serviceID string,
	artifactID string,
) (*prepareRestoreJobParams, error) {
	service, err := models.FindServiceByID(q, serviceID)
	if err != nil {
		return nil, err
	}

	artifact, err := models.FindArtifactByID(q, artifactID)
	if err != nil {
		return nil, err
	}
	if artifact.Status != models.SuccessBackupStatus {
		return nil, errors.Errorf("artifact %q status is not successful, status: %q", artifactID, artifact.Status)
	}

	location, err := models.FindBackupLocationByID(q, artifact.LocationID)
	if err != nil {
		return nil, err
	}

	dbConfig, err := models.FindDBConfigForService(q, service.ServiceID)
	if err != nil {
		return nil, err
	}

	pmmAgents, err := models.FindPMMAgentsForService(q, serviceID)
	if err != nil {
		return nil, err
	}
	if len(pmmAgents) == 0 {
		return nil, errors.Errorf("cannot find pmm agent for service %s", serviceID)
	}

	return &prepareRestoreJobParams{
		AgentID:      pmmAgents[0].AgentID,
		ArtifactName: artifact.Name,
		DBVersion:    artifact.DBVersion,
		Location:     location,
		ServiceType:  service.ServiceType,
		DBConfig:     dbConfig,
	}, nil
}

func (s *Service) startRestoreJob(jobID, serviceID string, params *prepareRestoreJobParams) error {
	locationConfig := &models.BackupLocationConfig{
		PMMServerConfig: params.Location.PMMServerConfig,
		PMMClientConfig: params.Location.PMMClientConfig,
		S3Config:        params.Location.S3Config,
	}

	switch params.ServiceType {
	case models.MySQLServiceType:
		if err := s.jobsService.StartMySQLRestoreBackupJob(
			jobID,
			params.AgentID,
			serviceID,
			0,
			params.ArtifactName,
			locationConfig,
		); err != nil {
			return err
		}
	case models.MongoDBServiceType:
		if err := s.jobsService.StartMongoDBRestoreBackupJob(
			jobID,
			params.AgentID,
			0,
			params.ArtifactName,
			params.DBConfig,
			locationConfig,
		); err != nil {
			return err
		}
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		return status.Errorf(codes.Unimplemented, "unimplemented service: %s", params.ServiceType)
	default:
		return status.Errorf(codes.Unknown, "unknown service: %s", params.ServiceType)
	}

	return nil
}

func (s *Service) prepareBackupJob(
	q *reform.Querier,
	service *models.Service,
	artifactID string,
	jobType models.JobType,
	retries uint32,
	retryInterval time.Duration,
) (*models.Job, *models.DBConfig, error) {
	dbConfig, err := models.FindDBConfigForService(q, service.ServiceID)
	if err != nil {
		return nil, nil, err
	}

	pmmAgents, err := models.FindPMMAgentsForService(q, service.ServiceID)
	if err != nil {
		return nil, nil, err
	}

	if len(pmmAgents) == 0 {
		return nil, nil, errors.Errorf("pmmAgent not found for service")
	}

	var jobData *models.JobData
	switch jobType {
	case models.MySQLBackupJob:
		jobData = &models.JobData{
			MySQLBackup: &models.MySQLBackupJobData{
				ServiceID:  service.ServiceID,
				ArtifactID: artifactID,
			},
		}
	case models.MongoDBBackupJob:
		jobData = &models.JobData{
			MongoDBBackup: &models.MongoDBBackupJobData{
				ServiceID:  service.ServiceID,
				ArtifactID: artifactID,
			},
		}
	case models.MySQLRestoreBackupJob,
		models.MongoDBRestoreBackupJob:
		return nil, nil, errors.Errorf("%s is not a backup job type", jobType)
	default:
		return nil, nil, errors.Errorf("unsupported backup job type: %s", jobType)
	}

	res, err := models.CreateJob(q, models.CreateJobParams{
		PMMAgentID: pmmAgents[0].AgentID,
		Type:       jobType,
		Data:       jobData,
		Retries:    retries,
		Interval:   retryInterval,
	})
	if err != nil {
		return nil, nil, err
	}

	return res, dbConfig, nil
}

func vendorToServiceType(vendor string) (models.ServiceType, error) {
	serviceType := models.ServiceType(vendor)
	switch serviceType {
	case models.MySQLServiceType,
		models.MongoDBServiceType:
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		return "", status.Errorf(codes.Unimplemented, "unimplemented service: %s", serviceType)
	default:
		return "", status.Errorf(codes.Internal, "unknown service: %s", serviceType)
	}

	return serviceType, nil
}

func (s *Service) findArtifactCompatibleServices(
	q *reform.Querier,
	serviceID string,
	serviceType models.ServiceType,
	artifactDBVersion string,
) ([]*models.Service, error) {
	// allow restore to the same service if db version is unknown or service type is MongoDB.
	if artifactDBVersion == "" || serviceType == models.MongoDBServiceType {
		service, err := models.FindServiceByID(q, serviceID)
		if err != nil {
			s.l.WithError(err).Warnf("restore is not possible to the same service id %q", serviceID)
			return []*models.Service{}, nil
		}

		return []*models.Service{service}, nil
	}

	filter := models.FindServicesSoftwareVersionsFilter{ServiceType: &serviceType}
	svs, err := models.FindServicesSoftwareVersions(q, filter, models.SoftwareVersionsOrderByServiceID)
	if err != nil {
		return nil, err
	}

	compatibleServiceIDs := make([]string, 0, len(svs))
	for _, sv := range svs {
		svm := softwareVersionsMap(sv.SoftwareVersions)
		serviceDBVersion := svm[models.MysqldSoftwareName]
		if artifactDBVersion != serviceDBVersion {
			s.l.Debugf("skip incompatible service id %q: artifact version %q != db version %q\"", sv.ServiceID,
				artifactDBVersion, serviceDBVersion,
			)
			continue
		}

		if err := mySQLSoftwaresInstalledAndCompatible(svm); err != nil {
			s.l.WithError(err).Debugf("skip incompatible service id %q", sv.ServiceID)
			continue
		}

		compatibleServiceIDs = append(compatibleServiceIDs, sv.ServiceID)
	}

	servicesMap, err := models.FindServicesByIDs(q, compatibleServiceIDs)
	if err != nil {
		return nil, err
	}

	compatibleServices := make([]*models.Service, 0, len(compatibleServiceIDs))
	for _, id := range compatibleServiceIDs {
		compatibleServices = append(compatibleServices, servicesMap[id])
	}

	return compatibleServices, nil
}

func softwareVersionsMap(svs models.SoftwareVersions) map[models.SoftwareName]string {
	m := make(map[models.SoftwareName]string, len(svs))
	for _, sv := range svs {
		m[sv.Name] = sv.Version
	}
	return m
}

func mySQLSoftwaresInstalledAndCompatible(svm map[models.SoftwareName]string) error {
	for _, name := range []models.SoftwareName{
		models.MysqldSoftwareName,
		models.XtrabackupSoftwareName,
		models.XbcloudSoftwareName,
		models.QpressSoftwareName,
	} {
		if svm[name] == "" {
			return errors.Errorf("software %q is not installed", name)
		}
	}

	if svm[models.XtrabackupSoftwareName] != svm[models.XbcloudSoftwareName] {
		return errors.Errorf("xtrabackup version %q != xbcloud version %q",
			svm[models.XtrabackupSoftwareName], svm[models.XbcloudSoftwareName])
	}

	ok, err := mysqlAndXtrabackupCompatible(svm[models.MysqldSoftwareName], svm[models.XtrabackupSoftwareName])
	if err != nil {
		return err
	}
	if !ok {
		return errors.Errorf("mysql version %q is not compatible with xtrabackup version %q",
			svm[models.MysqldSoftwareName], svm[models.XtrabackupSoftwareName])
	}

	return nil
}

// checkSoftwareCompatibilityForService checks if all the necessary backup tools are installed,
// and they are compatible with the db version.
// Returns db version.
func (s *Service) checkSoftwareCompatibilityForService(ctx context.Context, serviceID string) (string, error) {
	pmmAgent, err := s.findPMMAgentForService(ctx, serviceID)
	if err != nil {
		return "", err
	}

	if pmmAgent.serviceType != models.MySQLServiceType {
		return "", nil
	}

	softwares := []agents.Software{&agents.Mysqld{}, &agents.Xtrabackup{}, &agents.Xbcloud{}, &agents.Qpress{}}
	svs, err := s.v.GetVersions(pmmAgent.id, softwares)
	if err != nil {
		return "", err
	}
	if len(svs) != len(softwares) {
		return "", errors.Errorf("response slice len %d != request len %d", len(svs), len(softwares))
	}

	svm := make(map[models.SoftwareName]string, len(softwares))
	for i, software := range softwares {
		name, err := convertSoftwareName(software)
		if err != nil {
			return "", err
		}
		if svs[i].Error != "" {
			return "", errors.Errorf("failed to get software %s version: %s", name, svs[i].Error)
		}

		svm[name] = svs[i].Version
	}

	if err := mySQLSoftwaresInstalledAndCompatible(svm); err != nil {
		return "", err
	}

	return svm[models.MysqldSoftwareName], nil
}

func convertSoftwareName(s agents.Software) (models.SoftwareName, error) {
	var softwareName models.SoftwareName
	switch software := s.(type) {
	case *agents.Mysqld:
		softwareName = models.MysqldSoftwareName
	case *agents.Xtrabackup:
		softwareName = models.XtrabackupSoftwareName
	case *agents.Xbcloud:
		softwareName = models.XbcloudSoftwareName
	case *agents.Qpress:
		softwareName = models.QpressSoftwareName
	default:
		return "", errors.Errorf("invalid software type %T", software)
	}

	return softwareName, nil
}
