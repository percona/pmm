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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/utils/poll"
	agentv1 "github.com/percona/pmm/api/agent/v1"
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

	_, err := exec.LookPath(pbmBin)
	if err != nil {
		return fmt.Errorf("lookpath=%s: %w", pbmBin, err)
	}

	artifactFolder := j.folder

	// Old artifacts don't contain pbm backup name.
	if j.pbmBackupName == "" {
		artifactFolder = j.name
	}

	conf, err := createPBMConfig(&j.locationConfig, artifactFolder, false)
	if err != nil {
		return fmt.Errorf("failed to create PBM config: %w", err)
	}

	confFile, err := writePBMConfigFile(conf)
	if err != nil {
		return fmt.Errorf("failed to write to PBM config: %w", err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	configParams := pbmConfigParams{
		configFilePath: confFile,
		forceResync:    true,
		dsn:            j.dbURL,
	}
	err = pbmConfigure(ctx, j.l, configParams)
	if err != nil {
		return fmt.Errorf("failed to configure pbm: %w", err)
	}

	rCtx, cancel := context.WithTimeout(ctx, resyncTimeout)
	err = waitForPBMNoRunningOperations(rCtx, j.l, j.dbURL)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to wait pbm configuration completion: %w", err)
	}
	cancel()

	snapshot, err := j.findCurrentSnapshot(ctx, j.pbmBackupName)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return fmt.Errorf("failed to find current snapshot: %w", err)
	}

	if snapshot.Status == "error" { //nolint:goconst
		j.jobLogger.sendLog(send, snapshot.Error, false)
		return fmt.Errorf("%s: %w", snapshot.Error, ErrPBMArtifactProblem)
	}

	defer j.agentsRestarter.RestartAgents()
	restoreOut, err := j.startRestore(ctx, snapshot.Name)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return fmt.Errorf("failed to start backup restore: %w", err)
	}

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		err := j.jobLogger.streamLogs(streamCtx, send, restoreOut.Name)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			j.l.Errorf("stream logs: %v", err)
		}
	}()

	err = waitForPBMRestore(ctx, j.l, j.dbURL, restoreOut, snapshot.Type, confFile)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return fmt.Errorf("failed to wait backup restore completion: %w", err)
	}

	send(&agentv1.JobResult{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result: &agentv1.JobResult_MongodbRestoreBackup{
			MongodbRestoreBackup: &agentv1.JobResult_MongoDBRestoreBackup{},
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
	return nil, ErrNotFound
}

func (j *MongoDBRestoreJob) startRestore(ctx context.Context, backupName string) (*pbmRestore, error) {
	j.l.Infof("starting backup restore for: %s.", backupName)

	var restoreOutput pbmRestore
	startTime := time.Now()
	retryCount := 500
	started := false

	pollErr := poll.UntilContextTimeout(ctx, statusCheckInterval, func(ctx context.Context) (bool, error) {
		// Preserve previous behavior: first restore command runs after the first tick.
		if !started {
			started = true
			return false, nil
		}

		var cmdErr error
		if j.pitrTimestamp.Unix() == 0 {
			cmdErr = execPBMCommand(ctx, j.dbURL, &restoreOutput, "restore", backupName)
		} else {
			cmdErr = execPBMCommand(ctx, j.dbURL, &restoreOutput, "restore", "--time="+j.pitrTimestamp.Format("2006-01-02T15:04:05"))
		}

		if cmdErr != nil {
			if strings.HasSuffix(cmdErr.Error(), "another operation in progress") && retryCount > 0 {
				retryCount--
				return false, nil
			}
			return false, fmt.Errorf("pbm restore error: %w", cmdErr)
		}

		restoreOutput.StartedAt = startTime
		return true, nil
	})
	if pollErr != nil {
		return nil, pollErr
	}

	return &restoreOutput, nil
}
