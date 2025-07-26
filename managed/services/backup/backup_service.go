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

// Package backup provides backup functionality.
package backup

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services"
)

// Service represents core logic for db backup.
type Service struct {
	l                    *logrus.Entry
	db                   *reform.DB
	jobsService          jobsService
	agentService         agentService
	compatibilityService compatibilityService
	pbmPITRService       pbmPITRService
}

// NewService creates new backups logic service.
func NewService(db *reform.DB, jobsService jobsService, agentService agentService, cSvc compatibilityService, pbmPITRService pbmPITRService) *Service {
	return &Service{
		l:                    logrus.WithField("component", "management/backup/backup"),
		db:                   db,
		jobsService:          jobsService,
		agentService:         agentService,
		compatibilityService: cSvc,
		pbmPITRService:       pbmPITRService,
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
	Folder        string
	Compression   models.BackupCompression
}

// PerformBackup starts on-demand backup.
func (s *Service) PerformBackup(ctx context.Context, params PerformBackupParams) (string, error) { //nolint:cyclop
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

	// Because this transaction uses serializable isolation level it requires retries mechanism.
	var errTX error
	for i := 1; ; i++ {
		errTX = s.db.InTransactionContext(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable}, func(tx *reform.TX) error {
			var err error

			if err = services.CheckArtifactOverlapping(tx.Querier, params.ServiceID, params.LocationID, params.Folder); err != nil {
				return err
			}

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
					return errors.WithMessage(ErrIncompatibleLocationType, "the only supported location type for mySQL is S3")
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

				if err = services.CheckMongoDBBackupPreconditions(tx.Querier, params.Mode, svc.Cluster, svc.ServiceID, params.ScheduleID); err != nil {
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
					Folder:     params.Folder,
				}); err != nil {
					return err
				}
			} else {
				if artifact, err = models.UpdateArtifact(tx.Querier, artifact.ID, models.UpdateArtifactParams{
					Status: models.PendingBackupStatus.Pointer(),
				}); err != nil {
					return err
				}
			}

			if job, dbConfig, err = s.prepareBackupJob(tx.Querier, svc, artifact.ID, jobType, params.Mode, params.DataModel, params.Retries, params.RetryInterval); err != nil { //nolint:lll
				return err
			}
			return nil
		})

		// Cap the number of retries to 30
		if errTX == nil || i >= 30 {
			break
		}

		var pgErr *pq.Error
		if errors.As(errTX, &pgErr) {
			// Serialization failure error code
			if pgErr.Code == "40001" {
				s.l.Infof("Transaction serialization failure, retry iteration %d", i)
				time.Sleep(time.Duration(rand.Intn(100)*i) * time.Millisecond) //nolint:gosec // jitter
				continue
			}
			s.l.Infof("Unknown pq error, code: %s", pgErr.Code.Name())
		}

		return "", errTX
	}

	if errTX != nil {
		return "", errTX
	}

	locationConfig := &models.BackupLocationConfig{
		FilesystemConfig: locationModel.FilesystemConfig,
		S3Config:         locationModel.S3Config,
	}

	switch svc.ServiceType {
	case models.MySQLServiceType:
		err = s.jobsService.StartMySQLBackupJob(job.ID, job.PMMAgentID, 0, name, dbConfig, locationConfig, params.Folder)
	case models.MongoDBServiceType:
		err = s.jobsService.StartMongoDBBackupJob(svc, job.ID, job.PMMAgentID, 0, name, dbConfig,
			job.Data.MongoDBBackup.Mode, job.Data.MongoDBBackup.DataModel, locationConfig, params.Folder)
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		err = status.Errorf(codes.Unimplemented, "Unimplemented service: %s", svc.ServiceType)
	default:
		err = status.Errorf(codes.Unknown, "Unknown service: %s", svc.ServiceType)
	}
	if err != nil {
		var target models.AgentNotSupportedError
		if errors.As(err, &target) {
			_, dbErr := models.UpdateArtifact(s.db.Querier, artifact.ID, models.UpdateArtifactParams{
				Status: models.ErrorBackupStatus.Pointer(),
			})

			if dbErr != nil {
				s.l.WithError(err).Error("failed to update backup artifact status")
			}
			return "", status.Error(codes.FailedPrecondition, target.Error())
		}
		return "", err
	}

	return artifact.ID, nil
}

type restoreJobParams struct {
	JobID         string
	Service       *models.Service
	AgentID       string
	ArtifactName  string
	pbmBackupName string
	LocationModel *models.BackupLocation
	DBConfig      *models.DBConfig
	DataModel     models.DataModel
	PITRTimestamp time.Time
	Folder        string
}

// RestoreBackup starts restore backup job.
func (s *Service) RestoreBackup(ctx context.Context, serviceID, artifactID string, pitrTimestamp time.Time) (string, error) {
	if err := s.checkArtifactModePreconditions(ctx, artifactID, pitrTimestamp); err != nil {
		return "", err
	}

	targetDBVersion, err := s.compatibilityService.CheckSoftwareCompatibilityForService(ctx, serviceID)
	if err != nil {
		return "", err
	}
	if err := s.compatibilityService.CheckArtifactCompatibility(artifactID, targetDBVersion); err != nil {
		return "", err
	}

	var params restoreJobParams
	var restoreID string
	if errTx := s.db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		var err error
		service, err := models.FindServiceByID(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		dbConfig, err := models.FindDBConfigForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}

		pmmAgents, err := models.FindPMMAgentsForService(tx.Querier, serviceID)
		if err != nil {
			return err
		}
		if len(pmmAgents) == 0 {
			return errors.Errorf("cannot find pmm agent for service %s", serviceID)
		}
		agentID := pmmAgents[0].AgentID

		artifact, err := models.FindArtifactByID(tx.Querier, artifactID)
		if err != nil {
			return err
		}

		location, err := models.FindBackupLocationByID(tx.Querier, artifact.LocationID)
		if err != nil {
			return err
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
					ServiceID: serviceID,
					RestoreID: restoreID,
					DataModel: artifact.DataModel,
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
			PMMAgentID: agentID,
			Type:       jobType,
			Data:       jobData,
		})
		if err != nil {
			return err
		}

		var artifactFolder string

		// Only artifacts taken with new agents can be restored from a folder.
		if len(artifact.MetadataList) != 0 {
			artifactFolder = artifact.Folder
		}

		params = restoreJobParams{
			JobID:         job.ID,
			Service:       service,
			AgentID:       agentID,
			ArtifactName:  artifact.Name,
			LocationModel: location,
			DBConfig:      dbConfig,
			DataModel:     artifact.DataModel,
			PITRTimestamp: pitrTimestamp,
			Folder:        artifactFolder,
		}

		if len(artifact.MetadataList) != 0 &&
			artifact.MetadataList[0].BackupToolData != nil &&
			artifact.MetadataList[0].BackupToolData.PbmMetadata != nil {
			params.pbmBackupName = artifact.MetadataList[0].BackupToolData.PbmMetadata.Name
		}

		return nil
	}); errTx != nil {
		return "", errTx
	}

	if err := s.startRestoreJob(&params); err != nil {
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

func (s *Service) startRestoreJob(params *restoreJobParams) error {
	locationConfig := &models.BackupLocationConfig{
		FilesystemConfig: params.LocationModel.FilesystemConfig,
		S3Config:         params.LocationModel.S3Config,
	}

	switch params.Service.ServiceType {
	case models.MySQLServiceType:
		return s.jobsService.StartMySQLRestoreBackupJob(
			params.JobID,
			params.AgentID,
			params.Service.ServiceID, // TODO: It seems that this parameter is redundant
			0,
			params.ArtifactName,
			locationConfig,
			params.Folder)
	case models.MongoDBServiceType:
		return s.jobsService.StartMongoDBRestoreBackupJob(
			params.Service,
			params.JobID,
			params.AgentID,
			0,
			params.ArtifactName,
			params.pbmBackupName,
			params.DBConfig,
			params.DataModel,
			locationConfig,
			params.PITRTimestamp,
			params.Folder)
	case models.PostgreSQLServiceType,
		models.ProxySQLServiceType,
		models.HAProxyServiceType,
		models.ExternalServiceType:
		return status.Errorf(codes.Unimplemented, "Unimplemented service: %s", params.Service.ServiceType)
	default:
		return status.Errorf(codes.Unknown, "Unknown service: %s", params.Service.ServiceType)
	}
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

	if artifact.Status != models.SuccessBackupStatus {
		return errors.Wrapf(ErrArtifactNotReady, "artifact %q in status: %q", artifactID, artifact.Status)
	}

	if artifact.IsShardedCluster {
		return errors.Wrapf(ErrIncompatibleService,
			"artifact %q was made for a sharded cluster and cannot be restored from UI; for more information refer to "+
				"https://docs.percona.com/percona-monitoring-and-management/get-started/backup/backup_mongo.html", artifactID)
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

	storage := GetStorageForLocation(location)
	timeRanges, err := s.pbmPITRService.ListPITRTimeranges(ctx, storage, location, artifact)
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

	if artifact.Mode == models.PITR {
		if pitrTimestamp.Unix() == 0 {
			return errors.Wrapf(ErrIncompatibleArtifactMode, "artifact of type '%s' requires 'time' parameter to be restored to", artifact.Mode)
		}
		if artifact.DataModel == models.PhysicalDataModel {
			return errors.Wrap(ErrIncompatibleArtifactMode, "point in time recovery is only available for Logical data model")
		}
	} else if pitrTimestamp.Unix() != 0 {
		return errors.Wrapf(ErrIncompatibleArtifactMode, "artifact of type '%s' cannot be use to restore to point in time", artifact.Mode)
	}

	return nil
}

// inTimeSpan checks whether given time is in the given range.
func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}
