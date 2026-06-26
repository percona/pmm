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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/percona/pmm/agent/utils/poll"
)

const (
	cmdTimeout          = 60 * time.Minute
	resyncTimeout       = 5 * time.Minute
	statusCheckInterval = 5 * time.Second
	maxRestoreChecks    = 100

	maxDescribeRetries = 5
	// PBM waits up to ~33s for backup metadata. Allow extra margin for describe-backup.
	describeStartupGrace = 60 * time.Second
	// After the operation stops, status/list metadata can lag behind describe failures.
	describeCompletionGrace     = 5 * time.Minute
	describeRunningWarnInterval = 5 * time.Minute

	pbmCmdBackup  = "backup"
	pbmCmdRestore = "restore"

	pbmStatusDone       = "done"
	pbmStatusCanceled   = "canceled"
	pbmStatusError      = "error"
	pbmStatusPartlyDone = "partlyDone"
)

var errPBMOperationFailed = errors.New("operation failed")

type pbmSeverity int

type describeInfo struct {
	Status   string    `json:"status"`
	Error    string    `json:"error"`
	ReplSets []replSet `json:"replsets"`
}

type replSet struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
	Node   string `json:"node,omitempty"`
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

func execPBMCommand(ctx context.Context, dsn string, to any, args ...string) error {
	nCtx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()

	args = append(args, "--out=json", "--mongodb-uri="+dsn)
	cmd := exec.CommandContext(nCtx, pbmBin, args...)

	b, err := cmd.Output()
	if err != nil {
		// try to parse pbm error message
		if len(b) != 0 {
			var pbmErr pbmError
			e := json.Unmarshal(b, &pbmErr)
			if e == nil {
				return errors.New(pbmErr.Error)
			}
		}
		return err
	}

	return json.Unmarshal(b, to)
}

func retrieveLogs(ctx context.Context, dsn string, event string) ([]pbmLogEntry, error) {
	var logs []pbmLogEntry

	err := execPBMCommand(ctx, dsn, &logs, "logs", "--event="+event, "--tail=0")
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func waitForPBMNoRunningOperations(ctx context.Context, l logrus.FieldLogger, dsn string) error {
	l.Info("Waiting for no running pbm operations.")
	started := false

	return poll.UntilContextTimeout(ctx, statusCheckInterval, func(ctx context.Context) (bool, error) {
		// Preserve previous behavior: first status check runs after the first tick.
		if !started {
			started = true
			return false, nil
		}

		status, err := getPBMStatus(ctx, dsn)
		if err != nil {
			return false, err
		}
		return status.Running.Type == "", nil
	})
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
	err := execPBMCommand(ctx, dsn, &status, "status")
	if err != nil {
		return nil, fmt.Errorf("pbm status error: %w", err)
	}
	return &status, nil
}

type describePoller struct {
	l                logrus.FieldLogger
	dsn              string
	operation        string
	name             string
	startedAt        time.Time
	finishedAt       time.Time
	lastRunningWarn  time.Time
	pollEvery        time.Duration
	retries          int
	fetchDescribe    func(context.Context) (describeInfo, error)
	fetchStatus      func(context.Context, string) (*pbmStatus, error)
	fetchRestoreList func(context.Context) ([]pbmListRestore, error)
	isRunning        func(*pbmStatus) bool
	findSnapshot     func(*pbmStatus) *pbmSnapshot
}

func (cfg *describePoller) interval() time.Duration {
	if cfg.pollEvery > 0 {
		return cfg.pollEvery
	}
	return statusCheckInterval
}

func newDescribePoller(l logrus.FieldLogger, dsn, operation, name string, fetchDescribe func(context.Context) (describeInfo, error)) *describePoller {
	return &describePoller{
		l:             l,
		dsn:           dsn,
		operation:     operation,
		name:          name,
		startedAt:     time.Now(),
		retries:       maxDescribeRetries,
		fetchDescribe: fetchDescribe,
	}
}

func waitForPBMBackup(ctx context.Context, l logrus.FieldLogger, dsn string, name string) error {
	l.Infof("waiting for pbm backup: %s", name)

	return waitDescribe(ctx, newDescribePoller(l, dsn, pbmCmdBackup, name, func(ctx context.Context) (describeInfo, error) {
		var info describeInfo
		err := execPBMCommand(ctx, dsn, &info, "describe-backup", name)
		return info, err
	}))
}

func waitDescribe(ctx context.Context, cfg *describePoller) error {
	return poll.UntilContextTimeout(ctx, cfg.interval(), func(ctx context.Context) (bool, error) {
		return pollDescribeOnce(ctx, cfg)
	})
}

func pollDescribeOnce(ctx context.Context, cfg *describePoller) (bool, error) {
	info, describeErr := cfg.fetchDescribe(ctx)
	if describeErr == nil {
		cfg.retries = maxDescribeRetries
		return checkDescribe(info, cfg.operation)
	}

	status, statusErr := cfg.getPBMStatus(ctx)
	if statusErr != nil {
		if errors.Is(statusErr, context.Canceled) || errors.Is(statusErr, context.DeadlineExceeded) {
			return false, statusErr
		}

		cfg.l.Debugf("failed to get pbm status while waiting for %s %q: %v", cfg.operation, cfg.name, statusErr)
		if cfg.retryDescribeErr(describeErr) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get %s status: %w", cfg.operation, describeErr)
	}

	running := cfg.opRunning(status)
	cfg.trackFinished(running)

	if running {
		cfg.warnRunningDescribe(describeErr)
		cfg.logRunningDescribeErr(describeErr)
		return false, nil
	}

	if done, err := cfg.statusFallback(ctx, status); done {
		return true, err
	}

	if cfg.retryDescribeErr(describeErr) {
		return false, nil
	}

	return false, fmt.Errorf("failed to get %s status: %w", cfg.operation, describeErr)
}

func (cfg *describePoller) getPBMStatus(ctx context.Context) (*pbmStatus, error) {
	if cfg.fetchStatus != nil {
		return cfg.fetchStatus(ctx, cfg.dsn)
	}
	return getPBMStatus(ctx, cfg.dsn)
}

func (cfg *describePoller) trackFinished(running bool) {
	if running {
		cfg.finishedAt = time.Time{}
		return
	}
	if cfg.finishedAt.IsZero() {
		cfg.finishedAt = time.Now()
	}
}

func (cfg *describePoller) describeCmd() string {
	return "describe-" + cfg.operation
}

func (cfg *describePoller) warnRunningDescribe(describeErr error) {
	if cfg.startedAt.IsZero() {
		return
	}
	if time.Since(cfg.lastWarnAt()) < describeRunningWarnInterval {
		return
	}
	cfg.lastRunningWarn = time.Now()
	cfg.l.Warnf("%s %q is still running but %s keeps failing: %v",
		cfg.operation, cfg.name, cfg.describeCmd(), describeErr)
}

func (cfg *describePoller) lastWarnAt() time.Time {
	if !cfg.lastRunningWarn.IsZero() {
		return cfg.lastRunningWarn
	}
	return cfg.startedAt
}

func (cfg *describePoller) logRunningDescribeErr(describeErr error) {
	if retryTransient(describeErr, cfg, true) {
		cfg.l.Debugf("%s transient error while %s %q is still running: %v",
			cfg.describeCmd(), cfg.operation, cfg.name, describeErr)
		return
	}
	cfg.l.Debugf("%s error while %s %q is still running: %v",
		cfg.describeCmd(), cfg.operation, cfg.name, describeErr)
}

func (cfg *describePoller) retryDescribeErr(describeErr error) bool {
	if retryTransient(describeErr, cfg, false) {
		cfg.l.Debugf("%s transient error while waiting for %s %q completion metadata: %v",
			cfg.describeCmd(), cfg.operation, cfg.name, describeErr)
		return true
	}
	return cfg.retryDescribeCmd(describeErr)
}

func (cfg *describePoller) retryDescribeCmd(err error) bool {
	if cfg.retries <= 0 {
		return false
	}
	cfg.retries--
	cfg.l.Warnf("%s failed and will retry: %s", cfg.describeCmd(), err)
	return true
}

func (cfg *describePoller) statusFallback(ctx context.Context, status *pbmStatus) (bool, error) {
	if snapshot := cfg.targetSnapshot(status); snapshot != nil {
		return checkStatus(snapshot.Status, snapshot.Error, cfg.operation)
	}

	if cfg.operation != pbmCmdRestore {
		return false, nil
	}

	list, err := cfg.listRestores(ctx)
	if err != nil {
		cfg.l.Debugf("failed to get restore list for fallback: %s", err)
		return false, nil
	}
	if restore := restoreByName(list, cfg.name); restore != nil {
		return checkStatus(restore.Status, restore.Error, cfg.operation)
	}
	return false, nil
}

func (cfg *describePoller) listRestores(ctx context.Context) ([]pbmListRestore, error) {
	if cfg.fetchRestoreList != nil {
		return cfg.fetchRestoreList(ctx)
	}
	var list []pbmListRestore
	err := execPBMCommand(ctx, cfg.dsn, &list, "list", "--restore")
	return list, err
}

func retryTransient(err error, cfg *describePoller, running bool) bool {
	if !isTransientDescribeErr(err) {
		return false
	}
	if running {
		return time.Since(cfg.startedAt) < describeStartupGrace
	}
	since := cfg.startedAt
	if !cfg.finishedAt.IsZero() {
		since = cfg.finishedAt
	}
	return time.Since(since) < describeCompletionGrace
}

// isTransientDescribeErr reports whether describe-backup/restore may fail
// temporarily while PBM metadata is not ready yet. Matches are based on known
// PBM CLI error texts and should be updated when PBM changes them.
func isTransientDescribeErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "file is empty") ||
		strings.Contains(msg, "missed file") ||
		(strings.Contains(msg, "get backup meta") && strings.Contains(msg, "not found")) ||
		strings.Contains(msg, "get snapshot size")
}

func (cfg *describePoller) opRunning(status *pbmStatus) bool {
	if cfg.isRunning != nil {
		return cfg.isRunning(status)
	}
	if cfg.operation != pbmCmdBackup && cfg.operation != pbmCmdRestore {
		return false
	}
	return status.Running.Type == cfg.operation && status.Running.Name == cfg.name
}

func (cfg *describePoller) targetSnapshot(status *pbmStatus) *pbmSnapshot {
	if cfg.findSnapshot != nil {
		return cfg.findSnapshot(status)
	}
	if cfg.operation == pbmCmdBackup {
		return snapshotByName(status, cfg.name)
	}
	return nil
}

func snapshotByName(status *pbmStatus, name string) *pbmSnapshot {
	i := slices.IndexFunc(status.Backups.Snapshot, func(s pbmSnapshot) bool {
		return s.Name == name
	})
	if i < 0 {
		return nil
	}
	return &status.Backups.Snapshot[i]
}

func restoreByName(list []pbmListRestore, name string) *pbmListRestore {
	i := slices.IndexFunc(list, func(r pbmListRestore) bool {
		return r.Name == name
	})
	if i < 0 {
		return nil
	}
	return &list[i]
}

func checkDescribe(info describeInfo, operation string) (bool, error) {
	switch info.Status {
	case pbmStatusError:
		return true, describeErr(info, operation)
	case pbmStatusPartlyDone:
		return true, groupDescribeErrs(info)
	default:
		return checkStatus(info.Status, info.Error, operation)
	}
}

func checkStatus(status, errMsg, operation string) (bool, error) {
	switch status {
	case pbmStatusDone:
		return true, nil
	case pbmStatusCanceled:
		return true, fmt.Errorf("%s was canceled", operation)
	case pbmStatusError:
		if errMsg != "" {
			return true, errors.New(errMsg)
		}
		return true, fmt.Errorf("%s failed", operation)
	case pbmStatusPartlyDone:
		if errMsg != "" {
			return true, errors.New(errMsg)
		}
		return true, fmt.Errorf("%s partly completed", operation)
	default:
		return false, nil
	}
}

func describeErr(info describeInfo, operation string) error {
	err := groupDescribeErrs(info)
	if err != nil && !errors.Is(err, errPBMOperationFailed) {
		return err
	}
	return fmt.Errorf("%s failed", operation)
}

func findPITRRestore(list []pbmListRestore, restoreInfoPITRTime int64, startedAt time.Time) *pbmListRestore {
	for _, v := range slices.Backward(list) {
		// TODO when PITR restore invoked with wrong timestamp pbm marks this restore operation as "snapshot" type.
		if v.Type == "snapshot" && v.Snapshot != "" {
			continue
		}
		// list[i].Name is a string which represents time the restore was started.
		restoreStartedAt, err := time.Parse(time.RFC3339Nano, v.Name)
		if err != nil {
			continue
		}
		// Because of https://jira.percona.com/browse/PBM-723 to find our restore record in the list of all records we're checking:
		// 1. We received PITR field as a response on starting process
		// 2. There is a record with the same PITR field in the list of restoring records
		// 3. Start time of this record is not before the time we asked for restoring.
		if v.PITR == restoreInfoPITRTime && !restoreStartedAt.Before(startedAt) {
			return &v
		}
	}
	return nil
}

func findPITRRestoreName(ctx context.Context, dsn string, restoreInfo *pbmRestore) (string, error) {
	restoreInfoPITRTime, err := time.Parse("2006-01-02T15:04:05", restoreInfo.PITR)
	if err != nil {
		return "", err
	}

	var name string
	checks := 0
	err = poll.UntilContextTimeout(ctx, statusCheckInterval, func(ctx context.Context) (bool, error) {
		err = ctx.Err()
		if err != nil {
			return false, err
		}

		checks++
		var list []pbmListRestore
		err = execPBMCommand(ctx, dsn, &list, "list", "--restore")
		if err != nil {
			return false, fmt.Errorf("pbm status error: %w", err)
		}
		entry := findPITRRestore(list, restoreInfoPITRTime.Unix(), restoreInfo.StartedAt)
		if entry != nil {
			name = entry.Name
			return true, nil
		}
		if checks > maxRestoreChecks {
			return false, errors.New("failed to start restore")
		}
		return false, nil
	})
	if err != nil {
		return "", err
	}

	return name, nil
}

func fetchRestoreDescribe(ctx context.Context, dsn, name, backupType, confFile string) (describeInfo, error) {
	var info describeInfo
	args := []string{"describe-restore", name}
	if backupType == "physical" {
		args = append(args, "--config="+confFile)
	}
	err := execPBMCommand(ctx, dsn, &info, args...)
	return info, err
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

	return waitDescribe(ctx, newDescribePoller(l, dsn, pbmCmdRestore, name, func(ctx context.Context) (describeInfo, error) {
		return fetchRestoreDescribe(ctx, dsn, name, backupType, confFile)
	}))
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
		return fmt.Errorf("pbm config error: %s: %w", string(output), err)
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
			return fmt.Errorf("pbm config resync error: %s: %w", string(output), err)
		}
	}

	return nil
}

func writePBMConfigFile(conf *PBMConfig) (string, error) {
	tmp, err := os.CreateTemp("", "pbm-config-*.yml")
	if err != nil {
		return "", fmt.Errorf("failed to create pbm configuration file: %w", err)
	}

	bytes, err := yaml.Marshal(&conf)
	if err != nil {
		tmp.Close() //nolint:errcheck
		return "", fmt.Errorf("failed to marshal pbm configuration: %w", err)
	}

	_, err = tmp.Write(bytes)
	if err != nil {
		tmp.Close() //nolint:errcheck
		return "", fmt.Errorf("failed to write pbm configuration file: %w", err)
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

func groupDescribeErrs(info describeInfo) error {
	var errMsgs []string

	if info.Error != "" {
		errMsgs = append(errMsgs, info.Error)
	}

	for _, rs := range info.ReplSets {
		if rs.Error != "" {
			errMsgs = append(errMsgs, fmt.Sprintf("replset: %s, error: %s", rs.Name, rs.Error))
		}
		if rs.Status == pbmStatusPartlyDone {
			for _, n := range rs.Nodes {
				if n.Status == pbmStatusError {
					errMsgs = append(errMsgs, fmt.Sprintf("replset: %s, node: %s, error: %s", rs.Name, n.Name, n.Error))
				}
			}
		}
	}

	if len(errMsgs) == 0 {
		return errPBMOperationFailed
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
			return new(time.Unix(snapshot.RestoreTo, 0)), nil
		}
	}

	return nil, fmt.Errorf("couldn't find required snapshot: %w", ErrNotFound)
}

// getSnapshots returns all PBM snapshots found in configured location.
func getSnapshots(ctx context.Context, l logrus.FieldLogger, dsn string) ([]pbmSnapshot, error) {
	// Sometimes PBM returns empty list of snapshots, that's why we're trying to get them several times.
	var snapshots []pbmSnapshot
	checks := 0
	err := poll.UntilContextTimeout(ctx, listCheckInterval, func(ctx context.Context) (bool, error) {
		checks++
		status, err := getPBMStatus(ctx, dsn)
		if err != nil {
			return false, err
		}

		if len(status.Backups.Snapshot) == 0 {
			l.Debugf("Attempt %d to get a list of PBM artifacts has failed.", checks)
			if checks > maxListChecks {
				return false, fmt.Errorf("got no one snapshot: %w", ErrNotFound)
			}
			return false, nil
		}

		snapshots = status.Backups.Snapshot
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return snapshots, nil
}
