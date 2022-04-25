// pmm-agent
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
	"net/url"
	"os/exec"
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MongoDBRestoreJob implements Job for MongoDB restore.
type MongoDBRestoreJob struct {
	id        string
	timeout   time.Duration
	l         *logrus.Entry
	name      string
	timestamp *time.Time
	dbURL     *url.URL
	location  BackupLocationConfig
}

// NewMongoDBRestoreJob creates new Job for MongoDB backup restore.
func NewMongoDBRestoreJob(id string, timeout time.Duration, name string, dbConfig DBConnConfig, locationConfig BackupLocationConfig) *MongoDBRestoreJob {
	return &MongoDBRestoreJob{
		id:       id,
		timeout:  timeout,
		l:        logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_restore", "name": name}),
		name:     name,
		dbURL:    createDBURL(dbConfig),
		location: locationConfig,
	}
}

// ID returns Job id.
func (j *MongoDBRestoreJob) ID() string {
	return j.id
}

// Type returns Job type.
func (j *MongoDBRestoreJob) Type() JobType {
	return MongoDBRestore
}

// Timeout returns Job timeout.
func (j *MongoDBRestoreJob) Timeout() time.Duration {
	return j.timeout
}

// Run starts Job execution.
func (j *MongoDBRestoreJob) Run(ctx context.Context, send Send) error {
	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	conf := &PBMConfig{
		PITR: PITR{
			Enabled: false,
		},
	}
	switch {
	case j.location.S3Config != nil:
		conf.Storage = Storage{
			Type: "s3",
			S3: S3{
				EndpointURL: j.location.S3Config.Endpoint,
				Region:      j.location.S3Config.BucketRegion,
				Bucket:      j.location.S3Config.BucketName,
				Prefix:      j.name,
				Credentials: Credentials{
					AccessKeyID:     j.location.S3Config.AccessKey,
					SecretAccessKey: j.location.S3Config.SecretKey,
				},
			},
		}
	default:
		return errors.New("unknown location config")
	}

	if err := pbmConfigure(ctx, j.l, j.dbURL, conf); err != nil {
		return errors.Wrap(err, "failed to configure pbm")
	}

	rCtx, cancel := context.WithTimeout(ctx, resyncTimeout)
	if err := waitForPBMState(rCtx, j.l, j.dbURL, pbmNoRunningOperations); err != nil {
		cancel()
		return errors.Wrap(err, "failed to wait pbm configuration completion")
	}
	cancel()

	backupName, err := j.findSnapshotName(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	restoreOut, err := j.startRestore(ctx, backupName)
	if err != nil {
		return errors.Wrap(err, "failed to start backup restore")
	}

	if err := waitForPBMRestore(ctx, j.l, j.dbURL, restoreOut.Snapshot); err != nil {
		return errors.Wrap(err, "failed to wait backup restore completion")
	}

	send(&agentpb.JobResult{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result: &agentpb.JobResult_MongodbRestoreBackup{
			MongodbRestoreBackup: &agentpb.JobResult_MongoDBRestoreBackup{},
		},
	})

	return nil
}

func (j *MongoDBRestoreJob) findSnapshotName(ctx context.Context) (string, error) {
	j.l.Info("Finding backup entity name.")

	var list pbmList
	if err := execPBMCommand(ctx, j.dbURL, &list, "list"); err != nil {
		return "", err
	}

	if len(list.Snapshots) == 0 {
		return "", errors.New("failed to find backup entity")
	}

	return list.Snapshots[len(list.Snapshots)-1].Name, nil
}

func (j *MongoDBRestoreJob) startRestore(ctx context.Context, backupName string) (*pbmRestore, error) {
	j.l.Info("Starting backup restore.")

	var restoreOutput pbmRestore
	err := execPBMCommand(ctx, j.dbURL, &restoreOutput, "restore", backupName)
	if err != nil {
		return nil, errors.Wrapf(err, "pbm restore error: %v", err)
	}

	return &restoreOutput, nil
}
