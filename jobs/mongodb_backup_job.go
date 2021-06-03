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
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	pbmBin = "pbm"

	cmdTimeout          = time.Minute
	resyncTimeout       = 5 * time.Minute
	statusCheckInterval = 5 * time.Second
)

// This regexp checks that there is no running pbm operations.
var noRunningOperationsR = regexp.MustCompile(`Currently running:\n=*\n\(none\)`)

// MongoDBBackupJob implements Job from MongoDB backup.
type MongoDBBackupJob struct {
	id       string
	timeout  time.Duration
	l        logrus.FieldLogger
	name     string
	dbURL    *url.URL
	location BackupLocationConfig
}

// NewMongoDBBackupJob creates new Job for MongoDB backup.
func NewMongoDBBackupJob(id string, timeout time.Duration, name string, dbConfig DBConnConfig, locationConfig BackupLocationConfig) *MongoDBBackupJob {
	return &MongoDBBackupJob{
		id:       id,
		timeout:  timeout,
		l:        logrus.WithFields(logrus.Fields{"id": id, "type": "mongodb_backup", "name": name}),
		name:     name,
		dbURL:    createDBURL(dbConfig),
		location: locationConfig,
	}
}

// ID returns Job id.
func (j *MongoDBBackupJob) ID() string {
	return j.id
}

// Type returns Job type.
func (j *MongoDBBackupJob) Type() string {
	return "mongodb_backup"
}

// Timeout returns Job timeout.
func (j *MongoDBBackupJob) Timeout() time.Duration {
	return j.timeout
}

// Run starts Job execution.
func (j *MongoDBBackupJob) Run(ctx context.Context, send Send) error {
	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	switch {
	case j.location.S3Config != nil:
		if err := j.setupS3(ctx); err != nil {
			return errors.Wrap(err, "failed to setup S3 location")
		}
	default:
		return errors.New("unknown location config")
	}

	rCtx, cancel := context.WithTimeout(ctx, resyncTimeout)
	if err := j.waitForNoRunningOperations(rCtx); err != nil {
		cancel()
		return errors.Wrap(err, "failed to wait pbm resync completion")
	}
	cancel()

	if err := j.startBackup(ctx); err != nil {
		return errors.Wrap(err, "failed to start backup")
	}

	if err := j.waitForNoRunningOperations(ctx); err != nil {
		return errors.Wrap(err, "failed to wait backup completion")
	}

	send(&agentpb.JobResult{
		JobId:     j.id,
		Timestamp: ptypes.TimestampNow(),
		Result: &agentpb.JobResult_MongodbBackup{
			MongodbBackup: &agentpb.JobResult_MongoDBBackup{},
		},
	})

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

func (j *MongoDBBackupJob) startBackup(ctx context.Context) error {
	j.l.Info("Starting backup.")

	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	output, err := exec.CommandContext(nCtx, pbmBin, "backup", "--mongodb-uri="+j.dbURL.String()).CombinedOutput() // #nosec G204

	if err != nil {
		return errors.Wrapf(err, "pbm backup error: %s", string(output))
	}

	return nil
}

func (j *MongoDBBackupJob) checkBackupCompletion(ctx context.Context) (bool, error) {
	j.l.Debug("Checking backup status.")

	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	output, err := exec.CommandContext(nCtx, pbmBin, "status", "--mongodb-uri="+j.dbURL.String()).CombinedOutput() // #nosec G204
	if err != nil {
		return false, errors.Wrapf(err, "pbm status error: %s", string(output))
	}

	return noRunningOperationsR.Match(output), nil
}

func (j *MongoDBBackupJob) waitForNoRunningOperations(ctx context.Context) error {
	j.l.Info("Waiting for operations completion.")
	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			done, err := j.checkBackupCompletion(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to check backup completion")
			}

			if done {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (j *MongoDBBackupJob) setupS3(ctx context.Context) error {
	j.l.Info("Configuring S3 location.")
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	confFile, err := j.writePBMConfigFile()
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	output, err := exec.CommandContext( //nolint:gosec
		nCtx,
		pbmBin,
		"config",
		"--mongodb-uri="+j.dbURL.String(),
		"--file="+confFile,
	).CombinedOutput()

	if err != nil {
		return errors.Wrapf(err, "pbm config error: %s", string(output))
	}

	return nil
}

func (j *MongoDBBackupJob) writePBMConfigFile() (string, error) {
	tmp, err := ioutil.TempFile("", "pbm-config-*.yml")
	if err != nil {
		return "", errors.Wrap(err, "failed to create pbm configuration file")
	}

	var conf struct {
		Storage struct {
			Type string `yaml:"type"`
			S3   struct {
				Region      string `yaml:"region"`
				Bucket      string `yaml:"bucket"`
				Prefix      string `yaml:"prefix"`
				EndpointURL string `yaml:"endpointUrl"`
				Credentials struct {
					AccessKeyID     string `yaml:"access-key-id"`
					SecretAccessKey string `yaml:"secret-access-key"`
				}
			} `yaml:"s3"`
		} `yaml:"storage"`
	}

	conf.Storage.Type = "s3"
	conf.Storage.S3.EndpointURL = j.location.S3Config.Endpoint
	conf.Storage.S3.Region = j.location.S3Config.BucketRegion
	conf.Storage.S3.Bucket = j.location.S3Config.BucketName
	conf.Storage.S3.Prefix = j.name
	conf.Storage.S3.Credentials.AccessKeyID = j.location.S3Config.AccessKey
	conf.Storage.S3.Credentials.SecretAccessKey = j.location.S3Config.SecretKey

	bytes, err := yaml.Marshal(&conf)
	if err != nil {
		tmp.Close() //nolint:errcheck
		return "", errors.Wrap(err, "failed to marshall pbm configuration")
	}

	if _, err := tmp.Write(bytes); err != nil {
		tmp.Close() //nolint:errcheck
		return "", errors.Wrap(err, "failed to write pbm configuration file")
	}

	return tmp.Name(), tmp.Close()
}
