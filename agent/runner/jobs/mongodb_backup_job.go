// Copyright 2019 Percona LLC
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
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/api/agentpb"
	backuppb "github.com/percona/pmm/api/managementpb/backup"
)

const (
	pbmBin = "pbm"

	logsCheckInterval = 3 * time.Second
	waitForLogs       = 2 * logsCheckInterval
)

// MongoDBBackupJob implements Job from MongoDB backup.
type MongoDBBackupJob struct {
	id             string
	timeout        time.Duration
	l              logrus.FieldLogger
	name           string
	dbURL          *url.URL
	locationConfig BackupLocationConfig
	pitr           bool
	dataModel      backuppb.DataModel
	eventLog *pbmEventLog
}

// NewMongoDBBackupJob creates new Job for MongoDB backup.
func NewMongoDBBackupJob(
	id string,
	timeout time.Duration,
	name string,
	dbConfig DBConnConfig,
	locationConfig BackupLocationConfig,
	pitr bool,
	dataModel backuppb.DataModel,
) (*MongoDBBackupJob, error) {
	if dataModel != backuppb.DataModel_PHYSICAL && dataModel != backuppb.DataModel_LOGICAL {
		return nil, errors.Errorf("'%s' is not a supported data model for MongoDB backups", dataModel)
	}
	if dataModel != backuppb.DataModel_LOGICAL && pitr {
		return nil, errors.Errorf("PITR is only supported for logical backups")
	}

	dbURL := createDBURL(dbConfig)
	return &MongoDBBackupJob{
		id:             id,
		timeout:        timeout,
		l:              logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_backup", "name": name}),
		name:           name,
		dbURL:          dbURL,
		locationConfig: locationConfig,
		pitr:           pitr,
		dataModel:      dataModel,
		eventLog: newPbmEventLog(id, pbmBackupEvent, dbURL),
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
	defer j.eventLog.sendLog(send, "", true)

	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	conf, err := createPBMConfig(&j.locationConfig, j.name, j.pitr)
	if err != nil {
		return errors.WithStack(err)
	}

	confFile, err := writePBMConfigFile(conf)
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	if err := pbmConfigure(ctx, j.l, j.dbURL, confFile); err != nil {
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
		j.eventLog.sendLog(send, err.Error(), false)
		return errors.Wrap(err, "failed to start backup")
	}
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		err := j.eventLog.streamLogs(streamCtx, send, pbmBackupEvent, pbmBackupOut.Name)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			j.l.Errorf("stream logs: %v", err)
		}
	}()

	if err := waitForPBMBackup(ctx, j.l, j.dbURL, pbmBackupOut.Name); err != nil {
		j.eventLog.sendLog(send, err.Error(), false)
		return errors.Wrap(err, "failed to wait backup completion")
	}
	send(&agentpb.JobResult{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result: &agentpb.JobResult_MongodbBackup{
			MongodbBackup: &agentpb.JobResult_MongoDBBackup{},
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
	case backuppb.DataModel_PHYSICAL:
		pbmArgs = append(pbmArgs, "--type=physical")
	case backuppb.DataModel_LOGICAL:
		pbmArgs = append(pbmArgs, "--type=logical")
	case backuppb.DataModel_DATA_MODEL_INVALID:
	default:
		return nil, errors.Errorf("'%s' is not a supported data model for backups", j.dataModel)
	}

	if err := execPBMCommand(ctx, j.dbURL, &result, pbmArgs...); err != nil {
		return nil, err
	}

	return &result, nil
}

