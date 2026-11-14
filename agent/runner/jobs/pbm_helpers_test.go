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
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPoller(t *testing.T, opts ...func(*describePoller)) *describePoller {
	t.Helper()

	cfg := &describePoller{
		l:         logrus.New(),
		dsn:       "mongodb://localhost",
		operation: pbmCmdBackup,
		name:      "2024-01-01T00:00:00Z",
		startedAt: time.Now(),
		retries:   maxDescribeRetries,
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

func TestIsTransientDescribeErr(t *testing.T) {
	t.Parallel()

	assert.False(t, isTransientDescribeErr(nil))

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
			name: "backup meta permission denied",
			err:  errors.New("get backup meta: permission denied"),
			want: false,
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
			assert.Equal(t, tc.want, isTransientDescribeErr(tc.err))
		})
	}
}

func TestOpRunning(t *testing.T) {
	t.Parallel()

	backupCfg := &describePoller{
		operation: pbmCmdBackup,
		name:      "backup-1",
	}
	status := &pbmStatus{}
	status.Running.Type = pbmCmdBackup
	status.Running.Name = "backup-1"
	assert.True(t, backupCfg.opRunning(status))
	status.Running.Name = "backup-2"
	assert.False(t, backupCfg.opRunning(status))

	restoreCfg := &describePoller{
		operation: pbmCmdRestore,
		name:      "restore-1",
	}
	status.Running.Type = pbmCmdRestore
	status.Running.Name = "restore-1"
	assert.True(t, restoreCfg.opRunning(status))
	status.Running.Name = "restore-2"
	assert.True(t, restoreCfg.opRunning(status))

	customCfg := &describePoller{
		isRunning: func(*pbmStatus) bool { return true },
	}
	assert.True(t, customCfg.opRunning(status))

	unknownCfg := &describePoller{operation: "unknown"}
	assert.False(t, unknownCfg.opRunning(status))
}

func TestTargetSnapshot(t *testing.T) {
	t.Parallel()

	cfg := &describePoller{
		operation: pbmCmdBackup,
		name:      "snap-1",
	}
	status := &pbmStatus{}
	status.Backups.Snapshot = []pbmSnapshot{{Name: "snap-1"}}
	assert.NotNil(t, cfg.targetSnapshot(status))

	cfg.operation = pbmCmdRestore
	assert.Nil(t, cfg.targetSnapshot(status))
}

func TestRetryTransient(t *testing.T) {
	t.Parallel()

	transientErr := errors.New("no such file")

	t.Run("startup grace while running", func(t *testing.T) {
		t.Parallel()
		cfg := &describePoller{startedAt: time.Now()}
		assert.True(t, retryTransient(transientErr, cfg, true))
		cfg.startedAt = time.Now().Add(-describeStartupGrace)
		assert.False(t, retryTransient(transientErr, cfg, true))
		assert.False(t, retryTransient(errors.New("permission denied"), &describePoller{startedAt: time.Now()}, true))
	})

	t.Run("completion grace after operation finished", func(t *testing.T) {
		t.Parallel()
		cfg := &describePoller{
			startedAt:  time.Now().Add(-2 * time.Hour),
			finishedAt: time.Now().Add(-1 * time.Minute),
		}
		assert.True(t, retryTransient(transientErr, cfg, false))
		cfg.finishedAt = time.Now().Add(-describeCompletionGrace)
		assert.False(t, retryTransient(transientErr, cfg, false))
	})
}

func TestRetryDescribeCmd(t *testing.T) {
	t.Parallel()

	cfg := &describePoller{
		l:         logrus.New(),
		operation: pbmCmdBackup,
		retries:   1,
	}

	assert.True(t, cfg.retryDescribeCmd(errors.New("temporary")))
	assert.Equal(t, 0, cfg.retries)
	assert.False(t, cfg.retryDescribeCmd(errors.New("temporary")))
}

func TestDescribeErr(t *testing.T) {
	t.Parallel()

	err := describeErr(describeInfo{Status: pbmStatusError}, pbmCmdBackup)
	require.EqualError(t, err, "backup failed")

	err = describeErr(describeInfo{Status: pbmStatusError, Error: "oplog gap"}, pbmCmdBackup)
	require.EqualError(t, err, "oplog gap")
}

func TestGroupDescribeErrs_AllBranches(t *testing.T) {
	t.Parallel()

	err := groupDescribeErrs(describeInfo{})
	require.ErrorIs(t, err, errPBMOperationFailed)

	err = groupDescribeErrs(describeInfo{Error: "top level"})
	require.EqualError(t, err, "top level")

	err = groupDescribeErrs(describeInfo{
		ReplSets: []replSet{{Name: "rs0", Error: "rs failed"}},
	})
	require.EqualError(t, err, "replset: rs0, error: rs failed")
}

func TestPollDescribeOnce(t *testing.T) {
	t.Parallel()

	t.Run("describe done", func(t *testing.T) {
		t.Parallel()
		done, err := pollDescribeOnce(context.Background(), newTestPoller(t))
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("describe in progress", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: "running"}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("describe canceled", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: pbmStatusCanceled}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "backup was canceled")
		assert.True(t, done)
	})

	t.Run("describe partly done", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
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
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "replset: rs0, node: node1, error: failed node")
		assert.True(t, done)
	})

	t.Run("status fetch error retries polling", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("describe failed")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return nil, errors.New("status unavailable")
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
		assert.Equal(t, maxDescribeRetries-1, cfg.retries)
	})

	t.Run("status fetch context canceled", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("describe failed")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return nil, context.Canceled
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.ErrorIs(t, err, context.Canceled)
		assert.False(t, done)
	})

	t.Run("running backup with transient describe error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.name
				return status, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("running backup does not consume retries", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.retries = maxDescribeRetries
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.name
				return status, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
		assert.Equal(t, maxDescribeRetries, cfg.retries)
	})

	t.Run("snapshot done when describe fails", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Backups.Snapshot = []pbmSnapshot{{
					Name:   c.name,
					Status: pbmStatusDone,
				}}
				return status, nil
			}
			c.findSnapshot = func(status *pbmStatus) *pbmSnapshot {
				return snapshotByName(status, c.name)
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("snapshot terminal error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Backups.Snapshot = []pbmSnapshot{{
					Name:   c.name,
					Status: pbmStatusError,
					Error:  "storage error",
				}}
				return status, nil
			}
			c.findSnapshot = func(status *pbmStatus) *pbmSnapshot {
				return snapshotByName(status, c.name)
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "storage error")
		assert.True(t, done)
	})

	t.Run("restore done when describe fails", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.operation = pbmCmdRestore
			c.name = "2024-01-01T12:00:00Z"
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
			c.fetchRestoreList = func(context.Context) ([]pbmListRestore, error) {
				return []pbmListRestore{{
					Name:   c.name,
					Status: pbmStatusDone,
				}}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.True(t, done)
	})

	t.Run("restore terminal error when describe fails", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.operation = pbmCmdRestore
			c.name = "2024-01-01T12:00:00Z"
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.retries = 0
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
			c.fetchRestoreList = func(context.Context) ([]pbmListRestore, error) {
				return []pbmListRestore{{
					Name:   c.name,
					Status: pbmStatusError,
					Error:  "node copy failed",
				}}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.EqualError(t, err, "node copy failed")
		assert.True(t, done)
	})

	t.Run("startup grace for transient error", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("file is empty")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("retries after startup grace when retries remain", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.finishedAt = time.Now().Add(-describeCompletionGrace)
			c.retries = 2
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
		assert.Equal(t, 1, cfg.retries)
	})

	t.Run("running backup with exhausted retries keeps waiting", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.retries = 0
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.name
				return status, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("describe failure without running backup", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.startedAt = time.Now().Add(-describeCompletionGrace)
			c.finishedAt = time.Now().Add(-describeCompletionGrace)
			c.retries = 0
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.ErrorContains(t, err, "failed to get backup status")
		assert.False(t, done)
	})

	t.Run("restore list fetch error keeps polling", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.operation = pbmCmdRestore
			c.name = "2024-01-01T12:00:00Z"
			c.startedAt = time.Now().Add(-describeStartupGrace)
			c.finishedAt = time.Now().Add(-30 * time.Second)
			c.retries = 0
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("no such file")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
			c.fetchRestoreList = func(context.Context) ([]pbmListRestore, error) {
				return nil, errors.New("list unavailable")
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("transient error after operation finished uses completion grace", func(t *testing.T) {
		t.Parallel()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.startedAt = time.Now().Add(-2 * time.Hour)
			c.finishedAt = time.Now().Add(-30 * time.Second)
			c.retries = 0
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("file is empty")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				return &pbmStatus{}, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
	})

	t.Run("warns when describe keeps failing while running", func(t *testing.T) {
		t.Parallel()
		logger, hook := logrustest.NewNullLogger()
		cfg := newTestPoller(t, func(c *describePoller) {
			c.l = logger
			c.startedAt = time.Now().Add(-describeRunningWarnInterval - time.Second)
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{}, errors.New("permission denied")
			}
			c.fetchStatus = func(context.Context, string) (*pbmStatus, error) {
				status := &pbmStatus{}
				status.Running.Type = pbmCmdBackup
				status.Running.Name = c.name
				return status, nil
			}
		})
		done, err := pollDescribeOnce(context.Background(), cfg)
		require.NoError(t, err)
		assert.False(t, done)
		var found bool
		for _, entry := range hook.Entries {
			if entry.Level == logrus.WarnLevel && strings.Contains(entry.Message, "still running") {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestWaitDescribe(t *testing.T) {
	t.Run("completes when describe reports done", func(t *testing.T) {
		cfg := newTestPoller(t, func(c *describePoller) {
			c.pollEvery = time.Millisecond
		})
		err := waitDescribe(context.Background(), cfg)
		require.NoError(t, err)
	})

	t.Run("returns describe error", func(t *testing.T) {
		cfg := newTestPoller(t, func(c *describePoller) {
			c.pollEvery = time.Millisecond
			c.fetchDescribe = func(context.Context) (describeInfo, error) {
				return describeInfo{Status: pbmStatusCanceled}, nil
			}
		})
		err := waitDescribe(context.Background(), cfg)
		require.EqualError(t, err, "backup was canceled")
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := waitDescribe(ctx, newTestPoller(t, func(c *describePoller) {
			c.pollEvery = time.Millisecond
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

func TestCheckDescribe(t *testing.T) {
	t.Parallel()

	done, err := checkDescribe(describeInfo{Status: pbmStatusDone}, pbmCmdBackup)
	require.NoError(t, err)
	assert.True(t, done)

	done, err = checkDescribe(describeInfo{Status: pbmStatusCanceled}, pbmCmdBackup)
	require.EqualError(t, err, "backup was canceled")
	assert.True(t, done)

	done, err = checkDescribe(describeInfo{Status: pbmStatusError, Error: "oplog has insufficient range"}, pbmCmdBackup)
	require.EqualError(t, err, "oplog has insufficient range")
	assert.True(t, done)

	done, err = checkDescribe(describeInfo{Status: pbmStatusPartlyDone, Error: "partial"}, pbmCmdBackup)
	require.EqualError(t, err, "partial")
	assert.True(t, done)

	done, err = checkDescribe(describeInfo{Status: "running"}, pbmCmdBackup)
	require.NoError(t, err)
	assert.False(t, done)
}

func TestCheckStatus(t *testing.T) {
	t.Parallel()

	done, err := checkStatus(pbmStatusDone, "", pbmCmdBackup)
	require.NoError(t, err)
	assert.True(t, done)

	done, err = checkStatus(pbmStatusCanceled, "", pbmCmdBackup)
	require.EqualError(t, err, "backup was canceled")
	assert.True(t, done)

	done, err = checkStatus(pbmStatusError, "storage unavailable", pbmCmdBackup)
	require.EqualError(t, err, "storage unavailable")
	assert.True(t, done)

	done, err = checkStatus(pbmStatusError, "", pbmCmdBackup)
	require.EqualError(t, err, "backup failed")
	assert.True(t, done)

	done, err = checkStatus(pbmStatusPartlyDone, "", pbmCmdRestore)
	require.EqualError(t, err, "restore partly completed")
	assert.True(t, done)

	done, err = checkStatus("running", "", pbmCmdBackup)
	require.NoError(t, err)
	assert.False(t, done)
}

func TestRestoreByName(t *testing.T) {
	t.Parallel()

	list := []pbmListRestore{
		{Name: "2024-01-01T00:00:00Z", Status: pbmStatusDone},
		{Name: "2024-01-02T00:00:00Z", Status: pbmStatusError, Error: "failed"},
	}

	assert.Nil(t, restoreByName(list, "missing"))
	require.NotNil(t, restoreByName(list, "2024-01-02T00:00:00Z"))
}

func TestSnapshotByName(t *testing.T) {
	t.Parallel()

	status := &pbmStatus{}
	status.Backups.Snapshot = []pbmSnapshot{
		{Name: "2024-01-01T00:00:00Z", Status: pbmStatusDone},
		{Name: "2024-01-02T00:00:00Z", Status: pbmStatusError, Error: "failed"},
	}

	assert.Nil(t, snapshotByName(status, "missing"))
	require.NotNil(t, snapshotByName(status, "2024-01-02T00:00:00Z"))
}

func TestGroupDescribeErrs(t *testing.T) {
	t.Parallel()

	err := groupDescribeErrs(describeInfo{
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
