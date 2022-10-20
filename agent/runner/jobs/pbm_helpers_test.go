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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreatePBMConfig(t *testing.T) {
	s3Config := S3LocationConfig{
		Endpoint:     "test_endpoint",
		AccessKey:    "test_access_key",
		SecretKey:    "test_secret_key",
		BucketName:   "test_bucket_name",
		BucketRegion: "test_region",
	}

	pmmClientStorageConfig := PMMClientBackupLocationConfig{
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
				Type:               BackupLocationType("invalid type"),
				S3Config:           &s3Config,
				LocalStorageConfig: nil,
			},
			inputPitr: true,
			output:    nil,
			errString: "unknown location config",
		},
		{
			name: "s3 config type",
			inputLocation: BackupLocationConfig{
				Type:               S3BackupLocationType,
				S3Config:           &s3Config,
				LocalStorageConfig: nil,
			},
			inputPitr: true,
			output:    &expectedOutput1,
			errString: "",
		},
		{
			name: "pmm client config type",
			inputLocation: BackupLocationConfig{
				Type:               PMMClientBackupLocationType,
				S3Config:           nil,
				LocalStorageConfig: &pmmClientStorageConfig,
			},
			inputPitr: false,
			output:    &expectedOutput2,
			errString: "",
		},
		{
			name: "ignores filled up config instead relying on config type",
			inputLocation: BackupLocationConfig{
				Type:               PMMClientBackupLocationType,
				S3Config:           &s3Config,
				LocalStorageConfig: &pmmClientStorageConfig,
			},
			inputPitr: false,
			output:    &expectedOutput2,
			errString: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			res, err := createPBMConfig(&test.inputLocation, "test_prefix", test.inputPitr)
			if test.errString != "" {
				assert.ErrorContains(t, err, test.errString)
				assert.Nil(t, res)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, test.output, res)
		})
	}
}
