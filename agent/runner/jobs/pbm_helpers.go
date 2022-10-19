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
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// How many times check if backup operation was started
	maxBackupChecks = 10

	// how many times to check if a restore operation has completed.
	maxRestoreChecks = 25

	cmdTimeout          = time.Minute
	resyncTimeout       = 5 * time.Minute
	statusCheckInterval = 5 * time.Second
)

type pbmSeverity int

type restoreInfo struct {
	Name     string                 `json:"name"`
	Backup   string                 `json:"backup"`
	Type     string                 `json:"type"`
	Status   string                 `json:"status"`
	Error    map[string]interface{} `json:"error"`
	ReplSets []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"replsets"`
}

const (
	pbmFatal pbmSeverity = iota
	pbmError
	pbmWarning
	pbmInfo
	pbmDebug
)

func (s pbmSeverity) String() string {
	switch s {
	case pbmFatal:
		return "F"
	case pbmError:
		return "E"
	case pbmWarning:
		return "W"
	case pbmInfo:
		return "I"
	case pbmDebug:
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
	Name     string `json:"name"`
	Snapshot string `json:"snapshot"`
}

type pbmSnapshot struct {
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Error      json.RawMessage `json:"error"` // Temporary fix https://jira.percona.com/browse/PBM-988
	RestoreTo  int64           `json:"restoreTo"`
	PbmVersion string          `json:"pbmVersion"`
	Type       string          `json:"type"`
}

type pbmList struct {
	Snapshots []pbmSnapshot `json:"snapshots"`
	Pitr      struct {
		On     bool        `json:"on"`
		Ranges interface{} `json:"ranges"`
	} `json:"pitr"`
}

type pbmListRestore struct {
	Start    int    `json:"start"`
	Status   string `json:"status"`
	Type     string `json:"type"`
	Snapshot string `json:"snapshot"`
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

func execPBMCommand(ctx context.Context, dbURL *url.URL, to interface{}, args ...string) error {
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	args = append(args, "--out=json", "--mongodb-uri="+dbURL.String())
	cmd := exec.CommandContext(nCtx, pbmBin, args...) // #nosec G204

	b, err := cmd.Output()
	log.Println(string(b))
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return errors.New(string(exitErr.Stderr))
		}
		return err
	}

	return json.Unmarshal(b, to)
}

func retrieveLogs(ctx context.Context, dbURL *url.URL, event string) ([]pbmLogEntry, error) {
	var logs []pbmLogEntry

	if err := execPBMCommand(ctx, dbURL, &logs, "logs", "--event="+event, "--tail=0"); err != nil {
		return nil, err
	}

	return logs, nil
}

type pbmStatusCondition func(s pbmStatus) (bool, error)

func pbmNoRunningOperations(s pbmStatus) (bool, error) {
	return s.Running.Type == "", nil // for operations like storage resync, pbm might not report a status
}

func pbmBackupFinished(name string) pbmStatusCondition {
	started := false
	snapshotStarted := false
	checks := 0
	return func(s pbmStatus) (bool, error) {
		checks++
		if s.Running.Type == "backup" && s.Running.Name == name && s.Running.Status != "" {
			started = true
		}
		if !started && checks > maxBackupChecks {
			return false, errors.New("failed to start backup")
		}
		var snapshot *pbmSnapshot
		for i, snap := range s.Backups.Snapshot {
			if snap.Name == name {
				snapshot = &s.Backups.Snapshot[i]
				break
			}
		}
		if snapshot == nil {
			return false, nil
		}

		switch snapshot.Status {
		case "starting", "running", "dumpDone":
			snapshotStarted = true
			return false, nil
		case "canceled":
			return false, errors.New("backup was canceled")
		}

		if snapshotStarted && snapshot.Status == "error" {
			var errMsg string
			// Try to unmarshal error message to string variable https://jira.percona.com/browse/PBM-988
			if err := json.Unmarshal(snapshot.Error, &errMsg); err != nil {
				return false, errors.New("unknown pbm error")
			}
			return false, errors.New(errMsg)
		}

		return snapshot.Status == "done", nil
	}
}

func waitForPBMState(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL, cond pbmStatusCondition) error {
	l.Info("Waiting for pbm state condition.")

	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var status pbmStatus
			if err := execPBMCommand(ctx, dbURL, &status, "status"); err != nil {
				return errors.Wrapf(err, "pbm status error")
			}
			done, err := cond(status)
			if err != nil {
				return errors.Wrapf(err, "condition failed")
			}
			if done {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func waitForPBMRestore(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL, backupType, name string, conf *PBMConfig) error {
	l.Infof("waiting for pbm restore: %s", name)
	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()
	checks := 0

	confFile, err := writePBMConfigFile(conf)
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	var ri restoreInfo
	for {
		select {
		case <-ticker.C:
			checks++
			if backupType == "physical" {
				err = execPBMCommand(ctx, dbURL, &ri, "describe-restore", "--config="+confFile, name)
			} else {
				err = execPBMCommand(ctx, dbURL, &ri, "describe-restore", name)
			}
			if err != nil {
				return errors.Wrap(err, "failed to get restore status")
			}

			switch ri.Status {
			case "done", "canceled":
				return nil
			case "error":
				return errors.New(fmt.Sprintf("%+v", ri.Error))
			}

			if checks > maxRestoreChecks {
				return errors.Errorf("max restore checks attempt exceeded for restore: %")
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func pbmConfigure(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL, conf *PBMConfig) error {
	l.Info("Configuring S3 location.")
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	confFile, err := writePBMConfigFile(conf)
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	output, err := exec.CommandContext( //nolint:gosec
		nCtx,
		pbmBin,
		"config",
		"--mongodb-uri="+dbURL.String(),
		"--file="+confFile).CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "pbm config error: %s", string(output))
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

// Serialization helpers

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
	case PMMClientBackupLocationType:
		conf.Storage = Storage{
			Type: "filesystem",
			FileSystem: FileSystem{
				Path: path.Join(locationConfig.LocalStorageConfig.Path, prefix),
			},
		}
	default:
		return nil, errors.New("unknown location config")
	}
	return conf, nil
}
