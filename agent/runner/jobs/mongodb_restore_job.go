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
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/api/agentpb"
	backuppb "github.com/percona/pmm/api/managementpb/backup"
)

const (
	listCheckInterval = 1 * time.Second
	maxListChecks     = 100
)

// MongoDBRestoreJob implements Job for MongoDB restore.
type MongoDBRestoreJob struct {
	id              string
	timeout         time.Duration
	l               *logrus.Entry
	name            string
	pitrTimestamp   time.Time
	dbURL           string
	locationConfig  BackupLocationConfig
	agentsRestarter agentsRestarter
	jobLogger       *pbmJobLogger
	folder          string
	pbmBackupName   string
	compression     backuppb.BackupCompression
}

// NewMongoDBRestoreJob creates new Job for MongoDB backup restore.
func NewMongoDBRestoreJob(
	id string,
	timeout time.Duration,
	name string,
	pitrTimestamp time.Time,
	dbConfig string,
	locationConfig BackupLocationConfig,
	restarter agentsRestarter,
	folder string,
	pbmBackupName string,
	compression backuppb.BackupCompression,
) *MongoDBRestoreJob {
	return &MongoDBRestoreJob{
		id:              id,
		timeout:         timeout,
		l:               logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_restore", "name": name}),
		name:            name,
		pitrTimestamp:   pitrTimestamp,
		dbURL:           dbConfig,
		locationConfig:  locationConfig,
		agentsRestarter: restarter,
		jobLogger:       newPbmJobLogger(id, pbmRestoreJob, dbConfig),
		folder:          folder,
		pbmBackupName:   pbmBackupName,
		compression:     compression,
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

// DSN returns DSN required for the Job.
func (j *MongoDBRestoreJob) DSN() string {
	return j.dbURL
}

// Run starts Job execution.
func (j *MongoDBRestoreJob) Run(ctx context.Context, send Send) error {
	defer j.jobLogger.sendLog(send, "", true)

	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	artifactFolder := j.folder

	// Old artifacts don't contain pbm backup name.
	if j.pbmBackupName == "" {
		artifactFolder = j.name
	}

	conf, err := createPBMConfig(&j.locationConfig, artifactFolder, false)
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
		forceResync:    true,
		dsn:            j.dbURL,
	}
	if err := pbmConfigure(ctx, j.l, configParams); err != nil {
		return errors.Wrap(err, "failed to configure pbm")
	}

	rCtx, cancel := context.WithTimeout(ctx, resyncTimeout)
	if err := waitForPBMNoRunningOperations(rCtx, j.l, j.dbURL); err != nil {
		cancel()
		return errors.Wrap(err, "failed to wait pbm configuration completion")
	}
	cancel()

	snapshot, err := j.findCurrentSnapshot(ctx, j.pbmBackupName)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return errors.WithStack(err)
	}

	if snapshot.Status == "error" { //nolint:goconst
		j.jobLogger.sendLog(send, snapshot.Error, false)
		return errors.Wrap(ErrPBMArtifactProblem, snapshot.Error)
	}

	defer j.agentsRestarter.RestartAgents()
	restoreOut, err := j.startRestore(ctx, snapshot.Name)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return errors.Wrap(err, "failed to start backup restore")
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		err := j.jobLogger.streamLogs(streamCtx, send, restoreOut.Name)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			j.l.Errorf("stream logs: %v", err)
		}
	}()

	if err := waitForPBMRestore(ctx, j.l, j.dbURL, restoreOut, snapshot.Type, confFile); err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
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

func (j *MongoDBRestoreJob) findCurrentSnapshot(ctx context.Context, snapshotName string) (*pbmSnapshot, error) {
	j.l.Info("Finding backup entity name.")

	snapshots, err := getSnapshots(ctx, j.l, j.dbURL)
	if err != nil {
		return nil, err
	}

	// Old artifacts don't contain pbm backup name.
	if snapshotName == "" {
		return &snapshots[0], nil
	}

	for _, s := range snapshots {
		if s.Name == snapshotName {
			return &s, nil
		}
	}
	return nil, errors.WithStack(ErrNotFound)
}

func (j *MongoDBRestoreJob) startRestore(ctx context.Context, backupName string) (*pbmRestore, error) {
	j.l.Infof("starting backup restore for: %s.", backupName)

	var restoreOutput pbmRestore
	var err error
	startTime := time.Now()

	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()
	retryCount := 500

	for {
		select {
		case <-ticker.C:

			if j.pitrTimestamp.Unix() == 0 {
				err = execPBMCommand(ctx, j.dbURL, &restoreOutput, "restore", backupName)
			} else {
				err = execPBMCommand(ctx, j.dbURL, &restoreOutput, "restore", fmt.Sprintf(`--time=%s`, j.pitrTimestamp.Format("2006-01-02T15:04:05")))
			}

			if err != nil {
				if strings.HasSuffix(err.Error(), "another operation in progress") && retryCount > 0 {
					retryCount--
					continue
				}
				return nil, errors.Wrapf(err, "pbm restore error: %v", err)
			}

			restoreOutput.StartedAt = startTime
			return &restoreOutput, nil

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
