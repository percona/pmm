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

package client

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/percona/pmm/agent/runner/jobs"
	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/api/agentpb"
)

const pbmBin = "pbm"

func (c *Client) handlePBMSwitchRequest(ctx context.Context, req *agentpb.PBMSwitchPITRRequest, id uint32) error {
	c.l.Infof("Switching pbm Point-in-Time Recovery feature to the state enabled: %t", req.Enabled)
	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	dsn, err := templates.RenderDSN(req.Dsn, req.TextFiles, filepath.Join(c.cfg.Paths.TempDir, "pbm-switch-pitr", strconv.Itoa(int(id))))
	if err != nil {
		return errors.WithStack(err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(
		ctx,
		pbmBin,
		"config",
		"--set",
		"pitr.enabled="+strconv.FormatBool(req.Enabled),
		"--mongodb-uri="+dsn).
		CombinedOutput() // #nosec G204
	if err != nil {
		return errors.Wrapf(err, "pbm config error: %s", string(output))
	}

	return nil
}

func (c *Client) handlePBMListPitrTimeranges(ctx context.Context, req *agentpb.PBMListPitrTimerangesRequest) (*agentpb.PBMListPitrTimerangesResponse, error) {
	if _, err := exec.LookPath(pbmBin); err != nil {
		return nil, errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	dbConnCfg := jobs.DBConnConfig{
		User:     req.User,
		Password: req.Password,
		Address:  req.Address,
		Port:     int(req.Port),
		Socket:   req.Socket,
	}
	dbUrl := jobs.CreateDBURL(dbConnCfg)

	locationConfig := &jobs.BackupLocationConfig{}
	switch cfg := req.LocationConfig.(type) {
	case *agentpb.PBMListPitrTimerangesRequest_S3Config:
		locationConfig.Type = jobs.S3BackupLocationType
		locationConfig.S3Config = &jobs.S3LocationConfig{
			Endpoint:     cfg.S3Config.Endpoint,
			AccessKey:    cfg.S3Config.AccessKey,
			SecretKey:    cfg.S3Config.SecretKey,
			BucketName:   cfg.S3Config.BucketName,
			BucketRegion: cfg.S3Config.BucketRegion,
		}
	case *agentpb.PBMListPitrTimerangesRequest_FilesystemConfig:
		locationConfig.Type = jobs.FilesystemBackupLocationType
		locationConfig.FilesystemStorageConfig = &jobs.FilesystemBackupLocationConfig{
			Path: cfg.FilesystemConfig.Path,
		}
	default:
		return nil, errors.Errorf("unknown location config: %T", req.LocationConfig)
	}

	conf, err := jobs.CreatePBMConfig(locationConfig, req.BackupName, false)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	confFile, err := jobs.WritePBMConfigFile(conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer os.Remove(confFile) //nolint:errcheck

	if err := jobs.PBMConfigure(ctx, c.l, dbUrl, confFile); err != nil {
		return nil, errors.Wrap(err, "failed to configure pbm")
	}

	var status jobs.PbmStatus
	if err := jobs.ExecPBMCommand(ctx, dbUrl, &status, "status"); err != nil {
		return nil, err
	}

	timeranges := make([]*agentpb.PBMPitrTimerange, 0, len(status.Backups.PitrChunks.PitrChunks))
	for _, tr := range status.Backups.PitrChunks.PitrChunks {
		if !tr.NoBaseSnapshot && tr.Err == nil {
			timeranges = append(timeranges, &agentpb.PBMPitrTimerange{
				StartTimestamp: timestamppb.New(time.Unix(int64(tr.Range.Start), 0)),
				EndTimestamp:   timestamppb.New(time.Unix(int64(tr.Range.End), 0)),
			})
		}
	}
	return &agentpb.PBMListPitrTimerangesResponse{
		PitrTimeranges: timeranges,
	}, nil
}
