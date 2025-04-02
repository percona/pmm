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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	backuppb "github.com/percona/pmm/api/backup/v1"
)

func TestCreateDBURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dbConfig DBConnConfig
		url      string
	}{
		{
			name: "network",
			dbConfig: DBConnConfig{
				User:     "user",
				Password: "pass",
				Address:  "localhost",
				Port:     1234,
			},
			url: "mongodb://user:pass@localhost:1234",
		},
		{
			name: "network without credentials",
			dbConfig: DBConnConfig{
				Address: "localhost",
				Port:    1234,
			},
			url: "mongodb://localhost:1234",
		},
		{
			name: "socket",
			dbConfig: DBConnConfig{
				User:     "user",
				Password: "pass",
				Socket:   "/tmp/mongo",
			},
			url: "mongodb://user:pass@%2Ftmp%2Fmongo",
		},
		{
			name: "socket without credentials",
			dbConfig: DBConnConfig{
				Socket: "/tmp/mongo",
			},
			url: "mongodb://%2Ftmp%2Fmongo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.url, test.dbConfig.createDBURL().String())
		})
	}
}

func TestNewMongoDBBackupJob(t *testing.T) {
	t.Parallel()
	testJobDuration := 1 * time.Second

	tests := []struct {
		name      string
		dataModel backuppb.DataModel
		pitr      bool
		errMsg    string
	}{
		{
			name:      "logical backup model",
			dataModel: backuppb.DataModel_DATA_MODEL_LOGICAL,
			errMsg:    "",
		},
		{
			name:      "physical backup model",
			dataModel: backuppb.DataModel_DATA_MODEL_PHYSICAL,
			errMsg:    "",
		},
		{
			name:      "invalid backup model",
			dataModel: backuppb.DataModel_DATA_MODEL_UNSPECIFIED,
			errMsg:    "'DATA_MODEL_UNSPECIFIED' is not a supported data model for MongoDB backups",
		},
		{
			name:      "pitr fails for physical backups",
			pitr:      true,
			dataModel: backuppb.DataModel_DATA_MODEL_PHYSICAL,
			errMsg:    "PITR is only supported for logical backups",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewMongoDBBackupJob(t.Name(), testJobDuration, t.Name(), "", BackupLocationConfig{}, tc.pitr, tc.dataModel, "artifact_folder")
			if tc.errMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errMsg)
			}
		})
	}
}
