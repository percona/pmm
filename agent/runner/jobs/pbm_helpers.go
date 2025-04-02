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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	cmdTimeout          = 60 * time.Minute
	resyncTimeout       = 5 * time.Minute
	statusCheckInterval = 5 * time.Second
	maxRestoreChecks    = 100
)

type pbmSeverity int

type describeInfo struct {
	Status   string    `json:"status"`
	Error    string    `json:"error"`
	ReplSets []replSet `json:"replsets"`
}

type replSet struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Nodes  []node `json:"nodes"`
}

type node struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

const (
	pbmFatalSeverity pbmSeverity = iota
	pbmErrorSeverity
	pbmWarningSeverity
	pbmInfoSeverity
	pbmDebugSeverity
)

func (s pbmSeverity) String() string {
	switch s {
	case pbmFatalSeverity:
		return "F"
	case pbmErrorSeverity:
		return "E"
	case pbmWarningSeverity:
		return "W"
	case pbmInfoSeverity:
		return "I"
	case pbmDebugSeverity:
		return "D"
	default:
		return ""
	}
}

type pbmLogEntry struct {
	TS         int64 `json:"ts"`
	pbmLogKeys `json:",inline"`
	Msg        string `json:"msg"`
}

func (e pbmLogEntry) String() string {
	return fmt.Sprintf("%s %s [%s/%s] [%s/%s] %s",
		time.Unix(e.TS, 0).Format(time.RFC3339), e.Severity, e.RS, e.Node, e.Event, e.ObjName, e.Msg)
}

type pbmLogKeys struct {
	Severity pbmSeverity `json:"s"`
	RS       string      `json:"rs"`
	Node     string      `json:"node"`
	Event    string      `json:"e"`
	ObjName  string      `json:"eobj"`
	OPID     string      `json:"opid,omitempty"`
}

type pbmBackup struct {
	Name    string `json:"name"`
	Storage string `json:"storage"`
}

type pbmRestore struct {
	StartedAt time.Time
	Name      string `json:"name"`
	Snapshot  string `json:"snapshot"`
	PITR      string `json:"point-in-time"`
}

type pbmSnapshot struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	RestoreTo  int64  `json:"restoreTo"`
	PbmVersion string `json:"pbmVersion"`
	Type       string `json:"type"`
	Error      string `json:"error"`
}

type pbmListRestore struct {
	Start    int    `json:"start"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Snapshot string `json:"snapshot"`
	PITR     int64  `json:"point-in-time"`
	Name     string `json:"name"`
	Error    string `json:"error"`
}

type pbmStatus struct {
	Backups struct {
		Type       string        `json:"type"`
		Path       string        `json:"path"`
		Region     string        `json:"region"`
		Snapshot   []pbmSnapshot `json:"snapshot"`
		PitrChunks struct {
			Size int `json:"size"`
		} `json:"pitrChunks"`
	} `json:"backups"`
	Cluster []struct {
		Rs    string `json:"rs"`
		Nodes []struct {
			Host  string `json:"host"`
			Agent string `json:"agent"`
			Role  string `json:"role"`
			Ok    bool   `json:"ok"`
		} `json:"nodes"`
	} `json:"cluster"`
	Pitr struct {
		Conf bool `json:"conf"`
		Run  bool `json:"run"`
	} `json:"pitr"`
	Running struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		StartTS int    `json:"startTS"`
		Status  string `json:"status"`
		OpID    string `json:"opID"`
	} `json:"running"`
}

type pbmError struct {
	Error string `json:"Error"`
}

// pbmConfigParams groups the flags/options for configuring PBM.
type pbmConfigParams struct {
	configFilePath string
	forceResync    bool
	dsn            string
}

func execPBMCommand(ctx context.Context, dsn string, to interface{}, args ...string) error {
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	args = append(args, "--out=json", "--mongodb-uri="+dsn)
	cmd := exec.CommandContext(nCtx, pbmBin, args...)

	b, err := cmd.Output()
	if err != nil {
		// try to parse pbm error message
		if len(b) != 0 {
			var pbmErr pbmError
			if e := json.Unmarshal(b, &pbmErr); e == nil {
				return errors.New(pbmErr.Error)
			}
		}
		return err
	}

	return json.Unmarshal(b, to)
}

func retrieveLogs(ctx context.Context, dsn string, event string) ([]pbmLogEntry, error) {
	var logs []pbmLogEntry

	if err := execPBMCommand(ctx, dsn, &logs, "logs", "--event="+event, "--tail=0"); err != nil {
		return nil, err
	}

	return logs, nil
}

func waitForPBMNoRunningOperations(ctx context.Context, l logrus.FieldLogger, dsn string) error {
	l.Info("Waiting for no running pbm operations.")

	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status, err := getPBMStatus(ctx, dsn)
			if err != nil {
				return err
			}
			if status.Running.Type == "" {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func isShardedCluster(ctx context.Context, dsn string) (bool, error) {
	status, err := getPBMStatus(ctx, dsn)
	if err != nil {
		return false, err
	}

	if len(status.Cluster) > 1 {
		return true, nil
	}

	return false, nil
}

func getPBMStatus(ctx context.Context, dsn string) (*pbmStatus, error) {
	var status pbmStatus
	if err := execPBMCommand(ctx, dsn, &status, "status"); err != nil {
		return nil, errors.Wrap(err, "pbm status error")
	}
	return &status, nil
}

func waitForPBMBackup(ctx context.Context, l logrus.FieldLogger, dsn string, name string) error {
	l.Infof("waiting for pbm backup: %s", name)
	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	retryCount := 500

	for {
		select {
		case <-ticker.C:
			var info describeInfo
			err := execPBMCommand(ctx, dsn, &info, "describe-backup", name)
			if err != nil {
				// for the first couple of seconds after backup process starts describe-backup command may return this error
				if (strings.HasSuffix(err.Error(), "no such file") ||
					strings.HasSuffix(err.Error(), "file is empty")) && retryCount > 0 {
					retryCount--
					continue
				}

				return errors.Wrap(err, "failed to get backup status")
			}

			switch info.Status {
			case "done":
				return nil
			case "canceled":
				return errors.New("backup was canceled")
			case "error":
				return errors.New(info.Error)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func findPITRRestore(list []pbmListRestore, restoreInfoPITRTime int64, startedAt time.Time) *pbmListRestore {
	for i := len(list) - 1; i >= 0; i-- {
		// TODO when PITR restore invoked with wrong timestamp pbm marks this restore operation as "snapshot" type.
		if list[i].Type == "snapshot" && list[i].Snapshot != "" {
			continue
		}
		// list[i].Name is a string which represents time the restore was started.
		restoreStartedAt, err := time.Parse(time.RFC3339Nano, list[i].Name)
		if err != nil {
			continue
		}
		// Because of https://jira.percona.com/browse/PBM-723 to find our restore record in the list of all records we're checking:
		// 1. We received PITR field as a response on starting process
		// 2. There is a record with the same PITR field in the list of restoring records
		// 3. Start time of this record is not before the time we asked for restoring.
		if list[i].PITR == restoreInfoPITRTime && !restoreStartedAt.Before(startedAt) {
			return &list[i]
		}
	}
	return nil
}

func findPITRRestoreName(ctx context.Context, dsn string, restoreInfo *pbmRestore) (string, error) {
	restoreInfoPITRTime, err := time.Parse("2006-01-02T15:04:05", restoreInfo.PITR)
	if err != nil {
		return "", err
	}

	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	checks := 0
	for {
		select {
		case <-ticker.C:
			checks++
			var list []pbmListRestore
			if err := execPBMCommand(ctx, dsn, &list, "list", "--restore"); err != nil {
				return "", errors.Wrapf(err, "pbm status error")
			}
			entry := findPITRRestore(list, restoreInfoPITRTime.Unix(), restoreInfo.StartedAt)
			if entry == nil {
				if checks > maxRestoreChecks {
					return "", errors.Errorf("failed to start restore")
				}
				continue
			} else {
				return entry.Name, nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func waitForPBMRestore(ctx context.Context, l logrus.FieldLogger, dsn string, restoreInfo *pbmRestore, backupType, confFile string) error {
	l.Infof("Detecting restore name")
	var name string
	var err error

	// @TODO Do like this until https://jira.percona.com/browse/PBM-723 is not done.
	if restoreInfo.PITR != "" { // TODO add more checks of PBM responses.
		name, err = findPITRRestoreName(ctx, dsn, restoreInfo)
		if err != nil {
			return err
		}
	} else {
		name = restoreInfo.Name
	}

	l.Infof("waiting for pbm restore: %s", name)

	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	maxRetryCount := 5
	for {
		select {
		case <-ticker.C:
			var info describeInfo
			if backupType == "physical" {
				err = execPBMCommand(ctx, dsn, &info, "describe-restore", name, "--config="+confFile)
			} else {
				err = execPBMCommand(ctx, dsn, &info, "describe-restore", name)
			}
			if err != nil {
				if maxRetryCount > 0 {
					maxRetryCount--
					l.Warnf("PMM failed to get backup restore status and will retry: %s", err)
					continue
				} else { //nolint:revive
					return errors.Wrap(err, "failed to get restore status")
				}
			}
			// reset maxRetryCount if we were able to successfully get the current restore status
			maxRetryCount = 5

			switch info.Status {
			case "done":
				return nil
			case "canceled":
				return errors.New("restore was canceled")
			case "error":
				return errors.New(info.Error)
			// We consider partlyDone as an error because we cannot automatically recover cluster from this status to fully working.
			case "partlyDone":
				return groupPartlyDoneErrors(info)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func pbmConfigure(ctx context.Context, l logrus.FieldLogger, params pbmConfigParams) error {
	l.Info("Configuring PBM.")
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	args := []string{
		"config",
		"--out=json",
		"--mongodb-uri=" + params.dsn,
		"--file=" + params.configFilePath,
	}

	output, err := exec.CommandContext(nCtx, pbmBin, args...).CombinedOutput() //nolint:gosec
	if err != nil {
		return errors.Wrapf(err, "pbm config error: %s", string(output))
	}

	if params.forceResync {
		args := []string{
			"config",
			"--out=json",
			"--mongodb-uri=" + params.dsn,
			"--force-resync",
		}
		output, err := exec.CommandContext(nCtx, pbmBin, args...).CombinedOutput() //nolint:gosec
		if err != nil {
			return errors.Wrapf(err, "pbm config resync error: %s", string(output))
		}
	}

	return nil
}

func writePBMConfigFile(conf *PBMConfig) (string, error) {
	tmp, err := os.CreateTemp("", "pbm-config-*.yml")
	if err != nil {
		return "", errors.Wrap(err, "failed to create pbm configuration file")
	}

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

// Serialization helpers.

// Storage represents target storage parameters.
type Storage struct {
	Type       string     `yaml:"type"`
	S3         S3         `yaml:"s3"`
	FileSystem FileSystem `yaml:"filesystem"`
}

// S3 represents S3 storage parameters.
type S3 struct {
	Region      string      `yaml:"region"`
	Bucket      string      `yaml:"bucket"`
	Prefix      string      `yaml:"prefix"`
	EndpointURL string      `yaml:"endpointUrl"`
	Credentials Credentials `yaml:"credentials"`
}

// FileSystem  represents local storage parameters.
type FileSystem struct {
	Path string `yaml:"path"`
}

// Credentials contains S3 credentials.
type Credentials struct {
	AccessKeyID     string `yaml:"access-key-id"`
	SecretAccessKey string `yaml:"secret-access-key"`
}

// PITR contains Point-in-Time recovery reature related parameters.
type PITR struct {
	Enabled bool `yaml:"enabled"`
}

// PBMConfig represents pbm configuration file.
type PBMConfig struct {
	Storage Storage `yaml:"storage"`
	PITR    PITR    `yaml:"pitr"`
}

// createPBMConfig returns object that is ready to be serialized into YAML.
func createPBMConfig(locationConfig *BackupLocationConfig, prefix string, pitr bool) (*PBMConfig, error) {
	conf := &PBMConfig{
		PITR: PITR{
			Enabled: pitr,
		},
	}

	switch locationConfig.Type {
	case S3BackupLocationType:
		conf.Storage = Storage{
			Type: "s3",
			S3: S3{
				EndpointURL: locationConfig.S3Config.Endpoint,
				Region:      locationConfig.S3Config.BucketRegion,
				Bucket:      locationConfig.S3Config.BucketName,
				Prefix:      prefix,
				Credentials: Credentials{
					AccessKeyID:     locationConfig.S3Config.AccessKey,
					SecretAccessKey: locationConfig.S3Config.SecretKey,
				},
			},
		}
	case FilesystemBackupLocationType:
		conf.Storage = Storage{
			Type: "filesystem",
			FileSystem: FileSystem{
				Path: path.Join(locationConfig.FilesystemStorageConfig.Path, prefix),
			},
		}
	default:
		return nil, errors.New("unknown location config")
	}
	return conf, nil
}

func groupPartlyDoneErrors(info describeInfo) error {
	var errMsgs []string

	for _, rs := range info.ReplSets {
		if rs.Status == "partlyDone" {
			for _, node := range rs.Nodes {
				if node.Status == "error" {
					errMsgs = append(errMsgs, fmt.Sprintf("replset: %s, node: %s, error: %s", rs.Name, node.Name, node.Error))
				}
			}
		}
	}
	return errors.New(strings.Join(errMsgs, "; "))
}

// pbmGetSnapshotTimestamp returns time the backup restores target db to.
func pbmGetSnapshotTimestamp(ctx context.Context, l logrus.FieldLogger, dsn string, backupName string) (*time.Time, error) {
	snapshots, err := getSnapshots(ctx, l, dsn)
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		if snapshot.Name == backupName {
			return pointer.ToTime(time.Unix(snapshot.RestoreTo, 0)), nil
		}
	}

	return nil, errors.Wrap(ErrNotFound, "couldn't find required snapshot")
}

// getSnapshots returns all PBM snapshots found in configured location.
func getSnapshots(ctx context.Context, l logrus.FieldLogger, dsn string) ([]pbmSnapshot, error) {
	// Sometimes PBM returns empty list of snapshots, that's why we're trying to get them several times.
	ticker := time.NewTicker(listCheckInterval)
	defer ticker.Stop()

	checks := 0
	for {
		select {
		case <-ticker.C:
			checks++
			status, err := getPBMStatus(ctx, dsn)
			if err != nil {
				return nil, err
			}

			if len(status.Backups.Snapshot) == 0 {
				l.Debugf("Attempt %d to get a list of PBM artifacts has failed.", checks)
				if checks > maxListChecks {
					return nil, errors.Wrap(ErrNotFound, "got no one snapshot")
				}
				continue
			}

			return status.Backups.Snapshot, nil

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
