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
)

// Service represents core logic for db backup.
type Service struct {
	db          *reform.DB
	jobsService jobsService
	l           *logrus.Entry
}

// NewService creates new backups logic service.
func NewService(db *reform.DB, jobsService jobsService) *Service {
	return &Service{
		l:           logrus.WithField("component", "management/backup/backup"),
		db:          db,
		jobsService: jobsService,
	}
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
	var artifact *models.Artifact
	var location *models.BackupLocation
	var svc *models.Service
	var job *models.Job
	var config *models.DBConfig

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
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
	Location     *models.BackupLocation
	ServiceType  models.ServiceType
	DBConfig     *models.DBConfig
}

// RestoreBackup starts restore backup job.
func (s *Service) RestoreBackup(ctx context.Context, serviceID, artifactID string) (string, error) {
	var params *prepareRestoreJobParams
	var jobID, restoreID string

	err := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		params, err = s.prepareRestoreJob(tx.Querier, serviceID, artifactID)
		if err != nil {
			return err
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
	})
	if err != nil {
		return "", err
	}

	if err := s.startRestoreJob(jobID, serviceID, params); err != nil {
		return "", err
	}

	return restoreID, nil
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

	agents, err := models.FindPMMAgentsForService(q, serviceID)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, errors.Errorf("cannot find pmm agent for service %s", serviceID)
	}

	return &prepareRestoreJobParams{
		AgentID:      agents[0].AgentID,
		ArtifactName: artifact.Name,
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
