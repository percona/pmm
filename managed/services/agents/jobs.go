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

// Package agents provides jobs functionality.
package agents

import (
	"context"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/managed/models"
)

var (
	// ErrRetriesExhausted is returned when remaining retries are 0.
	ErrRetriesExhausted = errors.New("retries exhausted")

	pmmAgentMinVersionForMongoDBBackupAndRestore         = version.Must(version.NewVersion("2.19"))
	pmmAgentMinVersionForMySQLBackupAndRestore           = version.Must(version.NewVersion("2.23"))
	pmmAgentMinVersionForMongoDBUsePMMClientLocalStorage = version.Must(version.NewVersion("2.30.0-0"))
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
			PMMServerConfig: locationModel.PMMServerConfig,
			PMMClientConfig: locationModel.PMMClientConfig,
			S3Config:        locationModel.S3Config,
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
		if err := s.StartMySQLBackupJob(job.ID, job.PMMAgentID, job.Timeout, artifact.Name, dbConfig, locationConfig); err != nil {
			return errors.WithStack(err)
		}
	case models.MongoDBBackupJob:
		if err := s.StartMongoDBBackupJob(job.ID, job.PMMAgentID, job.Timeout, artifact.Name, dbConfig,
			job.Data.MongoDBBackup.Mode, locationConfig); err != nil {
			return errors.WithStack(err)
		}
	case models.MySQLRestoreBackupJob:
	case models.MongoDBRestoreBackupJob:
	}

	return nil
}

func (s *JobsService) handleJobResult(ctx context.Context, l *logrus.Entry, result *agentpb.JobResult) {
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

			artifact, err := models.UpdateArtifact(t.Querier, job.Data.MySQLBackup.ArtifactID, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.SuccessBackupStatus),
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

			artifact, err := models.UpdateArtifact(t.Querier, job.Data.MongoDBBackup.ArtifactID, models.UpdateArtifactParams{
				Status: models.BackupStatusPointer(models.SuccessBackupStatus),
			})
			if err != nil {
				return err
			}

			if artifact.Type == models.ScheduledArtifactType {
				scheduleID = artifact.ScheduleID
			}
		case *agentpb.JobResult_MysqlRestoreBackup:
			if job.Type != models.MySQLRestoreBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MySQLRestoreBackupJob, job.Type)
			}

			_, err := models.ChangeRestoreHistoryItem(
				t.Querier,
				job.Data.MySQLRestoreBackup.RestoreID,
				models.ChangeRestoreHistoryItemParams{
					Status: models.SuccessRestoreStatus,
				})
			if err != nil {
				return err
			}

		case *agentpb.JobResult_MongodbRestoreBackup:
			if job.Type != models.MongoDBRestoreBackupJob {
				return errors.Errorf("result type %s doesn't match job type %s", models.MongoDBRestoreBackupJob, job.Type)
			}

			_, err := models.ChangeRestoreHistoryItem(
				t.Querier,
				job.Data.MongoDBRestoreBackup.RestoreID,
				models.ChangeRestoreHistoryItemParams{
					Status: models.SuccessRestoreStatus,
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
			if err := s.retentionService.EnforceRetention(context.Background(), scheduleID); err != nil {
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
			Status: models.BackupStatusPointer(models.ErrorBackupStatus),
		})
	case models.MongoDBBackupJob:
		_, err = models.UpdateArtifact(s.db.Querier, job.Data.MongoDBBackup.ArtifactID, models.UpdateArtifactParams{
			Status: models.BackupStatusPointer(models.ErrorBackupStatus),
		})
	case models.MySQLRestoreBackupJob:
		_, err = models.ChangeRestoreHistoryItem(
			s.db.Querier,
			job.Data.MySQLRestoreBackup.RestoreID,
			models.ChangeRestoreHistoryItemParams{
				Status: models.ErrorRestoreStatus,
			})
	case models.MongoDBRestoreBackupJob:
		_, err = models.ChangeRestoreHistoryItem(
			s.db.Querier,
			job.Data.MongoDBRestoreBackup.RestoreID,
			models.ChangeRestoreHistoryItemParams{
				Status: models.ErrorRestoreStatus,
			})
	default:
		return errors.Errorf("unknown job type %s", job.Type)
	}

	go func() {
		restartCtx, cancel := context.WithTimeout(context.Background(), maxRestartInterval)
		defer cancel()
		restartErr := s.RestartJob(restartCtx, job.ID)
		if restartErr != nil && restartErr != ErrRetriesExhausted {
			s.l.Errorf("restart job %s: %v", job.ID, restartErr)
		}
	}()

	return err
}

func (s *JobsService) handleJobProgress(ctx context.Context, progress *agentpb.JobProgress) {
	switch result := progress.Result.(type) {
	case *agentpb.JobProgress_Logs_:
		_, err := models.CreateJobLog(s.db.Querier, models.CreateJobLogParams{
			JobID:     progress.JobId,
			ChunkID:   int(result.Logs.ChunkId),
			Data:      result.Logs.Data,
			LastChunk: result.Logs.Done,
		})
		if err != nil {
			s.l.WithError(err).Errorf("failed to create log for job %s [chunk: %d]", progress.JobId, result.Logs.ChunkId)
		}
	default:
		s.l.Errorf("unexpected job progress type: %T", result)
	}
}

// StartMySQLBackupJob starts mysql backup job on the pmm-agent.
func (s *JobsService) StartMySQLBackupJob(jobID, pmmAgentID string, timeout time.Duration, name string, dbConfig *models.DBConfig, locationConfig *models.BackupLocationConfig) error {
	if err := PMMAgentSupportedByAgentID(s.r.db.Querier, pmmAgentID,
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
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" {
		return errors.Errorf("failed to start MySQL job: %s", e)
	}

	return nil
}

// StartMongoDBBackupJob starts mongoDB backup job on the pmm-agent.
func (s *JobsService) StartMongoDBBackupJob(
	jobID string,
	pmmAgentID string,
	timeout time.Duration,
	name string,
	dbConfig *models.DBConfig,
	mode models.BackupMode,
	locationConfig *models.BackupLocationConfig,
) error {
	if err := PMMAgentSupportedByAgentID(s.r.db.Querier, pmmAgentID,
		"mongodb backup", pmmAgentMinVersionForMongoDBBackupAndRestore); err != nil {
		return err
	}

	mongoDBReq := &agentpb.StartJobRequest_MongoDBBackup{
		Name:       name,
		User:       dbConfig.User,
		Password:   dbConfig.Password,
		Address:    dbConfig.Address,
		Port:       int32(dbConfig.Port),
		Socket:     dbConfig.Socket,
		EnablePitr: mode == models.PITR,
	}

	switch {
	case locationConfig.S3Config != nil:
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBBackup_S3Config{
			S3Config: convertS3ConfigModel(locationConfig.S3Config),
		}
	case locationConfig.PMMClientConfig != nil:
		if err := PMMAgentSupportedByAgentID(s.r.db.Querier, pmmAgentID,
			"mongodb backup to client local storage",
			pmmAgentMinVersionForMongoDBUsePMMClientLocalStorage); err != nil {
			return err
		}
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBBackup_PmmClientConfig{
			PmmClientConfig: &agentpb.PMMClientLocationConfig{Path: locationConfig.PMMClientConfig.Path},
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	resp, err := agent.channel.SendAndWaitResponse(req)
	if err != nil {
		return err
	}
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" {
		return errors.Errorf("failed to start MongoDB job: %s", e)
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
) error {
	if err := PMMAgentSupportedByAgentID(s.r.db.Querier, pmmAgentID,
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
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" {
		return errors.Errorf("failed to start MySQL restore backup job: %s", e)
	}

	return nil
}

// StartMongoDBRestoreBackupJob starts mongo restore backup job on the pmm-agent.
func (s *JobsService) StartMongoDBRestoreBackupJob(
	jobID string,
	pmmAgentID string,
	timeout time.Duration,
	name string,
	dbConfig *models.DBConfig,
	locationConfig *models.BackupLocationConfig,
) error {
	if err := PMMAgentSupportedByAgentID(s.r.db.Querier, pmmAgentID,
		"mongodb restore", pmmAgentMinVersionForMongoDBBackupAndRestore); err != nil {
		return err
	}

	mongoDBReq := &agentpb.StartJobRequest_MongoDBRestoreBackup{
		Name:     name,
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
	case locationConfig.PMMClientConfig != nil:
		if err := PMMAgentSupportedByAgentID(s.r.db.Querier, pmmAgentID,
			"mongodb restore from client local storage",
			pmmAgentMinVersionForMongoDBUsePMMClientLocalStorage); err != nil {
			return err
		}
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBRestoreBackup_PmmClientConfig{
			PmmClientConfig: &agentpb.PMMClientLocationConfig{Path: locationConfig.PMMClientConfig.Path},
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

	agent, err := s.r.get(pmmAgentID)
	if err != nil {
		return err
	}

	resp, err := agent.channel.SendAndWaitResponse(req)
	if err != nil {
		return err
	}
	if e := resp.(*agentpb.StartJobResponse).Error; e != "" {
		return errors.Errorf("failed to start MonogDB restore backup job: %s", e)
	}

	return nil
}

// StopJob stops job with given given id.
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
