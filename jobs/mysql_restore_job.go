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
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/percona/pmm/api/agentpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	xbstreamBin      = "xbstream"
	mySQLServiceName = "mysql"
	mySQLUserName    = "mysql"
	mySQLGroupName   = "mysql"
	mySQLDirectory   = "/var/lib/mysql"
	systemctlTimeout = 10 * time.Second
)

// MySQLRestoreJob implements Job for MySQL backup restore.
type MySQLRestoreJob struct {
	id       string
	timeout  time.Duration
	l        logrus.FieldLogger
	name     string
	location BackupLocationConfig
}

// NewMySQLRestoreJob constructs new Job for MySQL backup restore.
func NewMySQLRestoreJob(id string, timeout time.Duration, name string, locationConfig BackupLocationConfig) *MySQLRestoreJob {
	return &MySQLRestoreJob{
		id:       id,
		timeout:  timeout,
		l:        logrus.WithFields(logrus.Fields{"id": id, "type": "mysql_restore"}),
		name:     name,
		location: locationConfig,
	}
}

// ID returns job id.
func (j *MySQLRestoreJob) ID() string {
	return j.id
}

// Type returns job type.
func (j *MySQLRestoreJob) Type() JobType {
	return MySQLRestore
}

// Timeout returns job timeout.
func (j *MySQLRestoreJob) Timeout() time.Duration {
	return j.timeout
}

// Run executes backup restore steps.
func (j *MySQLRestoreJob) Run(ctx context.Context, send Send) (rerr error) {
	if j.location.S3Config == nil {
		return errors.New("S3 config is not set")
	}

	if err := j.binariesInstalled(); err != nil {
		return errors.WithStack(err)
	}

	if _, _, err := mySQLUserAndGroupIDs(); err != nil {
		return errors.WithStack(err)
	}

	tmpDir, err := ioutil.TempDir("", "backup-restore")
	if err != nil {
		return errors.Wrap(err, "cannot create temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			j.l.WithError(err).Warn("failed to remove temporary directory")
		}
	}()

	if err := j.restoreMySQLFromS3(ctx, tmpDir); err != nil {
		return errors.WithStack(err)
	}

	active, err := mySQLActive(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	if active {
		if err := stopMySQL(ctx); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := restoreBackup(ctx, tmpDir, mySQLDirectory); err != nil {
		return errors.WithStack(err)
	}

	if err := startMySQL(ctx); err != nil {
		return errors.WithStack(err)
	}

	send(&agentpb.JobResult{
		JobId:     j.id,
		Timestamp: ptypes.TimestampNow(),
		Result: &agentpb.JobResult_MysqlRestoreBackup{
			MysqlRestoreBackup: &agentpb.JobResult_MySQLRestoreBackup{},
		},
	})

	return nil
}

func (j *MySQLRestoreJob) binariesInstalled() error {
	if _, err := exec.LookPath(xtrabackupBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", xtrabackupBin)
	}

	if _, err := exec.LookPath(xbcloudBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", xbcloudBin)
	}

	if _, err := exec.LookPath(xbstreamBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", xbstreamBin)
	}

	if _, err := exec.LookPath(qpressBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", qpressBin)
	}

	return nil
}

func prepareRestoreCommands(
	ctx context.Context,
	backupName string,
	config *BackupLocationConfig,
	targetDirectory string,
	stderr io.Writer,
	stdout io.Writer,
) (xbcloud, xbstream *exec.Cmd, _ error) {
	xbcloudCmd := exec.CommandContext( //nolint:gosec
		ctx,
		xbcloudBin,
		"get",
		"--storage=s3",
		"--s3-endpoint="+config.S3Config.Endpoint,
		"--s3-access-key="+config.S3Config.AccessKey,
		"--s3-secret-key="+config.S3Config.SecretKey,
		"--s3-bucket="+config.S3Config.BucketName,
		"--s3-region="+config.S3Config.BucketRegion,
		"--parallel=10",
		backupName,
	)
	xbcloudCmd.Stderr = stderr

	xbcloudStdout, err := xbcloudCmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get xbcloud stdout pipe")
	}

	xbstreamCmd := exec.CommandContext( //nolint:gosec
		ctx,
		xbstreamBin,
		"restore",
		"-x",
		"--directory="+targetDirectory,
		"--parallel=10",
	)
	xbstreamCmd.Stdin = xbcloudStdout
	xbstreamCmd.Stderr = stderr
	xbstreamCmd.Stdout = stdout

	return xbcloudCmd, xbstreamCmd, nil
}

func (j *MySQLRestoreJob) restoreMySQLFromS3(ctx context.Context, targetDirectory string) (rerr error) {
	pipeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var stderr, stdout bytes.Buffer
	xbcloudCmd, xbstreamCmd, err := prepareRestoreCommands(
		pipeCtx,
		j.name,
		&j.location,
		targetDirectory,
		&stderr,
		&stdout,
	)
	if err != nil {
		return err
	}

	wrapError := func(err error) error {
		return errors.Wrapf(err, "stderr: %s\n stdout: %s\n", stderr.String(), stdout.String())
	}

	if err := xbcloudCmd.Start(); err != nil {
		cancel()
		return errors.Wrap(wrapError(err), "xbcloud start failed")
	}
	defer func() {
		if err := xbcloudCmd.Wait(); err != nil {
			cancel()
			if rerr != nil {
				rerr = errors.Wrapf(rerr, "xbcloud wait error: %s", err)
			} else {
				rerr = errors.Wrap(wrapError(err), "xbcloud wait failed")
			}
		}
	}()

	if err := xbstreamCmd.Start(); err != nil {
		cancel()
		return errors.Wrap(wrapError(err), "xbstream start failed")
	}
	defer func() {
		if err := xbstreamCmd.Wait(); err != nil {
			cancel()
			if rerr != nil {
				rerr = errors.Wrapf(rerr, "xbstream wait error: %s", err)
			} else {
				rerr = errors.Wrap(wrapError(err), "xbstream wait failed")
			}
		}
	}()

	return nil
}

func mySQLActive(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, systemctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "--quiet", mySQLServiceName)
	if err := cmd.Start(); err != nil {
		return false, errors.Wrap(err, "starting systemctl is-active command failed")
	}

	// systemctl is-active returns an exit code 0 if service is active, or non-zero otherwise
	var exitError *exec.ExitError
	err := cmd.Wait()
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &exitError):
		return false, nil
	default:
		return false, errors.WithStack(err)
	}
}

func stopMySQL(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, systemctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "stop", mySQLServiceName)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting systemctl stop command failed")
	}

	return errors.Wrap(cmd.Wait(), "waiting systemctl stop command failed")
}

func startMySQL(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, systemctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "start", mySQLServiceName)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting systemctl start command failed")
	}

	return errors.Wrap(cmd.Wait(), "waiting systemctl start command failed")
}

func chownRecursive(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		return errors.WithStack(os.Chown(name, uid, gid))
	})
}

// mySQLUserAndGroupIDs returns uid, gid if error is nil.
func mySQLUserAndGroupIDs() (int, int, error) {
	u, err := user.Lookup(mySQLUserName)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	g, err := user.LookupGroup(mySQLGroupName)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	gid, err := strconv.Atoi(g.Gid)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	return uid, gid, nil
}

func isPathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, errors.WithStack(err)
	}
}

func restoreBackup(ctx context.Context, backupDirectory, mySQLDirectory string) error {
	if output, err := exec.CommandContext( //nolint:gosec
		ctx,
		xtrabackupBin,
		"--decompress",
		"--target-dir="+backupDirectory,
	).CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to decompress, output: %s", string(output))
	}

	if output, err := exec.CommandContext( //nolint:gosec
		ctx,
		xtrabackupBin,
		"--prepare",
		"--target-dir="+backupDirectory,
	).CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to prepare, output: %s", string(output))
	}

	exists, err := isPathExists(mySQLDirectory)
	if err != nil {
		return errors.WithStack(err)
	}
	if exists {
		postfix := ".old" + strconv.FormatInt(time.Now().Unix(), 10)
		if err := os.Rename(mySQLDirectory, mySQLDirectory+postfix); err != nil {
			return errors.WithStack(err)
		}
	}

	if output, err := exec.CommandContext( //nolint:gosec
		ctx,
		xtrabackupBin,
		"--copy-back",
		"--datadir="+mySQLDirectory,
		"--target-dir="+backupDirectory).CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to copy back, output: %s", string(output))
	}

	uid, gid, err := mySQLUserAndGroupIDs()
	if err != nil {
		return errors.WithStack(err)
	}
	if err := chownRecursive(mySQLDirectory, uid, gid); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
