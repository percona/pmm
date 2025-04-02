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
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/percona/pmm/api/agent/v1"
)

const (
	xbstreamBin          = "xbstream"
	mySQLSystemUserName  = "mysql"
	mySQLSystemGroupName = "mysql"
	// TODO make mySQLDirectory autorecognized as done in 'xtrabackup' utility; see 'xtrabackup --help' --datadir parameter.
	mySQLDirectory   = "/var/lib/mysql"
	systemctlTimeout = 10 * time.Second
)

var mysqlServiceRegex = regexp.MustCompile(`mysql(d)?\.service`) // this is used to lookup MySQL service in the list of all system services

// MySQLRestoreJob implements Job for MySQL backup restore.
type MySQLRestoreJob struct {
	id             string
	timeout        time.Duration
	l              logrus.FieldLogger
	name           string
	locationConfig BackupLocationConfig
	folder         string
}

// NewMySQLRestoreJob constructs new Job for MySQL backup restore.
func NewMySQLRestoreJob(id string, timeout time.Duration, name string, locationConfig BackupLocationConfig, folder string) *MySQLRestoreJob {
	return &MySQLRestoreJob{
		id:             id,
		timeout:        timeout,
		l:              logrus.WithFields(logrus.Fields{"id": id, "type": "mysql_restore"}),
		name:           name,
		locationConfig: locationConfig,
		folder:         folder,
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

// DSN returns DSN for the Job.
func (j *MySQLRestoreJob) DSN() string {
	return "" // not used for MySQL restore
}

// Run executes backup restore steps.
func (j *MySQLRestoreJob) Run(ctx context.Context, send Send) error {
	if j.locationConfig.S3Config == nil {
		return errors.New("S3 config is not set")
	}

	if err := j.binariesInstalled(); err != nil {
		return errors.WithStack(err)
	}

	if _, _, err := mySQLUserAndGroupIDs(); err != nil {
		return errors.WithStack(err)
	}

	tmpDir, err := os.MkdirTemp("", "backup-restore")
	if err != nil {
		return errors.Wrap(err, "cannot create temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			j.l.WithError(err).Warn("failed to remove temporary directory")
		}
	}()

	mySQLServiceName, err := getMysqlServiceName(ctx)
	if err != nil {
		return errors.WithStack(err)
	}
	j.l.Debugf("Using MySQL service name: %s", mySQLServiceName)

	if err := j.restoreMySQLFromS3(ctx, tmpDir); err != nil {
		return errors.WithStack(err)
	}

	active, err := mySQLActive(ctx, mySQLServiceName)
	if err != nil {
		return errors.WithStack(err)
	}
	if active {
		if err := stopMySQL(ctx, mySQLServiceName); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := restoreBackup(ctx, tmpDir, mySQLDirectory); err != nil {
		return errors.WithStack(err)
	}

	if err := startMySQL(ctx, mySQLServiceName); err != nil {
		return errors.WithStack(err)
	}

	send(&agentv1.JobResult{
		JobId:     j.id,
		Timestamp: timestamppb.Now(),
		Result: &agentv1.JobResult_MysqlRestoreBackup{
			MysqlRestoreBackup: &agentv1.JobResult_MySQLRestoreBackup{},
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

func prepareRestoreCommands( //nolint:nonamedreturns
	ctx context.Context,
	folder string,
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
		folder)
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
		"--parallel=10")
	xbstreamCmd.Stdin = xbcloudStdout
	xbstreamCmd.Stderr = stderr
	xbstreamCmd.Stdout = stdout

	return xbcloudCmd, xbstreamCmd, nil
}

func (j *MySQLRestoreJob) restoreMySQLFromS3(ctx context.Context, targetDirectory string) (rerr error) {
	pipeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var stderr, stdout bytes.Buffer

	artifactFolder := path.Join(j.folder, j.name)

	j.l.Debugf("Artifact folder is: %s", artifactFolder)

	xbcloudCmd, xbstreamCmd, err := prepareRestoreCommands(
		pipeCtx,
		artifactFolder,
		&j.locationConfig,
		targetDirectory,
		&stderr,
		&stdout)
	if err != nil {
		return err
	}

	wrapError := func(err error) error {
		return errors.Wrapf(err, "stderr: %s\n stdout: %s\n", stderr.String(), stdout.String()) //nolint:revive
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

func mySQLActive(ctx context.Context, mySQLServiceName string) (bool, error) {
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

func stopMySQL(ctx context.Context, mySQLServiceName string) error {
	ctx, cancel := context.WithTimeout(ctx, systemctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "stop", mySQLServiceName)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting systemctl stop command failed")
	}

	return errors.Wrap(cmd.Wait(), "waiting systemctl stop command failed")
}

func startMySQL(ctx context.Context, mySQLServiceName string) error {
	ctx, cancel := context.WithTimeout(ctx, systemctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "start", mySQLServiceName)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "starting systemctl start command failed")
	}

	return errors.Wrap(cmd.Wait(), "waiting systemctl start command failed")
}

func chownRecursive(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		return errors.WithStack(os.Chown(name, uid, gid))
	})
}

// mySQLUserAndGroupIDs returns uid, gid if error is nil.
func mySQLUserAndGroupIDs() (int, int, error) {
	u, err := user.Lookup(mySQLSystemUserName)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	g, err := user.LookupGroup(mySQLSystemGroupName)
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

func getPermissions(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get permissions for path: %s", path)
	}
	return info.Mode(), nil
}

func restoreBackup(ctx context.Context, backupDirectory, mySQLDirectory string) error {
	// TODO We should implement recognizing correct default permissions based on DB configuration.
	// Setting default value in case the base MySQL folder have been lost.
	mysqlDirPermissions := os.FileMode(0o750)

	if output, err := exec.CommandContext( //nolint:gosec
		ctx,
		xtrabackupBin,
		"--decompress",
		"--target-dir="+backupDirectory).CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to decompress, output: %s", string(output))
	}

	if output, err := exec.CommandContext( //nolint:gosec
		ctx,
		xtrabackupBin,
		"--prepare",
		"--target-dir="+backupDirectory).CombinedOutput(); err != nil {
		return errors.Wrapf(err, "failed to prepare, output: %s", string(output))
	}

	exists, err := isPathExists(mySQLDirectory)
	if err != nil {
		return errors.WithStack(err)
	}
	if exists {
		mysqlDirPermissions, err = getPermissions(mySQLDirectory)
		if err != nil {
			return errors.Wrap(err, "failed to get MySQL base directory permissions")
		}
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

	// Set such permissions as original directory has before restoring.
	// If original directory was absent, we set predefined permissions.
	// Permissions inside DB's main directory are managed by xtrabackup utility, and we don't change them.
	if err := os.Chmod(mySQLDirectory, mysqlDirPermissions); err != nil {
		return errors.Wrap(err, "failed to change permissions for MySQL base directory")
	}

	return nil
}

// getMysqlServiceName returns MySQL system service name.
func getMysqlServiceName(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, systemctlTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "list-unit-files", "--type=service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "failed to list system services, output: %s", string(output))
	}

	if serviceName := mysqlServiceRegex.Find(output); serviceName != nil {
		return string(serviceName), nil
	}

	return "", errors.New("mysql service not found in the system")
}
