// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobs

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
	backuppb "github.com/percona/pmm/api/backup/v1"
)

const (
	pbmBin = "pbm"

	logsCheckInterval = 3 * time.Second
	waitForLogs       = 2 * logsCheckInterval

	pbmArtifactJSONPostfix = ".pbm.json"
)

// MongoDBBackupJob implements Job from MongoDB backup.
type MongoDBBackupJob struct {
	id             string
	timeout        time.Duration
	l              logrus.FieldLogger
	name           string
	dbURL          *string
	locationConfig BackupLocationConfig
	pitr           bool
	dataModel      backuppb.DataModel
	jobLogger      *pbmJobLogger
	folder         string
}

// NewMongoDBBackupJob creates new Job for MongoDB backup.
func NewMongoDBBackupJob(
	id string,
	timeout time.Duration,
	name string,
	dbConfig *string,
	locationConfig BackupLocationConfig,
	pitr bool,
	dataModel backuppb.DataModel,
	folder string,
) (*MongoDBBackupJob, error) {
	if dataModel != backuppb.DataModel_DATA_MODEL_PHYSICAL && dataModel != backuppb.DataModel_DATA_MODEL_LOGICAL {
		return nil, errors.Errorf("'%s' is not a supported data model for MongoDB backups", dataModel)
	}
	if dataModel != backuppb.DataModel_DATA_MODEL_LOGICAL && pitr {
		return nil, errors.Errorf("PITR is only supported for logical backups")
	}

	return &MongoDBBackupJob{
		id:             id,
		timeout:        timeout,
		l:              logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_backup", "name": name}),
		name:           name,
		dbURL:          dbConfig,
		locationConfig: locationConfig,
		pitr:           pitr,
		dataModel:      dataModel,
		jobLogger:      newPbmJobLogger(id, pbmBackupJob, dbConfig),
		folder:         folder,
	}, nil
}

// ID returns Job id.
func (j *MongoDBBackupJob) ID() string {
	return j.id
}

// Type returns Job type.
func (j *MongoDBBackupJob) Type() JobType {
	return MongoDBBackup
}

// Timeout returns Job timeout.
func (j *MongoDBBackupJob) Timeout() time.Duration {
	return j.timeout
}

// Run starts Job execution.
func (j *MongoDBBackupJob) Run(ctx context.Context, send Send) error {
	defer j.jobLogger.sendLog(send, "", true)

	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	conf, err := createPBMConfig(&j.locationConfig, j.folder, j.pitr)
	if err != nil {
		return errors.WithStack(err)
	}

	confFile, err := writePBMConfigFile(conf)
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	configParams := pbmConfigParams{
		configFilePath: confFile,
		forceResync:    false,
		dbURL:          j.dbURL,
	}
	if err := pbmConfigure(ctx, j.l, configParams); err != nil {
		return errors.Wrap(err, "failed to configure pbm")
	}

	rCtx, cancel := context.WithTimeout(ctx, resyncTimeout)
	if err := waitForPBMNoRunningOperations(rCtx, j.l, j.dbURL); err != nil {
		cancel()
		return errors.Wrap(err, "failed to wait configuration completion")
	}
	cancel()

	pbmBackupOut, err := j.startBackup(ctx)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return errors.Wrap(err, "failed to start backup")
	}
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		err := j.jobLogger.streamLogs(streamCtx, send, pbmBackupOut.Name)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			j.l.Errorf("stream logs: %v", err)
		}
	}()

	if err := waitForPBMBackup(ctx, j.l, j.dbURL, pbmBackupOut.Name); err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return errors.Wrap(err, "failed to wait backup completion")
	}

	sharded, err := isShardedCluster(ctx, j.dbURL)
	if err != nil {
		return err
	}

	backupTimestamp, err := pbmGetSnapshotTimestamp(ctx, j.l, j.dbURL, pbmBackupOut.Name)
	if err != nil {
		return err
	}

	// mongoArtifactFiles returns list of files and folders the backup consists of (hardcoded).
	mongoArtifactFiles := func(pbmBackupName string) []*backuppb.File {
		res := []*backuppb.File{
			{Name: pbmBackupName + pbmArtifactJSONPostfix},
			{Name: pbmBackupName, IsDirectory: true},
		}
		return res
	}

	send(&agentv1.JobResult{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result: &agentv1.JobResult_MongodbBackup{
			MongodbBackup: &agentv1.JobResult_MongoDBBackup{
				IsShardedCluster: sharded,
				Metadata: &backuppb.Metadata{
					FileList:  mongoArtifactFiles(pbmBackupOut.Name),
					RestoreTo: timestamppb.New(*backupTimestamp),
					BackupToolMetadata: &backuppb.Metadata_PbmMetadata{
						PbmMetadata: &backuppb.PbmMetadata{Name: pbmBackupOut.Name},
					},
				},
			},
		},
	})

	select {
	case <-ctx.Done():
	case <-time.After(waitForLogs):
	}
	return nil
}

func (j *MongoDBBackupJob) startBackup(ctx context.Context) (*pbmBackup, error) {
	j.l.Info("Starting backup.")
	var result pbmBackup

	pbmArgs := []string{"backup"}
	switch j.dataModel {
	case backuppb.DataModel_DATA_MODEL_PHYSICAL:
		pbmArgs = append(pbmArgs, "--type=physical")
	case backuppb.DataModel_DATA_MODEL_LOGICAL:
		pbmArgs = append(pbmArgs, "--type=logical")
	case backuppb.DataModel_DATA_MODEL_UNSPECIFIED:
	default:
		return nil, errors.Errorf("'%s' is not a supported data model for backups", j.dataModel)
	}

	if err := execPBMCommand(ctx, j.dbURL, &result, pbmArgs...); err != nil {
		return nil, err
	}

	return &result, nil
}
