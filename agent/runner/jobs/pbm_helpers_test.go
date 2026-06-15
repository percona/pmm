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
	"errors"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDescribePollConfig(t *testing.T, opts ...func(*pbmDescribePollConfig)) *pbmDescribePollConfig {
	t.Helper()

	cfg := &pbmDescribePollConfig{
		l:               logrus.New(),
		dsn:             "mongodb://localhost",
		operation:       pbmCmdBackup,
		targetName:      "2024-01-01T00:00:00Z",
		startedAt:       time.Now(),
		describeRetries: maxDescribeCommandRetries,
		fetchDescribe: func(context.Context) (describeInfo, error) {
			return describeInfo{Status: pbmStatusDone}, nil
		},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func TestCreatePBMConfig(t *testing.T) {
	s3Config := S3LocationConfig{
		Endpoint:     "test_endpoint",
		AccessKey:    "test_access_key",
		SecretKey:    "test_secret_key",
		BucketName:   "test_bucket_name",
		BucketRegion: "test_region",
	}

	filesystemStorageConfig := FilesystemBackupLocationConfig{
		Path: "/test/path",
	}

	expectedOutput1 := PBMConfig{
		PITR: PITR{Enabled: true},
		Storage: Storage{
			Type: "s3",
			S3: S3{
				EndpointURL: "test_endpoint",
				Credentials: Credentials{
					AccessKeyID:     "test_access_key",
					SecretAccessKey: "test_secret_key",
				},
				Bucket: "test_bucket_name",
				Region: "test_region",
				Prefix: "test_prefix",
			},
		},
	}
	expectedOutput2 := PBMConfig{
		PITR: PITR{Enabled: false},
		Storage: Storage{
			Type: "filesystem",
			FileSystem: FileSystem{
				Path: "/test/path/test_prefix",
			},
		},
	}

	for _, test := range []struct {
		name          string
		inputLocation BackupLocationConfig
		inputPitr     bool
		output        *PBMConfig
		errString     string
	}{
		{
			name: "invalid location type",
			inputLocation: BackupLocationConfig{
				Type:                    BackupLocationType("invalid type"),
				S3Config:                &s3Config,
				FilesystemStorageConfig: nil,
			},
			inputPitr: true,
			output:    nil,
			errString: "unknown location config",
		},
		{
			name: "s3 config type",
			inputLocation: BackupLocationConfig{
				Type:                    S3BackupLocationType,
				S3Config:                &s3Config,
				FilesystemStorageConfig: nil,
			},
			inputPitr: true,
			output:    &expectedOutput1,
			errString: "",
		},
		{
			name: "filesystem config type",
			inputLocation: BackupLocationConfig{
				Type:                    FilesystemBackupLocationType,
				S3Config:                nil,
				FilesystemStorageConfig: &filesystemStorageConfig,
			},
			inputPitr: false,
			output:    &expectedOutput2,
			errString: "",
		},
		{
			name: "ignores filled up config instead relying on config type",
			inputLocation: BackupLocationConfig{
				Type:                    FilesystemBackupLocationType,
				S3Config:                &s3Config,
				FilesystemStorageConfig: &filesystemStorageConfig,
			},
			inputPitr: false,
			output:    &expectedOutput2,
			errString: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res, err := createPBMConfig(new(test.inputLocation), "test_prefix", test.inputPitr)
			if test.errString != "" {
				require.ErrorContains(t, err, test.errString)
				assert.Nil(t, res)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.output, res)
		})
	}
}

func TestIsTransientPBMDescribeError(t *testing.T) {
	t.Parallel()

	assert.False(t, isTransientPBMDescribeError(nil))

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no such file",
			err:  errors.New(`get file 2024-01-01T00:00:00Z/rs0/metadata.json: no such file`),
			want: true,
		},
		{
			name: "file is empty",
			err:  errors.New("get file foo: file is empty"),
			want: true,
		},
		{
			name: "backup meta not found",
			err:  errors.New("get backup meta: not found"),
			want: true,
		},
		{
			name: "get snapshot size",
			err:  errors.New("get snapshot size: missed file"),
			want: true,
		},
		{
			name: "generic not found",
			err:  errors.New("authentication failed: user not found"),
			want: false,
		},
		{
			name: "real failure",
			err:  errors.New("permission denied"),
			want: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isTransientPBMDescribeError(tc.err))
		})
	}
}

func TestOperationIsRunning(t *testing.T) {
	t.Parallel()

	backupCfg := &pbmDescribePollConfig{
		operation:  pbmCmdBackup,
		targetName: "backup-1",
	}
	status := &pbmStatus{}
	status.Running.Type = pbmCmdBackup
	status.Running.Name = "backup-1"
	assert.True(t, backupCfg.operationIsRunning(status))

	restoreCfg := &pbmDescribePollConfig{
		operation:  pbmCmdRestore,
		targetName: "restore-1",
	}
	status.Running.Type = pbmCmdRestore
	status.Running.Name = "restore-1"
	assert.True(t, restoreCfg.operationIsRunning(status))
	status.Running.Name = "restore-2"
	assert.False(t, restoreCfg.operationIsRunning(status))

	customCfg := &pbmDescribePollConfig{
		isRunning: func(*pbmStatus) bool { return true },
	}
	assert.True(t, customCfg.operationIsRunning(status))

	unknownCfg := &pbmDescribePollConfig{operation: "unknown"}
	assert.False(t, unknownCfg.operationIsRunning(status))
}

func TestSnapshotForTarget(t *testing.T) {
	t.Parallel()

	cfg := &pbmDescribePollConfig{
		operation:  pbmCmdBackup,
		targetName: "snap-1",
	}
	status := &pbmStatus{}
	status.Backups.Snapshot = []pbmSnapshot{{Name: "snap-1"}}
	assert.NotNil(t, cfg.snapshotForTarget(status))

	cfg.operation = pbmCmdRestore
	assert.Nil(t, cfg.snapshotForTarget(status))
}

func TestShouldRetryDescribeFailure(t *testing.T) {
	t.Parallel()

	transientErr := errors.New("no such file")
	assert.True(t, shouldRetryDescribeFailure(transientErr, time.Now()))
	assert.False(t, shouldRetryDescribeFailure(transientErr, time.Now().Add(-pbmDescribeStartupGrace)))
	assert.False(t, shouldRetryDescribeFailure(errors.New("permission denied"), time.Now()))
}

func TestRetryDescribeCommand(t *testing.T) {
	t.Parallel()

	cfg := &pbmDescribePollConfig{
		l:               logrus.New(),
		operation:       pbmCmdBackup,
		describeRetries: 1,
	}

	assert.True(t, cfg.retryDescribeCommand(errors.New("temporary")))
	assert.Equal(t, 0, cfg.describeRetries)
	assert.False(t, cfg.retryDescribeCommand(errors.New("temporary")))
}

func TestDescribeFailureError(t *testing.T) {
	t.Parallel()

	err := describeFailureError(describeInfo{Status: pbmStatusError}, pbmCmdBackup)
	require.EqualError(t, err, "backup failed")

	err = describeFailureError(describeInfo{Status: pbmStatusError, Error: "oplog gap"}, pbmCmdBackup)
	require.EqualError(t, err, "oplog gap")
}

func TestGroupDescribeErrors_AllBranches(t *testing.T) {
	t.Parallel()

	err := groupDescribeErrors(describeInfo{})
	require.ErrorIs(t, err, errPBMOperationFailed)

	err = groupDescribeErrors(describeInfo{Error: "top level"})
	require.EqualError(t, err, "top level")

	err = groupDescribeErrors(describeInfo{
		ReplSets: []replSet{{Name: "rs0", Error: "rs failed"}},
	})
	require.EqualError(t, err, "replset: rs0, error: rs failed")
}

func TestPollPBMDescribeOnce(t *testing.T) {
	t.Parallel()

	t.Run("describe done", func(t *testing.T) {
		t.Parallel()
		done, err := pollPBMDescribeOnce(context.Background(), newTestDescribePollConfig(t))
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("describe in progress", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: "running"}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("describe canceled", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: pbmStatusCanceled}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "backup was canceled")
		assert.True(t, done)
	})

	t.Run("describe partly done", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{
					Status: pbmStatusPartlyDone,
					ReplSets: []replSet{{
						Name:   "rs0",
						Status: pbmStatusPartlyDone,
						Nodes: []node{{
							Name:   "node1",
							Status: pbmStatusError,
							Error:  "failed node",
						}},
					}},
				}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "replset: rs0, node: node1, error: failed node")
		assert.True(t, done)
	})

	t.Run("status fetch error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("describe failed")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return nil, errors.New("status unavailable")
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.ErrorContains(t, err, "failed to get pbm status")
		assert.False(t, done)
	})

	t.Run("running backup with transient describe error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.targetName
				return status, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("running backup retries describe command", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.describeRetries = maxDescribeCommandRetries
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.targetName
				return status, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
		assert.Equal(t, maxDescribeCommandRetries-1, cfg.describeRetries)
	})

	t.Run("snapshot done when describe fails", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Backups.Snapshot = []pbmSnapshot{{
					Name:   c.targetName,
					Status: pbmStatusDone,
				}}
				return status, nil
			}
			c.findSnapshot = func(status *pbmStatus) *pbmSnapshot {
				return findPBMSnapshot(status, c.targetName)
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("snapshot terminal error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Backups.Snapshot = []pbmSnapshot{{
					Name:   c.targetName,
					Status: pbmStatusError,
					Error:  "storage error",
				}}
				return status, nil
			}
			c.findSnapshot = func(status *pbmStatus) *pbmSnapshot {
				return findPBMSnapshot(status, c.targetName)
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "storage error")
		assert.True(t, done)
	})

	t.Run("restore done when describe fails", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.operation = pbmCmdRestore
			c.targetName = "2024-01-01T12:00:00Z"
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
			c.fetchRestoreList = func(context.Context) ([]pbmListRestore, error) {
				return []pbmListRestore{{
					Name:   c.targetName,
					Status: pbmStatusDone,
				}}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("restore terminal error when describe fails", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.operation = pbmCmdRestore
			c.targetName = "2024-01-01T12:00:00Z"
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.describeRetries = 0
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
			c.fetchRestoreList = func(context.Context) ([]pbmListRestore, error) {
				return []pbmListRestore{{
					Name:   c.targetName,
					Status: pbmStatusError,
					Error:  "node copy failed",
				}}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "node copy failed")
		assert.True(t, done)
	})

	t.Run("startup grace for transient error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("file is empty")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("retries after startup grace when retries remain", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.describeRetries = 2
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("file is empty")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
		assert.Equal(t, 1, cfg.describeRetries)
	})

	t.Run("running backup with exhausted retries keeps waiting", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.describeRetries = 0
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.targetName
				return status, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("describe failure without running backup", func(t *testing.T) {
		t.Parallel()
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.startedAt = time.Now().Add(-pbmDescribeStartupGrace)
			c.describeRetries = 0
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollPBMDescribeOnce(context.Background(), cfg)
		require.ErrorContains(t, err, "failed to get backup status")
		assert.False(t, done)
	})
}

func TestWaitForPBMDescribe(t *testing.T) {
	t.Run("completes when describe reports done", func(t *testing.T) {
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.pollInterval = time.Millisecond
		})
		err := waitForPBMDescribe(context.Background(), cfg)
		require.NoError(t, err)
	})

	t.Run("returns describe error", func(t *testing.T) {
		cfg := newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.pollInterval = time.Millisecond
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: pbmStatusCanceled}, nil
			}
		})
		err := waitForPBMDescribe(context.Background(), cfg)
		require.EqualError(t, err, "backup was canceled")
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := waitForPBMDescribe(ctx, newTestDescribePollConfig(t, func(c *pbmDescribePollConfig) {
			c.pollInterval = time.Millisecond
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: "running"}, nil
			}
		}))
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestWritePBMConfigFile(t *testing.T) {
	t.Parallel()

	conf, err := createPBMConfig(&BackupLocationConfig{
		Type: FilesystemBackupLocationType,
		FilesystemStorageConfig: &FilesystemBackupLocationConfig{
			Path: "/tmp/pbm",
		},
	}, "artifact", false)
	require.NoError(t, err)

	path, err := writePBMConfigFile(conf)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Remove(path))
	})
	require.FileExists(t, path)
}

func TestDescribeTerminalError(t *testing.T) {
	t.Parallel()

	done, err := describeTerminalError(describeInfo{Status: pbmStatusDone}, pbmCmdBackup)
	require.NoError(t, err)
	assert.True(t, done)

	done, err = describeTerminalError(describeInfo{Status: pbmStatusCanceled}, pbmCmdBackup)
	require.EqualError(t, err, "backup was canceled")
	assert.True(t, done)

	done, err = describeTerminalError(describeInfo{Status: pbmStatusError, Error: "oplog has insufficient range"}, pbmCmdBackup)
	require.EqualError(t, err, "oplog has insufficient range")
	assert.True(t, done)

	done, err = describeTerminalError(describeInfo{Status: pbmStatusPartlyDone, Error: "partial"}, pbmCmdBackup)
	require.EqualError(t, err, "partial")
	assert.True(t, done)

	done, err = describeTerminalError(describeInfo{Status: "running"}, pbmCmdBackup)
	require.NoError(t, err)
	assert.False(t, done)
}

func TestTerminalStatusError(t *testing.T) {
	t.Parallel()

	done, err := terminalStatusError(pbmStatusDone, "", pbmCmdBackup)
	require.NoError(t, err)
	assert.True(t, done)

	done, err = terminalStatusError(pbmStatusCanceled, "", pbmCmdBackup)
	require.EqualError(t, err, "backup was canceled")
	assert.True(t, done)

	done, err = terminalStatusError(pbmStatusError, "storage unavailable", pbmCmdBackup)
	require.EqualError(t, err, "storage unavailable")
	assert.True(t, done)

	done, err = terminalStatusError(pbmStatusError, "", pbmCmdBackup)
	require.EqualError(t, err, "backup failed")
	assert.True(t, done)

	done, err = terminalStatusError(pbmStatusPartlyDone, "", pbmCmdRestore)
	require.EqualError(t, err, "restore partly completed")
	assert.True(t, done)

	done, err = terminalStatusError("running", "", pbmCmdBackup)
	require.NoError(t, err)
	assert.False(t, done)
}

func TestFindPBMListRestore(t *testing.T) {
	t.Parallel()

	list := []pbmListRestore{
		{Name: "2024-01-01T00:00:00Z", Status: pbmStatusDone},
		{Name: "2024-01-02T00:00:00Z", Status: pbmStatusError, Error: "failed"},
	}

	assert.Nil(t, findPBMListRestore(list, "missing"))
	require.NotNil(t, findPBMListRestore(list, "2024-01-02T00:00:00Z"))
}

func TestFindPBMSnapshot(t *testing.T) {
	t.Parallel()

	status := &pbmStatus{}
	status.Backups.Snapshot = []pbmSnapshot{
		{Name: "2024-01-01T00:00:00Z", Status: pbmStatusDone},
		{Name: "2024-01-02T00:00:00Z", Status: pbmStatusError, Error: "failed"},
	}

	assert.Nil(t, findPBMSnapshot(status, "missing"))
	require.NotNil(t, findPBMSnapshot(status, "2024-01-02T00:00:00Z"))
}

func TestGroupDescribeErrors(t *testing.T) {
	t.Parallel()

	err := groupDescribeErrors(describeInfo{
		Status: pbmStatusPartlyDone,
		ReplSets: []replSet{{
			Name:   "rs0",
			Status: pbmStatusPartlyDone,
			Nodes: []node{{
				Name:   "node1",
				Status: pbmStatusError,
				Error:  "copy failed",
			}},
		}},
	})
	require.EqualError(t, err, "replset: rs0, node: node1, error: copy failed")
}

func TestFindPITRRestoreSkipsInvalidEntries(t *testing.T) {
	t.Parallel()

	startedAt, err := time.Parse(time.RFC3339Nano, "2022-10-11T14:53:20.000000000Z")
	require.NoError(t, err)

	list := []pbmListRestore{
		{Name: "invalid-name", Type: "pitr", PITR: 1000000000},
		{Name: "2022-10-11T14:53:20.000000001Z", Type: "snapshot", Snapshot: "snap"},
	}
	assert.Nil(t, findPITRRestore(list, 1000000000, startedAt))
}

func TestFindPITRRestore(t *testing.T) {
	// Tested func searches from the end, so we place records to be skipped at the end.
	testList := []pbmListRestore{
		{
			Name: "2022-10-11T14:53:19.000000001Z",
			Type: "pitr",
			PITR: 1000000000,
		},
		{
			Name: "2022-10-11T14:53:20.000000001Z",
			Type: "pitr",
			PITR: 1000000000,
		},
		{
			Name: "2022-error-11T14:53:20.000000001Z",
			Type: "pitr",
			PITR: 1000000000,
		},
		{
			Name: "2022-10-11T14:53:20.000000001Z",
			Type: "snapshot",
		},
		{
			Name: "2022-10-11T14:53:20.000000010Z",
			Type: "pitr",
			PITR: 1000000001,
		},
	}

	for _, tc := range []struct {
		name                string
		restoreInfoPITRTime int64
		startedAtString     string
		expected            *pbmListRestore
	}{
		{
			name:                "case1",
			restoreInfoPITRTime: 1000000000,
			startedAtString:     "2022-10-11T14:53:20.000000000Z",
			expected:            &pbmListRestore{Name: "2022-10-11T14:53:20.000000001Z", Type: "pitr", PITR: 1000000000},
		},
		{
			name:                "case2",
			restoreInfoPITRTime: 1000000001,
			startedAtString:     "2022-10-11T14:53:20.000000002Z",
			expected:            &pbmListRestore{Name: "2022-10-11T14:53:20.000000010Z", Type: "pitr", PITR: 1000000001},
		},
		{
			name:                "case3",
			restoreInfoPITRTime: 1000000002,
			startedAtString:     "2022-10-11T14:53:20.000000000Z",
			expected:            nil,
		},
		{
			name:                "case4",
			restoreInfoPITRTime: 1000000000,
			startedAtString:     "2022-10-11T14:53:20.000000020Z",
			expected:            nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			startedAt, err := time.Parse(time.RFC3339Nano, tc.startedAtString)
			require.NoError(t, err)

			res := findPITRRestore(testList, tc.restoreInfoPITRTime, startedAt)
			assert.Equal(t, tc.expected, res)
		})
	}
}
