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
	"bytes"
	"context"
	"io"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	pbmBin = "pbm"

	logsCheckInterval = 3 * time.Second
	waitForLogs       = 2 * logsCheckInterval
)

// MongoDBBackupJob implements Job from MongoDB backup.
type MongoDBBackupJob struct {
	id         string
	timeout    time.Duration
	l          logrus.FieldLogger
	name       string
	dbURL      *url.URL
	location   BackupLocationConfig
	pitr       bool
	logChunkID uint32
}

// NewMongoDBBackupJob creates new Job for MongoDB backup.
func NewMongoDBBackupJob(
	id string,
	timeout time.Duration,
	name string,
	dbConfig DBConnConfig,
	locationConfig BackupLocationConfig,
	pitr bool,
) *MongoDBBackupJob {
	return &MongoDBBackupJob{
		id:       id,
		timeout:  timeout,
		l:        logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_backup", "name": name}),
		name:     name,
		dbURL:    createDBURL(dbConfig),
		location: locationConfig,
		pitr:     pitr,
	}
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
	defer j.sendLog(send, "", true)

	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	conf := &PBMConfig{
		PITR: PITR{
			Enabled: j.pitr,
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
		return errors.Wrap(err, "failed to wait configuration completion")
	}
	cancel()

	pbmBackupOut, err := j.startBackup(ctx)
	if err != nil {
		j.sendLog(send, err.Error(), false)
		return errors.Wrap(err, "failed to start backup")
	}
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	go func() {
		err := j.streamLogs(streamCtx, send, pbmBackupOut.Name)
		if err != nil && err != io.EOF && err != context.Canceled {
			j.l.Errorf("stream logs: %v", err)
		}
	}()

	if err := waitForPBMState(ctx, j.l, j.dbURL, pbmBackupFinished(pbmBackupOut.Name)); err != nil {
		j.sendLog(send, err.Error(), false)
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

func createDBURL(dbConfig DBConnConfig) *url.URL {
	var host string
	switch {
	case dbConfig.Address != "":
		if dbConfig.Port > 0 {
			host = net.JoinHostPort(dbConfig.Address, strconv.Itoa(dbConfig.Port))
		} else {
			host = dbConfig.Address
		}
	case dbConfig.Socket != "":
		host = dbConfig.Socket
	}

	var user *url.Userinfo
	if dbConfig.User != "" {
		user = url.UserPassword(dbConfig.User, dbConfig.Password)
	}

	return &url.URL{
		Scheme: "mongodb",
		User:   user,
		Host:   host,
	}
}

func (j *MongoDBBackupJob) startBackup(ctx context.Context) (*pbmBackup, error) {
	j.l.Info("Starting backup.")
	var result pbmBackup

	if err := execPBMCommand(ctx, j.dbURL, &result, "backup"); err != nil {
		return nil, err
	}

	return &result, nil
}

func (j *MongoDBBackupJob) streamLogs(ctx context.Context, send Send, name string) error {
	var (
		err    error
		logs   []pbmLogEntry
		buffer bytes.Buffer
		skip   int
	)
	j.logChunkID = 0

	ticker := time.NewTicker(logsCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logs, err = retrieveLogs(ctx, j.dbURL, "backup/"+name)
			if err != nil {
				return err
			}
			// @TODO Replace skip with proper paging after this is done https://jira.percona.com/browse/PBM-713
			logs = logs[skip:]
			skip += len(logs)
			if len(logs) == 0 {
				continue
			}
			from, to := 0, maxLogsChunkSize
			for from < len(logs) {
				if to > len(logs) {
					to = len(logs)
				}
				buffer.Reset()
				for i, log := range logs[from:to] {
					_, err := buffer.WriteString(log.String())
					if err != nil {
						return err
					}
					if i != to-from-1 {
						buffer.WriteRune('\n')
					}
				}
				j.sendLog(send, buffer.String(), false)
				from += maxLogsChunkSize
				to += maxLogsChunkSize
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (j *MongoDBBackupJob) sendLog(send Send, data string, done bool) {
	send(&agentpb.JobProgress{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result: &agentpb.JobProgress_Logs_{
			Logs: &agentpb.JobProgress_Logs{
				ChunkId: atomic.AddUint32(&j.logChunkID, 1) - 1,
				Data:    data,
				Done:    done,
			},
		},
	})
}
