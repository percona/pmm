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

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
)

// Service represents core logic for db backup.
type Service struct {
	db                   *reform.DB
	jobsService          jobsService
	agentService         agentService
	compatibilityService compatibilityService
	pitrTimerangeService pitrTimerangeService
}

// NewService creates new backups logic service.
func NewService(db *reform.DB, jobsService jobsService, agentService agentService, cSvc compatibilityService, pitrSvc pitrTimerangeService) *Service {
	return &Service{
		db:                   db,
		jobsService:          jobsService,
		agentService:         agentService,
		compatibilityService: cSvc,
		pitrTimerangeService: pitrSvc,
	}
}

// PerformBackupParams are params for performing backup.
type PerformBackupParams struct {
	ServiceID     string
	LocationID    string
	Name          string
	ScheduleID    string
	DataModel     models.DataModel
	Mode          models.BackupMode
	Retries       uint32
	RetryInterval time.Duration
}

// PerformBackup starts on-demand backup.
func (s *Service) PerformBackup(ctx context.Context, params PerformBackupParams) (string, error) {
	dbVersion, err := s.compatibilityService.CheckSoftwareCompatibilityForService(ctx, params.ServiceID)
	if err != nil {
		return "", err
	}

	var artifact *models.Artifact
	var locationModel *models.BackupLocation
	var svc *models.Service
	var job *models.Job
	var dbConfig *models.DBConfig

	name := params.Name
	if params.Mode == models.Snapshot {
		name = name + "_" + time.Now().Format(time.RFC3339)
	}

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		svc, err = models.FindServiceByID(tx.Querier, params.ServiceID)
		if err != nil {
			return err
		}

		locationModel, err = models.FindBackupLocationByID(tx.Querier, params.LocationID)
		if err != nil {
			return err
		}

		var jobType models.JobType
		switch svc.ServiceType {
		case models.MySQLServiceType:
			jobType = models.MySQLBackupJob

			if params.DataModel != models.PhysicalDataModel {
				return errors.WithMessage(ErrIncompatibleDataModel, "the only supported data model for mySQL is physical")
			}

			if locationModel.Type != models.S3BackupLocationType {
				return errors.WithMessage(ErrIncompatibleLocationType, "the only supported location type for mySQL is s3")
			}

			if params.Mode != models.Snapshot {
				return errors.New("the only supported backup mode for mySQL is snapshot")
			}
		case models.MongoDBServiceType:
			jobType = models.MongoDBBackupJob

			if params.Mode == models.PITR && params.DataModel != models.LogicalDataModel {
				return errors.WithMessage(ErrIncompatibleDataModel, "PITR is only supported for logical backups")
			}

			if params.Mode != models.Snapshot && params.Mode != models.PITR {
				return errors.New("the only supported backups mode for mongoDB is snapshot and PITR")
			}

			if err = checkMongoBackupPreconditions(tx.Querier, svc, params.ScheduleID); err != nil {
				return err
			}

			// For PITR backups reuse existing artifact if it's present.
			if params.Mode == models.PITR {
				artifact, err = models.FindArtifactByName(tx.Querier, name)
				if err != nil && !errors.Is(err, models.ErrNotFound) {
					return err
				}
			}

		case models.PostgreSQLServiceType,
			models.ProxySQLServiceType,
			models.HAProxyServiceType,
			models.ExternalServiceType:
			return status.Errorf(codes.Unimplemented, "Unimplemented service: %s", svc.ServiceType)
		default:
			return status.Errorf(codes.Unknown, "Unknown service: %s", svc.ServiceType)
		}

		if artifact == nil {
			if artifact, err = models.CreateArtifact(tx.Querier, models.CreateArtifactParams{
				Name:       name,
				Vendor:     string(svc.ServiceType),
				DBVersion:  dbVersion,
				LocationID: locationModel.ID,
				ServiceID:  svc.ServiceID,
				DataModel:  params.DataModel,
				Mode:       params.Mode,
				Status:     models.PendingBackupStatus,
				ScheduleID: params.ScheduleID,
			}); err != nil {
				return err
			}
		} else {
			if artifact, err = models.UpdateArtifact(tx.Querier, artifact.ID, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.PendingBackupStatus),
			}); err != nil {
				return err
			}
		}

		if job, dbConfig, err = s.prepareBackupJob(tx.Querier, svc, artifact.ID, jobType, params.Mode, params.DataModel, params.Retries, params.RetryInterval); err != nil {
			return err
		}
		return nil
	})

	if errTX != nil {
		return "", errTX
	}

	locationConfig := &models.BackupLocationConfig{
		PMMClientConfig: locationModel.PMMClientConfig,
		S3Config:        locationModel.S3Config,
	}

	switch svc.ServiceType {
	case models.MySQLServiceType:
		err = s.jobsService.StartMySQLBackupJob(job.ID, job.PMMAgentID, 0, name, dbConfig, locationConfig)
	case models.MongoDBServiceType:
		err = s.jobsService.StartMongoDBBackupJob(job.ID, job.PMMAgentID, 0, name, dbConfig, job.Data.MongoDBBackup.Mode, job.Data.MongoDBBackup.DataModel, locationConfig)
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		err = status.Errorf(codes.Unimplemented, "Unimplemented service: %s", svc.ServiceType)
	default:
		err = status.Errorf(codes.Unknown, "Unknown service: %s", svc.ServiceType)
	}
	if err != nil {
		return "", err
	}

	return artifact.ID, nil
}

func checkMongoBackupPreconditions(q *reform.Querier, service *models.Service, scheduleID string) error {
	tasks, err := models.FindScheduledTasks(q, models.ScheduledTasksFilter{
		Disabled:  pointer.ToBool(false),
		ServiceID: service.ServiceID,
		Mode:      models.PITR,
	})
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.ID != scheduleID {
			return status.Errorf(codes.FailedPrecondition, "Can't make a backup because service %s already has "+
				"scheduled PITR backups. Please disable them if you want to make another backup.", service.ServiceName)
		}
	}

	return nil
}

type prepareRestoreJobParams struct {
	AgentID       string
	ArtifactName  string
	DBVersion     string
	LocationModel *models.BackupLocation
	ServiceType   models.ServiceType
	DBConfig      *models.DBConfig
	DataModel     models.DataModel
	PITRTimestamp time.Time
}

// RestoreBackup starts restore backup job.
func (s *Service) RestoreBackup(ctx context.Context, serviceID, artifactID string, pitrTimestamp time.Time) (string, error) {
	if err := s.checkArtifactModePreconditions(ctx, artifactID, pitrTimestamp); err != nil {
		return "", err
	}

	dbVersion, err := s.compatibilityService.CheckSoftwareCompatibilityForService(ctx, serviceID)
	if err != nil {
		return "", err
	}

	var params *prepareRestoreJobParams
	var jobID, restoreID string
	if errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		params, err = s.prepareRestoreJob(tx.Querier, serviceID, artifactID, pitrTimestamp)
		if err != nil {
			return err
		}

		if params.ServiceType == models.MySQLServiceType && params.DBVersion != "" {
			if params.DBVersion != dbVersion {
				return errors.Wrapf(ErrIncompatibleTargetMySQL, "artifact db version %q != db version %q",
					params.DBVersion, dbVersion)
			}
		}

		restore, err := models.CreateRestoreHistoryItem(tx.Querier, models.CreateRestoreHistoryItemParams{
			ArtifactID:    artifactID,
			ServiceID:     serviceID,
			PITRTimestamp: &pitrTimestamp,
			Status:        models.InProgressRestoreStatus,
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
				},
			}
		case models.MongoDBServiceType:
			jobType = models.MongoDBRestoreBackupJob
			jobData = &models.JobData{
				MongoDBRestoreBackup: &models.MongoDBRestoreBackupJobData{
					RestoreID: restoreID,
				},
			}
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
	}); errTx != nil {
		return "", errTx
	}

	if err := s.startRestoreJob(jobID, serviceID, params); err != nil {
		return "", err
	}

	return restoreID, nil
}

// SwitchMongoPITR switches Point-in-Time recovery feature for mongoDB with given serviceID.
func (s *Service) SwitchMongoPITR(ctx context.Context, serviceID string, enabled bool) error {
	var pmmAgentID, dsn string
	var agent *models.Agent
	var service *models.Service

	errTX := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		service, err = models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		if service.ServiceType != models.MongoDBServiceType {
			return errors.Errorf("Point-in-Time recovery feature is only available for mongoDB services,"+
				"current service id: %s, service type: %s", serviceID, service.ServiceType)
		}

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}
		if len(pmmAgents) == 0 {
			return errors.Errorf("cannot find pmm agent for service %s", serviceID)
		}
		pmmAgentID = pmmAgents[0].AgentID

		dsn, agent, err = models.FindDSNByServiceIDandPMMAgentID(tx.Querier, serviceID, pmmAgentID, "")
		if err != nil {
			return err
		}
		return nil
	})
	if errTX != nil {
		return errTX
	}

	return s.agentService.PBMSwitchPITR(
		pmmAgentID,
		dsn,
		agent.Files(),
		agent.TemplateDelimiters(service),
		enabled)
}

func (s *Service) prepareRestoreJob(
	q *reform.Querier,
	serviceID string,
	artifactID string,
	pitrTimestamp time.Time,
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
		AgentID:       pmmAgents[0].AgentID,
		ArtifactName:  artifact.Name,
		DBVersion:     artifact.DBVersion,
		LocationModel: location,
		ServiceType:   service.ServiceType,
		DBConfig:      dbConfig,
		DataModel:     artifact.DataModel,
		PITRTimestamp: pitrTimestamp,
	}, nil
}

func (s *Service) startRestoreJob(jobID, serviceID string, params *prepareRestoreJobParams) error {
	locationConfig := &models.BackupLocationConfig{
		PMMClientConfig: params.LocationModel.PMMClientConfig,
		S3Config:        params.LocationModel.S3Config,
	}

	switch params.ServiceType {
	case models.MySQLServiceType:
		if err := s.jobsService.StartMySQLRestoreBackupJob(
			jobID,
			params.AgentID,
			serviceID,
			0,
			params.ArtifactName,
			locationConfig); err != nil {
			return err
		}
	case models.MongoDBServiceType:
		if err := s.jobsService.StartMongoDBRestoreBackupJob(
			jobID,
			params.AgentID,
			0,
			params.ArtifactName,
			params.DBConfig,
			params.DataModel,
			locationConfig,
			params.PITRTimestamp); err != nil {
			return err
		}
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		return status.Errorf(codes.Unimplemented, "Unimplemented service: %s", params.ServiceType)
	default:
		return status.Errorf(codes.Unknown, "Unknown service: %s", params.ServiceType)
	}

	return nil
}

func (s *Service) prepareBackupJob(
	q *reform.Querier,
	service *models.Service,
	artifactID string,
	jobType models.JobType,
	mode models.BackupMode,
	dataModel models.DataModel,
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
				Mode:       mode,
				DataModel:  dataModel,
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

// checkArtifactModePreconditions checks that artifact params and requested restore mode satisfy each other.
func (s *Service) checkArtifactModePreconditions(ctx context.Context, artifactID string, pitrTimestamp time.Time) error {
	artifact, err := models.FindArtifactByID(s.db.Querier, artifactID)
	if err != nil {
		return err
	}

	if err := checkArtifactMode(artifact, pitrTimestamp); err != nil {
		return err
	}

	// Continue checks only if user requested PITR restore.
	if pitrTimestamp.Unix() == 0 {
		return nil
	}

	location, err := models.FindBackupLocationByID(s.db.Querier, artifact.LocationID)
	if err != nil {
		return err
	}

	if location.Type != models.S3BackupLocationType {
		return errors.Wrapf(ErrIncompatibleLocationType, "point in time recovery available only for S3 locations")
	}

	timeRanges, err := s.pitrTimerangeService.ListPITRTimeranges(ctx, artifact.Name, location)
	if err != nil {
		return err
	}

	for _, tRange := range timeRanges {
		if inTimeSpan(time.Unix(int64(tRange.Start), 0), time.Unix(int64(tRange.End), 0), pitrTimestamp) {
			return nil
		}
	}

	return errors.Wrapf(ErrTimestampOutOfRange, "point in time recovery value %s", pitrTimestamp.String())
}

// checkArtifactMode crosschecks artifact params and requested restore mode.
func checkArtifactMode(artifact *models.Artifact, pitrTimestamp time.Time) error {
	if artifact.Vendor != string(models.MongoDBServiceType) && artifact.Mode == models.PITR {
		return errors.Wrapf(ErrIncompatibleService, "restore to point in time is only available for MongoDB")
	}

	if artifact.Mode != models.PITR {
		if pitrTimestamp.Unix() == 0 {
			return nil
		}
		if pitrTimestamp.Unix() != 0 {
			return errors.Wrapf(ErrIncompatibleArtifactMode, "artifact of type '%s' cannot be use to restore to point in time", artifact.Mode)
		}
	} else {
		if pitrTimestamp.Unix() == 0 {
			return errors.Wrapf(ErrIncompatibleArtifactMode, "artifact of type '%s' requires 'time' parameter to be restored to", artifact.Mode)
		}
		if artifact.DataModel == models.PhysicalDataModel {
			return errors.Wrap(ErrIncompatibleArtifactMode, "point in time recovery is only available for Logical data model")
		}
	}

	return nil
}

// inTimeSpan checks whether given time is in the given range
func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}
