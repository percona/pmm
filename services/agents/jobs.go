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

// Package agents provides jobs functionality.
package agents

import (
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm-managed/models"
)

// JobsService provides methods for managing jobs.
type JobsService struct {
	r  *Registry
	db *reform.DB
}

// NewJobsService returns new jobs service.
func NewJobsService(db *reform.DB, registry *Registry) *JobsService {
	return &JobsService{
		r:  registry,
		db: db,
	}
}

// StartEchoJob starts echo job on the pmm-agent.
func (s *JobsService) StartEchoJob(jobID, pmmAgentID string, timeout time.Duration, message string, delay time.Duration) error {
	req := &agentpb.StartJobRequest{
		JobId:   jobID,
		Timeout: durationpb.New(timeout),
		Job: &agentpb.StartJobRequest_Echo_{
			Echo: &agentpb.StartJobRequest_Echo{
				Message: message,
				Delay:   durationpb.New(delay),
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
		return errors.Errorf("failed to start echo job: %s", e)
	}

	return nil
}

// StartMySQLBackupJob starts mysql backup job on the pmm-agent.
func (s *JobsService) StartMySQLBackupJob(jobID, pmmAgentID string, timeout time.Duration, name string, dbConfig *models.DBConfig, locationConfig *models.BackupLocationConfig) error {
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
	locationConfig *models.BackupLocationConfig,
) error {
	mongoDBReq := &agentpb.StartJobRequest_MongoDBBackup{
		Name:     name,
		User:     dbConfig.User,
		Password: dbConfig.Password,
		Address:  dbConfig.Address,
		Port:     int32(dbConfig.Port),
		Socket:   dbConfig.Socket,
	}

	switch {
	case locationConfig.S3Config != nil:
		mongoDBReq.LocationConfig = &agentpb.StartJobRequest_MongoDBBackup_S3Config{
			S3Config: convertS3ConfigModel(locationConfig.S3Config),
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
	jobResult, err := models.FindJobResultByID(s.db.Querier, jobID)
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
