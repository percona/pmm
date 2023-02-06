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
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/api/agentpb"
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
	dbURL           *url.URL
	locationConfig  BackupLocationConfig
	agentsRestarter agentsRestarter
	jobLogger       *pbmJobLogger
}

// NewMongoDBRestoreJob creates new Job for MongoDB backup restore.
func NewMongoDBRestoreJob(
	id string,
	timeout time.Duration,
	name string,
	pitrTimestamp time.Time,
	dbConfig DBConnConfig,
	locationConfig BackupLocationConfig,
	restarter agentsRestarter,
) *MongoDBRestoreJob {
	dbURL := createDBURL(dbConfig)
	return &MongoDBRestoreJob{
		id:              id,
		timeout:         timeout,
		l:               logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_restore", "name": name}),
		name:            name,
		pitrTimestamp:   pitrTimestamp,
		dbURL:           dbURL,
		locationConfig:  locationConfig,
		agentsRestarter: restarter,
		jobLogger:       newPbmJobLogger(id, pbmRestoreJob, dbURL),
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
	defer j.jobLogger.sendLog(send, "", true)

	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	conf, err := createPBMConfig(&j.locationConfig, j.name, false)
	if err != nil {
		return errors.WithStack(err)
	}

	confFile, err := writePBMConfigFile(conf)
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	forceResync := conf.Storage.FileSystem.Path != ""
	if err := pbmConfigure(ctx, j.l, j.dbURL, forceResync, confFile); err != nil {
		return errors.Wrap(err, "failed to configure pbm")
	}

	rCtx, cancel := context.WithTimeout(ctx, resyncTimeout)
	if err := waitForPBMNoRunningOperations(rCtx, j.l, j.dbURL); err != nil {
		cancel()
		return errors.Wrap(err, "failed to wait pbm configuration completion")
	}
	cancel()

	snapshot, err := j.findSnapshot(ctx)
	if err != nil {
		j.jobLogger.sendLog(send, err.Error(), false)
		return errors.WithStack(err)
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

	select {
	case <-ctx.Done():
	case <-time.After(waitForLogs):
	}
	return nil
}

func (j *MongoDBRestoreJob) findSnapshot(ctx context.Context) (*pbmSnapshot, error) {
	j.l.Info("Finding backup entity name.")

	var list pbmList
	ticker := time.NewTicker(listCheckInterval)
	defer ticker.Stop()

	checks := 0
	for {
		select {
		case <-ticker.C:
			checks++
			if err := execPBMCommand(ctx, j.dbURL, &list, "list"); err != nil {
				return nil, err
			}

			if len(list.Snapshots) == 0 {
				j.l.Debugf("Try number %d of getting list of artifacts from PBM is failed.", checks)
				if checks > maxListChecks {
					return nil, errors.New("failed to find backup entity")
				}
				continue
			}

			return &list.Snapshots[len(list.Snapshots)-1], nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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
