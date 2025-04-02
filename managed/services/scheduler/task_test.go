// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

func TestValidation(t *testing.T) {
	t.Parallel()

	t.Run("mySQL task", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			params *BackupTaskParams
			errMsg string
		}{
			{
				name: "normal",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "",
			},
			{
				name: "empty name",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "backup name can't be empty",
			},
			{
				name: "empty serviceID",
				params: &BackupTaskParams{
					ServiceID:  "",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "service id can't be empty",
			},
			{
				name: "empty locationId",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "location id can't be empty",
			},
			{
				name: "empty data model",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  "",
					Mode:       models.Snapshot,
				},
				errMsg: "invalid argument: empty data model",
			},
			{
				name: "empty mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       "",
				},
				errMsg: "invalid argument: empty backup mode",
			},
			{
				name: "invalid data model",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  "invalid",
					Mode:       models.Snapshot,
				},
				errMsg: "invalid argument: invalid data model 'invalid'",
			},
			{
				name: "invalid backup mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       "invalid",
				},
				errMsg: "invalid argument: invalid backup mode 'invalid'",
			},
			{
				name: "unsupported data model",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.LogicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "unsupported backup data model for mySQL: logical",
			},
			{
				name: "unsupported incremental backup mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Incremental,
				},
				errMsg: "unsupported backup mode for mySQL: incremental",
			},
			{
				name: "unsupported PITR backup mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.PITR,
				},
				errMsg: "unsupported backup mode for mySQL: pitr",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := NewMySQLBackupTask(tt.params)

				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
					return
				}

				require.NoError(t, err)
			})
		}
	})

	t.Run("mongoDB task", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			params *BackupTaskParams
			errMsg string
		}{
			{
				name: "normal snapshot",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.LogicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "",
			},
			{
				name: "normal PITR",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.LogicalDataModel,
					Mode:       models.PITR,
				},
				errMsg: "",
			},
			{
				name: "empty name",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "backup name can't be empty",
			},
			{
				name: "empty serviceID",
				params: &BackupTaskParams{
					ServiceID:  "",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "service id can't be empty",
			},
			{
				name: "empty locationId",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "location id can't be empty",
			},
			{
				name: "empty data model",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  "",
					Mode:       models.Snapshot,
				},
				errMsg: "invalid argument: empty data model",
			},
			{
				name: "empty mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       "",
				},
				errMsg: "invalid argument: empty backup mode",
			},
			{
				name: "invalid data model",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  "invalid",
					Mode:       models.Snapshot,
				},
				errMsg: "invalid argument: invalid data model 'invalid'",
			},
			{
				name: "invalid backup mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.LogicalDataModel,
					Mode:       "invalid",
				},
				errMsg: "invalid argument: invalid backup mode 'invalid'",
			},
			{
				name: "unsupported incremental backup mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.LogicalDataModel,
					Mode:       models.Incremental,
				},
				errMsg: "unsupported backup mode for mongoDB: incremental",
			},
			{
				name: "no error on physical snapshot backups",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.Snapshot,
				},
				errMsg: "",
			},
			{
				name: "unsupported PITR backup mode",
				params: &BackupTaskParams{
					ServiceID:  "service-id",
					LocationID: "location-id",
					Name:       "name",
					DataModel:  models.PhysicalDataModel,
					Mode:       models.PITR,
				},
				errMsg: "PITR is only supported for logical backups: the specified backup model is not compatible with other parameters",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := NewMongoDBBackupTask(tt.params)

				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
					return
				}

				require.NoError(t, err)
			})
		}
	})
}
