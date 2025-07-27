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

// Package agents provides jobs functionality.
package agents

import (
	"context"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/agentpb"
	backuppb "github.com/percona/pmm/api/managementpb/backup"
	"github.com/percona/pmm/managed/models"
)

var (
	// ErrRetriesExhausted is returned when remaining retries are 0.
	ErrRetriesExhausted = errors.New("retries exhausted")

	pmmAgentMinVersionForMongoLogicalBackupAndRestore  = version.Must(version.NewVersion("2.19"))
	pmmAgentMinVersionForMySQLBackupAndRestore         = version.Must(version.NewVersion("2.23"))
	pmmAgentMinVersionForMongoPhysicalBackupAndRestore = version.Must(version.NewVersion("2.31.0-0"))
	pmmAgentMinVersionForMongoDBUseFilesystemStorage   = version.Must(version.NewVersion("2.32.0-0"))
	pmmAgentMinVersionForMongoPITRRestore              = version.Must(version.NewVersion("2.32.0-0"))
)

const (
	maxRestartInterval = 8 * time.Hour
)

// JobsService provides methods for managing jobs.
type JobsService struct {
	r  *Registry
	db *reform.DB

	retentionService retentionService
	l                *logrus.Entry
}

// NewJobsService returns new jobs service.
func NewJobsService(db *reform.DB, registry *Registry, retention retentionService) *JobsService {
	return &JobsService{
		db:               db,
		r:                registry,
		retentionService: retention,
		l:                logrus.WithField("component", "agents/jobsService"),
	}
}

// RestartJob restarts a job with the given jobID.
func (s *JobsService) RestartJob(ctx context.Context, jobID string) error {
	var job *models.Job
	var artifact *models.Artifact
	var locationModel *models.BackupLocation
	var locationConfig *models.BackupLocationConfig
	var dbConfig *models.DBConfig
	errTx := s.db.InTransaction(func(tx *reform.TX) error {
		var err error
		job, err = models.FindJobByID(tx.Querier, jobID)
		if err != nil {
			return errors.WithStack(err)
		}

		if job.Retries == 0 {
			return ErrRetriesExhausted
		}

		job.Retries--
		if err = tx.Update(job); err != nil {
			return err
		}

		switch job.Type {
		case models.MySQLBackupJob:
			artifact, err = models.FindArtifactByID(tx.Querier, job.Data.MySQLBackup.ArtifactID)
			if err != nil {
				return errors.WithStack(err)
			}

			locationModel, err = models.FindBackupLocationByID(tx.Querier, artifact.LocationID)
			if err != nil {
				return errors.WithStack(err)
			}
			dbConfig, err = models.FindDBConfigForService(tx.Querier, job.Data.MySQLBackup.ServiceID)
			if err != nil {
				return errors.WithStack(err)
			}
		case models.MongoDBBackupJob:
			artifact, err = models.FindArtifactByID(tx.Querier, job.Data.MongoDBBackup.ArtifactID)
			if err != nil {
				return errors.WithStack(err)
			}

			locationModel, err = models.FindBackupLocationByID(tx.Querier, artifact.LocationID)
			if err != nil {
				return errors.WithStack(err)
			}

			dbConfig, err = models.FindDBConfigForService(tx.Querier, job.Data.MongoDBBackup.ServiceID)
			if err != nil {
				return errors.WithStack(err)
			}

		case models.MySQLRestoreBackupJob, models.MongoDBRestoreBackupJob:
			fallthrough
		default:
			return errors.Errorf("job type %v can't be restarted", job.Type)
		}

		return nil
	})
	if errTx != nil {
		return errTx
	}

	if locationModel != nil {
		locationConfig = &models.BackupLocationConfig{
			FilesystemConfig: locationModel.FilesystemConfig,
			S3Config:         locationModel.S3Config,
		}
	}

	s.l.Debugf("restarting job: %s, delay: %v", jobID, job.Interval)

	select {
	case <-time.After(job.Interval):
	case <-ctx.Done():
		return ctx.Err()
	}

	switch job.Type {
	case models.MySQLBackupJob:
		if err := s.StartMySQLBackupJob(job.ID, job.PMMAgentID, job.Timeout, artifact.Name, dbConfig, locationConfig, artifact.Folder, artifact.Compression); err != nil {
			return errors.WithStack(err)
		}
	case models.MongoDBBackupJob:
		service, err := models.FindServiceByID(s.db.Querier, job.Data.MongoDBBackup.ServiceID)
		if err != nil {
			return err
		}

		if err := s.StartMongoDBBackupJob(service, job.ID, job.PMMAgentID, job.Timeout, artifact.Name, dbConfig,
			job.Data.MongoDBBackup.Mode, job.Data.MongoDBBackup.DataModel, locationConfig, artifact.Folder, artifact.Compression); err != nil {
			return errors.WithStack(err)
		}
	case models.MySQLRestoreBackupJob:
	case models.MongoDBRestoreBackupJob:
	}

	return nil
}

func (s *JobsService) handleJobResult(_ context.Context, l *logrus.Entry, result *agentpb.JobResult) { //nolint:cyclop
	var scheduleID string
	if errTx := s.db.InTransaction(func(t *reform.TX) error {
		job, err := models.FindJobByID(t.Querier, result.JobId)
		if err != nil {
			return err
		}

		switch result := result.Result.(type) {
		case *agentpb.JobResult_Error_:
			if err := s.handleJobError(job); err != nil {
				l.Errorf("failed to handle job error: %s", err)
			}
			job.Error = result.Error.Message
		case *agentpb.JobResult_MysqlBackup:
			if job.Type != models.MySQLBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MySQLBackupJob, job.Type)
			}

			artifact, err := models.UpdateArtifact(
				t.Querier,
				job.Data.MySQLBackup.ArtifactID,
				models.UpdateArtifactParams{
					Status:   models.SuccessBackupStatus.Pointer(),
					Metadata: artifactMetadataFromProto(result.MysqlBackup.Metadata),
				})
			if err != nil {
				return err
			}

			if artifact.Type == models.ScheduledArtifactType {
				scheduleID = artifact.ScheduleID
			}
		case *agentpb.JobResult_MongodbBackup:
			if job.Type != models.MongoDBBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MongoDBBackupJob, job.Type)
			}

			metadata := artifactMetadataFromProto(result.MongodbBackup.Metadata)

			artifact, err := models.UpdateArtifact(
				t.Querier,
				job.Data.MongoDBBackup.ArtifactID,
				models.UpdateArtifactParams{
					Status:           models.SuccessBackupStatus.Pointer(),
					IsShardedCluster: result.MongodbBackup.IsShardedCluster,
					Metadata:         metadata,
				})
			if err != nil {
				return err
			}

			if artifact.Type == models.ScheduledArtifactType {
				scheduleID = artifact.ScheduleID
			}

			// If task was running by an old agent. Hacky code to support artifacts created on new server and old agent.
			if metadata == nil && artifact.Mode == models.PITR && artifact.Folder != artifact.Name {
				artifact, err := models.UpdateArtifact(t.Querier, artifact.ID, models.UpdateArtifactParams{Folder: &artifact.Name})
				if err != nil {
					return errors.Wrapf(err, "failed to update artifact %s", artifact.ID)
				}

				task, err := models.FindScheduledTaskByID(t.Querier, scheduleID)
				if err != nil {
					return errors.Wrapf(err, "cannot get scheduled task %s", scheduleID)
				}
				taskData := task.Data
				taskData.MongoDBBackupTask.CommonBackupTaskData.Folder = artifact.Name

				params := models.ChangeScheduledTaskParams{
					Data: taskData,
				}

				_, err = models.ChangeScheduledTask(t.Querier, scheduleID, params)
				if err != nil {
					return errors.Wrapf(err, "failed to update scheduled task %s", scheduleID)
				}
			}

		case *agentpb.JobResult_MysqlRestoreBackup:
			if job.Type != models.MySQLRestoreBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MySQLRestoreBackupJob, job.Type)
			}

			_, err := models.ChangeRestoreHistoryItem(
				t.Querier,
				job.Data.MySQLRestoreBackup.RestoreID,
				models.ChangeRestoreHistoryItemParams{
					Status:     models.SuccessRestoreStatus,
					FinishedAt: pointer.ToTime(models.Now()),
				})
			if err != nil {
				return err
			}

		case *agentpb.JobResult_MongodbRestoreBackup:
			if job.Type != models.MongoDBRestoreBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MongoDBRestoreBackupJob, job.Type)
			}

			if job.Data.MongoDBRestoreBackup.DataModel == models.LogicalDataModel {
				s.l.Info("restore successfully completed")
			} else if job.Data.MongoDBRestoreBackup.DataModel == models.PhysicalDataModel {
				s.l.Info("restore successfully completed, PMM will restart mongod and pbm-agent")
				if err := s.runMongoPostRestore(t.Querier, job.Data.MongoDBRestoreBackup.ServiceID); err != nil {
					s.l.WithError(err).Error("failed to restart components after restore from a physical backup")

					_, err = models.ChangeRestoreHistoryItem(
						t.Querier,
						job.Data.MongoDBRestoreBackup.RestoreID,
						models.ChangeRestoreHistoryItemParams{
							Status:     models.ErrorRestoreStatus,
							FinishedAt: pointer.ToTime(models.Now()),
						})
					return err
				} else {
					s.l.Info("successfully restarted mongod and pbm-agent on all cluster members")
				}
			}

			_, err = models.ChangeRestoreHistoryItem(
				t.Querier,
				job.Data.MongoDBRestoreBackup.RestoreID,
				models.ChangeRestoreHistoryItemParams{
					Status:     models.SuccessRestoreStatus,
					FinishedAt: pointer.ToTime(models.Now()),
				})
			if err != nil {
				return err
			}
		default:
			return errors.Errorf("unexpected job result type: %T", result)
		}
		job.Done = true
		return t.Update(job)
	}); errTx != nil {
		l.Errorf("Failed to save job result: %+v", errTx)
	}

	if scheduleID != "" {
		go func() {
			if err := s.retentionService.EnforceRetention(scheduleID); err != nil {
				l.Errorf("failed to enforce retention: %v", err)
			}
		}()
	}
}

func (s *JobsService) handleJobError(job *models.Job) error {
	var err error
	switch job.Type {
	case models.MySQLBackupJob:
		_, err = models.UpdateArtifact(s.db.Querier, job.Data.MySQLBackup.ArtifactID, models.UpdateArtifactParams{
			Status: models.ErrorBackupStatus.Pointer(),
		})
	case models.MongoDBBackupJob:
		_, err = models.UpdateArtifact(s.db.Querier, job.Data.MongoDBBackup.ArtifactID, models.UpdateArtifactParams{
			Status: models.ErrorBackupStatus.Pointer(),
		})
	case models.MySQLRestoreBackupJob:
		_, err = models.ChangeRestoreHistoryItem(
			s.db.Querier,
			job.Data.MySQLRestoreBackup.RestoreID,
			models.ChangeRestoreHistoryItemParams{
				Status:     models.ErrorRestoreStatus,
				FinishedAt: pointer.ToTime(models.Now()),
			})
	case models.MongoDBRestoreBackupJob:
		_, err = models.ChangeRestoreHistoryItem(
			s.db.Querier,
			job.Data.MongoDBRestoreBackup.RestoreID,
			models.ChangeRestoreHistoryItemParams{
				Status:     models.ErrorRestoreStatus,
				FinishedAt: pointer.ToTime(models.Now()),
			})
	default:
		return errors.Errorf("unknown job type %s", job.Type)
	}

	go func() {
		restartCtx, cancel := context.WithTimeout(context.Background(), maxRestartInterval)
		defer cancel()
		restartErr := s.RestartJob(restartCtx, job.ID)
		if restartErr != nil && !errors.Is(restartErr, ErrRetriesExhausted) {
			s.l.Errorf("restart job %s: %v", job.ID, restartErr)
		}
	}()

	return err
}

func (s *JobsService) handleJobProgress(_ context.Context, progress *agentpb.JobProgress) {
	switch result := progress.Result.(type) {
	case *agentpb.JobProgress_Logs_:
		err := createJobLog(s.db.Querier, progress.JobId, result.Logs.Data, int(result.Logs.ChunkId), result.Logs.Done)
		if err != nil {
			s.l.WithError(err).Errorf("failed to create log for job %s [chunk: %d]", progress.JobId, result.Logs.ChunkId)
		}
	default:
		s.l.Errorf("unexpected job progress type: %T", result)
	}
}

// StartMySQLBackupJob starts mysql backup job on the pmm-agent.
func (s *JobsService) StartMySQLBackupJob(jobID, pmmAgentID string, timeout time.Duration, name string, dbConfig *models.DBConfig, locationConfig *models.BackupLocationConfig, folder string, compression models.BackupCompression) error { //nolint:lll
	if err := models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
		"mysql backup", pmmAgentMinVersionForMySQLBackupAndRestore); err != nil {
		return err
	}

	mySQLReq := &agentpb.StartJobRequest_MySQLBackup{
		Name:     name,
		User:     dbConfig.User,
		Password: dbConfig.Password,
		Address:  dbConfig.Address,
		Port:     int32(dbConfig.Port),
		Socket:   dbConfig.Socket,
		Folder:   folder,
	}

	var err error
	if mySQLReq.Compression, err = convertBackupCompression(compression); err != nil {
		return err
	}

	switch {
	case locationConfig.S3Config != nil:
		mySQLReq.LocationConfig = &agentpb.StartJobRequest_MySQLBackup_S3Config{
			S3Config: convertS3ConfigModel(locationConfig.S3Config),
		}
	default:
		return errors.Errorf("unsupported location config")
	}
	req := &agentpb.StartJobRequest{
		JobId:   jobID,
		Timeout: durationpb.New(timeout),
		Job: &agentpb.StartJobRequest_MysqlBackup{
			MysqlBackup: mySQLReq,
		},
	}

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	resp, err := agent.channel.SendAndWaitResponse(req)
	if err != nil {
		return err
	}
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" { //nolint:forcetypeassert
		return errors.Errorf("failed to start MySQL backup job: %s", e)
	}

	return nil
}

// StartMongoDBBackupJob starts mongoDB backup job on the pmm-agent.
func (s *JobsService) StartMongoDBBackupJob(
	service *models.Service,
	jobID string,
	pmmAgentID string,
	timeout time.Duration,
	name string,
	dbConfig *models.DBConfig,
	mode models.BackupMode,
	dataModel models.DataModel,
	locationConfig *models.BackupLocationConfig,
	folder string,
	compression models.BackupCompression,
) error {
	var err error
	switch dataModel {
	case models.PhysicalDataModel:
		err = models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb physical backup", pmmAgentMinVersionForMongoPhysicalBackupAndRestore)
	case models.LogicalDataModel:
		err = models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb logical backup", pmmAgentMinVersionForMongoLogicalBackupAndRestore)
	default:
		err = errors.Errorf("unknown data model: %s", dataModel)
	}
	if err != nil {
		return err
	}

	dsn, agent, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, service.ServiceID, pmmAgentID, "")
	if err != nil {
		return err
	}

	delimiters := agent.TemplateDelimiters(service)

	mongoDBReq := &agentpb.StartJobRequest_MongoDBBackup{
		Name:       name,
		EnablePitr: mode == models.PITR,
		Folder:     folder,
		Dsn:        dsn,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  delimiters.Left,
			TemplateRightDelim: delimiters.Right,
		},

		// Following group of parameters used only for legacy agents. Deprecated since v2.38.
		User:     dbConfig.User,
		Password: dbConfig.Password,
		Address:  dbConfig.Address,
		Port:     int32(dbConfig.Port),
		Socket:   dbConfig.Socket,
	}
	if mongoDBReq.DataModel, err = convertDataModel(dataModel); err != nil {
		return err
	}

	switch {
	case locationConfig.S3Config != nil:
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBBackup_S3Config{
			S3Config: convertS3ConfigModel(locationConfig.S3Config),
		}
	case locationConfig.FilesystemConfig != nil:
		if err := models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb backup to client local storage",
			pmmAgentMinVersionForMongoDBUseFilesystemStorage); err != nil {
			return err
		}
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBBackup_FilesystemConfig{
			FilesystemConfig: &agentpb.FilesystemLocationConfig{Path: locationConfig.FilesystemConfig.Path},
		}
	default:
		return errors.Errorf("unsupported location config")
	}
	req := &agentpb.StartJobRequest{
		JobId:   jobID,
		Timeout: durationpb.New(timeout),
		Job: &agentpb.StartJobRequest_MongodbBackup{
			MongodbBackup: mongoDBReq,
		},
	}

	agentInfo, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	resp, err := agentInfo.channel.SendAndWaitResponse(req)
	if err != nil {
		return err
	}
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" { //nolint:forcetypeassert
		return errors.Errorf("failed to start MongoDB backup job: %s", e)
	}

	return nil
}

// StartMySQLRestoreBackupJob starts mysql restore backup job on the pmm-agent.
func (s *JobsService) StartMySQLRestoreBackupJob(
	jobID string,
	pmmAgentID string,
	serviceID string,
	timeout time.Duration,
	name string,
	locationConfig *models.BackupLocationConfig,
	folder string,
	compression models.BackupCompression,
) error {
	if err := models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
		"mysql restore", pmmAgentMinVersionForMySQLBackupAndRestore); err != nil {
		return err
	}

	if locationConfig.S3Config == nil {
		return errors.Errorf("location config is not set")
	}

	req := &agentpb.StartJobRequest{
		JobId:   jobID,
		Timeout: durationpb.New(timeout),
		Job: &agentpb.StartJobRequest_MysqlRestoreBackup{
			MysqlRestoreBackup: &agentpb.StartJobRequest_MySQLRestoreBackup{
				ServiceId: serviceID,
				Name:      name,
				Folder:    folder,
				LocationConfig: &agentpb.StartJobRequest_MySQLRestoreBackup_S3Config{
					S3Config: convertS3ConfigModel(locationConfig.S3Config),
				},
			},
		},
	}

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	resp, err := agent.channel.SendAndWaitResponse(req)
	if err != nil {
		return err
	}
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" { //nolint:forcetypeassert
		return errors.Errorf("failed to start MySQL restore backup job: %s", e)
	}

	return nil
}

// StartMongoDBRestoreBackupJob starts mongo restore backup job on the pmm-agent.
func (s *JobsService) StartMongoDBRestoreBackupJob(
	service *models.Service,
	jobID string,
	pmmAgentID string,
	timeout time.Duration,
	name string,
	pbmBackupName string,
	dbConfig *models.DBConfig,
	dataModel models.DataModel,
	locationConfig *models.BackupLocationConfig,
	pitrTimestamp time.Time,
	folder string,
	compression models.BackupCompression,
) error {
	var err error
	switch dataModel {
	case models.PhysicalDataModel:
		err = models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb physical restore", pmmAgentMinVersionForMongoPhysicalBackupAndRestore)
	case models.LogicalDataModel:
		err = models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb logical restore", pmmAgentMinVersionForMongoLogicalBackupAndRestore)
	default:
		err = errors.Errorf("unknown data model: %s", dataModel)
	}
	if err != nil {
		return err
	}

	if pitrTimestamp.Unix() != 0 {
		// TODO refactor pmm agent version checking. First detect minimum required version needed for operations and
		// then invoke PMMAgentSupported
		if err = models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb pitr restore", pmmAgentMinVersionForMongoPITRRestore); err != nil {
			return err
		}
	}

	dsn, agent, err := models.FindDSNByServiceIDandPMMAgentID(s.db.Querier, service.ServiceID, pmmAgentID, "")
	if err != nil {
		return err
	}

	delimiters := agent.TemplateDelimiters(service)

	mongoDBReq := &agentpb.StartJobRequest_MongoDBRestoreBackup{
		Name:          name,
		PitrTimestamp: timestamppb.New(pitrTimestamp),
		Folder:        folder,
		PbmMetadata:   &backuppb.PbmMetadata{Name: pbmBackupName},
		Dsn:           dsn,
		TextFiles: &agentpb.TextFiles{
			Files:              agent.Files(),
			TemplateLeftDelim:  delimiters.Left,
			TemplateRightDelim: delimiters.Right,
		},

		// Following group of parameters used only for legacy agents. Deprecated since v2.38.
		User:     dbConfig.User,
		Password: dbConfig.Password,
		Address:  dbConfig.Address,
		Port:     int32(dbConfig.Port),
		Socket:   dbConfig.Socket,
	}

	switch {
	case locationConfig.S3Config != nil:
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBRestoreBackup_S3Config{
			S3Config: convertS3ConfigModel(locationConfig.S3Config),
		}
	case locationConfig.FilesystemConfig != nil:
		if err := models.PMMAgentSupported(s.r.db.Querier, pmmAgentID,
			"mongodb restore from client local storage",
			pmmAgentMinVersionForMongoDBUseFilesystemStorage); err != nil {
			return err
		}
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBRestoreBackup_FilesystemConfig{
			FilesystemConfig: &agentpb.FilesystemLocationConfig{Path: locationConfig.FilesystemConfig.Path},
		}
	default:
		return errors.Errorf("unsupported location config")
	}

	req := &agentpb.StartJobRequest{
		JobId:   jobID,
		Timeout: durationpb.New(timeout),
		Job: &agentpb.StartJobRequest_MongodbRestoreBackup{
			MongodbRestoreBackup: mongoDBReq,
		},
	}

	agentInfo, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	resp, err := agentInfo.channel.SendAndWaitResponse(req)
	if err != nil {
		return err
	}
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" { //nolint:forcetypeassert
		return errors.Errorf("failed to start MonogDB restore backup job: %s", e)
	}

	return nil
}

func (s *JobsService) runMongoPostRestore(querier *reform.Querier, serviceID string) error {
	service, err := models.FindServiceByID(querier, serviceID)
	if err != nil {
		return err
	}
	if service.Cluster == "" {
		return errors.Errorf("service '%s' has an empty cluster name and needs to be manually restarted", service.ServiceID)
	}

	serviceType := models.MongoDBServiceType
	clusterMembers, err := models.FindServices(
		querier,
		models.ServiceFilters{
			ServiceType: &serviceType,
			Cluster:     service.Cluster,
		})
	if err != nil {
		return err
	}

	clusterAgents := make([]*models.Agent, 0, len(clusterMembers))
	for _, service := range clusterMembers {
		s.l.Debugf("found service: %s in replica set: %s", service.ServiceName, service.ReplicationSet)
		serviceAgents, err := models.FindPMMAgentsForService(querier, service.ServiceID)
		if err != nil {
			return errors.Wrapf(err, "failed to get pmm agent for replica set member: %s", service.ServiceID)
		}
		if len(serviceAgents) == 0 {
			return errors.Errorf("cannot find pmm agent for service %s", service.ServiceID)
		}
		clusterAgents = append(clusterAgents, serviceAgents[0])
	}

	// mongoRestarts is a list of PMM agent IDs on which we successfully restarted mongod
	mongoRestarts := make(map[string]struct{})
	// pbmRestarts is a list of PMM agent IDs on which we successfully restarted pbm-agent
	pbmAgentRestarts := make(map[string]struct{})

	for _, pmmAgent := range clusterAgents {
		if err = s.restartSystemService(pmmAgent.AgentID, agentpb.StartActionRequest_RestartSystemServiceParams_MONGOD); err != nil {
			return err
		}
		mongoRestarts[pmmAgent.AgentID] = struct{}{}
	}
	s.l.Infof("successfully restarted mongod on all %d services", len(mongoRestarts))

	// pbm-agents will fail if all members of the mongo replica set are not available,
	// hence we restart them only if mongod have been started on all the member agents.
	for _, pmmAgent := range clusterAgents {
		if err = s.restartSystemService(pmmAgent.AgentID, agentpb.StartActionRequest_RestartSystemServiceParams_PBM_AGENT); err != nil {
			return err
		}
		pbmAgentRestarts[pmmAgent.AgentID] = struct{}{}
	}
	s.l.Infof("successfully restarted pbm-agent on all %d services", len(pbmAgentRestarts))
	return nil
}

func (s *JobsService) restartSystemService(agentID string, service agentpb.StartActionRequest_RestartSystemServiceParams_SystemService) error {
	s.l.Infof("sending request to restart %s on %s", service, agentID)
	action, err := models.CreateActionResult(s.db.Querier, agentID)
	if err != nil {
		return err
	}

	req := &agentpb.StartActionRequest{
		ActionId: action.ID,
		Params: &agentpb.StartActionRequest_RestartSysServiceParams{
			RestartSysServiceParams: &agentpb.StartActionRequest_RestartSystemServiceParams{
				SystemService: service,
			},
		},
	}

	agent, err := s.r.get(agentID)
	if err != nil {
		return errors.Wrapf(err, "failed to get information about PMM agent: %s", agentID)
	}
	_, err = agent.channel.SendAndWaitResponse(req)
	if err != nil {
		return errors.Wrapf(err, "failed to restart %s on agent: %s", service, agentID)
	}
	return nil
}

// StopJob stops job with given id.
func (s *JobsService) StopJob(jobID string) error {
	jobResult, err := models.FindJobByID(s.db.Querier, jobID)
	if err != nil {
		return errors.WithStack(err)
	}

	if jobResult.Done {
		// Job already finished
		return nil
	}

	agent, err := s.r.get(jobResult.PMMAgentID)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = agent.channel.SendAndWaitResponse(&agentpb.StopJobRequest{JobId: jobID})

	return err
}

func convertS3ConfigModel(config *models.S3LocationConfig) *agentpb.S3LocationConfig {
	return &agentpb.S3LocationConfig{
		Endpoint:     config.Endpoint,
		AccessKey:    config.AccessKey,
		SecretKey:    config.SecretKey,
		BucketName:   config.BucketName,
		BucketRegion: config.BucketRegion,
	}
}

func convertDataModel(model models.DataModel) (backuppb.DataModel, error) {
	switch model {
	case models.PhysicalDataModel:
		return backuppb.DataModel_PHYSICAL, nil
	case models.LogicalDataModel:
		return backuppb.DataModel_LOGICAL, nil
	default:
		return 0, errors.Errorf("unknown data model: %s", model)
	}
}

func convertBackupCompression(compression models.BackupCompression) (backuppb.BackupCompression, error) {
	switch compression {
	case models.QuickLZ:
		return backuppb.BackupCompression_QUICKLZ, nil
	case models.ZSTD:
		return backuppb.BackupCompression_ZSTD, nil
	case models.LZ4:
		return backuppb.BackupCompression_LZ4, nil
	case models.S2:
		return backuppb.BackupCompression_S2, nil
	case models.GZIP:
		return backuppb.BackupCompression_GZIP, nil
	case models.Snappy:
		return backuppb.BackupCompression_SNAPPY, nil
	case models.PGZIP:
		return backuppb.BackupCompression_PGZIP, nil
	case models.None:
		return backuppb.BackupCompression_NONE, nil
	default:
		return 0, errors.Errorf("invalid compression '%s'", compression)
	}
}

func createJobLog(querier *reform.Querier, jobID, data string, chunkID int, lastChunk bool) error {
	_, err := models.CreateJobLog(
		querier,
		models.CreateJobLogParams{
			JobID:     jobID,
			ChunkID:   chunkID,
			Data:      data,
			LastChunk: lastChunk,
		})
	return err
}

// artifactMetadataFromProto returns artifact metadata converted from protobuf to Go model format.
func artifactMetadataFromProto(metadata *backuppb.Metadata) *models.Metadata {
	if metadata == nil {
		return nil
	}

	files := make([]models.File, len(metadata.FileList))
	for i, file := range metadata.FileList {
		files[i] = models.File{Name: file.Name, IsDirectory: file.IsDirectory}
	}

	var res models.Metadata

	res.FileList = files

	if metadata.RestoreTo != nil {
		t := metadata.RestoreTo.AsTime()
		res.RestoreTo = &t
	}

	if metadata.BackupToolMetadata != nil {
		switch toolType := metadata.BackupToolMetadata.(type) {
		case *backuppb.Metadata_PbmMetadata:
			res.BackupToolData = &models.BackupToolData{PbmMetadata: &models.PbmMetadata{Name: toolType.PbmMetadata.Name}}
		default:
			// Do nothing.
		}
	}

	return &res
}
