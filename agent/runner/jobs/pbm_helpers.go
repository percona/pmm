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
	cmdTimeout          = time.Minute
	resyncTimeout       = 5 * time.Minute
	statusCheckInterval = 5 * time.Second
)

type pbmSeverity int

type describeInfo struct {
	Status string `json:"status"`
	Error  string `json:"error"`
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
	Name       string `json:"name"`
	Status     string `json:"status"`
	RestoreTo  int64  `json:"restoreTo"`
	PbmVersion string `json:"pbmVersion"`
	Type       string `json:"type"`
}

type pbmList struct {
	Snapshots []pbmSnapshot `json:"snapshots"`
	Pitr      struct {
		On     bool        `json:"on"`
		Ranges interface{} `json:"ranges"`
	} `json:"pitr"`
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

func waitForPBMNoRunningOperations(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL) error {
	l.Info("Waiting for no running pbm operations.")

	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var status pbmStatus
			if err := execPBMCommand(ctx, dbURL, &status, "status"); err != nil {
				return errors.Wrapf(err, "pbm status error")
			}
			if status.Running.Type == "" {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func waitForPBMBackup(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL, name string) error {
	l.Infof("waiting for pbm backup: %s", name)
	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var info describeInfo
			err := execPBMCommand(ctx, dbURL, &info, "describe-backup", name)
			if err != nil {
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

func waitForPBMRestore(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL, backupType, name, confFile string) error {
	l.Infof("waiting for pbm restore: %s", name)
	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	var err error
	for {
		select {
		case <-ticker.C:
			var info describeInfo
			if backupType == "physical" {
				err = execPBMCommand(ctx, dbURL, &info, "describe-restore", "--config="+confFile, name)
			} else {
				err = execPBMCommand(ctx, dbURL, &info, "describe-restore", name)
			}
			if err != nil {
				return errors.Wrap(err, "failed to get restore status")
			}

			switch info.Status {
			case "done":
				return nil
			case "canceled":
				return errors.New("restore was canceled")
			case "error":
				return errors.New(info.Error)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func pbmConfigure(ctx context.Context, l logrus.FieldLogger, dbURL *url.URL, confFile string) error {
	l.Info("Configuring S3 location.")
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

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
